// Package replay runs a rule against recorded history and gates promotion
// from warn to block on the result.
package replay

import (
	"fmt"
	"strconv"

	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

// Hit is one past tool call the rule would have matched.
type Hit struct {
	ID      string `json:"id"`
	Session string `json:"session"`
	Seq     int    `json:"seq"`
	Tool    string `json:"tool"`
	Subject string `json:"subject"`
}

// HitID builds the short id users reference when marking a hit legitimate.
func HitID(session string, seq int) string {
	if len(session) > 8 {
		session = session[:8]
	}
	return session + ":" + strconv.Itoa(seq)
}

// Replay returns every non-sidechain call in ss that r matches.
func Replay(r rules.Rule, ss []transcript.Session) []Hit {
	var out []Hit
	for _, s := range ss {
		for _, c := range s.Calls {
			if c.Sidechain || !r.Matches(c.Tool, c.Subject()) {
				continue
			}
			out = append(out, Hit{ID: HitID(s.ID, c.Seq), Session: s.ID, Seq: c.Seq, Tool: c.Tool, Subject: c.Subject()})
		}
	}
	return out
}

// Promote returns r with Action "block", unless any hit is marked legitimate:
// a rule that would have blocked work you wanted must stay a warning.
func Promote(r rules.Rule, hits []Hit, legit map[string]bool) (rules.Rule, error) {
	for _, h := range hits {
		if legit[h.ID] {
			return r, fmt.Errorf("replay: rule %s would have blocked a legitimate call (%s: %s); keeping it as warn", r.ID, h.ID, h.Subject)
		}
	}
	r.Action = "block"
	return r, nil
}
