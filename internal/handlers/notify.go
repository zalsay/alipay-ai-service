package handlers

import (
	"log"
	"net/http"

	"github.com/zalsay/alipay-ai-service/internal/config"
	"github.com/zalsay/alipay-ai-service/internal/utils"
)

func HandleNotify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("parse alipay notify form failed: %v", err)
		writeAlipayNotifyResult(w, "fail", http.StatusBadRequest)
		return
	}

	params := make(map[string]string, len(r.PostForm))
	for k, v := range r.PostForm {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	cfg, err := config.Load()
	if err != nil {
		log.Printf("load config for alipay notify failed: %v", err)
		writeAlipayNotifyResult(w, "fail", http.StatusInternalServerError)
		return
	}

	if err := utils.VerifyRSA2(params, cfg.AlipayPublicKey); err != nil {
		log.Printf("alipay notify verify failed: %v trade_no=%s out_trade_no=%s", err, params["trade_no"], params["out_trade_no"])
		writeAlipayNotifyResult(w, "fail", http.StatusOK)
		return
	}

	tradeNo := params["trade_no"]
	outTradeNo := params["out_trade_no"]
	tradeStatus := params["trade_status"]
	log.Printf("alipay notify verified: trade_no=%s out_trade_no=%s trade_status=%s", tradeNo, outTradeNo, tradeStatus)

	// TODO: persist notification and update local order state idempotently.
	// Recommended key: out_trade_no + trade_no + trade_status.
	writeAlipayNotifyResult(w, "success", http.StatusOK)
}

func writeAlipayNotifyResult(w http.ResponseWriter, body string, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
