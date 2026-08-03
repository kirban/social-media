package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const DefaultConfigPath = "configs/app-config.yaml"

type Config struct {
	Env      string       `yaml:"env" env:"ENV" env-required:"true"`
	Database DBConfig     `yaml:"app_db"`
	Server   ServerConfig `yaml:"app_server"`
	NATS     NATSConfig   `yaml:"nats"`
	Auth     AuthConfig   `yaml:"-"`
	LogLevel string       `yaml:"log_level" env:"LOG_LEVEL" env-default:"debug"`
}

type DBConfig struct {
	Host            string          `yaml:"host" env:"DB_HOST" env-required:"true"`
	Port            string          `yaml:"port" env:"DB_PORT" env-required:"true"`
	DBName          string          `yaml:"database" env:"DB_NAME" env-required:"true"`
	Username        string          `env:"DB_USER" env-required:"true"`
	Password        string          `env:"DB_PASSWORD" env-required:"true"`
	SSLMode         string          `yaml:"ssl_mode" env:"DB_SSL_MODE" env-required:"true"`
	MaxOpenConns    int             `yaml:"max_open_conns" env-default:"25"`
	MaxIdleConns    int             `yaml:"max_idle_conns" env-default:"25"`
	MaxConnLifetime time.Duration   `yaml:"max_conn_lifetime" env-default:"5m"`
	Replicas        []ReplicaConfig `yaml:"replicas"`
}

type ReplicaConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type ServerConfig struct {
	Host string `yaml:"host" env:"APP_HOST" env-default:"0.0.0.0"`
	Port string `yaml:"port" env:"APP_PORT" env-required:"true" env-default:"8080"`
	// WSAllowedOrigins are Origin header host patterns accepted on WebSocket
	// upgrade (e.g. "app.example.com", "*.example.com"). Empty means same-origin
	// only. Bearer-token auth already blocks cross-site hijacking, so widen this
	// only for legitimate cross-origin browser clients.
	WSAllowedOrigins []string `yaml:"ws_allowed_origins" env:"WS_ALLOWED_ORIGINS"`
	// CORSAllowedOrigins are full origins allowed to call the REST API from a
	// browser (e.g. "http://localhost:5173"). Empty disables CORS entirely,
	// which is correct when the frontend is served from the same origin or
	// reached through a dev proxy.
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins" env:"CORS_ALLOWED_ORIGINS"`
}

// NATSConfig configures the JetStream broker used for feed fan-out. The stream
// is a short-lived buffer (MaxAge) for post events, not a system of record.
type NATSConfig struct {
	URL            string        `yaml:"url" env:"NATS_URL" env-default:"nats://localhost:4222"`
	Stream         string        `yaml:"stream" env:"NATS_STREAM" env-default:"POSTS"`
	SubjectPrefix  string        `yaml:"subject_prefix" env:"NATS_SUBJECT_PREFIX" env-default:"posts"`
	MaxAge         time.Duration `yaml:"max_age" env:"NATS_MAX_AGE" env-default:"10m"`
	PublishRetries int           `yaml:"publish_retries" env:"NATS_PUBLISH_RETRIES" env-default:"3"`
}

type AuthConfig struct {
	JWTSecret string `yaml:"-" env:"JWT_SECRET" env-required:"true"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.New("config not found")
	}

	var cfg Config

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %w", err)
	}

	return &cfg, nil
}
