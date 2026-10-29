package main

import (
	"fmt"
	"httpServer/internal/http1"
	"log/slog"
	"net/http"
)

const addr = ":8080"

func main() {
	router := http1.NewRouter()
	router.HandleFunc("GET", "/hello", func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "hello")
		if err != nil {
			slog.Error("writing to response failed", "error", err)
		}
	})
	if err := http1.ListenAndServe(addr, router); err != nil {
		panic(err)
	}
}
