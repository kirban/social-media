package app

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kirban/social-media/internal/config"
)

type AppServer struct {
	config *config.Config
	// logger
	// db
	// http server
}

func NewAppServer() (*AppServer, error) {
	app := &AppServer{}

	if err := app.initDeps(); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *AppServer) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Panicf("uncaught panic %v", r)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		// todo: start http server
	}()

	slog.Info("Server started. Press CTRL+C to stop")
	<-ctx.Done()
	slog.Info("Got exit signal. Gracefully shutdown.")
}

func (s *AppServer) initDeps() error {
	deps := []func() error{
		s.initConfig,
	}

	for _, dep := range deps {
		if err := dep(); err != nil {
			return err
		}
	}

	return nil
}

func (s *AppServer) initConfig() error {
	cfgPath := os.Getenv("CONFIG_PATH")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	s.config = cfg
	return nil
}
