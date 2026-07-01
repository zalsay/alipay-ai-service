package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestSignAndVerifyRSA2(t *testing.T) {
	privateKey, publicKey := testKeyPair(t)
	params := map[string]string{
		"app_id":      "2021000000000000",
		"method":      "alipay.test.method",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   "2026-06-11 12:00:00",
		"version":     "1.0",
		"biz_content": `{"out_biz_no":"agent-call-001"}`,
	}

	sign, err := SignRSA2(params, privateKey)
	if err != nil {
		t.Fatalf("SignRSA2 failed: %v", err)
	}
	params["sign"] = sign

	if err := VerifyRSA2(params, publicKey); err != nil {
		t.Fatalf("VerifyRSA2 failed: %v", err)
	}
}

func TestVerifyRSA2RejectsTamperedPayload(t *testing.T) {
	privateKey, publicKey := testKeyPair(t)
	params := map[string]string{
		"app_id":      "2021000000000000",
		"method":      "alipay.test.method",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   "2026-06-11 12:00:00",
		"version":     "1.0",
		"biz_content": `{"out_biz_no":"agent-call-001"}`,
	}

	sign, err := SignRSA2(params, privateKey)
	if err != nil {
		t.Fatalf("SignRSA2 failed: %v", err)
	}
	params["sign"] = sign
	params["biz_content"] = `{"out_biz_no":"tampered"}`

	if err := VerifyRSA2(params, publicKey); err == nil {
		t.Fatal("expected VerifyRSA2 to reject tampered payload")
	}
}

func testKeyPair(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicDER,
	})
	return string(privatePEM), string(publicPEM)
}
