package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	// App
	AppEnv   string `mapstructure:"APP_ENV"`
	GRPCPort int    `mapstructure:"GRPC_PORT"`

	// JWT
	JWTSecret string `mapstructure:"JWT_SECRET"`

	// Logging
	LogLevel string `mapstructure:"LOG_LEVEL"`

	// CORS
	CORSAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`

	// MongoDB
	MongoURI string `mapstructure:"MONGO_URI"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("GRPC_PORT", 50052)
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	v.SetDefault("JWT_SECRET", "change_me_to_a_strong_secret_at_least_32_chars")

	// Read from environment variables only (not .env file)
	v.AutomaticEnv()
	v.BindEnv("GRPC_PORT")
	v.BindEnv("APP_ENV")
	v.BindEnv("JWT_SECRET")
	v.BindEnv("LOG_LEVEL")
	v.BindEnv("CORS_ALLOWED_ORIGINS")
	v.BindEnv("MONGO_URI")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate required fields
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	fmt.Printf("Config: %+v\n", cfg)
	if cfg.JWTSecret == "" || len(cfg.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET is required and must be at least 32 characters")
	}
	if cfg.GRPCPort <= 0 {
		return fmt.Errorf("GRPC_PORT must be greater than 0")
	}
	if cfg.MongoURI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}
	return nil
}
