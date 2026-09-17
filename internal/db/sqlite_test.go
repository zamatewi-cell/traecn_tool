package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStore(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	store, err := InitGlobalStore(dbPath)
	if err != nil {
		t.Fatalf("InitGlobalStore failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// 1. Record logs
	for i := 1; i <= 5; i++ {
		store.RecordLog(&LogRecord{
			TraceID:          "trace-test",
			Model:            "deepseek-v3",
			ClientIP:         "127.0.0.1",
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
			TTFTMs:           250,
			TokensPerSecond:  120.5,
			StatusCode:       200,
			CreatedAt:        time.Now().Unix(),
		})
	}

	// Allow worker to flush
	time.Sleep(300 * time.Millisecond)

	// 2. Query logs
	logs, total, err := store.QueryLogs(ctx, 1, 10, "")
	if err != nil {
		t.Fatalf("QueryLogs failed: %v", err)
	}
	if total != 5 || len(logs) != 5 {
		t.Fatalf("Expected 5 logs, got total=%d, len=%d", total, len(logs))
	}

	// 3. Get Stats
	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalRequests != 5 {
		t.Fatalf("Expected TotalRequests 5, got %d", stats.TotalRequests)
	}
	if stats.TotalTokens != 750 {
		t.Fatalf("Expected TotalTokens 750, got %d", stats.TotalTokens)
	}

	// 4. Clear logs
	if err := store.ClearLogs(ctx); err != nil {
		t.Fatalf("ClearLogs failed: %v", err)
	}

	logsAfter, totalAfter, _ := store.QueryLogs(ctx, 1, 10, "")
	if totalAfter != 0 || len(logsAfter) != 0 {
		t.Fatalf("Expected 0 logs after clear, got %d", totalAfter)
	}

	_ = os.Remove(dbPath)
}
