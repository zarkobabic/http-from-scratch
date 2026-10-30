package main

import (
	"context"
	"fmt"
	"httpServer/internal/http1"
	"httpServer/internal/telemetry"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const addr = ":8080"

func registerHandlers(router *http1.Router) {

	helloHandler := http1.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "hello")
		if err != nil {
			slog.Error("writing to response failed", "error", err)
		}
	})
	middlewareWrapper := metricsMiddleware(helloHandler)
	tracingWrapper := otelhttp.NewHandler(middlewareWrapper, "helloHandler")
	router.Handle("GET", "/hello", tracingWrapper)

	router.HandleFunc("GET", "/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.WritePrometheus(w, true)
	})
}

func main() {
	tracer, err := telemetry.InitTracer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracer.Shutdown(ctx); err != nil {
			log.Fatal("Failed to shutdown tracer provider:", err)
		}
	}()
	router := http1.NewRouter()
	registerHandlers(router)
	fmt.Println("Starting server on ", addr)
	if err := http1.ListenAndServe(addr, router); err != nil {
		panic(err)
	}
}
