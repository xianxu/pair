---
id: 000064
status: done
deps: []
github_issue:
created: 2026-06-16
updated: 2026-06-16
estimate_hours: 0.2
actual_hours: 0.00
---

# Park-nudge prompt: 5s timeout, auto-default N

## Problem

On Alt+x quit, `cleanup_quit_marker` (`bin/pair`) prompts:

```
pair: preserve "pair-N" scrollback to distill into a continuation later? [y/N]:
```

It blocks forever on `read` waiting for an answer. A quit shouldn't hang on an
unattended prompt — if the operator walks away (or a wrapper kills the pane
slowly), the cleanup stalls. The default is already N (preserve nothing), so an
unanswered prompt should just auto-pick N after a short timeout.

## Spec

1. Bound the `read` with a timeout (default **5s**) via `read -t`. On timeout
   the read exits non-zero, so the existing `&&` short-circuits to "no" — auto-N,
   no parking. (Verified: `read -t` integer timeouts work on macOS system bash
   3.2.57.)
2. The timeout is a seam: `PAIR_PARK_PROMPT_TIMEOUT` (default 5) — lets the
   operator tune/lengthen the nag and lets a test shorten it. Idiomatic with the
   existing `PAIR_*` env seams in `bin/pair`.
3. Advertise it in the prompt text: `… [y/N] (<n>s → N): ` so the operator knows
   it auto-dismisses.
4. On timeout/EOF (no Enter consumed) print a newline to `/dev/tty` so the
   un-terminated prompt line is closed before subsequent output.
5. Guard against bash 4+ leaving partial input in `$_ans` on timeout: only treat
   the answer as "yes" when the read genuinely SUCCEEDED (capture its status).

## Done when

- Unanswered park prompt auto-picks N after the timeout (no parking) instead of
  blocking forever.
- Answering `y`/`Y` within the window still preserves (unchanged).
- Prompt text shows the timeout; line is cleanly terminated on timeout.
- `PAIR_PARK_PROMPT_TIMEOUT` overrides the 5s default.
- `bash -n bin/pair` clean; behavior proven against the real system bash.

## Plan

- [x] `bin/pair` `cleanup_quit_marker`: add `read -t ${PAIR_PARK_PROMPT_TIMEOUT:-5}`, capture status, newline-on-timeout, prompt-text update; `bash -n` + construct-level proof on bash 3.2.

## Log

### 2026-06-16
- 2026-06-16: closed — Alt+x park prompt now auto-defaults to N after 5s (PAIR_PARK_PROMPT_TIMEOUT seam) instead of blocking forever; y/Y within window still preserves. Proven on system bash 3.2.57 — 4 cases: y→park, empty/Enter→N, no-input→timeout+newline+N, garbage-seam→fallback-to-5→N. bash -n clean; full make test green. Actual reads 0.00h (degenerate: commits bunched at end). Landing required restoring bin/ launcher scripts wrongly untracked by the 2026-06-16 propagate-base sweep (side-quest commit 0973062).; review verdict: SHIP

- Prompt lives at `bin/pair:1428`; default already N. Root cause: unbounded `read`.
- `read -t 1` confirmed working on local bash 3.2.57 (times out → non-zero → N).
- Adopted plan-quality reviewer's two refinements: the `if read -t …; then … else
  printf '\n' …; fi` shape (the `then` branch IS the read-success gate, so no
  separate `$?` var — partial input on timeout is never consulted); and a named,
  repeatable proof.
- **Verification.** Automated pty testing is NOT viable here: the prompt reads
  `</dev/tty` (intrinsic — at quit, stdin is the dying zellij pipe), and `openpty`
  is blocked in the sandboxed/CI test env (`script: openpty: Operation not
  permitted`), so a pty test would be *skipped* exactly where the suite runs —
  zero protection. Verified instead by (a) construct-fidelity proof on the real
  system bash 3.2.57 — `y`→PARK, Enter/empty→N, no-input→timeout+newline+N,
  garbage-seam(`abc`)→fallback-to-5→timeout+N (all four exact); (b) `bash -n
  bin/pair` clean; (c) full `make test` green (bash suites incl.
  `pair-continue-test` park_scrollback path + `go test ./...`); (d) live operator
  dogfood of the real Alt+x prompt.
- Atlas: noted the timeout/`PAIR_PARK_PROMPT_TIMEOUT` seam on the park-nudge line.
