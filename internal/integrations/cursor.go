package integrations

import "strings"

const (
	cursorRulesStartMarker = "<!-- slb:cursor-rules:start -->"
	cursorRulesEndMarker   = "<!-- slb:cursor-rules:end -->"
)

// CursorRulesSection returns the SLB section to add to a Cursor `.cursorrules` file.
// It is wrapped in markers so it can be replaced safely.
func CursorRulesSection() string {
	const tick = "`"

	var b strings.Builder
	b.WriteString(cursorRulesStartMarker)
	b.WriteString("\n\n")
	b.WriteString("## Dangerous Command Policy (slb)\n\n")
	b.WriteString("Before running any command that might be destructive, you MUST use slb for approval:\n\n")

	b.WriteString("1. Check if command needs approval:\n   ")
	b.WriteString(tick)
	b.WriteString(`slb patterns test "<command>"`)
	b.WriteString(tick)
	b.WriteString("\n\n")

	b.WriteString("2. If approval needed, request it:\n   ")
	b.WriteString(tick)
	b.WriteString(`slb request "<command>" --reason "..." --expected-effect "..." --goal "..." --safety "..."`)
	b.WriteString(tick)
	b.WriteString("\n\n")

	b.WriteString("3. Wait for approval:\n   ")
	b.WriteString(tick)
	b.WriteString("slb status <request-id> --wait")
	b.WriteString(tick)
	b.WriteString("\n\n")

	b.WriteString("4. Execute only when approved:\n   ")
	b.WriteString(tick)
	b.WriteString("slb execute <request-id>")
	b.WriteString(tick)
	b.WriteString("\n\n")

	b.WriteString("### Risk Tiers\n")
	b.WriteString("- 🔴 CRITICAL: Requires 2+ approvals (DROP DATABASE, terraform destroy)\n")
	b.WriteString("- 🟠 DANGEROUS: Requires 1 approval (rm -rf, git reset --hard)\n")
	b.WriteString("- 🟡 CAUTION: Auto-approved after 30s (rm *.log, git branch -d)\n\n")

	b.WriteString("### Quick Reference\n")
	b.WriteString("- Start session: ")
	b.WriteString(tick)
	b.WriteString(`slb session start --agent "<name>" --program "cursor" --model "<model>"`)
	b.WriteString(tick)
	b.WriteString("\n")
	b.WriteString("- Atomic run: ")
	b.WriteString(tick)
	b.WriteString(`slb run "<command>" --reason "..."`)
	b.WriteString(tick)
	b.WriteString("\n")
	b.WriteString("- Check pending: ")
	b.WriteString(tick)
	b.WriteString("slb pending")
	b.WriteString(tick)
	b.WriteString("\n\n")

	b.WriteString("Never bypass slb for dangerous commands. The point is peer review.\n\n")
	b.WriteString(cursorRulesEndMarker)
	b.WriteString("\n")

	return b.String()
}

// CursorRulesMode determines how the section is applied to an existing `.cursorrules`.
type CursorRulesMode int

const (
	// CursorRulesAppend appends the section only if it's missing.
	CursorRulesAppend CursorRulesMode = iota
	// CursorRulesReplace replaces the existing SLB section if present; otherwise appends it.
	CursorRulesReplace
)

// ApplyCursorRules upserts the SLB Cursor rules section into an existing `.cursorrules` content.
// It returns the new content and whether it changed.
func ApplyCursorRules(existing string, mode CursorRulesMode) (string, bool) {
	section := CursorRulesSection()

	if strings.TrimSpace(existing) == "" {
		return section, true
	}

	start := strings.Index(existing, cursorRulesStartMarker)
	end := strings.Index(existing, cursorRulesEndMarker)

	if start != -1 && end != -1 && end > start {
		if mode == CursorRulesAppend {
			return existing, false
		}

		end += len(cursorRulesEndMarker)
		after := existing[end:]
		if strings.HasPrefix(after, "\n") {
			after = after[1:]
		}

		return existing[:start] + section + after, true
	}

	// No existing SLB section found → append.
	out := existing
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	if !strings.HasSuffix(out, "\n\n") {
		out += "\n"
	}
	out += section
	return out, true
}
