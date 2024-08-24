//
// varenv_test.go
// Copyright (C) 2024 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package varenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupEnvOrString(t *testing.T) {
	// Test with existing environment variable
	t.Setenv("TEST_STRING", "env_value")
	result := LookupEnvOrString("TEST_STRING", "default_value")
	assert.Equal(t, "env_value", result, "Expected environment variable value")

	// Test with non-existing environment variable
	result = LookupEnvOrString("NON_EXISTENT_VAR", "default_value")
	assert.Equal(t, "default_value", result, "Expected default value")
}

func TestLookupEnvOrInt(t *testing.T) {
	// Test with existing environment variable having a valid integer value
	t.Setenv("TEST_INT", "42")
	result := LookupEnvOrInt("TEST_INT", 0)
	assert.Equal(t, 42, result, "Expected environment variable value")

	// Test with existing environment variable having an invalid integer value
	t.Setenv("INVALID_INT", "not_an_int")
	result = LookupEnvOrInt("INVALID_INT", 0)
	assert.Equal(t, 0, result, "Expected default value")

	// Test with non-existing environment variable
	result = LookupEnvOrInt("NON_EXISTENT_VAR", 99)
	assert.Equal(t, 99, result, "Expected default value")
}

func TestLookupEnvOrBool(t *testing.T) {
	// Test with existing environment variable having a valid boolean value
	t.Setenv("TEST_BOOL", "true")
	result := LookupEnvOrBool("TEST_BOOL", false)
	assert.True(t, result, "Expected environment variable value")

	// Test with existing environment variable having an invalid boolean value
	t.Setenv("INVALID_BOOL", "not_a_bool")
	result = LookupEnvOrBool("INVALID_BOOL", false)
	assert.False(t, result, "Expected default value")

	// Test with non-existing environment variable
	result = LookupEnvOrBool("NON_EXISTENT_VAR", true)
	assert.True(t, result, "Expected default value")
}

