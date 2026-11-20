# Show HN draft (do not post before v0.1.0 is tagged and CI is green)

**Title** (80 characters max):

Show HN: Hookdistill – turn repeated Claude Code corrections into tested hooks

**URL:** https://github.com/abhaythambusiness-tech/hookdistill

**First comment** (post it yourself right after submitting):

I kept telling Claude Code the same things ("don't commit with -am", "don't edit the migrations")
and it kept forgetting. A line in CLAUDE.md is a request; a hook is a rule it cannot skip. But
writing hooks by hand is tedious, and I never knew which rules were worth enforcing.

hookdistill reads the session transcripts already on your disk (~/.claude/projects) and your
CLAUDE.md, and:

- finds actions you corrected at least 3 times in at least 2 sessions,
- maps the prohibitions in CLAUDE.md that name a command,
- replays each candidate rule against your past sessions so you see every call it would have matched,
- only lets a rule become "block" if you did not mark any of those matches as legitimate,
- and acts as the hook itself (`hookdistill run`): it asks or blocks, and fails open on any error.

It is one Go binary with no dependencies. It runs offline and makes no model calls, so it spends no
tokens. Reports redact common secret formats, but redaction is best-effort.

Honest limits: the correction detector is phrase-based (English and Spanish so far), the transcript
format is undocumented and may change, and hooks are not guaranteed to run in every subagent, MCP or
non-interactive case. On my own 542 sessions only one repeated correction reached the threshold; the
other four useful candidates came from CLAUDE.md. So on real data expect a short list. A "match" in
the report is a past call that fits a pattern, not proof it was a mistake: that is what replay is for.

Similar tools: claudecode-rule2hook converts rule text to hooks, PRISM analyzes sessions and suggests
CLAUDE.md edits, hookify (official plugin) works from the current conversation. What I tried to add is
mining the whole history and testing a rule before it can block anything.

I would like to hear: which correction phrases in your language does it miss, and what rules would you
want it to find?

---

Notes for the author (delete before posting):

- Numbers above are the real ones from the real-data run in
  docs/superpowers/plans/2026-11-20-real-data-run.md. Re-run `hookdistill scan` before posting and
  update them if they changed.
- No claims about stars or token savings: neither was measured.
- Never paste raw `scan` output from your own history into the thread.
