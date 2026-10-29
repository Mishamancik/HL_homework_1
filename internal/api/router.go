package api

import (
	"encoding/json"
	"net/http"

	"github.com/mikhail/hometask1/internal/metrics"
)

// HandlerInfo describes a business HTTP route for /handlers and dashboards.
type HandlerInfo struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"` // "METHOD path", matches the Prometheus handler label
}

// Router registers instrumented business routes and exposes /handlers.
type Router struct {
	mux      *http.ServeMux
	metrics  *metrics.Collector
	handlers []HandlerInfo
}

func NewRouter(m *metrics.Collector) *Router {
	return &Router{
		mux:     http.NewServeMux(),
		metrics: m,
	}
}

// Handle registers a business route under method+path and instruments it.
func (rt *Router) Handle(method, path string, h http.HandlerFunc) {
	id := method + " " + path
	rt.handlers = append(rt.handlers, HandlerInfo{
		Method:  method,
		Path:    path,
		Handler: id,
	})
	rt.mux.HandleFunc(method+" "+path, rt.metrics.Middleware(id, h))
}

// HandleRaw registers a non-instrumented route (health, metrics, handlers).
func (rt *Router) HandleRaw(pattern string, h http.Handler) {
	rt.mux.Handle(pattern, h)
}

func (rt *Router) HandleRawFunc(pattern string, h http.HandlerFunc) {
	rt.mux.HandleFunc(pattern, h)
}

// ServeHTTP implements http.Handler.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.mux.ServeHTTP(w, r)
}

// HandlersJSON returns the list of instrumented business handlers.
func (rt *Router) HandlersJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"handlers": rt.handlers,
	})
}
