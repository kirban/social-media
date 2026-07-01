package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/kirban/social-media/internal/api"
	"github.com/kirban/social-media/internal/config"
	"github.com/kirban/social-media/internal/db"
	applogger "github.com/kirban/social-media/internal/logger"
	appmiddleware "github.com/kirban/social-media/internal/middleware"
	"github.com/kirban/social-media/internal/repository"
	"github.com/kirban/social-media/internal/service"
)

type repositories struct {
	user *repository.UserRepository
	post *repository.PostRepository
}

type services struct {
	user *service.UserService
	post *service.PostsService
}

type AppServer struct {
	config     *config.Config
	logger     *applogger.AppLogger
	db         *db.Cluster
	repos      *repositories
	svcs       *services
	httpServer *http.Server
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
	defer s.db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		s.logger.Info().Msgf("HTTP server listening on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	s.logger.Info().Msg("Server started. Press CTRL+C to stop")
	<-ctx.Done()
	s.logger.Info().Msg("Got exit signal. Gracefully shutting down.")
}

func (s *AppServer) initDeps() error {
	deps := []func() error{
		s.initConfig,
		s.initLogger,
		s.initDb,
		s.initMigrations,
		s.initRepositories,
		s.initServices,
		s.initHTTPServer,
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

func (s *AppServer) initRepositories() error {
	s.repos = &repositories{
		user: repository.NewUserRepository(s.db),
		post: repository.NewPostRepository(s.db),
	}
	return nil
}

func (s *AppServer) initServices() error {
	s.svcs = &services{
		user: service.NewUserService(s.repos.user, s.config.Auth.JWTSecret),
		post: service.NewPostsService(s.repos.post),
	}
	return nil
}

func (s *AppServer) initHTTPServer() error {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(appmiddleware.Logging(s.logger))
	r.Use(chimiddleware.Recoverer)

	so := api.ChiServerOptions{
		BaseRouter: r,
		BaseURL:    "/api/v1",
		Middlewares: []api.MiddlewareFunc{
			appmiddleware.Auth(s.config.Auth.JWTSecret, api.BearerAuthScopes),
		},
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(api.N5xx{Message: err.Error()})
		},
	}

	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr: addr,
		Handler: api.HandlerWithOptions(&api.Handlers{
			Logger:  s.logger,
			UserSvc: s.svcs.user,
			PostSvc: s.svcs.post,
		}, so),
	}
	return nil
}

func (s *AppServer) initDb() error {
	cluster, err := db.NewCluster(s.config.Database)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}

	s.db = cluster
	return nil
}

func (s *AppServer) initMigrations() error {
	return s.db.Migrate()
}
