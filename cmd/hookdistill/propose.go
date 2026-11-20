package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/abhaythambusiness-tech/hookdistill/internal/detect"
	"github.com/abhaythambusiness-tech/hookdistill/internal/replay"
	"github.com/abhaythambusiness-tech/hookdistill/internal/report"
	"github.com/abhaythambusiness-tech/hookdistill/internal/rules"
	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

const (
	defaultRules = ".claude/hookdistill.rules.json"
	replayFile   = "hookdistill.replay.json"
)

func init() {
	commands["propose"] = cmdPropose
	commands["replay"] = cmdReplay
	commands["promote"] = cmdPromote
}

func defaultRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// resolveClaudeMD returns the CLAUDE.md path to analyze: an explicit flag is
// used as given ("" skips it); the default is used only if the file exists.
func resolveClaudeMD(fs *flag.FlagSet, val string) string {
	explicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "claude-md" {
			explicit = true
		}
	})
	if !explicit {
		if _, err := os.Stat(val); err != nil {
			return ""
		}
	}
	return val
}

// splitID takes an optional leading positional id, since flag parsing stops
// at the first non-flag argument.
func splitID(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}

func cmdPropose(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("propose", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", defaultRoot(), "directory with Claude Code session transcripts")
	md := fs.String("claude-md", "./CLAUDE.md", "CLAUDE.md to analyze (skipped if the default is missing)")
	out := fs.String("out", defaultRules, "rules file to add candidates to")
	if fs.Parse(args) != nil {
		return 2
	}
	ss, err := transcript.LoadAll(*root)
	if err != nil {
		return fail(stderr, err)
	}
	cands := detect.Corrections(ss)
	if p := resolveClaudeMD(fs, *md); p != "" {
		c, _, err := detect.ClaudeMD(p, ss)
		if err != nil {
			return fail(stderr, err)
		}
		cands = append(cands, c...)
	}
	existing, err := rules.Load(*out)
	if err != nil {
		return fail(stderr, err)
	}
	seen := map[string]bool{}
	for _, r := range existing {
		seen[r.Tool+"|"+r.Pattern] = true
	}
	added := 0
	for _, c := range cands {
		key := c.Rule.Tool + "|" + c.Rule.Pattern
		if seen[key] || c.Rule.Validate() != nil {
			continue
		}
		seen[key] = true
		existing = append(existing, c.Rule)
		added++
		fmt.Fprintf(stdout, "  + %s  %s /%s/  (%s)\n", c.Rule.ID, c.Rule.Tool, report.Redact(c.Rule.Pattern), c.Rule.Origin)
	}
	if added > 0 {
		if err := rules.Save(*out, existing); err != nil {
			return fail(stderr, err)
		}
	}
	fmt.Fprintf(stdout, "added %d new rules as warn; %d already present; file: %s\n", added, len(existing)-added, *out)
	return 0
}

type replayState struct {
	Rule  string       `json:"rule"`
	Hits  []replay.Hit `json:"hits"`
	Legit []string     `json:"legit,omitempty"`
}

func findRule(rs []rules.Rule, id string) (int, error) {
	for i, r := range rs {
		if r.ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("unknown rule %q: run `hookdistill propose` first", id)
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func cmdReplay(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	id, args := splitID(args)
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", defaultRoot(), "directory with Claude Code session transcripts")
	rf := fs.String("rules", defaultRules, "rules file")
	legit := fs.String("legit", "", "comma-separated hit ids that were legitimate (e.g. a1b2c3d4:12)")
	if fs.Parse(args) != nil {
		return 2
	}
	if id == "" {
		fmt.Fprintln(stderr, "usage: hookdistill replay <id> [--root DIR] [--rules FILE] [--legit HIT,HIT]")
		return 2
	}
	rs, err := rules.Load(*rf)
	if err != nil {
		return fail(stderr, err)
	}
	i, err := findRule(rs, id)
	if err != nil {
		return fail(stderr, err)
	}
	ss, err := transcript.LoadAll(*root)
	if err != nil {
		return fail(stderr, err)
	}
	hits := replay.Replay(rs[i], ss)
	st := replayState{Rule: id, Hits: hits, Legit: splitList(*legit)}
	for _, h := range hits {
		mark := ""
		for _, l := range st.Legit {
			if l == h.ID {
				mark = "  [legitimate]"
			}
		}
		fmt.Fprintf(stdout, "  %s  %s: %s%s\n", h.ID, h.Tool, clip(report.Redact(h.Subject)), mark)
	}
	fmt.Fprintf(stdout, "%d hits for %s over %d sessions\n", len(hits), id, len(ss))
	b, err := json.MarshalIndent(st, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(filepath.Dir(*rf), replayFile), b, 0o644)
	}
	if err != nil {
		return fail(stderr, err)
	}
	return 0
}

func cmdPromote(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	id, args := splitID(args)
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	fs.SetOutput(stderr)
	rf := fs.String("rules", defaultRules, "rules file")
	legit := fs.String("legit", "", "comma-separated hit ids that were legitimate")
	if fs.Parse(args) != nil {
		return 2
	}
	if id == "" {
		fmt.Fprintln(stderr, "usage: hookdistill promote <id> [--rules FILE] [--legit HIT,HIT]")
		return 2
	}
	rs, err := rules.Load(*rf)
	if err != nil {
		return fail(stderr, err)
	}
	i, err := findRule(rs, id)
	if err != nil {
		return fail(stderr, err)
	}
	var st replayState
	b, err := os.ReadFile(filepath.Join(filepath.Dir(*rf), replayFile))
	if err != nil || json.Unmarshal(b, &st) != nil || st.Rule != id {
		return fail(stderr, fmt.Errorf("no replay recorded for %s: run replay first (`hookdistill replay %s`)", id, id))
	}
	marks := map[string]bool{}
	for _, l := range append(st.Legit, splitList(*legit)...) {
		marks[l] = true
	}
	promoted, err := replay.Promote(rs[i], st.Hits, marks)
	if err != nil {
		return fail(stderr, err)
	}
	rs[i] = promoted
	if err := rules.Save(*rf, rs); err != nil {
		return fail(stderr, err)
	}
	fmt.Fprintf(stdout, "%s is now block (%d past hits, none legitimate)\n", id, len(st.Hits))
	return 0
}

func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "hookdistill: %v\n", err)
	return 1
}

func clip(s string) string {
	if len(s) > 120 {
		return s[:120] + "..."
	}
	return s
}
