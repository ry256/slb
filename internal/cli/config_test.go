package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Dicklesworthstone/slb/internal/testutil"
	"github.com/spf13/cobra"
)

// newTestConfigCmd creates a fresh config command tree for testing.
func newTestConfigCmd(dbPath string) *cobra.Command {
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

	// Create fresh config commands
	cfgCmd := &cobra.Command{
		Use:   "config",
		Short: "Show or modify SLB configuration",
		RunE:  configCmd.RunE,
	}
	cfgCmd.PersistentFlags().BoolVar(&flagConfigGlobal, "global", false, "operate on user config")

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a specific configuration value",
		Args:  cobra.ExactArgs(1),
		RunE:  configGetCmd.RunE,
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE:  configSetCmd.RunE,
	}

	cfgCmd.AddCommand(getCmd, setCmd)
	root.AddCommand(cfgCmd)

	return root
}

func resetConfigFlags() {
	flagDB = ""
	flagOutput = "text"
	flagJSON = false
	flagProject = ""
	flagConfig = ""
	flagConfigGlobal = false
}

func TestConfigCommand_ShowsConfig(t *testing.T) {
	h := testutil.NewHarness(t)
	resetConfigFlags()

	cmd := newTestConfigCmd(h.DBPath)
	stdout, err := executeCommandCapture(t, cmd, "config", "-C", h.ProjectDir, "-j")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nstdout: %s", err, stdout)
	}

	// Should have general config section (may be "general" or "General" depending on struct tags)
	hasGeneral := false
	for key := range result {
		if strings.ToLower(key) == "general" {
			hasGeneral = true
			break
		}
	}
	if !hasGeneral {
		t.Errorf("expected config to have 'general' section, got keys: %v", result)
	}
}

func TestConfigGetCommand_RequiresKey(t *testing.T) {
	h := testutil.NewHarness(t)
	resetConfigFlags()

	cmd := newTestConfigCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "config", "get")

	if err == nil {
		t.Fatal("expected error when key is missing")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigGetCommand_GetsValue(t *testing.T) {
	h := testutil.NewHarness(t)
	resetConfigFlags()

	cmd := newTestConfigCmd(h.DBPath)
	stdout, err := executeCommandCapture(t, cmd, "config", "get", "general.min_approvals", "-C", h.ProjectDir, "-j")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nstdout: %s", err, stdout)
	}

	if result["key"] != "general.min_approvals" {
		t.Errorf("expected key=general.min_approvals, got %v", result["key"])
	}
	if result["value"] == nil {
		t.Error("expected value to be set")
	}
}

func TestConfigGetCommand_UnknownKey(t *testing.T) {
	h := testutil.NewHarness(t)
	resetConfigFlags()

	cmd := newTestConfigCmd(h.DBPath)
	_, err := executeCommandCapture(t, cmd, "config", "get", "nonexistent.key", "-C", h.ProjectDir, "-j")

	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigSetCommand_RequiresKeyAndValue(t *testing.T) {
	h := testutil.NewHarness(t)
	resetConfigFlags()

	cmd := newTestConfigCmd(h.DBPath)
	_, _, err := executeCommand(cmd, "config", "set", "some.key")

	if err == nil {
		t.Fatal("expected error when value is missing")
	}
	if !strings.Contains(err.Error(), "accepts 2 arg") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigSetCommand_SetsValue(t *testing.T) {
	h := testutil.NewHarness(t)
	resetConfigFlags()

	cmd := newTestConfigCmd(h.DBPath)
	stdout, err := executeCommandCapture(t, cmd, "config", "set", "general.min_approvals", "3", "-C", h.ProjectDir, "-j")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nstdout: %s", err, stdout)
	}

	if result["key"] != "general.min_approvals" {
		t.Errorf("expected key=general.min_approvals, got %v", result["key"])
	}
	// Value should be set (could be string or number depending on implementation)
	if result["value"] == nil {
		t.Error("expected value to be set")
	}
}

func TestConfigCommand_Help(t *testing.T) {
	h := testutil.NewHarness(t)
	resetConfigFlags()

	cmd := newTestConfigCmd(h.DBPath)
	stdout, _, err := executeCommand(cmd, "config", "--help")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "config") {
		t.Error("expected help to mention 'config'")
	}
	if !strings.Contains(stdout, "get") {
		t.Error("expected help to mention 'get' subcommand")
	}
	if !strings.Contains(stdout, "set") {
		t.Error("expected help to mention 'set' subcommand")
	}
	if !strings.Contains(stdout, "--global") {
		t.Error("expected help to mention '--global' flag")
	}
}
