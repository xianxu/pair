# Boundary Review — pair#399 (milestone M1)

| field | value |
|-------|-------|
| issue | 399 — couch: model orphaned zellij servers (alive, socket gone) and reap before resume |
| repo | pair |
| issue file | workshop/issues/000399-couch-model-orphaned-zellij-servers-alive-socket-gone-and-reap-before-resume.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f2b393be9457e4a40ab57532dc4d621a18cb344a..53f97e2916d9c4cc458f19ca7e3ba50a4fd020a6 |
| command | sdlc milestone-close --issue 399 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-06T22:27:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 does what it promises. A zellij server whose socket path is gone is now detected from the server's own command line (one `ps` per refresh plus one `Lstat` per server). It becomes `SessionOwnerOrphaned` in the probe, `SessionOrphaned` in session presence and `unusable/orphaned-server` in the thread classifier. Resume refuses it with a coded diagnostic, startup refuses to start a second primary beside it, and the recovery report holds it as `orphaned-server` with no steps. The design fails closed: a socket counts as gone only on ENOENT, an unreadable socket or two servers for one name read as unresolved, and a failed server snapshot fails the whole refresh. I ran the build, vet and the targeted launcher/couchcore/couchtty tests with a scrubbed environment and a scratch `TMPDIR`; all passed. I did not run the full `couchcore` suite, which takes about 500 s. One cheap fix should land before the boundary: the startup refusal tells the operator that killing the server takes its agent with it. The issue's own Problem section shows that is false, because the children ignored SIGTERM and were left behind with PPID 1.

**1. Strengths**
- `launcher/session_servers.go:23`: there is now one command-line parser and one server struct. The lowercase `sessionServerIdentity` copy is deleted and `zellijServerPIDs` delegates to the shared parser (ARCH-DRY). A fuzz target checks the parser's output against the exact-command rule.
- `ObserveSocket` (`session_servers.go:72`) treats only ENOENT as gone; every other answer is unknown, so an unreadable socket directory cannot turn live servers into orphans. `TestObserveSocketTrustsOnlyENOENTAsGone` pins this.
- `Probe` checks the socket before calling `list-panes` (`session_owner_os.go:65`). The test covers socket {present, gone, unknown} × list-panes {ok, error} and asserts an orphan is never asked for its panes.
- `orphanOf` (`actionableinventory.go:627`) derives the row's `Orphan` field from the classifier's reason, so the field and the reason cannot disagree.
- Startup was checked for a second-primary risk: unusable rows used to count as debris. The fix comes with a test that failed first (`couch_test.go:1930`) and asserts the refusal has no side effects.

**2. Critical findings**
None.

**3. Important findings**
- **`couchcore/couch.go:500`, startup refusal text.** It says `kill %d (the orphaned server; its agent goes with it)`. On 2026-10-06 that was not true: `pair wrap` needed SIGKILL and `pair title` helpers were left with PPID 1. Until M2's reap exists, operators following this advice will leave the children running. Fix: tell the operator to stop the server's whole process tree (e.g. `pkill -TERM -P %d; kill %d`, then SIGKILL anything that survives), or drop the "goes with it" claim and point to the reap that M2 will ship.

**4. Minor findings**
- `couchcore/launch_existing.go:466`, `sessionOwnerWord`: has no case for `SessionOwnerOrphaned`, so diagnostics print "unknown" (ARCH-PURPOSE: this is one of the places that lists the owner-state vocabulary).
- `couchcore/slotobserve.go:18`, `AgentRunning`: `AgentOrphaned` falls through to (false, false), i.e. not known. That is safe, but an orphan is a running agent known to be running; it should return (true, true).
- `couchcore/startup.go:124`, `ScopeHoldsOrphanedThread`: also requires `row.Orphan != nil`. A row with `ReasonOrphanedServer` but a nil `Orphan` silently becomes debris again. Match on the reason alone and handle a nil server in the message.
- `ClassifyServers` + `ProjectSessionPresence`: if a new live server starts under the name of an orphan, the name has two servers and reads unresolved. Because the live check runs first, the projection then reports `SessionPresent` and the orphan is hidden. This is an edge case; record it as a known limit.
- The ordering of `AgentOrphaned` against `AgentUnusableUnknown` is written in two places: the rank map in `slotreport.go:35` and the loop in `recoverplan.go:1199`, where the last row seen wins (ARCH-DRY). Share one ranking.
- `recoverplan.go:1208`: `f.agent.Orphan` is set from any orphaned row, even when the slot's agent state ends up `busy`.

**5. Test coverage notes**
The pure layers are well covered: the parser (unit test and fuzz), `ObserveSocket`, `ClassifyServers`, the probe table, the presence projection matrix, the classifier shapes, the archive/reboot checks over every classification, the recovery-report case, the resume advice table, the startup refusal and the switcher's notice. `OSServerStates` has no test of its own, which is acceptable because M3's live acceptance covers it. The plan's Task 5 asked for a startup test asserting no "exit status" text; the refusal test checks the diagnostic sentence instead, which is enough.

**6. Architectural notes**
- ARCH-DRY: pass (one minor note above).
- ARCH-PURE: pass. The parse, socket reading and classification rules are pure, and the IO is the thin `OSServerStates` / `osSessionOwnerIO.Socket`.
- ARCH-PURPOSE: pass for M1's scope; two places that list the vocabulary missed the new value (minor notes above).
- ARCH-MOCK: pass. `ServerStates` and `SessionOwnerIO.Socket` are injected interfaces, and production and tests use the same boundary.
- ARCH-CONSTRAINTS: pass. One bulk `ps` per refresh, `Lstat` per server, identity read only for orphans.
- ARCH-SECURE: pass. ENOENT-only, the exact command-line match, and failure paths that fail closed.
- ARCH-ORDER: pass. No new state is carried between events; the new value enters existing enumerations.
- ARCH-FUNERAL: pass. Nothing durable is created.

For M2, the reaper should take the process-tree snapshot while the parent chain is still intact, and re-read each process's identity before signalling it, as the plan says. Once reap exists, the startup message should point to it.

**7. Plan revision recommendations**
- Task 3 named `artifactcollision_fake.go SetOrphanedServer`; the implementation reuses `SetSessionPresence`. Note this in `## Revisions`.
- Task 5 planned the startup refusal inside `startupResumeRefusal`; it actually lives in `spawnResolved` via `ScopeHoldsOrphanedThread`. Note the move.

```findings
findings:
  - id: new
    severity: Important
    family: operator-advice-contradicts-evidence
    title: |
      Startup orphan refusal says "kill PID; its agent goes with it", which the issue shows is false
    detail: |
      couch.go:500 tells the operator to kill only the server. On 2026-10-06 wrap ignored SIGTERM and title helpers were left with PPID 1. Tell them to stop the whole process tree (or point to reap) instead.
  - id: new
    severity: Minor
    family: vocabulary-consumer-missing-member
    title: |
      sessionOwnerWord and AgentRunning do not handle the new orphaned values
    detail: |
      launch_existing.go:466 prints "unknown" for SessionOwnerOrphaned; slotobserve.go:18 returns not-known for AgentOrphaned, though an orphan is known to be running.
  - id: new
    severity: Minor
    family: derived-field-guard-redundant
    title: |
      ScopeHoldsOrphanedThread also requires Orphan != nil, so a nil-server orphan row reads as debris
  - id: new
    severity: Minor
    family: single-ranking-source
    title: |
      Ranking of orphaned vs unusable-unknown agent evidence is written in two places (slotreport rank map, recoverplan loop)
  - id: new
    severity: Minor
    family: orphan-shadowed-by-live-same-name
    title: |
      An orphan plus a new live server for the same name reads Present, hiding the orphan
```

---

## Re-review — 2026-10-06T22:40:36-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 399 — couch: model orphaned zellij servers (alive, socket gone) and reap before resume |
| repo | pair |
| issue file | workshop/issues/000399-couch-model-orphaned-zellij-servers-alive-socket-gone-and-reap-before-resume.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f2b393be9457e4a40ab57532dc4d621a18cb344a..c9fb4b5c40d73946e079457e2f6db3c7dda5c3cd |
| command | sdlc milestone-close --issue 399 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-06T22:40:36-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 does what its Plan row says. The server argv now carries the socket path. `ClassifyServers` trusts only ENOENT as "gone". `Probe` returns `SessionOwnerOrphaned` instead of a raw zellij exit status. Session presence gains `SessionOrphaned`, and the thread reads `unusable/orphaned-server`. Resume, startup, the switcher and the recovery report all name the orphan, and all of them fail closed. Five of the six prior findings are fixed and have tests. BR-2 is not fixed, and it blocks SHIP. The new startup refusal still gives the operator commands that would not have worked on 2026-10-06. Its `pkill -P` targets only the server's direct children. The SIGKILL follow-up runs after `kill <server>`, so any child that ignored SIGTERM has already moved to PID 1, and the follow-up matches nothing. The test only checks that certain strings are present, not that the steps work. Targeted tests pass at HEAD c9fb4b5c: launcher `-run 'Server|Orphan|SessionOwner|Socket'` and couchcore `-run 'Orphan|Contested|Vocabulary|AgentRanking|SessionPresence|Recover'`. I did not run the full couchcore suite (~500 s).

**Strengths**
- `launcher/session_servers.go:73` (`ObserveSocket`): an unreadable socket directory is `SocketUnknown`, never `Gone`. This prevents the worst false positive, where every live server reads as an orphan.
- `artifactcollision.go:495`: if the server snapshot fails, `SessionPresence` fails closed instead of reading the session as absent, which would offer a resume onto a running agent.
- `actionableinventory.go:540`: the orphan check outranks the record's own faults and every "no session" reading. `orphanOf` derives the row's `Orphan` field from the reason, so the two cannot disagree.
- Startup now refuses to start a second primary beside an orphaned one. This was found during design and written test-first.
- `quiesceZellijSession` and `zellijServerPIDs` now go through the one argv parser, `ParseServerProcesses`, which has a fuzz target (ARCH-DRY).

**Critical:** none.

**Important**
- BR-2 is still open; see the disposal below. This is the 2nd finding in family `operator-advice-contradicts-evidence`, so the fix should be a rule, not another wording change. Rule: operator advice must be steps that would have worked on the documented incident, and its test must check the order of the steps, not that words appear. Until M2's `PlanReap` can produce the steps itself, the refusal at `couch.go:1281` should:
  1. List the process tree first, e.g. `ps -axo pid,ppid,command`.
  2. Send SIGKILL to the descendants while the server is still their parent.
  3. Only then kill the server.
  
  Alternatively, it can say plainly that reaping arrives in M2 and print the tree-listing command. When M2 lands, make the refusal render `PlanReap`'s steps so the text and the behaviour come from one place.

**Minor**
- `ServerState` in `session_servers.go:86` is three booleans (`Orphaned`, `Unresolved`, `Contested`). The flag combinations allow states that should not exist, such as Orphaned and Unresolved together, and Contested implies Unresolved without saying so. A tagged enum (`Reachable | Orphaned | UnknownSocket | Contested`) would rule those out (ARCH-ORDER/ARCH-SECURE).
- A zellij server that is still starting may show up in `ps` before it has created its socket. For that moment it reads as orphaned. In M1 this only causes a refusal. In M2 a reap would kill a session that is just starting. M2 should treat one observation as provisional: require the condition across two snapshots or a minimum process age, and check the identity again before sending any signal (ARCH-ORDER).
- `TestOneAgentRankingForBothReports` checks the rank map but not the many-thread loop in `slotEvidenceOf` (e.g. orphan + unknown rows giving the orphaned class).

**Test coverage notes:** the Done-when items M1 claims have tests: socket absent → orphaned (never an error, never parked); contested names; vocabulary consumers; startup refusal. The refusal test should check what the steps do, not which strings appear.

**Architecture**
- ARCH-DRY: pass. One parser and one `agentRank`.
- ARCH-PURE: pass. `ClassifyServers`, `ObserveSocket` and `ProjectSessionPresence` are pure; `OSServerStates` is a thin IO layer.
- ARCH-PURPOSE: pass. All M1 consumers swept, and the switch statements I checked fail closed by default.
- ARCH-MOCK: pass. The `Servers` seam has a fake, and live acceptance is planned for M3.
- ARCH-CONSTRAINTS: pass. One `ps` plus one Lstat per server on each refresh.
- ARCH-SECURE: flagged (Minor) for the boolean constellation above. `ps` input is parsed defensively.
- ARCH-ORDER: flagged (Minor) for the provisional-observation issue above.
- ARCH-FUNERAL: pass. M1 creates nothing durable; the new orphan fields are in-memory and in the JSON report.

**Plan revisions:** add a `## Revisions` note for M2: before reap acts, an orphan verdict needs either two observations or a minimum process age.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      FuzzParseServerProcesses with a one-line Strategy comment, session_servers_test.go:28-46.
  - id: BR-2
    disposition: not-addressed
    note: |
      couch.go:1281 pkill -P reaches only direct children, and the KILL follow-up runs after kill <server>, so SIGTERM-ignoring children have already moved to PID 1 and it matches nothing. Same failure as 2026-10-06; the test checks strings only.
  - id: BR-3
    disposition: addressed
    note: |
      sessionOwnerWord and AgentRunning handle the orphan (TestVocabularyConsumersKnowTheOrphan); other SessionOwner switches fail closed by default.
  - id: BR-4
    disposition: addressed
    note: |
      startup.go:125 keys on the reason only; TestAnOrphanRowWithoutItsServerStillBlocksStartup.
  - id: BR-5
    disposition: addressed
    note: |
      One agentRank at recoverplan.go:1481 used by slotreport and slotEvidenceOf.
  - id: BR-6
    disposition: addressed
    note: |
      Contested outranks list-sessions live (sessionevidence.go:121, TestAContestedServerNameIsNeverPresent); Probe treats len!=1 as ambiguous.
findings:
  - id: new
    severity: Minor
    family: boolean-constellation-not-enum
    title: |
      ServerState is three booleans (Orphaned/Unresolved/Contested) with unwritten legal combinations
    detail: |
      Contested implies Unresolved and Orphaned+Unresolved can be represented; a tagged enum (Reachable, Orphaned, UnknownSocket, Contested) would rule the bad combinations out (ARCH-ORDER/ARCH-SECURE).
  - id: new
    severity: Minor
    family: point-observation-as-settled-state
    title: |
      A server still starting (in ps, socket not yet bound) reads as orphaned on a single snapshot
    detail: |
      Harmless in M1, where it only causes a refusal; M2 reap must require two observations or a minimum process age and re-check the identity before signalling, or it can kill a session that is starting (ARCH-ORDER).
```

---

## Re-review — 2026-10-06T23:00:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 399 — couch: model orphaned zellij servers (alive, socket gone) and reap before resume |
| repo | pair |
| issue file | workshop/issues/000399-couch-model-orphaned-zellij-servers-alive-socket-gone-and-reap-before-resume.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f2b393be9457e4a40ab57532dc4d621a18cb344a..2ddd480ee7767ce95f563bb21791eaf3d0f3abc1 |
| command | sdlc milestone-close --issue 399 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-06T23:00:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three open findings are fixed, and I found no new Critical or Important issues. Round 2 changed two things. The startup refusal now gives working steps in the right order: list the server's process tree while the server is still their parent, kill the descendants, then kill the server. A test checks that order and fails if the old wrong advice comes back. `ServerState`'s three booleans are now one tagged `ServerVerdict` (Unresolved / Reachable / Orphaned / Contested), so contradictory combinations can't be built. BR-8's single-snapshot risk can't do harm in M1, which only refuses, and it is now a written M2 requirement in the plan's `## Revisions`. The build and vet are clean. In the sandbox, the targeted orphan/server/presence/recover-plan tests pass in `couchcore` and `launcher`. The full run had 5 failures, all `ptychild: operation not permitted` or `mkdir /tmp/...: operation not permitted`. That is the sandbox blocking PTY child processes and writes to `/tmp`, not a fault in this diff.

1. **Strengths**
   - `launcher/session_servers.go`: `ServerVerdict`'s zero value is `ServerUnresolved`, so any path that misses a case falls back to "can't tell" rather than "orphaned". The `OSServerStates` identity-read failure also falls back to `ServerUnresolved`.
   - `sessionevidence.go` `hasServer`: it checks that the map has an entry before comparing, so a name with no server row never matches the zero verdict.
   - `startup_orphan_test.go`: the test checks the steps' order with string positions, not just that certain words appear. That follows the new rule in `workshop/lessons.md` ("operator advice is code").
   - `recoverplan_test.go` `TestManyThreadsWithAnOrphanReadOrphaned`: it pins the single `agentRank` rule (an orphan outranks an unknown row) for the many-thread case.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `startup_orphan_test.go`: the second guard (`first := strings.Index(text, "9090"); ...`) is hard to follow, and the earlier position check already covers it. Consider deleting it.
   - `couch.go`: the doc comment on `orphanStartRefusal` still says "names the whole tree"; it could mention the order the steps must follow.

5. **Test coverage:** the verdict table test, presence projection (including the contested name beside a live one), the refusal order and the many-thread ranking are all covered. M2 still needs the planned fixture showing a just-started server is never reaped.

6. **Architecture:**
   - ARCH-DRY: pass. One ranking source (BR-5), and the plan schedules the advice to come from `PlanReap` in M2.
   - ARCH-PURE: pass. `ClassifyServers` and `ProjectSessionPresence` are pure; `OSServerStates` is the thin layer that touches the OS.
   - ARCH-PURPOSE: pass for M1's scope (observe and name); reap is M2.
   - ARCH-MOCK: pass. A `fakeServerStates` stands in behind the same `Servers` seam that production uses.
   - ARCH-CONSTRAINTS: pass. One `ps` snapshot per check.
   - ARCH-SECURE: pass. `ps` output is parsed into typed values, and anything ambiguous becomes Unresolved.
   - ARCH-ORDER: pass after BR-7; BR-8 is carried to M2.
   - ARCH-FUNERAL: pass. M1 creates nothing durable: it only reads `ps` and the sockets and refuses.

7. **Plan revisions:** none needed. The 2026-10-07 revision already carries BR-8 and the single-source advice into M2.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      couch.go orphanStartRefusal now lists the tree, kills descendants, then the server; TestOrphanRefusalStepsWouldHaveWorkedOnTheIncident fails if the order changes or the old advice returns.
  - id: BR-7
    disposition: addressed
    note: |
      ServerState carries one ServerVerdict enum (zero value Unresolved); every consumer uses hasServer or a verdict comparison; the table test pins all four verdicts.
  - id: BR-8
    disposition: addressed
    note: |
      Harmless in M1 (it only refuses); the plan's 2026-10-07 Revisions requires two snapshots or a minimum process age, an identity re-read before each signal, and a just-started-server fixture in M2.
```
