package detect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

func TestClaudeMDConvertible(t *testing.T) {
	cands, _, err := ClaudeMD("testdata/CLAUDE.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, c := range cands {
		if c.Rule.Pattern == `git push --force` || c.Rule.Pattern == `git push \-\-force` {
			found = true
			if c.Rule.Origin != "CLAUDE.md" || c.Rule.Tool != "Bash" || c.Rule.Action != "warn" || c.Rule.Event != "PreToolUse" {
				t.Fatalf("rule=%+v", c.Rule)
			}
			if err := c.Rule.Validate(); err != nil {
				t.Fatal(err)
			}
			if !c.Rule.Matches("Bash", "git push --force origin main") || c.Rule.Matches("Bash", "git push origin main") {
				t.Fatalf("pattern %q", c.Rule.Pattern)
			}
		}
	}
	if !found {
		t.Fatalf("force-push rule not produced: %+v", cands)
	}
	if len(cands) != 2 {
		t.Fatalf("want 2 candidates (git push, rm -rf), got %d: %+v", len(cands), cands)
	}
}

func TestClaudeMDNotConvertible(t *testing.T) {
	_, nc, err := ClaudeMD("testdata/CLAUDE.md", nil)
	if err != nil {
		t.Fatal(err)
	}
	reasons := map[string]string{}
	for _, n := range nc {
		if n.Reason == "" || n.Line == 0 {
			t.Fatalf("incomplete: %+v", n)
		}
		reasons[n.Text] = n.Reason
	}
	if r := reasons["Write clean code."]; r != "positive instruction" {
		t.Errorf("clean code: %q", r)
	}
	if r := reasons["Do not leave debug output."]; r != "no backticked command or path" {
		t.Errorf("debug: %q", r)
	}
	if r := reasons["Don't touch `a` or `b` files."]; r != "ambiguous" {
		t.Errorf("ambiguous: %q", r)
	}
	for text := range reasons {
		if strings.Contains(text, "ignored in a fence") {
			t.Errorf("fenced line must be skipped: %q", text)
		}
	}
}

func TestClaudeMDCountsViolations(t *testing.T) {
	a := mkSession("a", call("Bash", "git push --force origin main"), call("Bash", "ls"))
	b := mkSession("b", call("Bash", "git push --force origin dev"), sideCall("Bash", "git push --force x"))
	cands, _, err := ClaudeMD("testdata/CLAUDE.md", []transcript.Session{a, b})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cands {
		if c.Rule.Matches("Bash", "git push --force") {
			if c.Count != 2 || c.Sessions != 2 || len(c.Rule.Evidence) != 2 {
				t.Fatalf("count=%d sessions=%d ev=%d", c.Count, c.Sessions, len(c.Rule.Evidence))
			}
			return
		}
	}
	t.Fatal("force-push candidate missing")
}

func TestClaudeMDMissingFile(t *testing.T) {
	_, _, err := ClaudeMD(filepath.Join(t.TempDir(), "nope.md"), nil)
	if err == nil || !strings.Contains(err.Error(), "nope.md") {
		t.Fatalf("err=%v", err)
	}
}

func TestClaudeMDEmptyFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	c, n, err := ClaudeMD(p, nil)
	if err != nil || len(c) != 0 || len(n) != 0 {
		t.Fatalf("c=%v n=%v err=%v", c, n, err)
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := EstimateTokens(400, 10); got != 1000 {
		t.Fatalf("got %d", got)
	}
	if got := EstimateTokens(0, 5); got != 0 {
		t.Fatalf("got %d", got)
	}
}
