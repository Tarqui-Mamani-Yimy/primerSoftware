// realtime_adapter.go links the underlying Service to the realtime hub's
// MembershipResolver. The boundary sits in the httpapi package so realtime
// stays decoupled from service; future adapters (envoy push, slack mirror,
// etc.) implement realtime.MembershipResolver directly without depending
// on the service package.

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ai-uml-architect/gobackend/internal/domain"
	"github.com/ai-uml-architect/gobackend/internal/realtime"
	"github.com/ai-uml-architect/gobackend/internal/service"
)

// ServiceResolver adapts Service.Store() into the realtime
// MembershipResolver contract. Users joining via the WS are checked
// against the same store-backed IsMember used by REST handlers so a
// WebSocket upgrade cannot bypass the membership gate.
type ServiceResolver struct {
	Service *service.Service
}

// DisplayName trims long userIDs so dashboards always render a short,
// stable label. The Service does not currently expose a display_name
// table; this implementation can be swapped out by replacing the
// resolver before NewHub returns.
func (s *ServiceResolver) DisplayName(userID string) string {
	if userID == "" {
		return "anonymous"
	}
	if len(userID) <= 12 {
		return userID
	}
	return userID[:12]
}

// IsMember forwards to the store-backed membership check that REST
// endpoints already use, so the WS gate cannot drift from REST.
func (s *ServiceResolver) IsMember(projectID, userID string) bool {
	if s.Service == nil {
		return false
	}
	st := s.Service.Store()
	if st == nil {
		return false
	}
	ok, err := st.IsMember(context.Background(), projectID, userID)
	return err == nil && ok
}

// LatestDocument returns the working document plus the latest
// version/review counters so the freshly-joined peer can reconcile
// without an extra GET.
func (s *ServiceResolver) LatestDocument(_ context.Context, projectID, diagramID string) (domain.DiagramDocument, int, int64, error) {
	if s.Service == nil {
		return domain.DiagramDocument{}, 0, 0, nil
	}
	st := s.Service.Store()
	if st == nil {
		return domain.DiagramDocument{}, 0, 0, nil
	}
	row, err := st.FindDiagram(context.Background(), projectID, diagramID)
	if err != nil {
		return domain.DiagramDocument{}, 0, 0, err
	}
	var doc domain.DiagramDocument
	if err := json.Unmarshal(row.Document, &doc); err != nil {
		return domain.DiagramDocument{}, 0, 0, err
	}
	version, err := st.CurrentVersion(context.Background(), diagramID)
	if err != nil {
		return domain.DiagramDocument{}, 0, 0, err
	}
	review, err := st.CurrentReview(context.Background(), diagramID)
	if err != nil {
		return domain.DiagramDocument{}, 0, 0, err
	}
	doc.Version = version
	doc.ReviewNumber = review
	doc.ID = &diagramID
	return doc, version, review, nil
}

// NewHub returns a Hub whose MembershipResolver is the supplied Service.
// The hub is independent of the Service at this point; main calls
// svc.AttachBroadcaster(hub.BroadcastDiagramChanged) afterwards so the
// service fans out diagram.changed envelopes after each successful
// autosave/checkpoint write.
func NewHub(svc *service.Service) *realtime.Hub {
	return realtime.NewHub(&ServiceResolver{Service: svc}, realtime.Config{})
}

// HubHandler returns the WebSocket upgrade handler wired to the supplied
// hub and the bearer-auth authenticator production wants. The router
// installs this on the WS route behind the standard withAuth gate so a
// non-member never reaches the upgrade.
func HubHandler(hub *realtime.Hub, svc *service.Service) http.HandlerFunc {
	return realtime.Handle(hub, AuthFuncFor(svc))
}

// AuthFuncFor returns the bearer-token authenticator the realtime
// handler binds. REST and WS share a single auth path so a stale or
// revoked token closes the WS upgrade with the same 401 envelope.
func AuthFuncFor(svc *service.Service) realtime.AuthFunc {
	return func(r *http.Request) (string, bool) {
		header := r.Header.Get("Authorization")
		if !isBearer(header) {
			return "", false
		}
		if svc == nil {
			return "", false
		}
		userID, err := svc.Authenticate(r.Context(), header[len("Bearer "):])
		if err != nil || userID == "" {
			return "", false
		}
		return userID, true
	}
}
