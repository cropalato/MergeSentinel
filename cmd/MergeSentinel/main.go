//
// main.go
// Copyright (C) 2023 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

// This package is runs a RESTApi server used to validate gitlab-ce. it should de called by git hook.
//
// You Should use env variables to config the service.
// ex.:
//
//	export GLCE_APPROV_PATH=/tmp/approval_cfg_rules.json
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cropalato/MergeSentinel/internal/varenv"
	"github.com/cropalato/MergeSentinel/internal/webservices"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	listen := flag.String("listen", varenv.LookupEnvOrString("GLCE_APPROV_LISTEN", ":8080"), "IP and port used by the service. format: '[<ip>]:<port>'. default: ':8080'")
	cfg_path := flag.String("cfg_path", varenv.LookupEnvOrString("GLCE_CONF_PATH", "msentinel.json"), "config file path")
	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	cfg, err := webservices.LoadConfig(*cfg_path)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed loading configuration")
	}
	defer cfg.DB.Close()

	err = cfg.ReinforceAllMrRule()
	if err != nil {
		log.Fatal().Msg("Failed updating database")
	}

	srv := http.Server{
		Addr:              *listen,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	r := mux.NewRouter()
	r.Use(mux.CORSMethodMiddleware(r))
	r.HandleFunc("/state", cfg.State).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/approve", cfg.PostApproval).Methods(http.MethodPost, http.MethodOptions)
	srv.Handler = r

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Str("listening", *listen).Msg("Starting http service")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	<-quit
	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}
