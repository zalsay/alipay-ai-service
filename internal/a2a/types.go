package a2a

type PaymentNeeded struct {
	Protocol PaymentNeededProtocol `json:"protocol"`
	Method   PaymentNeededMethod   `json:"method"`
}

type PaymentNeededProtocol struct {
	OutTradeNo      string `json:"out_trade_no"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	Network         string `json:"network"`
	ResourceID      string `json:"resource_id"`
	PayBefore       string `json:"pay_before"`
	SellerSignature string `json:"seller_signature"`
	SellerSignType  string `json:"seller_sign_type"`
	SellerUniqueID  string `json:"seller_unique_id"`
}

type PaymentNeededMethod struct {
	SellerName        string `json:"seller_name"`
	SellerID          string `json:"seller_id"`
	SellerAppID       string `json:"seller_app_id"`
	GoodsName         string `json:"goods_name"`
	SellerUniqueIDKey string `json:"seller_unique_id_key"`
	ServiceID         string `json:"service_id"`
}

type PaymentProof struct {
	Protocol PaymentProofProtocol `json:"protocol"`
	Method   PaymentProofMethod   `json:"method"`
}

type PaymentProofProtocol struct {
	PaymentProof string `json:"payment_proof"`
	TradeNo      string `json:"trade_no"`
}

type PaymentProofMethod struct {
	ClientSession string `json:"client_session"`
}

type VerifyResponse struct {
	Action     string                 `json:"action"`
	HTTPStatus int                    `json:"http_status"`
	Active     bool                   `json:"active"`
	TradeNo    string                 `json:"trade_no,omitempty"`
	OutTradeNo string                 `json:"out_trade_no,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	AlipayRaw  map[string]interface{} `json:"alipay_raw,omitempty"`
}

type FulfillmentResponse struct {
	Action     string                 `json:"action"`
	HTTPStatus int                    `json:"http_status"`
	TradeNo    string                 `json:"trade_no,omitempty"`
	AlipayRaw  map[string]interface{} `json:"alipay_raw,omitempty"`
}
