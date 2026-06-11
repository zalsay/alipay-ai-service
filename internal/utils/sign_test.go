package utils

import "testing"

const testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQCw8cRHFHS52dkfwwSOMaFcYp/UKkmo25K2Bm1xuK2MXJuLg9Sd
kbbEakYdb8uD2XDa8t9KsjwX3x6lAjuL+hi8CVYwXelzKbsFNxvjrpbyWVlqcIDJ
KDTR2e4skA180GSsR5sl1L+r7sQVXGmgoBNKFd9CIIJdWWID+oxyr81DKwIDAQAB
AoGABEVwLnqlpzFo6MsL7W9Tr3CIVxGtQr4CFQMMmvUOYV4413cJCHVD3+BM44gQ
sKfdGZlGq3gAB/j2Ix3kVZ7+uUzFbnHhqSlgmVq+Xnv+dnZdFkodrEu17fU+u4rv
1j0Uac9Okc0vqjwN18HFgta4EHJQ1DrShMP8fpAk30IUEvECQQDXaJ0whFNPX5AO
1Tc2q1kVMXnTyg8VmHTrc0lDuOfuEjp6gN2USr73PbZe96EQ3BcCxlKjjp+7RGsE
JWVHVtVvAkEA0dYVixKwEqrML/SA8kmDsj6cAKjcqjSivbEWlNQU85+2LiJ+RA1V
4lU3Vwrnduu3BRj4qh7dBuO1MbZkHhCu8wJAJZzWrEef8zZx5bWZPg05kv2Mx20m
7ayVdyX+Qu5oDL7krxbUYnh3vpkq5Qzz3Tt6pE0X25w5vrr4Pxs2IdbA2QJBAJxK
+6zwmq6qPNh3tkbZnCQ7CA8eebRjbHy+Eiw2AlWHLvIT01AU4vxQCIUl7wS/M8Ww
GzB6dm3OrwpJqSn4MxsCQG6sS4Gcg7wYtJhNSefALPcY03u9UqxfkpAcZnHlJwlc
Kba8zJVH9Qi/KxY02sS7wkVAeP5PAAEB6p+aY3TtEoY=
-----END RSA PRIVATE KEY-----`

const testPublicKey = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCw8cRHFHS52dkfwwSOMaFcYp/U
Kkmo25K2Bm1xuK2MXJuLg9SdkbbEakYdb8uD2XDa8t9KsjwX3x6lAjuL+hi8CVYw
XelzKbsFNxvjrpbyWVlqcIDJKDTR2e4skA180GSsR5sl1L+r7sQVXGmgoBNKFd9C
IIJdWWID+oxyr81DKwIDAQAB
-----END PUBLIC KEY-----`

func TestSignAndVerifyRSA2(t *testing.T) {
	params := map[string]string{
		"app_id":      "2021000000000000",
		"method":      "alipay.test.method",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   "2026-06-11 12:00:00",
		"version":     "1.0",
		"biz_content": `{"out_biz_no":"agent-call-001"}`,
	}

	sign, err := SignRSA2(params, testPrivateKey)
	if err != nil {
		t.Fatalf("SignRSA2 failed: %v", err)
	}
	params["sign"] = sign

	if err := VerifyRSA2(params, testPublicKey); err != nil {
		t.Fatalf("VerifyRSA2 failed: %v", err)
	}
}

func TestVerifyRSA2RejectsTamperedPayload(t *testing.T) {
	params := map[string]string{
		"app_id":      "2021000000000000",
		"method":      "alipay.test.method",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   "2026-06-11 12:00:00",
		"version":     "1.0",
		"biz_content": `{"out_biz_no":"agent-call-001"}`,
	}

	sign, err := SignRSA2(params, testPrivateKey)
	if err != nil {
		t.Fatalf("SignRSA2 failed: %v", err)
	}
	params["sign"] = sign
	params["biz_content"] = `{"out_biz_no":"tampered"}`

	if err := VerifyRSA2(params, testPublicKey); err == nil {
		t.Fatal("expected VerifyRSA2 to reject tampered payload")
	}
}
