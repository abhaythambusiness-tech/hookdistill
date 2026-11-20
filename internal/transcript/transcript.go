// Package transcript reads Claude Code session files (~/.claude/projects/*/*.jsonl).
// The format is not a documented contract, so the reader is deliberately lax:
// unknown line types are ignored and malformed lines are counted, never fatal.
package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ToolCall struct {
	Session   string
	Seq       int
	Time      time.Time
	Tool      string
	Input     map[string]any
	Sidechain bool
	Project   string
}

type UserMsg struct {
	Session string
	Seq     int
	Time    time.Time
	Text    string
	Project string
}

type Session struct {
	ID, Project string
	Calls       []ToolCall
	Msgs        []UserMsg
	Skipped     int
}

// Subject is the string a rule pattern is matched against: the command for
// Bash, the file path for file tools, empty for anything else.
func (c ToolCall) Subject() string {
	key := ""
	switch c.Tool {
	case "Bash":
		key = "command"
	case "Read", "Edit", "MultiEdit", "Write":
		key = "file_path"
	default:
		return ""
	}
	s, _ := c.Input[key].(string)
	return s
}

type line struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	IsSidechain bool   `json:"isSidechain"`
	Message     *struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type block struct {
	Type  string         `json:"type"`
	Text  string         `json:"text"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// LoadFile parses one session file.
func LoadFile(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()

	s := Session{
		ID:      strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Project: filepath.Base(filepath.Dir(path)),
	}
	r := bufio.NewReaderSize(f, 1<<20)
	for seq := 1; ; seq++ {
		raw, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(raw)) > 0 {
			s.add(seq, raw)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return s, nil
			}
			return s, err
		}
	}
}

func (s *Session) add(seq int, raw []byte) {
	var l line
	if json.Unmarshal(raw, &l) != nil {
		s.Skipped++
		return
	}
	if l.Type != "user" && l.Type != "assistant" {
		return // attachments, api-request, system, ... are expected noise
	}
	if l.Message == nil {
		s.Skipped++
		return
	}
	ts, _ := time.Parse(time.RFC3339, l.Timestamp)

	var str string
	if json.Unmarshal(l.Message.Content, &str) == nil {
		if l.Type == "user" && str != "" && !strings.HasPrefix(str, "<") {
			s.Msgs = append(s.Msgs, UserMsg{s.ID, seq, ts, str, s.Project})
		}
		return
	}
	var blocks []block
	if json.Unmarshal(l.Message.Content, &blocks) != nil {
		s.Skipped++
		return
	}
	var texts []string
	for _, b := range blocks {
		switch {
		case l.Type == "assistant" && b.Type == "tool_use":
			s.Calls = append(s.Calls, ToolCall{s.ID, seq, ts, b.Name, b.Input, l.IsSidechain, s.Project})
		case l.Type == "user" && b.Type == "text" && b.Text != "" && !strings.HasPrefix(b.Text, "<"):
			texts = append(texts, b.Text)
		}
	}
	if len(texts) > 0 {
		s.Msgs = append(s.Msgs, UserMsg{s.ID, seq, ts, strings.Join(texts, "\n"), s.Project})
	}
}

// LoadAll reads root/*/*.jsonl. A missing root is not an error: it returns nil.
func LoadAll(root string) ([]Session, error) {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	paths, err := filepath.Glob(filepath.Join(root, "*", "*.jsonl"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []Session
	for _, p := range paths {
		s, err := LoadFile(p)
		if err != nil {
			return out, err
		}
		out = append(out, s)
	}
	return out, nil
}
