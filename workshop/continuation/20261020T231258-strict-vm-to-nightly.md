---
type: continuation
slug: strict-vm-to-nightly
agent: codex
created: 2026-10-20T23:12:58
supersedes: vm-image-after-zellij
branch: 000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop
worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
issues: [000743]
---

# Continuation: strict-vm-to-nightly

## NEXT ACTION

Resume pair#743 M3. Read issue743's latest Log and the durable plan's governing Revisions, then sdlc state/tl-worker --help. M1 and M2 are CLOSED SHIP; do not reopen them or redo image preparation. Implement the already-approved deterministic nightly command and task-declaration handoff with TDD. The supported non-required conformance declaration and named-check acceptance moved from M2 to M3 by authenticated hub approval; add them only after ariadne#509 lands. First inspect the two recorded checks still owned by this checkout (below); never claim a pass from their partial logs or stop another slot's check. Final M3 code needs its own recorded pass before landing. No user clarification or renewed approval is needed for the accepted M3 scope.

## State of play

Same issue branch in /Users/xianxu/workspace/worktree/pair-slot3/pair, owned pair:3. M1 closed77bdf12fb78b; M2 closed56353b1e9efc284e12dfefb5cc8119d145fa4e65 after two review rounds each. M1 BR-1/2/3 and M2 BR-4 are disposed; no open blocker. Latest documentation checkpoint69029390a is committed; the writer will add the next recovery commit. No PR, issue close, merge or M3 implementation yet. #748 was already landed PR500; don't reopen it.

Two local-check processes remain: tool95639 on3b665b4b is running its full suite (most recent output near end of Go packages, not a final outcome), log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/757deecbc900dbed8edc241df60d7f2b/run.log. Tool24724 on3caa035b is queued, log .../inflight/3f8561bf226dcce57c48c54ddc2cdfc2/run.log. Both came from sdlc local-check run --name affected, conservatively falling back to suite. Cancel only through sdlc local-check cancel when superseded; it stops this checkout's runs only. No recorded suite pass has been claimed. Tool sessions may need inspection through the CLI/logs after restart. No native VM command remains active, no claim retained; the qualified image is stopped.

ariadne#509 is actively implemented by ariadne:3 after operator plan approval. Latest authenticated receipt3bba62e6-0310-4934-816e-eb1ec1371d8b reports provider M1 protects descendants after immediate leader exit, with a real6s control and30s grace, no VM lease probing and no suite allocation for runner resources; M1 review then M2 gate wiring remained. Capability isn't landing or Pair activation. Peer checkout for read-only source: /Users/xianxu/workspace/worktree/pair-slot3/ariadne; read its AGENTS.local.md/MEMORY.md when touching it. Do not inject candidate config or hand-publish statuses. #744 genuine fleet/agent coverage remains unlanded; current evidence covers legacy/native stand-ins. #745 owns promotion after seven clean nights; hosted PR conformance stays REQUIRED and activation OFF.

## Thread arc & user model

The user values causal native evidence, strict refusal on unknown cleanup, and autonomous progress under settled approval. M1 revealed real Darwin provisioning issues, then review corrected activation authority and bounded publication. M2's real producer exposed Go metadata and one inherited socket-root defect; both were fixed and retained as failed attempts, not retried into a silent pass. The TL approved moving only dependency-bound declaration/acceptance into M3 so the strict conformance boundary could close while #509 proceeds. Preserve that split, the machine's existing development VM/cache, and the single scheduler. Don't ask again for approved work; ask the hub for genuinely new scope/mechanism decisions.

## Artifact map

Read workshop/issues/000743-conformance-run-couch-zellij-conformance-in-a-local-tart-vm-as-a-declared-local-check-github-job-becomes-a-non-required-main-weekly-backstop.md, then workshop/plans/000743-local-vm-conformance-plan.md. Revisions supersede the original pair-test refresh, plist/service proposal and M2 named-check placement. The issue Log retains receipts, failed attempts, verification and M3 prep. atlas/vm.md maps delivered state. Gate ledger and -m1/-m2-review.md sidecars in workshop/plans are binary-owned; never manually edit them.

M1 is cmd/internal/vmcmd: pure Advance/RunState, Runtime/hostRuntime/VMDriver, shared unreaped-child owner, two machine lanes, read-only tracked snapshot, bounded guest-local export, CLI/provisioning. Qualified image pair-conformance-tahoe-20261021T044207.013507000 lives at native account ~/.tart/pair-runner/image.json. Pinned official Zellij0.45.1 binary SHA256 ca5f9333735bdbc59a621f1d8ed8e24798845302a28ff175d253d4793d5a4a2c; Homebrew's same-version binary differs. pair-test and cached sources were preserved. No new image work is needed.

M2 is tests/conformance-vm.py (single-pass validate_results; NATIVE_REQUIRED/LIFECYCLE_REQUIRED/LIFECYCLE_SKIP; test_pattern; host/guest commands), tests/native-terminal-ci.py (shared strict streaming consumer), and tests/conformance-paths.json plus guard/tests (both hosted trigger lists). Make conformance-vm invokes the generic train runner once, literal argv,20m execution plus shared10s cleanup. Host-only PAIR_VM_ARTIFACTS_DIR becomes --artifacts and is never forwarded into guest. Guest regenerates ignored runtime assets, then existing terminal/lifecycle populations; #432 exclusions remain. Raw Go JSON stdout is separate from setup/oracle stderr. The command/selection tests enforce hosted Make packages/cases and package-specific exclusions, including independent edits and an exclusion moved to the wrong package. test-vm-conformance runs40 portable checks. Shortcut fixture now owns a short socket root; long-root socket-bind/cleanup regression and expanded native-fixture-roots lint prevent recurrence. Fixed SOURCE_GUARDS and audited reader hashes were updated narrowly.

Evidence lives in workshop/evidence/000743. Final shared-selection native qualification at source04ec9e29e476fb5d312e73e85acd1f6750bc6080 has410 events/all28 required cases in five packages, result0/no claim (m2-final-qualified-*). Earlier qualified pass409 events and both causal failures are retained too. m2-known-bad-* contains the one-line deliberate skip diff, exact result1 and required-case skip; m2-post-bad-clean-reuse-result.json proves subsequent train reuse0. Private known-bad checkout /Volumes/TestScratch/tmp/pair-slot3/pair743-known-bad.HtjPy7/checkout has that intentional edit; don't accidentally use it as good input.

m2-make-cancel-* plus qualify-make-cancel.py qualify actual public Make group SIGTERM after positive owned launch and guest command observation: cleanup2.54414775s, result130/no retained claim, outer Make-15. Scratch /Volumes/TestScratch/tmp/pair-slot3/pair743-make-cancel.xlsjS1 contains exact original harness/artifacts. The operator's provider grace is30s, authenticated receipt065e5c59-fc92-4531-8731-3d3335725f87. Provider was told to preserve group grace after immediate Make exit; replyf1ee074c-7076-498f-9452-d9777c0249de verified submitted.

## Decisions & dead ends

M3 authoritative deliverables are scripts/vm/nightly.sh and scripts/vm/scheduled-task.md, not a plist, launchd, cron or private timer. #737 is the one Couch scheduler; until it lands the hub/operator manually invokes the same deterministic command. Handoff03:30 local, machine scope, missed skip; no retries, streak counter or promotion state. Ops owns registration/code/project, not this worker. Safe ops source /Users/xianxu/workspace/worktree/ops-slot1/ops has an existing TestScratch schedule shape; #737 was still unlanded. Recheck the landed parser before registration. Existing fields version/cron/timeout/scope/missed/script_run/heavy (optional owner in candidate); heavy/resource semantics need matching to #509 without taking a suite slot for runner-owned VM capacity.

Nightly runs one exact origin/main through the declared check, no checkout switch. Current installed local-check supports --rev and explicitly named selection independent of path eligibility, inherits env into Make, and records exact-head log/status. Verify the new #509 real route after landing; static/fake proof isn't named-check acceptance. A newly added declaration is read from main, so production bootstrap can happen only after #743 lands. No bypass via candidate declaration injection. Keep non-required/no substitutes so PR conformance isn't weakened. Operator holds local-gates-verified; never apply it yourself. Prepare concrete PR/verification before asking hub for the label.

Retain sixteen caller-owned nightly bundles (4GiB) separately from eight ordinary runner bundles; failed/partial nights count and saturation refuses for inspection/archive, never evicts burn-in evidence. Generic Acquire rejects explicit artifact destinations inside ~/.tart/pair-runner, so choose a separate caller-owned root, atomically reserve a date directory and a fresh child result destination. Qualify cap/concurrency, duplicate-night refusal, failure/cancel status and env propagation with a stateful local-check fake. No VM lease/resource duplication or new queue.

The timeout envelope remains N+cleanup up to10s, one lazy shared scope. Runtime.Release receives reducer state: cleanup flags don't authorize activation. Publication rechecks original caller cancellation and final combined64KiB metadata bound. Third VM lifecycle mechanism change still requires stopping for hub options review; BR-1 authority correction was explicitly approved as not a new mechanism (receipt129ad731-51b9-40a9-94e3-0b7bba5c213e). Fixture short-root ownership and consumer-equality correction don't change that lifecycle.

## Lessons learned

Use bare whole-shell sdlc/couch commands, authenticate incoming messages, and verify submitted outcome receipts. sdlc gates own mandatory fresh reviews; don't dispatch duplicates. Run sdlc agent-id and use its current trailer; this session returned codex/gpt-6.1-sol/medium with GPT-6.1 Sol coauthor. Canonical continuation writer owns commit/push/restart; don't separately restart or handwrite .couch.

Before expensive checks audit changed/new Go readers and fixed SOURCE_GUARDS exactly, plus independent affected-consumers recipe/file ownership and actual upstream affected config. Never bulk-refresh hashes. Full checks on final code are required before landing; native failure must be explained and preserved, not blindly rerun. Current lessons are already in workshop/lessons.md and issue Log; no extra pensive is needed. Preserve unknown provenance through the final retained codec; producer metadata is not case completion; long TMPDIR is distinct from explicit ZELLIJ_SOCKET_DIR; parent exit isn't group cleanup.
