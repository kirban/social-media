// Command identity-service is the extracted User/Identity domain as a
// standalone gRPC server, reusing internal/service.UserService and
// internal/repository.UserRepository unchanged. It owns the users table and
// mints the same JWTs the monolith did, via the shared JWT_SECRET.
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	identityv1 "github.com/kirban/social-media/api/proto/gen/identity/v1"
	identityserver "github.com/kirban/social-media/internal/app/identity"
	"github.com/kirban/social-media/internal/config"
	"github.com/kirban/social-media/internal/db"
	applogger "github.com/kirban/social-media/internal/logger"
	middleware_grpc "github.com/kirban/social-media/internal/middleware/grpc"
	"github.com/kirban/social-media/internal/repository"
	"github.com/kirban/social-media/internal/service"
)

const defaultGRPCPort = "50062"

func main() {
	// A "healthcheck" subcommand lets the Docker Compose healthcheck probe
	// this same binary over gRPC without needing a separate grpc_health_probe
	// binary baked into the image.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck(grpcPort()))
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func grpcPort() string {
	if port := os.Getenv("GRPC_PORT"); port != "" {
		return port
	}
	return defaultGRPCPort
}

// healthcheck dials this process's own gRPC port and calls the standard
// health service, returning a process exit code (0 = serving).
func healthcheck(port string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck: dial:", err)
		return 1
	}
	defer conn.Close()

	resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck: check:", err)
		return 1
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		fmt.Fprintln(os.Stderr, "healthcheck: status:", resp.GetStatus())
		return 1
	}
	return 0
}

func run() error {
	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := applogger.NewAppLogger(cfg)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	dbCluster, err := db.NewCluster(cfg.Database)
	if err != nil {
		return fmt.Errorf("init db: %w", err)
	}
	defer dbCluster.Close()

	repo := repository.NewUserRepository(dbCluster)
	svc := service.NewUserService(repo, cfg.Auth.JWTSecret)

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		middleware_grpc.UnaryServerInterceptor,
		middleware_grpc.LoggingUnaryServerInterceptor(logger),
	))
	identityv1.RegisterIdentityServiceServer(grpcServer, identityserver.New(svc))

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	port := grpcPort()

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen on :%s: %w", port, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		logger.Info().Msg("identity-service: shutting down")
		grpcServer.GracefulStop()
	}()

	logger.Info().Msgf("identity-service gRPC listening on :%s", port)
	return grpcServer.Serve(lis)
}
