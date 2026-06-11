package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/zalsay/alipay-ai-service/internal/aicollect"
	"github.com/zalsay/alipay-ai-service/internal/config"
)

func HandleCredentialQuery(w http.ResponseWriter, r *http.Request) {
	var req aicollect.CredentialQueryRequest
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

	resp, err := aicollect.NewService(cfg).QueryCredential(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
