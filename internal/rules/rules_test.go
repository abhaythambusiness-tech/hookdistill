package rules

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sample() Rule {
	return Rule{
		ID: "c-1", Evento: "PreToolUse", Herramienta: "Bash", Patron: `git commit -am`,
		Accion: "warn", Mensaje: "Do not use commit -am", Origen: "correccion",
		Evidencia: []Evidence{{Session: "s1", Seq: 4}},
	}
}

func TestRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".claude", "hookdistill.rules.json")
	in := []Rule{sample(), {ID: "c-2", Evento: "PreToolUse", Herramienta: "*", Patron: "rm -rf", Accion: "block", Mensaje: "no", Origen: "CLAUDE.md"}}
	if err := Save(p, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("got %+v want %+v", out, in)
	}
}

func TestLoadMissingFile(t *testing.T) {
	rs, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if rs != nil || err != nil {
		t.Fatalf("rs=%v err=%v", rs, err)
	}
}

func TestLoadBadLineReportsLineNumber(t *testing.T) {
	p := filepath.Join(t.TempDir(), "r.json")
	os.WriteFile(p, []byte("{\"id\":\"a\"}\n{broken\n"), 0o644)
	if _, err := Load(p); err == nil {
		t.Fatal("expected error")
	} else if want := "line 2"; !strings.Contains(err.Error(), want) {
		t.Fatalf("err=%q want mention of %q", err, want)
	}
}

func TestValidateRejectsBadRegex(t *testing.T) {
	r := sample()
	r.Patron = "("
	if r.Validate() == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestValidateRejectsBadAction(t *testing.T) {
	r := sample()
	r.Accion = "delete"
	if r.Validate() == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestValidateAcceptsSample(t *testing.T) {
	if err := sample().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		name          string
		r             Rule
		tool, subject string
		want          bool
	}{
		{"bash match", sample(), "Bash", "git commit -am 'x'", true},
		{"other tool", sample(), "Edit", "git commit -am", false},
		{"no match", sample(), "Bash", "git commit -m 'x'", false},
		{"wildcard tool", Rule{Herramienta: "*", Patron: "secret"}, "Read", "/tmp/secret.txt", true},
		{"empty tool means any", Rule{Patron: "secret"}, "Write", "/tmp/secret.txt", true},
		{"empty pattern never matches", Rule{Herramienta: "*"}, "Bash", "ls", false},
		{"invalid regex never matches", Rule{Herramienta: "*", Patron: "("}, "Bash", "(", false},
	}
	for _, c := range cases {
		if got := c.r.Matches(c.tool, c.subject); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestNewIDStable(t *testing.T) {
	a := NewID("Bash", "git commit -am")
	if a != NewID("Bash", "git commit -am") || a == NewID("Edit", "git commit -am") || !strings.HasPrefix(a, "c-") || len(a) != 10 {
		t.Fatalf("id=%q", a)
	}
}
