package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileBasic(t *testing.T) {
	s, err := LoadFile("testdata/session_basic.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "session_basic" || s.Project != "testdata" {
		t.Fatalf("id=%q project=%q", s.ID, s.Project)
	}
	if len(s.Calls) != 2 {
		t.Fatalf("calls=%d want 2", len(s.Calls))
	}
	c := s.Calls[0]
	if c.Tool != "Bash" || c.Subject() != "git commit -am x" || c.Seq != 2 || c.Sidechain {
		t.Fatalf("call0=%+v", c)
	}
	if !s.Calls[1].Sidechain || s.Calls[1].Subject() != "/a/b.go" {
		t.Fatalf("call1=%+v", s.Calls[1])
	}
}

func TestUserTextIgnoresInjectedBlocks(t *testing.T) {
	s, err := LoadFile("testdata/session_basic.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, m := range s.Msgs {
		got = append(got, m.Text)
	}
	want := []string{"please commit", "no, do not do that"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("msgs=%q want %q", got, want)
	}
	if s.Msgs[1].Seq != 5 {
		t.Fatalf("seq=%d want 5", s.Msgs[1].Seq)
	}
}

func TestLoadFileSkipsCorruptAndUnknown(t *testing.T) {
	s, err := LoadFile("testdata/session_corrupt.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if s.Skipped != 2 {
		t.Fatalf("skipped=%d want 2 (truncated line + unexpected content)", s.Skipped)
	}
	if len(s.Msgs) != 1 || len(s.Calls) != 1 {
		t.Fatalf("msgs=%d calls=%d want 1 and 1", len(s.Msgs), len(s.Calls))
	}
}

func TestSubject(t *testing.T) {
	cases := []struct {
		c    ToolCall
		want string
	}{
		{ToolCall{Tool: "Bash", Input: map[string]any{"command": "ls -la"}}, "ls -la"},
		{ToolCall{Tool: "Edit", Input: map[string]any{"file_path": "/x/y.go"}}, "/x/y.go"},
		{ToolCall{Tool: "Write", Input: map[string]any{"file_path": "/x/z.go"}}, "/x/z.go"},
		{ToolCall{Tool: "Task", Input: map[string]any{"prompt": "do"}}, ""},
		{ToolCall{Tool: "Bash", Input: map[string]any{"command": 7}}, ""},
	}
	for _, tc := range cases {
		if got := tc.c.Subject(); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.c.Tool, got, tc.want)
		}
	}
}

func TestLoadAllMissingRoot(t *testing.T) {
	ss, err := LoadAll(filepath.Join(t.TempDir(), "nope"))
	if err != nil || ss != nil {
		t.Fatalf("ss=%v err=%v", ss, err)
	}
}

func TestLoadAllFindsFiles(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "-home-x-proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("testdata/session_basic.jsonl")
	if err := os.WriteFile(filepath.Join(proj, "abc.jsonl"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	ss, err := LoadAll(root)
	if err != nil || len(ss) != 1 || ss[0].Project != "-home-x-proj" || ss[0].ID != "abc" {
		t.Fatalf("ss=%+v err=%v", ss, err)
	}
}
