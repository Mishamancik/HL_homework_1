package api

import (
	"math/rand"
	"net/http"
	"time"
)

// LatencyJitter adds a small base delay and an occasional longer tail so
// histogram percentiles (esp. p95/p99) are not flat bucket artifacts.
type LatencyJitter struct {
	BaseMaxMs int     // uniform sleep in [0, BaseMaxMs] ms
	TailRate  float64 // probability of an extra slow request
	TailMaxMs int     // extra sleep in [10, TailMaxMs] ms when tail hits
}

func (j LatencyJitter) sleep() {
	if j.BaseMaxMs > 0 {
		time.Sleep(time.Duration(rand.Intn(j.BaseMaxMs+1)) * time.Millisecond)
	}
	if j.TailRate <= 0 || j.TailMaxMs < 10 {
		return
	}
	if rand.Float64() >= j.TailRate {
		return
	}
	extra := 10 + rand.Intn(j.TailMaxMs-9)
	time.Sleep(time.Duration(extra) * time.Millisecond)
}

// WithLatencyJitter sleeps before calling next (observed by metrics middleware).
func WithLatencyJitter(j LatencyJitter, next http.HandlerFunc) http.HandlerFunc {
	if j.BaseMaxMs <= 0 && j.TailRate <= 0 {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		j.sleep()
		next(w, r)
	}
}
