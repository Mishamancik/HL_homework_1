package main

import (
	"log"
	"net/http"

	"github.com/mikhail/hometask1/internal/api"
	"github.com/mikhail/hometask1/internal/config"
	"github.com/mikhail/hometask1/internal/store"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	srv := api.NewServer(store.New())
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", srv.Health)
	mux.HandleFunc("POST /items", srv.CreateItem)
	mux.HandleFunc("GET /items/{id}", srv.GetItem)
	mux.HandleFunc("PUT /items/{id}", srv.UpdateItem)

	log.Printf("listening on %s (pod=%s dc=%s)", cfg.ListenAddr, cfg.PodID, cfg.Datacenter)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
