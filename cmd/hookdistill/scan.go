package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/abhaythambusiness-tech/hookdistill/internal/report"
)

func init() {
	commands["scan"] = func(args []string, _ io.Reader, stdout, stderr io.Writer) int {
		fs := flag.NewFlagSet("scan", flag.ContinueOnError)
		fs.SetOutput(stderr)
		home, _ := os.UserHomeDir()
		root := fs.String("root", filepath.Join(home, ".claude", "projects"), "directory with Claude Code session transcripts")
		md := fs.String("claude-md", "./CLAUDE.md", "CLAUDE.md to analyze (skipped if the default is missing)")
		if err := fs.Parse(args); err != nil {
			return 2
		}
		path := resolveClaudeMD(fs, *md)
		if err := report.Scan(*root, path, stdout); err != nil {
			fmt.Fprintf(stderr, "hookdistill: %v\n", err)
			return 1
		}
		return 0
	}
}
