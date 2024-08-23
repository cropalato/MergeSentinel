//
// conf.go
// Copyright (C) 2023 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package conf

import (
	"os"
	"io"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type ApprovRule struct {
	ProjectId int   `json:"project_id"`
	Approvals  []string `json:"approvals"`
	MinApprov  int      `json:"min_approv"`
}

type Config struct {
	GitlabToken string `json:"gitlab_token"`
	GitlabURL   string `json:"gitlab_url"`
	Projects    []ApprovRule `json:"projects"`
	PsqlConn    string `json:"psql_conn_url"`
	CorsOrigin  string `json:"cors_origin"`
}

// NewDefaultConfig reads configuration from environment variables and validates it
func NewConfig(config_file string) (*Config, error) {
	if config_file == "" {
		config_file = "config.json"
	}
	log.Debug().Str("config", "loading file")
	jsonFile, err := os.Open(config_file)
	// if we os.Open returns an error then handle it
	if err != nil {
	  log.Fatal().AnErr("config", err)
		return nil, errors.Wrap(err, "failed loading config file")
	}

	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// read our opened jsonFile as a byte array.
  byteValue, _ := io.ReadAll(jsonFile)

	// we initialize our Users array
	var conf Config
	json.Unmarshal(byteValue, &conf)
  
	return &conf, nil
}
