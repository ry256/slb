// Package integrations implements external service integrations for SLB.
// Supports Claude Code, Codex CLI, Cursor, and other agent frameworks.
package integrations

// AgentType represents a supported agent type.
type AgentType string

const (
	AgentClaudeCode AgentType = "claude-code"
	AgentCodexCLI   AgentType = "codex-cli"
	AgentCursor     AgentType = "cursor"
	AgentAider      AgentType = "aider"
	AgentCustom     AgentType = "custom"
)

// Agent represents an integrated agent.
type Agent struct {
	Name    string
	Type    AgentType
	Model   string
	Session string
}

// DetectAgent attempts to detect the current agent from environment.
func DetectAgent() *Agent {
	// TODO: Implement agent detection from env vars
	// Check for CLAUDE_CODE_*, CODEX_*, CURSOR_* etc.
	return nil
}
