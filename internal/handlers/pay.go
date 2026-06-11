package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

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
		http.Error(w, err.Error(), 400)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	method := req.Method
	if method == "" {
		method = cfg.AICollectMethod
	}

	bizContentBytes, _ := json.Marshal(req.BizContent)
	params := map[string]string{
		"app_id":     cfg.AppID,
		"method":     method,
		"format":     "JSON",
		"charset":    "utf-8",
		"sign_type":  "RSA2",
		"timestamp":  "2026-06-11 12:00:00",
		"version":    cfg.AICollectVersion,
		"notify_url": cfg.NotifyURL,
		"biz_content": string(bizContentBytes),
	}

	sign, err := utils.SignRSA2(params, cfg.AppPrivateKey)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	params["sign"] = sign

	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}

	resp, err := http.Post(cfg.Gateway, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
