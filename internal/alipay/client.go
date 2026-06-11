package alipay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zalsay/alipay-ai-service/internal/config"
	"github.com/zalsay/alipay-ai-service/internal/utils"
)

type Client struct {
	cfg        config.Config
	httpClient *http.Client
}

type Request struct {
	Method     string                 `json:"method"`
	BizContent map[string]interface{} `json:"biz_content"`
}

type Response struct {
	HTTPStatus int             `json:"http_status"`
	RawBody    json.RawMessage `json:"raw_body"`
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) Execute(ctx context.Context, req Request) (Response, error) {
	method := strings.TrimSpace(req.Method)
	if method == "" {
		return Response{}, fmt.Errorf("alipay method is required")
	}
	if req.BizContent == nil {
		return Response{}, fmt.Errorf("biz_content is required")
	}

	bizContentBytes, err := json.Marshal(req.BizContent)
	if err != nil {
		return Response{}, fmt.Errorf("marshal biz_content: %w", err)
	}

	params := map[string]string{
		"app_id":      c.cfg.AppID,
		"method":      method,
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     c.cfg.AICollectVersion,
		"biz_content": string(bizContentBytes),
	}
	if c.cfg.NotifyURL != "" {
		params["notify_url"] = c.cfg.NotifyURL
	}
	if c.cfg.AppAuthToken != "" {
		params["app_auth_token"] = c.cfg.AppAuthToken
	}

	sign, err := utils.SignRSA2(params, c.cfg.AppPrivateKey)
	if err != nil {
		return Response{}, fmt.Errorf("sign alipay request: %w", err)
	}
	params["sign"] = sign

	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Gateway, strings.NewReader(form.Encode()))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("call alipay openapi: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	return Response{
		HTTPStatus: resp.StatusCode,
		RawBody:    json.RawMessage(body),
	}, nil
}
