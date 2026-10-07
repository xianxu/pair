---
type: project
name: "cross-slot-work-scheduling"
goal: "Make cross-slot issue work assignable, observable, and recoverable using atomic SDLC claims and event-driven local messaging."
done_when: "Two local slots receive duplicate work requests with one claim winner; owner and workflow evidence are queryable; operator-directed recovery resumes the assigned work after restart; idle messaging performs no recurring global ownership discovery."
status: done
created: 2026-10-01
updated: 2026-10-06
operator: xianxu
mvp_scope: [pair#365, ariadne#277, ariadne#278, ariadne#279, ariadne#280, pair#366, pair#367, pair#362]
explicitly_out: [pair#203, pair#210]
---

# cross-slot-work-scheduling

Make issue work assignable across local slots, observable through SDLC, and recoverable after interruption without relying on the operator’s memory. **Multi-machine Couch and globally consistent runtime discovery are outside MVP.** Multiple operators on different machines may still share Ariadne repositories and issue trackers; each machine runs its own local Couch. This is a project definition for review, not an implementation plan or a commitment to a delivery date.

## PRD

### Why this project exists

The discussion began with a slow workbench after cross-slot messaging landed in pair#353. The editor measured fast, Couch repeatedly consumed more than a CPU core, and the investigation initially pursued a globally current view of processes and session ownership as if it were a prerequisite for safe delivery. The operator rejected that premise: Couch/Pair control their lifecycle protocol, cross-process events cannot be atomic, and minor delivery timing differences during replacement are acceptable.

Profiling then connected costly terminal work with repeated ownership discovery. The larger problem is how to coordinate useful work when requests and observations are imperfect. Place strong guarantees where they matter—atomic issue claims and guarded SDLC transitions—and let agents verify effects through stable read-only commands. Do not compensate for missing lifecycle integration with continuous global scans (ARCH-PURPOSE, ARCH-DRY).

### Goals and boundaries

1. Idle messaging creates no recurring global ownership discovery. Lifecycle events update routing; interaction failures repair the affected registration.
2. Issue claims atomically record who is responsible and where the work belongs. Humans and agents can discover the assignment after shutdown or forgotten context.
3. SDLC owns claim-before-new-work, ownership-checked continuation, workflow observations and operation recovery contracts. Scheduling prompts do not repeat defensive workflow instructions.
4. Agents can inspect another local slot directly through SDLC and other available tools. Observation does not require the other LLM to respond. Observability tools should be fast.
5. An operator-assisted recovery skill reconciles local slots, including parked ones, with tracker assignments and resumes appropriate work without discarding files or taking foreign ownership.

No remote actor routing, distributed Couch supervisor, exactly-once execution of arbitrary LLM instructions, global live-state database, or automatic takeover based on reachability. General build scheduling (pair#203) and the doctor timeout defect (pair#210) remain separate issues. Durable message queues and delivery journals are not presumed requirements; add only for a demonstrated recovery need.

### Messaging and imperfect observations

The current channel is sender CLI → Couch broker socket → recipient Pair wrapper socket → agent PTY. Zellij hosts the pane but does not route the message. The wrapper’s receipt is not an LLM acknowledgement. Composer text, a submitted user turn in the transcript, and a changed issue state are progressively different evidence; retained TTY/transcript observation can resolve an initially uncertain delivery.

Accept a request completing at the prior incarnation during a concurrent replacement: had it arrived milliseconds earlier, that would have been normal. Do not impose instantaneous revocation merely to make internal state appear consistent. Exact recipient identity and duplicate handling still need explicit semantics. A lost response is not evidence of no effect, and an uncertain submission must not be blindly repeated at another recipient.

Reason through broker, wrapper, agent and Zellij crashes using their actual connection/process/PTY behavior. Repair only what the failed interaction proves stale. Use lifecycle notifications for intentional changes and bounded reconnection for actual disconnects. Idle full-state polling is not a substitute for a missing protocol.

### Strong workflow operations and effect verification

The existing claim operation compares and swaps the issue card; a concurrent claim race has one winner. Repeating claim after it is working refuses. That is a useful duplicate-safe operation, not proof that all work following a claim is idempotent. Reopening or reclaiming changes the recovery context.

SDLC must enforce the workflow centrally: claim before starting new work; continue existing work only when the assignment matches. The sender can request “work on #444,” then inspect claim attribution, branch/worktree association, recent activity, milestones and publication. A roughly 30-second check is a revisit example, not a delivery-failure deadline. Unknown/unreadable evidence must remain unknown. Working status proves reservation; file changes suggest activity; passed gates provide stronger progress evidence.

Document each operation’s effects, observation command, repetition behavior, preconditions, lost-response recovery and guarantee lifetime. Distinguish read-only queries, safe refusal on duplicate, convergent retry and operations whose effect may remain uncertain. Agents use these contracts for multi-step reasoning, rather than re-deriving workflow semantics from filesystem probes.

### Durable claimant and operator-directed reclaim

At successful claim, publish operator name, machine identity/name, qualified slot (pair:1) and worktree path atomically with status=working in the ticket’s tracker card. Preserve responsibility while the process is stopped, the slot is parked or the machine is offline. Use repository identity where needed to disambiguate paths and slot labels across operators/machines.

The operator suggested a MAC address as machine identity. Its selection and stability are an open design question, not a settled implementation. No Couch ID is required in ownership: move toward one local supervisor. Git already supplies chronology; no extra claim timestamp is requested. A proposed separate claim ID would distinguish an earlier assignment from a later one, but the current design must first use the existing tracker/card revision if sufficient. A new ID needs a demonstrated consumer, not precautionary metadata.

Add `sdlc reclaim` for operator-directed recovery, assisted by an LLM. Show the old and proposed assignment, require explicit operator intent and use a guarded tracker update. Cross-machine/operator synchronization is out of band; unreachability does not authorize takeover. Reclaim records reassignment, not remote-process fencing. `sdlc move` is part of the publishing flow and is not the proposed reassignment mechanism; do not silently attach reclaim semantics to it.

### Canonical observations and recovery

Expose structured, read-only SDLC observations for assignment, workspace/branch, workflow checkpoint, file/commit activity, gate evidence and publication. Include source/revision and observation freshness where relevant. Reuse authorities and existing inspection commands instead of building a competing workflow database. Direct local invocation in another worktree is supported; an actor proxy is unnecessary. Multi-machine operation would require a separate project because many assumptions change.

Couch singleton scope is proposed as one supervisor per OS user per machine, to be settled with migration/test-isolation behavior. Recovery combines its local inventory with SDLC records. A report distinguishes already running work, a stopped matching assignment, explicitly parked work, a missing checkout, conflicting dirty/branch state, legacy unknown ownership and assignments on another machine. Resume the matching local work on operator request; leave conflicts visible and do not reclaim automatically. The recovery skill consumes binary-owned observations rather than implementing another scanner.

### Acceptance scenario

Two local slots receive duplicate requests for one issue. Exactly one claim wins, atomically recording its assignment; another recipient cannot start new work merely because the issue is working. The coordinator inspects SDLC evidence directly to distinguish reservation, activity and a passed gate, including when the recipient agent is unresponsive. Delayed/lost delivery and restart produce intelligible outcomes without idle global scans. After Couch/machine restart, recovery finds the responsible local slot and preserves its existing branch, dirty files and conversation evidence. Deliberate reassignment goes through operator-directed reclaim, retaining history and refusing stale concurrent updates.

## Estimate

Not estimated or scheduled yet. This definition spans Pair transport/lifecycle and Ariadne ownership/query contracts; implementation designs and migration costs remain to be reviewed. No deadline, planned finish, worker assignment or execution claim is implied. Establish the project baseline before committing to a timeline; issue estimates follow their approved implementation designs through SDLC.

## Breakdown

The first issue removes the measured regression without waiting for the entire ownership project. Ownership is the foundation for reclaim and owner-aware observations. Recovery contracts consume those guarantees; singleton and messaging lifecycle work can be designed independently. The recovery and scheduling skills consume the landed binary contracts. Dependencies below are recorded on the issue details; implementation must obey one issue per branch from main.

- [x] Replace idle messaging discovery with lifecycle events [pair#365]
- [x] Record claimant ownership atomically [ariadne#277]
- [x] Add operator-directed reclaim [ariadne#278]
- [x] Expose workflow observations [ariadne#279]
- [x] Publish operation recovery contracts [ariadne#280]
- [x] Establish local Couch singleton behavior [pair#366]
- [x] Recover locally assigned work [pair#367]
- [x] Schedule work and verify effects [pair#362]

<a id="pair-365"></a>
### pair#365 — messaging lifecycle and performance

Owns the immediate performance regression and local lifecycle/delivery contract. Requires a measured before/after profile, counted idle-probe invariants and failure/interleaving tests. Existing composer safeguards remain part of acceptance.

<a id="pair-365-m1"></a>
### pair#365 M1 — idle messaging baseline

**est:** 6.91
**actual:** 1.24h
**closed:** 2026-10-01

M1 added an in-tree acceptance test that counts the authority probes. It is skipped until M2, and its skip message records the baseline: one healthy wrapper costs 24 ownership probes and 6 `git status` per idle minute. M1 also added `probes/messageidle`, a shim that counts only couch-parented calls. The live baseline was taken read-only from the running Couch (11 wrappers) after the operator found relaunching under a shim too heavy. Couch used 101.6 CPU-seconds in 120 s, with 639 `ps`, 179 `zellij` and 141 `sdlc workspace` children. The unexpected part was the `sdlc` count: wrappers whose registration keeps failing re-run the full check and the workspace resolve every second. Under polling, a failing registration costs more than a healthy one, and lifecycle events remove that cost along with the idle polling.

<a id="pair-365-m2"></a>
### pair#365 M2 — lifecycle protocol replaces polling

**est:** 6.91
**actual:** 0.59h
**closed:** 2026-10-01

M2 replaces polling with lifecycle events:
- **Session:** each wrapper holds one session on a registry socket. The hello is checked against the kernel's peer PID, and EOF means death or exec.
- **Console:** it posts each thread's pane state to a coalescing mailbox, replaying panes that were attached before the service started.
- **Registry:** a pure registry decides which bindings may receive, and the service executes its effects. The full ps/zellij check runs once per admission. Use-time checks only read files and memory.
- **Deleted:** the heartbeat, the reconcile ticker, the verification window and messaging's background git.

Testing:
- **Randomized interleavings:** a broker driven only by the registry's effects has to agree with the registry after every step. The test caught one invariant stated too strongly. While a newer wrapper is still being checked, the older one legitimately stays connected; the spec accepts completion at the old incarnation.
- **Mutation check:** a reintroduced 1 s poll fails the idle test, moving launch checks from 3 to 9 in 2.5 s.
- **Event model:** pane changes are modelled as states rather than attach/exit edges, so coalescing is safe.

<a id="pair-365-m3"></a>
### pair#365 M3 — failure semantics, measurement, docs

**est:** 6.91
**actual:** 1.22h
**closed:** 2026-10-01

What M3 added:
- **Crash and interleaving suite:**
  - an absent endpoint;
  - wrapper death mid-delivery: Indeterminate, never resent;
  - replacement during an in-flight delivery: it completes or goes Indeterminate at the old incarnation, never redirected;
  - delayed frames from a displaced session, including an exec's identical binding;
  - one full start/detach/reattach/exec/Couch-restart sequence.
- **Retained receipts:** each wrapper keeps its last 64. After a Couch restart, `--message-status` recovers outcomes from connected recipients and answers *uncertain* when one is silent.
- **Planned fan-out dropped:** the per-send family fan-out was dropped because the CLI mints a fresh ID per send.
- **Review fixes:** M2-review findings were fixed as a class: newest *admitted* wins, failed effects report back, and broker tombstones are evicted.

Live measurement on the operator's 11-slot setup:
- **Messaging spawns:** about 800 `ps`/`zellij`/`sdlc` per 2 min fell to 0.
- **Couch CPU:** 60.9 fell to 22.5 CPU-s per 120 s, down 63%.
- **A hypothesis that didn't hold:** I thought the old wrappers' legacy `register` requests explained the remainder. Relaunching 6 slots didn't lower it. The remainder is mostly system time from non-messaging file-system polling, notably the Console's 500 ms continuation scan, and goes to a follow-up.
- **Not done:** a live send smoke test between relaunched slots.

<a id="ariadne-277"></a>
### ariadne#277 — claimant ownership

Owns tracker schema, atomic claimant publication, centralized admission and legacy/unattributed handling. Decide machine identity and outside-Couch claims here; avoid adding timestamp/claim-ID fields without a demonstrated need.

<a id="ariadne-278"></a>
### ariadne#278 — reclaim

Depends on ariadne#277. Operator-only reassignment after out-of-band coordination; guards stale updates and preserves work. Distinct from publishing-oriented `sdlc move`.

<a id="ariadne-279"></a>
### ariadne#279 — workflow observables

Depends on ariadne#277. Reuse existing command families and authorities; structured evidence remains available without a running recipient agent.

<a id="ariadne-280"></a>
### ariadne#280 — recovery contracts

Depends on ariadne#277, ariadne#278 and ariadne#279. Declare and test operation-specific retry/effect semantics; one source in SDLC, not duplicated prompt recipes.

<a id="pair-366"></a>
### pair#366 — singleton

Settle production singleton scope and migration from existing stores while retaining supported isolated test environments. No dependency on claim implementation is needed to design local ownership.

**estimate:** 2.66h. **actual:** 6.12h (SDLC measured). **closed:** 2026-10-02
(local codecomplete). Implementation on `000366-couch-singleton` passed the SDLC
boundary review with SHIP and no open findings; published as [PR #196](https://github.com/xianxu/pair/pull/196). Production ownership is per OS account on
machine-local storage; second launches refuse with owner/store context. Selection
preserves the adopted store, Pair data and identity roots in place. Ambiguous legacy
installations remain explicitly UNMIGRATED until operator reconciliation; automatic
inventory consolidation is outside the approved scope. Tests and diagnostics use an
explicit isolated root. No running installation has been cut over by this work.

<a id="pair-367"></a>
### pair#367 — local recovery

Depends on pair#366 and ariadne#277–280. The skill consumes canonical binary observations, reviews conflicting state and invokes reclaim only under operator direction.

<a id="pair-362"></a>
### pair#362 — scheduling skill

Reuses the existing contextual scheduling issue. Depends on pair#365, ariadne#277, ariadne#279 and ariadne#280. Teaches effect verification using direct local tools and existing SDLC rules, without a new centralized scheduler.

## Log

### 2026-10-01 — scope and design decisions

Captured the operator-led evolution from local performance diagnosis to autonomous work coordination. Rejected continuous attempts at global runtime consistency, redundant safety checks without concrete failure effects, automatic ownership takeover, unnecessary actor-proxied observation and premature multi-machine Couch support. Accepted lifecycle protocols, tolerable old-incarnation delivery, effect-based read repair, atomic claimant attribution, direct SDLC observation, operator-directed reclaim and a recovery skill.

The initial assistant proposals for separate claim timestamps/IDs and ownership transfer via `sdlc move` are not requirements. Git chronology is sufficient for time; assignment-generation identification is design work only if tracker revisions cannot supply it; reclaim is the requested transfer verb.

### 2026-10-01 — investigation evidence and limits

The operator’s performance capture at 08:55 showed fast editor legs, Couch at 170.5% CPU, an independent wrapcmd test at 149% and a headless Neovim at 105.5%; Couch remained above one core after those tests exited. The operator-collected CPU-only spindump at 09:04 recorded 13.586 CPU-seconds over 10 seconds. Running stacks included terminal parsing/cell work, allocation/GC and process creation. This does not attribute all machine CPU to Couch.

Source inspection of pair#353 found nominally six Zellij pane queries and twelve global process scans per successfully registered binding per second: two full owner probes per authority check, two checks per wrapper registration, plus one check per background reconciliation. Earlier two-snapshot process churn counts missed short-lived processes; a later rapid observation saw over one hundred distinct Couch children, including many ps/Zellij commands, over roughly three seconds. That is evidence of repeated discovery, not a precise process-spawn rate.

A valid isolated Zellij test observed zero idle bytes between queries and 10,629 bytes after each of three list-panes queries. A subsequent setup still contained a startup plugin despite attempts to disable it, and list-sessions also produced output there. Keep the test limitation explicit: query-driven redraw is demonstrated in that setup, but the live share attributable to messaging and a plugin-free reproduction remain implementation acceptance work. Initial failed probe launches are not evidence. No production performance fix was made in this session.

Local evidence (machine-specific, not portable dependencies):

- `/Users/xianxu/.local/share/pair/repos/990458a6956a804b/perf-capture-1790870123.txt`
- `/tmp/couch-cpu.spindump.txt` — operator’s running-thread profile
- `/tmp/couch-profile-10s.txt` — ordinary sample including waits
- `/tmp/couch-query-experiment/` — raw later experiment and results

`sdlc claim --help`, `ariadne/cmd/sdlc/claim.go` and `TestClaimRaceHasExactlyOneWinner` in `claimremote_test.go` establish the existing CAS/single-winner design; the test was inspected, not rerun during this capture. Read-only/runtime diagnosis and project authoring only; project work remains open for review.

### 2026-10-01 — authoring validation

Project schema validation and a fresh read-only review passed; all eight issue specs have concrete acceptance criteria. Details remain local review drafts, not implementation-ready published work. The project status board printed 0/8 and rendered ariadne#277 blocked even though its authoritative card is open and its local deps list is empty; it completed after a long delay with forecast unavailable. Its displayed zero remaining hours is not an estimate. Resolve that observation/discovery discrepancy before relying on the derived board for scheduling; no issue status was changed to match it.

### 2026-10-01 — approved publication

The operator approved publishing the project and all eight tasks. Initial issue details are now published on their respective main branches through `sdlc issue move-detail`; each publication confirmed creation complete. The project remains defined, with implementation unstarted and dependencies preserved. The independent entry tasks are pair#365 (messaging lifecycle/performance), ariadne#277 (claim ownership), and pair#366 (singleton). Publication does not waive downstream dependencies or implementation-design gates.

### 2026-10-06 — transition evidence

- reality-check: Never baselined: the project was defined 2026-10-01 and executed without a commit step; all 8 tasks were done by 2026-10-06. Forced to record completion, not to set a forecast.

### 2026-10-06 — transition evidence

- issues-cover-prd: All 8 MVP-scope issues (pair#365, ariadne#277-280, pair#366, pair#367, pair#362) covered the PRD and are landed; the Breakdown lists each.

### 2026-10-06 — retro

**Outcome.** All eight MVP issues landed between 2026-10-01 and 2026-10-06:
pair#365, ariadne#277–280, pair#366, pair#367 and pair#362. Each part of
`done_when` has evidence:
- **Duplicate requests, one claim winner:** pair#362's live exercise on the
  throwaway pair#401. `pair:5` won the claim; `pair:6` reported the owner back to
  the sender and did not start.
- **Owner and workflow evidence are queryable:** `sdlc issue show --json`
  (ariadne#279) and `couch --peek` (pair#362), both answering with the recipient
  idle or parked.
- **Operator-directed recovery:** pair#367's recovery report and skill, plus
  `sdlc reclaim` (ariadne#278).
- **No idle global discovery:** pair#365 removed idle messaging probes, with a
  counted-probe acceptance test and live measurement.

**Not baselined.** The project went from `defined` to execution without a commit
step, so there is no deadline, planned finish or fog factor. The
`committed`/`executing` transitions were forced on 2026-10-06 to record completion,
and the ledger row is skipped rather than invented.

**What went well:**
- One issue per branch from main held throughout.
- Each piece reused existing authorities instead of adding parallel ones: claims,
  observations, recovery contracts, the scrollback renderer and the transcript
  resolver.
- Live exercises found real defects that tests had not:
  - pair#362's peek resolved the wrong data directory;
  - a receipt's detail is empty while a message is held;
  - pair#387 (adjacent work) remembered setup failures without considering weave
    itself.

**What to change:**
- **Commit the next project to a baseline before executing it.** Otherwise velocity
  calibration learns nothing from it.
- **Fix the sdlc reviewer's verdict capture (ariadne#300).** It lost the verdict
  twice in two days, on pair#387 and pair#362.
- **Make fresh slots usable by their own agents (ariadne#299).** The dev-alias
  `couch` cannot build in a fresh slot.

**Follow-ups:**
- pair#396: TL-driven project execution from a driver repository;
- ariadne#297: verification authority and the smoke-test state;
- ariadne#298: the product-lead skill;
- pair#403: `repo:N` resolution latency;
- pair#402: add slot from a parked `:0`, fixed on 2026-10-06.

### 2026-10-06 — close

- fog: n/a

## Revisions

### 2026-10-01 — operator review and publication

Preserved the operator's requirement that observation tools be fast, and recorded approval to publish this definition and all eight issue details. No implementation scope or issue status changed.

### 2026-10-01 — #366 implementation scope

Recorded the operator-approved singleton/refusal and in-place adoption scope, its
2.66h estimate and current verification stage. Preserved the multi-store migration
limitation and downstream dependencies; no project baseline or completion claim changed.

### 2026-10-02 — #366 accepted locally

Replaced the earlier “awaiting verification/review” checkpoint with the final SHIP
verdict, measured 4.45h actual and draft PR #196. The gate marked #366 codecomplete
and ticked its task; it is not merged or deployed. The adoption limits and downstream
dependencies remain unchanged. All four review findings were resolved with regression
and mutation coverage; the two disclosed full-repository failures reproduce on base.

### 2026-10-02 — #366 integration acceptance

The operator requested merge. Main was integrated with a lessons-only conflict;
renewed review found and resolved derived isolation-root containment (BR-5). Final
review returned SHIP with all five findings addressed. Updated measured actual from
the initial 4.45h close to 6.12h and removed the stale draft-PR description. Regression
and full command/launcher verification passed; SDLC owns merge and archive completion.

[pair#365]: #pair-365
[ariadne#277]: #ariadne-277
[ariadne#278]: #ariadne-278
[ariadne#279]: #ariadne-279
[ariadne#280]: #ariadne-280
[pair#366]: #pair-366
[pair#367]: #pair-367
[pair#362]: #pair-362
[pair#365 M1]: #pair-365-m1
[pair#365 M2]: #pair-365-m2
[pair#365 M3]: #pair-365-m3
