package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalsay/alipay-ai-service/internal/payment"
)

func TestPaidResourceUsesUnlockedState(t *testing.T) {
	useTempPaymentStore(t)

	if _, err := payment.MarkUnlocked(payment.UnlockRecord{
		ResourceID: "MTF_SINGLE_STOCK_001",
		OutTradeNo: "ORDER_001",
		TradeNo:    "TRADE_001",
	}); err != nil {
		t.Fatal(err)
	}

	body := `{"resource_id":"MTF_SINGLE_STOCK_001","out_trade_no":"ORDER_001"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/paid-resource/prepare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	HandlePaidResource(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("expected paid resource response, got %s", rec.Body.String())
	}
}

func TestPaidResourceRejectsMismatchedBillResource(t *testing.T) {
	useTempPaymentStore(t)

	if err := payment.RememberBill("MTF_SINGLE_STOCK_001", "ORDER_001"); err != nil {
		t.Fatal(err)
	}

	body := `{"resource_id":"OTHER_RESOURCE","out_trade_no":"ORDER_001"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/paid-resource/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	HandlePaymentCheck(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPaymentStorePersistsUnlocks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "payment-state.json")
	store, err := payment.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkUnlocked(payment.UnlockRecord{
		ResourceID: "MTF_SINGLE_STOCK_001",
		OutTradeNo: "ORDER_001",
		TradeNo:    "TRADE_001",
	}); err != nil {
		t.Fatal(err)
	}

	reopened, err := payment.OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	record, ok, err := reopened.LookupUnlocked("MTF_SINGLE_STOCK_001", "ORDER_001", "")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected persisted unlock record")
	}
	if record.TradeNo != "TRADE_001" {
		t.Fatalf("expected TRADE_001, got %q", record.TradeNo)
	}
}

func useTempPaymentStore(t *testing.T) {
	t.Helper()
	store, err := payment.OpenStore(filepath.Join(t.TempDir(), "payment-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	restore := payment.UseStoreForTest(store)
	t.Cleanup(restore)
}
