package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/NishanthSaravanan/codzee-review-test/handlers"
	"github.com/NishanthSaravanan/codzee-review-test/store"
)

func main() {
	store := store.NewTaskStore()
	handler := handlers.NewAPI(store).Handler()

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("TaskFlow API listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
