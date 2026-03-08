//
// conf.go
// Copyright (C) 2023 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package conf

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	validate "github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

type ApprovRule struct {
	ProjectId    int      `json:"project_id"              validate:"gt=0,required"`
	Approvals    []string `json:"approvals"               validate:"gt=0,required"`
	MinApprov    int      `json:"min_approv"              validate:"gt=0,required"`
	WebHookToken string   `json:"webhook_token,omitempty" validate:"omitempty,gt=0"`
}

type Config struct {
	GitlabToken  string       `json:"gitlab_token"            validate:"required,startswith=glpat-"`
	GitlabURL    string       `json:"gitlab_url"              validate:"required,http_url"`
	Projects     []ApprovRule `json:"projects"                validate:"required"`
	PsqlConn     string       `json:"psql_conn_url"           validate:"required,startswith=postgres://"`
	CorsOrigin   string       `json:"cors_origin"             validate:"required"`
	WebHookToken string       `json:"webhook_token,omitempty" validate:"omitempty,gt=0"`
}

// NewConfig reads configuration from a JSON file and validates it
func NewConfig(config_file string) (*Config, error) {
	if config_file == "" {
		config_file = "config.json"
	}
	log.Debug().Str("config", "loading file")
	jsonFile, err := os.Open(config_file)
	if err != nil {
		return nil, fmt.Errorf("failed loading config file: %w", err)
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed reading config file: %w", err)
	}

	var conf Config
	err = json.Unmarshal(byteValue, &conf)
	if err != nil {
		return nil, fmt.Errorf("failed parsing config file: %w", err)
	}
	err = validate.New().Struct(conf)
	if err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &conf, nil
}
