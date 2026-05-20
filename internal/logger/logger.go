package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kirban/social-media/internal/config"
	"github.com/rs/zerolog"
)

type AppLogger struct {
	zerolog.Logger
}

func NewAppLogger(cfg *config.Config) (*AppLogger, error) {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", cfg.LogLevel, err)
	}

	var w io.Writer
	if cfg.Env == "local" {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		w = os.Stdout
	}

	zl := zerolog.New(w).Level(level).With().Timestamp().Logger()

	return &AppLogger{Logger: zl}, nil
}
