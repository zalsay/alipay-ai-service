# Alipay AI Collect Service (Agent / AI 收网关)

Golang backend service for Alipay AI 收 / A2A Agent paid-resource scenarios.

This project implements the documented flow:

1. Agent requests a paid resource.
2. Seller server checks the `Payment-Proof` request header.
3. If no valid proof exists, seller server returns `402 Payment Required` with a `Payment-Needed` header.
4. Agent/user completes payment with Alipay.
5. Agent retries the resource request with `Payment-Proof`.
6. Seller server calls `alipay.aipay.agent.payment.verify`.
7. If `active=true`, seller server returns the resource and asynchronously calls `alipay.aipay.agent.fulfillment.confirm`.

## Features

- Implements `402 Payment Required` + `Payment-Needed` header.
- Parses `Payment-Proof` header.
- Calls `alipay.aipay.agent.payment.verify` to verify paid credentials.
- Calls `alipay.aipay.agent.fulfillment.confirm` after resource delivery.
- Builds local `seller_signature` for the Payment-Needed bill using RSA2.
- Keeps `/v1/ai-collect/call` as a backward-compatible raw OpenAPI proxy.
- Keeps `/v1/alipay/notify` for compatibility with asynchronous Alipay notifications.

## Main Endpoint

### Paid Resource Access

```bash
curl -i -X POST http://localhost:8080/v1/paid-resource/prepare \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "goods_name": "Agent API Call",
    "amount": "0.01",
    "currency": "CNY"
  }'
```

Without `Payment-Proof`, the expected response is:

```http
HTTP/1.1 402 Payment Required
Payment-Needed: <base64url encoded bill>
Content-Type: application/json; charset=utf-8
```

Response body:

```json
{
  "error": "Payment Needed",
  "message": "payment is required to access this resource",
  "resource_id": "RES_001"
}
```

The `Payment-Needed` header decodes to:

```json
{
  "protocol": {
    "out_trade_no": "ORDER_001",
    "amount": "0.01",
    "currency": "CNY",
    "network": "alipay-a2a-prod",
    "resource_id": "RES_001",
    "pay_before": "2026-03-25T12:00:00+08:00",
    "seller_signature": "...",
    "seller_sign_type": "RSA2",
    "seller_unique_id": "2088..."
  },
  "method": {
    "seller_name": "测试商家",
    "seller_id": "2088...",
    "seller_app_id": "2019...",
    "goods_name": "Agent API Call",
    "seller_unique_id_key": "seller_id",
    "service_id": "..."
  }
}
```

After the Agent/user pays, retry with `Payment-Proof`:

```bash
curl -i -X POST http://localhost:8080/v1/paid-resource/prepare \
  -H 'Content-Type: application/json' \
  -H 'Payment-Proof: <base64 encoded proof from buyer agent>' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "goods_name": "Agent API Call",
    "amount": "0.01",
    "currency": "CNY"
  }'
```

If `alipay.aipay.agent.payment.verify` returns `active=true`, the service returns `200 OK` and sends fulfillment confirmation in the background.

## Other Endpoints

### Health check

```bash
curl http://localhost:8080/healthz
```

### Manual Fulfillment Confirm

```bash
curl -i -X POST http://localhost:8080/v1/ai-collect/fulfillment/confirm \
  -H 'Content-Type: application/json' \
  -d '{
    "biz_content": {
      "trade_no": "20260324008281172041220000012182"
    }
  }'
```

### Backward-compatible OpenAPI Proxy

```bash
POST /v1/ai-collect/call
```

### Alipay Async Notification

```text
POST /v1/alipay/notify
```

## Environment Variables

| Variable | Description |
| --- | --- |
| `SERVER_ADDR` | HTTP listen address, default `:8080` |
| `ALIPAY_GATEWAY` | Alipay OpenAPI Gateway, default `https://openapi.alipay.com/gateway.do` |
| `ALIPAY_APP_ID` | Open Platform AppID |
| `ALIPAY_APP_PRIVATE_KEY` / `ALIPAY_APP_PRIVATE_KEY_FILE` | Application private key PEM |
| `ALIPAY_PUBLIC_KEY` / `ALIPAY_PUBLIC_KEY_FILE` | Alipay public key PEM |
| `ALIPAY_PAYMENT_VERIFY_METHOD` | Default `alipay.aipay.agent.payment.verify` |
| `ALIPAY_AI_COLLECT_FULFILLMENT_METHOD` | Default `alipay.aipay.agent.fulfillment.confirm` |
| `ALIPAY_SELLER_ID` | Seller userId / 2088 ID |
| `ALIPAY_SELLER_NAME` | Seller display name |
| `ALIPAY_SELLER_APP_ID` | Seller app id; defaults to `ALIPAY_APP_ID` |
| `ALIPAY_SELLER_UNIQUE_ID_KEY` | Default `seller_id` |
| `ALIPAY_SERVICE_ID` | Service ID from Alipay AI 收 documentation |
| `ALIPAY_DEFAULT_GOODS_NAME` | Default goods title |
| `ALIPAY_DEFAULT_AMOUNT` | Default amount, e.g. `0.01` |
| `ALIPAY_DEFAULT_CURRENCY` | Default `CNY` |
| `ALIPAY_PAYMENT_NETWORK` | Default `alipay-a2a-prod` |
| `ALIPAY_PAYMENT_PROOF_TTL_MINUTES` | Payment bill expiration minutes, default `15` |
| `ALIPAY_AI_COLLECT_VERSION` | API version, default `1.0` |
| `ALIPAY_APP_AUTH_TOKEN` | Optional service provider token |

## Build & Run

```bash
make build
./dist/alipay-ai-service
```

### Docker

```bash
docker compose up --build -d
```

### Setup Script

```bash
chmod +x setup.sh
./setup.sh
```

## Notes

1. AI 收 currently does not support sandbox debugging according to the provided Alipay documentation.
2. Do not commit private keys to the repository.
3. `Payment-Needed` uses Base64URL encoding.
4. `Payment-Proof` is provided by the buyer Agent after payment.
5. This project currently returns placeholder paid content. Replace the `content` field in `HandlePaidResource` with your real resource delivery logic.
