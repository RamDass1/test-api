package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/RamDass1/test-api/internal/auth"
	"github.com/RamDass1/test-api/internal/cache"
	"github.com/RamDass1/test-api/internal/config"
	"github.com/RamDass1/test-api/internal/httpapi"
	"github.com/RamDass1/test-api/internal/service"
	"github.com/RamDass1/test-api/internal/store"
)

func main() {
	_ = godotenv.Load()
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.AutoMigrate {
		if err := store.Migrate(ctx, cfg.MySQLDSN); err != nil {
			slog.Error("migrate", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	db, err := store.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		slog.Error("mysql", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Warn("redis is unreachable, serving without cache", slog.String("error", err.Error()))
	}

	tokens := auth.NewTokens(cfg.JWTSecret, cfg.JWTTTL)
	taskCache := cache.NewTaskCache(rdb, cfg.CacheTTL)
	svc := service.New(db, taskCache, auth.Hasher{}, tokens)
	api := httpapi.New(svc, tokens)
	go api.Collect(ctx)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	go func() {
		slog.Info("listening", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
