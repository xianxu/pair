---
id: 000047
status: done
deps: []
github_issue:
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 1.5
actual_hours: 2
---

# pair-doctor stale-emitter-binary probe

## Problem

pair-doctor reads `adapt-<tag>.jsonl` and reports what's in it — but it's **blind
to why the log is thin**. The motivating incident (the thread that produced #46):
the flight recorder went silent for every Go-emitted aspect (1/2/4/5) while only
nvim's Lua emitter (aspect 7) logged. Root cause was not drift but a **stale
`pair-wrap`/`pair-slug` binary** — the installed copy predated #000045, so it had
no adapt-logging code at all. The doctor faithfully showed "only prompt-search
fired" and could not say *the emitters themselves are stale*. A human burned real
time concluding the telemetry was broken.

This is ironic: the doctor's whole job is making silent drift observable, yet it's
blind to the silent staleness of its own emitters — the exact failure class #46
(`pair-dev`) addresses at launch time. The doctor should be able to *diagnose* the
condition `pair-dev` *prevents*.

## Spec

Add an **emitter-health probe** to `doctor/doctor.sh` that detects when a Go
emitter binary lacks adapt-logging code (or is otherwise stale) and prints a loud
`STALE-BINARY` finding with the remedy — turning "log is mysteriously thin" into a
diagnosable line.

- **Primary signal — semantic, not mtime:** a binary that contains adapt logging
  has the signal strings compiled in (`return-remap`, `overlay-detect`,
  `output-filter` for pair-wrap; `slug-parse` for pair-slug). Probe via
  `strings <bin> | grep -q return-remap`. Absent ⇒ stale (predates #000045).
  mtime-vs-source is a fragile secondary at best (clones/`touch`); prefer the
  strings check.
- **Which binary:** prefer the *actually running* `pair-wrap` for the session
  (resolve via `$PAIR_DATA_DIR/agent-pid-<tag>` → its exe, or `pgrep -f
  pair-wrap`); fall back to the PATH-resolved / `~/.local/bin` binary when no
  process is live (e.g. doctor run against a saved log). Check pair-wrap and
  pair-slug.
- **Output:** an `-- emitter health --` section (before or beside the tallies),
  e.g.
  `⚠ STALE-BINARY: pair-wrap (<path>) has no adapt logging — aspects 1/2/5 can't
  log. Fix: make install (or launch via pair-dev). See atlas "Binary freshness".`
  Healthy ⇒ a one-line "emitters: ok" so the absence of a warning is meaningful.
- **Stay diagnostic:** always exit 0; degrade gracefully if `strings` is missing
  or the binary can't be located (note it, don't fail).
- **Test:** `tests/`/`doctor_test.sh` case with a fake binary lacking the marker
  string ⇒ STALE-BINARY surfaced; a fake containing it ⇒ no warning.

Out of scope: auto-rebuilding (that's #46's `pair-dev`); checking nvim/shell
emitters (interpreted, always fresh).

## Done when

- `doctor.sh` flags a `pair-wrap`/`pair-slug` binary that lacks the adapt signal
  strings as `STALE-BINARY`, with the rebuild remedy.
- Healthy emitters print an affirmative one-liner (silence-is-meaningful).
- Picks the running binary when a session is live, else the installed/PATH one.
- Degrades gracefully (exit 0) when `strings`/the binary is unavailable.
- A regression test covers stale-detected and fresh-not-flagged.

## Plan

- [x] Probe helper (`doctor/emitter-health.sh`, sourced + unit-testable): `_resolve_emitter` (running via `pair-wrap-pid-<tag>` pidfile + portable `_pid_exe`, else PATH), `_binary_has_marker` (`strings`-grep), `emitter_health_report`.
- [x] Emit an `-- emitter health --` section in `doctor.sh` (create + NO-DATA paths; `[ok]`/`[STALE]`/`[?]`; always exit 0).
- [x] Regression test (`tests/emitter-health-test.sh`): marker present/absent/can't-tell, **selection** (PATH vs pidfile-override), report formatting, + SIGPIPE guard; wired into `make test`. Plus a deterministic header assertion in `doctor_test.sh`.
- [x] Doc: `doctor/README.md` + `doctor/SKILL.md` describe the probe; atlas "Binary freshness" cross-refs it.

## Revisions

### 2026-06-03 — plan-quality sharpenings (sdlc change-code judge: INFO)
- **Selection is now tested** (refinement 1): logic extracted to a sourced lib;
  `_pid_exe` is overridable so the pidfile-vs-PATH choice is asserted without a
  live process / `ps` (sandbox blocks `ps`).
- **Portable PID→exe** (refinement 2): `/proc` (Linux) → `lsof` txt (macOS, full
  path) → `ps comm` last resort. The running-binary preference works on darwin.
- **DRY breadcrumb** (refinement 3): a comment in `emitter-health.sh` points at
  the Go emitter call sites so a signal-string rename doesn't silently false-flag.

### 2026-06-03 — spec corrections (from boundary review, verdict SHIP)
- **Pidfile corrected:** Spec §"Which binary" said `agent-pid-<tag>`, but that
  file holds the *child agent's* PID (`cmd/pair-wrap/main.go:1551`). The probe
  uses `pair-wrap-pid-<tag>` (`main.go:1564`, written with pair-wrap's own
  `os.Getpid()`) — following the spec literally would have probed the wrong
  binary. Implementation is correct; the Spec text is the stale part.
- **Wording:** the Spec/Done-when say `STALE-BINARY`; the shipped token is
  `[STALE]` (docs + tests consistent on `[STALE]`). Same loud named finding.

## Log

### 2026-06-03
- 2026-06-03: closed — emitter-health 9/9 + doctor 5/5 + dev-rebuild 3/3 green; live run reports [ok] for fresh running pair-wrap (via pidfile 71687) + pair-slug; selection (pidfile vs PATH) + SIGPIPE-under-pipefail + lsof-txt-resolution all covered; renders on create + NO-DATA paths, exits 0; review verdict: SHIP
- Filed as the natural follow-up to #000046 (pair-dev). Origin: the silent-emitter
  diagnosis earlier this session — stale pair-wrap/pair-slug predating #000045 made
  the doctor report a thin log with no way to name the cause. The boundary review
  on #46 also flagged this as "the natural next step on this thread."
- Two real bugs caught by running the probe against this box's live binaries (not
  by the unit fixtures): (1) `lsof -Fn` grabbed the process **cwd**, not the `txt`
  (executable) record — fixed by filtering `$4=="txt"`; (2) `strings | grep -q`
  under `set -o pipefail` SIGPIPEs `strings` on a match → false `[STALE]` on a
  large binary — fixed with `grep -c` (consumes all input). Added a pipefail +
  large-fixture test so the SIGPIPE class is guarded (the tiny fixtures couldn't
  trigger it).
- Verified live: with the running pair-wrap (pidfile 71687 → `bin/pair-wrap`) and
  `bin/pair-slug` both fresh, the probe reports `[ok] [ok]`; it correctly used the
  running binary via the pidfile. Shell tests 9/9 (emitter-health) + doctor 5/5 +
  dev-rebuild 3/3 green.
