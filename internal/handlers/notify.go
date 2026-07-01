package handlers

import (
	"log"
	"net/http"

	"github.com/zalsay/alipay-ai-service/internal/config"
	"github.com/zalsay/alipay-ai-service/internal/payment"
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

	if payment.IsPaidStatus(tradeStatus) {
		resourceID, ok, err := payment.BillResourceID(outTradeNo)
		if err != nil {
			log.Printf("lookup payment bill binding failed: trade_no=%s out_trade_no=%s err=%v", tradeNo, outTradeNo, err)
			writeAlipayNotifyResult(w, "fail", http.StatusInternalServerError)
			return
		}
		if ok {
			if _, err := payment.MarkUnlocked(payment.UnlockRecord{
				ResourceID:  resourceID,
				OutTradeNo:  outTradeNo,
				TradeNo:     tradeNo,
				TradeStatus: tradeStatus,
			}); err != nil {
				log.Printf("persist payment unlock failed: trade_no=%s out_trade_no=%s err=%v", tradeNo, outTradeNo, err)
				writeAlipayNotifyResult(w, "fail", http.StatusInternalServerError)
				return
			}
		} else {
			log.Printf("alipay notify paid but no local bill binding found: trade_no=%s out_trade_no=%s", tradeNo, outTradeNo)
		}
	}

	writeAlipayNotifyResult(w, "success", http.StatusOK)
}

func writeAlipayNotifyResult(w http.ResponseWriter, body string, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
