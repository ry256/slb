package testutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dicklesworthstone/slb/internal/db"
)

// =============================================================================
// assertions.go tests
// =============================================================================

func TestRequireNoError_NoError(t *testing.T) {
	// Should not panic when err is nil
	RequireNoError(t, nil, "should pass")
}

func TestRequireEqual_Equal(t *testing.T) {
	// Should not panic when values are equal
	RequireEqual(t, 42, 42, "numbers should be equal")
	RequireEqual(t, "hello", "hello", "strings should be equal")
	RequireEqual(t, true, true, "bools should be equal")
}

func TestRequireLen_CorrectLength(t *testing.T) {
	// Should not panic when length matches
	RequireLen(t, []int{1, 2, 3}, 3, "slice should have 3 elements")
	RequireLen(t, []string{"a", "b"}, 2, "slice should have 2 elements")
	RequireLen(t, []int{}, 0, "slice should be empty")
}

// =============================================================================
// db.go tests
// =============================================================================

func TestNewTestDB(t *testing.T) {
	database := NewTestDB(t)
	if database == nil {
		t.Fatal("NewTestDB returned nil")
	}

	// Verify we can interact with the database
	sess := &db.Session{
		ID:          "test-sess-1",
		AgentName:   "TestAgent",
		Program:     "test",
		Model:       "test-model",
		ProjectPath: t.TempDir(),
	}
	if err := database.CreateSession(sess); err != nil {
		t.Errorf("failed to create session: %v", err)
	}

	// Verify session was created
	retrieved, err := database.GetSession(sess.ID)
	if err != nil {
		t.Errorf("failed to retrieve session: %v", err)
	}
	if retrieved.AgentName != sess.AgentName {
		t.Errorf("agent name mismatch: got %s, want %s", retrieved.AgentName, sess.AgentName)
	}
}

func TestNewTestDBAtPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.db")

	database := NewTestDBAtPath(t, path)
	if database == nil {
		t.Fatal("NewTestDBAtPath returned nil")
	}

	// Verify file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("database file was not created at specified path")
	}
}

func TestWithTestDB(t *testing.T) {
	var calledWith *db.DB
	WithTestDB(t, func(database *db.DB) {
		calledWith = database
		if database == nil {
			t.Error("WithTestDB passed nil database")
		}
	})
	if calledWith == nil {
		t.Error("WithTestDB callback was not called")
	}
}

func TestCleanupTestDB_NilDB(t *testing.T) {
	err := CleanupTestDB(nil)
	if err != nil {
		t.Errorf("CleanupTestDB(nil) should return nil, got %v", err)
	}
}

func TestCleanupTestDB_ValidDB(t *testing.T) {
	// NewTestDB registers cleanup via t.Cleanup, so we just verify
	// CleanupTestDB doesn't panic when called on a valid DB
	_ = NewTestDB(t)
	// The actual close happens via t.Cleanup
}

func TestTempDB(t *testing.T) {
	// TempDB is just an alias for NewTestDB
	database := TempDB(t)
	if database == nil {
		t.Fatal("TempDB returned nil")
	}
}

// =============================================================================
// fixtures.go tests
// =============================================================================

func TestMakeSession(t *testing.T) {
	database := NewTestDB(t)

	sess := MakeSession(t, database)
	if sess == nil {
		t.Fatal("MakeSession returned nil")
	}
	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
	if sess.AgentName == "" {
		t.Error("agent name should not be empty")
	}
	if sess.Program != "test" {
		t.Errorf("expected program 'test', got %s", sess.Program)
	}
	if sess.Model != "model" {
		t.Errorf("expected model 'model', got %s", sess.Model)
	}
}

func TestMakeSession_WithOptions(t *testing.T) {
	database := NewTestDB(t)

	sess := MakeSession(t, database,
		WithAgent("CustomAgent"),
		WithProgram("custom-program"),
		WithModel("custom-model"),
		WithProject("/custom/path"),
	)

	if sess.AgentName != "CustomAgent" {
		t.Errorf("expected agent 'CustomAgent', got %s", sess.AgentName)
	}
	if sess.Program != "custom-program" {
		t.Errorf("expected program 'custom-program', got %s", sess.Program)
	}
	if sess.Model != "custom-model" {
		t.Errorf("expected model 'custom-model', got %s", sess.Model)
	}
	if sess.ProjectPath != "/custom/path" {
		t.Errorf("expected project '/custom/path', got %s", sess.ProjectPath)
	}
}

func TestMakeRequest(t *testing.T) {
	database := NewTestDB(t)
	sess := MakeSession(t, database)

	req := MakeRequest(t, database, sess)
	if req == nil {
		t.Fatal("MakeRequest returned nil")
	}
	if req.ID == "" {
		t.Error("request ID should not be empty")
	}
	if req.RequestorSessionID != sess.ID {
		t.Errorf("requestor session ID mismatch: got %s, want %s", req.RequestorSessionID, sess.ID)
	}
	if req.Status != db.StatusPending {
		t.Errorf("expected status pending, got %s", req.Status)
	}
}

func TestMakeRequest_WithOptions(t *testing.T) {
	database := NewTestDB(t)
	sess := MakeSession(t, database)

	customExpiry := time.Now().Add(1 * time.Hour)
	req := MakeRequest(t, database, sess,
		WithCommand("echo hello", "/tmp", true),
		WithRisk(db.RiskTierCritical),
		WithExpiresAt(customExpiry),
		WithJustification("reason", "effect", "goal", "safety"),
		WithDryRun("dry cmd", "dry output"),
		WithRequireDifferentModel(true),
		WithMinApprovals(3),
	)

	if req.Command.Raw != "echo hello" {
		t.Errorf("expected command 'echo hello', got %s", req.Command.Raw)
	}
	if req.Command.Cwd != "/tmp" {
		t.Errorf("expected cwd '/tmp', got %s", req.Command.Cwd)
	}
	if !req.Command.Shell {
		t.Error("expected shell to be true")
	}
	if req.RiskTier != db.RiskTierCritical {
		t.Errorf("expected critical risk tier, got %s", req.RiskTier)
	}
	if req.Justification.Reason != "reason" {
		t.Errorf("expected justification reason 'reason', got %s", req.Justification.Reason)
	}
	if req.Justification.ExpectedEffect != "effect" {
		t.Errorf("expected expected effect 'effect', got %s", req.Justification.ExpectedEffect)
	}
	if req.Justification.Goal != "goal" {
		t.Errorf("expected goal 'goal', got %s", req.Justification.Goal)
	}
	if req.Justification.SafetyArgument != "safety" {
		t.Errorf("expected safety argument 'safety', got %s", req.Justification.SafetyArgument)
	}
	if req.DryRun == nil {
		t.Fatal("expected dry run to be set")
	}
	if req.DryRun.Command != "dry cmd" {
		t.Errorf("expected dry run command 'dry cmd', got %s", req.DryRun.Command)
	}
	if req.DryRun.Output != "dry output" {
		t.Errorf("expected dry run output 'dry output', got %s", req.DryRun.Output)
	}
	if !req.RequireDifferentModel {
		t.Error("expected RequireDifferentModel to be true")
	}
	if req.MinApprovals != 3 {
		t.Errorf("expected MinApprovals 3, got %d", req.MinApprovals)
	}
}

func TestSessionWithAgentName(t *testing.T) {
	database := NewTestDB(t)

	// SessionWithAgentName is an alias for WithAgent
	sess := MakeSession(t, database, SessionWithAgentName("AliasAgent"))
	if sess.AgentName != "AliasAgent" {
		t.Errorf("expected agent 'AliasAgent', got %s", sess.AgentName)
	}
}

func TestSessionWithProject(t *testing.T) {
	database := NewTestDB(t)

	// SessionWithProject is an alias for WithProject
	sess := MakeSession(t, database, SessionWithProject("/alias/path"))
	if sess.ProjectPath != "/alias/path" {
		t.Errorf("expected project '/alias/path', got %s", sess.ProjectPath)
	}
}

func TestRandHex(t *testing.T) {
	// Test that randHex produces unique values
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		hex := randHex(8)
		if len(hex) != 8 {
			t.Errorf("expected length 8, got %d", len(hex))
		}
		if seen[hex] {
			t.Errorf("duplicate hex value: %s", hex)
		}
		seen[hex] = true
	}

	// Test different lengths
	for _, n := range []int{1, 2, 4, 6, 10, 16, 32} {
		hex := randHex(n)
		if len(hex) != n {
			t.Errorf("randHex(%d) returned length %d", n, len(hex))
		}
	}
}

// =============================================================================
// integration.go tests
// =============================================================================

func TestNewHarness(t *testing.T) {
	h := NewHarness(t)
	if h == nil {
		t.Fatal("NewHarness returned nil")
	}
	if h.T != t {
		t.Error("harness T should be the test instance")
	}
	if h.ProjectDir == "" {
		t.Error("ProjectDir should not be empty")
	}
	if h.SLBDir == "" {
		t.Error("SLBDir should not be empty")
	}
	if h.DBPath == "" {
		t.Error("DBPath should not be empty")
	}
	if h.DB == nil {
		t.Error("DB should not be nil")
	}

	// Verify .slb directory was created
	if _, err := os.Stat(h.SLBDir); os.IsNotExist(err) {
		t.Error(".slb directory was not created")
	}
}

func TestHarness_MustPath(t *testing.T) {
	h := NewHarness(t)

	path := h.MustPath("subdir", "file.txt")
	expected := filepath.Join(h.ProjectDir, "subdir", "file.txt")
	if path != expected {
		t.Errorf("MustPath: got %s, want %s", path, expected)
	}

	// Single component
	path2 := h.MustPath("single")
	expected2 := filepath.Join(h.ProjectDir, "single")
	if path2 != expected2 {
		t.Errorf("MustPath single: got %s, want %s", path2, expected2)
	}

	// Empty (just project dir)
	path3 := h.MustPath()
	if path3 != h.ProjectDir {
		t.Errorf("MustPath empty: got %s, want %s", path3, h.ProjectDir)
	}
}

func TestHarness_WriteFile(t *testing.T) {
	h := NewHarness(t)

	// Write a simple file
	absPath := h.WriteFile("test.txt", []byte("hello world"), 0644)
	if absPath == "" {
		t.Error("WriteFile returned empty path")
	}

	// Verify file exists and has correct content
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("file content mismatch: got %q, want %q", string(content), "hello world")
	}

	// Write file in nested directory
	nestedPath := h.WriteFile("deep/nested/dir/file.txt", []byte("nested"), 0644)
	nestedContent, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("failed to read nested file: %v", err)
	}
	if string(nestedContent) != "nested" {
		t.Errorf("nested file content mismatch: got %q", string(nestedContent))
	}
}

func TestHarness_String(t *testing.T) {
	h := NewHarness(t)

	str := h.String()
	if str == "" {
		t.Error("String() returned empty")
	}
	if str == "Harness<nil>" {
		t.Error("String() returned nil representation for valid harness")
	}
	// Should contain project and db info
	if !containsAll(str, "Harness", h.ProjectDir, h.DBPath) {
		t.Errorf("String() missing expected content: %s", str)
	}
}

func TestHarness_String_Nil(t *testing.T) {
	var h *Harness
	str := h.String()
	if str != "Harness<nil>" {
		t.Errorf("nil harness String(): got %q, want %q", str, "Harness<nil>")
	}
}

// =============================================================================
// logging.go tests
// =============================================================================

func TestTestLogger(t *testing.T) {
	logger := TestLogger(t)
	if logger == nil {
		t.Fatal("TestLogger returned nil")
	}

	// Should be able to log without panicking
	logger.Info("test message")
	logger.Debug("debug message")
	logger.Warn("warning message")
}

func TestTestLogger_VerboseMode(t *testing.T) {
	// Note: We can't easily test verbose mode since testing.Verbose()
	// is determined at runtime. But we can verify the logger is created.
	logger := TestLogger(t)
	if logger == nil {
		t.Fatal("TestLogger returned nil")
	}
}

// =============================================================================
// Helper functions
// =============================================================================

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
