package a2a

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zalsay/alipay-ai-service/internal/alipay"
	"github.com/zalsay/alipay-ai-service/internal/config"
)

type TradeQueryInput struct {
	OutTradeNo string
	TradeNo    string
}

type TradeQueryResponse struct {
	Action      string                 `json:"action"`
	HTTPStatus  int                    `json:"http_status"`
	Code        string                 `json:"code,omitempty"`
	Msg         string                 `json:"msg,omitempty"`
	SubCode     string                 `json:"sub_code,omitempty"`
	SubMsg      string                 `json:"sub_msg,omitempty"`
	TradeNo     string                 `json:"trade_no,omitempty"`
	OutTradeNo  string                 `json:"out_trade_no,omitempty"`
	TradeStatus string                 `json:"trade_status,omitempty"`
	AlipayRaw   map[string]interface{} `json:"alipay_raw,omitempty"`
}

func QueryTrade(ctx context.Context, cfg config.Config, in TradeQueryInput) (TradeQueryResponse, error) {
	if in.OutTradeNo == "" && in.TradeNo == "" {
		return TradeQueryResponse{}, fmt.Errorf("out_trade_no or trade_no is required")
	}

	biz := map[string]interface{}{}
	if in.OutTradeNo != "" {
		biz["out_trade_no"] = in.OutTradeNo
	}
	if in.TradeNo != "" {
		biz["trade_no"] = in.TradeNo
	}

	resp, err := alipay.NewClient(cfg).Execute(ctx, alipay.Request{
		Method:     cfg.PaymentStatusQueryMethod,
		BizContent: biz,
	})
	if err != nil {
		return TradeQueryResponse{}, err
	}

	result := TradeQueryResponse{Action: "trade_query", HTTPStatus: resp.HTTPStatus}
	var raw map[string]interface{}
	if err := json.Unmarshal(resp.RawBody, &raw); err != nil {
		return result, nil
	}
	result.AlipayRaw = raw
	body := objectValue(raw, "alipay_trade_query_response")
	result.Code = textValue(body["code"])
	result.Msg = textValue(body["msg"])
	result.SubCode = textValue(body["sub_code"])
	result.SubMsg = textValue(body["sub_msg"])
	result.TradeNo = textValue(body["trade_no"])
	result.OutTradeNo = textValue(body["out_trade_no"])
	result.TradeStatus = textValue(body["trade_status"])
	return result, nil
}
