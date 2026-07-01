---
name: alipay-ai-buyer-agent
description: Use when implementing or guiding a buyer Agent/client or merchant service Skill that calls this Alipay AI Collect service, obtains payment bill data, hands off to Alipay payment handling when an alipay_* payment link exists, submits Payment-Proof, and handles paid-resource responses.
---

# Alipay AI Buyer Agent

This skill guides a buyer Agent/client through the current Alipay AI Collect paid-resource flow exposed by this service.

It also records the official Alipay service Skill principle: the merchant service Skill owns the business/order flow, and the Alipay payment processing skill owns payment. If a real Alipay compressed payment link is available, it must be passed to the Alipay payment processing skill exactly as returned.

## Service Endpoints

Base URL:

```text
https://go-api.meetlife.com.cn/alipay
```

Client-friendly bill endpoint, useful for debugging or clients that cannot access response headers:

```http
POST /v1/paid-resource/payment-needed
```

Payment status confirmation endpoint, useful after a buyer payment tool confirms payment but cannot return a usable `Payment-Proof`:

```http
POST /v1/paid-resource/check
```

Standard paid-resource endpoint:

```http
POST /v1/paid-resource/prepare
```

Current default bill parameters:

```json
{
  "resource_id": "MTF_SINGLE_STOCK_001",
  "goods_name": "mtf 单股",
  "amount": "0.99",
  "currency": "CNY",
  "seller_name": "张颖",
  "seller_id": "2088102028041105",
  "seller_app_id": "2021006119631624",
  "service_id": "API_49744029FAB14B42"
}
```

## Workflow

1. Execute the business flow up to order confirmation: identify the resource, amount, goods title, and buyer intent.
2. Generate a unique `out_trade_no` for the attempted purchase. Reuse the same value when retrying with `Payment-Proof`.
   - The buyer client must send `service_id`, `goods_name`, and `amount`; the seller service no longer fills these from environment defaults.
3. If the runtime has the official `alipay-payment-skill`, initiate AI pay through the standard 402 protocol:
   - Call `POST /v1/paid-resource/prepare` without `Payment-Proof`.
   - Preserve the original request URL, method, body, and headers.
   - The service should return `402 Payment Required` with a `Payment-Needed` header.
   - Let `alipay-payment-skill` handle the 402 response. It will save `Payment-Needed`, run buyer payment, query status, and return the resource when payment succeeds.
4. If the runtime cannot use the official 402 payment skill, call `POST /v1/paid-resource/payment-needed` to get a JSON response containing:
   - `payment_needed`: Base64URL encoded bill. Treat this exactly like the `Payment-Needed` header.
   - `bill`: decoded bill JSON for display and validation.
5. Check whether the order/payment creation result contains a non-empty `paymentLink` whose full value starts with `alipay_`.
   - If yes, immediately load/call the Alipay payment processing skill with the complete original `paymentLink`. Do not open, rewrite, compress, truncate, or display the link.
   - If no, continue with the AI Collect `payment_needed` flow. Do not pretend `payment_needed` is an `alipay_*` short link.
6. After payment succeeds, obtain the buyer-side proof payload:

```json
{
  "protocol": {
    "payment_proof": "<payment proof from buyer payment flow>",
    "trade_no": "<Alipay trade number>"
  },
  "method": {
    "client_session": "<buyer client session>"
  }
}
```

7. Base64 encode the proof JSON and call `POST /v1/paid-resource/prepare` with:
   - `Payment-Proof: <base64 proof JSON>`
   - the same `resource_id` and `out_trade_no` used for the bill request.
8. If the buyer payment tool confirms payment by `trade_no` / `out_trade_no` but does not return a usable `Payment-Proof`, call `POST /v1/paid-resource/check` with `resource_id`, the original `out_trade_no`, and the Alipay `trade_no`. If Alipay returns `TRADE_SUCCESS` or `TRADE_FINISHED`, the seller service unlocks the resource locally.
9. After a successful `check`, call `POST /v1/paid-resource/prepare` again with the same `resource_id` and `out_trade_no`. It should return `200` from local unlocked state without requiring `Payment-Proof`.
10. If the response is `200`, show/use the paid resource content.
11. If the response is `402`, the payment proof is missing, invalid, expired, inactive, or the payment status check did not finish. Start the payment flow again with a new bill or ask the user to retry payment.
12. If the response is `403`, the proof or order number is valid for a different resource; do not show paid content.

Official handoff decision:

```text
IF paymentLink exists and starts with "alipay_"
  THEN call the Alipay payment processing skill with the exact paymentLink
  ELSE use the AI Collect Payment-Needed flow and wait for Payment-Proof
```

## Request Examples

Official 402 initiation request for AI pay:

```bash
curl -k -i -X POST 'https://go-api.meetlife.com.cn/alipay/v1/paid-resource/prepare' \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "MTF_SINGLE_STOCK_001",
    "out_trade_no": "ORDER_CLIENT_001",
    "service_id": "API_49744029FAB14B42",
    "goods_name": "mtf 单股",
    "amount": "0.99",
    "currency": "CNY"
  }'
```

Expected response for official payment skill handoff:

```http
HTTP/1.1 402 Payment Required
Payment-Needed: <base64url bill>
```

When the official payment skill is available, hand this 402 response to it with the original request metadata: URL, method `POST`, request body, and `Content-Type: application/json` header.

Get buyer payment parameters:

```bash
curl -k -sS -X POST 'https://go-api.meetlife.com.cn/alipay/v1/paid-resource/payment-needed' \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "MTF_SINGLE_STOCK_001",
    "out_trade_no": "ORDER_CLIENT_001",
    "service_id": "API_49744029FAB14B42",
    "goods_name": "mtf 单股",
    "amount": "0.99",
    "currency": "CNY"
  }'
```

Expected response shape:

```json
{
  "payment_needed": "<base64url bill>",
  "payment_header_name": "Payment-Needed",
  "bill": {
    "protocol": {
      "out_trade_no": "ORDER_CLIENT_001",
      "amount": "0.99",
      "currency": "CNY",
      "resource_id": "MTF_SINGLE_STOCK_001",
      "pay_before": "<ISO8601 time>",
      "seller_signature": "<RSA2 signature>",
      "seller_sign_type": "RSA2",
      "seller_unique_id": "2088102028041105"
    },
    "method": {
      "seller_name": "张颖",
      "seller_id": "2088102028041105",
      "seller_app_id": "2021006119631624",
      "goods_name": "mtf 单股",
      "seller_unique_id_key": "seller_id",
      "service_id": "API_49744029FAB14B42"
    }
  }
}
```

If a future order endpoint returns an official payment link, the response may include:

```json
{
  "paymentLink": "alipay_<complete compressed payment link>"
}
```

When `paymentLink` is non-empty, call the Alipay payment processing skill immediately and pass this exact value as the payment link. Do not pass `payment_needed` as `paymentLink`; it is a Base64URL bill, not an `alipay_*` short link.

Submit proof after buyer payment:

```bash
PAYMENT_PROOF_HEADER='<base64 proof JSON>'

curl -k -i -X POST 'https://go-api.meetlife.com.cn/alipay/v1/paid-resource/prepare' \
  -H 'Content-Type: application/json' \
  -H "Payment-Proof: ${PAYMENT_PROOF_HEADER}" \
  -d '{
    "resource_id": "MTF_SINGLE_STOCK_001",
    "out_trade_no": "ORDER_CLIENT_001",
    "service_id": "API_49744029FAB14B42",
    "goods_name": "mtf 单股",
    "amount": "0.99",
    "currency": "CNY"
  }'
```

Confirm and unlock by payment status after buyer payment:

```bash
curl -k -i -X POST 'https://go-api.meetlife.com.cn/alipay/v1/paid-resource/check' \
  -H 'Content-Type: application/json' \
  -d '{
    "resource_id": "MTF_SINGLE_STOCK_001",
    "out_trade_no": "ORDER_CLIENT_001",
    "trade_no": "20260701008281113044450000057755"
  }'
```

Expected successful response:

```json
{
  "status": "unlocked",
  "unlocked": true,
  "record": {
    "resource_id": "MTF_SINGLE_STOCK_001",
    "out_trade_no": "ORDER_CLIENT_001",
    "trade_no": "20260701008281113044450000057755",
    "trade_status": "TRADE_FINISHED"
  }
}
```

## Implementation Notes

- The buyer Agent must not invent `payment_proof`, `trade_no`, or `client_session`; they must come from the buyer-side payment flow.
- Official Alipay service Skill handoff expects a complete `alipay_*` payment short link. Preserve it byte-for-byte.
- Current service response provides `payment_needed`, not `paymentLink`; use it only with an AI Collect/A2A-capable buyer payment flow.
- The seller service verifies `Payment-Proof` by calling `alipay.aipay.agent.payment.verify`.
- If `Payment-Proof` is unavailable but Alipay payment status is confirmed, the seller service can unlock through `POST /v1/paid-resource/check`, which calls `alipay.trade.query`.
- On successful verification, the seller service asynchronously calls `alipay.aipay.agent.fulfillment.confirm`.
- The standard protocol path is still `POST /v1/paid-resource/prepare` without `Payment-Proof`, which returns `402 Payment Required` and a `Payment-Needed` header. Prefer `/payment-needed` when the client needs JSON parameters directly.
