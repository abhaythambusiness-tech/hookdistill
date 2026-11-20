# Real-data run (Task 11, step 4) — counts only, no content

Run on the author's own `~/.claude/projects` with `hookdistill scan` (read-only).

| Measure | Value |
|---|---|
| Sessions read / tool calls / user messages | 542 / 30107 / 3671 (0 unreadable lines) |
| Messages that read as corrections | 85 (2.3%) |
| ...attributed to a preceding tool call | 63, spread over 55 distinct action groups |
| Correction candidates (>=3 times, >=2 sessions) | 1 |
| CLAUDE.md lines: convertible / not convertible | 5 / 51 |

Design criterion ">=5 candidates with evidence": met (6), but 5 of the 6 come from CLAUDE.md
patterns, and a "match" is a past call that fits the pattern, not a confirmed violation
(`replay` is how a person confirms that).

Limits found:
- Corrections are phrased in many ways and about many different actions, so the action
  signature (command words) rarely reaches 3 occurrences. A coarser fallback grouping is
  possible but raises false positives; left for after v0.1 (deferred).
- The correction marks were not tuned against the 85 real messages: doing so means reading
  the author's private messages, so it is left to the author (see ledger).
