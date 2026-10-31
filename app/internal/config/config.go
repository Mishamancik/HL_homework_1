package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds runtime settings read from the environment.
type Config struct {
	ListenAddr      string
	PodID           string
	Datacenter      string
	Fault500Rate    float64 // probability of injected HTTP 500 on business handlers
	LatencyBaseMs   int     // uniform base jitter [0, LatencyBaseMs] ms
	LatencyTailRate float64
	LatencyTailMs   int // extra delay [10, LatencyTailMs] ms on tail hits
}

// FromEnv loads required POD_ID and DATACENTER, plus optional LISTEN_ADDR / fault / latency knobs.
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

	faultRate := 0.0
	if v := os.Getenv("FAULT_500_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return Config{}, fmt.Errorf("FAULT_500_RATE: %w", err)
		}
		if f < 0 || f > 1 {
			return Config{}, fmt.Errorf("FAULT_500_RATE must be between 0 and 1")
		}
		faultRate = f
	}

	baseMs := 2
	if v := os.Getenv("LATENCY_JITTER_MS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("LATENCY_JITTER_MS must be a non-negative integer")
		}
		baseMs = n
	}

	tailRate := 0.08
	if v := os.Getenv("LATENCY_TAIL_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return Config{}, fmt.Errorf("LATENCY_TAIL_RATE: %w", err)
		}
		if f < 0 || f > 1 {
			return Config{}, fmt.Errorf("LATENCY_TAIL_RATE must be between 0 and 1")
		}
		tailRate = f
	}

	tailMs := 40
	if v := os.Getenv("LATENCY_TAIL_MS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("LATENCY_TAIL_MS must be a non-negative integer")
		}
		tailMs = n
	}

	return Config{
		ListenAddr:      addr,
		PodID:           podID,
		Datacenter:      dc,
		Fault500Rate:    faultRate,
		LatencyBaseMs:   baseMs,
		LatencyTailRate: tailRate,
		LatencyTailMs:   tailMs,
	}, nil
}
