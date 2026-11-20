package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func exec(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunUnknownCommand(t *testing.T) {
	code, _, stderr := exec("nope")
	if code != 2 || !strings.Contains(stderr, "unknown command") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestRunVersion(t *testing.T) {
	code, stdout, _ := exec("version")
	if code != 0 || !strings.HasPrefix(stdout, "hookdistill ") {
		t.Fatalf("code=%d stdout=%q", code, stdout)
	}
}

func TestRunNoArgsPrintsUsage(t *testing.T) {
	code, _, stderr := exec()
	if code != 2 || !strings.Contains(stderr, "usage:") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestRunSubcommandAllowsWithoutRules(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	wd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	var out, errb bytes.Buffer
	in := strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"ls"}}`)
	if code := run([]string{"run"}, in, &out, &errb); code != 0 || out.Len() != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errb.String())
	}
}
