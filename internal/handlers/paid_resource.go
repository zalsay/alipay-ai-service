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
	"github.com/zalsay/alipay-ai-service/internal/payment"
)

type paidResourceRequest struct {
	ResourceID string `json:"resource_id"`
	OutTradeNo string `json:"out_trade_no"`
	TradeNo    string `json:"trade_no,omitempty"`
	GoodsName  string `json:"goods_name,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Amount     string `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
}

type paymentNeededResult struct {
	Bill    a2a.PaymentNeeded
	Encoded string
}

func HandlePaidResource(w http.ResponseWriter, r *http.Request) {
	req, err := readPaidResourceRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record, ok, err := payment.LookupUnlocked(req.ResourceID, req.OutTradeNo, req.TradeNo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		writePaidResource(w, req, record)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	matches, err := payment.BillResourceMatches(firstRequestValue(req.ResourceID, verified.ResourceID), firstRequestValue(req.OutTradeNo, verified.OutTradeNo))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !matches {
		http.Error(w, "out_trade_no does not match requested resource", http.StatusForbidden)
		return
	}

	tradeNo := verified.TradeNo
	if tradeNo == "" {
		tradeNo = proof.Protocol.TradeNo
	}
	record, err = payment.MarkUnlocked(payment.UnlockRecord{
		ResourceID: firstRequestValue(req.ResourceID, verified.ResourceID),
		OutTradeNo: firstRequestValue(req.OutTradeNo, verified.OutTradeNo),
		TradeNo:    tradeNo,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go func() {
		if _, err := svc.Confirm(context.Background(), tradeNo); err != nil {
			log.Printf("a2a fulfillment confirm failed: trade_no=%s err=%v", tradeNo, err)
		}
	}()

	writePaidResource(w, req, record)
}

func HandlePaymentCheck(w http.ResponseWriter, r *http.Request) {
	req, err := readPaidResourceRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ResourceID == "" {
		http.Error(w, "resource_id is required", http.StatusBadRequest)
		return
	}
	if req.OutTradeNo == "" && req.TradeNo == "" {
		http.Error(w, "out_trade_no or trade_no is required", http.StatusBadRequest)
		return
	}
	matches, err := payment.BillResourceMatches(req.ResourceID, req.OutTradeNo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !matches {
		http.Error(w, "out_trade_no does not match requested resource", http.StatusForbidden)
		return
	}

	record, ok, err := payment.LookupUnlocked(req.ResourceID, req.OutTradeNo, req.TradeNo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":   "unlocked",
			"unlocked": true,
			"record":   record,
		})
		return
	}

	cfg, err := config.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query, err := a2a.QueryTrade(r.Context(), cfg, a2a.TradeQueryInput{
		OutTradeNo: req.OutTradeNo,
		TradeNo:    req.TradeNo,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if !payment.IsPaidStatus(query.TradeStatus) {
		writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
			"status":       "payment_not_finished",
			"unlocked":     false,
			"resource_id":  req.ResourceID,
			"out_trade_no": firstRequestValue(req.OutTradeNo, query.OutTradeNo),
			"trade_no":     firstRequestValue(req.TradeNo, query.TradeNo),
			"trade_status": query.TradeStatus,
			"alipay":       query,
		})
		return
	}

	record, err = payment.MarkUnlocked(payment.UnlockRecord{
		ResourceID:  req.ResourceID,
		OutTradeNo:  firstRequestValue(req.OutTradeNo, query.OutTradeNo),
		TradeNo:     firstRequestValue(req.TradeNo, query.TradeNo),
		TradeStatus: query.TradeStatus,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "unlocked",
		"unlocked": true,
		"record":   record,
		"alipay":   query,
	})
}

func HandlePaymentNeeded(w http.ResponseWriter, r *http.Request) {
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

	result, err := buildPaymentNeeded(cfg, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"payment_needed":      result.Encoded,
		"payment_header_name": "Payment-Needed",
		"bill":                result.Bill,
	})
}

func readPaidResourceRequest(r *http.Request) (paidResourceRequest, error) {
	req := paidResourceRequest{
		ResourceID: r.URL.Query().Get("resource_id"),
		OutTradeNo: r.URL.Query().Get("out_trade_no"),
		TradeNo:    r.URL.Query().Get("trade_no"),
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
	result, err := buildPaymentNeeded(cfg, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Payment-Needed", result.Encoded)
	writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
		"error":       "Payment Needed",
		"message":     "payment is required to access this resource",
		"resource_id": req.ResourceID,
	})
}

func buildPaymentNeeded(cfg config.Config, req paidResourceRequest) (paymentNeededResult, error) {
	goodsName := firstRequestValue(req.GoodsName, req.Subject)
	bill, encoded, err := a2a.BuildPaymentNeeded(cfg, a2a.BillInput{
		OutTradeNo: req.OutTradeNo,
		ResourceID: req.ResourceID,
		GoodsName:  goodsName,
		Amount:     req.Amount,
		Currency:   req.Currency,
	})
	if err != nil {
		return paymentNeededResult{}, err
	}
	if err := payment.RememberBill(req.ResourceID, req.OutTradeNo); err != nil {
		return paymentNeededResult{}, err
	}
	return paymentNeededResult{Bill: bill, Encoded: encoded}, nil
}

func firstRequestValue(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func writePaidResource(w http.ResponseWriter, req paidResourceRequest, record payment.UnlockRecord) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "ok",
		"message":      "payment verified; resource access granted",
		"resource_id":  firstRequestValue(req.ResourceID, record.ResourceID),
		"out_trade_no": firstRequestValue(req.OutTradeNo, record.OutTradeNo),
		"trade_no":     firstRequestValue(req.TradeNo, record.TradeNo),
		"unlocked_at":  record.UnlockedAt,
		"content":      "replace this placeholder with the real paid resource content",
	})
}
