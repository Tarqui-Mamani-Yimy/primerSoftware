// Command gobackend serves the AI UML Architect API (GOBE-02): PostgreSQL
// persistence with embedded migrations, opaque-token auth, and the full
// /api/v1 diagram/versioning contract.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/ai-uml-architect/gobackend/internal/config"
	"github.com/ai-uml-architect/gobackend/internal/httpapi"
	"github.com/ai-uml-architect/gobackend/internal/migrate"
	"github.com/ai-uml-architect/gobackend/internal/service"
	"github.com/ai-uml-architect/gobackend/internal/store"
	"github.com/jackc/pgx/v5"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	databaseURL := cfg.EffectiveDatabaseURL()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("gobackend: connect database: %v", err)
	}
	defer conn.Close(ctx)
	if err := migrate.Up(ctx, conn); err != nil {
		log.Fatalf("gobackend: migrate: %v", err)
	}
	pool, err := store.OpenPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("gobackend: open pool: %v", err)
	}
	defer pool.Close()

	srv := httpapi.NewServer(service.New(store.NewPostgres(pool)), cfg.CORSAllowedOrigin)
	log.Printf("gobackend: listening on :%s", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, srv.Handler()))
}
