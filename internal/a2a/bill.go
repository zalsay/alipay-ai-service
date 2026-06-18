package a2a

import (
	"fmt"
	"strconv"
	"time"

	"github.com/zalsay/alipay-ai-service/internal/config"
	"github.com/zalsay/alipay-ai-service/internal/utils"
)

type BillInput struct {
	OutTradeNo string
	ResourceID string
	GoodsName  string
	Amount     string
	Currency   string
}

func BuildPaymentNeeded(cfg config.Config, in BillInput) (PaymentNeeded, string, error) {
	if in.OutTradeNo == "" {
		return PaymentNeeded{}, "", fmt.Errorf("out_trade_no is required")
	}
	if in.ResourceID == "" {
		return PaymentNeeded{}, "", fmt.Errorf("resource_id is required")
	}
	if cfg.SellerID == "" {
		return PaymentNeeded{}, "", fmt.Errorf("ALIPAY_SELLER_ID is required")
	}
	if cfg.SellerName == "" {
		return PaymentNeeded{}, "", fmt.Errorf("ALIPAY_SELLER_NAME is required")
	}
	if cfg.ServiceID == "" {
		return PaymentNeeded{}, "", fmt.Errorf("ALIPAY_SERVICE_ID is required")
	}

	amount := in.Amount
	if amount == "" {
		amount = cfg.DefaultAmount
	}
	currency := in.Currency
	if currency == "" {
		currency = cfg.DefaultCurrency
	}
	goodsName := in.GoodsName
	if goodsName == "" {
		goodsName = cfg.DefaultGoodsName
	}

	ttlMinutes, err := strconv.Atoi(cfg.PaymentProofTTLMinutes)
	if err != nil || ttlMinutes <= 0 {
		ttlMinutes = 15
	}
	payBefore := time.Now().Add(time.Duration(ttlMinutes) * time.Minute).Format(time.RFC3339)

	signParams := map[string]string{
		"amount":       amount,
		"currency":     currency,
		"goods_name":   goodsName,
		"out_trade_no": in.OutTradeNo,
		"pay_before":   payBefore,
		"resource_id":  in.ResourceID,
		"seller_id":    cfg.SellerID,
		"service_id":   cfg.ServiceID,
	}
	sellerSignature, err := utils.SignRSA2(signParams, cfg.AppPrivateKey)
	if err != nil {
		return PaymentNeeded{}, "", fmt.Errorf("sign seller payload: %w", err)
	}

	bill := PaymentNeeded{
		Protocol: PaymentNeededProtocol{
			OutTradeNo:      in.OutTradeNo,
			Amount:          amount,
			Currency:        currency,
			Network:         cfg.PaymentNetwork,
			ResourceID:      in.ResourceID,
			PayBefore:       payBefore,
			SellerSignature: sellerSignature,
			SellerSignType:  "RSA2",
			SellerUniqueID:  cfg.SellerID,
		},
		Method: PaymentNeededMethod{
			SellerName:        cfg.SellerName,
			SellerID:          cfg.SellerID,
			SellerAppID:       cfg.SellerAppID,
			GoodsName:         goodsName,
			SellerUniqueIDKey: cfg.SellerUniqueIDKey,
			ServiceID:         cfg.ServiceID,
		},
	}

	encoded, err := EncodeBase64URLJSON(bill)
	if err != nil {
		return PaymentNeeded{}, "", err
	}
	return bill, encoded, nil
}
