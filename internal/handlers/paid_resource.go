package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/zalsay/alipay-ai-service/internal/a2a"
	"github.com/zalsay/alipay-ai-service/internal/config"
)

type paidResourceRequest struct {
	ResourceID string `json:"resource_id"`
	OutTradeNo string `json:"out_trade_no"`
	GoodsName  string `json:"goods_name,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Amount     string `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
}

func HandlePaidResource(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, err := readPaidResourceRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	proofHeader := strings.TrimSpace(r.Header.Get("Payment-Proof"))
	if proofHeader == "" {
		writePaymentNeeded(w, cfg, req)
		return
	}

	proof, err := a2a.ParsePaymentProofHeader(proofHeader)
	if err != nil {
		writePaymentNeeded(w, cfg, req)
		return
	}

	svc := a2a.NewService(cfg)
	verified, err := svc.Verify(r.Context(), proof)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if !verified.Active {
		writePaymentNeeded(w, cfg, req)
		return
	}
	if req.ResourceID != "" && verified.ResourceID != "" && req.ResourceID != verified.ResourceID {
		http.Error(w, "Payment-Proof resource_id does not match requested resource", http.StatusForbidden)
		return
	}

	tradeNo := verified.TradeNo
	if tradeNo == "" {
		tradeNo = proof.Protocol.TradeNo
	}
	go func() {
		if _, err := svc.Confirm(context.Background(), tradeNo); err != nil {
			log.Printf("a2a fulfillment confirm failed: trade_no=%s err=%v", tradeNo, err)
		}
	}()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "ok",
		"message":      "payment verified; resource access granted",
		"resource_id":  firstRequestValue(req.ResourceID, verified.ResourceID),
		"out_trade_no": verified.OutTradeNo,
		"trade_no":     tradeNo,
		"content":      "replace this placeholder with the real paid resource content",
	})
}

func readPaidResourceRequest(r *http.Request) (paidResourceRequest, error) {
	req := paidResourceRequest{
		ResourceID: r.URL.Query().Get("resource_id"),
		OutTradeNo: r.URL.Query().Get("out_trade_no"),
		GoodsName:  r.URL.Query().Get("goods_name"),
		Subject:    r.URL.Query().Get("subject"),
		Amount:     r.URL.Query().Get("amount"),
		Currency:   r.URL.Query().Get("currency"),
	}
	if r.Body == nil {
		return req, nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return req, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return req, nil
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return req, err
	}
	return req, nil
}

func writePaymentNeeded(w http.ResponseWriter, cfg config.Config, req paidResourceRequest) {
	goodsName := firstRequestValue(req.GoodsName, req.Subject)
	_, encoded, err := a2a.BuildPaymentNeeded(cfg, a2a.BillInput{
		OutTradeNo: req.OutTradeNo,
		ResourceID: req.ResourceID,
		GoodsName:  goodsName,
		Amount:     req.Amount,
		Currency:   req.Currency,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Payment-Needed", encoded)
	writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
		"error":       "Payment Needed",
		"message":     "payment is required to access this resource",
		"resource_id": req.ResourceID,
	})
}

func firstRequestValue(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
