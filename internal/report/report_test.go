package report

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	secrets := []string{
		"sk-ant-api03-abcdefghijklmnopqrstuv",
		"ghp_" + strings.Repeat("a", 36),
		"AKIA" + strings.Repeat("A", 16),
		"Bearer abc.def",
	}
	for _, s := range secrets {
		if got := Redact("x " + s + " y"); got != "x [REDACTED] y" {
			t.Errorf("%q -> %q", s, got)
		}
	}
	if got := "plain text, git commit -m x"; Redact(got) != got {
		t.Errorf("normal text changed: %q", Redact(got))
	}
}

func line(typ, session, seq, content string) string {
	return fmt.Sprintf(`{"type":%q,"sessionId":%q,"timestamp":"2026-11-01T10:00:0%sZ","isSidechain":false,"message":{"content":%s}}`, typ, session, seq, content)
}

// writeHistory writes 2 sessions with 3 corrections of "git commit -am".
func writeHistory(t *testing.T, msg string) string {
	t.Helper()
	root := t.TempDir()
	proj := filepath.Join(root, "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := `[{"type":"tool_use","id":"t","name":"Bash","input":{"command":"git commit -am x"}}]`
	mk := func(sid string, n int) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteString(line("user", sid, "1", `"commit"`) + "\n")
			b.WriteString(line("assistant", sid, "2", tool) + "\n")
			b.WriteString(line("user", sid, "3", fmt.Sprintf("%q", msg)) + "\n")
		}
		return b.String()
	}
	if err := os.WriteFile(filepath.Join(proj, "a.jsonl"), []byte(mk("a", 2)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "b.jsonl"), []byte(mk("b", 1)+"{truncated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestScanEmptyHistory(t *testing.T) {
	var out bytes.Buffer
	if err := Scan(filepath.Join(t.TempDir(), "missing"), "", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no sessions found") {
		t.Fatalf("out=%q", out.String())
	}
}

func TestScanRedactsSecrets(t *testing.T) {
	tok := "ghp_" + strings.Repeat("Z", 36)
	root := writeHistory(t, "no, do not do that "+tok)
	var out bytes.Buffer
	if err := Scan(root, "", &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), tok) || !strings.Contains(out.String(), "[REDACTED]") {
		t.Fatalf("report leaks or lacks redaction:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "corrected 3 times in 2 sessions") {
		t.Fatalf("candidate missing:\n%s", out.String())
	}
}

func TestScanLabelsEstimates(t *testing.T) {
	root := writeHistory(t, "no, do not do that")
	md := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(md, []byte("- Never run `git commit -am`.\n- Write clean code.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Scan(root, md, &out); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	for _, want := range []string{"estimated", "measured", "convertible", "not convertible", "positive instruction"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestScanReportsSkippedLines(t *testing.T) {
	root := writeHistory(t, "no, do not do that")
	var out bytes.Buffer
	if err := Scan(root, "", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "skipped 1") {
		t.Fatalf("out:\n%s", out.String())
	}
}

func TestScanDoesNotCallMatchesViolations(t *testing.T) {
	root := writeHistory(t, "no, do not do that")
	md := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(md, []byte("- Never run `git commit -am`.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Scan(root, md, &out); err != nil {
		t.Fatal(err)
	}
	if s := out.String(); strings.Contains(s, "broken") || !strings.Contains(s, "matched 3 past calls in 2 sessions") {
		t.Fatalf("wording:\n%s", s)
	}
}
