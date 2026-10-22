package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	baseURL := os.Getenv("INFRAI_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	s := service{client: infraiClient{baseURL: baseURL, key: key, http: &http.Client{Timeout: 10 * time.Second}}, ledger: &ledger{}}
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}
