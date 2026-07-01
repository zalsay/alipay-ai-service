package main

import (
	"log"
	"net/http"

	"github.com/zalsay/alipay-ai-service/internal/config"
	"github.com/zalsay/alipay-ai-service/internal/handlers"
	"github.com/zalsay/alipay-ai-service/internal/payment"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := payment.Configure(cfg.PaymentStateBackend, cfg.PaymentStateDBPath, cfg.PaymentStateDBDSN); err != nil {
		log.Fatalf("open payment state db: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handlers.HandleHealthz)

	// Backward-compatible generic OpenAPI proxy.
	mux.HandleFunc("POST /v1/ai-collect/call", handlers.HandlePay)

	// AI Collect gateway routes for Agent scenarios.
	mux.HandleFunc("POST /v1/ai-collect/credential/query", handlers.HandleCredentialQuery)
	mux.HandleFunc("POST /v1/ai-collect/fulfillment/confirm", handlers.HandleFulfillmentConfirm)
	mux.HandleFunc("POST /v1/paid-resource/payment-needed", handlers.HandlePaymentNeeded)
	mux.HandleFunc("POST /v1/paid-resource/check", handlers.HandlePaymentCheck)
	mux.HandleFunc("POST /v1/paid-resource/prepare", handlers.HandlePaidResource)

	// Alipay async notification callback.
	mux.HandleFunc("POST /v1/alipay/notify", handlers.HandleNotify)

	log.Printf("alipay ai service listening on %s", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, mux); err != nil {
		log.Fatal(err)
	}
}
