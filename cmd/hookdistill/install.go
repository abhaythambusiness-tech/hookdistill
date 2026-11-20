package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/abhaythambusiness-tech/hookdistill/internal/install"
)

func settingsPath(project bool) (string, error) {
	if project {
		return filepath.Join(".claude", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func init() {
	commands["install"] = func(args []string, _ io.Reader, stdout, stderr io.Writer) int {
		fs := flag.NewFlagSet("install", flag.ContinueOnError)
		fs.SetOutput(stderr)
		project := fs.Bool("project", false, "use ./.claude/settings.json instead of ~/.claude/settings.json")
		bin := fs.String("bin", "", "path of the hookdistill binary (default: this executable)")
		if fs.Parse(args) != nil {
			return 2
		}
		path, err := settingsPath(*project)
		if err != nil {
			return fail(stderr, err)
		}
		b := *bin
		if b == "" {
			if b, err = os.Executable(); err != nil {
				return fail(stderr, err)
			}
		}
		if err := install.Install(path, b); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stdout, "installed PreToolUse hook (%s run) in %s\n", b, path)
		return 0
	}
	commands["uninstall"] = func(args []string, _ io.Reader, stdout, stderr io.Writer) int {
		fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
		fs.SetOutput(stderr)
		project := fs.Bool("project", false, "use ./.claude/settings.json instead of ~/.claude/settings.json")
		if fs.Parse(args) != nil {
			return 2
		}
		path, err := settingsPath(*project)
		if err != nil {
			return fail(stderr, err)
		}
		if err := install.Uninstall(path); err != nil {
			return fail(stderr, err)
		}
		fmt.Fprintf(stdout, "removed hookdistill hook entry from %s (if present)\n", path)
		return 0
	}
}
