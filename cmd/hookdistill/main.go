// hookdistill turns the corrections you repeat to Claude Code into hooks,
// tested against your own session history.
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// version is overridden at release time with -ldflags "-X main.version=...".
var version = "dev"

type cmdFunc func(args []string, stdin io.Reader, stdout, stderr io.Writer) int

// commands is filled by init functions in the files that implement each subcommand.
var commands = map[string]cmdFunc{
	"version": func(_ []string, _ io.Reader, stdout, _ io.Writer) int {
		fmt.Fprintf(stdout, "hookdistill %s\n", version)
		return 0
	},
}

func usage(w io.Writer) {
	names := make([]string, 0, len(commands))
	for n := range commands {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Fprintf(w, "usage: hookdistill <command> [flags]\ncommands: %s\n", strings.Join(names, ", "))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	c, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "hookdistill: unknown command %q\n", args[0])
		usage(stderr)
		return 2
	}
	return c(args[1:], stdin, stdout, stderr)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
