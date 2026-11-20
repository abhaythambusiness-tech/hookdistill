package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/abhaythambusiness-tech/hookdistill/internal/hook"
)

func init() {
	commands["run"] = func(_ []string, stdin io.Reader, stdout, stderr io.Writer) int {
		files := []string{filepath.Join(".claude", "hookdistill.rules.json")}
		if home, err := os.UserHomeDir(); err == nil {
			files = append(files, filepath.Join(home, ".claude", "hookdistill.rules.json"))
		} else {
			fmt.Fprintf(stderr, "hookdistill: no home directory, using project rules only: %v\n", err)
		}
		return hook.Run(stdin, stdout, stderr, files)
	}
}
