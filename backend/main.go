// Command gobackend serves the AI UML Architect API: PostgreSQL persistence
// with embedded migrations, opaque-token auth, the /api/v1 diagram/versioning
// REST contract, and the /api/v1/projects/{projectId}/diagrams/{id}/ws
// WebSocket presence room built on top of gorilla/websocket.
//
// Routing precedence: real-time tickets and WS upgrades share the SAME
// single membership gate the REST endpoints use, so a leaked token or
// a tampered ticket cannot grant presence to a non-member. The
// realtime hub re-evaluates the membership table on every Join so a
// revocation takes effect on the next reconnect.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/ai-uml-architect/gobackend/internal/config"
	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/httpapi"
	"github.com/ai-uml-architect/gobackend/internal/jhipster"
	"github.com/ai-uml-architect/gobackend/internal/migrate"
	"github.com/ai-uml-architect/gobackend/internal/realtime"
	"github.com/ai-uml-architect/gobackend/internal/service"
	"github.com/ai-uml-architect/gobackend/internal/store"
	"github.com/jackc/pgx/v5"
)

// hubAdapter binds *realtime.Hub to service.DiagramPresenceBroadcaster so
// the service can stay decoupled from the realtime package while main
// wires both together.
type hubAdapter struct{ hub *realtime.Hub }

func (h hubAdapter) BroadcastDiagramChanged(projectID, diagramID string, evt domain.DiagramChangedEvent) {
	h.hub.BroadcastDiagramChanged(projectID, diagramID, evt)
}

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	st := store.NewPostgres(pool)
	svc := service.NewWithJhipster(st, jhipster.NewGenerator())
	hub := httpapi.NewHub(svc)
	svc.AttachBroadcaster(hubAdapter{hub: hub})
	hub.Run(ctx)
	defer hub.Close()

	tickets := realtime.NewTicketSigner([]byte(cfg.RealtimeTicketSecret), nil)
	hub.SetHubOptions(realtime.HubOptions{
		AllowOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Empty Origin means a non-browser client (curl, tests, or a
			// same-origin navigation): Bearer/ticket auth plus the
			// membership gate still apply, so there is no ambient-auth
			// CSRF vector to close by denying them.
			return origin == "" || origin == cfg.CORSAllowedOrigin
		},
		Tickets: tickets,
	})
	srv := httpapi.NewServer(svc, cfg.CORSAllowedOrigin, hub, tickets)
	srv.SetVoiceTranscriber(httpapi.NewDeepgramTranscriber(httpapi.DeepgramConfig{
		APIKey: cfg.DeepgramAPIKey, Model: cfg.DeepgramModel, Language: cfg.DeepgramLanguage, BaseURL: cfg.DeepgramBaseURL,
	}))
	srv.SetImageImporter(httpapi.NewGeminiImageImporter(cfg.GeminiAPIKey, cfg.GeminiModels, cfg.GeminiBaseURL))
	log.Printf("gobackend: listening on :%s", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, srv.Handler()))
}
