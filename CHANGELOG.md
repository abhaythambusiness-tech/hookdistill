# Changelog

## 0.1.0 - 2026-11-20

First release.

- `scan`: reads local Claude Code sessions and `CLAUDE.md`; reports repeated corrections,
  convertible and non-convertible `CLAUDE.md` rules, measured counts and estimated token cost.
  Secrets are redacted in the report.
- `propose`: writes candidate rules (always as `warn`) without duplicating or touching existing ones.
- `replay`: lists the past tool calls a rule would have matched.
- `promote`: turns a rule into `block`, refused if any replayed hit was marked legitimate.
- `install` / `uninstall`: add or remove one `PreToolUse` entry in `settings.json`, with a backup.
- `run`: the hook. Fails open on any error.
- Single static binary for Linux, macOS and Windows; checksum-verified install script.
