# 支付宝 AI 收服务（Agent / AI 收网关）

[English](README.md)

这是一个面向支付宝 AI 收 / A2A Agent 付费资源场景的 Go 后端服务。

## 功能概览

- 支持 `402 Payment Required` 和 `Payment-Needed` 响应头。
- 提供 `/v1/paid-resource/payment-needed`，方便客户端以 JSON 方式获取支付账单。
- 解析 `Payment-Proof` 请求头，并通过 `alipay.aipay.agent.payment.verify` 验证支付证明。
- 提供 `/v1/paid-resource/check`，在买家支付工具只返回交易号或支付状态时，通过 `alipay.trade.query` 确认支付并解锁资源。
- 支持将支付账单绑定和资源解锁状态持久化到 Postgres，或使用本地文件后端。
- 资源交付后异步调用 `alipay.aipay.agent.fulfillment.confirm` 发送履约确认。
- 使用 RSA2 生成 `Payment-Needed` 中的 `seller_signature`。
- 保留 `/v1/ai-collect/call` 作为兼容旧流程的通用 OpenAPI 代理。
- 保留 `/v1/alipay/notify` 处理支付宝异步通知。

## 支付资源流程

1. Agent 请求付费资源。
2. 卖家服务检查 `Payment-Proof` 请求头。
3. 如果没有有效支付证明，返回 `402 Payment Required`，并附带 `Payment-Needed` 响应头。
4. Agent / 用户在支付宝侧完成支付。
5. Agent 携带 `Payment-Proof` 重试资源请求；如果买家支付工具只返回交易号或支付状态，则调用 `/v1/paid-resource/check`。
6. 卖家服务通过 `alipay.aipay.agent.payment.verify` 验证支付证明，或通过 `alipay.trade.query` 查询交易状态。
7. 如果支付证明有效，或交易状态为 `TRADE_SUCCESS` / `TRADE_FINISHED`，服务持久化解锁状态并返回资源。
8. 通过支付证明路径交付资源后，服务会异步调用 `alipay.aipay.agent.fulfillment.confirm`。

## 主要接口

### 获取支付账单 JSON

```bash
curl -i -X POST https://your-domain.example.com/alipay/v1/paid-resource/payment-needed \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "goods_name": "Agent API Call",
    "amount": "0.01",
    "currency": "CNY"
  }'
```

响应中的 `payment_needed` 等价于标准 402 流程里的 `Payment-Needed` 响应头。

### 请求付费资源

本地路径：

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

公网 Nginx `/alipay` 前缀路径：

```bash
curl -i -X POST https://your-domain.example.com/alipay/v1/paid-resource/prepare \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "goods_name": "Agent API Call",
    "amount": "0.01",
    "currency": "CNY"
  }'
```

没有 `Payment-Proof` 且资源未解锁时，服务返回：

```http
HTTP/1.1 402 Payment Required
Payment-Needed: <base64url encoded bill>
Content-Type: application/json; charset=utf-8
```

响应体：

```json
{
  "error": "Payment Needed",
  "message": "payment is required to access this resource",
  "resource_id": "RES_001"
}
```

支付完成后，如果客户端拿到了 `Payment-Proof`，可重试：

```bash
curl -i -X POST https://your-domain.example.com/alipay/v1/paid-resource/prepare \
  -H 'Content-Type: application/json' \
  -H 'Payment-Proof: <base64 encoded proof from buyer agent>' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001"
  }'
```

### 支付状态确认并解锁

如果买家支付工具确认支付成功，但没有返回可用的 `Payment-Proof`，可调用：

```bash
curl -i -X POST https://your-domain.example.com/alipay/v1/paid-resource/check \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "RES_001",
    "out_trade_no": "ORDER_001",
    "trade_no": "20260701008281113044450000057755"
  }'
```

服务会调用 `alipay.trade.query`。如果支付宝返回 `TRADE_SUCCESS` 或 `TRADE_FINISHED`，会持久化资源解锁状态。之后使用相同 `resource_id` 和 `out_trade_no` 调用 `/v1/paid-resource/prepare`，即使没有 `Payment-Proof` 也会返回 `200 OK`。

## 状态持久化

支付账单和资源解锁记录通过 `PAYMENT_STATE_BACKEND` 持久化。

Docker Compose 推荐使用 Postgres：

```text
PAYMENT_STATE_BACKEND=postgres
PAYMENT_STATE_DB_DSN=<your private postgres DSN>
```

也可以使用本地文件后端：

```text
PAYMENT_STATE_BACKEND=file
PAYMENT_STATE_DB_PATH=/data/payment-state.json
```

本地 `.env`、`secrets/`、本地状态数据库和构建产物已在 `.gitignore` 中忽略。请使用 `.env.example` 作为非敏感配置模板，不要提交真实数据库密码、私钥或令牌。

## Nginx `/alipay` 前缀

Go 服务监听 `/v1/...`。如果需要通过 `/alipay/v1/...` 暴露公网路径，可运行：

```bash
sudo bash scripts/install-nginx-alipay-prefix.sh your-domain.example.com 127.0.0.1:8080
```

映射关系：

```text
/alipay/v1/... -> /v1/...
/alipay/healthz -> /healthz
```

## 其他接口

健康检查：

```bash
curl http://localhost:8080/healthz
curl http://your-domain.example.com/alipay/healthz
```

手动发送履约确认：

```bash
curl -i -X POST http://localhost:8080/v1/ai-collect/fulfillment/confirm \
  -H 'Content-Type: application/json' \
  -d '{"biz_content":{"trade_no":"20260324008281172041220000012182"}}'
```

兼容旧流程的通用 OpenAPI 代理：

```text
POST /v1/ai-collect/call
POST /alipay/v1/ai-collect/call
```

支付宝异步通知：

```text
POST /v1/alipay/notify
POST /alipay/v1/alipay/notify
```

## 环境变量

| 变量 | 说明 |
| --- | --- |
| `SERVER_ADDR` | HTTP 监听地址，默认 `:8080` |
| `ALIPAY_GATEWAY` | 支付宝 OpenAPI 网关地址 |
| `ALIPAY_APP_ID` | 支付宝开放平台应用 AppID |
| `ALIPAY_APP_PRIVATE_KEY` / `ALIPAY_APP_PRIVATE_KEY_FILE` | 应用私钥 PEM 内容或文件路径 |
| `ALIPAY_PUBLIC_KEY` / `ALIPAY_PUBLIC_KEY_FILE` | 支付宝公钥 PEM 内容或文件路径 |
| `ALIPAY_NOTIFY_URL` | 支付宝异步通知回调地址 |
| `ALIPAY_AI_COLLECT_METHOD` | 可选旧版通用 AI 收接口方法名 |
| `ALIPAY_AI_COLLECT_CREDENTIAL_METHOD` | 可选凭证查询接口方法名，未设置时默认使用 `ALIPAY_AI_COLLECT_METHOD` |
| `ALIPAY_PAYMENT_VERIFY_METHOD` | 支付证明验证接口方法名 |
| `ALIPAY_AI_COLLECT_FULFILLMENT_METHOD` | 履约确认接口方法名 |
| `ALIPAY_PAYMENT_STATUS_QUERY_METHOD` | `/v1/paid-resource/check` 使用的支付状态查询接口方法名 |
| `PAYMENT_STATE_BACKEND` | 支付状态持久化后端：`postgres` 或 `file` |
| `PAYMENT_STATE_DB_DSN` | Postgres 后端 DSN |
| `PAYMENT_STATE_DB_PATH` | file 后端状态文件路径 |
| `ALIPAY_SELLER_ID` | 卖家支付宝用户 ID / 2088 ID |
| `ALIPAY_SELLER_NAME` | 卖家展示名称 |
| `ALIPAY_SELLER_APP_ID` | 卖家应用 AppID，未设置时默认使用 `ALIPAY_APP_ID` |
| `ALIPAY_SELLER_UNIQUE_ID_KEY` | Payment-Needed 中卖家唯一标识字段名 |
| `ALIPAY_SERVICE_ID` | 支付宝 AI 收服务 ID |
| `ALIPAY_DEFAULT_GOODS_NAME` | 默认商品名称 |
| `ALIPAY_DEFAULT_AMOUNT` | 默认支付金额 |
| `ALIPAY_DEFAULT_CURRENCY` | 默认币种 |
| `ALIPAY_PAYMENT_NETWORK` | Payment-Needed 中的支付网络标识 |
| `ALIPAY_PAYMENT_PROOF_TTL_MINUTES` | 支付账单有效期，单位分钟 |
| `ALIPAY_AI_COLLECT_VERSION` | 支付宝接口版本 |
| `ALIPAY_APP_AUTH_TOKEN` | 可选服务商 / ISV 模式 app_auth_token |

## 构建与运行

```bash
make build
./dist/alipay-ai-service
```

Docker：

```bash
docker compose up --build -d
```

## 注意事项

1. 根据当前支付宝文档，AI 收暂不支持沙箱调试。
2. 不要提交私钥、数据库密码、`.env` 或其他敏感文件。
3. `Payment-Needed` 使用 Base64URL 编码。
4. `Payment-Proof` 由买家 Agent 在支付完成后提供。
5. 当前服务返回的是占位付费内容，请将 `HandlePaidResource` 中的 `content` 替换为真实资源交付逻辑。
