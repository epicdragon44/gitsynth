package main

import (
	"github.com/palantir/go-baseapp/baseapp"
	"goji.io/pat"
)

func main() {
	// Configure logger
	logger := baseapp.NewLogger(baseapp.LoggingConfig{
		Level:  "INFO",
		Pretty: true,
	})

	// Create server with default parameters
	serverParams := baseapp.DefaultParams(logger, "gitsynth.")
	server, err := baseapp.NewServer(baseapp.HTTPConfig{
		Address: "127.0.0.1",
		Port:    8080,
	}, serverParams...)
	if err != nil {
		panic(err)
	}

	// Register routes with the server
	server.Mux().Handle(pat.Get("/"), &HomeHandler{})

	// Start the server (blocking)
	logger.Info().Msg("Starting server...")
	if err = server.Start(); err != nil {
		logger.Fatal().Err(err).Msg("Server failed")
	}
}
