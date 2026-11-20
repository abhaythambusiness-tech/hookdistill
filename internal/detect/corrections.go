// Package detect finds repeated user corrections and CLAUDE.md rules in
// session transcripts and turns them into candidate hook rules.
package detect

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

// Candidate is a proposed rule plus how often the behavior was corrected.
type Candidate struct {
	Rule     rules.Rule
	Count    int
	Sessions int
}

const (
	minCount    = 3
	minSessions = 2
	maxMsgLen   = 400
	maxEvidence = 5
)

var correctionMarks = []string{
	"don't", "do not", "stop", "never", "revert", "undo", "i said", "i told you", "wrong", "that's wrong",
	"no hagas", "ya te dije", "otra vez", "nunca", "para ya", "deja de", "revierte", "deshaz", "está mal", "esta mal",
}

var benign = []string{
	"no worries", "don't worry", "do not worry", "no hay problema", "no pasa nada", "no te preocupes", "never mind",
}

// isCorrection reports whether a user message reads as a correction of the
// action Claude just took.
func isCorrection(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" || len(t) > maxMsgLen {
		return false
	}
	for _, b := range benign {
		if strings.Contains(t, b) {
			return false
		}
	}
	if strings.HasPrefix(t, "no,") || strings.HasPrefix(t, "no ") && !strings.HasPrefix(t, "no hay") && !strings.HasPrefix(t, "no pasa") {
		rest := strings.TrimSpace(strings.TrimPrefix(t, "no,"))
		for _, ok := range []string{"gracias", "thanks", "sí", "si ", "yes"} {
			if strings.HasPrefix(rest, ok) {
				return false
			}
		}
		return true
	}
	for _, m := range correctionMarks {
		if strings.Contains(t, m) {
			return true
		}
	}
	return false
}

// signature groups similar actions: Bash by command words, file tools by dir.
func signature(c transcript.ToolCall) string {
	s := c.Subject()
	if c.Tool != "Bash" {
		return c.Tool + " " + path.Dir(s)
	}
	w := strings.Fields(s)
	if len(w) == 0 {
		return ""
	}
	out := []string{w[0]}
	rest := w[1:]
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		out = append(out, rest[0])
		rest = rest[1:]
	}
	for _, x := range rest {
		if strings.HasPrefix(x, "-") {
			out = append(out, x)
			break
		}
	}
	return strings.Join(out, " ")
}

// patternFor builds an RE2 pattern matching actions with the same signature.
func patternFor(c transcript.ToolCall) string {
	if c.Tool != "Bash" {
		return "^" + regexp.QuoteMeta(path.Dir(c.Subject())) + "/"
	}
	parts := strings.Fields(signature(c))
	q := make([]string, len(parts))
	for i, p := range parts {
		q[i] = regexp.QuoteMeta(p)
	}
	return strings.Join(q, `\s+(?:\S+\s+)*?`)
}

type group struct {
	tool, pattern string
	ev            []rules.Evidence
	sessions      map[string]bool
}

// Corrections returns candidates for actions the user corrected at least
// minCount times across at least minSessions sessions.
func Corrections(ss []transcript.Session) []Candidate {
	groups := map[string]*group{}
	for _, s := range ss {
		prev := 0
		for _, m := range s.Msgs {
			if isCorrection(m.Text) {
				if c, ok := lastCall(s.Calls, prev, m.Seq); ok {
					key := c.Tool + "|" + patternFor(c)
					g := groups[key]
					if g == nil {
						g = &group{tool: c.Tool, pattern: patternFor(c), sessions: map[string]bool{}}
						groups[key] = g
					}
					g.ev = append(g.ev, rules.Evidence{Session: s.ID, Seq: m.Seq})
					g.sessions[s.ID] = true
				}
			}
			prev = m.Seq
		}
	}
	var out []Candidate
	for _, g := range groups {
		if len(g.ev) < minCount || len(g.sessions) < minSessions {
			continue
		}
		ev := g.ev
		if len(ev) > maxEvidence {
			ev = ev[:maxEvidence]
		}
		out = append(out, Candidate{
			Count:    len(g.ev),
			Sessions: len(g.sessions),
			Rule: rules.Rule{
				ID:          rules.NewID(g.tool, g.pattern),
				Evento:      "PreToolUse",
				Herramienta: g.tool,
				Patron:      g.pattern,
				Accion:      "warn",
				Origen:      "correccion",
				Mensaje:     fmt.Sprintf("You corrected this action %d times (%d sessions). Check before repeating it.", len(g.ev), len(g.sessions)),
				Evidencia:   ev,
			},
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Rule.ID < out[j].Rule.ID
	})
	return out
}

// lastCall returns the last non-sidechain call with a subject whose Seq is
// strictly between prev and before.
func lastCall(calls []transcript.ToolCall, prev, before int) (transcript.ToolCall, bool) {
	var best transcript.ToolCall
	found := false
	for _, c := range calls {
		if c.Sidechain || c.Subject() == "" || c.Seq <= prev || c.Seq >= before {
			continue
		}
		if !found || c.Seq > best.Seq {
			best, found = c, true
		}
	}
	return best, found
}
