package a2a

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zalsay/alipay-ai-service/internal/alipay"
	"github.com/zalsay/alipay-ai-service/internal/config"
)

type Service struct {
	cfg    config.Config
	client *alipay.Client
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg, client: alipay.NewClient(cfg)}
}

func (s *Service) Verify(ctx context.Context, proof PaymentProof) (VerifyResponse, error) {
	biz := map[string]interface{}{
		"trade_no": proof.Protocol.TradeNo,
		"payment" + "_proof": proof.Protocol.PaymentProof,
		"client" + "_session": proof.Method.ClientSession,
	}
	resp, err := s.client.Execute(ctx, alipay.Request{Method: s.cfg.PaymentVerifyMethod, BizContent: biz})
	if err != nil {
		return VerifyResponse{}, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(resp.RawBody, &raw); err != nil {
		return VerifyResponse{Action: "payment_verify", HTTPStatus: resp.HTTPStatus}, nil
	}
	body := objectValue(raw, "alipay_aipay_agent_payment_verify_response")
	active, _ := body["active"].(bool)
	return VerifyResponse{
		Action:     "payment_verify",
		HTTPStatus: resp.HTTPStatus,
		Active:     active,
		TradeNo:    textValue(body["trade_no"]),
		OutTradeNo: textValue(body["out_trade_no"]),
		ResourceID: textValue(body["resource_id"]),
		AlipayRaw:  raw,
	}, nil
}

func (s *Service) Confirm(ctx context.Context, tradeNo string) (FulfillmentResponse, error) {
	if tradeNo == "" {
		return FulfillmentResponse{}, fmt.Errorf("trade_no is required")
	}
	resp, err := s.client.Execute(ctx, alipay.Request{
		Method: s.cfg.AICollectFulfillmentMethod,
		BizContent: map[string]interface{}{"trade_no": tradeNo},
	})
	if err != nil {
		return FulfillmentResponse{}, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(resp.RawBody, &raw); err != nil {
		return FulfillmentResponse{Action: "fulfillment_confirm", HTTPStatus: resp.HTTPStatus, TradeNo: tradeNo}, nil
	}
	body := objectValue(raw, "alipay_aipay_agent_fulfillment_confirm_response")
	confirmedTradeNo := textValue(body["trade_no"])
	if confirmedTradeNo == "" {
		confirmedTradeNo = tradeNo
	}
	return FulfillmentResponse{Action: "fulfillment_confirm", HTTPStatus: resp.HTTPStatus, TradeNo: confirmedTradeNo, AlipayRaw: raw}, nil
}

func objectValue(raw map[string]interface{}, key string) map[string]interface{} {
	if v, ok := raw[key].(map[string]interface{}); ok {
		return v
	}
	return map[string]interface{}{}
}

func textValue(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
