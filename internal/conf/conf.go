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
	validate "github.com/go-playground/validator/v10"
)

type ApprovRule struct {
	ProjectId int       `json:"project_id"  validate:"required,gt=0"`
	Approvals []string  `json:"approvals"   validate:"required"`
	MinApprov int       `json:"min_approv"  validate:"required,gt=0"`
}

type Config struct {
	GitlabToken string       `json:"gitlab_token"   validate:"required,startswith=glpat-"`
	GitlabURL   string       `json:"gitlab_url"     validate:"required,http_url"`
	Projects    []ApprovRule `json:"projects"       validate:"required"`
	PsqlConn    string       `json:"psql_conn_url"  validate:"required,startswith=postgres://"`
	CorsOrigin  string       `json:"cors_origin"    validate:"required"`
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
	err = json.Unmarshal(byteValue, &conf)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	err = validate.New().Struct(conf)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
  
	return &conf, nil
}
