package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ibra172/go-ffmpeg-pipeline/docs"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/config"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/auth"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/middleware"

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
	cfg := config.MustNew()
	// set swagger host dynamically from config
	docs.SwaggerInfo.Host = "127.0.0.1" + cfg.Port

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	taskRepository := task.NewRamRepository()
	taskService := task.NewService(taskRepository)
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

	select {
	case err := <-errCh:
		logger.Error("server failed", "error", err)
	case <-ctx.Done():
		logger.Info("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			server.Close()
		}

		if err := taskService.Shutdown(shutdownCtx); err != nil {
			logger.Warn("some background tasks did not finish before shutdown", "error", err)
		}
	}
}
