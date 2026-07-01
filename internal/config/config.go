package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ServerAddr                 string
	Gateway                    string
	AppID                      string
	AppPrivateKey              string
	AlipayPublicKey            string
	NotifyURL                  string
	AICollectMethod            string
	AICollectCredentialMethod  string
	AICollectFulfillmentMethod string
	AICollectVersion           string
	AppAuthToken               string

	SellerID                 string
	SellerName               string
	SellerAppID              string
	SellerUniqueIDKey        string
	DefaultCurrency          string
	PaymentNetwork           string
	PaymentProofTTLMinutes   string
	PaymentVerifyMethod      string
	PaymentStatusQueryMethod string
	PaymentStateBackend      string
	PaymentStateDBPath       string
	PaymentStateDBDSN        string
}

func Load() (Config, error) {
	legacyMethod := os.Getenv("ALIPAY_AI_COLLECT_METHOD")
	cfg := Config{
		ServerAddr:                 getenv("SERVER_ADDR", ":8080"),
		Gateway:                    getenv("ALIPAY_GATEWAY", "https://openapi.alipay.com/gateway.do"),
		AppID:                      os.Getenv("ALIPAY_APP_ID"),
		NotifyURL:                  os.Getenv("ALIPAY_NOTIFY_URL"),
		AICollectMethod:            legacyMethod,
		AICollectCredentialMethod:  getenv("ALIPAY_AI_COLLECT_CREDENTIAL_METHOD", legacyMethod),
		AICollectFulfillmentMethod: getenv("ALIPAY_AI_COLLECT_FULFILLMENT_METHOD", "alipay.aipay.agent.fulfillment.confirm"),
		AICollectVersion:           getenv("ALIPAY_AI_COLLECT_VERSION", "1.0"),
		AppAuthToken:               os.Getenv("ALIPAY_APP_AUTH_TOKEN"),
		SellerID:                   os.Getenv("ALIPAY_SELLER_ID"),
		SellerName:                 os.Getenv("ALIPAY_SELLER_NAME"),
		SellerAppID:                getenv("ALIPAY_SELLER_APP_ID", os.Getenv("ALIPAY_APP_ID")),
		SellerUniqueIDKey:          getenv("ALIPAY_SELLER_UNIQUE_ID_KEY", "seller_id"),
		DefaultCurrency:            getenv("ALIPAY_DEFAULT_CURRENCY", "CNY"),
		PaymentNetwork:             getenv("ALIPAY_PAYMENT_NETWORK", "alipay-a2a-prod"),
		PaymentProofTTLMinutes:     getenv("ALIPAY_PAYMENT_PROOF_TTL_MINUTES", "15"),
		PaymentVerifyMethod:        getenv("ALIPAY_PAYMENT_VERIFY_METHOD", "alipay.aipay.agent.payment.verify"),
		PaymentStatusQueryMethod:   getenv("ALIPAY_PAYMENT_STATUS_QUERY_METHOD", "alipay.trade.query"),
		PaymentStateBackend:        getenv("PAYMENT_STATE_BACKEND", "file"),
		PaymentStateDBPath:         getenv("PAYMENT_STATE_DB_PATH", "/data/payment-state.json"),
		PaymentStateDBDSN:          os.Getenv("PAYMENT_STATE_DB_DSN"),
	}

	var err error
	cfg.AppPrivateKey, err = readSecret("ALIPAY_APP_PRIVATE_KEY", "ALIPAY_APP_PRIVATE_KEY_FILE")
	if err != nil {
		return cfg, fmt.Errorf("load app private key: %w", err)
	}
	cfg.AlipayPublicKey, err = readSecret("ALIPAY_PUBLIC_KEY", "ALIPAY_PUBLIC_KEY_FILE")
	if err != nil {
		return cfg, fmt.Errorf("load alipay public key: %w", err)
	}

	if cfg.AppPrivateKey == "" {
		return cfg, errors.New("ALIPAY_APP_PRIVATE_KEY or ALIPAY_APP_PRIVATE_KEY_FILE is required")
	}
	if cfg.AlipayPublicKey == "" {
		return cfg, errors.New("ALIPAY_PUBLIC_KEY or ALIPAY_PUBLIC_KEY_FILE is required")
	}
	if cfg.AICollectFulfillmentMethod == "" {
		return cfg, errors.New("ALIPAY_AI_COLLECT_FULFILLMENT_METHOD is required for AI Collect fulfillment confirmation")
	}
	if cfg.PaymentVerifyMethod == "" {
		return cfg, errors.New("ALIPAY_PAYMENT_VERIFY_METHOD is required for Payment-Proof verification")
	}
	if cfg.PaymentStatusQueryMethod == "" {
		return cfg, errors.New("ALIPAY_PAYMENT_STATUS_QUERY_METHOD is required for paid-resource payment status checks")
	}
	switch strings.ToLower(strings.TrimSpace(cfg.PaymentStateBackend)) {
	case "", "file":
		if cfg.PaymentStateDBPath == "" {
			return cfg, errors.New("PAYMENT_STATE_DB_PATH is required")
		}
	case "postgres", "postgresql":
		if cfg.PaymentStateDBDSN == "" {
			return cfg, errors.New("PAYMENT_STATE_DB_DSN is required when PAYMENT_STATE_BACKEND=postgres")
		}
	default:
		return cfg, fmt.Errorf("unsupported PAYMENT_STATE_BACKEND %q", cfg.PaymentStateBackend)
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func readSecret(valueEnv, fileEnv string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(valueEnv)); v != "" {
		return normalizePEM(v), nil
	}
	path := os.Getenv(fileEnv)
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return normalizePEM(string(b)), nil
}

func normalizePEM(v string) string {
	v = strings.TrimSpace(v)
	if strings.Contains(v, "\\n") {
		v = strings.ReplaceAll(v, "\\n", "\n")
	}
	return v
}
