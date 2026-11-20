package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const bin = "/usr/local/bin/hookdistill"

func read(t *testing.T, p string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, b)
	}
	return m
}

func pre(t *testing.T, m map[string]any) []any {
	t.Helper()
	h, _ := m["hooks"].(map[string]any)
	l, _ := h["PreToolUse"].([]any)
	return l
}

func ours(l []any) int {
	n := 0
	for _, g := range l {
		for _, h := range g.(map[string]any)["hooks"].([]any) {
			if h.(map[string]any)["command"] == bin+" run" {
				n++
			}
		}
	}
	return n
}

func TestInstallCreatesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".claude", "settings.json")
	if err := Install(p, bin); err != nil {
		t.Fatal(err)
	}
	l := pre(t, read(t, p))
	g := l[0].(map[string]any)
	if len(l) != 1 || g["matcher"] != "*" || ours(l) != 1 {
		t.Fatalf("entry=%v", l)
	}
	if h := g["hooks"].([]any)[0].(map[string]any); h["type"] != "command" {
		t.Fatalf("hook=%v", h)
	}
}

const other = `{"permissions":{"allow":["Bash(ls)"]},"model":"x","big":12345678901234567890,
"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"/x/other"}]}],"Stop":[{"hooks":[{"type":"command","command":"/x/stop"}]}]}}`

func TestInstallPreservesOtherHooksAndKeys(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(p, []byte(other), 0o600)
	if err := Install(p, bin); err != nil {
		t.Fatal(err)
	}
	m := read(t, p)
	raw, _ := os.ReadFile(p)
	l := pre(t, m)
	if len(l) != 2 || ours(l) != 1 || m["model"] != "x" || m["permissions"] == nil || m["hooks"].(map[string]any)["Stop"] == nil {
		t.Fatalf("settings=%s", raw)
	}
	if !strings.Contains(string(raw), "12345678901234567890") {
		t.Fatalf("big number mangled: %s", raw)
	}
	// Windows has no unix permission bits.
	if st, _ := os.Stat(p); runtime.GOOS != "windows" && st.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v", st.Mode())
	}
}

func TestInstallIdempotent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	for i := 0; i < 2; i++ {
		if err := Install(p, bin); err != nil {
			t.Fatal(err)
		}
	}
	if n := ours(pre(t, read(t, p))); n != 1 {
		t.Fatalf("entries=%d", n)
	}
}

func TestInstallRejectsInvalidJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(p, []byte("{broken"), 0o644)
	if err := Install(p, bin); err == nil {
		t.Fatal("want error")
	}
	if b, _ := os.ReadFile(p); string(b) != "{broken" {
		t.Fatalf("file changed: %q", b)
	}
}

func TestInstallWritesBackup(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(p, []byte(other), 0o644)
	Install(p, bin)
	b, err := os.ReadFile(p + ".hookdistill.bak")
	if err != nil || string(b) != other {
		t.Fatalf("backup err=%v content=%q", err, b)
	}
	Install(p, "/other/hookdistill") // a second change must not clobber the first backup
	if b, _ := os.ReadFile(p + ".hookdistill.bak"); string(b) != other {
		t.Fatal("backup overwritten")
	}
}

func TestUninstallRemovesOnlyOurEntry(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(p, []byte(other), 0o644)
	Install(p, bin)
	if err := Uninstall(p); err != nil {
		t.Fatal(err)
	}
	m := read(t, p)
	l := pre(t, m)
	if len(l) != 1 || ours(l) != 0 || l[0].(map[string]any)["matcher"] != "Bash" || m["permissions"] == nil {
		t.Fatalf("settings=%v", m)
	}
}

func TestUninstallDropsEmptyHooksAndMissingFileIsFine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := Uninstall(p); err != nil {
		t.Fatal(err)
	}
	Install(p, bin)
	Uninstall(p)
	if m := read(t, p); m["hooks"] != nil {
		t.Fatalf("settings=%v", m)
	}
}
