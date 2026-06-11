# Alipay AI Collect Service (Agent / AI 收网关)

Golang backend service for Alipay AI Collect API, designed for Agent scenarios.

## Features

- AI 收网关模式：支持 Agent 调用付费资源
- 支持凭证查询、履约确认、异步通知
- 原有通用 OpenAPI 代理 `/v1/ai-collect/call` 保留兼容
- 支持静态二进制构建和 Docker 容器化部署
- 支持生产级 RSA2 签名与异步通知验签

## API Endpoints

### Health check

```bash
curl http://localhost:8080/healthz
```

### AI Collect - Credential Query

```bash
POST /v1/ai-collect/credential/query
Content-Type: application/json

{
  "biz_content": {
    "out_biz_no": "agent-call-001",
    "resource_id": "resource-001"
  }
}
```

### AI Collect - Fulfillment Confirm

```bash
POST /v1/ai-collect/fulfillment/confirm
Content-Type: application/json

{
  "biz_content": {
    "out_biz_no": "agent-call-001",
    "trade_no": "2023061122001401234567890"
  }
}
```

### Paid Resource Entry for Agent

```bash
POST /v1/paid-resource/prepare
Content-Type: application/json

{
  "resource_id": "resource-001",
  "out_biz_no": "agent-call-001",
  "subject": "Agent API Call",
  "amount": "0.01"
}
```

### Alipay Async Notification

Configure in Alipay Open Platform:
```
https://your-domain.example.com/v1/alipay/notify
```
The service validates the signature and returns `success` on success, otherwise `fail`.

### Backward-compatible OpenAPI Proxy

```bash
POST /v1/ai-collect/call
```

This endpoint can still be used to call arbitrary AI Collect OpenAPI methods.

## Environment Variables

| Variable | Description |
| --- | --- |
| SERVER_ADDR | HTTP listen address, default `:8080` |
| ALIPAY_GATEWAY | Alipay OpenAPI Gateway, default `https://openapi.alipay.com/gateway.do` |
| ALIPAY_APP_ID | Open Platform AppID |
| ALIPAY_APP_PRIVATE_KEY / ALIPAY_APP_PRIVATE_KEY_FILE | Application private key PEM |
| ALIPAY_PUBLIC_KEY / ALIPAY_PUBLIC_KEY_FILE | Alipay public key PEM |
| ALIPAY_NOTIFY_URL | Asynchronous notification callback URL |
| ALIPAY_AI_COLLECT_CREDENTIAL_METHOD | AI Collect credential query method |
| ALIPAY_AI_COLLECT_FULFILLMENT_METHOD | AI Collect fulfillment confirmation method |
| ALIPAY_AI_COLLECT_VERSION | API version, default `1.0` |
| ALIPAY_APP_AUTH_TOKEN | Optional, service provider token for acting on behalf of merchant |

## Build & Run

```bash
make build        # Build static binary
./dist/alipay-ai-service  # Run locally
```

### Docker

```bash
docker compose up --build -d
```

### Setup Script

```bash
chmod +x setup.sh
./setup.sh  # Build and restart docker-compose
```

## Notes

1. Always use the public key provided by Alipay for production.
2. `biz_content` fields must follow the fields defined in your Alipay Open Platform AI Collect documentation.
3. For Agent calls, always set `out_biz_no` uniquely to ensure idempotency.
4. New AI Collect gateway endpoints:
   - `/v1/ai-collect/credential/query`
   - `/v1/ai-collect/fulfillment/confirm`
   - `/v1/paid-resource/prepare`