# Boundary Review — 000115#115 (issue close)

| field | value |
|-------|-------|
| boundary | issue close |
| window | 4593c8c..HEAD |
| command | sdlc judge milestone-review --issue 115 --base 4593c8c --head HEAD |
| reviewer | claude |
| verdict | REWORK |

```verdict
verdict: REWORK
confidence: high

## 1. Strengths

- **Pure journal + recovery planner.** `handoff_state.go:203` (`Advance`) and `handoff_state.go:359` (`PlanRecovery`) keep every ordering decision deterministic and IO-free, with `handoff_state_test.go` covering each state boundary, snapshot variants, and both rollback/forward tails. This is the right seam and it is honestly tested.
- **Durability primitives are not hand-waved.** `writeExclusive` (`handoff_store.go:475`) publishes a fully written, fsynced inode by no-replace link so a losing racer can never read partial JSON; `retainInput`/`inputMatchesSnapshot` (`handoff_store.go:531,561`) fixed the M3 hard-link aliasing bug properly (immutable copy + size/digest), and `TestOSHandOffStoreRejectsSameInodeDraftMutationAndKeepsImmutableBackup` pins it.
- **Queue push-front consolidated.** `queuecmd.PlanPushFront`/`PushFrontPlanned` (`cmd/internal/queuecmd/queue.go`) is now the single allocator for both Neovim and handoff storage, with real concurrency coverage (`queue_test.go:57`) — a clean ARCH-DRY win over the previous Go/Lua duplication.
- **Shared readiness contract.** `cmd/internal/readiness/record.go` is one validated wire schema consumed by both producer (`wrapcmd`) and consumer (`launcher`), with a golden round-trip test on each side — exactly what the code-entry gate asked for.
- **Post-merge layout work is sound.** `runHandoff` resolving through `resolveLiveLayout` rather than the record alone (`createflow.go:381`) and holding the tag lock across the destructive layout-conflict branch (`createflow.go:224-232`) are correct, and `TestRunHandoffLaunchesTheTagsResolvedLayout` is a real mutation-proof regression.

## 2. Critical findings

**C1. The source-quiescence acknowledgement has no production writer; every live handoff destroys the source and then times out.** `lifecycle.go:54`

`runCleanup` returns at `lifecycle.go:54` unless `TakeQuitMarker` finds `~/.cache/pair/quit-<session>`, which is written only by Alt+x / Alt+n (`restart.go:35,48`, `compaction.go:78`). The handoff stops the source with `zellij delete-session --force` (`handoff_process_os.go:56`), which writes no quit marker — so the handoff-marker branch at `lifecycle.go:66`, the *only* caller of `WriteHandoffCleanupAck`, is unreachable from the handoff flow. `ObserveJournaledSource` (`handoff_process_os.go:70`) then returns `partially-stopping` forever, because both of its progress branches require `ackPresent`.

Failure scenario (reproduced): neutralize only the fake's `ack_handoff_cleanup` in `tests/pair-agent-handoff-test.sh` and the same test fails after 16.4s with
`pair: handoff failed for 'work': timed out waiting for source session "pair-work-work" to become quiescent: partially-stopping` — source session already deleted, no target, journal left at `source-stop-requested`, `handoff-<tag>.lock` and `handoff-marker-<tag>` retained by design.

Note that reordering `runCleanup` is *not* a complete fix: a **detached** source (which the picker offers) has no launcher process at all, so no process exists to acknowledge. Quiescence needs evidence the coordinator can observe itself — e.g. accept "zellij row absent **and** no recorded pair-wrap/agent/nvim PID alive" as quiescent when no launcher can be expected to ack, and keep the ack as a fast path. Whatever the shape, the fake must stop supplying it (see I5).

**C2. A different-driver row with no live session routes to handoff, stops `pair-<tag>` instead of the tag's indexed session, and aborts on real zellij.** `pick.go:130-145`, `handoff.go:48`

`resolveAgentPickSelection` returns `ActionHandoff` for any row whose driver differs from the requested agent, regardless of `work.State` — including `SessionInactive` and `SessionExited`. `runHandoff` then unconditionally `SetHandoffMarker` + `StopJournaledLaunch`. Two consequences: (a) `pick.go:138` falls back to `sessionName(work.Tag)` = `pair-<tag>`, bypassing the session-name index, so the handoff targets a *different public session identity* than the tag's canonical one (spec: "the same canonical repo-scoped public session identity"); (b) `zellij delete-session <missing> --force` exits 2 (verified against the installed zellij), so `StopJournaledLaunch` errors out.

Reproduced end to end (fake `zellij` patched to mimic real exit 2, tag with no live session):
```
picker row : work/work  codex  (inactive)   [⏎ 2 queued]
pair: handoff failed for 'work': delete zellij session "pair-work": exit status 2
```
Fix sketch: for a non-live row, skip the stop/quiescence phase entirely (there is nothing to stop — go straight to snapshot + target launch), and resolve the public session through the session-name index (`sessionNameForTag`/`AssignSessionName`) rather than `sessionName(tag)`.

**C3. A leaked recovery claim permanently disables the tag.** `handoff_store.go:85`, `handoff.go:220`

`AcquireTagLock` refuses on the mere *existence* of `handoff-<tag>.recovery.lock` without checking owner liveness, and it returns before any `Stale` result — so `ClaimStaleTagLock`'s dead-owner quarantine (`handoff_store.go:117`) is unreachable. `RecoverTag` returns at `handoff.go:220-221` on any `applyRecoveryPlan` error without `ReleaseRecoveryClaim`, so any failed or crashed recovery leaks the claim.

Reproduced (continuation of C1/C2): entry 2 fails recovering, entry 3 and every entry after it print
`pair: failed to acquire tag lock for 'work': handoff recovery claim is active for tag "work"`
for `pair`, `pair <agent>`, and `pair resume <tag>` alike. Only `rm` of the claim file restores the tag. Fix sketch: gate the refusal on `procutil.Alive(claim.Claim.OwnerPID)` and fall through to the stale path otherwise; release/quarantine the claim on the recovery error path. (This is what the uncommitted working-tree edit does.)

**C4. `queue_push_front` uses `$PAIR_HOME/bin/pair`, which does not exist in embedded-runtime installs.** `nvim/init.lua:701-705`

The runtime bundle never contains `bin/pair` (`runtimebundle/embed_test.go:34`; `tests/pair-embedded-runtime-test.sh:56` asserts `test ! -e "$root/bin/pair"`), and `PAIR_HOME` is exported as exactly that extracted root. The launcher already fronts the running binary's dir on the pane PATH (`createflow.go:42`), which is why every other call site uses bare `'pair'` (`init.lua:3227,3282,3330,3359,3629`).

Reproduced: with a bundle-shaped `PAIR_HOME` (valid asset root, no `bin/pair`), `bash tests/queue-send-test.sh` fails `send +3 w/ draft` — the draft is never pushed to the front. The new `make test-queue` cannot see this because it pins `PAIR_HOME=$(CURDIR)` (`Makefile.local:172`). Fix sketch: invoke bare `pair` (or keep the existing `(home ~= '') and (home..'/bin/pair') or 'pair'` fallback used at `init.lua:959,1008`), and add a bundle-shaped-`PAIR_HOME` case to `test-queue`.

## 3. Important findings

- **I1. The handoff never writes `config-<tag>-<agent>.json`.** `createflow.go:887` writes it on create; `planHandoffTargetLaunch` has no equivalent, and `sessionwatch.SpecForAgent` (`sessionwatch.go:39`) covers only codex and agy. So a handoff **to claude** records its minted `--session-id` only in the ledger, while `readSavedConfigCandidate` (`createflow.go:990`) prefers the config file and only falls back to the ledger when the read *fails*. A stale `config-<tag>-claude.json` therefore permanently shadows it: every subsequent launch resumes/mints from the stale id and orphans the conversation the handoff just created — breaking the spec's "returning to a previous driver resumes that agent's prior native conversation". Fix: call `writeConfig` for the target when `sessionID != ""`, as create does.
- **I2. Selecting a refused picker row exits 0 in silence.** `createflow.go:303-306`. A conflict / unknown-live-driver row is rendered but `Selectable: false`; `resolveAgentPickSelection` returns `ok=false`, which `resolveLaunchPick` maps to `aborted`, indistinguishable from ESC. The spec requires Pair to "refuse the handoff, list every conflicting session/evidence source, and ask the user to resolve it". The row label also lists evidence *kinds* only (`"conflict: live-session, agent-file"`, `pick.go:99`), not the conflicting session names. Fix: return a distinct refusal with the concrete evidence rendered by `formatDriverEvidence` (which already exists, `handoff_preflight.go:70`).
- **I3. `GatherDriverSnapshot` derives the scope key with `filepath.Base(r.DataDir)`.** `handoff_driver_os.go:36`. The canonical helper is `scopeKeyFromDataDir` (`datadir.go:21`), used by `runcli.go:68` and by `handoffPaths` two files away (ARCH-DRY). They agree only when `DataDir` is literally `repos/<key>`; with `PAIR_DATA_DIR` set (`runcli.go:46`, an explicitly supported override) `Base` yields a key matching nothing, so **all** live-session and session-index evidence is dropped and `ClassifyDriver` silently falls back to the ledger — disabling exactly the exclusivity-conflict detection the spec makes authoritative, and skipping the "selected session changed" guard (`handoff_preflight.go:53`, which only fires when `driver.Session != ""`).
- **I4. Docs gate: new durable surface missing from atlas.** `atlas/session-identity.md:29-41` was updated for `agent-ready-*`, `agent-default-*`, `handoff-*.lock`, the journal, and `parked/`, but omits `handoff-marker-<tag>`, `handoff-<tag>-<txn>.cleanup-ack`, `handoff-<tag>.recovery.lock`, and the transaction `backup/` subtree. The cleanup-ack is the gate for the entire quiescence contract (and the subject of C1) and is documented nowhere. `PAIR_CONFIRM_HANDOFF` (`osruntime.go:293`) and `PAIR_TEST_OUTSIDE_ZELLIJ` (`runcli.go`) are likewise undocumented, unlike their `PAIR_FORCE_IN_SESSION`/`PAIR_KILL_CMD` peers.
- **I5. ARCH-MOCK: the acceptance fake models behavior the real dependency does not have.** `tests/pair-agent-handoff-test.sh:32-47` has the fake `zellij delete-session` write the handoff cleanup ack — an effect that belongs to a *different `pair` process*, not to zellij — and it succeeds unconditionally where real zellij exits 2 for a missing session. Production flow and test flow therefore do not share the same boundary: the fake is what makes C1 and C2 invisible. Fix: the fake should model only zellij's own observable contract (including its exit codes); if the ack is a real handshake, drive it through the real `runCleanup`.
- **I6. Two divergent picker-row builders.** `buildPickRows` (`pick.go:~165`) vs `buildAgentPickRows` (`pick.go:56`) render the same underlying rows differently (ARCH-DRY). The explicit-agent picker drops the ANSI colouring, the `(Nd ago, no live session)` annotation, and the age entirely — `AgentPickWork.MTime` is carried for it and never used (observed row: `work/work  codex  (inactive)   [⏎ 2 queued]`). README's picker section (`README.md:255`) documents only the bare-`pair` shape, so `pair <agent>` now silently contradicts the docs.

## 4. Minor findings

- Transaction directories `handoff-<tag>-<txn>/` (with backups), `.cleanup-ack` files, and `.quarantine-<claimID>` files are never garbage-collected; `FindUnresolvedHandoff` re-globs and re-parses every one on each stale-lock recovery.
- `sourceStopIdentity` (`createflow.go:592`) hand-rolls `json.Unmarshal` into `ReadyRecord` instead of `sharedreadiness.Decode`, skipping the shared validation (ARCH-DRY).
- `tests/pair-agent-handoff-test.sh:123,137` writes to hardcoded `/tmp/pair-agent-handoff-*.{out,err}` despite the file header calling the test hermetic — it escapes its own temp root and collides between concurrent runs.
- A crash between the `source-stop-requested` journal write (`handoff.go:41`) and `SetHandoffMarker` (`handoff.go:45`) makes recovery classify an untouched live source as `partially-stopping` and stop it, contradicting the spec's "a bounded-stable intact source is left running".
- `resolveAgentPickSelection`'s `!policy.AgentExplicit` branch (`pick.go:141-146`) is dead — `resolveLaunchPick` delegates to `resolvePick` before reaching it.
- `validAgentIdentifier` (`agent_defaults.go:16`) is now the validator for tags and transaction ids too (`handoff_transcript.go:44`, `scoped_paths.go:45`); the name no longer describes its role.
- `runRenameScoped` uses `time.Now()` (`rename.go:180`) rather than the injected clock the rest of the launcher threads through `Env.Now`/`rt.Now()`.

## 5. Test coverage notes

- Ran and green: `go test ./... -count=1`; `go test ./cmd/internal/{launcher,queuecmd,wrapcmd} -race -count=1`; `git diff --check 4593c8c..HEAD`; `bash tests/pair-agent-handoff-test.sh` (PASS, 2.8s).
- PURE entities are genuinely IO-free and mock-free: `handoff_state_test.go`, `launch_args_policy_test.go`, `driver_test.go`, `pick_test.go`, `agent_defaults_test.go`, `queuecmd` planning. INTEGRATION behavior is injected through `Runtime`/`SessionLaunch`. No PURE→INTEGRATION promotion needed.
- The coverage gap is precisely the bug class shipped: **no test drives the real `runCleanup`→ack path**. The three tests that exercise `lifecycle.go:66` (`TestRunLaunchQuitCleanupHandoffMarker*`) pre-seed a quit marker that the handoff flow never produces, so they validate an unreachable branch. Likewise nothing covers a handoff whose source is not live, and nothing runs queue push-front under a non-checkout `PAIR_HOME`.
- Suggested additions with the fixes: (1) an acceptance case whose fake `zellij` writes no ack and returns 2 for a missing session; (2) a `resolveAgentPickSelection` case asserting the inactive/exited row's decision and its session name from the index; (3) `AcquireTagLock` with a dead claim owner; (4) `test-queue` under a bundle-shaped `PAIR_HOME`.

## 6. Architectural notes

- **ARCH-DRY — flag.** Three concrete duplications: the scope-key derivation (I3), the two picker-row builders (I6), and the readiness decode in `sourceStopIdentity` (Minor). The `pair queue push-front` consolidation and the shared `readiness` package are the counterweight and are done well.
- **ARCH-PURE — pass.** Journal transitions, recovery planning, driver classification, argument precedence, picker rows, and queue planning are all pure and unit-tested; zellij/fs/tty/process work sits behind `Runtime`. `HandoffCoordinator.Run` is long but is an effect interpreter over the pure plan, which is the intended shape.
- **ARCH-PURPOSE — flag.** The issue's purpose is a *working* same-work driver switch, and C1/C2 mean the delivered flow succeeds only under a fake that supplies an effect production cannot. The picker offers detached/inactive/exited rows — the motivating "provider is degraded, move yesterday's work" case — and those paths are the ones that fail. The infrastructure is right; the last consumer is not derived from it.
- **ARCH-MOCK — flag.** See I5. Also worth noting for the future: there is no conformance check comparing the fake `zellij` against the real binary, which is how the exit-2 divergence in C2 went unnoticed.

## 7. Plan revision recommendations

- **`## Revisions` — source quiescence contract.** The plan and spec assume the source launcher acknowledges handoff-mode cleanup; that is unreachable for a deleted session and impossible for a detached or historical source. Record the quiescence evidence the coordinator can actually observe on its own, and which source states are eligible for the stop phase at all.
- **`## Revisions` — non-live source handoff.** State explicitly that a handoff over a detached/exited/inactive row skips the stop/quiescence phase, and that the public session is always resolved through the session-name index, never `pair-<tag>`.
- **Task 16/17 wording.** The M4 tasks still read as if the process fake proves the live switch; given I5 it currently proves the switch *modulo* an effect the fake injects. Either add the missing coverage or say plainly what layer proves what — this is the same class of overstatement the M4 FIX-THEN-SHIP already flagged, and `workshop/lessons.md:1050` ("Match acceptance claims to the layer that proves them") is the lesson it should now satisfy.
- **M3 integration table (`plan.md:793`).** `HandoffStoreOps` is described as wrapping an "`O_EXCL` lock"; the delivered implementation is a fsynced temp inode published by no-replace `os.Link`. The Revisions already record the change; the table does not.
  [1;33m[!][0m Post-milestone code review (AGENTS.md §3): findings reported — review above
