package detect

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

// NotConvertible is a CLAUDE.md line hookdistill cannot turn into a rule.
type NotConvertible struct {
	Line         int
	Text, Reason string
}

var (
	negation = regexp.MustCompile(`(?i)\b(never|don't|do not|must not|nunca|prohibido|no)\b`)
	ticks    = regexp.MustCompile("`([^`]+)`")
	marker   = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+`)
)

// ClaudeMD maps prohibitions in a CLAUDE.md file to Bash rules and counts how
// often the history broke each one. Lines it cannot convert come back with a
// reason. A missing file is an error.
func ClaudeMD(path string, ss []transcript.Session) ([]Candidate, []NotConvertible, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("claudemd: cannot read %s: %w", path, err)
	}
	defer f.Close()

	var cands []Candidate
	var nc []NotConvertible
	inFence := false
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for n := 1; sc.Scan(); n++ {
		raw := sc.Text()
		if strings.HasPrefix(strings.TrimSpace(raw), "```") {
			inFence = !inFence
			continue
		}
		text := strings.TrimSpace(marker.ReplaceAllString(raw, ""))
		if inFence || text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		if !negation.MatchString(text) {
			nc = append(nc, NotConvertible{Line: n, Text: text, Reason: "positive instruction"})
			continue
		}
		frags := ticks.FindAllStringSubmatch(text, -1)
		switch {
		case len(frags) == 0:
			nc = append(nc, NotConvertible{Line: n, Text: text, Reason: "no backticked command or path"})
		case len(frags) > 1:
			nc = append(nc, NotConvertible{Line: n, Text: text, Reason: "ambiguous"})
		default:
			cands = append(cands, claudeMDCandidate(n, text, frags[0][1], ss))
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("claudemd: reading %s: %w", path, err)
	}
	return cands, nc, nil
}

func claudeMDCandidate(line int, text, frag string, ss []transcript.Session) Candidate {
	r := rules.Rule{
		ID:      rules.NewID("Bash", regexp.QuoteMeta(frag)),
		Event:   "PreToolUse",
		Tool:    "Bash",
		Pattern: regexp.QuoteMeta(frag),
		Action:  "warn",
		Origin:  "CLAUDE.md",
		Message: fmt.Sprintf("CLAUDE.md line %d: %s", line, text),
	}
	seen := map[string]bool{}
	count := 0
	for _, s := range ss {
		for _, c := range s.Calls {
			if c.Sidechain || !r.Matches(c.Tool, c.Subject()) {
				continue
			}
			count++
			seen[s.ID] = true
			if len(r.Evidence) < maxEvidence {
				r.Evidence = append(r.Evidence, rules.Evidence{Session: s.ID, Seq: c.Seq})
			}
		}
	}
	return Candidate{Rule: r, Count: count, Sessions: len(seen)}
}

// EstimateTokens approximates the tokens a CLAUDE.md of the given size costs
// across the sessions that loaded it (chars/4 per session). It is an estimate.
func EstimateTokens(chars, sessionsLoaded int) int {
	return chars / 4 * sessionsLoaded
}
