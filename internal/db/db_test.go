// Package db tests
package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndInitSchema(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "slb-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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

func TestPartialUniqueIndex(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "slb-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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
	tmpDir, err := os.MkdirTemp("", "slb-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

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
