---
id: 000650
status: codecomplete
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'e2693fd703e5e413da07850c525cfee6d0d57152' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T12:13:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d3724836", done: "d2346f2b"}
actual_hours: 0.86
---

# Slug line off by default under Couch (PAIR_SLUG=on to re-enable): no per-turn pair slug spawn or Haiku call

## Problem

## Spec

Operator decision, 2026-10-17. The draft's slug line (line 1, a summary of what the slot is doing) is off by default when the wrapper is Couch-hosted. Measured by pair:2: about 70 `pair slug` spawns and about 36 Haiku calls an hour across the fleet, with ~570 MB transient memory peaks per call and about 3% of a core on average. The operator doesn't use the slug under Couch.

- When `launcher.CouchHostedEnv` is true, `wrap.go`'s `maybeSpawnSlug` returns before spawning anything: no `pair slug` process and no model call.
- `PAIR_SLUG=on` turns it back on under Couch. Outside Couch, behaviour is unchanged.
- Help, the README and the atlas say so.

## Done when

- A test: Couch-hosted env runs nothing — neither pair-wrap's turn-end spawn nor the `slugcmd.Run` body the Claude Stop hook reaches through `bin/pair-slug`; with `PAIR_SLUG=on` both run; outside Couch both run as before.
- After the next restart and relaunch, no wrapper-spawned `pair slug` process and no slug model call (`claude -p`) appears across the fleet during normal turns. Checked live.

## Plan

Quick flow — a few lines.

- [x] Test first (`cmd/internal/wrapcmd/slug_spawn_test.go`): couch-hosted env → `maybeSpawnSlug` spawns nothing; couch-hosted + `PAIR_SLUG=on` → spawns; non-Couch → spawns as before. `t.Setenv` pins both couch vars and `PAIR_SLUG` in every case so the ambient (hosted) test env can't leak in.
- [x] Gate at the top of `maybeSpawnSlug` (`wrap.go`): `launcher.CouchHostedEnv(os.Getenv) && os.Getenv("PAIR_SLUG") != "on"` → return before the debounce bookkeeping or the spawn. Reuses `launcher.CouchHostedEnv` — no new env parsing (ARCH-DRY). `PAIR_SLUG` re-enables only on the exact value `on` (the knob the operator named); any other value stays off, so a hosted session fails closed to the operator's default.
- [x] Docs: `maybeSpawnSlug`'s comment in `wrap.go`, a README line (Notifications section, next to the wrapper env knobs), and the trigger bullet in `atlas/architecture.md`'s auto-orientation slug section. `pair --help`'s usage text deliberately excludes behavior notes (its own comment says so), so "help" = the wrapper's behavior docs; flagged to the hub in the plan post. Hub confirmed: leave `pair --help` as is.
- [x] Record in Log that the live fleet check needs the hub's restart + relaunch.
- [x] Hub fold-in: the same Couch gate in `slugcmd.Run` (the body shared by the `bin/pair-slug` Stop-hook symlink and `pair slug`) + `TestRunSkipsUnderCouchHosting` + doc lines + the dispatcher slug-route test's couch-var pin.

## Revisions

### 2026-10-17 — hub scope fold-in (external Claude Stop-hook path)

The hub ruled before close: if the Claude Stop-hook `pair-slug` path still
spawns under Couch, #650 is not done. It does — the hook runs as claude's child,
inherits the hosted env, and reaches `slugcmd.Run` through the surviving
`bin/pair-slug` busybox symlink without ever passing pair-wrap. Delta: the same
`launcher.CouchHostedEnv && PAIR_SLUG != "on"` gate now sits at the top of
`slugcmd.Run`, so both entries no-op when hosted; plus its test, doc lines, and
an env pin on the existing dispatcher slug-route test.

### 2026-10-17 — close review round 1 (BR-1/BR-2/BR-3, FIX-THEN-SHIP)

- **BR-1 (ARCH-DRY):** the rule written out at both call sites now lives in
  `launcher.SlugEnabled(getenv)` — one owner for the predicate, the knob name
  and its exact `on` value; both sites call it. `TestSlugEnabledTruthTable`
  covers hosting via scope OR tag and the fail-closed values ("ON", "1", "0",
  "off").
- **BR-2 (stale Done-when):** `## Done when` updated (above) to name both gates
  and to make clause 2 about the model call and the wrapper spawn — a hosted
  Stop hook may still start a short-lived `pair-slug` process that stops at
  the gate, so "no pair-slug process at all" was wrong as written.
- **BR-3 (live check):** clause 2 remains pending the hub's Couch restart +
  slot relaunch on the new binary; the evidence it needs is process-table
  output across the hosted slots over normal turns after relaunch, showing no
  wrapper-spawned `pair slug` and no slug `claude -p` call. Recorded here and
  flagged in the landing report; a review checkout cannot restart the fleet.

## Log

### 2026-10-17
- 2026-10-17: closed — RED: TestMaybeSpawnSlugOffByDefaultUnderCouch, TestRunSkipsUnderCouchHosting, TestSlugEnabledTruthTable fail on main code (no gate). GREEN: launcher, wrapcmd, slugcmd, dispatcher tests pass. Counterfactual: reverting the slugcmd.Run gate re-reds TestRunSkipsUnderCouchHosting (observed). Both entry points — pair-wrap turn-end spawn and slugcmd.Run via bin/pair-slug, including the Claude Stop hook — gate on launcher.SlugEnabled, one source. Full suite exit 0 on 7b72b05c0 (10m36s) after one load-flake re-run (run-headless-test clean-quit timeout under concurrent fleet suites; passes in isolation). --no-ledger: BR-3 (live-fleet-acceptance, kind: unverifiable) is waived because its evidence — ps across hosted slots after relaunch onto this fix — can only exist post-landing (merge, pair:0 rebuild, hub Couch restart, slot relaunch); close gates the merge, so waiting would deadlock. The live check remains assigned to this slot after the hub restart, recorded in the issue Revisions/Log and flagged in the landing report.; review verdict: FIX-THEN-SHIP

- Claimed via the hub's dispatch (pair:8). Quick flow confirmed: gate + test + three doc lines.
- `launcher` is already imported by `wrap.go`; `launcher.CouchHostedEnv` is the existing hosted predicate (scope or tag non-empty).
- TDD: red first — couch-hosted case spawned (`spawns = 1, want 0`) — then the gate landed and all three cases pass; whole `wrapcmd` package passes (124.5s) with `go vet`/`gofmt` clean.
- Docs landed: `wrap.go` comment, README "The orientation slug" paragraph in Notifications, `atlas/architecture.md` trigger bullet, plus a coherence note in `atlas/couch.md` (hosted rows carry a slug only with the knob on).
- Added a `SLUG-skip` debug line in the gated branch (e2b491373): a hosted session with `PAIR_WRAP_LOG` set shows why the slug is quiet — the BEL-skip/EMIT-skip forensic pattern — instead of looking like drift. Targeted tests re-run green.
- Full suite green twice (sdlc-local/suite exit 0: 12m2s on 3ee6e4199, 11m56s on b050580d6).
- Close attempt refused on actual measurement ("suite attempt 2a875dd8 has 0 matching pinned slot engagements"); reported to the hub. Hub: ariadne:1 investigates the Qoder slot measurement gap; close with --no-actual naming it.
- Stop-hook fold-in (see Revisions): `slugcmd.Run` gate landed with `TestRunSkipsUnderCouchHosting` — red proof against ungated HEAD showed the hosted run reaching the session lookup (`native session is unbound`) instead of the gate line; with the gate, whole `slugcmd` package green, dispatcher slug-route and `pair-go` slug-route smokes green (the latter's clean testpair env is non-hosted by design — "no ambient Pair, Couch … values cross this boundary").
- Close review round 1: FIX-THEN-SHIP, 3 Important + 1 Minor. Fixed the class of each: BR-1 → `launcher.SlugEnabled` + `TestSlugEnabledTruthTable` (scope-or-tag hosting, fail-closed values); BR-2 → Done-when rewritten (see Revisions); BR-3 → recorded pending the hub's restart + relaunch (see Revisions). Reviewer independently red-proofed both gate tests by reverting each gate in the review checkout.
- Live fleet check (Done-when 2) needs a Couch restart + slot relaunch on the new binary — the hub's/operator's move; flagged in the landing report.
- Full suite green on the fix head too (sdlc-local/suite exit 0, 10m36s on 7b72b05c0), after one load-flake re-run: run-headless-test's clean-nvim-quit hit its timeout under concurrent fleet suites and passes in isolation — diff-irrelevant (no nvim/embed surface in this window).
- Close review round 2: 0 new findings; BR-1/2/4 disposed addressed. BR-3 (live-fleet-acceptance, kind: unverifiable) stays open — its evidence (ps across hosted slots after relaunch onto this fix) can only exist post-landing: merge → pair:0 rebuild → hub restart → relaunch. Closed via the precise `--no-ledger` flag; the live check remains assigned to this slot after the hub's restart and is flagged in the landing report.

### 2026-10-18 — pair:3 takeover and integration flake

Hub dispatched the pushed unclaimed handoff f046dcd0a58fb7c1b5ad08b423e2e74db4855131; claim resumed it with codecomplete unchanged. Existing close 7007a51996279afc656897a52e7c2980e913414e is finalized, reviewed code 7b72b05c05b6484fd33174ba17f011e2bd8a80ec; its recorded suite pass is exit 0 in 10m36s, log 74b8607e5004. No new close generation was needed. Opened PR 434; normal merge integrated main cleanly to aa8176700b0e50b230c72eeaf190c9b284342810, and all four GitHub checks passed.

The integration suite attempt 8f2fa5ea50afe6009e976e25261f2029 failed in startup cancellation (SIGINT at 20ms), exit 2 in 2m45s after 1565s queue; retained log .git/sdlc/test-stats/failed/8f2fa5ea50afe6009e976e25261f2029.log, SHA256 prefix ccfba26078cd. Captured KeyboardInterrupt was ignored in an importlib module-lock weakref callback; helper21478 remained alive with descendant group21486, caller default_int_handler. This differs from #598 inherited SIG_IGN; its fix is present. Helper and fixture have no #650 diff. Filed unclaimed #674; #671 is the duplicate from #659 and is already wontfix. Its evidence will be consolidated into #674.

The merge runner automatically queued a diagnostic main baseline (db5ddb4e, attempt4135a602541e54d2c39ebb218696ab89). Hub explicitly authorized cancelling that queued owned baseline through sdlc local-check cancel and one issue-backed unchanged integration rerun (receipt8377ca3c-8fee-4168-b2b0-021ea15cbade, verified submitted). Cancel succeeded; the original merge exited1 with retained failed attempt, and baseline recorded no result. One rerun is now queued on exact unchanged aa8176700b0e50b230c72eeaf190c9b284342810, attempt5ce10fffa65a318e1fa64856b2fb2133. A second failure requires stop/report, not another retry. After #650 lands the hub assigned #674 to this slot.

Live fleet acceptance is still pending the hub restart plus slot relaunch; no fleet result is claimed by this takeover.
