package payment

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const defaultStatePath = "/data/payment-state.json"

type UnlockRecord struct {
	ResourceID  string    `json:"resource_id"`
	OutTradeNo  string    `json:"out_trade_no,omitempty"`
	TradeNo     string    `json:"trade_no,omitempty"`
	TradeStatus string    `json:"trade_status,omitempty"`
	UnlockedAt  time.Time `json:"unlocked_at"`
}

type Store interface {
	RememberBill(resourceID, outTradeNo string) error
	BillResourceMatches(resourceID, outTradeNo string) (bool, error)
	BillResourceID(outTradeNo string) (string, bool, error)
	MarkUnlocked(record UnlockRecord) (UnlockRecord, error)
	LookupUnlocked(resourceID, outTradeNo, tradeNo string) (UnlockRecord, bool, error)
}

type FileStore struct {
	path string
	mu   sync.RWMutex
	data stateData
}

type stateData struct {
	Bills        map[string]string       `json:"bills"`
	ByOutTradeNo map[string]UnlockRecord `json:"by_out_trade_no"`
	ByTradeNo    map[string]UnlockRecord `json:"by_trade_no"`
}

var defaultStore Store = NewFileStore(statePathFromEnv())

func NewFileStore(path string) *FileStore {
	path = strings.TrimSpace(path)
	if path == "" {
		path = defaultStatePath
	}
	return &FileStore{path: path, data: newStateData()}
}

func OpenStore(path string) (*FileStore, error) {
	store := NewFileStore(path)
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func OpenPostgresStore(dsn string) (*PostgresStore, error) {
	store := &PostgresStore{dsn: strings.TrimSpace(dsn)}
	if store.dsn == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}
	db, err := sql.Open("postgres", store.dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres payment state db: %w", err)
	}
	store.db = db
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func UseStoreForTest(store Store) func() {
	previous := defaultStore
	defaultStore = store
	return func() {
		defaultStore = previous
	}
}

func Configure(backend, path, dsn string) error {
	var (
		store Store
		err   error
	)
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "", "file":
		store, err = OpenStore(path)
	case "postgres", "postgresql":
		store, err = OpenPostgresStore(dsn)
	default:
		err = fmt.Errorf("unsupported payment state backend %q", backend)
	}
	if err != nil {
		return err
	}
	defaultStore = store
	return nil
}

func RememberBill(resourceID, outTradeNo string) error {
	return defaultStore.RememberBill(resourceID, outTradeNo)
}

func BillResourceMatches(resourceID, outTradeNo string) (bool, error) {
	return defaultStore.BillResourceMatches(resourceID, outTradeNo)
}

func BillResourceID(outTradeNo string) (string, bool, error) {
	return defaultStore.BillResourceID(outTradeNo)
}

func MarkUnlocked(record UnlockRecord) (UnlockRecord, error) {
	return defaultStore.MarkUnlocked(record)
}

func LookupUnlocked(resourceID, outTradeNo, tradeNo string) (UnlockRecord, bool, error) {
	return defaultStore.LookupUnlocked(resourceID, outTradeNo, tradeNo)
}

func (s *FileStore) RememberBill(resourceID, outTradeNo string) error {
	resourceID = strings.TrimSpace(resourceID)
	outTradeNo = strings.TrimSpace(outTradeNo)
	if resourceID == "" || outTradeNo == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Bills[outTradeNo] = resourceID
	return s.persistLocked()
}

func (s *FileStore) BillResourceMatches(resourceID, outTradeNo string) (bool, error) {
	resourceID = strings.TrimSpace(resourceID)
	outTradeNo = strings.TrimSpace(outTradeNo)
	if resourceID == "" || outTradeNo == "" {
		return true, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	billResourceID, ok := s.data.Bills[outTradeNo]
	return !ok || billResourceID == resourceID, nil
}

func (s *FileStore) BillResourceID(outTradeNo string) (string, bool, error) {
	outTradeNo = strings.TrimSpace(outTradeNo)
	if outTradeNo == "" {
		return "", false, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	resourceID, ok := s.data.Bills[outTradeNo]
	return resourceID, ok, nil
}

func (s *FileStore) MarkUnlocked(record UnlockRecord) (UnlockRecord, error) {
	record.ResourceID = strings.TrimSpace(record.ResourceID)
	record.OutTradeNo = strings.TrimSpace(record.OutTradeNo)
	record.TradeNo = strings.TrimSpace(record.TradeNo)
	record.TradeStatus = strings.TrimSpace(record.TradeStatus)
	if record.UnlockedAt.IsZero() {
		record.UnlockedAt = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if record.ResourceID != "" && record.OutTradeNo != "" {
		s.data.Bills[record.OutTradeNo] = record.ResourceID
	}
	if record.OutTradeNo != "" {
		s.data.ByOutTradeNo[unlockKey(record.ResourceID, record.OutTradeNo)] = record
	}
	if record.TradeNo != "" {
		s.data.ByTradeNo[unlockKey(record.ResourceID, record.TradeNo)] = record
	}
	return record, s.persistLocked()
}

func (s *FileStore) LookupUnlocked(resourceID, outTradeNo, tradeNo string) (UnlockRecord, bool, error) {
	resourceID = strings.TrimSpace(resourceID)
	outTradeNo = strings.TrimSpace(outTradeNo)
	tradeNo = strings.TrimSpace(tradeNo)

	s.mu.RLock()
	defer s.mu.RUnlock()
	if outTradeNo != "" {
		record, ok := s.data.ByOutTradeNo[unlockKey(resourceID, outTradeNo)]
		if ok {
			return record, true, nil
		}
	}
	if tradeNo != "" {
		record, ok := s.data.ByTradeNo[unlockKey(resourceID, tradeNo)]
		if ok {
			return record, true, nil
		}
	}
	return UnlockRecord{}, false, nil
}

func IsPaidStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return true
	default:
		return false
	}
}

func ResetUnlocksForTest() {
	store := NewFileStore("")
	defaultStore = store
}

func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read payment state db: %w", err)
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return fmt.Errorf("decode payment state db: %w", err)
	}
	s.data.ensure()
	return nil
}

func (s *FileStore) persistLocked() error {
	s.data.ensure()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create payment state db dir: %w", err)
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode payment state db: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write payment state db temp file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace payment state db: %w", err)
	}
	return nil
}

func newStateData() stateData {
	return stateData{
		Bills:        map[string]string{},
		ByOutTradeNo: map[string]UnlockRecord{},
		ByTradeNo:    map[string]UnlockRecord{},
	}
}

func (d *stateData) ensure() {
	if d.Bills == nil {
		d.Bills = map[string]string{}
	}
	if d.ByOutTradeNo == nil {
		d.ByOutTradeNo = map[string]UnlockRecord{}
	}
	if d.ByTradeNo == nil {
		d.ByTradeNo = map[string]UnlockRecord{}
	}
}

func statePathFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("PAYMENT_STATE_DB_PATH")); v != "" {
		return v
	}
	return defaultStatePath
}

func unlockKey(resourceID, id string) string {
	return resourceID + "\x00" + id
}
