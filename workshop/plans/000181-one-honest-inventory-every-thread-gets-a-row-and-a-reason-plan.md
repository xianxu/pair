# One Honest Inventory Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every thread in couch's manifest appears in the switcher with a total
state — `live | detached | parked | busy | unusable(reason)` — and the only way
a row leaves is a deliberate move to an archive.

**Architecture:** Today "should this row exist" is fused with "can this row be
acted on", and the fused answer is computed half in an IO loop
(`ActionableThreadInventoryContext`) and half in a pure projector, with
`continue` as the only vocabulary for "no". This plan splits them: the IO shell
*gathers evidence* and decides nothing (ARCH-PURE); one pure classifier turns
`(record, evidence)` into a state that is **total** — never absent. Two repairs
then convert the two recoverable reasons back into usable rows, and retirement
becomes the single exit.

**Tech Stack:** Go. `cmd/internal/couchcore` (durable model + projection),
`cmd/internal/couchtty` (pure menu + terminal shell), `cmd/internal/launcher`
(Pair's launch flow), `cmd/internal/sessionledger` (the launch/binding ledger
vocabulary — where pair#168 actually lives), `cmd/internal/sessioninventory`
(the query over it).

**Standing constraint — do not degrade Pair.** Pair must keep working as a
standalone program at a lower level; couch layers above it. This plan touches
`launcher/` **not at all**: warm reattach works by couch declining to send an
authority pair only honours at a create boundary, so the path it lands on is
the one standalone `pair resume <tag>` already uses. `sessionledger` is Pair's
package and Task 9 edits it; its whole-suite sweep is that task's guard, and
Task 10 confirms the standalone paths on the real stack anyway.

**Measured starting state** (operator's live store, 2026-09-03), which this plan
is verified against rather than against fixtures:

```
13 records in manifest → 4 rows in the switcher
  live       established        3   (only 2 hosted; brain-couch-19 is stale)
  detached   established        1   tools-couch-2   → M2 makes it reattach
  detached   provisional/no-id  1   pair-couch-24   → M1 shows it, M2 repairs it
  parked     established        1   parley          → the only working park
  parked     provisional/no-id  8   → M1 shows them, M2 repairs what it can
```

---

## Chunk 1: M1 — one total inventory

### Core concepts

M1 changes no lifecycle behaviour. It changes who decides, and it makes the
nine hidden threads visible with the reason they are not actionable.

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `ThreadReason` | `cmd/internal/couchcore/threadreason.go` | new |
| `AllThreadReasons` | `cmd/internal/couchcore/threadreason.go` | new |
| `ThreadEvidence` | `cmd/internal/couchcore/actionableinventory.go` | new |
| `ClassifyThread` | `cmd/internal/couchcore/actionableinventory.go` | new |
| `liveProofMatches` | `cmd/internal/couchcore/actionableinventory.go` | new |
| `ActionableThreadState` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `ActionableThreadSummary` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `ProjectActionableThreads` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `actionableThreadState` | `cmd/internal/couchcore/actionableinventory.go` | deleted |
| `menuActionItems` | `cmd/internal/couchtty/menu.go` | modified |
| `rootStateText` | `cmd/internal/couchtty/menu_render.go` | new |

`ThreadReason` gets its **own file** rather than growing
`actionableinventory.go` (~300 lines today, and this milestone adds an evidence
type, a classifier and a helper to it). The reason vocabulary is consumed by
three packages and by M3's retirement rule, so it is the one piece with a
different change cadence from the projector.

- **ThreadReason** — why a thread is not actionable, one closed vocabulary:
  `binding-lost`, `stale-incarnation`, `unrecorded-child`, `session-gone`,
  `never-started`, `invalid`, `path-missing`, `profile-missing`,
  `agent-unsupported`, `unknown`. Empty for every actionable state.
  `unknown` means the evidence could not be resolved this round — it is the
  only reason that is both transient and never archive-eligible, and it is what
  keeps a failed subprocess call from reading as a finished thread. `AllThreadReasons()`
  returns them all, so display and archive tables iterate the vocabulary
  instead of restating it — Go has no exhaustive-switch check, so the
  enumeration is the guard.
  - **Relationships:** 1:1 with a summary row; set only when
    `State == ThreadUnusable`.
  - **DRY rationale:** the shell loop and `actionableThreadState` between them
    encode eight distinct refusal *conditions* and discard every one. One
    vocabulary replaces them, and M3's archive rule is written over this enum
    rather than re-deriving what "finished" means.
  - **Future extensions:** additive. A new reason that no display label covers
    fails `TestEveryReasonRendersADistinctLabel`.
  - **Note on the two incarnation reasons:** they are opposite directions of
    one disagreement and stay distinct because their exits differ.
    `stale-incarnation` = the record claims live and nothing hosts it (pair#171,
    the crash shape; reconcilable). `unrecorded-child` = a live observation for
    a record with no incarnation (should be unreachable; fail-closed rather
    than guessing the row is fine).

- **ThreadEvidence** — everything the IO shell resolved about one record, and
  **whether it managed to ask**:

  ```go
  type ProofStatus uint8
  const (
      ProofUnresolved ProofStatus = iota // never asked, or asking failed
      ProofResolved                      // asked and answered, positively or not
  )

  type ThreadEvidence struct {
      Live           []ProcessIdentity
      Parked         []ParkedResumeObservation
      ParkedStatus   ProofStatus
      Detached       []DetachedSessionObservation
      DetachedStatus ProofStatus
      PathError      error
  }
  ```

  **Absence of proof is not proof of absence, and the type has to say which.**
  Without the status fields a total classifier turns every unresolved question
  into a positive claim: a transient `DetachedSessions` failure (today handled
  at `actionableinventory.go:285-292` as "not proved this round") would assert
  `session-gone` on every detached row — and `session-gone` is a reason M3's
  `DecideRetirement` acts on. That is a path from one failed subprocess call to
  archiving live sessions. `ProofUnresolved` classifies `unusable(unknown)`,
  which is **never archive-eligible**.
  - **Relationships:** 1:1 with a record, built per refresh, keyed by
    `ThreadAddress` (not positionally — `ProjectActionableThreads` sorts its
    output at `:135-140`, and positional pairing would rot the first time
    someone sorts the input).
  - **Deliberately absent:** the raw `NativeBindingResolution`. Its only
    consumer is the resume proof, and the proof is already a field. Carrying
    both would give two sources for one fact (ARCH-DRY). What changes is that
    the *absence* of a proof is now classified as a reason instead of dropping
    the row in the shell.
  - **DRY rationale:** the shell currently applies evidence as filters at eight
    conditions. Carrying it as one value lets the pure classifier apply it once,
    and a test constructs any combination with no IO seam (ARCH-PURE).
  - **Future extensions:** M3's retirement needs "does the native transcript
    still exist" — one more field, resolved in the same shell pass.

- **ClassifyThread** — `func ClassifyThread(record ThreadRecord, evidence
  ThreadEvidence) (ActionableThreadState, ThreadReason)`. **Total**: every input
  produces a state; there is no third return value and no way to say "hidden".
  - **DRY rationale:** replaces `actionableThreadState` plus the shell's filter
    cascade, which encoded overlapping rules in two layers.
  - **Future extensions:** M2 relaxes exactly one branch (a detached row stops
    requiring `NativeID`); M3 reads the reason. Both are edits to one function.

- **ActionableThreadState** — gains `ThreadBusy` (park transaction in flight)
  and `ThreadUnusable`. `Live()`, `Detached()` and `Resumable()` keep their
  meanings, so `SelectUniqueResumableRoot` (`startup.go:29`) can never select an
  unusable row — verified: it filters on `Resumable()`.

- **ActionableThreadSummary** — gains `Reason ThreadReason`. (Superseded: the
  plan also had it gain `Incarnations`; see Revisions.)

- **rootStateText** — pure `func rootStateText(thread ActionableThreadSummary,
  now time.Time) string` for the right-hand column. Today `menu_render.go:295-298`
  renders anything not `Live()` as `parked · <age>`, so the operator's one
  detached row is labelled parked.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ActionableThreadInventoryContext` | `cmd/internal/couchcore/actionableinventory.go:209-295` | modified | ledger + zellij + path resolution |

- **ActionableThreadInventoryContext** — resolves evidence for every record in
  the snapshot. It never skips a record; where resolution fails it records the
  failure *in* the evidence.
  - **Injected into:** `ProjectActionableThreads(records, evidence
    map[ThreadAddress]ThreadEvidence) []ActionableThreadSummary` — the new
    signature, replacing today's four parallel slices.
  - **Cost discipline, one contract:** both `Physical` and the binding resolver
    are called only for **resume-shaped** records — no incarnation, no park, not
    a reservation, a saved profile with a supported agent and non-nil argv — and
    the resolver additionally requires a successful `Physical()`. That is today's
    condition set (`:237-247`) restated once. Step 5's call-count test is the
    binding statement; prose and snippet follow it. The zellij snapshot still runs
    only when at least one detach candidate exists (`:278`); the candidate set
    grows (bad bindings now qualify) but `DetachedSessions` fans out per session
    on the host, not per candidate (`artifactcollision.go:198-204`), so the
    subprocess count is unchanged.
  - **The guard is a call-counting test, not a benchmark.** `BenchmarkMenu100`
    (`menu_perf_test.go:91`) runs over a fixture `[]ActionableThreadSummary`
    and never calls the inventory, the resolver, `Physical` or zellij, so it
    cannot observe any of this. Task 2 extends the existing class instead
    (`TestActionableInventoryAsksOnlyAboutDetachCandidates`,
    `TestActionableInventorySkipsTheQueryWithNoCandidates` from pair#170 M2):
    count resolver and `Physical` calls against a fake and assert they are made
    only for resume-shaped records (ARCH-CONSTRAINTS).

### Task 1: `ThreadReason` and a total `ClassifyThread`

**Files:**
- Create: `cmd/internal/couchcore/threadreason.go`, `cmd/internal/couchcore/threadreason_test.go`
- Modify: `cmd/internal/couchcore/actionableinventory.go:11-23` (state vocabulary), `:144-176` (`actionableThreadState` → `ClassifyThread` + `liveProofMatches`)
- Test: `cmd/internal/couchcore/actionableinventory_test.go`

- [x] **Step 1: Write the failing test — the totality property**

```go
type classifyCase struct {
	name       string
	record     ThreadRecord
	evidence   ThreadEvidence
	wantState  ActionableThreadState
	wantReason ThreadReason
}

// The property that makes the inventory honest: every record produces a row.
// The cross product, not a sample -- a future branch that forgets to classify
// something fails here instead of silently vanishing.
func TestClassifyThreadIsTotalOverEveryRecordShape(t *testing.T) {
	for _, tc := range everyThreadShape(t) { // reservation, parking, verified park
		// with and without proof, detached with proof, detached with a session
		// but no binding, detached with nothing, live hosted, live unhosted,
		// unrecorded child, invalid record, unphysicalizable path, no profile,
		// unsupported agent
		t.Run(tc.name, func(t *testing.T) {
			state, reason := ClassifyThread(tc.record, tc.evidence)
			if state == "" {
				t.Fatalf("ClassifyThread returned no state for %s", tc.name)
			}
			if (state == ThreadUnusable) != (reason != "") {
				t.Fatalf("%s: state=%q reason=%q -- a reason iff unusable", tc.name, state, reason)
			}
			if state != tc.wantState || reason != tc.wantReason {
				t.Fatalf("%s = (%q, %q), want (%q, %q)", tc.name, state, reason, tc.wantState, tc.wantReason)
			}
		})
	}
}

// The accepting branches must be byte-for-byte what actionableThreadState
// accepted. This is the characterization half: same inputs, same actionable
// answers, so M1 provably changes only the refusals.
func TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted(t *testing.T) {
	for _, tc := range everyThreadShape(t) {
		state, _ := ClassifyThread(tc.record, tc.evidence)
		actionable := state == ThreadLive || state == ThreadParked || state == ThreadDetached
		if actionable != tc.wasActionableBefore {
			t.Fatalf("%s: actionable=%v, previously %v", tc.name, actionable, tc.wasActionableBefore)
		}
	}
}
```

- [x] **Step 2: Run it to verify it fails**

Run: `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/internal/couchcore/ -run TestClassifyThread -v`
Expected: FAIL — `undefined: ClassifyThread`

- [x] **Step 3: Write `ClassifyThread`, keeping every acceptance rule unchanged**

Branch order is preserved from `actionableThreadState:144-176`: reservation and
park first, verified park second, detached third (**before** live, so a stale
incarnation can never masquerade as a clean detach), live last.

```go
func ClassifyThread(record ThreadRecord, evidence ThreadEvidence) (ActionableThreadState, ThreadReason) {
	if ValidateThreadRecord(record) != nil {
		return ThreadUnusable, ReasonInvalid
	}
	if record.Reservation {
		return ThreadUnusable, ReasonNeverStarted
	}
	if record.Park != nil {
		return ThreadBusy, ""
	}
	// The live branch comes BEFORE the resume-shaped refusals, because today
	// they never touch a live row: actionableinventory.go:237 skips records
	// carrying an incarnation, so such a record is never physicalized and its
	// profile is never read. Ordering path/profile/agent first would make a
	// running agent whose directory was removed -- or a nil c.Path -- classify
	// unusable, which is a behaviour change M1 explicitly does not make.
	if len(record.Incarnations) != 0 {
		if liveProofMatches(record, evidence.Live) {
			return ThreadLive, ""
		}
		return ThreadUnusable, ReasonStaleIncarnation
	}
	if len(evidence.Live) != 0 {
		// A hosted child for a record carrying no incarnation: record and
		// console disagree in the direction that should be impossible.
		return ThreadUnusable, ReasonUnrecordedChild
	}
	// Everything below is resume-shaped, and only here do the shell's
	// resolution failures matter.
	if evidence.PathError != nil {
		// Today's :245-247 drop. It must stay a refusal: an unphysicalized
		// WorkingPath would reach SelectUniqueResumableRoot, which compares
		// paths by exact string (startup.go:29).
		return ThreadUnusable, ReasonPathMissing
	}
	if record.LatestLaunchProfile == nil {
		return ThreadUnusable, ReasonProfileMissing
	}
	if !launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) || record.LatestLaunchProfile.Argv == nil {
		return ThreadUnusable, ReasonAgentUnsupported
	}
	if record.VerifiedPark != nil {
		switch {
		case evidence.ParkedStatus == ProofUnresolved:
			return ThreadUnusable, ReasonUnknown
		case parkedResumeProofMatches(record, evidence.Parked):
			return ThreadParked, ""
		default:
			return ThreadUnusable, ReasonBindingLost
		}
	}
	switch {
	case evidence.DetachedStatus == ProofUnresolved:
		// Not "no session" -- "we could not ask". The distinction is the
		// difference between a row that waits and a row M3 archives.
		return ThreadUnusable, ReasonUnknown
	case detachedResumeProofMatches(record, evidence.Detached):
		return ThreadDetached, ""
	case len(evidence.Detached) != 0:
		// A live session whose binding is unusable: pair#168's shape, and the
		// row the operator could not reattach.
		return ThreadUnusable, ReasonBindingLost
	default:
		return ThreadUnusable, ReasonSessionGone
	}
}

// liveProofMatches is actionableThreadState:165-175, extracted verbatim so the
// live rule sits beside the two proof matchers it is a sibling of.
func liveProofMatches(record ThreadRecord, observations []ProcessIdentity) bool {
	if len(record.Incarnations) != 1 || len(observations) != 1 {
		return false
	}
	incarnation := record.Incarnations[0]
	if incarnation.State != IncarnationLive || incarnation.PID <= 0 || incarnation.Identity == "" {
		return false
	}
	return observations[0] == ProcessIdentity{PID: incarnation.PID, Identity: incarnation.Identity}
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/internal/couchcore/ -count=1`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add cmd/internal/couchcore/threadreason.go cmd/internal/couchcore/threadreason_test.go \
        cmd/internal/couchcore/actionableinventory.go cmd/internal/couchcore/actionableinventory_test.go
git commit -m "#181 M1: couch: classification becomes total, and refusals get names"
```

### Task 2: The shell gathers evidence and decides nothing

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go:101-142` (`ProjectActionableThreads` signature), `:209-295` (`ActionableThreadInventoryContext`), `:206-207` (`ActionableThreadInventory`)
- Test: `cmd/internal/couchcore/actionableinventory_test.go`

- [x] **Step 1: Write the failing test — the shell drops nothing**

```go
// The regression this whole issue is: nine of thirteen records reached no row.
// Asserted as an identity between the store and the projection, over a store
// holding one record of every unusable shape.
func TestInventoryEmitsOneRowPerManifestRecord(t *testing.T) {
	couch, addresses := couchWithOneRecordOfEveryShape(t) // no binding, stale
	// incarnation, unsupported agent, invalid record, reservation, no profile
	rows, err := couch.ActionableThreadInventoryContext(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(addresses) {
		t.Fatalf("rows = %d, records = %d -- the shell dropped %d",
			len(rows), len(addresses), len(addresses)-len(rows))
	}
	for _, address := range addresses {
		if _, ok := findRow(rows, address); !ok {
			t.Fatalf("record %+v produced no row", address)
		}
	}
}
```

- [x] **Step 2: Run it to verify it fails**

Expected: FAIL — `rows = 2, records = 7 -- the shell dropped 5`

- [x] **Step 3: Replace the filter cascade with an evidence pass**

Each refusal at `:228-272` becomes a field. The two cost bounds survive as
*lazy resolution*, not filtering — note the guard reproduces today's conditions
exactly, including the ones the record must satisfy before a binding lookup is
worth doing:

```go
evidence := map[ThreadAddress]ThreadEvidence{}
for i := range snapshot.Records {
	record := snapshot.Records[i]
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	item := ThreadEvidence{Live: observed[record.Address]}
	// Physicalization is resume-shaped work too: today only a record with no
	// incarnation reaches it (:237), and a live row must not become
	// path-missing because its directory moved. One contract, stated once --
	// the call-count guard in Step 5 is what binds it.
	resumeShaped := len(record.Incarnations) == 0 && record.Park == nil && !record.Reservation &&
		record.LatestLaunchProfile != nil &&
		launcher.IsSupportedAgent(record.LatestLaunchProfile.Agent) &&
		record.LatestLaunchProfile.Argv != nil
	if resumeShaped {
		switch {
		case c.Path == nil:
			// Kept from :237. Path is an interface field (couch.go:25) and
			// plan_contract_test.go mutation-tests this guard; dropping it is a
			// nil dereference.
			item.PathError = errors.New("path operations are unavailable")
		default:
			physical, err := c.Path.Physical(record.WorkingPath)
			if err != nil {
				item.PathError = err
			} else {
				snapshot.Records[i].WorkingPath = physical
			}
		}
	}
	if resumeShaped && item.PathError == nil && resolver != nil {
		agent := record.LatestLaunchProfile.Agent
		binding, err := resolver.ResolveEstablished(ctx, record.Address.RepoScope, string(record.Address.Tag), agent)
		if record.VerifiedPark != nil {
			if err == nil && bindingResumeDiagnostic(binding) == "" {
				resumable = append(resumable, ParkedResumeObservation{
					Address: record.Address, Agent: agent, NativeID: binding.NativeID,
				})
			}
		} else {
			// A detach candidate regardless of binding health: whether the
			// SESSION is alive is a different question from whether the
			// agent's transcript id resolved, and conflating them is what hid
			// pair-couch-24.
			detachedCandidates = append(detachedCandidates, DetachedCandidate{
				Address: record.Address, Agent: agent, NativeID: binding.NativeID,
			})
		}
	}
	evidence[record.Address] = item
}
```

Then fold the resolved proofs into the map before projecting, and change
`ProjectActionableThreads` to `(records []ThreadRecord, evidence
map[ThreadAddress]ThreadEvidence)`, emitting one row per record with
`ClassifyThread`'s answer.

- [x] **Step 4: Run the whole suite, not just the three couch packages**

Run: `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/... -count=1`
Expected: PASS, except the tests that assert an unprovable record produces NO
row. 62 references across 15 files reach the projector or the inventory,
including `couchcore/detachedsessions_test.go`, `couchcore/startup_test.go`,
`couchcmd/readme_test.go` and `artifactpath/deadsymbols_test.go` — the last is
outside the couch packages. **Convert each to assert the row's reason; do not
delete them.** The fail-closed property becomes "not actionable", not
"invisible", and that is exactly what those tests should now pin.

- [x] **Step 5: Pin the cost bounds with call counts**

```go
// The bound that matters is per-record work, and no existing test or benchmark
// observes it: BenchmarkMenu100 runs over a fixture slice and never reaches the
// inventory at all.
func TestEvidencePassAsksOnlyAboutResumeShapedRecords(t *testing.T) {
	couch, fake := couchWithCountingResolver(t) // records: live, parked, detached,
	// reservation, parking, profile-less
	_, err := couch.ActionableThreadInventoryContext(context.Background(), liveObservations)
	if err != nil {
		t.Fatal(err)
	}
	if fake.ResolveCalls() != 2 { // the parked and the detached record only
		t.Fatalf("resolver called %d times, want 2 -- live and non-startable records must cost nothing", fake.ResolveCalls())
	}
	if fake.PhysicalCalls() != 2 {
		t.Fatalf("Physical called %d times, want 2", fake.PhysicalCalls())
	}
}
```

- [x] **Step 6: Commit**

```bash
git commit -am "#181 M1: couch: the IO shell gathers evidence and decides nothing"
```

### Task 3: Rows say why, and detached stops calling itself parked

**Files:**
- Modify: `cmd/internal/couchtty/menu_render.go:295-298`, adding `rootStateText` in the same file
- Test: `cmd/internal/couchtty/menu_render_test.go`

- [x] **Step 1: Write the failing tests**

```go
func TestRootRowStateTextNamesEveryState(t *testing.T) {
	for _, tc := range []struct {
		state  couchcore.ActionableThreadState
		reason couchcore.ThreadReason
		want   string
	}{
		{couchcore.ThreadLive, "", "live"},
		{couchcore.ThreadDetached, "", "detached · 4h ago"},
		{couchcore.ThreadParked, "", "parked · 4h ago"},
		{couchcore.ThreadBusy, "", "parking…"},
		{couchcore.ThreadUnusable, couchcore.ReasonBindingLost, "binding lost"},
		{couchcore.ThreadUnusable, couchcore.ReasonStaleIncarnation, "stale — couch exited unexpectedly"},
	} {
		if got := rootStateText(summaryWith(tc.state, tc.reason), fixedNow); got != tc.want {
			t.Fatalf("%s/%s = %q, want %q", tc.state, tc.reason, got, tc.want)
		}
	}
}

// Go cannot check a switch for exhaustiveness, so the vocabulary checks itself.
func TestEveryReasonRendersADistinctLabel(t *testing.T) {
	seen := map[string]couchcore.ThreadReason{}
	for _, reason := range couchcore.AllThreadReasons() {
		label := rootStateText(summaryWith(couchcore.ThreadUnusable, reason), fixedNow)
		if label == "" {
			t.Fatalf("reason %q renders nothing", reason)
		}
		if other, clash := seen[label]; clash {
			t.Fatalf("reasons %q and %q both render %q", reason, other, label)
		}
		seen[label] = reason
	}
}
```

- [x] **Step 2: Run — FAIL, `undefined: rootStateText`**
- [x] **Step 3: Implement, switching on state then reason, defaulting to the raw reason string so a new reason is legible before it is styled**
- [x] **Step 4: Run `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/internal/couchtty/ -count=1`** — PASS
- [x] **Step 5: Commit**

### Task 4: Enter on an unusable row explains instead of doing nothing

**Files:**
- Modify: `cmd/internal/couchtty/menu.go:388-401` (`reduceRootKey` Enter), `:954-961` (`menuActionItems`), and check `hiddenThreadNotice` (`:1175`)
- Test: `cmd/internal/couchtty/menu_test.go`

`hiddenThreadNotice` exists to explain a row that vanished from the inventory
between keystrokes. After M1 nothing vanishes until archive, so it should
either narrow to the archive case or go; decide with the diff in hand and say
which in the commit message.

- [x] **Step 1: Write the failing test**

```go
// Silence is what the operator reports as a bug. An unusable row explains
// itself and offers what it still CAN do (rename, describe), never resume.
func TestEnterOnAnUnusableRowExplainsAndDispatchesNothing(t *testing.T) {
	state := NewMenuState([]couchcore.ActionableThreadSummary{
		unusableRow(couchcore.ReasonBindingLost),
	}, couchcore.ThreadAddress{})
	got, effects := reduceKey(state, PanelKey{Kind: KeyEnter})
	if len(effects) != 0 {
		t.Fatalf("unusable row dispatched %+v", effects)
	}
	if got.Notice.Level != MenuNoticeError || !strings.Contains(got.Notice.Text, "binding") {
		t.Fatalf("notice = %+v, want the reason", got.Notice)
	}
}

func TestUnusableRowOffersOnlyMetadataActions(t *testing.T) {
	items := menuActionItems(unusableRow(couchcore.ReasonSessionGone))
	if slices.Contains(items, "resume") || slices.Contains(items, "detach") || slices.Contains(items, "park") {
		t.Fatalf("action items = %v, want metadata only", items)
	}
}
```

- [x] **Step 2: Run — FAIL** (today Enter dispatches `switch` for a non-resumable row)
- [x] **Step 3: Implement**
- [x] **Step 4: Run the couchtty suite** — PASS
- [x] **Step 5: Commit**

### Task 5: One population — `couch --list` and the switcher agree

**Files:**
- Modify: `cmd/internal/couchcore/threadinventory.go`, `cmd/internal/couchcmd/run.go:570-607` (`renderThreadRows`)
- Test: `cmd/internal/couchcmd/run_test.go`

`couch --list` walks `ThreadInventory` (`ThreadSummary`); the switcher walks
`ActionableThreadInventoryContext`. That is why `--list` showed 13 and the
switcher 4 with nothing reconciling them (ARCH-DRY). The CLI has no console, so
it supplies process liveness instead: an incarnation is live when `c.Proc`
(`couch.go:27`) confirms the exact `{PID, Identity}` — the same recycled-PID
defence the TTY proof gives.

- [x] **Step 1: Write the failing test** — over one store, `--list` and the
      switcher return the same set of addresses.
- [x] **Step 2: Run — FAIL** (different populations)
- [x] **Step 3: Implement** — `ThreadInventory` classifies through
      `ClassifyThread`, supplying `Live` from `ProcOps` **and resolving the same
      parked/detached proofs the switcher does**. Passing a stub with no proofs
      would make every healthy park read `binding-lost` and every healthy detach
      read `session-gone` in `--list` — permanently, and on a reason M3 acts on.
      One inventory means one evidence pass, not one classifier fed two
      qualities of evidence. The cost is a zellij snapshot per `--list`
      (~1.5s measured in pair#170 M3); acceptable for a diagnostic verb, and
      stated here rather than discovered.
      Per-incarnation diagnostic lines stay; the *population* is identical by
      construction.
- [x] **Step 4: Run `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/... -count=1`** — PASS
- [x] **Step 5: Commit**

### Task 6: Docs, real-store measurement, close M1

**Files:**
- Modify: `atlas/couch.md`, `README.md:385-387`, `cmd/internal/couchcmd/readme_test.go:49`

- [x] **Step 1: Fix the pinned README sentence that M1 makes false.**
      `README.md:385-387` currently claims unsupported or ambiguous lifecycle
      records "stay available through `couch --list` / `couch --show`
      diagnostics rather than being mislabeled in the ordinary switcher" — the
      exact invariant M1 reverses. It is pinned as a required substring by
      `readme_test.go:49`, so the suite stays green while the doc lies. Update
      both together.
- [x] **Step 2: Update `atlas/couch.md`** — the projection is total, the reason
      vocabulary and its exits, and the one-population rule.
- [x] **Step 3: Measure the real store.** Run the switcher against the
      operator's live store and record the row count. Expected **13 rows**, up
      from 4: 3 live (one reading `stale — couch exited unexpectedly`), 1
      detached, 1 parked, 8 `binding lost`. Record the actual counts in the
      issue Log; if they differ, the classifier is wrong, not the expectation.
- [x] **Step 4: `env -u PAIR_SESSION_ID -u PAIR_TAG make test`** — exit 0.
- [x] **Step 5: `sdlc milestone-close --issue 181 --milestone M1`**

---

## Chunk 2: M2 — get back in

### Core concepts

M1 made the nine visible. M2 turns as many as possible back into rows the
operator can enter. Two small, independent repairs — **not** a new mechanism.

**Warm reattach already exists.** `AttachExistingSession`
(`launcher/lifecycle.go:36-64`) is the whole of it: set env, touch the draft,
set the terminal title, record the outer tty, spawn the title poller, then
`zellij attach`. It consults no launch profile, no native binding and no agent
args — the `agent` parameter only labels the title poller. Standalone
`pair resume <tag>` reaches it today for any live session.

**What blocks couch is one flag it should not be sending.** couch spawns
`pair resume <tag> --layout2` with a trusted profile carrying
`ResumeRequired: true` (`launch_existing.go:32`, `resume.go:275-278`), and
`createflow.go:238` honours that authority only at a create boundary. The flag
exists to force `--resume <native id>` and to skip the config picker — the two
things a **cold** resume needs and a warm reattach does not. The picker is
never reached on the attach path anyway (`:453`, after the branch returns at
`:275`).

An earlier draft of this plan threaded a `ResumeMode` through
`TrustedLaunchProfile`, `ValidateTrustedLaunchProfile`, `LaunchArgs` and a new
boundary guard, and then had to defend two hazards inside pair's attach branch
(a skipped `RegisterExistingCouchThread`, a layout conflict that calls
`DeleteSession`). Both hazards were artifacts of that design: a couch child that
sends no profile behaves **exactly** as the operator's own
`pair resume <tag>` does, which is the baseline they already trust, so there is
nothing new to defend. Dropping `--layout2` on reattach removes the layout
conflict outright — a running session already has its layout. **No pair-side
change at all** (ARCH-SIMPLE; and it keeps the standing constraint trivially,
since `launcher/` is not touched).

Cold resume is unchanged and needs no work: it genuinely is create plus
`--resume <id>`, and it works today — `parley` resumes. The eight broken parks
are broken by pair#168, not by the resume path.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `DecideResume` | `cmd/internal/couchcore/resume.go:93-126` | modified |
| `detachedResumeProofMatches` | `cmd/internal/couchcore/actionableinventory.go:183-192` | modified |
| `CurrentLaunch` | `cmd/internal/sessionledger/record.go:499-540` | modified |

- **DecideResume** — `bindingResumeDiagnostic` applies when the authority is a
  verified park. A detached thread consumes no native id, so demanding one is
  proof for a step that does not happen. `RequiredSessionID` stays empty for
  warm, which is what tells the launch path it is a reattach.

- **detachedResumeProofMatches** — stops requiring `NativeID`. Every other
  requirement stays, so the row remains fail-closed on the question it actually
  answers: is there exactly one live, client-less zellij session bound to this
  exact address.

- **CurrentLaunch** — a `launch` row with no `binding` row is a **pending
  attempt**, not new authority; the last established binding stays current until
  a new binding commits (pair#168's Spec, in its own words). Today it takes the
  highest-ordinal launch (`:502-510`) then accepts only bindings whose
  `LaunchOrdinal` equals it (`:516`), so a trailing launch orphans every earlier
  binding. Fixed here rather than in `sessioninventory/query.go`, which only
  reads `current.Binding == nil` (`query.go:113-121`) — pending-vs-committed
  belongs to the package owning the ledger vocabulary (ARCH-DRY).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `launchTrackedThread` | `cmd/internal/couchcore/launch_existing.go:24-52` | modified | spawning `pair` |

- **launchTrackedThread** — gains one branch: a warm reattach spawns
  `pair resume <tag>` with **no** `COUCH_LAUNCH_PROFILE` and **no**
  `--layout2`. The `COUCH_THREAD_SCOPE` / `COUCH_THREAD_TAG` / `COUCH_TREE`
  env stays — that is what marks the child couch-owned and is independent of
  the profile.
  - **Injected into:** nothing pure; it is the seam.
  - **What couch keeps:** the child is a pty-hosted process couch tracks like
    any other, so incarnations, `alt+d` detach and `leave` are unchanged.

### Task 7: Prove both refusals before fixing either

**Files:**
- Test: `cmd/internal/couchcore/resume_test.go`, `cmd/internal/sessionledger/record_test.go`

- [ ] **Step 1: Write two failing tests** for the behaviour we want:

```go
// The operator's exact case: the zellij session is alive with zero clients.
// Today DecideResume refuses because the agent's transcript id did not resolve
// -- a proof for a step warm reattach does not take.
func TestDecideResumeAcceptsADetachedThreadWithNoNativeBinding(t *testing.T) { /* … */ }

// Measured ledger shapes, 2026-09-03:
//   resumable:  legacy launch binding … legacy launch binding
//   LOST:       legacy launch binding … legacy launch binding launch  <- trailing launch
func TestTrailingLaunchDoesNotShadowTheLastEstablishedBinding(t *testing.T) { /* … */ }
```

- [ ] **Step 2: Run — both FAIL.** Record the exact refusal strings
      (`resume-binding-provisional`, and the provisional status from
      `CurrentLaunch`) in the issue Log: they are the evidence the tests bite.
- [ ] **Step 3-5:** no implementation. Commit the red tests alone so the
      repair's effect shows in one diff.

### Task 8: A warm reattach sends no resume profile

**Files:**
- Modify: `cmd/internal/couchcore/resume.go:93-126,**212-215**,269-278`, `cmd/internal/couchcore/launch_existing.go:24-52`, `cmd/internal/couchcore/actionableinventory.go:183-192`
- Test: `cmd/internal/couchcore/resume_test.go`, `cmd/internal/couchcore/launchhelper_test.go`

- [ ] **Step 1:** `DecideResume` applies `bindingResumeDiagnostic` only when
      `record.VerifiedPark != nil`; warm carries an empty `RequiredSessionID`.
- [ ] **Step 1b: the earlier refusal site, which relaxing `DecideResume` alone
      does not reach.** `ResumeContext:212-215` calls
      `bindings.ResolveEstablished`, and that resolver **returns an error itself**
      for a provisional binding (`resume.go:172-175`; the fake does the same at
      `artifactcollision_fake.go:123`), so `pair-couch-24` refuses before
      `DecideResume` ever runs. The warm path must not consult the resolver at
      all: resolve the binding only when `thread.VerifiedPark != nil`. Without
      this step M2's Done-when is unreachable.
- [ ] **Step 2:** `ResumeContext:269-278` skips `RequireNativeResumeBinding` and
      `BuildCouchResumeLaunchProfile` for warm; instead it re-checks that the
      detached proof still holds immediately before launch, since that proof —
      a live, client-less session — is the only precondition a reattach has.
- [ ] **Step 3:** `launchTrackedThread` spawns `pair resume <tag>` with no
      profile env and no `--layout2` when the launch is warm. Assert the argv
      and env in a test: the profile env must be **absent**, not empty, because
      `ApplyCouchLaunchProfile` distinguishes them.
- [ ] **Step 4:** `detachedResumeProofMatches` stops requiring `NativeID`, and
      `ActionableThreadInventoryContext` stops needing a binding for a detach
      candidate (already true after M1's Task 2). Run
      `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/... -count=1` — PASS.
- [ ] **Step 5:** Commit.

### Task 8b: What startup does when a warm reattach refuses

**Files:**
- Modify: `cmd/internal/couchcore/startup.go:41-60`
- Test: `cmd/internal/couchcore/startup_test.go`

`StartInteractive` returns a resume error directly (`startup.go:55-58`), so a
refusal **stops couch in that tree** — the incident the comment at
`actionableinventory.go:249-258` says the detached binding gate exists to
prevent. M2 removes that gate, so this milestone owes an answer.

**The answer is: keep no-fallback, and make the refusal actionable.** Starting a
second thread in a tree that already holds a running detached one is exactly the
confusion couch exists to prevent, so silently creating one is worse than
refusing. What was wrong was refusing *mutely*.

- [ ] **Step 1: Write the failing test** — a startup whose unique candidate is a
      detached row that cannot be reattached (its session died between
      projection and launch) exits non-zero with a message naming the thread,
      the reason, and the two ways forward (`couch --list`, or start explicitly).
- [ ] **Step 2:** Run — FAIL (today the raw `ResumeRefusal` surfaces).
- [ ] **Step 3:** Wrap the refusal at the startup seam only; `ResumeContext`
      keeps returning its typed error for every other caller.
- [ ] **Step 4:** Run `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/... -count=1` — PASS.
- [ ] **Step 5:** Commit.

### Task 9: A pending launch stops shadowing committed binding authority

**Files:**
- Modify: `cmd/internal/sessionledger/record.go:499-540` (`CurrentLaunch`)
- Test: `cmd/internal/sessionledger/record_test.go`

- [ ] **Step 1:** Add the fail-closed counterpart to Task 7's red test:

```go
// A thread that never had a binding does NOT get one invented from the legacy
// row. Fewer threads recover; that is the honest answer.
func TestNoPriorBindingStaysProvisionalRatherThanGuessing(t *testing.T) { /* … */ }
```

Whether the legacy row's `session_id` is admissible evidence is **out of scope**
— it would recover the two threads that never bound, and it could resume the
wrong transcript. Recorded as an open question below.

- [ ] **Step 2:** Run — FAIL.
- [ ] **Step 2b: a property test, because the input is not ours.** The ledger is
      append-only, written by other processes and older Pair versions, and is
      hand-editable. Two hand-written cases cannot guard that class: add a
      property test over shuffled, perforated and duplicated record sets
      asserting the invariant — *the current binding is the newest committed
      one, and no uncommitted launch changes it* (ARCH-SECURE).
- [ ] **Step 3:** Implement pending-vs-committed: a launch with no binding for
      its ordinal does not displace the newest launch that has one. Keep the
      conflict rule (`:534-538`) intact.
- [ ] **Step 4:** Run `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./cmd/... -count=1` — PASS.
      `sessionledger` is Pair's own package, so this sweep is the
      standalone-pair guard for this task.
- [ ] **Step 5:** Commit.

### Task 10: Measure the repair on the real store

- [ ] **Step 1:** Reattach `tools-couch-2` from the switcher on the real stack,
      and confirm the agent inside is the same process — a reattach, not a
      restart.
- [ ] **Step 2:** Re-run the thread-health measurement. Report how many of the
      eight `binding-lost` parks recovered: expected, the ones whose ledger
      holds an earlier `binding` row; not expected,
      `couch-05156384da12af64` and `couch-869035d630f40ce1`, which never bound.
      **Record the actual number.**
- [ ] **Step 3:** Standalone pair sanity check. `launcher/` is untouched by this
      milestone, so this is confirmation rather than a guard: `pair` and
      `pair resume <tag>` behave as before.
- [ ] **Step 4:** `env -u PAIR_SESSION_ID -u PAIR_TAG make test` — exit 0.
- [ ] **Step 5:** `sdlc milestone-close --issue 181 --milestone M2`

---

## Chunk 3: M3 — archive as the only exit

**M3 is deliberately sketched, not fully decomposed.** Its retirement rule is
written against the population that survives M2's repair, and that population is
not knowable until Task 10 measures it. Expand these tasks into full TDD steps
after Task 10, in the same shape as Chunks 1-2.

### Core concepts

Only now is retirement safe: after M2, a row that is still unusable is one whose
evidence genuinely says finished, not one the system lost.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `RetirementVerdict` | `cmd/internal/couchcore/retire.go` | new |
| `DecideRetirement` | `cmd/internal/couchcore/retire.go` | new |

```go
type RetirementVerdict struct {
	Retire bool
	Reason ThreadReason // the reason that justified retiring, empty when keeping
	Keep   string       // why it stays, for the dry-run table
}

func DecideRetirement(summary ActionableThreadSummary, evidence ThreadEvidence) RetirementVerdict
```

- **DecideRetirement** — retires only `never-started`, `invalid`, and
  `session-gone` **with the transcript confirmed absent**. Never retires
  `binding-lost` or `stale-incarnation`: those are repairable, and retiring one
  is how a recoverable session becomes unrecoverable.
  - **DRY rationale:** written over M1's reason vocabulary, so the archive rule
    cannot drift from the display rule.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ThreadStore.Archive` | `cmd/internal/couchcore/threadstore.go` | new | the records directory + manifest |
| `couch prune` | `cmd/internal/couchcmd/run.go` | new | operator invocation |

- **ThreadStore.Archive** — moves `records/<scope>/<tag>.json` to
  `archive/<scope>/<tag>.json` and drops it from the manifest. Restoring is
  therefore a move back **plus a manifest re-add**, not a bare file move:
  `Snapshot` enumerates `manifest.Threads` (`threadstore.go:504-526`), so a
  restored file the manifest does not list produces no row. The manifest is
  not a plain file write: commits go through the journaled two-phase path
  (`threadstore.go:557-660`, `:778-807`), so archive must join that path rather
  than invent a second way to mutate the manifest.
- **couch prune** — operator-invoked, prints what it would retire and why,
  confirms, then archives. **Registry question to settle in Step 1 below:**
  `readme_test.go:87-104` fails if README names a `couch <op>` whose
  presentation is not List/Show, `:145-158` requires `atlas/couch.md` to name
  every `couchcore.Operations()` entry, and `:160-186` errors on an unknown
  presentation (`ops.go:84-88`). Either declare `prune` with `PresentationList`
  and document it in both places, or keep it a non-registry CLI flag and say so
  explicitly in the plan and atlas.

Automatic retirement is deliberately **not** built: deleting state the operator
has not seen is how a recoverable session becomes an unrecoverable one, and this
issue exists because the system already did that once by hiding rows.

### Task 11: `DecideRetirement`

- [x] NOT BUILT -- superseded by the archive action (see Revisions). The
      operator decides a thread is finished; a predicate guessing for them is
      what this milestone replaced.

### Task 12: `ThreadStore.Archive` and the archive reader

- [x] **Step A:** write the record to `archive/<scope>/<tag>.json` (journaled).
- [x] **Step B:** drop it from the manifest through the existing two-phase
      commit, under the store lock. Both land in ONE journal entry, so a crash
      cannot leave a record in both sets or neither.
- [x] **Step C:** `ArchivedThreads` + `couch --archived`. Restore is documented
      as the move reversed plus a manifest re-add; not automated, because the
      operator has not needed it and an untested restore verb is worse than a
      documented file move.
- [x] **Step D (not in the plan):** quiesce the session before moving the
      record, or archiving leaves a running agent nothing tracks.

### Task 13: `couch prune`, and the Pair-artifact question

- [x] **Step 1:** Settled: `archive` is a declared operation (TUI presentation,
      confirmed) and `archived` a List one, both documented in `atlas/couch.md`.
      `couch prune` was NOT built -- see Revisions.
- [x] **Step 2:** One-time cleanup ran against the live store: 10 archived, 3
      live kept, one thread per repo remaining.
- [x] **Step 3:** **Do not touch `pair/repos/<scope>/`.** 728 files for 31 couch
      tags is the larger clutter, but it is Pair's surface with its own rules
      about what a live session still needs. File a pair-side issue carrying the
      measurement; couch does not reach into pair's directory.
- [x] **Step 4:** Atlas updated.
- [ ] **Step 5:** `sdlc close --issue 181`, and close pair#168, #171, #179 and
      #180 against their milestones with the measured evidence.

---

## Open questions, recorded rather than decided

1. **Is the legacy ledger row's `session_id` admissible as binding evidence?**
   It would recover the two threads that never bound, and where both exist it
   matches the established native id. It could also resume the wrong
   transcript. Task 9 deliberately does not decide this.
2. **Should `stale-incarnation` reconcile automatically at startup** (pair#171)
   — clearing an incarnation whose process is provably gone and re-classifying
   the row — or stay a displayed reason the operator resolves?
3. **Does the thread-health view become `couch doctor`?** The state × binding ×
   last-active table answered "why is this thread invisible" in one line. After
   M1 the switcher answers it, which may make the verb unnecessary.


## Revisions

### 2026-09-03 — M3 as shipped: an action, not a predicate

Reason: the operator's decisions replaced M3's rule with an affordance, and
building it found one thing the design had wrong.

Delta:

- **`DecideRetirement` was not built.** The plan had a predicate deciding what
  is finished, over the reason vocabulary. The operator asked for a delete
  action instead, and they are right: an operator action beats a rule guessing
  on their behalf, and the guessing is what the retirement matrix was. What
  shipped is `archive` -- a declared operation, confirmed, offered on every row
  couch is not hosting.
- **`couch prune` was not built either.** With one thread per path enforced and
  archive available per row, a bulk sweep has nothing to sweep. The one-time
  cleanup ran as a throwaway program against the real store (10 archived, 3 live
  kept) rather than becoming a permanent verb for a problem that no longer
  recurs.
- **Archive stops the session first, which the plan did not say.** Task 12
  described a record move. A record move alone leaves a running agent nothing
  tracks -- the forgotten thread couch exists to prevent -- and the one-time
  cleanup produced exactly one before this was fixed. Park cannot do the
  stopping: it needs a live incarnation, which the debris archive exists for
  does not have. `Artifacts.Quiesce` reaches it, runs FIRST, and its failure
  refuses the archive.
- **`couch --archived` exists** so a reversible decision is visible, and
  archived rows carry `ThreadArchived` rather than being classified -- running
  the classifier over them asked whether a session is alive for a thread couch
  no longer tracks, and rendered every row "checking...".
- **Labels come from directories now** (`brain`, `pair`, `arc-agi-3`), with
  colliding labels qualified by the tag's tail. Not in the plan at all; the
  operator asked for it once the honest inventory made the tag columns visible.

### 2026-09-03 — M3 review round 2: what the entity table should have said

Reason: the M3 boundary review returned REWORK on five findings, three of which
were rule-level rather than site-level.

**Entities that landed** (the plan's table named only `ThreadStore.Archive` and
`DecideRetirement`): `ThreadStore.ArchiveThread`, `ThreadStore.ArchivedThreads`,
`Couch.ArchiveThread`, `BuildArchivedInventory`, `ThreadArchived`,
`archivableRecord`, `occupiedIncarnation`, `ThreadSnapshot.Malformed`, plus the
unplanned `SelectResumableRoot`, `PathHoldsUsableThread`, `threadLabel`,
`LabelRow` / `DisambiguateLabels` / `LabelsFor`, `startupResumeRefusal`,
`confirmStillDetached`, `ResumeSessionGone`, and `trackedThreadLaunch.Warm`.

**The Spec's `invalid -> archive, inspectable` exit was NOT delivered, and is
now.** `Snapshot` raised on any undecodable record, so one corrupt file failed
the whole inventory and `ReasonInvalid` could not be produced by a real store --
a documented, labelled, Enter-explained reason that was unreachable. The
Done-when "rows == records, always, with no exceptions" was satisfied only by a
fixture that structurally could not hold an invalid record. `Snapshot` now
carries them (as `Unreadable`, renamed in the next round: "could not read" is
not the verdict "invalid"), the projectors emit them as rows, and archive moves
bytes it cannot decode.

**Three findings were rules, not sites**, and are recorded in
`workshop/lessons.md` rather than only fixed here: reversing a documented rule
sweeps every prose restatement in the same commit (7 sites stood); one fact gets
one predicate (occupancy had five sites and four definitions, and the gap killed
a session mid-start); test the seam, not both sides (`Tab -> archive` never
worked -- the switcher sent `tag`, the executor read `ref`, and a green test sat
on each side of that boundary).

### 2026-09-03 — M2: Task 9 is superseded by measurement

Reason: measuring the operator's ledgers before implementing Task 9 showed the
fix it specifies recovers 1 of the 8 lost threads, and that the effective
recovery runs through a mechanism the plan never named.

Delta:

- **Task 9 as written stays correct but nearly inert.** Seven of eight
  `binding-lost` records never had a binding row at all (`legacy → launch`), so
  "a pending launch does not shadow the last committed binding" has nothing to
  restore. Only `couch-e78b962be29c4d9a` matches the shape the task describes.
- **The missing evidence lives in the legacy row**, whose `session_id` equals
  the v2 binding root in 19 of 21 ledgers -- the same fact in an older schema,
  with the 2 exceptions detectable as multi-id.
- **The correct fix is proof migration, not trust.** `query.go:122` keeps a
  binding provisional when it has no `AuthorizationProof`, and
  `sessioninventory/proof_migration.go` already re-derives one by scanning. A
  recovery that mints authority from a legacy id without that scan asserts a
  transcript exists; running the migration PROVES it.
- **Not implemented under this milestone.** It is a change to Pair's binding
  authority, and the standing constraint says Pair's semantics are chosen, not
  drifted into. M2's headline -- reattaching a detached session -- does not
  depend on it and is landed.

### 2026-09-03 — M1 as shipped, and three claims corrected

Reason: the M1 boundary review checked the plan against the tree and found the
Core-concepts table describing a design that did not land, plus a reference
count nobody had measured.

Delta:

- **`ActionableThreadSummary` does NOT gain `Incarnations`.** The plan had the
  switcher row carry the diagnostic detail so both views could share one
  population. What shipped instead gives `ThreadSummary` its own `State` and
  `Reason` and deletes its `Parked` bool, so the two views share the
  *classifier* rather than the row shape — the switcher row stays free of
  lifecycle detail it does not render. M3's tables are written against the
  shipped shape.
- **The refactor's reach was stated without measuring it.** "36 call sites
  across six files" appeared in the plan and twice in the issue; measured at
  `c761b3cd` it is **62 references across 15 files**. It was the one line in
  the estimate's risk section a reader could check, and it was wrong.
- **`ReasonAgentUnsupported`'s value is `unsupported-agent`, not
  `agent-unsupported`.** The obvious spelling contains the `agent-` artifact
  family token, so `TestProductionArtifactReferencesAreExactlyClassified`
  flagged the file. Renaming the value beat requesting an exemption; recorded
  here because it had lived only in a commit message.
- **Entities that landed and were not in the table:** `ProofStatus`,
  `claimsLiveIncarnation`/`startInFlight`, `ThreadReason.Label`,
  `gatherThreadEvidence`, `ObserveRecordedProcesses`, `ThreadInventoryContext`,
  `BuildThreadInventory` (now takes evidence), `threadStateText`,
  `menuThreadActionable`, `unusableThreadNotice`.
- **Live proof is the union of both sources.** The plan had the console pass
  its pty children and the CLI pass OS liveness. That let one store tell two
  stories — a thread the console does not host reading `stale-incarnation`
  while `--list` called it `live`. `gatherThreadEvidence` now unions both,
  deduplicated, so the two views answer from one proof.
- **A start in flight is `busy`, not stale.** `ClassifyThread` returned
  `stale-incarnation` for any unhosted incarnation, which is the normal state
  of every thread between `start` and Pair registration. A `creating`
  incarnation whose process is still observable is `ThreadBusy`; one whose
  process is gone is stale, which is what that word should mean.
