package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// LogRecord represents a single LLM request/response activity log
type LogRecord struct {
	ID               int64   `json:"id"`
	TraceID          string  `json:"trace_id"`
	Model            string  `json:"model"`
	ClientIP         string  `json:"client_ip"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	TTFTMs           int64   `json:"ttft_ms"`
	TokensPerSecond  float64 `json:"tokens_per_second"`
	CachedTokens     int64   `json:"cached_tokens"`
	StatusCode       int     `json:"status_code"`
	ErrorMsg         string  `json:"error_msg"`
	CreatedAt        int64   `json:"created_at"`
}

// SystemStats represents aggregated statistics
type SystemStats struct {
	TotalRequests int64   `json:"total_requests"`
	TodayRequests int64   `json:"today_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	AvgTTFTMs     float64 `json:"avg_ttft_ms"`
	AvgTPS        float64 `json:"avg_tps"`
}

// Store encapsulates the SQLite database and async writer
type Store struct {
	db        *sql.DB
	logChan   chan *LogRecord
	closeChan chan struct{}
	wg        sync.WaitGroup
}

var (
	defaultStore *Store
	storeOnce    sync.Once
)

// InitGlobalStore initializes the global SQLite storage
func InitGlobalStore(dbPath string) (*Store, error) {
	var initErr error
	storeOnce.Do(func() {
		if dbPath == "" {
			dbPath = filepath.Join("data", "trae_proxy.db")
		}

		dir := filepath.Dir(dbPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				initErr = fmt.Errorf("create db dir %s: %w", dir, err)
				return
			}
		}

		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			initErr = fmt.Errorf("open sqlite db %s: %w", dbPath, err)
			return
		}

		// Configure connection pool and pragmas for high concurrency & speed
		db.SetMaxOpenConns(1) // SQLite works best with 1 writer or WAL mode
		db.SetMaxIdleConns(1)

		pragmas := []string{
			"PRAGMA journal_mode=WAL;",
			"PRAGMA synchronous=NORMAL;",
			"PRAGMA busy_timeout=5000;",
		}
		for _, pragma := range pragmas {
			_, _ = db.Exec(pragma)
		}

		schema := `
		CREATE TABLE IF NOT EXISTS request_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trace_id TEXT,
			model TEXT,
			client_ip TEXT,
			prompt_tokens INTEGER DEFAULT 0,
			completion_tokens INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			ttft_ms INTEGER DEFAULT 0,
			tokens_per_second REAL DEFAULT 0,
			cached_tokens INTEGER DEFAULT 0,
			status_code INTEGER DEFAULT 200,
			error_msg TEXT DEFAULT '',
			created_at INTEGER
		);

		CREATE INDEX IF NOT EXISTS idx_logs_created_at ON request_logs(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_logs_model ON request_logs(model);
		`
		if _, err := db.Exec(schema); err != nil {
			initErr = fmt.Errorf("init schema: %w", err)
			return
		}

		s := &Store{
			db:        db,
			logChan:   make(chan *LogRecord, 2000),
			closeChan: make(chan struct{}),
		}

		s.wg.Add(1)
		go s.worker()

		defaultStore = s
	})

	return defaultStore, initErr
}

// GetGlobalStore returns the initialized global store or nil
func GetGlobalStore() *Store {
	return defaultStore
}

// RecordLog sends a log record to the async channel (non-blocking)
func (s *Store) RecordLog(rec *LogRecord) {
	if s == nil || s.logChan == nil || rec == nil {
		return
	}
	if rec.CreatedAt == 0 {
		rec.CreatedAt = time.Now().Unix()
	}
	select {
	case s.logChan <- rec:
	default:
		// Drop if buffer full to avoid blocking LLM streaming
	}
}

// Close gracefully flushes remaining logs and closes the database
func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	close(s.closeChan)
	s.wg.Wait()
	return s.db.Close()
}

func (s *Store) worker() {
	defer s.wg.Done()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var batch []*LogRecord

	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.insertBatch(batch)
		batch = nil
	}

	for {
		select {
		case <-s.closeChan:
			// Drain remaining records
			for {
				select {
				case rec := <-s.logChan:
					batch = append(batch, rec)
				default:
					flush()
					return
				}
			}
		case rec := <-s.logChan:
			batch = append(batch, rec)
			if len(batch) >= 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *Store) insertBatch(records []*LogRecord) {
	if len(records) == 0 {
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO request_logs (
			trace_id, model, client_ip, prompt_tokens, completion_tokens,
			total_tokens, ttft_ms, tokens_per_second, cached_tokens,
			status_code, error_msg, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return
	}
	defer stmt.Close()

	for _, r := range records {
		_, _ = stmt.Exec(
			r.TraceID, r.Model, r.ClientIP, r.PromptTokens, r.CompletionTokens,
			r.TotalTokens, r.TTFTMs, r.TokensPerSecond, r.CachedTokens,
			r.StatusCode, r.ErrorMsg, r.CreatedAt,
		)
	}

	_ = tx.Commit()
}

// QueryLogs returns paginated logs ordered by created_at DESC
func (s *Store) QueryLogs(ctx context.Context, page, pageSize int, model string) ([]*LogRecord, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, fmt.Errorf("db not initialized")
	}

	if pageSize <= 0 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	var total int
	countQuery := "SELECT COUNT(*) FROM request_logs"
	var countArgs []interface{}
	if model != "" {
		countQuery += " WHERE model = ?"
		countArgs = append(countArgs, model)
	}

	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, trace_id, model, client_ip, prompt_tokens, completion_tokens,
		       total_tokens, ttft_ms, tokens_per_second, cached_tokens,
		       status_code, error_msg, created_at
		FROM request_logs
	`
	var args []interface{}
	if model != "" {
		query += " WHERE model = ?"
		args = append(args, model)
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*LogRecord
	for rows.Next() {
		var r LogRecord
		if err := rows.Scan(
			&r.ID, &r.TraceID, &r.Model, &r.ClientIP, &r.PromptTokens, &r.CompletionTokens,
			&r.TotalTokens, &r.TTFTMs, &r.TokensPerSecond, &r.CachedTokens,
			&r.StatusCode, &r.ErrorMsg, &r.CreatedAt,
		); err != nil {
			continue
		}
		logs = append(logs, &r)
	}

	return logs, total, nil
}

// GetStats returns aggregated usage metrics
func (s *Store) GetStats(ctx context.Context) (*SystemStats, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("db not initialized")
	}

	stats := &SystemStats{}

	// Total counts
	row := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_tokens), 0), COALESCE(AVG(ttft_ms), 0), COALESCE(AVG(tokens_per_second), 0)
		FROM request_logs WHERE status_code = 200
	`)
	_ = row.Scan(&stats.TotalRequests, &stats.TotalTokens, &stats.AvgTTFTMs, &stats.AvgTPS)

	// Today's requests
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM request_logs WHERE created_at >= ?`, todayStart).Scan(&stats.TodayRequests)

	return stats, nil
}

// ClearLogs truncates the logs table
func (s *Store) ClearLogs(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("db not initialized")
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM request_logs")
	return err
}
