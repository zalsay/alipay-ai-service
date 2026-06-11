# alipay-ai-service

Golang 服务端，用于给 Agent/智能体场景封装支付宝 **AI 收** 调用能力。

这个项目故意不把 AI 收的 OpenAPI `method` 和 `biz_content` 字段写死：不同商户开通的 AI 收产品、灰度版本和文档页面可能存在差异。服务通过环境变量 `ALIPAY_AI_COLLECT_METHOD` 和请求体中的 `biz_content` 透传，负责统一完成：

- 支付宝 OpenAPI 公共参数组装
- RSA2 签名
- 支付宝响应验签
- 异步通知验签
- Agent 调用侧的幂等请求处理
- 健康检查与 Docker 化部署

## 快速开始

```bash
cp .env.example .env
# 编辑 .env，填入 AppID、私钥、公钥、notify_url、AI 收 method
make run
```

## API

### 健康检查

```bash
curl http://localhost:8080/healthz
```

### 调用 AI 收 OpenAPI

```bash
curl -X POST http://localhost:8080/v1/ai-collect/call \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-001' \
  -d '{
    "method": "replace_with_ai_collect_method_from_opendocs",
    "biz_content": {
      "out_biz_no": "agent-call-001",
      "subject": "agent api call",
      "amount": "0.01"
    }
  }'
```

`method` 可省略，省略时使用环境变量 `ALIPAY_AI_COLLECT_METHOD`。

### 支付宝异步通知

把支付宝开放平台里的异步通知地址配置为：

```text
https://your-domain.example.com/v1/alipay/notify
```

服务会验签，通过后返回 `success`，否则返回 `fail`。

## 环境变量

| 变量 | 说明 |
| --- | --- |
| `SERVER_ADDR` | HTTP 监听地址，默认 `:8080` |
| `ALIPAY_GATEWAY` | 支付宝网关，默认 `https://openapi.alipay.com/gateway.do` |
| `ALIPAY_APP_ID` | 开放平台应用 AppID |
| `ALIPAY_APP_PRIVATE_KEY` | 应用私钥 PEM 文本；优先级高于文件 |
| `ALIPAY_APP_PRIVATE_KEY_FILE` | 应用私钥 PEM 文件路径 |
| `ALIPAY_PUBLIC_KEY` | 支付宝公钥 PEM 文本；优先级高于文件 |
| `ALIPAY_PUBLIC_KEY_FILE` | 支付宝公钥 PEM 文件路径 |
| `ALIPAY_NOTIFY_URL` | 异步通知地址 |
| `ALIPAY_AI_COLLECT_METHOD` | AI 收 OpenAPI method，以开通后文档为准 |
| `ALIPAY_AI_COLLECT_VERSION` | 接口版本，默认 `1.0` |
| `ALIPAY_APP_AUTH_TOKEN` | 可选，服务商代商户调用时使用 |

## 重要说明

1. 不要把私钥提交到仓库。
2. 生产环境必须使用 HTTPS 的 `notify_url`。
3. `biz_content` 的字段必须以你开通后的支付宝 AI 收文档为准。
4. 建议把 `Idempotency-Key` 设置为 Agent 单次意图/任务 ID，避免重复扣款或重复创建交易。
