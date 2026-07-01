# Alipay AI Collect Service (Agent / AI 收网关)

[中文](README_CN.md)

Golang backend service for Alipay AI 收 / A2A Agent paid-resource scenarios.

Official Alipay AI 收 product page:
https://b.alipay.com/page/product-workspace/product-detail/I1080300001000160457/newProductList?

This project implements the documented flow:

1. Agent requests a paid resource.
2. Seller server checks the `Payment-Proof` request header.
3. If no valid proof exists, seller server returns `402 Payment Required` with a `Payment-Needed` header.
4. Agent/user completes payment with Alipay.
5. Agent retries the resource request with `Payment-Proof`, or calls `/v1/paid-resource/check` when the buyer payment tool only returns payment status / trade number.
6. Seller server verifies `Payment-Proof` through `alipay.aipay.agent.payment.verify`, or checks trade status through `alipay.trade.query`.
7. If payment is active, `TRADE_SUCCESS`, or `TRADE_FINISHED`, seller server persists the unlock state and returns the resource.
8. After resource delivery, seller server asynchronously calls `alipay.aipay.agent.fulfillment.confirm` when the proof-verification path provides a trade number.

## Features

- Implements `402 Payment Required` + `Payment-Needed` header.
- Provides `/v1/paid-resource/payment-needed` for clients that need bill JSON instead of a response header.
- Parses `Payment-Proof` header.
- Calls `alipay.aipay.agent.payment.verify` to verify paid credentials.
- Provides `/v1/paid-resource/check` to unlock a paid resource through `alipay.trade.query`.
- Persists payment bill bindings and resource unlock records through Postgres or a local file backend.
- Calls `alipay.aipay.agent.fulfillment.confirm` after resource delivery.
- Builds local `seller_signature` for the Payment-Needed bill using RSA2.
- Keeps `/v1/ai-collect/call` as a backward-compatible raw OpenAPI proxy.
- Keeps `/v1/alipay/notify` for compatibility with asynchronous Alipay notifications.

## Main Endpoint

### Client-Friendly Payment Bill

If a buyer client cannot conveniently read the `Payment-Needed` response header from a
`402 Payment Required` response, it can request the same bill as JSON:

```bash
curl -i -X POST https://your-domain.example.com/alipay/v1/paid-resource/payment-needed \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "service_id": "SERVICE_001",
    "goods_name": "Agent API Call",
    "amount": "0.01",
    "currency": "CNY"
  }'
```

The response includes `payment_needed`, which is the value the buyer Agent should
treat exactly like the `Payment-Needed` header, and `bill`, the decoded JSON.
The client must provide `service_id`, `goods_name`, and `amount` in the request;
the service does not read those values from environment variables.

### Paid Resource Access

Local service path:

```bash
curl -i -X POST http://localhost:8080/v1/paid-resource/prepare \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "service_id": "SERVICE_001",
    "goods_name": "Agent API Call",
    "amount": "0.01",
    "currency": "CNY"
  }'
```

Public Nginx path with `/alipay` prefix:

```bash
curl -i -X POST https://your-domain.example.com/alipay/v1/paid-resource/prepare \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "service_id": "SERVICE_001",
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
curl -i -X POST https://your-domain.example.com/alipay/v1/paid-resource/prepare \
  -H 'Content-Type: application/json' \
  -H 'Payment-Proof: <base64 encoded proof from buyer agent>' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "service_id": "SERVICE_001",
    "goods_name": "Agent API Call",
    "amount": "0.01",
    "currency": "CNY"
  }'
```

If `alipay.aipay.agent.payment.verify` returns `active=true`, the service returns `200 OK` and sends fulfillment confirmation in the background.

### Payment Status Check And Unlock

If the buyer payment tool confirms payment by `out_trade_no` or `trade_no` but cannot
return a usable `Payment-Proof`, confirm and unlock the resource with:

```bash
curl -i -X POST https://your-domain.example.com/alipay/v1/paid-resource/check \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "trade_no": "20260701008281113044450000057755"
  }'
```

The service calls `alipay.trade.query`. If Alipay returns `TRADE_SUCCESS` or
`TRADE_FINISHED`, the local resource state is marked unlocked. A later
`POST /v1/paid-resource/prepare` with the same `resource_id` and `out_trade_no`
returns `200 OK` even without `Payment-Proof`.

Payment bills and unlock records are persisted through `PAYMENT_STATE_BACKEND`.
For Docker Compose, pass a Postgres DSN through your local `.env` file. Do not
commit real database credentials:

```text
PAYMENT_STATE_BACKEND=postgres
PAYMENT_STATE_DB_DSN=<your private postgres DSN>
```

The file backend remains available with `PAYMENT_STATE_BACKEND=file` and
`PAYMENT_STATE_DB_PATH=/data/payment-state.json`.

The local `.env`, `secrets/`, and local state database files are ignored by git.
Use `.env.example` as the non-secret template.

## Nginx `/alipay` Prefix

The Go service listens on `/v1/...`. To expose public URLs under `/alipay/v1/...`, run:

```bash
sudo bash scripts/install-nginx-alipay-prefix.sh your-domain.example.com 127.0.0.1:8080
```

This creates:

```text
/alipay/v1/... -> /v1/...
/alipay/healthz -> /healthz
```

Test after installation:

```bash
curl -i http://your-domain.example.com/alipay/healthz
curl -i -X POST http://your-domain.example.com/alipay/v1/paid-resource/prepare \
  -H 'Content-Type: application/json' \
  -d '{"resource_id":"RES_001","out_trade_no":"ORDER_001","service_id":"SERVICE_001","goods_name":"Agent API Call","amount":"0.01","currency":"CNY"}'
```

For HTTPS, install a certificate with your preferred tool, such as certbot, and keep the same `/alipay/` location rules.

## Other Endpoints

### Health check

```bash
curl http://localhost:8080/healthz
curl http://your-domain.example.com/alipay/healthz
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

With Nginx prefix:

```bash
curl -i -X POST https://your-domain.example.com/alipay/v1/ai-collect/fulfillment/confirm \
  -H 'Content-Type: application/json' \
  -d '{"biz_content":{"trade_no":"20260324008281172041220000012182"}}'
```

### Backward-compatible OpenAPI Proxy

```bash
POST /v1/ai-collect/call
POST /alipay/v1/ai-collect/call
```

### Alipay Async Notification

```text
POST /v1/alipay/notify
POST /alipay/v1/alipay/notify
```

## Environment Variables

| Variable | Description |
| --- | --- |
| `SERVER_ADDR` | HTTP listen address, default `:8080` |
| `ALIPAY_GATEWAY` | Alipay OpenAPI Gateway, default `https://openapi.alipay.com/gateway.do` |
| `ALIPAY_APP_ID` | Optional Open Platform AppID |
| `ALIPAY_APP_PRIVATE_KEY` / `ALIPAY_APP_PRIVATE_KEY_FILE` | Application private key PEM |
| `ALIPAY_PUBLIC_KEY` / `ALIPAY_PUBLIC_KEY_FILE` | Alipay public key PEM |
| `ALIPAY_AI_COLLECT_METHOD` | Optional legacy generic method used by `/v1/ai-collect/call` |
| `ALIPAY_AI_COLLECT_CREDENTIAL_METHOD` | Optional credential query method; defaults to `ALIPAY_AI_COLLECT_METHOD` |
| `ALIPAY_PAYMENT_VERIFY_METHOD` | Default `alipay.aipay.agent.payment.verify` |
| `ALIPAY_AI_COLLECT_FULFILLMENT_METHOD` | Default `alipay.aipay.agent.fulfillment.confirm` |
| `ALIPAY_PAYMENT_STATUS_QUERY_METHOD` | Default `alipay.trade.query`; used by `/v1/paid-resource/check` |
| `PAYMENT_STATE_BACKEND` | Payment state backend: `postgres` or `file`; Docker default `postgres` |
| `PAYMENT_STATE_DB_DSN` | Postgres DSN when `PAYMENT_STATE_BACKEND=postgres` |
| `PAYMENT_STATE_DB_PATH` | File backend state path, default `/data/payment-state.json` |
| `ALIPAY_SELLER_ID` | Seller userId / 2088 ID |
| `ALIPAY_SELLER_NAME` | Seller display name |
| `ALIPAY_SELLER_APP_ID` | Seller app id; defaults to `ALIPAY_APP_ID` |
| `ALIPAY_SELLER_UNIQUE_ID_KEY` | Default `seller_id` |
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
