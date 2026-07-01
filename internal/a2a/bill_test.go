package a2a

import (
	"strings"
	"testing"

	"github.com/zalsay/alipay-ai-service/internal/config"
)

func TestBuildPaymentNeededRequiresClientBillFields(t *testing.T) {
	cfg := config.Config{
		SellerID:               "2088102028041105",
		SellerName:             "测试商家",
		SellerAppID:            "2021000000000000",
		SellerUniqueIDKey:      "seller_id",
		DefaultCurrency:        "CNY",
		PaymentNetwork:         "alipay-a2a-prod",
		PaymentProofTTLMinutes: "15",
		AppPrivateKey:          "unused by this test",
	}

	tests := []struct {
		name string
		in   BillInput
		want string
	}{
		{
			name: "service_id",
			in: BillInput{
				OutTradeNo: "ORDER_001",
				ResourceID: "RES_001",
				GoodsName:  "Agent API Call",
				Amount:     "0.01",
			},
			want: "service_id is required",
		},
		{
			name: "goods_name",
			in: BillInput{
				OutTradeNo: "ORDER_001",
				ResourceID: "RES_001",
				ServiceID:  "SERVICE_001",
				Amount:     "0.01",
			},
			want: "goods_name is required",
		},
		{
			name: "amount",
			in: BillInput{
				OutTradeNo: "ORDER_001",
				ResourceID: "RES_001",
				ServiceID:  "SERVICE_001",
				GoodsName:  "Agent API Call",
			},
			want: "amount is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := BuildPaymentNeeded(cfg, tt.in)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %q", tt.want, err.Error())
			}
		})
	}
}
