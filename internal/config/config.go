package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ServerAddr      string
	Gateway         string
	AppID           string
	AppPrivateKey   string
	AlipayPublicKey string
	NotifyURL       string
	AICollectMethod string
	AICollectVersion string
	AppAuthToken    string
}

func Load() (Config, error) {
	cfg := Config{
		ServerAddr:       getenv("SERVER_ADDR", ":8080"),
		Gateway:          getenv("ALIPAY_GATEWAY", "https://openapi.alipay.com/gateway.do"),
		AppID:            os.Getenv("ALIPAY_APP_ID"),
		NotifyURL:        os.Getenv("ALIPAY_NOTIFY_URL"),
		AICollectMethod:  os.Getenv("ALIPAY_AI_COLLECT_METHOD"),
		AICollectVersion: getenv("ALIPAY_AI_COLLECT_VERSION", "1.0"),
		AppAuthToken:     os.Getenv("ALIPAY_APP_AUTH_TOKEN"),
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

	if cfg.AppID == "" {
		return cfg, errors.New("ALIPAY_APP_ID is required")
	}
	if cfg.AppPrivateKey == "" {
		return cfg, errors.New("ALIPAY_APP_PRIVATE_KEY or ALIPAY_APP_PRIVATE_KEY_FILE is required")
	}
	if cfg.AlipayPublicKey == "" {
		return cfg, errors.New("ALIPAY_PUBLIC_KEY or ALIPAY_PUBLIC_KEY_FILE is required")
	}
	if cfg.AICollectMethod == "" {
		return cfg, errors.New("ALIPAY_AI_COLLECT_METHOD is required unless each call supplies method")
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
