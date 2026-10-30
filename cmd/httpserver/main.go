package main

import (
	"fmt"
	"httpServer/internal/http1"
	"log/slog"
	"net/http"

	"github.com/VictoriaMetrics/metrics"
)

const addr = ":8080"

func registerHandlers(router *http1.Router) {

	helloHandler := http1.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "hello")
		if err != nil {
			slog.Error("writing to response failed", "error", err)
		}
	})

	router.Handle("GET", "/hello", metricsMiddleware(helloHandler))
	router.HandleFunc("GET", "/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.WritePrometheus(w, true)
	})
}

func main() {
	router := http1.NewRouter()
	registerHandlers(router)
	fmt.Println("Starting server on ", addr)
	if err := http1.ListenAndServe(addr, router); err != nil {
		panic(err)
	}
}
