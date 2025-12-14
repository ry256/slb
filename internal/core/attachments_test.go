package core

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/slb/internal/db"
)

func TestDefaultAttachmentConfig(t *testing.T) {
	cfg := DefaultAttachmentConfig()
	if cfg.MaxFileSize <= 0 {
		t.Fatalf("expected MaxFileSize > 0, got %d", cfg.MaxFileSize)
	}
	if cfg.MaxOutputSize <= 0 {
		t.Fatalf("expected MaxOutputSize > 0, got %d", cfg.MaxOutputSize)
	}
	if cfg.MaxCommandRuntime <= 0 {
		t.Fatalf("expected MaxCommandRuntime > 0, got %s", cfg.MaxCommandRuntime)
	}
	if cfg.MaxImageSize <= 0 {
		t.Fatalf("expected MaxImageSize > 0, got %d", cfg.MaxImageSize)
	}
}

func TestAttachmentError_Error(t *testing.T) {
	err := &AttachmentError{
		Type:    db.AttachmentTypeFile,
		Path:    "foo.txt",
		Message: "nope",
	}
	if got := err.Error(); !strings.Contains(got, "attachment error") || !strings.Contains(got, "nope") {
		t.Fatalf("unexpected Error(): %q", got)
	}
}

func TestCappedBuffer(t *testing.T) {
	var nilBuf *cappedBuffer
	if n, err := nilBuf.Write([]byte("abc")); err != nil || n != 3 {
		t.Fatalf("nil Write() = (%d,%v)", n, err)
	}
	if nilBuf.String() != "" {
		t.Fatalf("nil String()=%q", nilBuf.String())
	}
	if nilBuf.Truncated() {
		t.Fatalf("nil Truncated()=true")
	}

	b := &cappedBuffer{max: 5}
	if _, err := b.Write([]byte("hello")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if b.String() != "hello" {
		t.Fatalf("String=%q want %q", b.String(), "hello")
	}
	if b.Truncated() {
		t.Fatalf("expected not truncated")
	}
	if _, err := b.Write([]byte("world")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !b.Truncated() {
		t.Fatalf("expected truncated")
	}
	if b.String() != "hello" {
		t.Fatalf("String=%q want %q", b.String(), "hello")
	}
}

func TestCreateDiffAttachmentAndHelpers(t *testing.T) {
	diff := CreateDiffAttachment("diff --git a/x b/x\n", "HEAD")
	if diff.Type != db.AttachmentTypeGitDiff {
		t.Fatalf("Type=%q want %q", diff.Type, db.AttachmentTypeGitDiff)
	}
	if diff.Metadata["ref"] != "HEAD" {
		t.Fatalf("ref=%v want %q", diff.Metadata["ref"], "HEAD")
	}

	if !isImageFile("x.PNG") || isImageFile("x.txt") {
		t.Fatalf("isImageFile unexpected results")
	}
	if detectImageMimeType("x.png") != "image/png" {
		t.Fatalf("detectImageMimeType png unexpected")
	}
	if detectImageMimeType("x.unknown") != "application/octet-stream" {
		t.Fatalf("detectImageMimeType default unexpected")
	}
	if !isDiffFile("x.diff") || !isDiffFile("x.patch") || isDiffFile("x.txt") {
		t.Fatalf("isDiffFile unexpected results")
	}
	if !isDiffContent([]byte("diff --git a/x b/x\n")) || !isDiffContent([]byte("--- a/x\n")) || !isDiffContent([]byte("@@ -1 +1 @@\n")) {
		t.Fatalf("isDiffContent expected true for diff markers")
	}
	if isDiffContent([]byte("nope")) {
		t.Fatalf("isDiffContent expected false")
	}
}

func TestCreateLogExcerpt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("a\nb\nc\nd\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	att, err := CreateLogExcerpt(path, 2, 3, nil)
	if err != nil {
		t.Fatalf("CreateLogExcerpt: %v", err)
	}
	if att.Type != db.AttachmentTypeFile {
		t.Fatalf("Type=%q want %q", att.Type, db.AttachmentTypeFile)
	}
	if strings.TrimSpace(att.Content) != "b\nc" {
		t.Fatalf("Content=%q", att.Content)
	}
	if att.Metadata["type"] != "log_excerpt" {
		t.Fatalf("metadata.type=%v", att.Metadata["type"])
	}
}

func TestLoadAttachmentFromFile_DiffAndSizeLimit(t *testing.T) {
	dir := t.TempDir()

	diffPath := filepath.Join(dir, "x.diff")
	if err := os.WriteFile(diffPath, []byte("diff --git a/x b/x\n"), 0o644); err != nil {
		t.Fatalf("write diff: %v", err)
	}

	att, err := LoadAttachmentFromFile(diffPath, nil)
	if err != nil {
		t.Fatalf("LoadAttachmentFromFile(diff): %v", err)
	}
	if att.Type != db.AttachmentTypeGitDiff {
		t.Fatalf("Type=%q want %q", att.Type, db.AttachmentTypeGitDiff)
	}

	smallCfg := &AttachmentConfig{MaxFileSize: 1}
	if _, err := LoadAttachmentFromFile(diffPath, smallCfg); err == nil {
		t.Fatalf("expected size limit error")
	}
}

func TestLoadScreenshot_ValidAndRejectsNonImage(t *testing.T) {
	dir := t.TempDir()

	txtPath := filepath.Join(dir, "x.txt")
	if err := os.WriteFile(txtPath, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadScreenshot(txtPath, nil); err == nil {
		t.Fatalf("expected error for non-image")
	}

	pngPath := filepath.Join(dir, "x.png")
	writeTinyPNG(t, pngPath, 8, 8)

	att, err := LoadScreenshot(pngPath, nil)
	if err != nil {
		t.Fatalf("LoadScreenshot: %v", err)
	}
	if att.Type != db.AttachmentTypeScreenshot {
		t.Fatalf("Type=%q want %q", att.Type, db.AttachmentTypeScreenshot)
	}
	if !strings.HasPrefix(att.Content, "data:image/png;base64,") {
		t.Fatalf("unexpected data URI prefix: %q", att.Content[:min(len(att.Content), 24)])
	}

	tooSmall := &AttachmentConfig{MaxImageSize: 1}
	if _, err := LoadScreenshot(pngPath, tooSmall); err == nil {
		t.Fatalf("expected image too large error")
	}
}

func TestLoadAttachmentFromFile_ImageBecomesScreenshot(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "x.png")
	writeTinyPNG(t, pngPath, 2, 2)

	att, err := LoadAttachmentFromFile(pngPath, nil)
	if err != nil {
		t.Fatalf("LoadAttachmentFromFile: %v", err)
	}
	if att.Type != db.AttachmentTypeScreenshot {
		t.Fatalf("Type=%q want %q", att.Type, db.AttachmentTypeScreenshot)
	}
	if !strings.HasPrefix(att.Content, "data:image/png;base64,") {
		t.Fatalf("unexpected data URI prefix")
	}
}

func TestRunContextCommand_BasicAndTruncation(t *testing.T) {
	cfg := DefaultAttachmentConfig()
	cfg.MaxCommandRuntime = 0

	att, err := RunContextCommand(context.Background(), "echo hello", &cfg)
	if err != nil {
		t.Fatalf("RunContextCommand: %v", err)
	}
	if att.Type != db.AttachmentTypeContext {
		t.Fatalf("Type=%q want %q", att.Type, db.AttachmentTypeContext)
	}
	if !strings.Contains(strings.ToLower(att.Content), "hello") {
		t.Fatalf("Content=%q", att.Content)
	}

	// Truncation + stderr separator path.
	cfg.MaxOutputSize = 5
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo out & echo err 1>&2"
	} else {
		cmd = "echo out; echo err 1>&2"
	}
	att, err = RunContextCommand(context.Background(), cmd, &cfg)
	if err != nil {
		t.Fatalf("RunContextCommand(trunc): %v", err)
	}
	if _, ok := att.Metadata["truncated"]; !ok {
		t.Fatalf("expected truncated metadata")
	}
	if !strings.Contains(att.Content, "[truncated]") {
		t.Fatalf("expected content to include truncation marker, got %q", att.Content)
	}
}

func TestRunContextCommand_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip timeout test on windows")
	}

	cfg := DefaultAttachmentConfig()
	cfg.MaxCommandRuntime = 10 * time.Millisecond

	att, err := RunContextCommand(context.Background(), "sleep 2", &cfg)
	if err != nil {
		t.Fatalf("RunContextCommand(timeout): %v", err)
	}
	if _, ok := att.Metadata["timed_out"]; !ok {
		t.Fatalf("expected timed_out metadata, got %v", att.Metadata)
	}
}

func writeTinyPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

