// Package db tests
package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAndInitSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database (should create and init schema)
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Verify schema
	if err := db.ValidateSchema(); err != nil {
		t.Fatalf("schema validation failed: %v", err)
	}

	// Check version
	version, err := db.GetSchemaVersion()
	if err != nil {
		t.Fatalf("failed to get schema version: %v", err)
	}
	if version != SchemaVersion {
		t.Errorf("expected schema version %d, got %d", SchemaVersion, version)
	}

	// Get stats
	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if stats.SchemaVersion != SchemaVersion {
		t.Errorf("stats schema version mismatch")
	}
}

func TestOpenAndMigrate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenAndMigrate(dbPath)
	if err != nil {
		t.Fatalf("OpenAndMigrate failed: %v", err)
	}
	defer db.Close()

	if err := db.ValidateSchema(); err != nil {
		t.Fatalf("ValidateSchema failed: %v", err)
	}
}

func TestOpenAndMigrate_OpenError(t *testing.T) {
	// Passing a directory path should cause Open() to fail.
	_, err := OpenAndMigrate(t.TempDir())
	if err == nil {
		t.Fatalf("expected OpenAndMigrate to fail for directory path")
	}
}

func TestOpenProjectDB(t *testing.T) {
	projectDir := t.TempDir()
	db, err := OpenProjectDB(projectDir)
	if err != nil {
		t.Fatalf("OpenProjectDB failed: %v", err)
	}
	defer db.Close()

	wantPath := filepath.Join(projectDir, ".slb", "state.db")
	if got := db.Path(); got != wantPath {
		t.Fatalf("Path()=%q want %q", got, wantPath)
	}
	if err := db.ValidateSchema(); err != nil {
		t.Fatalf("ValidateSchema failed: %v", err)
	}
}

func TestOpenUserDB(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	db, err := OpenUserDB()
	if err != nil {
		t.Fatalf("OpenUserDB failed: %v", err)
	}
	defer db.Close()

	wantPath := filepath.Join(home, ".slb", "history.db")
	if got := db.Path(); got != wantPath {
		t.Fatalf("Path()=%q want %q", got, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected db file to exist: %v", err)
	}
}

func TestValidateSchema_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES(999, ?)`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("insert schema_migrations failed: %v", err)
	}

	if err := db.ValidateSchema(); err == nil {
		t.Fatalf("expected schema version mismatch error")
	}
}

func TestApplyMigrations_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if err := db.ApplyMigrations(context.Background()); err != nil {
		t.Fatalf("ApplyMigrations failed: %v", err)
	}
}

func TestDB_ReturnsErrorsWhenClosed(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if _, err := db.GetSchemaVersion(); err == nil {
		t.Fatalf("expected GetSchemaVersion to fail on closed DB")
	}
	if err := db.ValidateSchema(); err == nil {
		t.Fatalf("expected ValidateSchema to fail on closed DB")
	}
	if err := db.ApplyMigrations(context.Background()); err == nil {
		t.Fatalf("expected ApplyMigrations to fail on closed DB")
	}
	if _, err := db.GetStats(); err == nil {
		t.Fatalf("expected GetStats to fail on closed DB")
	}
	if _, err := db.ListOutcomes(10); err == nil {
		t.Fatalf("expected ListOutcomes to fail on closed DB")
	}
	if _, err := db.ListPendingRequestsAllProjects(); err == nil {
		t.Fatalf("expected ListPendingRequestsAllProjects to fail on closed DB")
	}
	if _, err := db.SearchRequests("test"); err == nil {
		t.Fatalf("expected SearchRequests to fail on closed DB")
	}
	if _, err := db.ListActiveSessions("/test/project"); err == nil {
		t.Fatalf("expected ListActiveSessions to fail on closed DB")
	}
	if err := db.UpdateRequestExecution("req", &Execution{LogPath: "/tmp/log"}); err == nil {
		t.Fatalf("expected UpdateRequestExecution to fail on closed DB")
	}
}

func TestOpenWithOptions_ReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	_ = db.Close()

	ro, err := OpenWithOptions(dbPath, OpenOptions{
		CreateIfNotExists: false,
		InitSchema:        false,
		ReadOnly:          true,
	})
	if err != nil {
		t.Fatalf("OpenWithOptions(readonly) failed: %v", err)
	}
	defer ro.Close()

	// Writes should fail in read-only mode.
	_, err = ro.Exec(`
		INSERT INTO sessions (id, agent_name, program, model, project_path, session_key, started_at, last_active_at)
		VALUES ('sess-ro-1', 'AgentRO', 'codex-cli', 'gpt-5', '/ro', 'key', ?, ?)
	`, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
	if err == nil {
		t.Fatalf("expected write to fail in read-only mode")
	}
}

func TestDB_Transaction_CommitsRollsBackAndPanics(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)

	if err := db.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO sessions (id, agent_name, program, model, project_path, session_key, started_at, last_active_at)
			VALUES ('sess-tx-1', 'Agent1', 'codex-cli', 'gpt-5', '/p1', 'key1', ?, ?)
		`, now, now)
		return err
	}); err != nil {
		t.Fatalf("Transaction (commit) failed: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = 'sess-tx-1'`).Scan(&count); err != nil {
		t.Fatalf("count sessions failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 committed session, got %d", count)
	}

	rollbackErr := db.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO sessions (id, agent_name, program, model, project_path, session_key, started_at, last_active_at)
			VALUES ('sess-tx-2', 'Agent2', 'codex-cli', 'gpt-5', '/p2', 'key2', ?, ?)
		`, now, now)
		if err != nil {
			return err
		}
		return errors.New("boom")
	})
	if rollbackErr == nil {
		t.Fatalf("expected rollback error")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = 'sess-tx-2'`).Scan(&count); err != nil {
		t.Fatalf("count sessions failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to remove insert, got %d rows", count)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic from Transaction")
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = 'sess-tx-3'`).Scan(&count); err != nil {
			t.Fatalf("count sessions failed: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected panic rollback to remove insert, got %d rows", count)
		}
	}()

	_ = db.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			INSERT INTO sessions (id, agent_name, program, model, project_path, session_key, started_at, last_active_at)
			VALUES ('sess-tx-3', 'Agent3', 'codex-cli', 'gpt-5', '/p3', 'key3', ?, ?)
		`, now, now)
		if err != nil {
			return err
		}
		panic("boom")
	})
}

func TestPartialUniqueIndex(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Insert first active session
	_, err = db.Exec(`
		INSERT INTO sessions (id, agent_name, program, model, project_path, session_key, started_at, last_active_at)
		VALUES ('sess1', 'Agent1', 'claude-code', 'opus-4.5', '/test/project', 'key1', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("failed to insert first session: %v", err)
	}

	// Try to insert second active session for same agent+project (should fail)
	_, err = db.Exec(`
		INSERT INTO sessions (id, agent_name, program, model, project_path, session_key, started_at, last_active_at)
		VALUES ('sess2', 'Agent1', 'claude-code', 'opus-4.5', '/test/project', 'key2', '2024-01-01T00:01:00Z', '2024-01-01T00:01:00Z')
	`)
	if err == nil {
		t.Fatal("expected unique constraint violation for duplicate active session")
	}

	// End first session
	_, err = db.Exec(`UPDATE sessions SET ended_at = '2024-01-01T00:02:00Z' WHERE id = 'sess1'`)
	if err != nil {
		t.Fatalf("failed to end session: %v", err)
	}

	// Now should be able to insert new active session
	_, err = db.Exec(`
		INSERT INTO sessions (id, agent_name, program, model, project_path, session_key, started_at, last_active_at)
		VALUES ('sess2', 'Agent1', 'claude-code', 'opus-4.5', '/test/project', 'key2', '2024-01-01T00:03:00Z', '2024-01-01T00:03:00Z')
	`)
	if err != nil {
		t.Fatalf("failed to insert session after ending previous: %v", err)
	}
}

func TestFTSTriggers(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Insert a session first
	_, err = db.Exec(`
		INSERT INTO sessions (id, agent_name, program, model, project_path, session_key, started_at, last_active_at)
		VALUES ('sess1', 'Agent1', 'claude-code', 'opus-4.5', '/test/project', 'key1', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("failed to insert session: %v", err)
	}

	// Insert a request
	_, err = db.Exec(`
		INSERT INTO requests (id, project_path, command_raw, command_cwd, command_hash, risk_tier,
			requestor_session_id, requestor_agent, requestor_model, justification_reason, status, min_approvals, created_at)
		VALUES ('req1', '/test/project', 'rm -rf /tmp/test', '/tmp', 'hash1', 'critical',
			'sess1', 'Agent1', 'opus-4.5', 'Need to clean up test files', 'pending', 2, '2024-01-01T00:00:00Z')
	`)
	if err != nil {
		t.Fatalf("failed to insert request: %v", err)
	}

	// Search FTS
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM requests_fts WHERE requests_fts MATCH 'clean'`).Scan(&count)
	if err != nil {
		t.Fatalf("FTS search failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 FTS match, got %d", count)
	}
}
