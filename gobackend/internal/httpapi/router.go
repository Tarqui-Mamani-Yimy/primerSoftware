// Package httpapi wires the /api/v1 route table as stubs (GOBE-01).
//
// Every handler is a deliberately unimplemented stub: GOBE-02 ports the
// auth/membership logic, CRUD, and versioning. The route list, methods,
// Bearer scheme, and error envelope already match the Java backend contract.
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ai-uml-architect/gobackend/internal/domain"
)

// Route describes one public API endpoint.
type Route struct {
	Method  string
	Pattern string
}

// Routes returns the full public /api/v1 route table. Order is stable.
func Routes() []Route {
	return []Route{
		{Method: http.MethodPost, Pattern: "/api/v1/auth/login"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}"},
		{Method: http.MethodPut, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}"},
		{Method: http.MethodGet, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/versions"},
		{Method: http.MethodPost, Pattern: "/api/v1/projects/{projectId}/diagrams/{id}/versions/{version}/restore"},
	}
}

// loginPattern is the only route that does not require a Bearer token,
// mirroring SecurityConfig's permitAll for POST /api/v1/auth/login.
const loginPattern = "/api/v1/auth/login"

// NewMux builds a ServeMux serving the route table with stub handlers.
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	for _, route := range Routes() {
		mux.HandleFunc(route.Method+" "+route.Pattern, stub)
	}
	return mux
}

func stub(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != loginPattern && !isBearer(r.Header.Get("Authorization")) {
		writeError(w, http.StatusUnauthorized, "Missing or invalid Authorization header")
		return
	}
	writeError(w, http.StatusNotImplemented, "not implemented (GOBE-01 skeleton stub)")
}

// isBearer accepts only the "Bearer <token>" scheme, mirroring BearerFilter.
func isBearer(header string) bool {
	const prefix = "Bearer "
	return len(header) > len(prefix) && header[:len(prefix)] == prefix
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(domain.ErrorEnvelope{Message: message})
}
