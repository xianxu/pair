---
id: 000098
status: wontfix
deps: []
github_issue:
created: 2026-07-01
updated: 2026-07-05
estimate_hours:
started: 2026-07-05T21:45:50-07:00
---

# context-window count on agent pane stuck (e.g. at 151k)

## Problem

The context-window size shown in the agent-pane frame title — the `(<count>)` in
`<agent> (<count>) [<cwd>]` (#71) — appears **stuck**. Reported: it used to track
down and, when it dropped to low 100k, compaction happened; now it seems frozen
at ~151k and no longer moves.

## Spec

Not yet root-caused — this issue captures the report + the leads for whoever picks
it up.

### Where the number comes from

- `pair-title.sh` → `update_frame_titles()` sets the count via
  `count=$(pair-context "$TAG" "$agent")`
  (`cmd/internal/runtimebundle/assets/runtime/files/bin/pair-title.sh:105`).
- The count itself is computed by `pair-context`
  (`cmd/pair-context/main.go`, `cmd/internal/contextcmd/contextcmd.go`) — it reads
  the agent's live transcript and derives the current context-window size.

### Leads to check

- **Pinned-file assumption (most likely).** `pair-title.sh:178-184` documents that
  claude's `/clear` + compaction keep writing the **same pinned** `…/<sid>.jsonl`
  in place (verified for #71), so the cached path stays valid. If that assumption
  is now violated — e.g. a newer claude version rotates the transcript on
  compaction, or writes a new session file — `pair-context` would keep reading the
  **old** file and report a frozen count. "Stuck right around a compaction boundary
  (151k)" fits a stale-file-handle / stale-path read.
- **Caching:** confirm neither `pair-context` nor the frame updater caches the
  count across compaction (the `agent_file_cache` in pair-title.sh is only for the
  mtime check, but verify `pair-context` re-reads fresh each call).
- **Counting method:** confirm the size is derived from the *current* transcript
  tail, not a monotonic max that can't decrease.

### Reproduce / verify

- Watch `pair-context <tag> claude` directly across a compaction and compare to
  the agent's own reported context size; confirm whether the stuck value tracks a
  stale transcript path.

### Root-cause analysis (2026-07-05) — prime suspect REFUTED, mechanism narrowed

Investigated end-to-end (code trace + claude-docs + 2,303-transcript scan + live
pin check). Findings:

- **Code is correct for the suspected cases.** The count resolves
  `config-<tag>-claude.json` `session_id` → `~/.claude/projects/<enc-cwd>/<sid>.jsonl`
  (`transcript.go:19-31,45-47`, ported to Go in #93 — `pair-title.sh`/`pair-context`
  no longer exist as such), read **fresh every poll** (no cache), counting the
  **last** assistant `usage` (`ctxmeter.go:57-73`, last-wins not a monotonic max).
  Caching and counting are **not** the bug.
- **The pinned-file assumption STILL HOLDS — rotation refuted.** Authoritative
  Claude Code docs: `/clear`, auto-compaction, and `--resume`/`--continue` all keep
  the **same session-id and file**; a new session-id only appears on `/branch`,
  `--fork-session`, or a fresh session (no `--resume`). Confirmed empirically: a
  scan of 2,303 real transcripts found in-place compaction (`isCompactSummary`,
  same `sessionId`, e.g. 997k→46k **in one file** — the meter tracks it), **zero**
  `summary`-type rotation records, and **zero** filename≠internal-sessionId
  mismatches. Live check of all active tags (ariadne/brain/pair): every
  active-agent pin points at the **actively-written** transcript. So the reporter's
  attributed trigger — *"compaction happened, now frozen"* — **cannot** be the
  cause; compaction keeps the count moving.
- **The freeze mechanism (confirmed) = stale pin.** The count freezes iff
  `config`'s `session_id` stops matching claude's live file. Every way that happens
  is an **edge case**, none triggered by compaction:
  1. user passes `--fork-session` → claude forks to a new uuid, pair's
     `shouldMintClaudeSessionID` declines to mint and never captures it
     (`agentargs.go:115-118`; gap at `createflow.go:254-257`); persists across
     restarts (not stripped by `persistedConfigArgs`). **Strongest pair-side gap.**
  2. user passes their own `--session-id <uuid>` → pair never records it.
  3. mint exhaustion (`MintUUID` "" / 5× collision) → claude launched bare, invents
     its own uuid, config not written. Rare.
  4. claude `/branch` → new file, pair has no hook to observe it.
  5. *Would* be common **iff** some claude version forks-on-`--resume` (Alt+n then
     staleifies config every restart) — but docs + the mismatch-free scan refute this.
- **All normal pair relaunch paths keep config in sync** (fresh→mint+write;
  Alt+n→`--resume savedSid`+write-same-sid; Shift+Alt+N & #55-compaction→drop+mint+write;
  `pair rename`→moves file preserving sid). No race window in pair's own relaunches.

**Bottom line:** the bug as filed (compaction/rotation → stale path) is **not
reproducible** with current claude + pair. The real fragility is that the resolver
**blindly trusts the pin with no staleness detection or recovery** — so the edge
cases above (and any future claude fork-on-resume regression) fail **silently** as
a frozen number. Direction pending user steer (see Log 2026-07-05).

### ACTUAL root cause (2026-07-05) — this is #97, not a resolver bug

The reporter's live example (`pair-ariadne`) showed **both** the wrong agent
(codex, should be claude) **and** the stuck 151k — on the tag carrying the stale
`pane-ariadne-codex.json` twin. That is not a coincidence: **the frozen count is a
symptom of #97.** Confirmed on the real files:

- `agent-ariadne` = `claude` (active). `pair context ariadne claude` = **761k** (live).
- `pair context ariadne codex` = **151k** — the *exact* frozen value reported.
- The old poller globs both `pane-ariadne-{claude,codex}.json` (both `pane_id 0`),
  and codex (alpha-last) wins → the frame renders **codex's name AND codex's
  count**. Codex isn't running, so its transcript is frozen at its last value
  (151k) forever → "stuck count."

So the meter/resolver were correct all along; the frame was simply showing the
**wrong agent's** (dead) transcript. The #97 active-agent filter fixes both: with
`pane.Agent == opts.Agent`, ariadne renders `claude (762k)` and the codex twin is
ignored — **verified against ariadne's real pane files** with the merged #97 code.

**Resolution: #98 is resolved by #97** (merged, PR #68). No separate resolver
change is warranted — the "rotation" fix would have addressed a non-problem, and
"read newest file" would have been actively wrong (subagent/judge transcripts
pollute the same project dir). The theoretical stale-pin edge cases
(`--fork-session`, user `--session-id`, `/branch`, mint-exhaustion) remain, but are
rare, user-driven, and not what was reported — a separate low-priority defensive
guard at most, not this issue.

## Done when

- [x] Root cause identified — it is **#97** (stale pane-twin → frame renders the
      wrong/dead agent's frozen count), NOT a transcript-path/caching/counting bug.
- [x] `pair-context` reports a moving count across `/clear` + compaction — verified:
      it already does (293k live == claude's `/context`; 997k→46k across a real
      in-place compaction). The freeze was the frame showing *codex's* dead count.
- [~] Regression coverage — carried by #97's `TestUpdateFrameTitlesIgnoresStaleAgentTwin`
      (the active-agent filter is what stops the wrong-agent count). No separate
      #98 test warranted (no separate #98 code).

## Plan

- [ ] Reproduce: `pair-context <tag> claude` across a compaction; check the file
      it reads vs. the live transcript.
- [ ] Fix the stale-path / caching / counting bug in `pair-context`
      (+ `contextcmd`), and the pinned-file assumption in `pair-title.sh` if claude
      now rotates.
- [ ] Add a test with a rotated/compacted transcript fixture.

## Log

### 2026-07-01

- Moved from `ariadne#150` (filed in ariadne's inbox; the code lives here in
  pair: `pair-context` + `pair-title.sh`). Reported symptom: count stuck ~151k,
  no longer dropping/tracking compaction. Not yet root-caused — leads captured
  above (pinned-transcript-file assumption from #71 is the prime suspect).

### 2026-07-05

- Claimed + diagnosed (3 subagent traces + direct dogfood in a live claude pair
  session). See Spec ▸ Root-cause analysis. Net: **prime suspect (rotation on
  compaction) refuted** by claude docs + a 2,303-transcript scan + live-pin check;
  the resolver/counting/caching are correct for `/clear`+compaction. The freeze is
  a **stale-pin** failure, reachable only via edge cases (`--fork-session`, user
  `--session-id`, mint-exhaustion, claude `/branch`) — none triggered by compaction.
- Could not reproduce locally: this session is opus-4-8 with a **1M-token window**,
  so compaction never fires (219k of 1M); all my transcripts stay single-file.
- **Blocked on user steer** — the fix direction depends on whether the freeze is
  still reproducible and its real trigger (a `/branch`? a restart on an older
  claude? still happening?), vs. building a generic staleness-detection defense,
  vs. closing as refuted/not-reproducible.
- **User supplied the smoking gun:** live `pair-ariadne` shows BOTH wrong agent
  (codex, should be claude) AND stuck 151k. Confirmed `pair context ariadne codex`
  = 151k (the reported value), `pair context ariadne claude` = 761k (live). So the
  freeze = **#97** (stale codex twin wins the glob → frame shows codex's name AND
  its dead transcript's count). The merged #97 active-agent filter renders
  `claude (762k)` — verified against ariadne's real pane files. See Spec ▸ ACTUAL
  root cause.
- **Closed wontfix — resolved by #97** (PR #68, merged). No separate code: the
  "rotation" fix would have been a non-problem and "newest-file" resolution would
  have been wrong (judge/subagent transcripts share the project dir). Live
  displays reflect the fix once each poller is rebuilt+respawned, or immediately
  by removing the stale `pane-<tag>-codex.json` twins (#97(b) also auto-cleans them
  on each tag's next clean quit). Edge-case stale-pin guard (`--fork-session` /
  `/branch`) left as possible separate low-priority work, not this issue.
