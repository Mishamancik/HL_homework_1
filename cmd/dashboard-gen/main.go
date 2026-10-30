package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type handlersResponse struct {
	Handlers []handlerInfo `json:"handlers"`
}

type handlerInfo struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`
}

func main() {
	grafanaURL := envOr("GRAFANA_URL", "http://grafana:3000")
	user := envOr("GRAFANA_USER", "admin")
	password := mustEnv("GRAFANA_PASSWORD")
	handlersURL := envOr("APP_HANDLERS_URL", "http://app-msk:8080/handlers")
	datasourceUID := envOr("PROMETHEUS_DATASOURCE_UID", "prometheus")

	handlers, err := fetchHandlers(handlersURL)
	if err != nil {
		log.Fatalf("handlers: %v", err)
	}
	if len(handlers) == 0 {
		log.Fatal("handlers: empty list from application")
	}
	log.Printf("got %d handlers from %s", len(handlers), handlersURL)

	dashboard := buildDashboard(handlers, datasourceUID)
	if err := publishDashboard(grafanaURL, user, password, dashboard); err != nil {
		log.Fatalf("publish: %v", err)
	}
	log.Printf("dashboard published (%d handler groups)", len(handlers))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is required", key)
	}
	return v
}

func fetchHandlers(url string) ([]handlerInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, body)
			time.Sleep(2 * time.Second)
			continue
		}
		var parsed handlersResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, err
		}
		return parsed.Handlers, nil
	}
	return nil, fmt.Errorf("giving up: %w", lastErr)
}

func publishDashboard(grafanaURL, user, password string, dashboard map[string]any) error {
	payload := map[string]any{
		"dashboard": dashboard,
		"overwrite": true,
		"message":   "generated from /handlers",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		req, err := http.NewRequest(http.MethodPost, grafanaURL+"/api/dashboards/db", bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.SetBasicAuth(user, password)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("grafana response: %s", body)
			return nil
		}
		lastErr = fmt.Errorf("status %d: %s", resp.StatusCode, body)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("giving up: %w", lastErr)
}

func buildDashboard(handlers []handlerInfo, dsUID string) map[string]any {
	ds := map[string]any{"type": "prometheus", "uid": dsUID}
	panels := make([]any, 0, len(handlers)*3)
	y := 0

	for i, h := range handlers {
		title := h.Handler
		// Latency: p50 / p95 / p99
		panels = append(panels, timeseriesPanel(
			1000+i*10+1,
			title+" — Latency",
			0, y, 8, 8,
			ds,
			[]queryTarget{
				{RefID: "A", Legend: "p50", Expr: histogramQuantile(0.50, h.Handler)},
				{RefID: "B", Legend: "p95", Expr: histogramQuantile(0.95, h.Handler)},
				{RefID: "C", Legend: "p99", Expr: histogramQuantile(0.99, h.Handler)},
			},
			"s",
		))
		// RPS
		panels = append(panels, timeseriesPanel(
			1000+i*10+2,
			title+" — RPS",
			8, y, 8, 8,
			ds,
			[]queryTarget{
				{RefID: "A", Legend: "rps", Expr: fmt.Sprintf(
					`sum(rate(http_requests_total{handler="%s",datacenter=~"$datacenter",pod_id=~"$pod_id"}[$__rate_interval]))`,
					h.Handler,
				)},
			},
			"reqps",
		))
		// Errors by status code
		panels = append(panels, timeseriesPanel(
			1000+i*10+3,
			title+" — Errors",
			16, y, 8, 8,
			ds,
			[]queryTarget{
				{RefID: "A", Legend: "{{code}}", Expr: fmt.Sprintf(
					`sum by (code) (rate(http_requests_total{handler="%s",code=~"4..|5..",datacenter=~"$datacenter",pod_id=~"$pod_id"}[$__rate_interval]))`,
					h.Handler,
				)},
			},
			"reqps",
		))
		y += 8
	}

	return map[string]any{
		"uid":           "items-handlers",
		"title":         "Items API — golden signals",
		"timezone":      "browser",
		"schemaVersion": 39,
		"version":       1,
		"editable":      false,
		"graphTooltip":  1,
		"time":          map[string]string{"from": "now-15m", "to": "now"},
		"refresh":       "5s",
		"tags":          []string{"generated", "items"},
		"templating": map[string]any{
			"list": []any{
				queryVariable("datacenter", "Datacenter", dsUID,
					`label_values(http_requests_total, datacenter)`),
				queryVariable("pod_id", "Pod", dsUID,
					`label_values(http_requests_total{datacenter=~"$datacenter"}, pod_id)`),
			},
		},
		"annotations": map[string]any{"list": []any{}},
		"panels":      panels,
	}
}

func histogramQuantile(q float64, handler string) string {
	return fmt.Sprintf(
		`histogram_quantile(%.2f, sum by (le) (rate(http_request_duration_seconds_bucket{handler="%s",datacenter=~"$datacenter",pod_id=~"$pod_id"}[$__rate_interval])))`,
		q, handler,
	)
}

type queryTarget struct {
	RefID  string
	Legend string
	Expr   string
}

func timeseriesPanel(id int, title string, x, y, w, h int, ds map[string]any, targets []queryTarget, unit string) map[string]any {
	tgts := make([]any, 0, len(targets))
	for _, t := range targets {
		tgts = append(tgts, map[string]any{
			"refId":         t.RefID,
			"expr":          t.Expr,
			"legendFormat":  t.Legend,
			"datasource":    ds,
			"editorMode":    "code",
			"range":         true,
			"instant":       false,
			"exemplar":      false,
		})
	}

	return map[string]any{
		"id":          id,
		"type":        "timeseries",
		"title":       title,
		"datasource":  ds,
		"gridPos":     map[string]int{"x": x, "y": y, "w": w, "h": h},
		"targets":     tgts,
		"fieldConfig": map[string]any{
			"defaults": map[string]any{
				"unit": unit,
				"custom": map[string]any{
					"drawStyle":   "line",
					"lineWidth":   2,
					"fillOpacity": 10,
					"showPoints":  "never",
					"spanNulls":   true,
				},
			},
			"overrides": []any{},
		},
		"options": map[string]any{
			"legend":  map[string]any{"displayMode": "list", "placement": "bottom", "showLegend": true},
			"tooltip": map[string]any{"mode": "multi", "sort": "desc"},
		},
	}
}

func queryVariable(name, label, dsUID, query string) map[string]any {
	return map[string]any{
		"type":        "query",
		"name":        name,
		"label":       label,
		"datasource":  map[string]any{"type": "prometheus", "uid": dsUID},
		"query":       query,
		"definition":  query,
		"includeAll":  true,
		"multi":       true,
		"allValue":    ".*",
		"refresh":     2,
		"sort":        1,
		"current":     map[string]any{"text": "All", "value": "$__all", "selected": true},
		"options":     []any{},
	}
}
