// Package httpapi is the HTTP adapter: it routes requests to use cases and maps results to DTOs.
package httpapi

import "net/http"

// NewRouter returns the HTTP handler for the whole API.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	return mux
}

// healthz reports that the process is up. It never touches dependencies, so orchestrators can use it as
// a liveness probe.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
