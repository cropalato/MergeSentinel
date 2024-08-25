//
// conf_test.go
// Copyright (C) 2024 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package conf

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewConfig tests various scenarios for the NewConfig function.
func TestNewConfig(t *testing.T) {
	// Create a temporary configuration file for testing
	configFile := "test_config.json"
	defer os.Remove(configFile) // Clean up the file after the test

	// Define a valid configuration
	validConfig := `{
		"gitlab_token": "glpat-Phuki3Xu5Rohghohrode",
		"gitlab_url": "https://gitlab.com",
		"webhook_token": "ddalsdnHAS8OP",
		"projects": [
			{
				"webhook_token": "aaKJHJhasa122AS",
				"project_id": 1,
				"approvals": ["user1", "user2"],
				"min_approv": 2
			}
		],
		"psql_conn_url": "postgres://user:password@localhost/dbname",
		"cors_origin": "https://example.com"
	}`

	// Write the valid configuration to the file
	if err := os.WriteFile(configFile, []byte(validConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test loading a valid configuration file
	t.Run("Valid config file", func(t *testing.T) {
		conf, err := NewConfig(configFile)
		assert.NoError(t, err, "Expected no error when loading a valid config file")
		assert.NotNil(t, conf, "Expected a non-nil config object")
		assert.Equal(t, "glpat-Phuki3Xu5Rohghohrode", conf.GitlabToken, "Unexpected GitlabToken")
		assert.Equal(t, "https://gitlab.com", conf.GitlabURL, "Unexpected GitlabURL")
		assert.Equal(t, "ddalsdnHAS8OP", conf.WebHookToken, "Unexpected WebHookToken")
		assert.Len(t, conf.Projects, 1, "Expected one project in the config")
		assert.Equal(t, 1, conf.Projects[0].ProjectId, "Unexpected ProjectId")
		assert.Equal(t, []string{"user1", "user2"}, conf.Projects[0].Approvals, "Unexpected Approvals")
		assert.Equal(t, 2, conf.Projects[0].MinApprov, "Unexpected MinApprov")
		assert.Equal(t, "aaKJHJhasa122AS", conf.Projects[0].WebHookToken, "Unexpected WebHookToken")
		assert.Equal(t, "postgres://user:password@localhost/dbname", conf.PsqlConn, "Unexpected PsqlConn")
		assert.Equal(t, "https://example.com", conf.CorsOrigin, "Unexpected CorsOrigin")
	})

	// Test loading a non-existent configuration file
	t.Run("Non-existent config file", func(t *testing.T) {
		_, err := NewConfig("non_existent_file.json")
		assert.Error(t, err, "Expected error when loading a non-existent config file")
	})
}


