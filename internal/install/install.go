// Package install adds and removes hookdistill's PreToolUse entry in a Claude
// Code settings.json without disturbing anything else in the file.
package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const backupSuffix = ".hookdistill.bak"

// isOurs reports whether a hook command is "<path>/hookdistill run".
func isOurs(cmd string) bool {
	f := strings.Fields(cmd)
	return len(f) == 2 && filepath.Base(f[0]) == "hookdistill" && f[1] == "run"
}

func load(path string) (map[string]any, []byte, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, raw, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil || m == nil {
		return nil, nil, fmt.Errorf("install: %s is not a valid JSON object, leaving it untouched: %v", path, err)
	}
	return m, raw, nil
}

// each calls fn for every hook object under hooks.PreToolUse.
func each(m map[string]any, fn func(h map[string]any)) {
	hooks, _ := m["hooks"].(map[string]any)
	groups, _ := hooks["PreToolUse"].([]any)
	for _, g := range groups {
		gm, _ := g.(map[string]any)
		inner, _ := gm["hooks"].([]any)
		for _, h := range inner {
			if hm, ok := h.(map[string]any); ok {
				fn(hm)
			}
		}
	}
}

// Install adds the hookdistill PreToolUse entry (idempotent). The first time
// it changes an existing file it saves a backup next to it.
func Install(settingsPath, binPath string) error {
	m, raw, err := load(settingsPath)
	if err != nil {
		return err
	}
	want := binPath + " run"
	found, changed := false, false
	each(m, func(h map[string]any) {
		if c, _ := h["command"].(string); isOurs(c) {
			found = true
			if c != want {
				h["command"] = want
				changed = true
			}
		}
	})
	if found && !changed {
		return nil
	}
	if !found {
		hooks, _ := m["hooks"].(map[string]any)
		if hooks == nil {
			hooks = map[string]any{}
			m["hooks"] = hooks
		}
		groups, _ := hooks["PreToolUse"].([]any)
		hooks["PreToolUse"] = append(groups, map[string]any{
			"matcher": "*",
			"hooks":   []any{map[string]any{"type": "command", "command": want}},
		})
	}
	return write(settingsPath, m, raw)
}

// Uninstall removes only hookdistill's entry, dropping groups and keys that
// become empty because of it. A missing file is not an error.
func Uninstall(settingsPath string) error {
	m, raw, err := load(settingsPath)
	if err != nil {
		return err
	}
	if raw == nil {
		return nil
	}
	hooks, _ := m["hooks"].(map[string]any)
	groups, _ := hooks["PreToolUse"].([]any)
	removed := false
	var keepGroups []any
	for _, g := range groups {
		gm, ok := g.(map[string]any)
		if !ok {
			keepGroups = append(keepGroups, g)
			continue
		}
		inner, _ := gm["hooks"].([]any)
		var keep []any
		for _, h := range inner {
			if hm, ok := h.(map[string]any); ok {
				if c, _ := hm["command"].(string); isOurs(c) {
					removed = true
					continue
				}
			}
			keep = append(keep, h)
		}
		if len(keep) > 0 || len(inner) == 0 {
			gm["hooks"] = keep
			keepGroups = append(keepGroups, gm)
		}
	}
	if !removed {
		return nil
	}
	if len(keepGroups) > 0 {
		hooks["PreToolUse"] = keepGroups
	} else {
		delete(hooks, "PreToolUse")
	}
	if len(hooks) == 0 {
		delete(m, "hooks")
	}
	return write(settingsPath, m, raw)
}

func write(path string, m map[string]any, original []byte) error {
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	if len(original) > 0 {
		bak := path + backupSuffix
		if _, err := os.Stat(bak); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(bak, original, mode); err != nil {
				return fmt.Errorf("install: backup failed, settings untouched: %w", err)
			}
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".hookdistill-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
