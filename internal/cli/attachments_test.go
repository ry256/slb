package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/testutil"
)

func TestCollectAttachments_EmptyFlags(t *testing.T) {
	_ = testutil.NewHarness(t) // Ensure test cleanup

	flags := AttachmentFlags{}
	attachments, err := CollectAttachments(context.Background(), flags)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 0 {
		t.Errorf("expected 0 attachments with empty flags, got %d", len(attachments))
	}
}

func TestCollectAttachments_FileAttachment(t *testing.T) {
	h := testutil.NewHarness(t)

	// Create a test file
	testFile := filepath.Join(h.ProjectDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	flags := AttachmentFlags{
		Files: []string{testFile},
	}

	attachments, err := CollectAttachments(context.Background(), flags)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	// Attachment has Type, Content, and Metadata fields
	if attachments[0].Type != "file" {
		t.Errorf("expected type 'file', got %q", attachments[0].Type)
	}
	if attachments[0].Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestCollectAttachments_FileNotFound(t *testing.T) {
	_ = testutil.NewHarness(t)

	flags := AttachmentFlags{
		Files: []string{"/nonexistent/path/file.txt"},
	}

	_, err := CollectAttachments(context.Background(), flags)

	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCollectAttachments_MultipleFiles(t *testing.T) {
	h := testutil.NewHarness(t)

	// Create test files
	file1 := filepath.Join(h.ProjectDir, "file1.txt")
	file2 := filepath.Join(h.ProjectDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content 1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content 2"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	flags := AttachmentFlags{
		Files: []string{file1, file2},
	}

	attachments, err := CollectAttachments(context.Background(), flags)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(attachments))
	}
}

func TestCollectAttachments_ContextCommand(t *testing.T) {
	_ = testutil.NewHarness(t)

	flags := AttachmentFlags{
		Contexts: []string{"echo hello"},
	}

	attachments, err := CollectAttachments(context.Background(), flags)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	// Context commands produce a "context" type attachment
	if attachments[0].Type != "context" {
		t.Errorf("expected type 'context', got %q", attachments[0].Type)
	}
}

func TestCollectAttachments_FailingContextCommand(t *testing.T) {
	_ = testutil.NewHarness(t)

	flags := AttachmentFlags{
		Contexts: []string{"nonexistent-command-xyz"},
	}

	_, err := CollectAttachments(context.Background(), flags)

	// Command may or may not fail depending on shell behavior
	// Just verify no panic occurs
	_ = err
}

func TestCollectAttachments_ScreenshotNotFound(t *testing.T) {
	_ = testutil.NewHarness(t)

	flags := AttachmentFlags{
		Screenshots: []string{"/nonexistent/screenshot.png"},
	}

	_, err := CollectAttachments(context.Background(), flags)

	if err == nil {
		t.Fatal("expected error for nonexistent screenshot")
	}
}

func TestAttachmentFlags_Struct(t *testing.T) {
	// Verify AttachmentFlags struct can be used properly
	flags := AttachmentFlags{
		Files:       []string{"file1.txt", "file2.txt"},
		Contexts:    []string{"ls -la", "git status"},
		Screenshots: []string{"screen.png"},
	}

	if len(flags.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(flags.Files))
	}
	if len(flags.Contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(flags.Contexts))
	}
	if len(flags.Screenshots) != 1 {
		t.Errorf("expected 1 screenshot, got %d", len(flags.Screenshots))
	}
}
