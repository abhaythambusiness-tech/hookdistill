// Package report renders the scan output. Everything printed from a
// transcript passes through Redact first.
package report

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/abhaythambusiness-tech/hookdistill/internal/detect"
	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

const snippetLen = 120

// Scan reads every session under root (and claudeMDPath if non-empty) and
// writes a report to w. An empty history is not an error.
func Scan(root, claudeMDPath string, w io.Writer) error {
	ss, err := transcript.LoadAll(root)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	if len(ss) == 0 {
		fmt.Fprintf(w, "no sessions found under %s\n", root)
		return nil
	}
	var calls, msgs, skipped int
	for _, s := range ss {
		calls += len(s.Calls)
		msgs += len(s.Msgs)
		skipped += s.Skipped
	}
	fmt.Fprintf(w, "measured: %d sessions, %d tool calls, %d user messages; skipped %d unreadable lines\n\n", len(ss), calls, msgs, skipped)

	cands := detect.Corrections(ss)
	fmt.Fprintf(w, "Repeated corrections (>=3 times in >=2 sessions): %d\n", len(cands))
	for _, c := range cands {
		fmt.Fprintf(w, "  %s  %s /%s/  corrected %d times in %d sessions\n", c.Rule.ID, c.Rule.Tool, Redact(c.Rule.Pattern), c.Count, c.Sessions)
		for _, e := range c.Rule.Evidence {
			fmt.Fprintf(w, "    evidence %s:%d  %q\n", short(e.Session), e.Seq, snippet(ss, e.Session, e.Seq))
		}
	}
	if claudeMDPath == "" {
		return nil
	}
	return claudeMDSection(claudeMDPath, ss, w)
}

func claudeMDSection(path string, ss []transcript.Session, w io.Writer) error {
	conv, notConv, err := detect.ClaudeMD(path, ss)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "\nCLAUDE.md rules: %d convertible, %d not convertible\n", len(conv), len(notConv))
	for _, c := range conv {
		fmt.Fprintf(w, "  convertible  %s /%s/  matched %d past calls in %d sessions (run replay to check they were mistakes)\n", c.Rule.ID, Redact(c.Rule.Pattern), c.Count, c.Sessions)
	}
	for _, n := range notConv {
		fmt.Fprintf(w, "  not convertible  line %d (%s): %s\n", n.Line, n.Reason, Redact(n.Text))
	}
	if info, err := os.Stat(path); err == nil {
		per := detect.EstimateTokens(int(info.Size()), 1)
		fmt.Fprintf(w, "estimated cost: ~%d tokens per session, ~%d across the %d sessions read (file chars / 4; not measured)\n", per, detect.EstimateTokens(int(info.Size()), len(ss)), len(ss))
	}
	return nil
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func snippet(ss []transcript.Session, session string, seq int) string {
	for _, s := range ss {
		if s.ID != session {
			continue
		}
		for _, m := range s.Msgs {
			if m.Seq == seq {
				t := strings.Join(strings.Fields(Redact(m.Text)), " ")
				if len(t) > snippetLen {
					t = t[:snippetLen] + "..."
				}
				return t
			}
		}
	}
	return ""
}
