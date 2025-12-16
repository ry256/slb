package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/db"
	"github.com/Dicklesworthstone/slb/internal/testutil"
	"github.com/spf13/cobra"
)

// newTestRejectCmd creates a fresh reject command for testing.
func newTestRejectCmd(dbPath string) *cobra.Command {
	root := &cobra.Command{
		Use:           "slb",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagDB, "db", dbPath, "database path")
	root.PersistentFlags().StringVarP(&flagOutput, "output", "o", "text", "output format")
	root.PersistentFlags().BoolVarP(&flagJSON, "json", "j", false, "json output")
	root.PersistentFlags().StringVarP(&flagProject, "project", "C", "", "project directory")
	root.PersistentFlags().StringVarP(&flagConfig, "config", "c", "", "config file")

	root.AddCommand(rejectCmd)

	return root
}

func resetRejectFlags() {
	flagDB = ""
	flagOutput = "text"
	flagJSON = false
	flagProject = ""
	flagConfig = ""
	flagRejectSessionID = ""
	flagRejectSessionKey = ""
	flagRejectReason = ""
	flagRejectComments = ""
}

func TestRejectCommand_RequiresRequestID(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	cmd := newTestRejectCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "reject")

	if err == nil {
		t.Fatal("expected error when request ID is missing")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRejectCommand_RequiresSessionID(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	cmd := newTestRejectCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "reject", "some-request-id")

	if err == nil {
		t.Fatal("expected error when --session-id is missing")
	}
	if !strings.Contains(err.Error(), "--session-id is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRejectCommand_RequiresSessionKey(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	cmd := newTestRejectCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "reject", "some-request-id", "-s", "session-123")

	if err == nil {
		t.Fatal("expected error when --session-key is missing")
	}
	if !strings.Contains(err.Error(), "--session-key is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRejectCommand_RequiresReason(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	cmd := newTestRejectCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "reject", "some-request-id",
		"-s", "session-123",
		"-k", "some-key",
	)

	if err == nil {
		t.Fatal("expected error when --reason is missing")
	}
	if !strings.Contains(err.Error(), "--reason is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRejectCommand_RejectsRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	// Create requestor session
	requestorSess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("Requestor"),
		testutil.WithModel("model-a"),
	)

	// Create reviewer session with different model
	reviewerSess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("Reviewer"),
		testutil.WithModel("model-b"),
	)

	// Create request
	req := testutil.MakeRequest(t, h.DB, requestorSess,
		testutil.WithCommand("rm -rf ./build", h.ProjectDir, true),
		testutil.WithRisk(db.RiskTierDangerous),
	)
	// Set MinApprovals to 1 and RequireDifferentModel to false for simpler test
	h.DB.Exec(`UPDATE requests SET min_approvals = 1, require_different_model = false WHERE id = ?`, req.ID)

	cmd := newTestRejectCmd(h.DBPath)
	stdout, err := executeCommandCapture(t, cmd, "reject", req.ID,
		"-s", reviewerSess.ID,
		"-k", reviewerSess.SessionKey,
		"-r", "Command too dangerous",
		"-C", h.ProjectDir,
		"-j",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nstdout: %s", err, stdout)
	}

	// Verify result
	if result["request_id"] != req.ID {
		t.Errorf("expected request_id=%s, got %v", req.ID, result["request_id"])
	}
	if result["decision"] != "reject" {
		t.Errorf("expected decision=reject, got %v", result["decision"])
	}
	if result["reason"] != "Command too dangerous" {
		t.Errorf("expected reason='Command too dangerous', got %v", result["reason"])
	}
	if result["rejections"].(float64) != 1 {
		t.Errorf("expected rejections=1, got %v", result["rejections"])
	}
	if result["request_status_changed"] != true {
		t.Errorf("expected request_status_changed=true, got %v", result["request_status_changed"])
	}
	if result["new_request_status"] != string(db.StatusRejected) {
		t.Errorf("expected new_request_status=rejected, got %v", result["new_request_status"])
	}
}

func TestRejectCommand_WithComments(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	requestorSess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("Requestor"),
	)
	reviewerSess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("Reviewer"),
	)

	req := testutil.MakeRequest(t, h.DB, requestorSess)
	h.DB.Exec(`UPDATE requests SET min_approvals = 1, require_different_model = false WHERE id = ?`, req.ID)

	cmd := newTestRejectCmd(h.DBPath)
	stdout, err := executeCommandCapture(t, cmd, "reject", req.ID,
		"-s", reviewerSess.ID,
		"-k", reviewerSess.SessionKey,
		"-r", "Insufficient justification",
		"-m", "Please add more context about why this is needed",
		"-C", h.ProjectDir,
		"-j",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nstdout: %s", err, stdout)
	}

	// Verify review was created with combined comments
	reviews, _ := h.DB.ListReviewsForRequest(req.ID)
	if len(reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(reviews))
	}
	// Comments should be reason + "\n\n" + comments
	expectedComments := "Insufficient justification\n\nPlease add more context about why this is needed"
	if reviews[0].Comments != expectedComments {
		t.Errorf("expected comments=%q, got %q", expectedComments, reviews[0].Comments)
	}
}

func TestRejectCommand_SelfReviewPrevented(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	// Create a session that is both requestor and reviewer
	sess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("SelfReviewer"),
	)

	req := testutil.MakeRequest(t, h.DB, sess)
	h.DB.Exec(`UPDATE requests SET min_approvals = 1, require_different_model = false WHERE id = ?`, req.ID)

	cmd := newTestRejectCmd(h.DBPath)
	_, err := executeCommandCapture(t, cmd, "reject", req.ID,
		"-s", sess.ID,
		"-k", sess.SessionKey,
		"-r", "Trying to reject own request",
		"-C", h.ProjectDir,
		"-j",
	)

	if err == nil {
		t.Fatal("expected error for self-review")
	}
	if !strings.Contains(err.Error(), "own request") && !strings.Contains(err.Error(), "self") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRejectCommand_InvalidSessionKey(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	requestorSess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("Requestor"),
	)
	reviewerSess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("Reviewer"),
	)

	req := testutil.MakeRequest(t, h.DB, requestorSess)

	cmd := newTestRejectCmd(h.DBPath)
	_, err := executeCommandCapture(t, cmd, "reject", req.ID,
		"-s", reviewerSess.ID,
		"-k", "wrong-key-not-matching",
		"-r", "Some reason",
		"-C", h.ProjectDir,
		"-j",
	)

	if err == nil {
		t.Fatal("expected error for invalid session key")
	}
	if !strings.Contains(err.Error(), "key") && !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRejectCommand_RequestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	reviewerSess := testutil.MakeSession(t, h.DB,
		testutil.WithProject(h.ProjectDir),
		testutil.WithAgent("Reviewer"),
	)

	cmd := newTestRejectCmd(h.DBPath)
	_, err := executeCommandCapture(t, cmd, "reject", "nonexistent-request",
		"-s", reviewerSess.ID,
		"-k", reviewerSess.SessionKey,
		"-r", "Some reason",
		"-C", h.ProjectDir,
		"-j",
	)

	if err == nil {
		t.Fatal("expected error for nonexistent request")
	}
}

func TestRejectCommand_Help(t *testing.T) {
	h := testutil.NewHarness(t)
	resetRejectFlags()

	cmd := newTestRejectCmd(h.DBPath)
	stdout, _, err := executeCommand(cmd, "reject", "--help")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "reject") {
		t.Error("expected help to mention 'reject'")
	}
	if !strings.Contains(stdout, "--session-id") {
		t.Error("expected help to mention '--session-id' flag")
	}
	if !strings.Contains(stdout, "--session-key") {
		t.Error("expected help to mention '--session-key' flag")
	}
	if !strings.Contains(stdout, "--reason") {
		t.Error("expected help to mention '--reason' flag")
	}
}
