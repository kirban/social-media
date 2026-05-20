package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

const DefaultConfigPath = "app-config.yaml"

type Config struct {
	Env      string       `yaml:"env" env:"ENV" env-required:"true"`
	Database DBConfig     `yaml:"app_db"`
	Server   ServerConfig `yaml:"app_server"`
}

type DBConfig struct {
	Host     string `yaml:"host" env:"DB_HOST" env-required:"true"`
	Port     string `yaml:"port" env:"DB_PORT" env-required:"true"`
	DBName   string `yaml:"database" env:"DB_NAME" env-required:"true"`
	Username string `env:"DB_USER" env-required:"true"`
	Password string `env:"DB_PASSWORD" env-required:"true"`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSL_MODE" env-required:"true"`
}

type ServerConfig struct {
	Host string `yaml:"host" env:"APP_HOST" env-default:"0.0.0.0"`
	Port string `yaml:"port" env:"APP_PORT" env-required:"true" env-default:"8080"`
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
