# Boundary Review — pair#618 (whole-issue close)

| field | value |
|-------|-------|
| issue | 618 — Slots recovered by reboot or resume after a Couch restart can send but not receive |
| repo | pair |
| issue file | workshop/issues/000618-slots-recovered-by-reboot-or-resume-after-a-couch-restart-can-send-but-not-receive.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9e5c1bd33ac31606f68fdc5bc3c1f0485d1d98e0..d6a2652056b9135969d642f9d70b29487a4ae517 |
| command | sdlc close --issue 618 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T19:40:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Now read the code diff.

Checking how a refusal is keyed against `couch --list` row labels, then the relaunch transport budget and the M1 open findings.

Confirming whether relaunch runs async under an operation deadline shorter than its new 60 s wait.

Checking the pasted envelope matches what's logged, plus the Pair log's growth bound.

Running the affected test packages, then a revert check on the relaunch fix.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M2 works and the tests back it. When I reverted the two behaviour fixes, their tests failed. Removing the relaunch completion-wait override (`relaunch.go:156`) turns `TestRelaunchRidesOutASlowQuitOnTheFirstCall` red. Dropping the `Submitted` guard (`peer_delivery.go:607`) turns `TestPeerDeliverySubmitsAfterDelayThenConfirms` red. All affected packages pass (couchcore, couchcmd, couchmessage, pairlog, sessionwatch, wrapcmd), and so do both Lua pairlog tests. Three things stand between this and SHIP. First, the new unauthenticated `refusals` op sends back the sender's full `Binding`, which includes the session and nonce that `Broker.Caller` uses to authenticate requests, plus the message body. Second, the Pair log now grows with every delivered peer message, and nothing bounds it or says how big it can get. Third, the M2 live check (Task 8 and the Done-when "Live" line) has no observation in the Log, even though M2 is ticked. The three M1 Minors (BR-7/8/9) are still in the code at head; this range doesn't touch them.

### 1. Strengths
- **Logging peer turns in the Pair log (TL option B)** reuses the existing correlation path and the locked, retention-managed writer (`pairlog/runcli.go:551`). There is no new ledger record kind for older readers to choke on. The append ID `couch-peer-<id>` stops a repeat from adding a duplicate, and `pairlog/peer_turn_test.go` covers that.
- **`sessionwatch/peer_turn_test.go`** builds the log with the wrapper's real writer rather than a hand-written literal, and has a negative case (an unrelated round confirms nothing).
- **The Lua history filter** is careful. `replace` counts only operator entries but splices by byte position, so peer entries survive an edit (`nvim/pairlog.lua:100-108`, test `pairlog_test.lua:844-851`).
- **The relaunch wait is scoped to relaunch.** It goes through the context (`WithParkCompletionWait`), so a plain park keeps its 15 s bound. The CLI polls for up to 5 minutes (`messages.go:204`), so the new 60 s wait fits.
- **`receiveStateText`** is a pure function, and it splits refusals into "relaunch" vs "send again" by who is at fault.

### 2. Critical findings
None.

### 3. Important findings
- **ARCH-SECURE: `refusals` exposes credentials.** The handler returns `s.broker.LatestRefusals()` (`message_service.go:815-820`) to an unauthenticated op. Each `Receipt.Message.From` is the sender's full `Binding`, which `refuseLocked` stores (`refusal.go:53-54`). That binding holds Scope, Tag, Session and Nonce, exactly the tuple `Broker.Caller` checks (`broker.go:357-365`). The body comes along too. `--list` only needs `Detail`. **Fix:** return a projection such as `map[string]struct{Detail string; At time.Time}`, and add a test that the response carries no binding or body.
- **ARCH-FUNERAL / ARCH-CONSTRAINTS: the Pair log now grows per peer message.** Each `Submitted` delivery appends up to 8 KB (`MaxBodyBytes`) to `log-<tag>.md`. A hub slot takes dozens of deliveries an hour (the Log counts 37 in 1.2 h). Every append re-reads and parses the whole log under its lock (`store.go:85,136`). The nvim status line re-reads and parses the log through `read_history()` (`init.lua:2377,2435`). The plan has no line saying how big this writer lets the log get or what removes it. **Fix:** add a Revisions line naming per-message growth and its bound. Or cap peer entries (for example, keep the last N `couch-peer-` entries, since correlation only needs entries after `--pid-not-before`).
- **The M2 live step has no recorded observation** (family `unmeasured-operating-claim`, 2nd finding). Task 8 calls for rebooting one codex slot and resuming another, sending to each, then relaunching the rebooted one after peer messages only. The Done-when "Live" line asks the same. The Log has M1's after-numbers but no M2 live result, yet M2 is ticked. **Rule covering this family:** every Live step in the Plan or Done-when gets a dated Log entry with its observation before its milestone is ticked, or a Revisions entry that defers or drops it. Check both milestones' Live steps against that rule, not just this one.

### 4. Minor findings
- **Refusal keys can miss list rows.** `LatestRefusals` keys by the raw `route.Target` (`refusal.go:87`). A send spelled with a repository alias (rewritten later by `canonicalTarget`) or to a family target is never shown on a slot row. Supersession compares that raw key against the resolved `To.Slot` (`refusal.go:96`). Disambiguated duplicate labels (`label (…tag)`) also miss. The TL's `pair:N` case works.
- **A refusal outlives its fix.** After a successful relaunch, `--list` keeps advising "relaunch" for up to 10 minutes, until a later send is accepted.
- **The `--list` wiring has no test.** The call in `run.go:664-667` that hooks `listRefusals` into the list output is untested; only its two halves are.
- **The new park tests depend on wall-clock timing.** `park_wait_test.go` and `TestRelaunchRidesOutASlowQuitOnTheFirstCall` use `time.Since` with sleeps of 100–300 ms. That is one interleaving, with some risk of flakiness under load (ARCH-ORDER).
- **The managed write path isn't tested for the wrapper.** `PersistPeerTurn` is only exercised with an empty getenv, i.e. unmanaged. `managedLogWrite` under the wrapper's real `PAIR_*` environment is not covered.

### 5. Test coverage notes
- Revert check: removing the relaunch override and removing the `Submitted` guard each turned their test red.
- The Lua peer-filter test passes.
- No test covers `RelaunchWait == 0` defaulting to `RelaunchParkWait` (one-line logic, low risk).
- No test pins what the `refusals` response omits; the ARCH-SECURE fix needs one.

### 6. Architectural notes

| Marker | Result | Note |
|---|---|---|
| ARCH-DRY | pass | `ValidateRequest` now shares the no-fields rule between `resources` and `refusals`. The `couch-peer-` prefix is defined twice, in Go and Lua; acceptable across languages, and each copy cross-references the other. |
| ARCH-PURE | pass | `receiveStateText` is pure; the IO is the thin `listRefusals` call. |
| ARCH-PURPOSE | pass | Every M2 item in the Spec is wired end to end. The unrecorded live proof is filed under its own family above. |
| ARCH-MOCK | pass | The socket call is injected as `messageCall`, and the watcher runs on the fake runtime. |
| ARCH-CONSTRAINTS | flag | The log's read cost grows with every peer message (finding above). |
| ARCH-SECURE | flag | The `refusals` response carries the nonce (finding above). |
| ARCH-ORDER | pass with note | The race between an nvim history rewrite and a peer append is acknowledged as a residual in the Log. Workers are joined via `r.workers`. Tests sample a single interleaving. |
| ARCH-FUNERAL | flag | The Pair log writer's growth is unstated (finding above). |

### 7. Plan revision recommendations
- A Revisions entry for the Pair-log growth bound, or the cap chosen.
- A Revisions entry or Log observation for Task 8 / Done-when Live.
- BR-9 is still open: either implement "reported once per failure streak" or remove that phrase from the ListingPoller row with a Revisions entry.

```findings
dispose:
  - id: BR-7
    disposition: not-addressed
    note: |
      Still present at head: broker.go:523 starts a fresh AdmissionTimeout after the sender authority.current check at message_service.go:851; this range does not touch it.
  - id: BR-8
    disposition: not-addressed
    note: |
      Still present at head: transport.go serve still classifies the request via requestBudget plus requestIsRecoveryPreview twice; Call still marshals twice.
  - id: BR-9
    disposition: not-addressed
    note: |
      zellij_listing_poller.go Refresh still only sets p.failing; the plan Core concepts still promise once-per-streak reporting, and no Revisions entry removes it.
findings:
  - id: new
    severity: Important
    kind: defect
    family: unauthenticated-op-exposes-credentials
    title: |
      The unauthenticated refusals op returns full sender Bindings (session and nonce) and message bodies
    detail: |
      message_service.go:815-820 returns broker.LatestRefusals(); each Receipt.Message.From is the sender's Binding (refusal.go:53-54), which carries the Scope/Tag/Session/Nonce tuple Broker.Caller authenticates with (broker.go:357-365), plus Body. couch --list needs only Detail. Fix: return a Detail-only projection and add a test that the response holds no binding or body (ARCH-SECURE).
  - id: new
    severity: Important
    kind: defect
    family: writer-growth-unbounded
    title: |
      Every submitted peer delivery now appends up to 8 KB to the Pair log, with no stated bound or removal
    detail: |
      peer_turn_log.go adds one log entry per Submitted delivery (MaxBodyBytes 8192); hub slots take dozens per hour. Each append re-parses the whole log under its lock (store.go:85,136), and the nvim status line re-parses it through read_history (init.lua:2377,2435). The plan has no ARCH-FUNERAL line for this writer. State the growth bound in Revisions, or cap retained couch-peer- entries (ARCH-FUNERAL/ARCH-CONSTRAINTS).
  - id: new
    severity: Important
    kind: defect
    family: unmeasured-operating-claim
    title: |
      M2 is ticked without the Task 8 / Done-when live observation (rebooted and resumed codex slots receive; relaunch after peer messages only)
    detail: |
      2nd finding in this family. Rule: every Live step in the Plan or Done-when has a dated Log observation before its milestone ticks, or a Revisions entry that defers or drops it. Check both milestones' Live steps against that rule, not just this one.
  - id: new
    severity: Minor
    kind: defect
    family: lookup-key-not-canonicalized
    title: |
      LatestRefusals keys by the raw route target, so alias- or family-spelled refusals never match a --list row
    detail: |
      refusal.go:87 keys by route.Target before canonicalTarget runs; supersession compares that raw key with the resolved To.Slot (refusal.go:96); disambiguated duplicate labels also miss (run.go renderThreadRows lookup).
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ff7adff3-88eb-4d11-8642-409f3ebf6b59","repository":"github.com/xianxu/pair","issue_id":"000618","recipe":"milestone-review","round":4,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/775d8f9444af31a2","reviewed_head":"d6a2652056b9135969d642f9d70b29487a4ae517","slot":"pair:2","requested_at":"2026-10-17T02:34:53.770676Z"},"phase":"finished","pid":55149,"started_at":"2026-10-17T02:34:53.971944Z","finished_at":"2026-10-17T02:40:30.312892Z","session_id":"d8d3fb12-a1e6-4387-9e3f-63a05c656ace","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.776883958s
head: "d6a2652056b9135969d642f9d70b29487a4ae517"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
