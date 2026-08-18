package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	"github.com/RamDass1/test-api/internal/auth"
	"github.com/RamDass1/test-api/internal/config"
	"github.com/RamDass1/test-api/internal/httpapi"
	"github.com/RamDass1/test-api/internal/service"
	"github.com/RamDass1/test-api/internal/store"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.AutoMigrate {
		if err := store.Migrate(ctx, cfg.MySQLDSN); err != nil {
			log.Fatal(err)
		}
	}

	db, err := store.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tokens := auth.NewTokens(cfg.JWTSecret, cfg.JWTTTL)
	svc := service.New(newServiceDB(db), auth.Hasher{}, tokens)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(svc, tokens).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
