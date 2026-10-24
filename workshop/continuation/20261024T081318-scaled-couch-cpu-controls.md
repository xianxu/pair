---
type: continuation
slug: scaled-couch-cpu-controls
agent: codex
created: 2026-10-24T08:13:18
supersedes: editorial-snapshot-compiler
branch: 000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode
worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
issues: [000865]
---

# Continuation: scaled-couch-cpu-controls

## NEXT ACTION

Run sdlc state, read pair#865's latest Log and the scaled-control revision at the end of workshop/plans/000865-live-couch-cpu-plan.md, then start the APPROVED disposable Claude scaled matrix from this clean committed branch. No recorded check, merge runner or native fleet is active; all bounded subagents are done. Do not repeat change-code/plan/estimate gates and do not follow the obsolete #823 NEXT ACTION in .couch/continuation.md.

Create a fresh parent with mktemp -d "${TMPDIR:-/tmp}/pair865-scale.XXXXXX" (prefer /Volumes/TestScratch/tmp when available), verify the nonempty absolute result and choose its new evidence leaf. Log exact invocation/evidence path in the issue; commit it before starting. Use env TMPDIR=/Volumes/TestScratch/tmp GOTMPDIR=/Volumes/TestScratch/tmp PAIR_NATIVE_COUCH_PROFILE_SCALE=1 PAIR_COUCH_PROFILE_EVIDENCE=<fresh-parent>/evidence go test ./cmd/couch -run '^TestDisposableCouchProfileScale$/claude' -count=1 -v -timeout=45m, directing output to /tmp/pair865-scale-collection.log. Monitor every <=60s and preserve exact-owner cleanup evidence on any failed subcase; stop the matrix rather than blindly retry. Native harness owns cleanup, never signal guessed PIDs. A source-clean commit is required for truthful compiled build identity.

## State of play

Couch pair:9, hub ops:0; cwd /Users/xianxu/workspace/worktree/pair-slot9/pair. Current issue #865: Live Couch sustains ~1.3 cores: attribute and bound it. Full-flow change-code already passed at93b8d5593. M1 finite opt-in profiler and initial native controls are implemented; M1 scaled attribution is next. M2 optimization still needs TL approval naming measured owner, workload and budget. No operator live Couch relaunch or environment change is authorized.

Latest implementation commit is332b4664766d8923a3d5e26290aab2466af79826. Worktree clean, gofmt clean. Final focused combined tests passed fixture2.689s, fixturecmd0.256s, testcouch0.788s, wrapcmd0.484s. Workload race count10 passed18.119s, final client/identity/options/provisioning race count3 passed14.926s; repolint corrected PASS8.395s, artifactpath PASS28.534s. Final staged reader inventory and diff checks pass. The issue Log records RED controls and exact reader audits.

TL9f0916d7-6ff1-4cc4-abde-6e1745f3a52a approved the complete scaled revision: N1/5/10/20; fixed100ms redraw through existing mailbox/Run/stdout owner;0 or2000 history lines per actor;15min cap; unprofiled cohort then Detach/RestartSupervisor; generation-fenced CPU/RSS/writes; long-history N20 and same-Couch age. No optimization code yet. Older #860 PR582 is landed and no runner remains; #868 dispatch was withdrawn as pre-#825 stale evidence, do not claim it. #863 landed109986901f7410e0120135121f9e906054603634 is already in this branch and enables the genuinely empty explicit-hub harness. Native startup#866/Codex idle#867 remain known external limits; do not claim a full real-model matrix.

## Thread arc & user model

The original recorder-dominates hypothesis was disproved: image leaves were only about3.37% of sampled stacks; GC/memory, terminal and JSON were larger. The operator/TL wants measured root cause and a bounded correction, not image tuning based on the title. Initial four-agent native controls used500ms SetState rounds and measured only1.20–2.22% CPU, so the TL refined the hypothesis toward25 continuously redrawing slots, long scrollbacks and9h uptime. Scale and same-process age controls are the next approved discriminators. Preserve native authority/cleanup even in synthetic fixtures; those qualify transport and workload, not real model semantics.

## Artifact map

Read first workshop/issues/000865-live-couch-burns-1-2-cores-continuously-in-the-chrome-cast-recorder-png-decode.md, then workshop/plans/000865-live-couch-cpu-plan.md. Its end owns the approved scaled design, bounds and lifecycle; the long-stem plan-gate.md is binary-owned. Initial estimate2.92325h is provisional, especially M2 and experimental work; append revisions after a measured M2 design, never backfit or type actuals. Normal measured actuals apply after ariadne#591; only the separate64KiB snapshot refusal retains its explicit no-actual authorization.

cmd/internal/couchcmd/profiling_model.go is the pure profiler lifecycle; profiling.go is its sole effect interpreter/runtime/file adapter. Launch-only COUCH_PROFILE_DIR is disabled by default, with private new leaf,60s warmup/30s CPU, alloc snapshots, no HTTP,32MiB/file and64KiB metadata, joined shutdown. Singleton child env clears the opt-in. Runtime build identity is compiled information, never a hash of a replaced on-disk executable.

cmd/internal/testcouch owns native fleet lifetime. It now centralizes0..19 workers (1..20 actors); production128 actor,32KiB observations,128 mailbox and4 CLI limits are unchanged. Fleet.SupervisorIdentity returns the CURRENT owned console generation; ConfigureWorkload reuses SetState's correlated cancellation/retirement. agentfixture/workload.go owns pure state,0/2000 history lines, fixed100ms,15min lifetime and counted completed output; Run select is the sole stdout/timer owner. Prefill is exactly2000 lines of118 printable columns plus CRLF=240000 bytes, below256KiB with Working repaint; ticks <=128 bytes. Alternating accepted bullet glyphs preserve native Working authority. Tests cover controlled clock/output/errors/expiry, duplicate starts, cancellation and ownership. Root's real wrapper regression is TestDisposableFixtureRedrawPreservesNativeAuthority in fixture_composer_conformance_test.go.

cmd/couch/profiling_scale_acceptance_test.go contains TestDisposableCouchProfileScale. Six cases per selected personality: N1,N5,N10,N20,N20_history2000,N20_age. Exact observed cohorts persist across unprofiled Detach/RestartSupervisor; every workload acknowledgment must precede capture. Profiles cover fresh reattached Couch60–90s. External30s CPU/RSS/physical-write windows are LATER separate windows; age repeats external window after five EXTRA minutes on the SAME measured Couch generation. Do not call that later CPU profile attribution. Stop receipts capture actual ticks/bytes, not promised throughput. All actor history inputs are real primary CRLF writes. Bounded manifest/profile reports retained before Close/AssertCleaned, partial outcomes preserved, no passing retry. Test exposes Claude/Codex but start Claude only to bound runtime. Reconcile any retained exact owner before another native run.

atlas/couch-profiling.md and couch-testing.md document limits and timing truthfulness. workload.go was individually classified NonArtifactSources. Reader certificates: native scaled consumer and fleet_test are non_go_full_fallback (native helper API); provision_test retains local_or_literal_fixture for its generated topology/literal manifest. Others use only their own fixtures. Always stage new tests then run make test-affected-inventory before any close/milestone; audit named inputs individually, never bulk refresh.

## Decisions & dead ends

Initial native matrix at88c6b3288e1ac5dadfdf707b36b776e9621c5c13 passed415.469s with exact all-generation cleanup. Summary workshop/evidence/000865/native-terminal-controls.md; retained binary/profiles/manifests /Volumes/TestScratch/tmp/pair-slot9/pair865-native.f2WNZr/evidence, candidate SHA256 fcf38decfd8d61415e9eedfd939b908f2bbedad422bd01dbe16f7ea43a5801f8. No native fleet survives that run. Cast off, viewers0, no Chrome, synthetic personalities only. Alloc snapshots are sampled cumulative accounting that can lag GC, not exact window rate or RSS.

Initial read-only live window was135.69% CPU/25s,180.4KB/s process physical writes; no fseventsd causality. Initial sample and current on-disk binary have different UUIDs, so current file revision must never be attributed to that mapped live process. workshop/evidence/000865/initial-measurements.md owns this evidence. No live profiling relaunch permission; TL will ask operator only if disposable scale fails to reproduce.

After scaled attribution: run inventory and sdlc milestone-close --issue865 --milestone M1 for the mandatory binary-owned fresh review. Send concrete measured owner/workload/budget M2 plan to hub and wait before optimization. Final pushed-head pre-close uses sdlc local-check run --pre-close, then close/merge per gates. Do not add a separate redundant boundary code review. No manual main integration after a passing check; merge owns it.

## Lessons learned

Native fixture spinner glyphs are classification authority: ASCII looked valid but destroyed Codex Working, caught RED in the real wrapper and fixed before native scale. Check reducer errors in every Run transition. A mailbox client test must not guess tick timing with a fixed sleep; controlled-clock Run tests prove ticks while client tests prove copied receipt/retirement. Outer native watchdog must include setup/scenario and cleanup, or it destroys the evidence owner.

Use bare whole couch/sdlc commands, no cd/pipes/substitution. Ask hub for scope/operator choices, not in-pane prompts. All reported SHAs come from returned Git. Agent trailer is codex/gpt-6.1-sol/medium; coauthor GPT-6.1 Sol <noreply@openai.com>. Skills sdlc,couch,tl-worker,systematic-debugging,TDD,executing-plans,writing-plans and datatype continuation were loaded. Bounded subagents are all done; no agent edits remain. This continuation is before the long native unit, with no check FIFO or native process to lose.
