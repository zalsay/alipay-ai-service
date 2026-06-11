package handlers

import (
	"log"
	"net/http"
)

func HandleNotify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}

	params := make(map[string]string, len(r.PostForm))
	for k, v := range r.PostForm {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}

	// TODO: add RSA2 verify before production use.
	log.Printf("received alipay notify: trade_no=%s out_trade_no=%s trade_status=%s", params["trade_no"], params["out_trade_no"], params["trade_status"])
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("success"))
}
