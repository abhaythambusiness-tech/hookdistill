package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
)

func ruleFile(t *testing.T, rs ...rules.Rule) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "r.json")
	if err := rules.Save(p, rs); err != nil {
		t.Fatal(err)
	}
	return p
}

func rule(id, accion, patron, msg string) rules.Rule {
	return rules.Rule{ID: id, Evento: "PreToolUse", Herramienta: "Bash", Patron: patron, Accion: accion, Mensaje: msg, Origen: "correccion"}
}

const bashCommit = `{"tool_name":"Bash","tool_input":{"command":"git commit -am wip"}}`

func call(input string, files ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Run(strings.NewReader(input), &out, &errb, files)
	return code, out.String(), errb.String()
}

func TestRunBlocksMatchingBlockRule(t *testing.T) {
	code, out, errs := call(bashCommit, ruleFile(t, rule("c-1", "block", "git commit -am", "never use commit -am")))
	if code != 2 || out != "" || !strings.Contains(errs, "never use commit -am") || !strings.Contains(errs, "c-1") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestRunWarnsMatchingWarnRule(t *testing.T) {
	code, out, _ := call(bashCommit, ruleFile(t, rule("c-1", "warn", "git commit -am", "careful")))
	if code != 0 || out != string(warnOutput("careful")) {
		t.Fatalf("code=%d out=%q", code, out)
	}
	if !strings.Contains(out, `"permissionDecision":"ask"`) || !strings.Contains(out, `"hookEventName":"PreToolUse"`) {
		t.Fatalf("unexpected warn contract: %s", out)
	}
}

func TestRunAllowsNonMatching(t *testing.T) {
	code, out, errs := call(`{"tool_name":"Bash","tool_input":{"command":"ls"}}`, ruleFile(t, rule("c-1", "block", "git commit -am", "x")))
	if code != 0 || out != "" || errs != "" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestRunFailsOpenOnBadRules(t *testing.T) {
	broken := filepath.Join(t.TempDir(), "broken.json")
	os.WriteFile(broken, []byte("{nope\n"), 0o644)
	badRegex := ruleFile(t, rule("c-2", "block", "(", "x"))
	code, out, errs := call(bashCommit, broken, badRegex)
	if code != 0 || out != "" || !strings.Contains(errs, "hookdistill:") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestRunFailsOpenOnBadInput(t *testing.T) {
	code, out, errs := call("this is not json", ruleFile(t, rule("c-1", "block", ".", "x")))
	if code != 0 || out != "" || !strings.Contains(errs, "hookdistill:") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestRunMergesProjectAndGlobalRules(t *testing.T) {
	warn := ruleFile(t, rule("w", "warn", "git commit", "warn msg"))
	block := ruleFile(t, rule("b", "block", "commit -am", "block msg"))
	code, out, errs := call(bashCommit, warn, block)
	if code != 2 || out != "" || !strings.Contains(errs, "block msg") {
		t.Fatalf("block must win: code=%d out=%q err=%q", code, out, errs)
	}
}
