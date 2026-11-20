package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
)

// seeded is a fake token planted in examples/ to prove reports redact it.
const seeded = "ghp_0123456789abcdefghijklmnopqrstuvwxyz"

func TestE2EScanProposeReplay(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "sessions")
	md := filepath.Join("..", "..", "examples", "CLAUDE.md")
	rf := filepath.Join(t.TempDir(), ".claude", "hookdistill.rules.json")
	var all strings.Builder

	code, out, e := do("scan", "--root", root, "--claude-md", md)
	all.WriteString(out + e)
	if code != 0 || !strings.Contains(out, "measured: 6 sessions") {
		t.Fatalf("scan exit %d: %s%s", code, out, e)
	}
	code, out, e = do("propose", "--root", root, "--claude-md", md, "--out", rf)
	all.WriteString(out + e)
	if code != 0 {
		t.Fatalf("propose exit %d: %s", code, e)
	}
	rs, err := rules.Load(rf)
	if err != nil || len(rs) < 3 {
		t.Fatalf("want >=3 rules, got %d (err %v)", len(rs), err)
	}
	for _, r := range rs {
		if r.Action != "warn" {
			t.Fatalf("rule %s not born as warn", r.ID)
		}
	}
	hits := 0
	for _, r := range rs {
		code, out, e = do("replay", r.ID, "--root", root, "--rules", rf)
		all.WriteString(out + e)
		if code != 0 {
			t.Fatalf("replay %s exit %d: %s", r.ID, code, e)
		}
		hits += strings.Count(out, "\n  ")
	}
	if hits < 1 {
		t.Fatal("no replay hits")
	}
	if strings.Contains(all.String(), seeded) {
		t.Fatalf("seeded secret leaked into output:\n%s", all.String())
	}
}
