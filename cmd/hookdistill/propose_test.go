package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
)

// history writes 2 sessions in which "git commit -am" is corrected 3 times.
func history(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	tool := `[{"type":"tool_use","id":"t","name":"Bash","input":{"command":"git commit -am x"}}]`
	mk := func(sid string, n int) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			for j, c := range []struct{ typ, content string }{{"user", `"commit"`}, {"assistant", tool}, {"user", `"no, do not do that"`}} {
				fmt.Fprintf(&b, `{"type":%q,"sessionId":%q,"timestamp":"2026-11-01T10:00:0%dZ","isSidechain":false,"message":{"content":%s}}`+"\n", c.typ, sid, j, c.content)
			}
		}
		return b.String()
	}
	proj := filepath.Join(root, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"a.jsonl": mk("a", 2), "b.jsonl": mk("b", 1)} {
		if err := os.WriteFile(filepath.Join(proj, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func do(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

func proposed(t *testing.T) (root, rulesFile, id string) {
	t.Helper()
	root = history(t)
	rulesFile = filepath.Join(t.TempDir(), ".claude", "hookdistill.rules.json")
	if code, _, e := do("propose", "--root", root, "--claude-md", "", "--out", rulesFile); code != 0 {
		t.Fatalf("propose exit %d: %s", code, e)
	}
	rs, err := rules.Load(rulesFile)
	if err != nil || len(rs) != 1 {
		t.Fatalf("rules=%v err=%v", rs, err)
	}
	return root, rulesFile, rs[0].ID
}

func TestProposeWritesWarnRules(t *testing.T) {
	_, f, _ := proposed(t)
	rs, _ := rules.Load(f)
	if rs[0].Action != "warn" || rs[0].Origin != "correction" || rs[0].Validate() != nil {
		t.Fatalf("rule=%+v", rs[0])
	}
}

func TestProposeIsIdempotent(t *testing.T) {
	root, f, _ := proposed(t)
	code, out, _ := do("propose", "--root", root, "--claude-md", "", "--out", f)
	rs, _ := rules.Load(f)
	if code != 0 || len(rs) != 1 || !strings.Contains(out, "added 0") {
		t.Fatalf("code=%d rules=%d out=%q", code, len(rs), out)
	}
}

func TestProposeKeepsExistingRules(t *testing.T) {
	root := history(t)
	f := filepath.Join(t.TempDir(), "r.json")
	mine := rules.Rule{ID: "mine", Event: "PreToolUse", Tool: "Bash", Pattern: "foo", Action: "block", Origin: "correction", Message: "keep"}
	if err := rules.Save(f, []rules.Rule{mine}); err != nil {
		t.Fatal(err)
	}
	if code, _, e := do("propose", "--root", root, "--claude-md", "", "--out", f); code != 0 {
		t.Fatal(e)
	}
	rs, _ := rules.Load(f)
	if len(rs) != 2 || rs[0].ID != "mine" || rs[0].Action != "block" {
		t.Fatalf("rules=%+v", rs)
	}
}

func TestReplayPrintsHits(t *testing.T) {
	root, f, id := proposed(t)
	code, out, e := do("replay", id, "--root", root, "--rules", f)
	if code != 0 || strings.Count(out, "git commit -am x") != 3 {
		t.Fatalf("code=%d err=%s out=%q", code, e, out)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(f), "hookdistill.replay.json")); err != nil {
		t.Fatal(err)
	}
}

func TestPromoteRefusesLegit(t *testing.T) {
	root, f, id := proposed(t)
	do("replay", id, "--root", root, "--rules", f)
	code, _, e := do("promote", id, "--rules", f, "--legit", "a:2")
	rs, _ := rules.Load(f)
	if code != 1 || !strings.Contains(e, "legitimate") || rs[0].Action != "warn" {
		t.Fatalf("code=%d err=%q action=%s", code, e, rs[0].Action)
	}
}

func TestPromoteOK(t *testing.T) {
	root, f, id := proposed(t)
	do("replay", id, "--root", root, "--rules", f)
	if code, _, e := do("promote", id, "--rules", f); code != 0 {
		t.Fatalf("exit %d: %s", code, e)
	}
	if rs, _ := rules.Load(f); rs[0].Action != "block" {
		t.Fatalf("rule=%+v", rs[0])
	}
}

func TestPromoteRequiresReplay(t *testing.T) {
	_, f, id := proposed(t)
	code, _, e := do("promote", id, "--rules", f)
	if code != 1 || !strings.Contains(e, "run replay first") {
		t.Fatalf("code=%d err=%q", code, e)
	}
}

func TestPromoteUnknownID(t *testing.T) {
	_, f, _ := proposed(t)
	code, _, e := do("promote", "c-nope", "--rules", f)
	if code != 1 || !strings.Contains(e, "unknown rule") {
		t.Fatalf("code=%d err=%q", code, e)
	}
}
