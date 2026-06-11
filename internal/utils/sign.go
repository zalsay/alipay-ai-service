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

	priv, err := parseRSAPrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	hashed := sha256.Sum256([]byte(CanonicalString(params, nil)))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func VerifyRSA2(params map[string]string, publicKey string) error {
	if publicKey == "" {
		return errors.New("public key is empty")
	}

	sign := params["sign"]
	if sign == "" {
		return errors.New("missing sign")
	}
	if signType := strings.ToUpper(params["sign_type"]); signType != "" && signType != "RSA2" {
		return fmt.Errorf("unsupported sign_type %q", params["sign_type"])
	}

	pub, err := parseRSAPublicKey(publicKey)
	if err != nil {
		return err
	}

	sig, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return fmt.Errorf("decode sign: %w", err)
	}

	hashed := sha256.Sum256([]byte(CanonicalString(params, map[string]bool{"sign": true, "sign_type": true})))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], sig); err != nil {
		return fmt.Errorf("rsa2 verify failed: %w", err)
	}
	return nil
}

func CanonicalString(params map[string]string, excludes map[string]bool) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "" || v == "" {
			continue
		}
		if excludes != nil && excludes[k] {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	return strings.Join(parts, "&")
}

func parseRSAPrivateKey(privateKey string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(privateKey)))
	if block == nil || !strings.Contains(block.Type, "PRIVATE KEY") {
		return nil, errors.New("invalid private key PEM")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

func parseRSAPublicKey(publicKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(publicKey)))
	if block == nil || !strings.Contains(block.Type, "PUBLIC KEY") {
		return nil, errors.New("invalid public key PEM")
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		key, ok := parsed.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key is not RSA")
		}
		return key, nil
	}

	key, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return key, nil
}
