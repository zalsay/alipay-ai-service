package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/zalsay/alipay-ai-service/internal/aicollect"
	"github.com/zalsay/alipay-ai-service/internal/config"
)

func HandlePaidResource(w http.ResponseWriter, r *http.Request) {
	var req aicollect.PaidResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := aicollect.NewService(cfg).PreparePaidResource(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
