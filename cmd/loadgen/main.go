package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Background load: ramps to TARGET_RPS across TARGETS, hits all CRUD handlers,
// injects a small share of 404s via missing IDs. Runs until the process is stopped.
func main() {
	targets := splitCSV(envOr("TARGETS", "http://app-msk:8080,http://app-spb:8080"))
	targetRPS := envFloat("TARGET_RPS", 40)
	ramp := envDuration("RAMP_DURATION", 3*time.Minute)
	err404 := envFloat("ERROR_404_RATE", 0.03)
	workers := envInt("WORKERS", 32)

	if len(targets) == 0 {
		log.Fatal("TARGETS is empty")
	}
	if targetRPS < 1 {
		targetRPS = 1
	}

	client := &http.Client{Timeout: 5 * time.Second}
	pools := make([]*idPool, len(targets))
	for i := range targets {
		pools[i] = newIDPool()
	}

	var (
		okCount   atomic.Int64
		errCount  atomic.Int64
		sentCount atomic.Int64
	)

	jobs := make(chan int, workers*4)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				status, err := doRequest(client, targets[idx], pools[idx], err404)
				sentCount.Add(1)
				if err != nil || status >= 400 {
					errCount.Add(1)
				} else {
					okCount.Add(1)
				}
			}
		}()
	}

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			log.Printf("stats sent=%d ok=%d err=%d", sentCount.Load(), okCount.Load(), errCount.Load())
		}
	}()

	log.Printf("loadgen targets=%v target_rps=%.1f ramp=%s err404=%.2f workers=%d",
		targets, targetRPS, ramp, err404, workers)

	started := time.Now()
	rr := 0
	for {
		elapsed := time.Since(started)
		rps := currentRPS(elapsed, ramp, targetRPS)
		interval := time.Second / time.Duration(rps)
		if interval < time.Millisecond {
			interval = time.Millisecond
		}

		jobs <- rr % len(targets)
		rr++
		time.Sleep(interval)
	}
}

func currentRPS(elapsed, ramp time.Duration, target float64) float64 {
	const floor = 2.0
	if ramp <= 0 || elapsed >= ramp {
		return target
	}
	frac := float64(elapsed) / float64(ramp)
	rps := floor + (target-floor)*frac
	if rps < floor {
		return floor
	}
	return rps
}

func doRequest(client *http.Client, base string, pool *idPool, err404 float64) (int, error) {
	// ~3% intentional 404s; rest mix of create/get/update on known IDs for this target.
	if rand.Float64() < err404 {
		return do(client, http.MethodGet, base+"/items/missing-"+strconv.FormatInt(rand.Int63(), 36), nil)
	}

	id := pool.randomOrEmpty()
	switch {
	case id == "" || rand.Float64() < 0.35:
		id = fmt.Sprintf("item-%d-%d", time.Now().UnixNano(), rand.Int63())
		body := fmt.Sprintf(`{"id":%q,"name":%q}`, id, "n-"+id)
		status, err := do(client, http.MethodPost, base+"/items", []byte(body))
		if err == nil && (status == http.StatusCreated || status == http.StatusConflict) {
			pool.add(id)
		}
		return status, err
	case rand.Float64() < 0.5:
		return do(client, http.MethodGet, base+"/items/"+id, nil)
	default:
		body := fmt.Sprintf(`{"name":%q}`, "upd-"+strconv.FormatInt(rand.Int63(), 36))
		return do(client, http.MethodPut, base+"/items/"+id, []byte(body))
	}
}

func do(client *http.Client, method, url string, body []byte) (int, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode, nil
}

type idPool struct {
	mu   sync.Mutex
	ids  []string
	capN int
}

func newIDPool() *idPool { return &idPool{capN: 500} }

func (p *idPool) add(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, id)
	if len(p.ids) > p.capN {
		p.ids = p.ids[len(p.ids)-p.capN:]
	}
}

func (p *idPool) randomOrEmpty() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.ids) == 0 {
		return ""
	}
	return p.ids[rand.Intn(len(p.ids))]
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.TrimRight(p, "/"))
		}
	}
	return out
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envFloat(k string, def float64) float64 {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
