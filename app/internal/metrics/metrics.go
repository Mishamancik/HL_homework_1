package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector holds Prometheus instruments for the golden signals.
type Collector struct {
	duration *prometheus.HistogramVec
	total    *prometheus.CounterVec
}

// New creates a collector with pod/datacenter as constant labels and registers it.
func New(podID, datacenter string) (*Collector, error) {
	constLabels := prometheus.Labels{
		"pod_id":     podID,
		"datacenter": datacenter,
	}

	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request latency in seconds (histogram for correct multi-instance percentiles).",
			// Fine-grained low end for typical in-process CRUD latencies.
			Buckets:     []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
			ConstLabels: constLabels,
		},
		[]string{"handler"},
	)

	total := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total HTTP requests by handler and status code (RPS and errors).",
			ConstLabels: constLabels,
		},
		[]string{"handler", "code"},
	)

	c := &Collector{duration: duration, total: total}
	if err := prometheus.Register(duration); err != nil {
		return nil, err
	}
	if err := prometheus.Register(total); err != nil {
		return nil, err
	}
	return c, nil
}

// Handler returns the Prometheus scrape endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// Middleware instruments a named business handler (method + path template).
func (c *Collector) Middleware(handlerID string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rw, r)

		code := strconv.Itoa(rw.status)
		c.duration.WithLabelValues(handlerID).Observe(time.Since(start).Seconds())
		c.total.WithLabelValues(handlerID, code).Inc()
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}
