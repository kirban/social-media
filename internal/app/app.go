package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kirban/social-media/internal/config"
	applogger "github.com/kirban/social-media/internal/logger"
)

type AppServer struct {
	config *config.Config
	logger *applogger.AppLogger
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
			s.logger.Panic().Msgf("uncaught panic: %v", r)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		// todo: start http server
	}()

	s.logger.Info().Msg("Server started. Press CTRL+C to stop")
	<-ctx.Done()
	s.logger.Info().Msg("Got exit signal. Gracefully shutting down.")
}

func (s *AppServer) initDeps() error {
	deps := []func() error{
		s.initConfig,
		s.initLogger,
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

func (s *AppServer) initLogger() error {
	l, err := applogger.NewAppLogger(s.config)
	if err != nil {
		return err
	}

	s.logger = l
	return nil
}
