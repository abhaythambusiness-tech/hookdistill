// Package hook is the PreToolUse engine behind `hookdistill run`.
// It fails open: any problem of its own (bad input, unreadable or invalid
// rules) is reported on stderr and never blocks the user's tool call.
package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

type input struct {
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
}

// warnOutput is the only place that knows how a warning is shown to the user:
// PreToolUse permissionDecision "ask" with the rule message as the reason
// (contract checked against the Claude Code hooks docs on 2026-11-20).
func warnOutput(msg string) []byte {
	b, _ := json.Marshal(map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "ask",
			"permissionDecisionReason": msg,
		},
	})
	return b
}

// Run evaluates one tool call read from stdin against the rule files.
// It returns the process exit code: 2 blocks, 0 allows (with an optional warning on stdout).
func Run(stdin io.Reader, stdout, stderr io.Writer, ruleFiles []string) int {
	raw, err := io.ReadAll(io.LimitReader(stdin, 8<<20))
	if err != nil {
		fmt.Fprintf(stderr, "hookdistill: cannot read input: %v\n", err)
		return 0
	}
	var in input
	if err := json.Unmarshal(raw, &in); err != nil {
		fmt.Fprintf(stderr, "hookdistill: input is not valid hook JSON, allowing: %v\n", err)
		return 0
	}
	call := transcript.ToolCall{Tool: in.ToolName, Input: in.ToolInput}
	subject := call.Subject()

	var warns []string
	for _, f := range ruleFiles {
		rs, err := rules.Load(f)
		if err != nil {
			fmt.Fprintf(stderr, "hookdistill: ignoring unreadable rules: %v\n", err)
			continue
		}
		for _, r := range rs {
			if err := r.Validate(); err != nil {
				fmt.Fprintf(stderr, "hookdistill: ignoring invalid rule: %v\n", err)
				continue
			}
			if !r.Matches(call.Tool, subject) {
				continue
			}
			if r.Action == "block" {
				fmt.Fprintf(stderr, "hookdistill: blocked by rule %s: %s\n", r.ID, r.Message)
				return 2
			}
			warns = append(warns, r.Message)
		}
	}
	if len(warns) > 0 {
		stdout.Write(warnOutput(strings.Join(warns, "\n")))
	}
	return 0
}
