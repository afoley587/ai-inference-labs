package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/afoley587/ai-inference-labs/internal/gateway"
)

func main() {
	target := os.Getenv("BACKEND_URL")
	if target == "" {
		target = "http://localhost:8081"
	}
	handler, err := gateway.New(target)
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("GATEWAY_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		// WriteTimeout is intentionally unset for long-lived response streams.
	}
	log.Printf("gateway listening on %s; upstream %s", addr, target)
	log.Fatal(server.ListenAndServe())
}
