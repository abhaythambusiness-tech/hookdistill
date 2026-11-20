// Package rules defines the on-disk rule format (one JSON object per line)
// and how a rule matches a tool call.
package rules

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

type Evidence struct {
	Session string `json:"session"`
	Seq     int    `json:"seq"`
}

type Rule struct {
	ID       string     `json:"id"`
	Event    string     `json:"event"`
	Tool     string     `json:"tool"`
	Pattern  string     `json:"pattern"`
	Action   string     `json:"action"`
	Message  string     `json:"message"`
	Origin   string     `json:"origin"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

// Load reads a rules file. A missing file is not an error: it returns nil.
func Load(path string) ([]Rule, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Rule
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for n := 1; sc.Scan(); n++ {
		b := bytes.TrimSpace(sc.Bytes())
		if len(b) == 0 {
			continue
		}
		var r Rule
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, fmt.Errorf("%s: line %d: %w", path, n, err)
		}
		out = append(out, r)
	}
	return out, sc.Err()
}

// Save writes rules as JSON lines, atomically (temp file in the same dir + rename).
func Save(path string, rs []Rule) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".hookdistill-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	enc := json.NewEncoder(tmp)
	for _, r := range rs {
		if err := enc.Encode(r); err != nil {
			tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (r Rule) Validate() error {
	switch {
	case r.ID == "":
		return errors.New("rule has no id")
	case r.Event != "PreToolUse":
		return fmt.Errorf("rule %s: event must be PreToolUse, got %q", r.ID, r.Event)
	case r.Action != "warn" && r.Action != "block":
		return fmt.Errorf("rule %s: action must be warn or block, got %q", r.ID, r.Action)
	case r.Pattern == "":
		return fmt.Errorf("rule %s: pattern is empty", r.ID)
	}
	if _, err := regexp.Compile(r.Pattern); err != nil {
		return fmt.Errorf("rule %s: invalid pattern: %w", r.ID, err)
	}
	return nil
}

var reCache sync.Map // pattern -> *regexp.Regexp (nil when invalid)

func compiled(p string) *regexp.Regexp {
	if v, ok := reCache.Load(p); ok {
		return v.(*regexp.Regexp)
	}
	re, _ := regexp.Compile(p) // RE2: no catastrophic backtracking
	reCache.Store(p, re)
	return re
}

// Matches reports whether the rule applies to this tool call. An empty or "*"
// Tool means any tool. An empty or invalid pattern never matches.
func (r Rule) Matches(tool, subject string) bool {
	if r.Tool != "" && r.Tool != "*" && r.Tool != tool {
		return false
	}
	if r.Pattern == "" {
		return false
	}
	re := compiled(r.Pattern)
	return re != nil && re.MatchString(subject)
}

// NewID returns a stable rule id ("c-" + 8 hex chars) derived from the tool
// and pattern, so the same candidate keeps the same id across scans.
func NewID(tool, pattern string) string {
	sum := sha1.Sum([]byte(tool + "\x00" + pattern))
	return "c-" + hex.EncodeToString(sum[:])[:8]
}
