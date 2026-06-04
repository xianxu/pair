---
id: 000047
status: working
deps: []
github_issue:
created: 2026-06-03
updated: 2026-06-03
estimate_hours: 1.5
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

- [ ] Probe helper: locate the relevant binary (running → pidfile/pgrep, else PATH/~/.local/bin) and `strings`-grep for the adapt markers.
- [ ] Emit an `-- emitter health --` section in `doctor.sh` (warn on stale, affirm on healthy, graceful when undetectable).
- [ ] Regression test (fake binaries: marker-absent ⇒ flagged, marker-present ⇒ clean); wire into `make test`.
- [ ] Doc: note the probe in `doctor/README.md` + `doctor/SKILL.md`; cross-ref atlas "Binary freshness".

## Log

### 2026-06-03
- Filed as the natural follow-up to #000046 (pair-dev). Origin: the silent-emitter
  diagnosis earlier this session — stale pair-wrap/pair-slug predating #000045 made
  the doctor report a thin log with no way to name the cause. The boundary review
  on #46 also flagged this as "the natural next step on this thread."
