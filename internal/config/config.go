package config

import (
	"fmt"
	"os"
)

// Config holds runtime settings read from the environment.
type Config struct {
	ListenAddr  string
	PodID       string
	Datacenter  string
}

// FromEnv loads required POD_ID and DATACENTER, plus optional LISTEN_ADDR.
func FromEnv() (Config, error) {
	podID := os.Getenv("POD_ID")
	if podID == "" {
		return Config{}, fmt.Errorf("POD_ID environment variable is required")
	}
	dc := os.Getenv("DATACENTER")
	if dc == "" {
		return Config{}, fmt.Errorf("DATACENTER environment variable is required")
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	return Config{
		ListenAddr: addr,
		PodID:      podID,
		Datacenter: dc,
	}, nil
}
