package db

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *Store {
	// Use in-memory database for testing
	// modernc.org/sqlite supports in-memory databases with the name ":memory:"
	store, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	// Silence logs during tests
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})))

	return store
}

func TestInsertAndGetAuditLogs(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// 1. Initially, it should be empty
	logs, err := store.GetAuditLogs(10)
	if err != nil {
		t.Fatalf("unexpected error getting logs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}

	// 2. Insert a log
	err = store.InsertAuditLog("admin", "test_action", "agent-123", `{"key":"value"}`)
	if err != nil {
		t.Fatalf("unexpected error inserting log: %v", err)
	}

	// 3. Verify it was inserted
	logs, err = store.GetAuditLogs(10)
	if err != nil {
		t.Fatalf("unexpected error getting logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	log := logs[0]
	if log.Username != "admin" {
		t.Errorf("expected username 'admin', got '%s'", log.Username)
	}
	if log.Action != "test_action" {
		t.Errorf("expected action 'test_action', got '%s'", log.Action)
	}
	if log.Target != "agent-123" {
		t.Errorf("expected target 'agent-123', got '%s'", log.Target)
	}
	if log.Details != `{"key":"value"}` {
		t.Errorf("expected details '{\"key\":\"value\"}', got '%s'", log.Details)
	}
	if log.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt time")
	}

	// 4. Insert another log to check order
	// Sleep briefly to ensure the created_at timestamp is different
	time.Sleep(10 * time.Millisecond)
	err = store.InsertAuditLog("user", "another_action", "agent-456", "none")
	if err != nil {
		t.Fatalf("unexpected error inserting second log: %v", err)
	}

	logs, err = store.GetAuditLogs(10)
	if err != nil {
		t.Fatalf("unexpected error getting logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}

	// GetAuditLogs orders by created_at DESC, so the newest should be first
	if logs[0].Username != "user" {
		t.Errorf("expected newest log to be first (got user '%s')", logs[0].Username)
	}
	if logs[1].Username != "admin" {
		t.Errorf("expected oldest log to be second (got user '%s')", logs[1].Username)
	}
}

func TestGetAuditLogsLimit(t *testing.T) {
	store := setupTestDB(t)
	defer store.Close()

	// Insert 5 logs
	for i := 0; i < 5; i++ {
		store.InsertAuditLog("admin", "action", "agent", "detail")
	}

	// Request with limit 3
	logs, err := store.GetAuditLogs(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs due to limit, got %d", len(logs))
	}
}
