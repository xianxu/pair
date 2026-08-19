---
id: 000134
status: codecomplete
deps: []
github_issue:
created: 2026-08-14
updated: 2026-08-19
started: 2026-08-14T17:25:00-07:00
estimate_hours: 4
actual_hours: N/A
---

# Muse harness support for pair

## Problem

Pair currently supports `claude`, `codex`, and `agy` as TUI harnesses (see `atlas/how-to-bring-up-a-new-harness-cli.md`). Meta's `muse` (`~/.local/bin/muse`, `Muse Code 0.1.0`) stores sessions at `~/.local/share/muse/sessions/YYYY/MM/DD/<uuid>/session.jsonl`, resumes via `muse resume <uuid>` / `--last`, and runs headless via `muse exec`. It needs the same 7 integration surfaces as the other agents to feel native inside the `zellij`+`nvim` workbench.

## Spec

Bring up `muse` as agent `muse` to parity with existing harnesses, following the 7 aspects in `atlas/how-to-bring-up-a-new-harness-cli.md`:

- **Aspect 1 – Return remap:** `Enter` → newline, `Alt+Enter` → send. Wire `sendKeymapByAgent["muse"]` in `cmd/internal/wrapcmd/wrap.go` (likely `plainCR=\n`, `altCR=\r` matching `codex`/`agy`; verify against live `muse` TUI).
- **Aspect 2 – Overlay suspension:** If `muse` shows pickers/modals, register `overlayDetectorByAgent["muse"]` so plain `Enter` bypasses remap while overlay is active. Emit `overlay-detect` `fired`/`near-miss`.
- **Aspect 3 – Session watcher + launcher recovery:** `SpecForAgent("muse",home)` watching `~/.local/share/muse/sessions` with `Match` extracting `<uuid>` from `*/<uuid>/session.jsonl` (recursive), `AgentSessionExists("muse",sid,cwd)`, `InferAgent` already generic, `StripResumeArgs` + resume flag mapping (`muse resume <sid>`), config `config-<tag>-muse.json`, ledger wiring. Spawned via `pair-wrap` → `pair session-watch muse ...` (same as `codex`/`agy`).
- **Aspect 4 – Slug generation:** `parseTranscript("muse",data)` for `muse` jsonl (extract user turns from `session.jsonl`), plus `model.Run` / `DefaultModel` support for `muse` (likely `muse exec` path); sandbox `Dir=os.TempDir()`.
- **Aspect 5 – PTY output filter:** Only if `muse` emits DEC sync-output/markers that break mouse scrollback (`stripCodexOutputMarkers` generalization). No-op if clean.
- **Aspect 6 – Settings whitelist:** Whitelist common commands in `muse` settings to reduce confirmation fatigue (static config, no flight signal).
- **Aspect 7 – Prompt glyph:** `PROMPT_PATTERN_BY_AGENT["muse"]` in `nvim/scrollback.lua` for `Alt+b` jump (capture live `muse` prompt prefix).

Also: `renameAgents` + any hardcoded agent switch in `cmd/internal/launcher` (`osruntime.go`, `createflow.go`, `agentargs.go`) must include `muse`. Verify `pair muse`, `pair muse -- --help`, `pair resume <tag>` inferred to `muse`, `pair list` shows `muse`, `Alt+b` jumps, `pair-slug` generates.

## Done when

- `pair muse` launches a workbench with draft `Enter` = newline / `Alt+Enter` = send, and an active picker (if any) confirms on plain `Enter` without leaking a newline.
- `pair session-watch muse` captures `session.jsonl`'s `<uuid>` into `config-<tag>-muse.json` + `ledger-<tag>.jsonl` via `lsof` + birth-time fallback; `AgentSessionExists("muse",sid,cwd)` returns true and `pair resume <tag>` re-attaches via `muse resume <sid>`.
- `pair slug` / `pair-slug` parses `muse` `session.jsonl` into turns; `model.Run` for `muse` produces a slug.
- `Alt+b` / `Alt+Shift+B` in `Alt+/` jumps between user prompts in a `muse` scrollback.
- Mouse scrollback and `pair scrollback render` show no glitches for `muse`.
- All harness drift signals for `muse` emit on the `adapt-<tag>.jsonl` flight recorder (`return-remap`, `overlay-detect`, `session-id`, `slug-parse`, `output-filter`, `prompt-search`) and `doctor` can diagnose drift.

## Plan

- [x] Probe live `muse` TUI: capture return-key bytes, overlay strings, prompt glyph, and `session.jsonl` line shapes for spec
- [x] Extend `cmd/internal/wrapcmd/wrap.go` (`sendKeymapByAgent`, `overlayDetectorByAgent`, span extraction if needed)
- [x] Extend `cmd/internal/sessionwatch` (`SpecForAgent` for `muse`, `Match` regex for `sessions/.../<uuid>/session.jsonl`) and `cmd/internal/launcher` (`MuseSessionPath`, `AgentSessionExists`, `StripResumeArgs`, rename/ledger, resume wiring)
- [x] Extend `cmd/internal/slugcmd/slug.go` (`parseMuse`) and `cmd/internal/model/model.go` (`DefaultModel` + `runMuse` via `muse exec`)
- [x] Add `nvim/scrollback.lua` `PROMPT_PATTERN_BY_AGENT.muse` and adapt emission
- [x] Build, run `go test ./...`, manual `pair muse` smoke + edge cases, update `atlas/how-to-bring-up-a-new-harness-cli.md` + `atlas/index.md` if needed
- [x] Fix Alt+n resume: exclude `…/subagent/…/session.jsonl` from `AgentSpec.Match`/`discover`/`discoverByBirth`/`transcript.Resolve`; lock `muse` resume parity in `launcher` with `TestResumeToken`/`TestCompose`/`TestMuseResumeArgs`/`TestStripResumeArgs` (ARCH-DRY/ARCH-PURE)

## Log

### 2026-08-14

- Investigated harness contract in `atlas/how-to-bring-up-a-new-harness-cli.md` and verified `muse` binary (`Muse Code 0.1.0`, `~/.local/bin/muse`), session layout (`~/.local/share/muse/sessions/YYYY/MM/DD/<uuid>/session.jsonl`), and CLI (`muse resume <uuid>`, `muse exec`, `muse export`, `muse trace`). Created issue 000134 after `sdlc issue new` was blocked by sandbox `.git` read-only lock; filed manually per vocabulary/issue.json scaffold.

### 2026-08-14 (implementation)

- Implemented Muse (Meta) harness across 7 aspects: `sendKeymapByAgent["muse"]` + `overlayDetectorByAgent["muse"]` with `musePickerMarkers` in `cmd/internal/wrapcmd/wrap.go`; `SpecForAgent("muse")` + `Match` for `*/session.jsonl` with UUID parent in `cmd/internal/sessionwatch/sessionwatch.go` plus `StripResumeArgs` handling `resume <sid>`; `MuseSessionsDir` + `AgentSessionExists("muse")` + `renameAgents+=muse` + `resumeToken`/`composeResumeArgs` for `muse resume` in `cmd/internal/launcher/*`; `transcript.Resolve("muse")` with glob `~/.local/share/muse/sessions/*/*/*/<sid>/session.jsonl`; `parseMuse` extracting `payload.event.prompt` from `runtime.session/run/started` in `cmd/internal/slugcmd/slug.go`; `DefaultModel` + `runMuse` via `muse exec` in `cmd/internal/model/model.go`; `PROMPT_PATTERN_BY_AGENT.muse="^>"` in `nvim/scrollback.lua`. Fixed `keymap_registry_test.go` to expect 4 agents. Verified `GOCACHE=/tmp/gocache go test ./cmd/internal/launcher ./cmd/internal/sessionwatch ./cmd/internal/slugcmd ./cmd/internal/transcript ./cmd/internal/wrapcmd -run TestSendKeymap` pass; full targeted suite `go test ./cmd/internal/... -run "TestAgentSpec|TestResolve|TestParse|TestSendKeymap|TestCompose|TestResume"` all PASS.
- Updated `atlas/how-to-bring-up-a-new-harness-cli.md` for `muse` parity (overlay detector map, session watcher example, recovery `resume <sid>`, slug `payload.event.prompt`, prompt `^>`, marker list, status line) and `atlas/index.md` intro to list Muse. Verified `go test ./...` — same 3 sandbox flakes as baseline (httptest bind, pty, wrapper pid); all harness suites PASS.
- Known environmental flakes unrelated to this change: `TestSIGUSR2ReExecsWrapperWithoutReplacingPaneProcess` fails on both this branch and base under sandbox (cannot write `.git/refs/stash.lock` / `GOCACHE` permission), and `cmd/internal/model` httptest server fails under sandbox network isolation — both reproduced on the untouched baseline.

### 2026-08-14 (Alt+n resume fix — plan 000134-muse-alt-n-resume-fix)

- **Fix:** Muse nests subagent sessions as `…/<root-uuid>/subagent/<sub-uuid>/session.jsonl`; the watcher was capturing the *subagent* id (newer birth, valid UUID parent) so `Alt+n` ran `muse resume <sub-uuid>` which is not resumable and fell back to fresh. Hardened `AgentSpec.Match("muse")` in `cmd/internal/sessionwatch/sessionwatch.go` to reject any path containing `/subagent/`, added `isMuseSubagentPath` helper in `cmd/internal/sessionwatch/run.go` to filter `discover` (lsof) and `discoverByBirth` (birth-time fallback) and `legacyExisting` scan, and hardened `transcript.Resolve("muse")` in `cmd/internal/transcript/transcript.go` to skip `Glob` matches containing `/subagent/` (ARCH-PURE/ARCH-DRY — single predicate for the resumability invariant).
- **Launcher:** verified `resumeToken("muse")="resume <id>"`, `composeResumeArgs("muse")` leads like `codex`, `extractExplicitResume("muse")` and `persistedConfigArgs` strip `resume <id>`; added `TestResumeTokenPerAgent`/`TestComposeResumeArgsOrdering` muse rows, `TestMuseResumeArgs` (strip/compose/extract), and `TestStripResumeArgsRemovesCanonicalResumeBindings/muse_leading_resume`; extended `cmd/internal/sessionwatch/sessionwatch_test.go` with `TestMuseMatchExtractsRootSessionID`/`TestMuseMatchIgnoresSubagentSession`/`TestMuseMatchReportsNearMissForBadID` and `cmd/internal/transcript/transcript_test.go` with `TestResolveMuseIgnoresSubagent`. Updated `atlas/how-to-bring-up-a-new-harness-cli.md` to note subagent exclusion.
- **Verification:** `GOCACHE=/tmp/gocache go test ./cmd/internal/sessionwatch -run TestMuse` PASS (3), `go test ./cmd/internal/transcript -run TestResolveMuseIgnoresSubagent` PASS, `go test ./cmd/internal/launcher -run "TestResume|TestCompose|TestMuse|TestExtract"` PASS, full `go test ./...` same 3 flakes as baseline. `go vet` clean. `Alt+n` smoke to be verified live: `pair muse` → prompt → `Alt+n` must resume same `session.jsonl` (check `config-<tag>-muse.json` + `ledger-<tag>.jsonl` hold root UUID, `adapt-<tag>.jsonl: session-id:fired`).

### 2026-08-19 — Close: superseded surface, real-corpus verification
- 2026-08-19: closed — Muse harness support is live on main across all seven aspects (harnessTTYProfiles["muse"] with a snapshot composer gate, SpecForAgent("muse") with subagent exclusion, launcher resume token/compose/strip, transcript.Resolve("muse"), parseMuse, runMuse, nvim prompt pattern); code landed in e4d1557. The Alt+n resume bug this issue fixed — the watcher capturing a subagent uuid so muse resume fell back to a fresh session — was verified against the real Muse session tree: 361 session.jsonl files, 314 under /subagent/; AgentSpec.Match matched 47/47 roots with the correct uuid and 0/314 subagents with zero near-misses, and transcript.Resolve resolved 47/47 root ids to their own file and 0 subagent ids. go test ./... and make test pass on main; all committed muse suites pass. The literal Alt+n keypress through a live zellij pane remains unexercised and is recorded as such in the plan Revisions.; review verdict: not-run

**The wrapcmd half of this issue has been superseded by #139.** The Log entries
above describe `sendKeymapByAgent["muse"]` and `overlayDetectorByAgent["muse"]`;
both registries were deleted when #139 replaced them with one
`harnessTTYProfiles` registry (`cmd/internal/wrapcmd/harness_tty.go`). Muse's
Return handling is now a pure snapshot recognizer (`museComposerActive`) behind
a positive composer gate, with literal bytes captured from the live CLI at
`cmd/internal/wrapcmd/testdata/tty/muse/0.1.0-R708.1/` and an every-split
conformance replay through the production proxy. Two Muse defects were found and
fixed during that rework: an unqualified `›` was accepted as a prompt, and the
recognizer only ever matched a one-line composer, so the second Enter submitted
mid-message. Muse's Return behavior is therefore better covered now than when
this issue wrote it — but the prose above is stale about *how* it works, and
`musePickerMarkers` is now the only thing defending Muse's selection menus
(tracked in `ttyFixtureDiscriminationGaps`, pair#139).

The other six aspects landed as described and are unchanged on main:
`SpecForAgent("muse")` + subagent exclusion, launcher resume token/compose/strip,
`transcript.Resolve("muse")`, `parseMuse`, `runMuse`, and the nvim prompt
pattern. All of it merged in `e4d1557`.

**Alt+n verification, the item this issue left open.** The Log's last line said
"`Alt+n` smoke to be verified live" and it was never done. Instead of a single
manual keypress, the invariant was verified against the real Muse session tree
on this machine — 361 `session.jsonl` files, 314 of them under `/subagent/`:

- `AgentSpec.Match("muse")`: 47/47 roots matched, each returning its own
  directory uuid; 0 of 314 subagent paths matched; 0 near-misses.
- `transcript.Resolve("muse", <id>)`: 47/47 root ids resolved to their own file;
  0 subagent ids resolved.

That is the bug this issue fixed — the watcher capturing a subagent uuid so
`muse resume` fell back to a fresh session — checked against every session Muse
has written here. The literal `Alt+n` keypress through a live zellij pane
remains unexercised; it is covered at the argument level by the launcher resume
tests. Recorded in the plan's `## Revisions` rather than left implicit.

**Close verification:** `go test ./...` and `make test` pass on main; the muse
suites (`TestMuseMatchExtractsRootSessionID`, `TestMuseMatchIgnoresSubagentSession`,
`TestMuseMatchReportsNearMissForBadID`, `TestResolveMuseIgnoresSubagent`,
`TestMuseResumeArgs`, `TestResumeTokenPerAgent`, `TestComposeResumeArgsOrdering`)
all pass. The boundary review is skipped with `--no-judge`: this issue's code
landed in `e4d1557` on 2026-08-15 and the review window from that boundary to
HEAD is now dominated by pair#139's 39 commits, which had their own three-round
boundary review — a review here would report on #139's surface, not this one's.

**`actual_hours: N/A`.** Measurement is unavailable for this window: the
implementation commits (`e4d1557` and its neighbours) never carried a `#134`
reference, so `sdlc actual` has no commit range to measure active time over.
Closing with `--no-actual` rather than hand-typing a number — a guessed value
would pollute velocity calibration, which is the exact failure the actual-hours
gate exists to prevent. The estimate of 4h stands unreconciled. Lesson recorded:
tag implementation commits with the issue number or the close loses its
measurement window permanently.
