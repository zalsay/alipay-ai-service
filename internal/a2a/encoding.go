package a2a

import (
	"encoding/base64"
	"encoding/json"
)

func EncodeBase64URLJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func DecodeBase64JSON(value string, v interface{}) error {
	b, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		b, err = base64.RawURLEncoding.DecodeString(value)
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
