# Boundary Review — pair#695 (milestone M1)

| field | value |
|-------|-------|
| issue | 695 — Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown |
| repo | pair |
| issue file | workshop/issues/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 26df2fe40f70ab2b9e3d1bfd88c3b243ee42e02e..2649a9775a4f0163d8aeb5c41450f8865dc4210e |
| command | sdlc milestone-close --issue 695 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T07:40:56-07:00 |
| verdict | REWORK |

## Review

Still reviewing: I've read the plan and the binder, and now I'm checking that every thread-record writer keeps unknown fields.

Checking whether slot recovery's writer drops unknown fields, since it bypasses the shared encoder.

Confirmed one real issue: slot recovery's writer drops unknown fields. Next I'm checking the shared codec and the lifecycle address.

Found a second issue: a bad stored binding makes the whole record unreadable instead of just refusing the resume. Running the affected test packages now.

Most couchcore tests pass; I'm rerunning one failing spawn test alone to see whether it's caused by this change or the environment.

```verdict
verdict: REWORK
confidence: high
```

M1's core design works. `planTranscriptBind` is a clean pure rule. The binding is stored inside the resume claim's revision-checked write, which settles the concurrent-rewrite question without a retry loop. Moving #683's codec onto the shared `strictjson` helpers removes real duplication, and the counted tests (1 ledger read, 1 stat, 0 listings) run the production binder over the stateful `FakeRuntime`. Three things block SHIP:

1. **The head fails the repo's own tests.** Two tests in `cmd/internal/artifactpath` fail at head and pass at base.
2. **One record writer drops unknown fields.** Slot recovery rewrites a thread record without the preserving encoder, so those fields are lost. I reproduced this with a probe test. That breaks the main promise of §2a.
3. **The row note isn't shown anywhere.** The "ledger unreadable; resumed the stored binding" note has no production consumer, though the spec says the row shows it.

### 1. Strengths
- `couchcore/transcript_bind.go:73` `planTranscriptBind` is pure and table-tested (`TestPlanTranscriptBind`). IO stays in `bindTranscriptWith` and `readLedgerTranscript` (ARCH-PURE).
- `couchcore/starttransaction.go:97` validates the binding and writes it in the claim's own transition. `TestColdResumeStoresTheBindingOnTheRecord` pins this (ARCH-ORDER).
- `strictjson/object.go` is one shared preserving codec. It refuses duplicate keys at every depth, writes preserved values verbatim, and has a fuzz test. #683's failure codec now uses it with no byte change (ARCH-DRY).
- `threadrecord/transcript.go:46` `ValidateLocation` checks the stored path with the scanners' own path parsers (`RootTranscriptID`). The root's location comes from the runtime, never from the record (ARCH-SECURE).
- `slotLifecycleKey` is the single definition shared by `LifecycleAddress.Key()` and `LifecycleFailureSlotKey`, and `TestLifecycleAddressSlotKeyMatchesFailureSlotKey` pins it.

### 2. Critical
- **The head's test suite is red.** `go test ./cmd/internal/artifactpath` fails two tests at head; both passed at base, where the only failure was the environment's missing embed assets:
  - `TestProductionArtifactReferencesAreExactlyClassified`: "cmd/internal/couchcore/transcript_bind.go: non-artifact source imports artifactpath".
  - `TestNoProductionSymbolIsReferencedOnlyByTests`: `NewSlotAddress` and `NewTeamAddress` have no production reference.
  - **Fix:** classify `transcript_bind.go` correctly (it really does resolve the ledger path), not by adding it to the known-violations list. Then either wire `NewSlotAddress` into production (the plan says `lifecycleFailureTargetKey` delegates to `Key()` in M1, but it calls `slotLifecycleKey` directly), or allowlist both constructors with the reason ("consumed by M2 / #694").

### 3. Important
- **Slot recovery drops unknown fields (ARCH-EVOLVE, ARCH-PURPOSE).**
  - `couchcore/slotrecovery.go:148` `replaceSlotCurrent` calls `json.MarshalIndent(next)` on the couchcore `ThreadRecord`. `Unknown` is tagged `json:"-"`, so it is dropped.
  - Its callers rewrite records they decoded: the cloned survivor at `slotrecovery.go:524` and `:594`, `resume_route.go:292`, and `recovery_prepared.go:497`.
  - I confirmed it with a probe in this checkout (since removed): create a record with an unknown member, observe it, clone it, call `replaceSlotCurrent`, and the member is gone.
  - The guard test `TestThreadRecordWritersUseThePreservingEncoder` only matches `json.Marshal(…(toPersistedThreadRecord(` on one line, so it misses this writer. The atlas claim that `encodeThreadRecord` is the only writer is false.
  - **Fix:** use `encodeThreadRecord(next)` there, widen the guard (for example, ban `MarshalIndent` of any `ThreadRecord` value in couchcore), and add a recovery round-trip test that keeps unknown members.
- **The row note has no consumer.** `TranscriptBind.Note` (`transcript_bind.go:145`) is never read in production:
  - `resumeEvidence` keeps only `Resolution`, and `resumeContextWith` uses only `Resolution` and `Store`.
  - The spec's row note ("ledger unreadable; resumed the stored binding") therefore never reaches the operator; tests check only that the field is non-empty.
  - **Fix:** carry it into the resume result or row detail and test it at that consumer, or record a Revision deferring it.
- **An invalid stored binding makes the whole record unreadable.**
  - `threadrecord.Validate` (`record.go:149`) runs `Transcript.Validate()` at decode. A hand-edited or invalid binding therefore makes the entire thread record fail to decode.
  - The spec says it should produce "Start fresh, invalid stored binding". `ResumeTranscriptInvalid` is now effectively reachable only for ledger-derived candidates.
  - The same strictness hits records from newer builds: `NativeRootNames` and `RootTranscriptID` are this build's catalogues. A newer build that adds an agent or a root name writes a binding this tolerant reader refuses, which defeats §2a.
  - **Fix:** keep only type-level strictness at decode. Do the semantic location check in Bind, so it yields `ResumeTranscriptInvalid`.

### 4. Minor
- `bindByResolver` and `bindTranscriptWith` both extract the agent from `LatestLaunchProfile`; one small helper would do.
- A stored binding for a different agent (after a switch-agent) is labelled `legacy-ledger` by `planTranscriptBind`, but `ledger` by `bindByResolver`. The two should agree.
- The plan says the `legacy-ledger` source is shown in the row detail, but no display reads `Transcript.Source`.
- `NewSlotAddress` doesn't canonicalize `physicalRoot` (no EvalSymlinks), although PQ-7's comment says "canonical root". Either canonicalize it or document that the caller must.

### 5. Test coverage notes
- `strictjson`, `threadrecord`, `sessioninventory` and the new couchcore M1 tests all pass.
- `TestSpawnComposesProductionPairRegistrationBoundary` fails the same way at base ("couch session repository scope changed"), so it comes from the environment, not this diff.
- The full couchcore run hit the 10-minute go-test timeout here.
- Coverage is missing for:
  - an unknown member surviving recovery's `replaceSlotCurrent`;
  - the `Note` reaching the operator;
  - a newer-build binding (unknown root name) being tolerated by this reader.

### 6. Architectural notes
- **ARCH-DRY:** pass, except the encoder bypass above.
- **ARCH-PURE:** pass.
- **ARCH-PURPOSE:** flag. Three gaps:
  - the writer sweep is incomplete;
  - `Note` is a field with no consumer;
  - `LifecycleAddress` has no production consumer, despite the plan's statement that `lifecycleFailureTargetKey` delegates to `Key()`.
- **ARCH-MOCK:** pass (stateful `FakeRuntime`, counted operations).
- **ARCH-CONSTRAINTS:** pass. A resume binds twice (before the claim and in its re-check), so it costs 2 ledger reads and 2 stats. The plan's revision accepts that.
- **ARCH-SECURE:** pass on provenance (the root is resolved by the runtime). The failure path is too wide, though: a bad binding refuses the whole record.
- **ARCH-ORDER:** pass (the binding rides the revision-checked claim).
- **ARCH-FUNERAL:** pass (one bounded field, archived with the record).
- **ARCH-EVOLVE:** flag (the recovery writer drops unknown fields; decode-time validation is bound to this build's catalogues).
- For M2: the agent sidecar and `orphans.json` should go through one preserving encoder per format, with a guard test that can't be dodged by marshalling the struct directly.

### 7. Plan revision recommendations
- `## Revisions`: "Binding validation at decode is type-only. The location check (root name, path parser) runs in Bind and yields `ResumeTranscriptInvalid`." Alternatively, document why the whole record is refused.
- `## Revisions`: "`lifecycleFailureTargetKey` delegates to `Key()` in M1." Either do it or strike it. The code shares `slotLifecycleKey` instead.
- `## Revisions`: "The row note for an unreadable or missing ledger, and the `legacy-ledger` row detail, are surfaced at <consumer>." Or defer them explicitly.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: repo-invariant-tests-red
    title: |
      artifactpath enforcement tests fail at head (pass at base): unclassified artifactpath import and dead constructors
    detail: |
      TestProductionArtifactReferencesAreExactlyClassified flags couchcore/transcript_bind.go importing artifactpath; TestNoProductionSymbolIsReferencedOnlyByTests flags NewSlotAddress and NewTeamAddress (lifecycleaddress.go) as test-only. Classify transcript_bind.go correctly, and wire or allowlist the constructors with a reason.
  - id: new
    severity: Important
    kind: defect
    family: preserving-encoder-single-writer
    title: |
      replaceSlotCurrent marshals ThreadRecord directly and drops Unknown members on recovery rewrites
    detail: |
      slotrecovery.go:148 uses json.MarshalIndent(next), and Unknown is tagged json:"-". Recovered survivors (slotrecovery.go:524/594, resume_route.go:292, recovery_prepared.go:497) lose a newer build's fields; a probe test reproduced it. The guard TestThreadRecordWritersUseThePreservingEncoder only matches toPersistedThreadRecord, so it misses this writer. Use encodeThreadRecord, widen the guard, and add a recovery round-trip test.
  - id: new
    severity: Important
    kind: defect
    family: field-without-consumer
    title: |
      TranscriptBind.Note is never read in production; the spec's row note never reaches the operator
    detail: |
      resumeEvidence and resumeContextWith keep only Resolution and Store. The "ledger unreadable/missing; resumed the stored binding" note required by P2 is dropped. Surface it at the row or resume result and test it there, or record a deferral Revision.
  - id: new
    severity: Important
    kind: defect
    family: decode-strictness-beyond-types
    title: |
      Decode-time Transcript.Validate refuses the whole record instead of "Start fresh, invalid stored binding"
    detail: |
      threadrecord.Validate (record.go:149) runs the semantic location check (NativeRootNames and RootTranscriptID, this build's catalogues) during DecodePersisted. A hand-edited binding, or a newer build's binding with a new root or agent, makes the whole record undecodable, which contradicts the P2 spec and §2a tolerance. ResumeTranscriptInvalid becomes nearly unreachable. Move the location check into Bind.
  - id: new
    severity: Minor
    kind: defect
    family: plan-code-drift
    title: |
      lifecycleFailureTargetKey does not delegate to LifecycleAddress.Key() as the plan claims; NewSlotAddress does not canonicalize its root
    detail: |
      The code shares slotLifecycleKey instead, which is fine in substance. Update the plan, or wire Key(), and document that the caller must canonicalize the root.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"0f2ec6a2-c6fa-4c46-b43e-34ad69f5e40d","repository":"github.com/xianxu/pair","issue_id":"000695","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/a92d2faa5f12cfb7","reviewed_head":"2649a9775a4f0163d8aeb5c41450f8865dc4210e","slot":"pair:1","requested_at":"2026-10-19T14:20:40.378919Z"},"phase":"finished","pid":8399,"started_at":"2026-10-19T14:20:40.604433Z","finished_at":"2026-10-19T14:40:49.656073Z","session_id":"e1eea7b2-f8df-41e6-8cb7-94f6cc4cdb0f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.8193985s
head: "2649a9775a4f0163d8aeb5c41450f8865dc4210e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410

---

## Re-review — 2026-10-19T08:17:38-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 695 — Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown |
| repo | pair |
| issue file | workshop/issues/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 26df2fe40f70ab2b9e3d1bfd88c3b243ee42e02e..b5c0b8135c7873cf1a1ff198227c29bb142d34db |
| command | sdlc milestone-close --issue 695 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T08:17:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing: checking which couchcore tests fail at head.

Reading the resume/start-claim write path now.

Still verifying prior findings — checking that BR-4's deferral to #684 is actually recorded there.

Running the #695 tests at head, then reverting the BR-3 and BR-5 fixes to confirm their regression tests go red.

Both regression tests go red when their fix is reverted. Next I'm testing how nested unknown members inside `transcript` are handled (ARCH-EVOLVE).

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All six prior findings are resolved, and for the two behavioral fixes I checked that their tests fail without them. In a throwaway copy, I removed `ThreadRecord.MarshalJSON` and `TestThreadRecordJSONKeepsUnknownMembers` failed. I put the decode-time `Transcript.Validate` back and `TestInvalidStoredBindingKeepsTheRecordReadable` failed with `transcript root "claude-projects-v2" is not a claude root`. The repo-guard tests in `artifactpath` pass at head. The #695 tests in couchcore pass (`-run 'Transcript|Bind|ThreadRecord|SlotRecovery|LifecycleAddress|LifecycleFailure|ColdResume|Legacy|Resume'`, 63s). One new Important finding remains, under ARCH-EVOLVE. Records now keep unknown members at the top level, but the new nested `transcript` object is still strict. I ran a small throwaway test that added `"minted_at"` inside `transcript`, and `GetThread` failed with `json: unknown field "minted_at"`. So a later build that adds any member to the binding makes the whole record unreadable for this build. That is the same "one binding hides the thread" outcome BR-5 just fixed, coming in through a different path.

**1. Strengths**
- `ThreadRecord.MarshalJSON` (`cmd/internal/couchcore/thread.go:283`) puts the preserving encoding on the type itself, so no writer can skip it. This fixes the BR-3 class, not just the one site. No struct with JSON tags holds a `ThreadRecord`, and none embeds it, so changing its JSON shape has no side effects elsewhere.
- `planTranscriptBind` (`transcript_bind.go:70`) is pure and covered by a table test. The IO shell around it is small: one ledger read and one `StatFile`. The ledger's root and path are checked with `ValidateLocation` before the stat (`:133`), so a hand-edited ledger cannot point the stat anywhere it shouldn't.
- The binding is written by the start claim's own revision-checked write (`starttransaction.go:97`), and `BoundAt` is checked there. There is no second writer and no retry loop.
- The counted tests (`TestResumeBindsByOneStat`, `TestBindAdoptsNewerLedgerRootAfterReset`) assert zero listings on each native root through the stateful `FakeRuntime`.
- `LifecycleAddress` matches PQ-7: the constructors do the IO, while `Validate` and `Key` stay pure.

**2. Critical findings:** none.

**3. Important findings**
- **The nested `transcript` object is strict on decode (ARCH-EVOLVE).** The code comment at `threadrecord/record.go:113` says nested objects stay strict. ARCH-EVOLVE requires unknown nested members to survive decode and rewrite too.
  - Rule to apply: any nested object this issue introduces, or that its milestones will extend, must keep unknown members the same way the top level does.
  - Fix sketch: add `Unknown` plus `UnmarshalJSON`/`MarshalJSON` to `TranscriptBinding`, built on `strictjson.SplitObject`/`EncodeObject`, then test a round-trip where a nested member survives.
  - Plan: write the nested-object policy into §2a. The agent sidecar in M2 will hit the same question.

**4. Minor findings**
- `LifecycleFailureSlotKey` now requires a clean absolute path (`workspaceAbsolute`), not just `filepath.IsAbs`. A failure record whose `StartingPath` is absolute but not clean now falls back to the `thread:` key instead of the `slot:` key. This is a small behavior change to #683 with no test pinning it.
- When the ledger's proof names an invalid root, the refusal says "its stored transcript binding is invalid" (`transcript_bind.go:134`), but the bad input came from the ledger, not the record.
- BR-4's deferred row note and the `legacy-ledger` row detail are recorded only in #695's plan. The #684 tracker file doesn't mention them, so the receiving issue doesn't know it owns them.

**5. Test coverage notes**
- `TestSpawnComposesProductionPairRegistrationBoundary` fails at both base and head with "couch session repository scope changed". This is environmental and not caused by this diff.
- The full couchcore package run hits the 10-minute `go test` timeout in this checkout, so I ran the #695 tests by name.
- `TestSlotRecoveryRewritePreservesUnknownMembers` is redundant with `MarshalJSON` in place. It is still a useful wiring check.

**6. Architectural notes**
- ARCH-DRY: pass. The shared `strictjson` codec is used, and there is one `threadAgent` and one `transcriptSource` helper.
- ARCH-PURE: pass.
- ARCH-PURPOSE: pass. The child's listing check is deferred to M2, and that deferral is recorded.
- ARCH-MOCK: pass. The tests use the counted `FakeRuntime`.
- ARCH-CONSTRAINTS: pass. One read, one stat and zero listings are asserted.
- ARCH-SECURE: pass. Every binding is checked before its stat, whether it came from the ledger or the record.
- ARCH-ORDER: pass. The binding rides the revision-checked claim, and the binding is re-checked after the claim.
- ARCH-FUNERAL: pass. Nothing durable is added beyond one field per record.
- ARCH-EVOLVE: flag (nested strictness, above).
- For M2, follow the plan's rule: the sidecar and the orphan registry should do their preserving encoding in the type's own `MarshalJSON`, nested members included.

**7. Plan revision recommendations**
- §2a: state the nested-object policy, and that `TranscriptBinding` keeps unknown members.
- Record the #683 key tightening: non-clean absolute paths are no longer slot-keyed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      PQ-7 Revision gives the constructor signatures (NewSlotAddress/NewTeamAddress); code Validate is a pure shape check, Scope filled by NewTeamAddress (lifecycleaddress.go:55-79).
  - id: BR-2
    disposition: addressed
    note: |
      artifactpath tests pass at head; transcript_bind.go classified scoped-ledger ResolvedConsumer (manifest.go), NewSlotAddress now used by LifecycleFailureSlotKey, NewTeamAddress allowlisted with pair#694 reason.
  - id: BR-3
    disposition: addressed
    note: |
      ThreadRecord.MarshalJSON = encodeThreadRecord; reverting it makes TestThreadRecordJSONKeepsUnknownMembers fail (verified in checkout).
  - id: BR-4
    disposition: addressed
    note: |
      Note field removed and deferral Revision recorded (plan M1 review round 1, BR-4) as the finding allowed; receiver #684 tracker does not yet carry it (new Minor).
  - id: BR-5
    disposition: addressed
    note: |
      Decode no longer runs Transcript.Validate; restoring it makes TestInvalidStoredBindingKeepsTheRecordReadable fail (verified); Bind refuses or replaces via ValidateLocation.
  - id: BR-6
    disposition: addressed
    note: |
      LifecycleFailureSlotKey delegates to NewSlotAddress(...).Key(); NewSlotAddress documents caller canonicalizes; pinned by TestLifecycleAddressSlotKeyMatchesFailureSlotKey.
findings:
  - id: new
    severity: Important
    kind: defect
    family: nested-unknown-preservation
    title: |
      Nested transcript object is strict: a newer build's member inside "transcript" makes the whole record unreadable
    detail: |
      Probe: CreateThread with a binding, add "minted_at" inside transcript on disk, GetThread fails with json: unknown field "minted_at". record.go:113 keeps nested objects strict, contrary to ARCH-EVOLVE "including nested members"; the result is the same whole-record refusal BR-5 removed. Rule: every nested object this issue introduces or will extend preserves unknown members. Give TranscriptBinding Unknown plus UnmarshalJSON/MarshalJSON via strictjson.SplitObject/EncodeObject, add a nested round-trip test, and state the nested policy in plan §2a.
  - id: new
    severity: Minor
    kind: defect
    family: plan-code-drift
    title: |
      LifecycleFailureSlotKey now requires a clean absolute path; a non-clean absolute StartingPath silently moves from slot: to thread: key
    detail: |
      This is the 2nd finding in family plan-code-drift. Rule: a Revision that routes an existing #683 contract through a new constructor records any narrowing of its accepted inputs. Previously filepath.IsAbs; now workspaceAbsolute (Clean==p, valid UTF-8, no NUL). Record the narrowing in the plan, or pin it with a test.
  - id: new
    severity: Minor
    kind: defect
    family: deferral-recorded-at-receiver
    title: |
      BR-4's row note and legacy-ledger row detail are deferred to #684 M2 only in #695's plan; #684's tracker does not mention them
    detail: |
      Add the deferred items to the #684 issue (or its plan) so the receiving issue owns them. Also, transcript_bind.go:134 says "stored transcript binding is invalid" when the invalid binding came from the ledger proof.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"432b3500-2296-4c6e-8f70-ae2ce14871da","repository":"github.com/xianxu/pair","issue_id":"000695","recipe":"milestone-review","round":3,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/eb4c71944b5a22f4","reviewed_head":"b5c0b8135c7873cf1a1ff198227c29bb142d34db","slot":"pair:1","requested_at":"2026-10-19T14:55:36.449084Z"},"phase":"finished","pid":2534,"started_at":"2026-10-19T14:55:36.560935Z","finished_at":"2026-10-19T15:17:36.530325Z","session_id":"6059e620-ff97-4232-b53f-d86c71cc6d68","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.908824875s
head: "b5c0b8135c7873cf1a1ff198227c29bb142d34db"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-19T08:44:31-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 695 — Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown |
| repo | pair |
| issue file | workshop/issues/000695-lifecycle-primitives-transcript-binding-on-the-record-the-two-kind-address-proven-teardown.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 26df2fe40f70ab2b9e3d1bfd88c3b243ee42e02e..0f1c1bed112bd36f3f7d7936886fed3887d46d4c |
| command | sdlc milestone-close --issue 695 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T08:44:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Focus on the last round's changes first (prior-finding dispositions).

Checking BR-7's fix by reverting it and running the regression test, then running the full test suite.

Comparing that failing spawn test against base in a separate worktree to see if it's environmental.

Spawn test fails at base too, so it's environmental. Now a cleaner BR-7 revert (keep the field, drop the JSON methods).

BR-7's regression goes red without the fix. Checking plan §2a and the other nested types for the class sweep.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 3 closes the one blocking finding left from the earlier rounds. BR-7 is fixed. `TranscriptBinding` now decodes the fields it owns strictly and keeps unknown ones through its own `UnmarshalJSON`/`MarshalJSON`, built on the shared `strictjson` helpers. I reverted just those two methods in this checkout and `TestTranscriptBindingNestedUnknownMembersSurvive` went red with `json: unknown field "future_tty"`; restored, it passes. `TranscriptBinding` is the only nested persisted type this range adds, so the class is fully covered. BR-8 is fixed too: the narrowing is recorded in the plan's Revisions and pinned by a test. Two Minor items remain. First, BR-9's receiver half: the plan says the deferred items were "added to pair#684's plan", but no #684 plan file exists in any ref of this checkout, and the #684 issue file hasn't changed since `f56df570d`. Second, a code comment on `Record.Unknown` still says nested objects stay strict, which is now false. Neither blocks the boundary.

On tests: `threadrecord`, `strictjson`, `sessioninventory` and `artifactpath` pass. In `couchcore` the only failure is `TestSpawnComposesProductionPairRegistrationBoundary` ("couch session repository scope changed"), and it fails the same way at base in a separate worktree, so it comes from the environment, not this diff. The full `couchcore` run also goes over its write budget (≈180 MB against a 128 MB cap). Running the package directly rather than through the recorded suite is a known cause of that (see #657's history).

1. **Strengths**
   - There is now one preserving encoder: `ThreadRecord.MarshalJSON` (`cmd/internal/couchcore/thread.go:283`) means every writer emits the persisted shape, and `encodeThreadRecord` replaced all three `json.MarshalIndent` sites in `threadstore.go`.
   - The `strictjson` helpers (`SplitObject`, `EncodeObject`, `AppendMembers`, `CloneRaw`, `MergeRaw`) are reused by the thread record, the transcript binding and the #683 failure codec, so the logic isn't duplicated (ARCH-DRY).
   - The nested regression test does more than show unknown fields are tolerated. It checks the unknown member survives a real `ApplyThreadMetadata` rewrite, and that a duplicate key or wrong type inside the binding is still refused (`thread_preserve_test.go:163-223`).
   - The binding is validated in the start claim's own transition, `AdvanceStartTransaction` (`starttransaction.go:97-102`), not in the IO shell.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `cmd/internal/threadrecord/record.go:110-114`: the comment on `Record.Unknown` says "nested objects stay strict". Since BR-7 that is false for `transcript`. Reword it to match the Revision's rule: types born in this issue preserve unknown fields from birth; older nested types stay strict.
   - BR-9 (receiver half): the #684 tracker in this tree doesn't own the row note or the `legacy-ledger` row detail. Add them to `workshop/issues/000684-*.md`, or commit #684's plan where this repo can see it.

5. **Test coverage**
   - BR-7: confirmed red without the fix.
   - BR-8: `TestLifecycleAddressSlotKeyMatchesFailureSlotKey` covers a trailing slash, a `..` path and a relative path, and checks the fallback to the thread key.
   - Gap: there is no pure, store-free round-trip test of `TranscriptBinding` in `threadrecord/transcript_test.go`. The `couchcore` test covers it through the store; a pure one would be cheap to add.

6. **Architecture**
   - **ARCH-DRY: pass.** The shared `strictjson` codec is used everywhere it applies.
   - **ARCH-PURE: pass.** The start transition and the binding plan are pure; `bindTranscriptWith` keeps the IO to one stat through the injected `Runtime`.
   - **ARCH-PURPOSE: pass.** The nested sweep covers every new nested type. The BR-9 receiver gap is about tracking, not the purpose.
   - **ARCH-MOCK: pass.** File and ledger access go through `sessioninventory.Runtime` fakes.
   - **ARCH-CONSTRAINTS: pass.** Binding stays at 1 ledger read, 1 stat, 0 listings.
   - **ARCH-SECURE: pass.** Decode checks types only; location is checked at use (`ValidateLocation`), and a bad binding refuses the resume, not the whole record.
   - **ARCH-ORDER: pass.** The binding enters through the start transition.
   - **ARCH-FUNERAL: pass.** The binding is one bounded member per record and is superseded on re-bind; the range creates no new file family.
   - **ARCH-EVOLVE: pass.** Unknown fields are preserved at the top level and inside `transcript`; known-field types and duplicate keys stay strict; there is no version bump, and §2a gives the reason.
   - For M2: the agent sidecar and `orphans.json` must follow the same pattern from the start (`Unknown` plus their own JSON methods, with a nested round-trip test), as the Revision commits.

7. **Plan revisions**
   - Correct the round-2 Revision bullet "added to pair#684's plan as M2 items" so it points to where the deferral actually lives, once it is recorded there.
   - Optionally, move the nested rule ("types born in this issue are tolerant and preserving from birth") into §2a's body, not just the Revisions log.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      TranscriptBinding has its own JSON methods on strictjson (transcript.go:34-73). With them reverted, TestTranscriptBindingNestedUnknownMembersSurvive fails with unknown field future_tty; it is the only new nested persisted type.
  - id: BR-8
    disposition: addressed
    note: |
      Narrowing recorded in the plan's round-2 Revision and pinned in TestLifecycleAddressSlotKeyMatchesFailureSlotKey (lifecycleaddress_test.go:157-168), which passes at head.
  - id: BR-9
    disposition: not-addressed
    note: |
      The refusal text is fixed (transcript_bind.go:125,134). But no #684 plan exists in any ref here, and the #684 issue file is unchanged since f56df570d, so the receiver still does not own the deferred items.
findings:
  - id: new
    severity: Minor
    kind: defect
    family: nested-unknown-preservation
    title: |
      Record.Unknown comment still says nested objects stay strict after BR-7 made transcript tolerant
    detail: |
      This is the 2nd finding in family nested-unknown-preservation. The rule is already in the round-2 Revision: a persisted type born in this issue is tolerant and preserving from birth; older nested types stay strict. record.go:110-114 still states the old policy. Reword it to that rule, so the code no longer contradicts the plan.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"583522e8-c04d-4833-80be-4c648b6a4cb7","repository":"github.com/xianxu/pair","issue_id":"000695","recipe":"milestone-review","round":4,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/8c093c960c259dd5","reviewed_head":"0f1c1bed112bd36f3f7d7936886fed3887d46d4c","slot":"pair:1","requested_at":"2026-10-19T15:27:52.857883Z"},"phase":"finished","pid":37722,"started_at":"2026-10-19T15:27:52.984179Z","finished_at":"2026-10-19T15:44:30.770286Z","session_id":"4a19ae9d-ac38-4a0e-b456-0afbb16d3789","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.800533625s
head: "0f1c1bed112bd36f3f7d7936886fed3887d46d4c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
