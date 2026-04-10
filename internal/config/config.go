package config

import (
	"context"
	"fmt"
	"os"
)

// GlobalSettings holds API-wide configuration loaded from environment variables.
type GlobalSettings struct {
	// Port is the HTTP listen address (env: API_PORT, default: ":8080").
	Port string
	// LogLevel controls verbosity: "debug", "info", "warn", "error" (env: API_LOG_LEVEL, default: "info").
	LogLevel string
	// DefaultAWSAccount is the account used when a request omits the account field (env: AWS_DEFAULT_ACCOUNT).
	DefaultAWSAccount string
	// DefaultGCPProject is the project alias used when a request omits the account field (env: GCP_DEFAULT_PROJECT).
	DefaultGCPProject string
}

// Config is the top-level application configuration.
type Config struct {
	Global GlobalSettings
	AWS    AWSConfig
	GCP    GCPConfig
}

// Load reads configuration from environment variables and optional config files
// (aws.config, gcp.config in the working directory).
// It returns an error if the configuration is structurally invalid (e.g. a
// partial credential set). Missing credentials are not an error at this stage;
// call Validate to enforce that at least one provider is reachable.
func Load() (*Config, error) {
	cfg := &Config{
		Global: GlobalSettings{
			Port:              envOrDefault("API_PORT", ":8080"),
			LogLevel:          envOrDefault("API_LOG_LEVEL", "info"),
			DefaultAWSAccount: os.Getenv("AWS_DEFAULT_ACCOUNT"),
			DefaultGCPProject: os.Getenv("GCP_DEFAULT_PROJECT"),
		},
	}

	var err error
	cfg.AWS, err = loadAWSConfig()
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	cfg.GCP, err = loadGCPConfig()
	if err != nil {
		return nil, fmt.Errorf("gcp config: %w", err)
	}
	return cfg, nil
}

// Validate checks that at least one cloud provider is configured.
// Call this after Load, before starting the server, to ensure the application
// has usable credentials.
func (c *Config) Validate(_ context.Context) error {
	if len(c.AWS.Accounts) == 0 && len(c.GCP.Projects) == 0 {
		return fmt.Errorf("no cloud provider credentials configured: " +
			"set AWS credentials via AWS_{ACCOUNT}_ACCESS_KEY_ID / AWS_{ACCOUNT}_SECRET_ACCESS_KEY " +
			"or an aws.config file, and/or GCP credentials via GCP_{PROJECT}_PROJECT_ID / " +
			"GCP_{PROJECT}_CREDENTIALS_FILE or a gcp.config file")
	}
	return nil
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
