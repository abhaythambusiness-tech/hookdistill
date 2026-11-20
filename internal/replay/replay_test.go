package replay

import (
	"testing"

	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

func bash(session string, seq int, cmd string, side bool) transcript.ToolCall {
	return transcript.ToolCall{Session: session, Seq: seq, Tool: "Bash", Input: map[string]any{"command": cmd}, Sidechain: side}
}

func rule() rules.Rule {
	return rules.Rule{ID: "c-1", Evento: "PreToolUse", Herramienta: "Bash", Patron: `git commit -am`, Accion: "warn", Origen: "correccion", Mensaje: "m"}
}

func history() []transcript.Session {
	return []transcript.Session{
		{ID: "abcdefghijkl", Calls: []transcript.ToolCall{bash("abcdefghijkl", 3, "git commit -am x", false), bash("abcdefghijkl", 5, "git commit -m x", false)}},
		{ID: "zz", Calls: []transcript.ToolCall{bash("zz", 2, "git commit -am y", false)}},
	}
}

func TestReplayFindsHits(t *testing.T) {
	got := Replay(rule(), history())
	if len(got) != 2 || got[0].ID != "abcdefgh:3" || got[1].ID != "zz:2" || got[0].Subject != "git commit -am x" || got[0].Tool != "Bash" {
		t.Fatalf("hits=%+v", got)
	}
}

func TestReplayIgnoresSidechain(t *testing.T) {
	ss := []transcript.Session{{ID: "s", Calls: []transcript.ToolCall{bash("s", 1, "git commit -am x", true)}}}
	if got := Replay(rule(), ss); len(got) != 0 {
		t.Fatalf("hits=%+v", got)
	}
}

func TestPromoteRefusesWhenLegit(t *testing.T) {
	r := rule()
	hits := Replay(r, history())
	out, err := Promote(r, hits, map[string]bool{"zz:2": true})
	if err == nil || out.Accion == "block" || r.Accion != "warn" {
		t.Fatalf("out=%+v err=%v orig=%+v", out, err, r)
	}
}

func TestPromoteOK(t *testing.T) {
	r := rule()
	out, err := Promote(r, Replay(r, history()), nil)
	if err != nil || out.Accion != "block" || r.Accion != "warn" {
		t.Fatalf("out=%+v err=%v orig=%+v", out, err, r)
	}
}
