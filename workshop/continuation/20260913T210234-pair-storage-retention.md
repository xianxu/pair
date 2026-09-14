---
type: continuation
slug: pair-storage-retention
agent: codex
created: 2026-09-13T21:02:34
branch: 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it
worktree: /Users/xianxu/workspace/worktree/pair/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it
issues: [000239]
---

# Continuation: pair-storage-retention

## NEXT ACTION

In the #239 worktree, run `sdlc state`, inspect `/tmp/pair-239-couch-regression-final.log`, then rerun `go test ./... -count=1` and the plan's race checks. The first full-tree run found capture path alias regressions; these are fixed in checkpoint commit `4d18da7f`, but the final full-tree run is outstanding. Finish verification, real-store metadata-only preview, mandatory M1/M2/issue reviews and publication. The user already authorized implementation; continue without asking for approval again.

_Parked draft at compaction:_

next up #245

## State of play

#239 is working. Implementation is committed as a checkpoint, explicitly awaiting integration verification and review. Neither M1 nor M2 has closed. No binary has been installed, no real Couch-store migration acknowledged, and no user data deleted. All implementation layers landed together; be honest when reconciling plan milestones and review evidence. Do not claim an earlier milestone review occurred.

Final policy: session data60days since meaningful use; Couch-visible including parked/unreadable session data protected indefinitely; archive starts fresh60day grace. Debug logs7days with24h/64MiB segmentation, independent of active tags. Immutable parked captures7days independently, raw/events/creation metadata together, active readers and unfinished handoffs retain. Legacy session metadata gets fresh60day grace; legacy diagnostics/captures may expire on conservative existing age. Exact ownership, unknown liveness/malformed metadata/symlinks/incomplete inventory retain. No global disk ceiling; live-tag session history is not row-pruned. Agent-native stores excluded.

## Thread arc & user model

The thread progressed from Couch issue housekeeping to #239 storage retention. The user distinguished Couch switcher membership from Pair tag lifetime, asked about how much space the policy reclaims, and refined debug and parked-capture retention to7days. They want useful session data kept safely and debug accumulation bounded, with automation after agreement. They dislike repeated permission requests and want questions in ordinary chat, not Alt+up. The latest instruction is to implement our agreed policy; do not reopen the settled spec.

## Artifact map

Read `workshop/issues/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it.md` and `workshop/plans/000239-storage-gc-plan.md` first. The plan revisions preserve superseded capture policy; latest7day agreement is authoritative. The change-code gate artifact in workshop/plans records the approved technical plan. `atlas/storage-retention.md` maps implementation, commands and migration. Both milestone checklists still need honest reconciliation and review.

Worktree: `/Users/xianxu/workspace/worktree/pair/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it` on the same-named branch. Main checkout `/Users/xianxu/workspace/pair` still has earlier dirty plan and untracked gate copies from change-code transfer; preserve those exact paths with a path-scoped stash before merging if needed. Do not discard unrelated main changes (#245 etc.).

Core code: `artifactpath/gc.go` exact ownership; `storagegc/` use intents, actual-process leases, startup handoffs, Couch-store registry, pure policies, bounded inventory, exact quarantine/recovery and scheduler; `gcruntime/` composition and post-readiness workers; `gccmd/` public command; `diagnosticlog/` shared coordinated writer, pagination and collection. Production users wired through launcher, Couch, pairlog, sessionwatch/titlepoller, opener/scrollback/changelog and nvim/retention.lua. New source files need exact artifactpath manifest classification. Lua assets regenerate with `make runtimebundle-generate` before build/full suite.

Newest capture work: producers publish UTC creation metadata with dev/inode/size/mtime identities under a capture-producer lease; collector prefers it after validation. Missing sidecar uses conservative legacy filename/mtime. Invalid sidecar or modified/missing payload retains. ParkScrollback preserves caller path spelling when returning (/var versus /private/var), while metadata uses canonical paths. Keep capture-producer capture-blocking. Process handoff gates register actual child before accessing files; orientation intents bridge selector to renderer; Lua unchanged saves do not touch clocks. Pairlog path aliases normalize the parent and reject leaf symlinks.

## Decisions & dead ends

Automatic deletion and explicit apply require the one-time complete Couch-store acknowledgment: old custom stores cannot be inferred. The approved plan forbids real-store migration/apply as a test. Preview is read-only. CLI `pair gc --register-store PATH`, then `--complete-migration --store PATH` for the complete exact list; empty list explicit. Post-readiness background workers run bounded batches and retain failures. Root lock orders before Couch/diagnostic locks; log writes do not acquire root lock.

Do not infer activity from atime/mtime or owner-directory scans. Session recovery uses durable intents, not assumed atomic content+clock writes. Collection journals exact entries and identities, quarantines before deletion, and never revisits source names after detach. Owner activity incarnation protects tag reuse. Diagnostic exact registry covers external trace paths; old unknown external files are not globbed. Giant unknown or over-budget directories retain rather than opportunistically deleting partial inventories.

## Lessons learned

Use selected explicit environment/owner context in helpers; inherited live session env contaminates standalone test fixtures. Generic changelog/stdin tests now clear Pair owner env. Directory aliases are legitimate on macOS, but leaf symlinks remain unsafe. A deliberately idle Go subprocess helper must sleep on a timer rather than bare `select {}`, which the runtime can declare deadlocked.

## Live deliberations

Outstanding verification: full Go suite, plan race command, latest source-inventory check after metadata consumption, real-store preview and representative apply acceptance with all protections. `make test-lua test-retention` passed `/tmp/pair-239-final-shell.log`; adapt schema passed (60 contending optional appenders yielded8 intact records, permitted); full storagegc passed8.797s; diagnostics race and Linux crosscompile passed; focused main fixtures passed1.761s. Agent gc_process final Couch test log `/tmp/pair-239-couch-regression-final.log` is pending at checkpoint (session18847); all files are stable. The initial full-tree failure log is `/tmp/pair-239-full-go.log`, now stale regarding fixes. No other code workers remain.

Potential review details to assess: empty activity-only sessions now produce cleanup candidates; owner-state enumeration is not separately budgeted beyond inventory; CLI explicit apply uses bounded100groups/logpaths; document retries if needed. README public-command mention and checked managed-reader/writer inventory may still need final documentation. Do not rerun a separate full fresh-eyes reviewer at an SDLC boundary; the binary dispatches it. Resolve all Important/Critical findings before closing. Record measured actuals via sdlc, never guessed.
