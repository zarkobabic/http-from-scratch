package main

import (
	"httpServer/internal/http1"
	"net/http"

	"github.com/VictoriaMetrics/metrics"
)

var requestCounter = metrics.NewCounter("http_server_requests_total")

func metricsMiddleware(next http.Handler) http.Handler {
	return http1.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCounter.Inc()
		next.ServeHTTP(w, r)
	})
}
