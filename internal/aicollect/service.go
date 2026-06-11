package aicollect

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

type CredentialQueryRequest struct {
	Method     string                 `json:"method,omitempty"`
	BizContent map[string]interface{} `json:"biz_content"`
}

type FulfillmentConfirmRequest struct {
	Method     string                 `json:"method,omitempty"`
	BizContent map[string]interface{} `json:"biz_content"`
}

type PaidResourceRequest struct {
	ResourceID  string                 `json:"resource_id"`
	AgentID     string                 `json:"agent_id,omitempty"`
	OutBizNo    string                 `json:"out_biz_no"`
	Subject     string                 `json:"subject"`
	Amount      string                 `json:"amount"`
	Currency    string                 `json:"currency,omitempty"`
	ExtraParams map[string]interface{} `json:"extra_params,omitempty"`
}

type GatewayResponse struct {
	Action     string          `json:"action"`
	HTTPStatus int             `json:"http_status"`
	AlipayRaw  json.RawMessage `json:"alipay_raw"`
}

func NewService(cfg config.Config) *Service {
	return &Service{cfg: cfg, client: alipay.NewClient(cfg)}
}

func (s *Service) QueryCredential(ctx context.Context, req CredentialQueryRequest) (GatewayResponse, error) {
	method := req.Method
	if method == "" {
		method = s.cfg.AICollectCredentialMethod
	}
	resp, err := s.client.Execute(ctx, alipay.Request{Method: method, BizContent: req.BizContent})
	if err != nil {
		return GatewayResponse{}, err
	}
	return GatewayResponse{Action: "credential_query", HTTPStatus: resp.HTTPStatus, AlipayRaw: resp.RawBody}, nil
}

func (s *Service) ConfirmFulfillment(ctx context.Context, req FulfillmentConfirmRequest) (GatewayResponse, error) {
	method := req.Method
	if method == "" {
		method = s.cfg.AICollectFulfillmentMethod
	}
	resp, err := s.client.Execute(ctx, alipay.Request{Method: method, BizContent: req.BizContent})
	if err != nil {
		return GatewayResponse{}, err
	}
	return GatewayResponse{Action: "fulfillment_confirm", HTTPStatus: resp.HTTPStatus, AlipayRaw: resp.RawBody}, nil
}

func (s *Service) PreparePaidResource(ctx context.Context, req PaidResourceRequest) (GatewayResponse, error) {
	if req.ResourceID == "" {
		return GatewayResponse{}, fmt.Errorf("resource_id is required")
	}
	if req.OutBizNo == "" {
		return GatewayResponse{}, fmt.Errorf("out_biz_no is required")
	}
	if req.Subject == "" {
		return GatewayResponse{}, fmt.Errorf("subject is required")
	}
	if req.Amount == "" {
		return GatewayResponse{}, fmt.Errorf("amount is required")
	}

	bizContent := map[string]interface{}{
		"out_biz_no":  req.OutBizNo,
		"resource_id": req.ResourceID,
		"subject":     req.Subject,
		"amount":      req.Amount,
	}
	if req.AgentID != "" {
		bizContent["agent_id"] = req.AgentID
	}
	if req.Currency != "" {
		bizContent["currency"] = req.Currency
	}
	for k, v := range req.ExtraParams {
		if _, exists := bizContent[k]; !exists {
			bizContent[k] = v
		}
	}

	return s.QueryCredential(ctx, CredentialQueryRequest{BizContent: bizContent})
}
