package payment

import (
	"os"
	"testing"
)

func TestPostgresStorePersistsUnlocks(t *testing.T) {
	dsn := os.Getenv("PAYMENT_STATE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("PAYMENT_STATE_TEST_POSTGRES_DSN is not set")
	}

	store, err := OpenPostgresStore(dsn)
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.MarkUnlocked(UnlockRecord{
		ResourceID:  "TEST_RESOURCE",
		OutTradeNo:  "TEST_ORDER_001",
		TradeNo:     "TEST_TRADE_001",
		TradeStatus: "TRADE_FINISHED",
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.ResourceID != "TEST_RESOURCE" {
		t.Fatalf("expected TEST_RESOURCE, got %q", record.ResourceID)
	}

	reopened, err := OpenPostgresStore(dsn)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := reopened.LookupUnlocked("TEST_RESOURCE", "TEST_ORDER_001", "")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected persisted unlock")
	}
	if got.TradeNo != "TEST_TRADE_001" {
		t.Fatalf("expected TEST_TRADE_001, got %q", got.TradeNo)
	}
}
