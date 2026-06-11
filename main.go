package main

import (
	"log"
	"net/http"

	"github.com/zalsay/alipay-ai-service/internal/config"
	"github.com/zalsay/alipay-ai-service/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.HandleHealthz)
	mux.HandleFunc("POST /v1/ai-collect/call", handlers.HandlePay)
	mux.HandleFunc("POST /v1/alipay/notify", handlers.HandleNotify)

	log.Printf("alipay ai service listening on %s", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, mux); err != nil {
		log.Fatal(err)
	}
}
