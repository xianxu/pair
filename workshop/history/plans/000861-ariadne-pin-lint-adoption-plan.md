# Ariadne pin and lint adoption implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Bounded tasks may use subagents after the plan gate. Steps use checkbox syntax.

**Goal:** Land the normal Ariadne dependency required by #838 with all newly active lint rules qualified and source causes repaired.

**Architecture:** Keep one upstream lint catalog. Adapt Pair-only AST rules at its boundary, reuse the existing short socket allocator, and promote Pair's existing duplicate-JSON parser to a shared internal authority. Tests drive expiry after the event they test; production defaults and caller budget semantics stay intact.

**Tech stack:** Go, standard contexts/timers, existing testwait/testpair fakes, normal Go module dependency.

## Approved scope and alternatives

Hub dispatch a942ac93 assigns this separate main-based adoption before #838; no stacked branch. Required pin is at least landed ariadne#581 5a17f789619d3dcaafec62ae51b350833836a3e7. #838 production needs ChatSections, RetiredKeys, ChatItems, Legacy/Tags/Headings and ValidAnswerTarget; reverting to main's81c832d5 cannot work. A forked/copied sidedoc parser would duplicate authority and is rejected (ARCH-DRY).

Broad disabling of catalog rules is rejected. Read-only source qualification found51 observations:40 shrunk deadlines,5 socket roots,5 recursive JSON,1 tmp literal. Two JSON observations are proven role false positives. Authenticated hub f9c009c1 authorizes exactly two temporary per-site allows naming ariadne#590: the independent strictjson fuzz oracle and standalone example DOM materializer. Remove them when the upstream correction lands; do not extend these exceptions to any other site.

No new product state, persisted format or operator action is added. No live fleet/store access. Seeded local-gates workflow regeneration remains its own operator-verified issue and is deferred by normal weave; do not include it. If generated vocabulary changes emerge during ordinary adoption, retain them, audit their origin and coordinate their owning refresh rather than discarding them.

## Core concepts

| Name | Kind | Lives in | Status |
|---|---|---|---|
| Shared JSON structural scan, Options | PURE | internal/jsonstrict/scan.go | modified ownership: move existing optimized parser from cmd/internal/strictjson/decode.go |
| AST-only Match adapter | PURE | internal/repolint/local_catalog_test.go | new; preserves existing local matchers |
| Cue-owned deadline context | INTEGRATION | cmd/internal/testwait/deadline.go | new; test helper, explicit expiration plus parent propagation |
| Virtual watcher runtime | INTEGRATION (test clock/runtime) | cmd/internal/sessionwatch/*_test.go | modified tests; reuse existing Now/Sleep/onSleep |

JSON Options explicitly selects exact versus Unicode-simple-fold key equality, preserved versus legacy float-range number acceptance, maximum depth, and an optional all-depth key visitor. Defaults preserve existing cmd/internal/strictjson behavior. Historyarchive requests folded keys, UseNumber and depth64. Submission keeps exact duplicate policy and its sensitive-key visitor, including opaque nested extension keys. Debt keeps its established exact policy. Thin owner wrappers retain typed strictness, required/null checks and raw unknown-field preservation (ARCH-EVOLVE); do not change serialized meanings or introduce a new version.

The root internal location is needed because internal/repolint cannot import cmd/internal/strictjson. Ariadne's suggested cmd/sdlc/internal/jsonstrict is inaccessible downstream; no public upstream package exists at this pin. Move the current optimized parser rather than creating another recursive decoder. The independent legacy oracle stays independent.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Normal upstream source snapshot | go.mod, go.sum, tests/affected-config-test.py | modified | actual provider packages and module metadata |
| Provision request deadline | cmd/internal/couchcore/provision_io.go | modified | context.WithTimeout via per-instance factory |
| VM request deadline | cmd/internal/vmcmd/runtime.go | modified | private run variant with deadline factory; exported Run uses normal factory |
| Agent fixture request deadline | cmd/internal/testcouch/agentfixture/run.go | modified | private run variant with factory; normal CLI uses real deadlines |
| Tunnel startup timer | cmd/internal/broadcast/cloudflared.go | modified | internal timer channel factory, normal default unchanged |
| Seat grace timer | cmd/internal/webcast/drive/controller_mem.go | modified | internal AfterFunc factory and Stop-capable timer; normal default time.AfterFunc |
| PTY capture waits | cmd/internal/wrapcmd/harness_tty_live_test.go | modified | existing test-only capture seam, real child/PTY Wait and reader join |
| Short socket root | cmd/internal/testpair/shortroot.go | reused unchanged | private canonical bounded directory with immediate cleanup |

Factories are request/instance-local dependencies, never package globals, launch flags, serialized state or new retry engines. Cue contexts must propagate parent cancellation and preserve context.Cause/Err; a request-timeout test injects its own deadline factory rather than expiring the caller and accidentally testing the wrong branch (ARCH-PURE).

## Constraints, costs and lifetime

JSON validation performs json.Valid once plus one byte pass, one decoded-key operation per key and one callback per key. Memory is live nesting plus key sets. Count bytes/key visits at N/2N; reject quadratic rescans. Include escaped/Unicode keys and nested arrays, not only flat ASCII. Fold keys with encoding/json-equivalent Unicode simple-fold equality; preserve current per-owner number/depth acceptance, including scalar-child depth.

Deadline creation/expiration/parent propagation is O(1) per request, no polling thread per cue. Each child test waits for its exact readiness marker/stage before expiration and joins its actual command/reader. Context watchers/timers cancel in cleanup; test roots remove at t.Cleanup; subprocesses remain within existing exact group owners. No durable store/cache family is added. Normal logs and issue evidence follow existing workshop archive lifecycle (ARCH-LIFECYCLE).

## Function-scoped test strategies

The cue API is `testwait.DeadlineOnCue(parent context.Context, budget time.Duration) (context.Context, func(), context.CancelFunc)`: the second result expires the request with DeadlineExceeded; the third cancels and releases the parent notification. Factory tests capture the selected duration before returning this context. It is not a replacement parent deadline for own-request-budget tests.

| Risky function / seam | Adversarial strategy and mechanical guard |
|---|---|
| internal/jsonstrict.Scan(raw, Options) and its private scan work-counter seam | Independent legacy-token differential fuzz oracle plus owner-specific option cross-product; assert same accept/refuse and error authority for exact/folded Unicode/escaped keys, number-range policy, trailing bytes and depth boundaries. Count byte/key visits at N/2N and visitor invocations; reject prefix rescans or missed nested keys. |
| strictjson.Decode, RejectDuplicateKeys, RejectDuplicateKeysPreserveNumbers, SplitObject and EncodeObject | Existing wrapper tests plus independent raw round-trip assertions: preserve defaults, required/null and unknown raw members across scanner extraction, including huge retained numbers and escaped duplicate keys. |
| historyarchive.decodeJSONNoDuplicateKeys, DecodeUploadReceipt and label/receipt JSON codecs | Existing typed fixture oracle against folded Unicode aliases, strict/non-strict unknown members, depth64/65 and retained raw extensions; assert old accepted bytes remain accepted and ambiguous owned fields refuse before typed adoption. |
| submission.checkJSONKeys and DecodeManifest | Nested unknown-extension adversaries and visitor-error sentinel: assert every sensitive alias is visited, exact duplicate policy remains, opaque values survive and owner schema still rejects invalid authority. |
| repolint.uniqueJSONKeys and decodeDebt | Existing debt codec/ratchet tests with exact duplicate, unknown and trailing-document mutations; assert same established exact-key policy and unchanged shrink authority. |
| testwait.DeadlineOnCue | Controlled parent cancel versus explicit expire/cancel ordering, including simultaneous calls under race: assert initially live, first terminal cause wins with correct Err, and cancel releases its parent notification without a retained watcher. Use test deadline only as deadlock guard. |
| OSProvisionIO.Run and its deadline factory | TERM-deaf real child emits marker only after trap installation; capture requested budget then expire on that marker. Assert own-request DeadlineExceeded and actual exact-group/pipe joins, never caller-deadline substitution. |
| vmcmd.Run / private runWithDeadline | Stateful Runtime entered/gates oracle: capture selected budget, reach each named stage, expire, then assert stage-specific refusal and cleanup with an independently live context. |
| agentfixture.runCLI / private runCLIWithDeadline and commandCLITimeout | Pure default/override duration assertion plus real-child startup marker before injected own deadline expiry; assert selected budget, result and exact child/group joins. CLIAdmissionTransition uses fixed queue/deadline times; native expired absolute deadline at full capacity proves no spawn. |
| Cloudflared.Open and its private startup timer factory | Controlled fake-process readiness before timer firing, assert selected duration and exact no-tunnel branch; retain real process/reap observation. Session hidden-indicator uses manualClock.fireGrace only after hidden frame activation. |
| MemController.idleLocked grace callback and instance afterFunc factory | Hold CtlLeave sink write, invoke captured expiry callback outside controller lock, assert credential and turn revocation before release, then join callback. Existing controller contract remains authority. |
| harnessTTY capture startup/teardown helpers | Distinct ready-after-handlers marker gates startup timer; controlled wait results gate escalation while real Kill/Wait/reader joins prove cleanup. Output/classification/geometry tests use normal defaults; stubbed teardown tests assert exact selected grace. |
| launcher independent-retirement observation | Fake world records kill followed by a fresh surviving-owner observation before cue caller expiry; assert exact signal target and refusal, preventing a test that exits before the independent recheck. |
| testpair.ShortRoot and startup optional socket-TMPDIR helper | Existing real listener contract plus long inherited TMPDIR: assert physical path bound/privacy, normal temp preservation, opt-in isolated root, cleanup after nested subtest and environment restoration. |

## Task 1: pin and qualify catalog boundary

Files: go.mod/go.sum; tests/affected-config-test.py; internal/repolint/{local_catalog_test,catalog_test,zellij_query_boundary_rule_test}.go and any newly landed equivalent catalog adapter files; tests/affected-readers for audited changed readers.

- [ ] Preserve a targeted RED catalog compile against the normal581 pin.
- [ ] Set the normal pin, no replace. Copy the qualified fixture-only repair from #838 as a source patch, never its issue commits/branch history. That repair copies affected/jsonstrict/processgroup/pkg-procgroup and same-snapshot go.mod/go.sum into a disposable root module.
- [ ] Adapt all local Match callbacks (including latest main FIFO rule) to CheckContext. A single AST-only adapter preserves direct matcher tests; upstream Match receives its original context. Supply real filename/package context to direct sleep matching.
- [ ] Replay qualification in sibling, exact-ref and installed-module modes; real upstream planner must compile and qualify the declared graph. Run the pinned catalog to inventory newly active findings on this fresh main base; classify any additional sites before widening scope.
- [ ] Record a red/green ledger and commit with #861 trailers. Do not include seeded workflow configuration.

## Task 2: shared JSON authority

Files: internal/jsonstrict/{scan.go,scan_test.go}; cmd/internal/strictjson/{decode.go,decode_equivalence_test.go,*_test.go}; cmd/internal/historyarchive/drive_model.go and receipt/label tests; cmd/internal/submission/privacy.go and privacy/codec tests; internal/repolint/{catalog_test,debt_test}.go; examples/mock-to-real/internal/jsonvalue/value.go.

- [ ] Run the function-scoped Scan and owner-wrapper adversarial strategies above before extraction; preserve their independent oracle and mechanical assertions in colocated named tests.
- [ ] Extract the existing optimized parser into root internal/jsonstrict.Scan(raw, Options), retaining thin Decode/RejectDuplicateKeys wrappers and owner typed validation. Add the options needed by the three readers; no second parser.
- [ ] Route historyarchive duplicate scan through folded/depth64/preserve-number options; preserve strict/non-strict typed decode. Route submission duplicate scan through exact/preserve-number visitor, keeping sensitive-alias policy in submission. Route debt scan through exact structural validation; preserve unknown/trailing checks.
- [ ] Keep the fuzz legacy tokenizer and standalone DOM builder unchanged except the two authorized per-site json-recursive-scanner allows naming ariadne#590. Preserve sample's independent dependency-free module.
- [ ] Run Scan visitor and submission.checkJSONKeys strategies above; retain the independent FuzzRejectDuplicateKeysEquivalent oracle with bounded fuzz runs.
- [ ] Count byte/key work at N and2N, assert linear work and one visitor call per key. Run strictjson/historyarchive/submission/repolint targeted contracts and standalone example contract tests; commit.

## Task 3: bounded canonical socket fixtures

Files: cmd/internal/couchcmd/owned_query_test.go; cmd/internal/layoutcmd/fullscreen_reply_live_test.go; cmd/internal/testcouch/{actor_cancel_test,drive_observation_test}.go; cmd/internal/testcouch/agentfixture/run_test.go; cmd/internal/couchcore/startup_detach_timing_live_test.go.

- [ ] Reuse testpair.ShortRoot(t) for TestCouchQueriesCollectOwnedDescendants, TestNativeReplyNoiseCancellationJoinsOwnedConnections, startActorCLIFixture, fixtureConfig, TestObserveDriveReadinessFailureReleasesFleetAdmission. Preserve exact child/socket shutdown and joins.
- [ ] For TestStartupManagedAttachDetachTimingLive preserve inherited os.TempDir before environment clearing. Only PAIR_STARTUP_PERF_ISOLATE_SOCKET=1 selects a private ShortRoot as fixture TMPDIR. Remove hardcoded /tmp overrides; ordinary build/data temps retain caller TMPDIR.
- [ ] Extract only that environment setup into a test helper; a non-live regression checks long inherited TMPDIR, optional isolation, canonical/private bounds, restoration and immediate cleanup. Existing ShortRoot listener/cleanup contract remains authority.
- [ ] Run named affected tests under deliberately long TMPDIR plus existing ShortRoot contract. Do not enable unrelated startup perf live prerequisites. Commit.

## Task 4: virtual clocks and pure budget assertions

Files: cmd/internal/sessionwatch/{peer_turn_test,probation_test,run_test}.go; cmd/internal/couchcore/{operationdispatch_test,park_wait_test}.go; cmd/internal/wrapcmd/harness_tty_live_test.go.

- [ ] Remove13 virtual Timeout/PIDWait shortcuts in sessionwatch. Drive existing watcherRuntime.Now/Sleep/onSleep across actual production phase boundaries on cue. Missing-PID fakes jump to the applicable boundary; negative cases expire after intended observations. Assert scan/observation sequence so removing a shortcut cannot make success vacuous.
- [ ] Operation dispatch's immediate fake completion uses normal CompletionTimeout. Park selection uses existing completionDeadline/newSpendableBudget, asserts default duration and a caller override greater than default, then explicitly spends/completes.
- [ ] Three harness teardown unit tests already stub waits: use the normal shutdown grace and assert waitBounded/joinReader received it. Do not hide tiny literals behind locals.
- [ ] Run affected sessionwatch/couchcore/harness tests, including races where appropriate. Commit.

## Task 5: cue-owned real expiry

Files: cmd/internal/testwait/{deadline.go,deadline_test.go}; cmd/internal/broadcast/{cloudflared.go,cloudflared_test.go,session_test.go}; cmd/internal/couchcore/{provision_io.go,provision_subprocess_test.go}; cmd/internal/vmcmd/{runtime.go,runtime_test.go}; cmd/internal/launcher/{recovery_terminal_test,recovery_terminal_confirmed_test}.go; cmd/internal/webcast/drive/{controller_mem.go,drawing_test.go}; cmd/internal/testcouch/agentfixture/{run.go,run_test.go,cli_group_test.go,request_lifetime_test.go}; cmd/internal/wrapcmd/{harness_tty_live_test,peer_live_tall_test}.go.

- [ ] Implement and verify testwait.DeadlineOnCue against its controlled-order function strategy above; keep own-request budget factories distinct from caller cancellation.
- [ ] Broadcast hidden-indicator uses existing manualClock after/ticks/fireGrace; cloudflared no-URL fixture publishes process readiness before injected startup timer fires. Assert selected startup budget and exact no-tunnel failure branch.
- [ ] Provision TERM-deaf fixture publishes readiness after installing trap; injected request deadline expires then. Assert selected own request duration and group cleanup. VM RunRequestTimeout waits for each fake named stage before expiry; assert stage reached and independent cleanup context remains live.
- [ ] Launcher strict/confirmed retirement fakes expire the caller only after exact kill plus a subsequent independent observation of the still-live owner. Preserve signal authority assertions.
- [ ] Drawing seat expiry uses instance-local Stop-capable timer factory. Capture callback, invoke outside lock, await held CtlLeave write, assert credentials/turn authority, release sink and join callback.
- [ ] Agent fixture CLI group-leader test removes unrelated short timeout and preserves live descendant proof. Per-request budget test asserts selected duration and expires after child readiness through own deadline factory. Queue deadline uses existing pure admission transition with fixed times; native full-capacity integration supplies already-expired absolute deadline and proves no spawn.
- [ ] PTY normal/geometry/output/classification tests use normal capture budgets. Ignore-signals child emits fixture-ready after handlers; injected startup timer fires after that distinct marker, never the startup predicate. Inject wait outcomes for escalation but retain real Kill/Wait/reader joins. Remove short cleanup grace shortcuts.
- [ ] Run focused native expiry tests repeatedly under race, checking readiness-before-expiry and no retained children/readers. A passing error without stage/readiness proof is insufficient. Commit.

## Task 6: integration and close

- [ ] Re-run full pinned catalog, JSON owner tests and affected native cases. Resolve every named finding; no extra allow beyond the two authorized false positives.
- [ ] Audit each changed reader's inputs and preflight/fallback ownership before its exact printed certificate command. Run make test-affected-inventory; never bulk refresh.
- [ ] Update atlas/index and relevant test-infrastructure map for promoted JSON and timer seams; add lesson for qualifying all dependency API consumers before long recorded runs. Preserve project ownership with hub.
- [ ] Push clean tested head; sdlc local-check run --pre-close once. Send queued attempt to hub for offered priority; retain runner/FIFO. Unknown failures get classified, never blind retried.
- [ ] On PASS run inventory again, sdlc close --issue861 with measured actual and explicit evidence (binary owns one fresh review), then PR/merge. No live operator workflow changed by this adoption; #838 owns the later joint acceptance.
- [ ] Report landed PR/revision to hub; return to #838, merge landed main, rerun its own pre-close, then coordinate combined :0 stage with #851.

## Revisions

### 2026-10-24 — initial source-qualified plan

Separates substrate adoption from #838. Preserves all51 observations and records exactly two hub-approved false-positive sites with upstream ariadne#590. Implementation waits hub approval and change-code; no estimate before plan-quality.

### 2026-10-24 — PQ-1 function-scoped verification refinement

Named the cue API and private deadline variants, and replaced prose-only case inventories with a per-risky-function adversarial strategy and mechanical guard table. Existing approved scope, normal budgets, independent oracle, work counters and readiness ordering remain unchanged.

### 2026-10-24 — implementation integration notes

Corrected source function references to checkJSONKeys and idleLocked. Pin compilation also found removed vocab.CollaborationModes in main's old editorial catalog; preserve its exact three accepted historical strings locally until #838 deletes it. No product mode behavior changes. PTY helper uses a readiness-expiry channel beside its unchanged normal safety timer; real shutdown grace and actual child/reader joins stay unchanged.

### 2026-10-24 — BR-2 explicit concept classifications

Added PURE/INTEGRATION kinds to the core table. The scanner and AST adapter are deterministic transformations; cue deadlines and virtual watcher runtime wrap clock/context notification seams. This documents actual dependencies without changing the approved implementation.
