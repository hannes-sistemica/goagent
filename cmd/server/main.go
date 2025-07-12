package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"agent-server/internal/api"
	"agent-server/internal/config"
	contextpkg "agent-server/internal/context"
	"agent-server/internal/llm"
	"agent-server/internal/llm/ollama"
	"agent-server/internal/storage/sqlite"

	"github.com/sirupsen/logrus"
)

func main() {
	// Parse command line flags
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg.Logging)

	logrus.Info("Starting Agent Server...")

	// Ensure data directory exists
	if err := ensureDataDir(cfg.Database.Path); err != nil {
		logrus.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize storage
	repo, err := sqlite.NewRepository(cfg.Database.Path)
	if err != nil {
		logrus.Fatalf("Failed to initialize storage: %v", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			logrus.Errorf("Failed to close repository: %v", err)
		}
	}()

	// Initialize context strategy registry
	ctxRegistry := contextpkg.NewStrategyRegistry()

	// Initialize LLM provider registry
	llmRegistry := llm.NewRegistry()

	// Register Ollama provider
	if providerCfg, exists := cfg.LLM.Providers["ollama"]; exists {
		ollamaProvider := ollama.NewProvider(providerCfg.BaseURL)
		llmRegistry.Register(ollamaProvider)
		logrus.Info("Registered Ollama LLM provider")
	}

	// Create and setup server
	server := api.NewServer(cfg, repo, ctxRegistry, llmRegistry)
	server.SetupRoutes()

	logrus.Infof("Server starting on %s", cfg.GetAddress())

	// Start server
	if err := server.Start(); err != nil {
		logrus.Fatalf("Failed to start server: %v", err)
	}
}

// setupLogging configures the logging system
func setupLogging(cfg config.LoggingConfig) {
	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', using 'info'", cfg.Level)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set log format
	switch cfg.Format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{})
	default:
		logrus.Warnf("Invalid log format '%s', using 'json'", cfg.Format)
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	// Output to stdout
	logrus.SetOutput(os.Stdout)
}

// ensureDataDir creates the data directory if it doesn't exist
func ensureDataDir(dbPath string) error {
	dir := filepath.Dir(dbPath)
	return os.MkdirAll(dir, 0755)
}