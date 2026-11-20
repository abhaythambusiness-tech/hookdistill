# Good first issues (drafted; publish them with the `good first issue` label)

Each one is small, has a pointer to the file, and says how to know it is done.

---

### 1. Recognize corrections in French

**Body:** hookdistill recognizes corrections ("no, don't do that", "I said never...") by phrases.
French is missing.

- Add French phrases to `correctionMarks` and harmless ones to `benign` in
  `internal/detect/corrections.go` (e.g. "arrête", "ne fais pas", "je te l'ai déjà dit" / "pas de souci").
- Add at least 3 positive and 3 negative examples to `TestIsCorrection` in
  `internal/detect/corrections_test.go`.
- Done when `go test ./internal/detect/` passes.

### 2. Recognize corrections in German

Same as above for German ("nein, das nicht", "hör auf", "ich habe dir gesagt" / "kein Problem").

### 3. Recognize corrections in Portuguese

Same as above for Portuguese ("não faça isso", "já te disse", "para com isso" / "sem problema").

### 4. Add an `Edit` rule example to the README

The README shows a `Bash` rule only. Add a short example of a rule for the `Edit` tool (for instance
protecting a `migrations/` folder) and show how `hookdistill replay` lists its hits.

- Use `examples/` data; do not use real sessions.
- Done when the example in the README runs as written.

### 5. Add a Windows test run for `install`

`internal/install` is tested on Linux. The CI matrix already builds on Windows; make sure
`TestInstall*` also checks the generated `command` string and the backup file on Windows paths.

- Look at `internal/install/install_test.go`; fix any assumption about `/` in paths.
- Done when `go test ./internal/install/` passes on the `windows-latest` job.

---

Extra, if one is too easy: a clearer error when `~/.claude/projects` does not exist
(`hookdistill scan` currently prints "no sessions found"; add a hint about `--root`).
