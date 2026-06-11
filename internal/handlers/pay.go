package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zalsay/alipay-ai-service/internal/config"
	"github.com/zalsay/alipay-ai-service/internal/utils"
)

type AICollectRequest struct {
	Method     string                 `json:"method"`
	BizContent map[string]interface{} `json:"biz_content"`
}

func HandlePay(w http.ResponseWriter, r *http.Request) {
	var req AICollectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.BizContent == nil {
		http.Error(w, "biz_content is required", http.StatusBadRequest)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	method := strings.TrimSpace(req.Method)
	if method == "" {
		method = cfg.AICollectMethod
	}

	bizContentBytes, err := json.Marshal(req.BizContent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	params := map[string]string{
		"app_id":      cfg.AppID,
		"method":      method,
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     cfg.AICollectVersion,
		"notify_url":  cfg.NotifyURL,
		"biz_content": string(bizContentBytes),
	}
	if cfg.AppAuthToken != "" {
		params["app_auth_token"] = cfg.AppAuthToken
	}

	sign, err := utils.SignRSA2(params, cfg.AppPrivateKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	params["sign"] = sign

	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}

	resp, err := http.Post(cfg.Gateway, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}
