package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ibra172/go-ffmpeg-pipeline/docs"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/config"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/auth"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/grpc/taskpb"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/middleware"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/queue/rabbitmq"
	"google.golang.org/grpc"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title        Media Pipeline App API
// @version      1.0
// @description  API Server for a Multimedia Processor Application using FFmpeg

// @host         127.0.0.1:8000
// @BasePath     /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.MustNew()
	// set swagger host dynamically from config
	docs.SwaggerInfo.Host = "127.0.0.1" + cfg.Port

	rabbitmqClient, err := rabbitmq.NewClient(cfg.AMQPURL)
	if err != nil {
		return fmt.Errorf("failed to create rabbitMQ client: %w", err)
	}
	defer rabbitmqClient.Close()

	taskSender, err := rabbitmq.NewSender(rabbitmqClient, cfg.TaskQueueName)
	if err != nil {
		return fmt.Errorf("failed to create rabbitMQ sender: %w", err)
	}

	taskRepository := task.NewRamRepository()

	taskService := task.NewService(taskRepository, taskSender, cfg.DataDir)
	taskHandler := task.NewHandler(taskService)

	userRepository := auth.NewUserRamRepository()
	sessionRepository := auth.NewSessionRamRepo()
	authService := auth.NewService(userRepository, sessionRepository)
	authHandler := auth.NewHandler(authService)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	authMiddleware := auth.RequireAuth(authService)

	mux.Handle("POST /task", authMiddleware(http.HandlerFunc(taskHandler.CreateTask)))
	mux.Handle("GET /status/{id}", authMiddleware(http.HandlerFunc(taskHandler.GetStatus)))
	mux.Handle("GET /result/{id}", authMiddleware(http.HandlerFunc(taskHandler.GetResult)))

	mux.Handle("/swagger.json", http.FileServer(http.Dir(cfg.SwaggerDir)))
	mux.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger.json")))

	var handler http.Handler = mux
	handler = middleware.Panic(handler)
	handler = middleware.Logging(handler)
	handler = middleware.RequestID(logger)(handler)

	server := &http.Server{
		Addr:    cfg.Port,
		Handler: handler,
	}

	grpcListener, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %s: %w", cfg.GRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	taskpb.RegisterTaskServiceServer(grpcServer, task.NewGRPCHandler(taskService))

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	grpcErrCh := make(chan error, 1)
	go func() {
		logger.Info("starting gRPC server", "addr", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			grpcErrCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("the HTTP server has failed: %w", err)
	case err := <-grpcErrCh:
		return fmt.Errorf("the gRPC server has failed: %w", err)
	case <-ctx.Done():
		logger.Info("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error("graceful shutdown of HTTP server failed", "error", err)
				server.Close()
			}
		}()

		go func() {
			defer wg.Done()
			done := make(chan struct{})
			go func() {
				grpcServer.GracefulStop()
				close(done)
			}()

			select {
			case <-done:
				logger.Info("gRPC server stopped gracefully")
			case <-shutdownCtx.Done():
				logger.Error("graceful shutdown of gRPC server failed")
				grpcServer.Stop()
			}
		}()

		wg.Wait()
		logger.Info("shutdown complete")
	}

	return nil
}
