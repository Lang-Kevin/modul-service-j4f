package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"contract-service/internal/handler"
	"contract-service/internal/middleware"
	"contract-service/internal/repository"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	_ = godotenv.Load()

	for _, key := range []string{"DATABASE_URL", "JWT_SECRET"} {
		if os.Getenv(key) == "" {
			log.Error("missing required environment variable", "key", key)
			os.Exit(1)
		}
	}

	dbURL     := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	port      := envOr("PORT", "8080")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Error("database ping failed", "err", err)
		os.Exit(1)
	}
	log.Info("database connection established")

	repo   := repository.New(pool)
	h      := handler.New(repo, log)
	status := handler.NewStatusHandler(pool)
	auth   := middleware.JWTAuth(jwtSecret)

	mux := http.NewServeMux()

	// Business routes — JWT geschützt
	mux.Handle("POST /time-slices",     auth(http.HandlerFunc(h.CreateTimeSlice)))
	mux.Handle("GET /time-slices/{id}", auth(http.HandlerFunc(h.GetTimeSlices)))

	// Health routes — kein Auth
	mux.HandleFunc("GET /health/live",  status.Live)
	mux.HandleFunc("GET /health/ready", status.Ready)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-stop
	log.Info("shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "err", err)
	}
	log.Info("server stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}