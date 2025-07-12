package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Server   ServerConfig           `mapstructure:"server"`
	Database DatabaseConfig         `mapstructure:"database"`
	LLM      LLMConfig             `mapstructure:"llm"`
	Logging  LoggingConfig         `mapstructure:"logging"`
	Context  ContextConfig         `mapstructure:"context"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type string `mapstructure:"type"`
	Path string `mapstructure:"path"`
}

// LLMConfig holds LLM provider configurations
type LLMConfig struct {
	Providers map[string]ProviderConfig `mapstructure:"providers"`
}

// ProviderConfig holds configuration for a specific LLM provider
type ProviderConfig struct {
	APIKey  string `mapstructure:"api_key"`
	BaseURL string `mapstructure:"base_url"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// ContextConfig holds context strategy configurations
type ContextConfig struct {
	Strategies map[string]map[string]interface{} `mapstructure:"strategies"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)

	// Database defaults
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.path", "./data/agents.db")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")

	// Context strategy defaults
	viper.SetDefault("context.strategies.last_n.default_count", 10)
	viper.SetDefault("context.strategies.sliding_window.window_size", 5)
	viper.SetDefault("context.strategies.sliding_window.overlap", 2)
	viper.SetDefault("context.strategies.summarize.max_context_length", 20)
	viper.SetDefault("context.strategies.summarize.keep_recent", 5)
	viper.SetDefault("context.strategies.summarize.summary_model", "gpt-3.5-turbo")
}

// GetAddress returns the server address
func (c *Config) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Type != "sqlite" {
		return fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}

	if c.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	return nil
}