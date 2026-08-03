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
	"github.com/go-chi/cors"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/kirban/social-media/internal/api"
	"github.com/kirban/social-media/internal/broker"
	"github.com/kirban/social-media/internal/cache"
	"github.com/kirban/social-media/internal/config"
	"github.com/kirban/social-media/internal/db"
	applogger "github.com/kirban/social-media/internal/logger"
	appmiddleware "github.com/kirban/social-media/internal/middleware"
	"github.com/kirban/social-media/internal/repository"
	"github.com/kirban/social-media/internal/service"
	"github.com/kirban/social-media/internal/transport/websocket"
)

type repositories struct {
	user    *repository.UserRepository
	post    *repository.PostRepository
	friends *repository.FriendsRepository
	dialog  *repository.DialogRepository
}

type services struct {
	user    *service.UserService
	post    *service.PostsService
	friends *service.FriendsService
	dialog  *service.DialogService
}

// hubs holds one WebSocket hub per async channel. Each channel is an
// independent endpoint with its own connection registry, so a message pushed on
// one channel never leaks onto another.
type hubs struct {
	feedPosted *websocket.Hub
}

type AppServer struct {
	config       *config.Config
	logger       *applogger.AppLogger
	db           *db.Cluster
	cache        cache.Cache
	repos        *repositories
	svcs         *services
	hubs         *hubs
	httpServer   *http.Server
	natsConn     *nats.Conn
	stream       jetstream.Stream
	publisher    *broker.Publisher
	feedConsumer *broker.Consumer
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

	go s.hubs.feedPosted.Run(ctx)

	go func() {
		if err := s.feedConsumer.Run(ctx); err != nil {
			s.logger.Error().Err(err).Msg("feed fan-out consumer stopped")
		}
	}()

	go func() {
		s.logger.Info().Msgf("HTTP server listening on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	s.logger.Info().Msg("Server started. Press CTRL+C to stop")
	<-ctx.Done()
	s.logger.Info().Msg("Got exit signal. Gracefully shutting down.")
	if mc, ok := s.cache.(*cache.MemoryCache); ok {
		mc.Stop()
	}
	s.natsConn.Close()
}

func (s *AppServer) initDeps() error {
	deps := []func() error{
		s.initConfig,
		s.initLogger,
		s.initDb,
		s.initMigrations,
		s.initCache,
		s.initRepositories,
		s.initHubs,
		s.initBroker,
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

func (s *AppServer) initCache() error {
	s.cache = cache.NewMemoryCache()
	return nil
}

func (s *AppServer) initRepositories() error {
	s.repos = &repositories{
		user:    repository.NewUserRepository(s.db),
		post:    repository.NewPostRepository(s.db, s.logger),
		friends: repository.NewFriendsRepository(s.db, s.logger),
		dialog:  repository.NewDialogRepository(s.db, s.logger),
	}
	return nil
}

// initBroker connects to NATS, ensures the POSTS stream exists, and builds the
// event publisher used by PostsService. The per-instance fan-out consumer is
// built in initServices (it needs the friends service) and started in Run.
func (s *AppServer) initBroker() error {
	nc, js, err := broker.Connect(s.config.NATS)
	if err != nil {
		return err
	}

	stream, err := broker.EnsureStream(context.Background(), js, s.config.NATS)
	if err != nil {
		nc.Close()
		return err
	}

	s.natsConn = nc
	s.stream = stream
	s.publisher = broker.NewPublisher(js, s.config.NATS, s.logger)
	return nil
}

func (s *AppServer) initServices() error {
	friendsSvc := service.NewFriendsService(s.repos.friends, s.cache, s.logger)
	s.svcs = &services{
		user:    service.NewUserService(s.repos.user, s.config.Auth.JWTSecret),
		post:    service.NewPostsService(s.repos.post, s.cache, s.logger, s.publisher),
		friends: friendsSvc,
		dialog:  service.NewDialogService(s.logger, s.repos.dialog),
	}

	fanout := broker.NewFeedFanout(friendsSvc, s.cache, s.hubs.feedPosted, s.logger)
	s.feedConsumer = broker.NewConsumer(s.stream, s.config.NATS, fanout, s.logger)
	return nil
}

func (s *AppServer) initHTTPServer() error {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(appmiddleware.Logging(s.logger))
	r.Use(chimiddleware.Recoverer)

	// CORS must run before routing so preflight OPTIONS requests are answered
	// even on paths that only register GET/POST/PUT. With no configured origins
	// the middleware is skipped entirely, keeping same-origin deployments as
	// they were.
	if len(s.config.Server.CORSAllowedOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   s.config.Server.CORSAllowedOrigins,
			AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodOptions},
			AllowedHeaders:   []string{"Authorization", "Content-Type"},
			AllowCredentials: false,
			MaxAge:           300,
		}))
	}

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

	r.Group(func(r chi.Router) {
		// The generated API seeds this scope key per route; a hand-mounted route
		// must do the same, or Auth treats the endpoint as public and skips the
		// JWT check (see middleware.Auth).
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), api.BearerAuthScopes, []string{})
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		r.Use(appmiddleware.Auth(s.config.Auth.JWTSecret, api.BearerAuthScopes))
		// Async channel /post/feed/posted (AsyncAPI spec): friends' new-post feed.
		// Served under /api/v1 alongside the REST routes; the unprefixed path is
		// kept so existing clients keep working.
		r.Get(so.BaseURL+"/post/feed/posted", s.hubs.feedPosted.ServeWS)
		r.Get("/post/feed/posted", s.hubs.feedPosted.ServeWS)
	})

	s.httpServer = &http.Server{
		Addr: addr,
		Handler: api.HandlerWithOptions(&api.Handlers{
			Logger:     s.logger,
			UserSvc:    s.svcs.user,
			PostSvc:    s.svcs.post,
			FriendsSvc: s.svcs.friends,
			DialogSvc:  s.svcs.dialog,
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

func (s *AppServer) initHubs() error {
	s.hubs = &hubs{
		feedPosted: websocket.NewHub(s.logger, s.config.Server.WSAllowedOrigins),
	}
	return nil
}
