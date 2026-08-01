package config

import (
	"fmt"
	"os"

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

		// RetryDelaySeconds is how long a failed message waits in the
		// retry queue before being redelivered to its original queue.
		RetryDelaySeconds int `envconfig:"RETRY_DELAY_SECONDS" default:"30"`
		// MaxRetryAttempts is how many times a message may be redelivered
		// before it is routed to the parked queue for manual triage.
		MaxRetryAttempts int `envconfig:"MAX_RETRY_ATTEMPTS" default:"5"`
	}

	// BreakerConfig configures the circuit breakers guarding calls to
	// OneSignal and Resend. The same thresholds apply to both providers —
	// nothing today suggests they need independent tuning.
	BreakerConfig struct {
		ConsecutiveFailures uint32 `envconfig:"BREAKER_CONSECUTIVE_FAILURES" default:"5"`
		OpenTimeoutSeconds  int    `envconfig:"BREAKER_OPEN_TIMEOUT_SECONDS" default:"30"`
		HalfOpenMaxRequests uint32 `envconfig:"BREAKER_HALF_OPEN_MAX_REQUESTS" default:"1"`
	}

	// OneSignal configuration
	OneSignalConfig struct {
		AppID      string `envconfig:"ONESIGNAL_APP_ID"`
		RestAPIKey string `envconfig:"ONESIGNAL_REST_API_KEY"`
	}

	// Resend configuration
	ResendConfig struct {
		ResendAPIKey         string   `envconfig:"RESEND_API_KEY"`
		AllowedSenderDomains []string `envconfig:"RESEND_ALLOWED_SENDER_DOMAINS" default:"@posta.opencrafts.io"`
	}
}

// The LoadConfig function loads the env file specified and returns
// a valid configuration object ready for use
func LoadConfig() (*Config, error) {
	cfg := Config{}

	// load the configs. A missing .env file is fine — deployments such as
	// Dokploy inject environment variables directly into the container
	// instead of mounting a .env file.
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("Failed to load environment variables: %v", err)
	}
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("Failed to load environment variables: %v", err)
	}
	return &cfg, nil
}
