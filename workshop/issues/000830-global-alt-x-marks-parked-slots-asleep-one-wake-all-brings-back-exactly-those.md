---
id: 000830
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '25a30036bbf0cdc9e14a4abcf0fdafe25b0b2057' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T07:18:19-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d12da403", done: "f6148d87"}
---

# Global Alt+x marks parked slots 'asleep'; one 'wake all' brings back exactly those

## Problem

## Spec

Operator idea (10-22, night): when the operator does the **global Alt+x** (park everything and quit), Couch should
mark the slots it parks that way as **asleep** ("put to sleep" during a restart), a flag on the parked slot distinct
from a slot the operator parked individually. That also answers the operator's earlier observation that "parked"
mixes two kinds: slots not in use, and slots that were merely running when parked.

- Global Alt+x (and `couch --drain`-driven full shutdowns) records each parked slot as asleep, with the time; an
  individual park, or a slot already parked, stays plain parked.
- On the next start, asleep slots are what a **"wake all"** action (one explicit operator or TL gesture, consistent
  with pair#739's no-automatic-resume rule) brings back; plain parked slots stay parked. tl-lead's fleet resume
  (ariadne#564) uses the asleep set instead of guessing from issue ownership.
- Waking or resuming a slot clears its flag; the switcher shows asleep distinctly from parked.
- Related: pair#827 (Alt+x must finish in one press), pair#739 (cold start stays manual).


**Operator refinement (10-22, night): wake in two stages.** The operator's one gesture wakes only the **asleep hubs**.
Each hub then wakes its own asleep worker slots: **first those with pending work, then the remaining asleep slots**.
So "wake all" at the Couch level means "wake the asleep hubs"; the per-hub ordering lives in the hub's fleet resume
(ariadne#564).

## Done when

- After a global Alt+x, previously live slots show asleep and previously parked ones show parked (test).
- The operator's wake gesture resumes only asleep hubs (test); each hub then wakes its asleep workers, pending-work first (test with ariadne#564's resume); individually parked slots are untouched.
- The flag survives a full restart and clears on wake.

## Plan

Durable plan: `workshop/plans/000830-global-alt-x-marks-parked-slots-asleep-one-wake-all-brings-back-exactly-those-plan.md` (awaiting TL approval).

- [x] T1 record field `asleep` (SleepMark, born additive)
- [x] T2 pure `ThreadAsleep` predicate (nonce-bound)
- [x] T3 BeginPark cause; Leave fresh drives mark asleep; resume/reattach clear
- [x] T4 summary + switcher shows asleep
- [x] T5 `asleep_at` in `couch --recover-plan-from-sdlc` rows
- [x] T5b `couch --wake`: main hub only, from a live hub; resumes the other asleep hubs; idempotent; one summary. Drop the Alt+Shift+x key.
- [x] T6 native fleet acceptance (Claude, Codex): full quit, resume main hub, `couch --wake` -- written as `TestDisposableCouchFleetWake` and skipped until pair#863 lands (see Revisions)
- [x] T7 atlas, lessons, ariadne#564 contract

## Revisions

- 2026-10-23 (TL-approved): the wake key's design is checked against #739's startup, not main's. Under #739 a hub is only reattached when its session survived; a cold resume is the operator's action. So after a full shutdown, asleep hubs stay down and the key is the gesture. Two surfaces changed:
  - **No `couch --wake`.** Broker CLI ops need a live calling slot, and after a full shutdown none exists.
  - **No `couch --asleep --json`.** `asleep_at` is instead added to each thread row of the existing `couch --recover-plan-from-sdlc` report (pair#367). The field is additive (omitempty) and the schema version is unchanged. That report already joins each slot's claims and work evidence, so ariadne#564 reads the asleep set and the pending work in one call. The TL adds a line to ariadne#564's spec.
- 2026-10-23 (operator, via TL): **no wake key; waking is `couch --wake`, run by the TL from the live main hub.** After a full shutdown the operator resumes the main hub from the switcher and asks the TL, which runs `couch --wake`. This reverses the first revision's "No `couch --wake`": that objection assumed no live slot exists, but the resumed main hub is one. Delta:
  - Alt+Shift+x is removed, along with its confirmation frame and the couchkeys rule loosened for it.
  - `couch --wake` is a broker op. Only the main hub may call it. It resumes every other hub whose thread is asleep, each through the ordinary slot-resume queue, and polls them to one summary. It is idempotent: a woken hub is no longer asleep, and an in-flight one is refused as busy.
  - Asleep workers stay asleep for each woken hub's tl-lead resume (ariadne#564), which reads `asleep_at` from the recover-plan rows as agreed.
  - The fleet test drives: full quit, resume the main hub, `couch --wake`.

- 2026-10-24 (TL): T6 is blocked by pair#863. #830 lands on the T5b rig tests. The T6 test is committed but skipped, with a reason naming #863, so it can run as soon as #863 lands. There will be no staged operator check of the wake flow before then, since it would hit the same bug.

## Log

### 2026-10-24

- T6 written (`TestDisposableCouchFleetWake`, in `make test-couch-fleet`). It drives the operator's flow: start Couch with two explicit hubs (the primary, and worker 1's worktree), start the main hub from the switcher's start form, launch the workers, full quit, check all three threads are asleep, start Couch cold (nothing comes back, #739), resume the main hub with "Resume from disk", run `couch --wake`, check that the second hub returns and the plain worker stays asleep, then repeat `couch --wake`. The harness gained an `Options.Hubs` setting (workers that are also configured hubs), `StartCold` (start without waiting for the hub agent) and `QuitConsole`.
- Blocked by #739-era startup, filed as pair#863. Since #739, `Fleet.Start` cannot start an explicit-hub fleet: no hub launches, even in a fresh namespace. On clean main 54c4a59ca, `TestDisposableEditedRestart` fails the same way. A fresh two-hub start also logged `hub failed … attach record/handle process identity mismatch`. Driving the switcher start form brings the hub's agent up, but it never becomes a live broker caller: `couch --actors --json` from it returns `unavailable: caller lookup: recipient is not live` for more than 120 s (head 375adcb6c). So the main hub can neither run `couch --wake` nor launch workers.

- T6's switcher gesture now follows pair#863's measured sequence. Wait for the switcher root (a cold start opens it by itself, so a blind Ctrl+Space would open the start form). Then Ctrl+Space opens the form; type the hub path; press Down, which requests the preview; wait for `args  repo-default`; press Enter once. The second hub is started the same way, since the main hub may not launch another hub's slot. With the skip lifted locally, one Claude run got through both hub starts, the worker, the full quit (all asleep) and the cold start (nothing back), then hit a switcher race on resume, since fixed. Two runs instead failed at the first hub start with a blank screen and `zellij=absent`. Hub start under this harness is still flaky; that belongs with #863. My earlier "recipient is not live" reading came from my own helper pressing Enter repeatedly, not from Couch: pair:7's two-hub run on main registers on the first poll.

### 2026-10-23

- T5 and T7 are done; head aab4ad4f2.
  - Wake is a switcher confirmation, `menu_wake.go`. It dispatches the row resume that Enter uses for each asleep hub, so no new core op was needed (ARCH-DRY). The main hub goes last so the operator lands on it.
  - The Alt+Shift+x binding is Couch-only. `TestSwitcherChordsCarryPairsEncodings` now states the rule for such a key: its bytes decode to no Pair chord.
  - `asleep_at` is in the recover-plan rows.
  - Tests: the wake reducer (exactly the asleep hubs, main last; plain parked untouched; the set re-read at Enter), with a mutation check on the hub filter. Targeted packages are green after three guard fixes: the refusal-case enumeration, the README chord check and the artifact inventory.
- T6 (native fleet) waits for #739 to land, for two reasons. On main a Couch started from the primary has no hub ownership, so wake finds no hub. And main's startup cold-resumes the hub, which would confound the test. After #739, the harness needs a cold start that doesn't wait for the hub agent, then a wake press.

- T1–T4 done (9dcc4a36c, then the switcher commit). A mutation check that writes no mark at BeginPark fails both Leave mark tests. T4 adds no read: asleep comes from the record already in hand, so the planned counted test would have nothing to count.
- **Design discovery, waiting on the TL:** Couch already starts every explicit hub at startup. `startExplicitHubs` queues `start-hub` for each, and `StartInteractive` cold-resumes a proven conversation. Restarting Couch with its hubs therefore already wakes them, asleep or not. A separate wake gesture for hubs would only repeat what startup does. Asked the TL whether wake stays (startup would then skip asleep hubs and wait for the gesture) or is dropped (startup is the gesture, and #830 ships the mark, the display and `--asleep --json`).

- pair:6 claimed and planned. The mark is written in BeginPark's CAS and bound to the park nonce; asleep is derived (receipt nonce = mark nonce), so every park-ending path and older builds leave it inert (ARCH-ORDER, ARCH-EVOLVE). It is a top-level member because nested park types are strict. tl-lead reads `couch --asleep --json`. `couch --drain` (#722) is not implemented yet; the cause is a Leave parameter so it can choose.

### 2026-10-22
- 2026-10-23: T5b done (62c24192e). `couch --wake` is the broker op `wake`, main hub only (`HubOwnership.Wakes`). It reads the asleep set from the store alone (`Couch.AsleepSlots`) and admits one resume per other asleep hub, each with a re-proving effect gate. The CLI polls the receipts (shared `pollSlotOperation`) to one summary. The Alt+Shift+x key is removed. Tests: Wakes table, asleepSlotRows, a service rig (worker and extra hub refused; main hub admits exactly bench and third; a repeat is busy; an empty set gives an empty report), CLI parse, summary and poll round trip, and protocol validation. Mutations of the target filter and the main-hub check both fail the rig. couchkeys, couchmessage, artifactpath, threadrecord, couchcore, couchcmd, couchtty and cmd/couch pass, and test-affected-inventory is clean. T6 still waits on #739: on main a primary launch has no hub ownership, so wake has no other hubs.
