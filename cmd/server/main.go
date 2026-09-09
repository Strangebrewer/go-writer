package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Strangebrewer/go-writer/app"
	"github.com/Strangebrewer/go-writer/config"
	"github.com/Strangebrewer/go-writer/db_connection"
	"github.com/Strangebrewer/go-writer/middleware"
	"github.com/Strangebrewer/go-writer/server"
	"github.com/Strangebrewer/go-writer/tracer"
	"github.com/Strangebrewer/go-writer/writer"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	ctx := context.Background()
	client, db, err := db_connection.Connect(ctx, cfg.DatabaseURL, cfg.DBName)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			slog.Error("failed to disconnect from database", "error", err)
		}
	}()

	authMiddleware, err := middleware.RequireAuth(cfg.JWTPublicKey)
	if err != nil {
		slog.Error("failed to parse JWT public key", "error", err)
		os.Exit(1)
	}

	var tracerClient *tracer.Client
	if cfg.TracerURL != "" && cfg.TracerServiceKey != "" {
		tracerClient = tracer.NewClient(cfg.TracerURL, cfg.TracerServiceKey, "go-writer")
	}

	application := &app.Application{
		ProjectStore: writer.NewProjectStore(db),
		SubjectStore: writer.NewSubjectStore(db),
		TextStore:    writer.NewTextStore(db),
		Tracer:       tracerClient,
	}

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	srv := server.New(":"+port, cfg.AllowedOrigins, application, authMiddleware)

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.HTTPServer.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
