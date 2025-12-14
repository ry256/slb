// Package db tests for request CRUD operations.
package db

import (
	"testing"
	"time"
)

func TestCreateRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a session first
	sess := &Session{
		AgentName:   "GreenLake",
		Program:     "claude-code",
		Model:       "opus-4.5",
		ProjectPath: "/test/project",
	}
	if err := db.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	r := &Request{
		ProjectPath:        "/test/project",
		RequestorSessionID: sess.ID,
		RequestorAgent:     "GreenLake",
		RequestorModel:     "opus-4.5",
		RiskTier:           RiskTierCritical,
		MinApprovals:       2,
		Command: CommandSpec{
			Raw:  "rm -rf /tmp/test",
			Cwd:  "/tmp",
			Argv: []string{"rm", "-rf", "/tmp/test"},
		},
		Justification: Justification{
			Reason:         "Need to clean up test files",
			ExpectedEffect: "Removes test directory",
		},
	}

	err := db.CreateRequest(r)
	if err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	// Verify UUID was generated
	if r.ID == "" {
		t.Error("Expected UUID to be generated")
	}

	// Verify hash was computed
	if r.Command.Hash == "" {
		t.Error("Expected command hash to be computed")
	}

	// Verify timestamps
	if r.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if r.ExpiresAt == nil {
		t.Error("Expected ExpiresAt to be set")
	}
}

func TestGetRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sess, r := createTestRequest(t, db)

	retrieved, err := db.GetRequest(r.ID)
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}

	if retrieved.ID != r.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, r.ID)
	}
	if retrieved.Command.Raw != r.Command.Raw {
		t.Errorf("Command.Raw mismatch: got %s, want %s", retrieved.Command.Raw, r.Command.Raw)
	}
	if retrieved.RequestorAgent != r.RequestorAgent {
		t.Errorf("RequestorAgent mismatch: got %s, want %s", retrieved.RequestorAgent, r.RequestorAgent)
	}
	if retrieved.RequestorSessionID != sess.ID {
		t.Errorf("RequestorSessionID mismatch: got %s, want %s", retrieved.RequestorSessionID, sess.ID)
	}
}

func TestGetRequestNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.GetRequest("nonexistent-id")
	if err != ErrRequestNotFound {
		t.Errorf("Expected ErrRequestNotFound, got: %v", err)
	}
}

func TestListPendingRequests(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create multiple requests
	for i := 0; i < 3; i++ {
		createTestRequest(t, db)
	}

	requests, err := db.ListPendingRequests("/test/project")
	if err != nil {
		t.Fatalf("ListPendingRequests failed: %v", err)
	}

	if len(requests) != 3 {
		t.Errorf("Expected 3 pending requests, got %d", len(requests))
	}

	for _, r := range requests {
		if r.Status != StatusPending {
			t.Errorf("Expected status pending, got %s", r.Status)
		}
	}
}

func TestUpdateRequestStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, r := createTestRequest(t, db)

	// Valid transition: pending -> approved
	err := db.UpdateRequestStatus(r.ID, StatusApproved)
	if err != nil {
		t.Fatalf("UpdateRequestStatus failed: %v", err)
	}

	retrieved, err := db.GetRequest(r.ID)
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if retrieved.Status != StatusApproved {
		t.Errorf("Expected status approved, got %s", retrieved.Status)
	}

	// Valid transition: approved -> executing
	err = db.UpdateRequestStatus(r.ID, StatusExecuting)
	if err != nil {
		t.Fatalf("UpdateRequestStatus to executing failed: %v", err)
	}

	// Valid transition: executing -> executed (terminal)
	err = db.UpdateRequestStatus(r.ID, StatusExecuted)
	if err != nil {
		t.Fatalf("UpdateRequestStatus to executed failed: %v", err)
	}

	retrieved, err = db.GetRequest(r.ID)
	if err != nil {
		t.Fatalf("GetRequest failed: %v", err)
	}
	if retrieved.ResolvedAt == nil {
		t.Error("Expected ResolvedAt to be set for terminal state")
	}
}

func TestUpdateRequestStatusInvalid(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, r := createTestRequest(t, db)

	// Invalid transition: pending -> executed (skip approved)
	err := db.UpdateRequestStatus(r.ID, StatusExecuted)
	if err == nil {
		t.Error("Expected error for invalid transition")
	}
}

func TestCountPendingBySession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sess, _ := createTestRequest(t, db)
	createTestRequest(t, db) // Creates another with a new session

	count, err := db.CountPendingBySession(sess.ID)
	if err != nil {
		t.Fatalf("CountPendingBySession failed: %v", err)
	}

	// First session should have 1 pending request
	if count != 1 {
		t.Errorf("Expected 1 pending request for session, got %d", count)
	}
}

func TestFindExpiredRequests(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, r := createTestRequest(t, db)

	// Manually set expiry to past
	pastTime := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	_, err := db.Exec(`UPDATE requests SET expires_at = ? WHERE id = ?`, pastTime, r.ID)
	if err != nil {
		t.Fatalf("Failed to set old expires_at: %v", err)
	}

	expired, err := db.FindExpiredRequests()
	if err != nil {
		t.Fatalf("FindExpiredRequests failed: %v", err)
	}

	if len(expired) != 1 {
		t.Errorf("Expected 1 expired request, got %d", len(expired))
	}
}

func TestComputeCommandHash(t *testing.T) {
	cmd := CommandSpec{
		Raw:   "rm -rf /tmp/test",
		Cwd:   "/tmp",
		Argv:  []string{"rm", "-rf", "/tmp/test"},
		Shell: false,
	}

	hash1 := ComputeCommandHash(cmd)
	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	// Same command should produce same hash
	hash2 := ComputeCommandHash(cmd)
	if hash1 != hash2 {
		t.Error("Expected same hash for same command")
	}

	// Different command should produce different hash
	cmd.Raw = "rm -rf /tmp/other"
	hash3 := ComputeCommandHash(cmd)
	if hash1 == hash3 {
		t.Error("Expected different hash for different command")
	}
}

func TestSearchRequests(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, r := createTestRequest(t, db)

	// Search by command
	results, err := db.SearchRequests("rm")
	if err != nil {
		t.Fatalf("SearchRequests failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Search by justification
	results, err = db.SearchRequests("clean")
	if err != nil {
		t.Fatalf("SearchRequests by justification failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result for justification search, got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != r.ID {
		t.Errorf("Expected result ID %s, got %s", r.ID, results[0].ID)
	}

	// Search with no matches
	results, err = db.SearchRequests("nonexistent")
	if err != nil {
		t.Fatalf("SearchRequests for nonexistent failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for nonexistent search, got %d", len(results))
	}
}

func TestGetRequestWithReviews(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, r := createTestRequest(t, db)

	// Get request with no reviews yet
	retrieved, reviews, err := db.GetRequestWithReviews(r.ID)
	if err != nil {
		t.Fatalf("GetRequestWithReviews failed: %v", err)
	}

	if retrieved.ID != r.ID {
		t.Errorf("Request ID mismatch: got %s, want %s", retrieved.ID, r.ID)
	}
	if len(reviews) != 0 {
		t.Errorf("Expected 0 reviews, got %d", len(reviews))
	}

	// Test with nonexistent request
	_, _, err = db.GetRequestWithReviews("nonexistent-id")
	if err != ErrRequestNotFound {
		t.Errorf("Expected ErrRequestNotFound, got: %v", err)
	}
}

func TestListRequestsByStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create requests
	for i := 0; i < 3; i++ {
		createTestRequest(t, db)
	}

	// Get one and approve it
	requests, _ := db.ListPendingRequests("/test/project")
	if len(requests) > 0 {
		db.UpdateRequestStatus(requests[0].ID, StatusApproved)
	}

	// List by different statuses
	pending, err := db.ListRequestsByStatus(StatusPending, "/test/project")
	if err != nil {
		t.Fatalf("ListRequestsByStatus(pending) failed: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending, got %d", len(pending))
	}

	approved, err := db.ListRequestsByStatus(StatusApproved, "/test/project")
	if err != nil {
		t.Fatalf("ListRequestsByStatus(approved) failed: %v", err)
	}
	if len(approved) != 1 {
		t.Errorf("Expected 1 approved, got %d", len(approved))
	}
}

func TestListPendingRequestsAllProjects(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create requests in multiple projects
	sess1 := &Session{
		AgentName:   "Agent1",
		Program:     "claude-code",
		Model:       "opus-4.5",
		ProjectPath: "/test/project1",
	}
	db.CreateSession(sess1)

	sess2 := &Session{
		AgentName:   "Agent2",
		Program:     "claude-code",
		Model:       "opus-4.5",
		ProjectPath: "/test/project2",
	}
	db.CreateSession(sess2)

	r1 := &Request{
		ProjectPath:        "/test/project1",
		RequestorSessionID: sess1.ID,
		RequestorAgent:     sess1.AgentName,
		RequestorModel:     "opus-4.5",
		RiskTier:           RiskTierDangerous,
		MinApprovals:       1,
		Command: CommandSpec{
			Raw: "rm -rf /tmp/test1",
			Cwd: "/tmp",
		},
		Justification: Justification{Reason: "Test 1"},
	}
	db.CreateRequest(r1)

	r2 := &Request{
		ProjectPath:        "/test/project2",
		RequestorSessionID: sess2.ID,
		RequestorAgent:     sess2.AgentName,
		RequestorModel:     "opus-4.5",
		RiskTier:           RiskTierDangerous,
		MinApprovals:       1,
		Command: CommandSpec{
			Raw: "rm -rf /tmp/test2",
			Cwd: "/tmp",
		},
		Justification: Justification{Reason: "Test 2"},
	}
	db.CreateRequest(r2)

	// List all pending
	all, err := db.ListPendingRequestsAllProjects()
	if err != nil {
		t.Fatalf("ListPendingRequestsAllProjects failed: %v", err)
	}

	if len(all) != 2 {
		t.Errorf("Expected 2 pending across all projects, got %d", len(all))
	}
}

// createTestRequest creates a test session and request.
func createTestRequest(t *testing.T, db *DB) (*Session, *Request) {
	t.Helper()

	// Each call creates a unique session
	sess := &Session{
		AgentName:   "Agent-" + time.Now().Format("150405.000000"),
		Program:     "claude-code",
		Model:       "opus-4.5",
		ProjectPath: "/test/project",
	}
	if err := db.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	r := &Request{
		ProjectPath:        "/test/project",
		RequestorSessionID: sess.ID,
		RequestorAgent:     sess.AgentName,
		RequestorModel:     "opus-4.5",
		RiskTier:           RiskTierDangerous,
		MinApprovals:       1,
		Command: CommandSpec{
			Raw:  "rm -rf ./build",
			Cwd:  "/test/project",
			Argv: []string{"rm", "-rf", "./build"},
		},
		Justification: Justification{
			Reason: "Clean build directory",
		},
	}
	if err := db.CreateRequest(r); err != nil {
		t.Fatalf("CreateRequest failed: %v", err)
	}

	return sess, r
}
