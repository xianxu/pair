---
id: 000349
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: 'a944c75689c131a2659b113579eba84c29319004' # card fields mirrored from issue-cards; edit via sdlc
---

# Preserve TTY captures on startup and use them for naming

## Problem

Same-tag startup truncates raw TTY captures, quit asks whether to preserve them, and automatic naming still parses native transcripts. This is the unfinished TTY scope explicitly moved out of #346 after binding smoke tests passed.

## Spec

### TTY history and feature ownership

Printable TTY is the primary human-attention-level conversation history. Omission of hidden or collapsed tool payloads is intentional and useful, not a reason to require native transcripts for ordinary text features. Keep Pair prompt history separately for exact input/editor comments. Native parsing remains narrow: identity correlation and optional usage/turn-completion telemetry. Native file mtime can inform activity without parsing contents; those optional features must degrade locally.

| Feature | Current source | Agreed direction |
| --- | --- | --- |
| Scrollback/search/annotations | Rendered raw TTY plus timing/resize sidecar | Retain TTY source |
| Alt+l changelog | Unlimited printable TTY with timestamps | Retain TTY source |
| Agent-switch orientation | TTY first; Pair prompts and native paths supplementary | TTY remains primary; native parsing is not an admission gate |
| Automatic thread naming | Parsed native user/assistant messages | Move conversational text input to printable TTY |
| Prompt history/navigation/resend | Pair editor prompt history | Retain exact input source |
| Context/token meter | Native usage records | Optional structured telemetry |
| Notifications | Live terminal signals plus Codex lifecycle records | Optional structured telemetry; printable text is not an exact turn-end signal |
| Idle fading/title heat | Native file mtime, Pair prompt mtime, launch time | No body parsing required; avoid treating redraws as work |
| Binding discovery/confirmation | Pair sends correlated with native messages/progress | Narrow current-launch identity bridge or explicit UUID handshake |
| Continuation | Explicit checkpoint with preserved TTY support | Keep checkpoint contract |
| Resume/review scoping/changelog naming | Session identifier | Identity use, not transcript-content parsing |

Leave raw TTY capture and timing/resize sidecar intact on quit; remove the preservation question and discard path. Before startup (including wrapper restart) reuses a same-tag pathname, copy the prior family to a unique archive and finish/sync it before truncation. Preservation failure leaves originals untouched and refuses destructive reuse. Protect against a competing live writer, but tolerate redundant archives after an interrupted startup: no exactly-once archive transaction is required. Keep current compaction behavior: make a specifically named copy on request for continuation references. Permanent per-launch capture identities are a separate improvement in #347. Retention duration is unchanged.

One live session's capture has no row cap. The existing 2,000-row default is a rendering/view limit and may remain; changelog can continue requesting unlimited rendering. This does not require concatenating every historical capture into the default view.

## Done when

- Human-readable text features use printable TTY; exact prompt history stays separate. Native telemetry/parser failures affect only their consumers, not startup/resume.
- Quit leaves capture files in place without asking or deleting; same-tag startup preserves raw/sidecar before reuse, including crash/reboot leftovers. Test archive failures and competing writers so no old data is silently overwritten; redundant archives after interrupted startup are acceptable. Compaction retains its specifically named copy and continuation reference.
- Capture retains output beyond 2,000 rows for a single live session; a bounded render remains allowed and unlimited rendering can recover the full retained output.

## Plan

### Implementation outline transferred from #346

**Files:** shared archive helper in `cmd/internal/launcher` and tests; `cmd/internal/wrapcmd/wrap.go`; `cmd/internal/launcher/osruntime.go`, `lifecycle.go`, runtime/quit/compaction tests; `cmd/internal/artifactpath`, `storagegc` artifact/retention integration; `cmd/internal/slugcmd/slugcmd.go`, `slug.go`, tests; reusable plain-render API in `cmd/internal/scrollbackcmd`; quit UI/help references in `nvim/init.lua`, `pairlifecycle`, README/atlas.

- [ ] Implement red production archive/create tests, then the startup preservation contract above. Reuse artifactpath/retention publication and ordinary raw/events formats; no pending journal.
- [ ] Replace independent destructive wrapper opens with writer ownership and preservation before creation. Close/sync/release on restart. Quit leaves raw/events intact, removes prompt/discard paths, and keeps unrelated cleanup. Keep current compaction copy behavior and exact continuation references.
- [ ] Expose a bounded plain replay API and use it for naming without invented role labels; preserve existing model budgets and proposal validation. Leave exact Pair prompt history intact and optional telemetry locally degradable.
- [ ] Run `go test ./cmd/internal/wrapcmd ./cmd/internal/launcher ./cmd/internal/slugcmd ./cmd/internal/scrollbackcmd ./cmd/internal/storagegc -count=1`, relevant race suites and applicable Lua tests. Update help/atlas with the behavior change.
- [ ] Run `go test ./cmd/... -count=1` and `make build`. Close this issue through SDLC after its own plan approval and verification.

| Risky function | Test strategy |
| --- | --- |
| Startup archive helper | Failure injection around member copy/sync/publication and fresh-process retry; assert old data remains in sources or completed archive before any reuse, allowing redundant archives. |
| Capture writer acquisition / wrapper startup | Competing real processes and exec restart with existing capture; assert one writer and byte preservation on contention/failure. |
| ParkScrollback / compaction | Existing production compaction tests retain specifically named copy and exact continuation reference without writer-lock acquisition. |
| PreserveScrollback / CleanupSidecars | Production quit cleanup asserts no prompt/archive requirement, retained raw/events and independent sidecar cleanup. |
| Plain replay / slug input | Large terminal stream with control sequences; assert bounded model input, unlimited retained raw output and naming without native binding. |

## Log

### 2026-09-29

Operator approved moving unfinished TTY archival/naming work from #346 to this follow-up. No TTY implementation has started. #347 remains the separate permanent-capture-identity improvement; compaction continues making a specifically named copy. Before implementation reconcile this outline with current code and run the planning gates. Native parser failures must remain local to optional consumers.
