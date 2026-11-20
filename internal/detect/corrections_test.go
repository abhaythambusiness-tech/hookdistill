package detect

import (
	"regexp"
	"testing"

	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

func TestCorrectionsClusters(t *testing.T) {
	a := mkSession("a",
		say("commit please"), call("Bash", "git commit -am x"), say("no, do not do that"),
		call("Bash", "git commit -am y"), say("stop that again"))
	b := mkSession("b", say("commit"), call("Bash", "git commit -am z"), say("I said never use -am"))
	got := Corrections([]transcript.Session{a, b})
	if len(got) != 1 {
		t.Fatalf("candidates=%d want 1: %+v", len(got), got)
	}
	c := got[0]
	if c.Count != 3 || c.Sessions != 2 || c.Rule.Origin != "correction" || c.Rule.Action != "warn" || c.Rule.Tool != "Bash" {
		t.Fatalf("candidate=%+v", c)
	}
	if !c.Rule.Matches("Bash", "git commit -am 'wip'") || c.Rule.Matches("Bash", "git commit -m 'wip'") {
		t.Fatalf("pattern %q should match -am but not -m", c.Rule.Pattern)
	}
	if err := c.Rule.Validate(); err != nil || len(c.Rule.Evidence) != 3 {
		t.Fatalf("validate=%v evidence=%d", err, len(c.Rule.Evidence))
	}
}

func TestCorrectionsFileTool(t *testing.T) {
	a := mkSession("a", say("fix"), call("Edit", "/p/migrations/001.sql"), say("no, do not touch migrations"),
		call("Edit", "/p/migrations/002.sql"), say("stop editing migrations"))
	b := mkSession("b", say("fix"), call("Edit", "/p/migrations/003.sql"), say("I said never edit those"))
	got := Corrections([]transcript.Session{a, b})
	if len(got) != 1 || got[0].Rule.Tool != "Edit" {
		t.Fatalf("got=%+v", got)
	}
	if !got[0].Rule.Matches("Edit", "/p/migrations/009.sql") || got[0].Rule.Matches("Edit", "/p/src/a.go") {
		t.Fatalf("pattern %q", got[0].Rule.Pattern)
	}
}

func TestCorrectionsBelowThreshold(t *testing.T) {
	a := mkSession("a", say("go"), call("Bash", "rm -rf x"), say("no, do not do that"))
	b := mkSession("b", say("go"), call("Bash", "rm -rf y"), say("stop that"))
	if got := Corrections([]transcript.Session{a, b}); len(got) != 0 {
		t.Fatalf("got=%+v", got)
	}
}

func TestCorrectionsSameSessionOnlyNotEnough(t *testing.T) {
	a := mkSession("a", say("go"),
		call("Bash", "rm -rf x"), say("no, do not do that"),
		call("Bash", "rm -rf y"), say("stop that"),
		call("Bash", "rm -rf z"), say("I said never"))
	if got := Corrections([]transcript.Session{a}); len(got) != 0 {
		t.Fatalf("got=%+v", got)
	}
}

func TestCorrectionsSpanish(t *testing.T) {
	a := mkSession("a", say("sube"), call("Bash", "git push --force origin main"), say("ya te dije que no"),
		call("Bash", "git push --force origin dev"), say("otra vez lo mismo"))
	b := mkSession("b", say("sube"), call("Bash", "git push --force origin x"), say("no hagas eso"))
	got := Corrections([]transcript.Session{a, b})
	if len(got) != 1 || got[0].Count != 3 {
		t.Fatalf("got=%+v", got)
	}
}

func TestIsCorrection(t *testing.T) {
	yes := []string{"no, do not do that", "stop doing that", "ya te dije que no", "otra vez lo mismo", "I said never commit", "no hagas eso", "No, eso está mal"}
	no := []string{"no hay problema, sigue", "sounds good, thanks", "perfect, continue", "no worries, go ahead", "don't worry about it", "great: now run the tests", "yes, do it", "no pasa nada"}
	for _, s := range yes {
		if !isCorrection(s) {
			t.Errorf("should be correction: %q", s)
		}
	}
	for _, s := range no {
		if isCorrection(s) {
			t.Errorf("should NOT be correction: %q", s)
		}
	}
}

func TestSignatureAndPattern(t *testing.T) {
	cases := []struct{ tool, subj, wantSig string }{
		{"Bash", "git commit -am x", "git commit -am"},
		{"Bash", "ls -la /x", "ls -la"},
		{"Bash", "npm install left-pad", "npm install"},
		{"Bash", "make", "make"},
	}
	for _, c := range cases {
		tc := transcript.ToolCall{Tool: c.tool, Input: map[string]any{"command": c.subj}}
		if got := signature(tc); got != c.wantSig {
			t.Errorf("%q: sig=%q want %q", c.subj, got, c.wantSig)
		}
	}
}

func TestSidechainCallsIgnored(t *testing.T) {
	a := mkSession("a", say("go"), sideCall("Bash", "rm -rf x"), say("no, do not do that"), sideCall("Bash", "rm -rf y"), say("stop that"))
	b := mkSession("b", say("go"), sideCall("Bash", "rm -rf z"), say("I said never"))
	if got := Corrections([]transcript.Session{a, b}); len(got) != 0 {
		t.Fatalf("got=%+v", got)
	}
}

func TestPatternIsValidRegexp(t *testing.T) {
	tc := transcript.ToolCall{Tool: "Bash", Input: map[string]any{"command": "git commit -am x"}}
	if _, err := regexp.Compile(patternFor(tc)); err != nil {
		t.Fatal(err)
	}
}
