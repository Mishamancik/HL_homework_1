package main

import (
	"log"
	"net/http"

	"github.com/mikhail/hometask1/internal/api"
	"github.com/mikhail/hometask1/internal/config"
	"github.com/mikhail/hometask1/internal/metrics"
	"github.com/mikhail/hometask1/internal/store"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	collector, err := metrics.New(cfg.PodID, cfg.Datacenter)
	if err != nil {
		log.Fatalf("metrics: %v", err)
	}

	srv := api.NewServer(store.New())
	latency := api.LatencyJitter{
		BaseMaxMs: cfg.LatencyBaseMs,
		TailRate:  cfg.LatencyTailRate,
		TailMaxMs: cfg.LatencyTailMs,
	}
	rt := api.NewRouter(collector, cfg.Fault500Rate, latency)

	// Business routes — instrumented (golden signals).
	rt.Handle("POST", "/items", srv.CreateItem)
	rt.Handle("GET", "/items/{id}", srv.GetItem)
	rt.Handle("PUT", "/items/{id}", srv.UpdateItem)

	// Service routes — not instrumented.
	rt.HandleRawFunc("GET /health", srv.Health)
	rt.HandleRaw("GET /metrics", metrics.Handler())
	rt.HandleRawFunc("GET /handlers", rt.HandlersJSON)

	log.Printf("listening on %s (pod=%s dc=%s fault500=%.3f jitter=%dms tail=%.0f%%/%dms)",
		cfg.ListenAddr, cfg.PodID, cfg.Datacenter, cfg.Fault500Rate,
		cfg.LatencyBaseMs, cfg.LatencyTailRate*100, cfg.LatencyTailMs)
	if err := http.ListenAndServe(cfg.ListenAddr, rt); err != nil {
		log.Fatalf("server: %v", err)
	}
}
