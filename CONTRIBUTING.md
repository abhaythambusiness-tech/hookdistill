# Contributing

Thanks for helping. This is a small, dependency-free Go project, so setup is short.

## Build and test

```bash
go build ./cmd/hookdistill
go vet ./...
go test ./...
```

Go 1.22 or newer. The standard library only: please do not add dependencies without opening an
issue first.

## Add a correction phrase in another language

Corrections are recognized in `internal/detect/corrections.go`:

1. Add your phrases to `correctionMarks` (things people say when Claude did the wrong thing).
2. Add harmless phrases that contain them to `benign` (for example "no worries").
3. Add examples to `TestIsCorrection` in `internal/detect/corrections_test.go`:
   a few that must be corrections, a few that must not. A phrase with no "must not" example
   will not be accepted.
4. Run `go test ./internal/detect/`.

Keep phrases specific. A mark that fires on normal chat creates false positives, and a false
positive means a rule that annoys people.

## Tests come first

Write the failing test, watch it fail, then write the code. Bug reports with a failing test are
the best kind.

## Never commit real sessions

Session transcripts contain code, paths and sometimes secrets. Test data must be synthetic;
see `examples/` for the style. The CI runs gitleaks on the full history.

## Commit style

Short, imperative, with a type prefix: `feat:`, `fix:`, `docs:`, `test:`, `ci:`, `refactor:`.
One logical change per commit.

## Reporting a problem

Use the issue templates. Run `hookdistill scan` and read the output before pasting it: it redacts
common secrets, but you are the last check.
