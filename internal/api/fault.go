package api

import (
	"math/rand"
	"net/http"
)

// WithFault500 randomly returns HTTP 500 before calling next (for load demos).
func WithFault500(rate float64, next http.HandlerFunc) http.HandlerFunc {
	if rate <= 0 {
		return next
	}
	if rate > 1 {
		rate = 1
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if rand.Float64() < rate {
			writeError(w, http.StatusInternalServerError, "injected fault")
			return
		}
		next(w, r)
	}
}
