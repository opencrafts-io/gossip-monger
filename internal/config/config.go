package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	// Application configuration
	AppConfig struct {
		Port    int    `envconfig:"GOSSIP_MONGER_PORT"`
		Address string `envconfig:"GOSSIP_MONGER_ADDRESS"`
	}

	// Database configuration
	DatabaseConfig struct {
		DatabaseHost                      string `envconfig:"DB_HOST"`
		DatabaseDriver                    string `envconfig:"DB_DRIVER"`
		DatabaseUser                      string `envconfig:"DB_USER"`
		DatabasePassword                  string `envconfig:"DB_PASSWORD"`
		DatabaseName                      string `envconfig:"DB_NAME"`
		DatabasePort                      int32  `envconfig:"DB_PORT"`
		DatabasePoolMaxConnections        int32  `envconfig:"DB_MAX_CON"`
		DatabasePoolMinConnections        int32  `envconfig:"DB_POOL_MIN_CON"`
		DatabasePoolMaxConnectionLifetime int    `envconfig:"DB_POOL_MAX_LIFETIME"`
	}

	// Rabbitmq configs
	RabbitMQConfig struct {
		RabbitMQUser    string `envconfig:"RABBITMQ_USER"`
		RabbitMQPass    string `envconfig:"RABBITMQ_PASSWORD"`
		RabbitMQAddress string `envconfig:"RABBITMQ_ADDRESS"`
		RabbitMQPort    int    `envconfig:"RABBITMQ_PORT"`
	}
}

// The LoadConfig function loads the env file specified and returns
// a valid configuration object ready for use
func LoadConfig() (*Config, error) {
	cfg := Config{}

	// load the configs
	if err := godotenv.Load(".env"); err != nil {
		return nil, fmt.Errorf("Failed to load environment variables: %v", err)
	}
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("Failed to load environment variables: %v", err)
	}
	return &cfg, nil
}
