package harness

import (
	"fmt"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/db"
)

func TestNewE2EEnvironment(t *testing.T) {
	env := NewE2EEnvironment(t)

	// Verify directories created
	env.Step("Verifying environment structure")

	env.AssertFileExists(".slb")
	env.AssertFileExists(".slb/state.db")
	env.AssertFileExists(".slb/logs")
	env.AssertFileExists(".slb/pending")

	// Verify git initialized
	env.Step("Verifying git repository")
	env.AssertFileExists(".git")

	head := env.GitHead()
	if len(head) < 7 {
		t.Errorf("GitHead too short: %s", head)
	}
	env.Result("Git HEAD: %s", head[:7])

	env.DBState()
	env.Logger.Elapsed()
}

func TestE2EEnvironment_Sessions(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Creating a test session")
	sess := env.CreateSession("TestAgent", "test-program", "test-model")

	if sess.ID == "" {
		t.Error("session ID empty")
	}
	if sess.AgentName != "TestAgent" {
		t.Errorf("agent name: got %s, want TestAgent", sess.AgentName)
	}

	env.AssertActiveSessionCount(1)
	env.AssertSessionActive(sess)

	env.DBState()
}

func TestE2EEnvironment_Requests(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Creating requestor session")
	requestor := env.CreateSession("Requestor", "claude-code", "opus")

	env.Step("Submitting a request")
	req := env.SubmitRequest(requestor, "rm -rf ./build", "Clean build artifacts")

	if req.ID == "" {
		t.Error("request ID empty")
	}
	env.AssertRequestStatus(req, db.StatusPending)
	env.AssertPendingCount(1)

	env.Step("Creating reviewer session")
	reviewer := env.CreateSession("Reviewer", "codex", "gpt-4")

	env.Step("Approving the request")
	_ = env.ApproveRequest(req, reviewer)

	env.AssertReviewCount(req, 1)
	env.AssertApprovalCount(req, 1)

	env.DBState()
}

func TestE2EEnvironment_GitOperations(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Creating test file")
	env.WriteTestFile("test.txt", []byte("hello world"))
	env.AssertFileExists("test.txt")

	env.Step("Committing changes")
	hash1 := env.GitCommit("Add test file")

	if len(hash1) < 7 {
		t.Errorf("commit hash too short: %s", hash1)
	}

	env.Step("Creating another file")
	env.WriteTestFile("other.txt", []byte("other content"))

	env.Step("Second commit")
	hash2 := env.GitCommit("Add other file")

	if hash1 == hash2 {
		t.Error("commits should have different hashes")
	}

	env.Logger.Elapsed()
}

func TestStepLogger(t *testing.T) {
	logger := NewStepLogger(t)

	logger.Step(1, "First step")
	logger.Result("got value %d", 42)
	logger.DBState(2, 3)
	logger.Info("information")
	logger.Expected("foo", "bar", "bar", true)
	logger.Expected("fail", "a", "b", false)
	logger.Elapsed()

	// No assertions - just verify it doesn't panic
}

func TestLogBuffer(t *testing.T) {
	buf := NewLogBuffer()

	_, _ = buf.Write([]byte("test message"))
	_, _ = buf.Write([]byte("another message"))

	if len(buf.Entries()) != 2 {
		t.Errorf("expected 2 entries, got %d", len(buf.Entries()))
	}

	if !buf.Contains("test") {
		t.Error("buffer should contain 'test'")
	}

	if buf.Contains("nonexistent") {
		t.Error("buffer should not contain 'nonexistent'")
	}

	buf.Clear()
	if len(buf.Entries()) != 0 {
		t.Error("buffer should be empty after clear")
	}
}

func TestE2EEnvironment_RequestTier(t *testing.T) {
	env := NewE2EEnvironment(t)

	sess := env.CreateSession("TestAgent", "test-program", "test-model")

	// Test CRITICAL tier (rm -rf command)
	env.Step("Testing CRITICAL tier classification")
	criticalReq := env.SubmitRequest(sess, "rm -rf /important", "Test critical")
	env.AssertRequestTier(criticalReq, db.RiskTierCritical)

	// Test DANGEROUS tier (rm without -rf)
	env.Step("Testing DANGEROUS tier classification")
	dangerousReq := env.SubmitRequest(sess, "rm sensitive.txt", "Test dangerous")
	env.AssertRequestTier(dangerousReq, db.RiskTierDangerous)

	// Test CAUTION tier (safe command)
	env.Step("Testing CAUTION tier classification")
	cautionReq := env.SubmitRequest(sess, "go build ./...", "Test caution")
	env.AssertRequestTier(cautionReq, db.RiskTierCaution)
}

func TestE2EEnvironment_RequestStatusByID(t *testing.T) {
	env := NewE2EEnvironment(t)

	sess := env.CreateSession("TestAgent", "test-program", "test-model")
	req := env.SubmitRequest(sess, "echo hello", "Test status by ID")

	env.Step("Asserting request status by ID")
	env.AssertRequestStatusByID(req.ID, db.StatusPending)

	// Get status helper
	status := env.GetRequestStatus(req.ID)
	if status != db.StatusPending {
		t.Errorf("expected pending status, got %s", status)
	}
}

func TestE2EEnvironment_SessionEnded(t *testing.T) {
	env := NewE2EEnvironment(t)

	sess := env.CreateSession("TestAgent", "test-program", "test-model")
	env.AssertSessionActive(sess)

	env.Step("Ending session")
	if err := env.DB.EndSession(sess.ID); err != nil {
		t.Fatalf("EndSession: %v", err)
	}

	env.AssertSessionEnded(sess)
}

func TestE2EEnvironment_RejectRequest(t *testing.T) {
	env := NewE2EEnvironment(t)

	requestor := env.CreateSession("Requestor", "claude-code", "opus")
	reviewer := env.CreateSession("Reviewer", "codex", "gpt-4")

	req := env.SubmitRequest(requestor, "dangerous command", "Test rejection")
	env.AssertRequestStatus(req, db.StatusPending)

	env.Step("Rejecting the request")
	review := env.RejectRequest(req, reviewer, "Not safe")

	if review.Decision != db.DecisionReject {
		t.Errorf("expected reject decision, got %s", review.Decision)
	}

	env.AssertReviewCount(req, 1)
}

func TestE2EEnvironment_GitHead(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Getting initial HEAD")
	head1 := env.GitHead()
	if len(head1) < 7 {
		t.Errorf("HEAD too short: %s", head1)
	}

	env.Step("Creating a commit")
	env.WriteTestFile("test.txt", []byte("content"))
	hash := env.GitCommit("Add test file")

	env.Step("Asserting HEAD matches commit")
	env.AssertGitHead(hash)
}

func TestE2EEnvironment_FileNotExists(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Asserting non-existent file")
	env.AssertFileNotExists("nonexistent.txt")

	env.Step("Creating file")
	env.WriteTestFile("exists.txt", []byte("content"))
	env.AssertFileExists("exists.txt")
}

func TestE2EEnvironment_NoErrorAndError(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Testing AssertNoError with nil")
	env.AssertNoError(nil, "should pass")

	env.Step("Testing AssertError with actual error")
	env.AssertError(fmt.Errorf("expected error"), "should pass with error")

	env.Step("Testing error logging")
	env.Logger.Error("Test error message: %s", "test")
}

func TestE2EEnvironment_Elapsed(t *testing.T) {
	env := NewE2EEnvironment(t)

	env.Step("Checking elapsed time")
	elapsed := env.Elapsed()
	if elapsed < 0 {
		t.Error("elapsed time should be non-negative")
	}
}

func TestGitError(t *testing.T) {
	err := &gitError{
		op:  "test",
		err: fmt.Errorf("mock error"),
		out: "mock output",
	}

	msg := err.Error()
	if msg == "" {
		t.Error("error message should not be empty")
	}
	if !containsAny(msg, "test", "mock error", "mock output") {
		t.Errorf("error message missing expected content: %s", msg)
	}
}
