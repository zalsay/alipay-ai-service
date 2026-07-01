package payment

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type PostgresStore struct {
	dsn string
	db  *sql.DB
}

func (s *PostgresStore) RememberBill(resourceID, outTradeNo string) error {
	resourceID = strings.TrimSpace(resourceID)
	outTradeNo = strings.TrimSpace(outTradeNo)
	if resourceID == "" || outTradeNo == "" {
		return nil
	}

	_, err := s.db.Exec(`
		INSERT INTO payment_bills (out_trade_no, resource_id, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (out_trade_no)
		DO UPDATE SET resource_id = EXCLUDED.resource_id, updated_at = now()
	`, outTradeNo, resourceID)
	if err != nil {
		return fmt.Errorf("persist payment bill: %w", err)
	}
	return nil
}

func (s *PostgresStore) BillResourceMatches(resourceID, outTradeNo string) (bool, error) {
	resourceID = strings.TrimSpace(resourceID)
	outTradeNo = strings.TrimSpace(outTradeNo)
	if resourceID == "" || outTradeNo == "" {
		return true, nil
	}

	var billResourceID string
	err := s.db.QueryRow(`SELECT resource_id FROM payment_bills WHERE out_trade_no = $1`, outTradeNo).Scan(&billResourceID)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("lookup payment bill: %w", err)
	}
	return billResourceID == resourceID, nil
}

func (s *PostgresStore) BillResourceID(outTradeNo string) (string, bool, error) {
	outTradeNo = strings.TrimSpace(outTradeNo)
	if outTradeNo == "" {
		return "", false, nil
	}

	var resourceID string
	err := s.db.QueryRow(`SELECT resource_id FROM payment_bills WHERE out_trade_no = $1`, outTradeNo).Scan(&resourceID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lookup payment bill resource: %w", err)
	}
	return resourceID, true, nil
}

func (s *PostgresStore) MarkUnlocked(record UnlockRecord) (UnlockRecord, error) {
	record.ResourceID = strings.TrimSpace(record.ResourceID)
	record.OutTradeNo = strings.TrimSpace(record.OutTradeNo)
	record.TradeNo = strings.TrimSpace(record.TradeNo)
	record.TradeStatus = strings.TrimSpace(record.TradeStatus)
	if record.UnlockedAt.IsZero() {
		record.UnlockedAt = time.Now()
	}

	tx, err := s.db.Begin()
	if err != nil {
		return UnlockRecord{}, fmt.Errorf("begin payment unlock tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if record.ResourceID != "" && record.OutTradeNo != "" {
		if _, err = tx.Exec(`
			INSERT INTO payment_bills (out_trade_no, resource_id, updated_at)
			VALUES ($1, $2, now())
			ON CONFLICT (out_trade_no)
			DO UPDATE SET resource_id = EXCLUDED.resource_id, updated_at = now()
		`, record.OutTradeNo, record.ResourceID); err != nil {
			return UnlockRecord{}, fmt.Errorf("persist payment bill in unlock tx: %w", err)
		}
	}

	if record.OutTradeNo != "" {
		result, err := tx.Exec(`
			UPDATE payment_unlocks
			SET trade_no = nullif($3, ''), trade_status = nullif($4, ''), unlocked_at = $5, updated_at = now()
			WHERE resource_id = $1 AND out_trade_no = $2
		`, record.ResourceID, record.OutTradeNo, record.TradeNo, record.TradeStatus, record.UnlockedAt)
		if err != nil {
			return UnlockRecord{}, fmt.Errorf("update payment unlock by out_trade_no: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return UnlockRecord{}, fmt.Errorf("inspect payment unlock update by out_trade_no: %w", err)
		}
		if affected == 0 {
			if _, err = tx.Exec(`
				INSERT INTO payment_unlocks (resource_id, out_trade_no, trade_no, trade_status, unlocked_at, updated_at)
				VALUES ($1, $2, nullif($3, ''), nullif($4, ''), $5, now())
			`, record.ResourceID, record.OutTradeNo, record.TradeNo, record.TradeStatus, record.UnlockedAt); err != nil {
				return UnlockRecord{}, fmt.Errorf("insert payment unlock by out_trade_no: %w", err)
			}
		}
	}
	if record.TradeNo != "" {
		result, err := tx.Exec(`
			UPDATE payment_unlocks
			SET out_trade_no = nullif($2, ''), trade_status = nullif($4, ''), unlocked_at = $5, updated_at = now()
			WHERE resource_id = $1 AND trade_no = $3
		`, record.ResourceID, record.OutTradeNo, record.TradeNo, record.TradeStatus, record.UnlockedAt)
		if err != nil {
			return UnlockRecord{}, fmt.Errorf("update payment unlock by trade_no: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return UnlockRecord{}, fmt.Errorf("inspect payment unlock update by trade_no: %w", err)
		}
		if affected == 0 {
			if _, err = tx.Exec(`
				INSERT INTO payment_unlocks (resource_id, out_trade_no, trade_no, trade_status, unlocked_at, updated_at)
				VALUES ($1, nullif($2, ''), $3, nullif($4, ''), $5, now())
			`, record.ResourceID, record.OutTradeNo, record.TradeNo, record.TradeStatus, record.UnlockedAt); err != nil {
				return UnlockRecord{}, fmt.Errorf("insert payment unlock by trade_no: %w", err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return UnlockRecord{}, fmt.Errorf("commit payment unlock tx: %w", err)
	}
	return record, nil
}

func (s *PostgresStore) LookupUnlocked(resourceID, outTradeNo, tradeNo string) (UnlockRecord, bool, error) {
	resourceID = strings.TrimSpace(resourceID)
	outTradeNo = strings.TrimSpace(outTradeNo)
	tradeNo = strings.TrimSpace(tradeNo)

	if outTradeNo != "" {
		record, ok, err := s.lookup(`
			SELECT resource_id, coalesce(out_trade_no, ''), coalesce(trade_no, ''), coalesce(trade_status, ''), unlocked_at
			FROM payment_unlocks
			WHERE resource_id = $1 AND out_trade_no = $2
			ORDER BY updated_at DESC
			LIMIT 1
		`, resourceID, outTradeNo)
		if err != nil || ok {
			return record, ok, err
		}
	}
	if tradeNo != "" {
		return s.lookup(`
			SELECT resource_id, coalesce(out_trade_no, ''), coalesce(trade_no, ''), coalesce(trade_status, ''), unlocked_at
			FROM payment_unlocks
			WHERE resource_id = $1 AND trade_no = $2
			ORDER BY updated_at DESC
			LIMIT 1
		`, resourceID, tradeNo)
	}
	return UnlockRecord{}, false, nil
}

func (s *PostgresStore) lookup(query, resourceID, id string) (UnlockRecord, bool, error) {
	var record UnlockRecord
	err := s.db.QueryRow(query, resourceID, id).Scan(
		&record.ResourceID,
		&record.OutTradeNo,
		&record.TradeNo,
		&record.TradeStatus,
		&record.UnlockedAt,
	)
	if err == sql.ErrNoRows {
		return UnlockRecord{}, false, nil
	}
	if err != nil {
		return UnlockRecord{}, false, fmt.Errorf("lookup payment unlock: %w", err)
	}
	return record, true, nil
}

func (s *PostgresStore) init() error {
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("ping postgres payment state db: %w", err)
	}
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS payment_bills (
			out_trade_no text PRIMARY KEY,
			resource_id text NOT NULL,
			updated_at timestamptz NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS payment_unlocks (
			id bigserial PRIMARY KEY,
			resource_id text NOT NULL,
			out_trade_no text NULL,
			trade_no text NULL,
			trade_status text NULL,
			unlocked_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		);

		CREATE UNIQUE INDEX IF NOT EXISTS payment_unlocks_resource_out_trade_no_uidx
			ON payment_unlocks (resource_id, out_trade_no)
			WHERE out_trade_no IS NOT NULL;

		CREATE UNIQUE INDEX IF NOT EXISTS payment_unlocks_resource_trade_no_uidx
			ON payment_unlocks (resource_id, trade_no)
			WHERE trade_no IS NOT NULL;
	`)
	if err != nil {
		return fmt.Errorf("init postgres payment state schema: %w", err)
	}
	return nil
}
