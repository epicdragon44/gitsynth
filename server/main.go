package main

import (
	"os"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/palantir/go-baseapp/baseapp"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/rcrowley/go-metrics"
	"goji.io/pat"
)

func main() {
	config, err := ReadConfig("config.yml")
	if err != nil {
		panic(err)
	}

	// Configure logger
	logger := baseapp.NewLogger(config.Logging)

	// Create server with default parameters
	serverParams := baseapp.DefaultParams(logger, "gitsynth.")
	server, err := baseapp.NewServer(config.Server, serverParams...)
	if err != nil {
		panic(err)
	}

	// Configure GitHub client creator with metrics
	metricsRegistry := metrics.DefaultRegistry
	cc, err := githubapp.NewDefaultCachingClientCreator(
		config.Github,
		githubapp.WithClientUserAgent("gitsynth/1.0.0"),
		githubapp.WithClientTimeout(300*time.Second),
		githubapp.WithClientCaching(false, func() httpcache.Cache { return httpcache.NewMemoryCache() }),
		githubapp.WithClientMiddleware(
			githubapp.ClientMetrics(metricsRegistry),
		),
	)
	if err != nil {
		panic(err)
	}

	// Create temporary directory for git operations
	tmpDir, err := os.MkdirTemp("", "gitsynth-*")
	if err != nil {
		panic(err)
	}

	// Initialize GitService
	gitService := NewGitService()

	// Initialize handler
	prMergeHandler := &PRMergeHandler{
		ClientCreator: cc,
		workdir:      tmpDir,
		gitService:   gitService,
	}

	// Create GitHub webhook handler
	webhookHandler := githubapp.NewDefaultEventDispatcher(config.Github, prMergeHandler)

	// Register routes with the server
	server.Mux().Handle(pat.Post(githubapp.DefaultWebhookRoute), webhookHandler)
	server.Mux().Handle(pat.Get("/"), &HomeHandler{})

	// Start the server (blocking)
	logger.Info().Msg("Starting server...")
	if err = server.Start(); err != nil {
		logger.Fatal().Err(err).Msg("Server failed")
	}
}
