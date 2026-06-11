package utils

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"strings"
)

func SignRSA2(params map[string]string, privateKey string) (string, error) {
	if privateKey == "" {
		return "", errors.New("private key is empty")
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	signStr := ""
	for i, k := range keys {
		if i > 0 {
			signStr += "&"
		}
		signStr += fmt.Sprintf("%s=%s", k, params[k])
	}

	block, _ := pem.Decode([]byte(privateKey))
	if block == nil || !strings.Contains(block.Type, "PRIVATE KEY") {
		return "", errors.New("invalid private key PEM")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write([]byte(signStr))
	hashed := h.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}