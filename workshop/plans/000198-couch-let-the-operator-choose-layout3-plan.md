# couch --layout3 Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `couch --layout3` starts every thread in pair's three-pane layout, and couch refuses to start in a layout that would mix with sessions already running in the other one.

**Architecture:** Layout becomes a typed value with a process-wide owner (`Couch.Layout`) and a per-thread *witness* (`ThreadRecord.Layout`) recording what each thread's session actually is. A pure guard compares the requested layout against the witnesses of session-holding threads and refuses on conflict, running inside `StartInteractive` on rows it already read. The cold/warm split from #179 is untouched: the layout flag reaches argv only at a cold boundary, never at a warm reattach.

**Tech Stack:** Go; `couchcore` (domain), `couchcmd` (CLI), existing `ThreadStore` CAS persistence and `FakeThreadArtifactCollisionChecker`.

---

## Why this shape (the two corrections that drove it)

Recorded in the issue's `## Revisions`; repeated here because they invert the Spec's mechanism.

**Layout is couch-global, by operator decision.** Not per-thread. `couch --layout3`
applies to every thread couch starts or cold-resumes. Mixed layouts are refused
rather than supported, because the operator wants the layout predictable.

**`StartArgs` cannot carry it.** `StartArgs` is not the durable record its doc
comment implies. What persists is `ActorRecord.Args` in the Registry, which
documents itself as *"a transitional display/handle cache. It decides nothing:
ThreadStore is the durable authority"* (`registry.go:17-23`), and a parked thread
has no ActorRecord at all. `resume.go:445` **rebuilds** `StartArgs` from
`ThreadRecord` + `LatestLaunchProfile`. A `StartArgs.Layout` field would be
discarded by precisely the park/resume cycle it needed to survive.

**The guard is a predictability feature, not a safety mechanism.** Safety already
exists and is not being extended: #179's rule that a warm reattach sends *no*
layout flag is what prevents pair's conflict path from offering to delete a live
session. That holds whether or not the guard runs. So a guard that races a
session appearing between the inventory read and a reattach is **cosmetic, not
destructive**, which is why a plain startup check suffices and nothing here needs
to be transactional with the store. Do not weaken the #179 invariant to make the
guard stronger — that would trade a real safety property for a cosmetic one.

## Non-goals

Deliberately not built, though each is reachable from this design:

- **A menu affordance for layout.** The issue's Spec left the operator surface
  open ("a CLI flag at start, a menu affordance on the thread, or both"); the
  Revisions narrowed it to the flag. A menu entry would imply per-thread choice,
  which is the fork the operator rejected.
- **Mid-session layout switching.** `Couch.Layout` is immutable for the process
  lifetime. This is what collapses the ordering problem (see ARCH-ORDER).
- **A per-thread override.** Same rejected fork.
- **Migrating a live session between layouts.** The only migration is a cold
  resume, which is a fresh session. Nothing rewrites a running one.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Layout` | `cmd/internal/couchcore/layout.go` | new |
| `ParseLayout` / `NormalizeLayout` | `cmd/internal/couchcore/layout.go` | new |
| `LayoutConflict` | `cmd/internal/couchcore/layout.go` | new |
| `ResolveLayoutConflicts` | `cmd/internal/couchcore/layout.go` | new |
| `ThreadRecord.Layout` | `cmd/internal/couchcore/thread.go` | modified |
| `ActionableThreadSummary.Layout` | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `StartEvent.Layout` | `cmd/internal/couchcore/starttransaction.go` | modified |
| `cliInvocation.layout` (+ `ParseCLI`) | `cmd/internal/couchcmd/cli.go` | modified |
| `Couch.Layout` | `cmd/internal/couchcore/couch.go` | modified |

- **Layout** — a typed layout choice (`layout2` / `layout3`, plus the
  `LayoutUnknown` sentinel below) with an explicit parse from both the CLI flag
  and the persisted string.
  - **Relationships:** 1 per couch process (`Couch.Layout`); 1 per thread
    (`ThreadRecord.Layout`) as a witness of what that thread's session is.
  - **DRY rationale:** replaces the string literal `"--layout2"` at
    `launch_existing.go:50` and stops a second literal appearing for layout3.
    One place converts a Layout to argv; nothing else formats it.
  - **Future extensions:** a third layout is one enum value plus one flag arm.

- **The normalization boundary (ARCH-SECURE).** The persisted field is untrusted
  input — absent (written before this change) or hand-edited. There are exactly
  **two** places a raw layout string becomes a decision input, and both must
  normalize. This enumeration is the deliverable, not just the first site:

  | raw source | normalizer | on unrecognised value |
  |---|---|---|
  | `ThreadRecord.Layout` → row, in `ProjectActionableThreads` | `NormalizeLayout` | `LayoutUnknown` (total; the projection has no error return) |
  | CLI flag → `Couch.Layout`, in `ParseCLI` | `ParseLayout` | error, exit 2 |

  Absent parses to `Layout2`, which is the truth for every record written before
  this change — couch pinned layout2 from 2026-08-22 until now. An unrecognised
  value never silently becomes layout2: `LayoutUnknown` **conflicts with every
  requested layout**, so the guard refuses and names the thread. The failure is
  visible rather than a fabricated value the guard would then trust.
  `LayoutUnknown` can never reach `Flag()`, because `Couch.Layout` is only ever
  set from `ParseLayout`, which errors instead of returning it.

- **LayoutConflict** — one thread whose session disagrees with the requested
  layout. Carries address, layout, and state.
  - **DRY rationale:** the refusal message and any future affordance both render
    from this rather than re-deriving "which threads block".

- **ResolveLayoutConflicts(requested Layout, rows []ActionableThreadSummary) []LayoutConflict**
  — pure; the whole guard decision.
  - **ARCH-PURE:** takes rows and a Layout, returns conflicts. No IO, no clock;
    unit-tested with hand-built summaries and no fake.
  - **One rule, not a list of special cases: a thread blocks iff it currently
    holds a zellij session whose layout couch cannot change.** All six values of
    `ActionableThreadState` (`actionableinventory.go:19-37`) are dispositioned:

  | state | holds a session? | blocks? | why |
  |---|---|---|---|
  | `ThreadLive` | yes | **yes** | couch would reattach into a session in the other layout |
  | `ThreadDetached` | yes | **yes** | same; warm reattach cannot change its layout (#179) |
  | `ThreadBusy` | yes — park in flight | **yes** | the session is alive *now*, and a park that fails leaves it in the old layout. Blocking is the conservative arm and the remedy is to wait: the comment at `:22-24` says it resolves on its own |
  | `ThreadParked` | no — park ended it | no | its next cold resume takes couch's layout freely. **This is what makes "park them first" the reachable remedy** |
  | `ThreadUnusable` | not actionable | no | couch will not reattach it, so it cannot produce a mixed view |
  | `ThreadArchived` | out of the working set | no | `ClassifyThread` never returns it (`:29-32`); only the archive listing sets it |

  - **Not `Resumable()`.** That helper is `Parked || Detached` and is the wrong
    predicate here: it would refuse startups that are safe (parked) while the
    rule we want is about holding a session.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| layout argv emission | `cmd/internal/couchcore/launch_existing.go:50` | modified | the `pair` child process |
| witness write | `cmd/internal/couchcore/starttransaction.go:74-83` | modified | ThreadStore CAS |
| guard invocation | `cmd/internal/couchcore/startup.go:132` | modified | startup refusal |
| layout plumbing | `cmd/internal/couchcmd/run.go:252` | modified | CLI → domain |

- **layout argv emission** — `argv` takes the flag from `c.Layout` at a cold
  boundary only. The `in.Warm` branch stays byte-identical.
  - **ARCH-MOCK:** `pair` is already behind the `Runner` seam with a fake that
    records argv, so assertions run against recorded argv and no new external
    double is needed. `pair resume <tag> --layout3` is already confirmed to parse
    (`couch.go:433-439`, measured).

- **witness write** — rides the **existing** `StartRegistered` transaction.
  `StartEvent` already carries `Profile *LaunchProfile` (`starttransaction.go:35`),
  which is the precedent: launch shape is recorded inside the start transaction.
  Add `Layout Layout` beside it, applied in `AdvanceStartTransaction`'s
  `StartRegistered` case (`:74-83`), committed by the CAS that
  `launch_existing.go:124` already performs.
  - **This is why there is no revision-conflict path to design:** there is no
    second write. A separate `UpdateExistingThread` (`threadstore.go:262`) would
    need an `expectedRevision` that `AdvanceStart` has just moved, which is the
    conflict PQ-4 identified — avoided rather than handled (ARCH-DRY: one
    transaction already exists for "this thread just started with this shape").

- **guard invocation** — inside `StartInteractive` (`startup.go:123`), placed
  immediately after the `rows, err := c.ActionableThreadInventoryContext(ctx, nil)`
  at `:132` and before `SelectResumableRoot` — i.e. before any effect.
  - **Why not in `couchcmd`:** `RunWithRuntime`'s `cliLaunch` branch is
    `run.go:159-166`, but the Couch is not built until `:252`, and the only
    startup inventory read is the one inside `StartInteractive`. A guard in the
    CLI branch would have no Couch and would have to enumerate sessions a second
    time — which the ARCH-CONSTRAINTS budget below declares wrong by
    construction.

- **layout plumbing** — `Couch.Layout` is an exported field defaulted to
  `Layout2` in `New`'s result literal (`couch.go:99`), set by the CLI right after
  `rt.NewCouchWith(runner, namespace)` returns (`run.go:252`). The layout travels
  from `ParseCLI` as a **typed parameter** through `runTypedOperation` →
  `runTypedOperationWithConsole`, not smuggled through the `map[string]string`
  operation args.
  - **No `Runtime` interface change**, so `testRT` (`run_test.go:96`) and
    `NewCouchWith`'s signature (`run.go:36-39`) are untouched. Defaulting in
    `New` is what keeps `Layout("")` — the zero value, which would format as
    `"--"` — unreachable at `Flag()`.

## ARCH-CONSTRAINTS — operating envelope

- **Interaction path:** couch startup (one-shot), not a keystroke path.
- **Added cost budget:** **zero additional session enumeration.** The guard
  consumes the rows `StartInteractive` already reads at `startup.go:132`. Basis:
  measured — that projection already resolves detached sessions via
  `DetachedSessionResolver`. Exceeded → the guard is doing its own IO and is
  wrong by construction. (This budget is what settled the guard's placement.)
- **Scale:** thread counts are single-digit to low tens (6 today). A linear scan
  over rows is correct; nothing needs an index.
- **Overload behavior / latency:** N/A — no concurrency or fan-out is
  introduced; the guard is a scan of an in-memory slice.

## ARCH-ORDER — states, events, and the ones the caller cannot block

`Couch.Layout` is **immutable for the process lifetime**, fixed at construction
from argv. That is the most important ordering property here: there is no
mid-session layout change, so the mixed-state question reduces to what is on disk
at startup.

State carried between events: each thread's `Layout` witness and its
`ActionableThreadState`.

| event | state | -> effect |
|---|---|---|
| startup, requested L | any session-holding thread with layout ≠ L | refuse, name the threads, exit non-zero |
| startup, requested L | no such thread | run; `Couch.Layout = L` |
| start new thread | — | argv gets L's flag; witness := L in the StartRegistered CAS |
| cold resume (parked) | witness may be ≠ L | argv gets L's flag; **witness := L** in the same CAS |
| warm reattach (detached) | witness normally == L | **no layout flag at all** (#179, unchanged) |
| park | session ends | thread leaves the blocking set; witness left as-is |

Events the caller cannot block, targeted rather than swept:

- **Process death between the child launching and the CAS committing.** The
  session exists in layout L; the record still says the old layout. The next
  startup in L is refused, citing a thread that is actually already L. Chosen
  resolution: **ignore** — no rollback. It is self-correcting (park or re-resume)
  and it fails toward the *conservative* answer, refusing rather than mixing.
  Recording the witness before the launch would invert this into the dangerous
  direction, so the order is deliberate: **launch, then record.**
- **A failed park on a `ThreadBusy` thread.** It returns to its old layout with
  its session alive. Handled by blocking on `ThreadBusy` in the table above,
  rather than by assuming the park succeeds.
- **A session appearing between the guard's read and a reattach.** Cosmetic
  only — #179 keeps the warm path safe independently. **The warm-reattach row
  above therefore claims no guarantee from the guard**: the guard makes the
  mismatch rare, #179 makes it harmless. An earlier draft of this plan asserted
  the witness "== L, guaranteed by the guard", which is false — the guard is a
  startup-time snapshot, not an invariant.
- **A second couch on the same store.** Does not apply: couch is a singleton per
  namespace, which is what makes a startup-time read sufficient.
- **Out-of-order completions / retries:** N/A — the guard holds no state between
  events; it is a pure function of one inventory read.

Nondeterminism enters only through the inventory's view of live zellij sessions,
which `FakeThreadArtifactCollisionChecker` already controls; a failing ordering
is reproduced by seeding it with the detached set.

## ARCH-PURPOSE — what "done" includes

The purpose is a *predictable* layout choice, so the deliverable is the class,
not the flag: the doc sites and test premises encoding "couch pins layout2" are
part of this change, not follow-up. The enumerations are written out — the
normalization table above, the six-state table above, and the file lists in
Tasks 7 and 8 — rather than left as "sweep the docs".

---

## Task 1: The Layout type and its two normalizers

**Files:**
- Create: `cmd/internal/couchcore/layout.go`, `cmd/internal/couchcore/layout_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestParseLayoutAcceptsKnownValues(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Layout
	}{{"layout2", Layout2}, {"layout3", Layout3}} {
		got, err := ParseLayout(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("ParseLayout(%q) = %v, %v; want %v, nil", tc.in, got, err, tc.want)
		}
	}
}

// An absent field is every record written before this change, and those are
// layout2 with certainty -- couch pinned it from 2026-08-22 until #198.
func TestParseLayoutTreatsEmptyAsLayout2(t *testing.T) {
	got, err := ParseLayout("")
	if err != nil || got != Layout2 {
		t.Fatalf("ParseLayout(\"\") = %v, %v; want Layout2, nil", got, err)
	}
}

// ARCH-SECURE: a hand-edited or newer-version value must not coerce to a
// plausible default the guard would then trust.
func TestParseLayoutRejectsUnknown(t *testing.T) {
	if _, err := ParseLayout("layout9"); err == nil {
		t.Fatal("ParseLayout(\"layout9\") = nil error; want a refusal")
	}
}

// NormalizeLayout is the total variant for the row projection, which has no
// error return.
func TestNormalizeLayoutIsTotal(t *testing.T) {
	if got := NormalizeLayout(""); got != Layout2 {
		t.Fatalf("NormalizeLayout(\"\") = %v; want Layout2", got)
	}
	if got := NormalizeLayout("layout9"); got != LayoutUnknown {
		t.Fatalf("NormalizeLayout(\"layout9\") = %v; want LayoutUnknown", got)
	}
}

func TestLayoutFlagIsTheOnlyFormatter(t *testing.T) {
	if Layout2.Flag() != "--layout2" || Layout3.Flag() != "--layout3" {
		t.Fatalf("flags = %q, %q", Layout2.Flag(), Layout3.Flag())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/internal/couchcore/ -run "Layout" -v`
Expected: FAIL — undefined: ParseLayout

- [ ] **Step 3: Implement**

```go
package couchcore

import "fmt"

// Layout is which pair layout couch launches its threads in. It is chosen once
// per couch process (Couch.Layout) and recorded per thread
// (ThreadRecord.Layout) as a witness of what that thread's session actually is.
type Layout string

const (
	Layout2 Layout = "layout2"
	Layout3 Layout = "layout3"
	// LayoutUnknown is a persisted value this binary does not recognise. It is
	// never chosen and never formatted: it exists so a row can carry "we cannot
	// prove this session's layout" instead of a fabricated layout2 the guard
	// would trust. It conflicts with every requested layout.
	LayoutUnknown Layout = "unknown"
)

// ParseLayout turns an untrusted string -- a CLI flag or a persisted field --
// into a Layout. Empty means a record written before layout was recorded, and
// those are layout2 with certainty: couch pinned layout2 from 2026-08-22 until
// #198. An unrecognised value is refused rather than defaulted.
func ParseLayout(raw string) (Layout, error) {
	switch Layout(raw) {
	case "":
		return Layout2, nil
	case Layout2:
		return Layout2, nil
	case Layout3:
		return Layout3, nil
	}
	return "", fmt.Errorf("unknown layout %q", raw)
}

// NormalizeLayout is ParseLayout for the one caller that cannot return an
// error: ProjectActionableThreads. An unrecognised value becomes LayoutUnknown
// rather than a default, so the guard refuses visibly instead of proceeding on
// a value nothing verified.
func NormalizeLayout(raw string) Layout {
	layout, err := ParseLayout(raw)
	if err != nil {
		return LayoutUnknown
	}
	return layout
}

// Flag is the sole place a Layout becomes argv. LayoutUnknown never reaches it:
// Couch.Layout is only ever set from ParseLayout, which errors instead of
// returning it.
func (l Layout) Flag() string { return "--" + string(l) }
```

- [ ] **Step 4: Verify green.** `go test ./cmd/internal/couchcore/ -run "Layout" -v`
- [ ] **Step 5: Commit** — `#198: couch: introduce the Layout type and its normalizers`

## Task 2: The guard as a pure function

**Files:**
- Modify: `cmd/internal/couchcore/layout.go`, `cmd/internal/couchcore/actionableinventory.go`
- Test: `cmd/internal/couchcore/layout_test.go`

- [ ] **Step 1: Write the failing tests** — one per row of the six-state table,
      so the disposition is pinned rather than asserted in prose:

```go
func summary(tag string, state ActionableThreadState, layout Layout) ActionableThreadSummary {
	return ActionableThreadSummary{
		Address: ThreadAddress{RepoScope: "scope", Tag: PairTag(tag)},
		State:   state, Layout: layout,
	}
}

func TestBlockingSetIsExactlyTheSessionHoldingStates(t *testing.T) {
	for _, tc := range []struct {
		state ActionableThreadState
		block bool
	}{
		{ThreadLive, true}, {ThreadDetached, true}, {ThreadBusy, true},
		{ThreadParked, false}, {ThreadUnusable, false}, {ThreadArchived, false},
	} {
		rows := []ActionableThreadSummary{summary("a", tc.state, Layout2)}
		got := ResolveLayoutConflicts(Layout3, rows)
		if (len(got) == 1) != tc.block {
			t.Fatalf("state %s: conflicts %+v; want block=%v", tc.state, got, tc.block)
		}
	}
}

func TestMatchingLayoutDoesNotConflict(t *testing.T) {
	rows := []ActionableThreadSummary{summary("a", ThreadDetached, Layout3)}
	if got := ResolveLayoutConflicts(Layout3, rows); len(got) != 0 {
		t.Fatalf("same layout blocked startup: %+v", got)
	}
}

// LayoutUnknown cannot be proved to agree, so it conflicts with both.
func TestUnknownLayoutConflictsWithEveryRequest(t *testing.T) {
	for _, requested := range []Layout{Layout2, Layout3} {
		rows := []ActionableThreadSummary{summary("a", ThreadDetached, LayoutUnknown)}
		if got := ResolveLayoutConflicts(requested, rows); len(got) != 1 {
			t.Fatalf("requested %v: got %+v; want one conflict", requested, got)
		}
	}
}

// Pins the predicate against the wrong-but-tempting Resumable()
// (Parked||Detached): swapping to it makes the parked row block and this fails.
func TestBlockingSetIsNotResumable(t *testing.T) {
	rows := []ActionableThreadSummary{
		summary("parked", ThreadParked, Layout2),
		summary("detached", ThreadDetached, Layout2),
	}
	got := ResolveLayoutConflicts(Layout3, rows)
	if len(got) != 1 || got[0].Address.Tag != PairTag("detached") {
		t.Fatalf("got %+v; want exactly the detached thread", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails** — undefined `ResolveLayoutConflicts`;
      `Layout` not a field of `ActionableThreadSummary`.

- [ ] **Step 3: Implement.** Add `Layout Layout \`json:"layout,omitempty"\`` to
      `ActionableThreadSummary` (`actionableinventory.go:110`), then:

```go
// LayoutConflict is one thread whose existing session disagrees with the
// layout couch was asked to start in.
type LayoutConflict struct {
	Address ThreadAddress
	Layout  Layout
	State   ActionableThreadState
}

// holdsSession reports the states whose zellij session is alive right now, and
// whose layout couch therefore cannot change: asking a live session for a
// different layout sends pair down the conflict path that offers to DELETE it
// (#179). Busy is included because a park in flight can still fail, leaving the
// session alive in its old layout.
//
// Parked is excluded deliberately -- park ends the session via the lifecycle
// quit protocol, so a parked thread's next cold resume takes couch's layout
// freely. That exclusion is what makes "park them first" the reachable remedy
// for a refusal.
func (s ActionableThreadSummary) holdsSession() bool {
	return s.State == ThreadLive || s.State == ThreadDetached || s.State == ThreadBusy
}

// ResolveLayoutConflicts reports the threads that stop couch starting in
// `requested`. LayoutUnknown conflicts with everything: it means the record
// carried a value this binary cannot read, so agreement cannot be proved.
func ResolveLayoutConflicts(requested Layout, rows []ActionableThreadSummary) []LayoutConflict {
	var conflicts []LayoutConflict
	for _, row := range rows {
		if !row.holdsSession() || row.Layout == requested {
			continue
		}
		conflicts = append(conflicts, LayoutConflict{
			Address: row.Address, Layout: row.Layout, State: row.State,
		})
	}
	return conflicts
}
```

- [ ] **Step 4: Verify green.** `go test ./cmd/internal/couchcore/ -run "Layout|Blocking" -v`
- [ ] **Step 5: Commit** — `#198: couch: resolve layout conflicts as a pure function`

## Task 3: Persist the witness, normalized at the projection

**Files:**
- Modify: `cmd/internal/couchcore/thread.go`, `cmd/internal/threadrecord/record.go`,
  `cmd/internal/couchcore/actionableinventory.go`
- Test: `cmd/internal/couchcore/thread_test.go`, `actionableinventory_test.go`

- [ ] **Step 1: Write the failing tests.** The second is the Critical from the
      plan gate (PQ-1) — without it, every pre-change record blocks a default
      startup:

```go
// Records written before this change are schema 2 with no layout field.
func TestThreadRecordWithoutLayoutRoundTrips(t *testing.T) {
	var record ThreadRecord
	raw := `{"schema_version":2,"address":{"repo_scope":"s","tag":"t"}}`
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		t.Fatal(err)
	}
	if record.Layout != "" {
		t.Fatalf("layout = %q; want empty", record.Layout)
	}
}

// PQ-1: the projection must normalize, or Layout("") != Layout2 makes every
// pre-change thread a conflict against a bare `couch`.
func TestProjectionNormalizesAbsentLayoutToLayout2(t *testing.T) {
	rows := ProjectActionableThreads(/* one record, no layout field, detached */)
	if rows[0].Layout != Layout2 {
		t.Fatalf("row layout = %q; want Layout2", rows[0].Layout)
	}
	if got := ResolveLayoutConflicts(Layout2, rows); len(got) != 0 {
		t.Fatalf("pre-change record blocked a default startup: %+v", got)
	}
}

func TestProjectionMarksUnreadableLayoutUnknown(t *testing.T) {
	// record with Layout "layout9" -> row.Layout == LayoutUnknown
}
```

- [ ] **Step 2: Run to verify they fail.**

- [ ] **Step 3: Implement.** Add to `ThreadRecord` (`thread.go:54`):

```go
	// Layout witnesses what this thread's pair session actually is, so the
	// startup guard can refuse a couch layout that would mix with it. Absent on
	// every record written before #198, and those are layout2: couch pinned
	// layout2 from 2026-08-22 until then.
	Layout Layout `json:"layout,omitempty"`
```

Mirror it in `threadrecord.Record` and both conversions (`toPersistedThreadRecord`
at `thread.go:90` and the inbound one). In `ProjectActionableThreads`
(`actionableinventory.go:209-221`) set `Layout: NormalizeLayout(string(record.Layout))`
on the row — **this is the normalization point**. The `input.Unreadable` rows
(`:203-206`) get no layout and are `ThreadUnusable`, which does not block.

**No schema version bump.** `threadrecord.SchemaVersion` is 2 and `Validate`
hard-refuses a mismatch (`record.go:108-109`), so bumping would refuse to load
every existing record. The field is additive and `omitempty`: an older binary
ignores it, a newer one reads its absence as layout2 — which is true.

- [ ] **Step 4: Verify green.** `go test ./cmd/internal/couchcore/ ./cmd/internal/threadrecord/ -v`
- [ ] **Step 5: Commit** — `#198: couch: record and normalize each thread's layout witness`

## Task 4: Emit the flag at cold boundaries, record it in the same CAS

**Files:**
- Modify: `cmd/internal/couchcore/couch.go`, `launch_existing.go`, `starttransaction.go`
- Test: `couch_test.go`; `warmresume_test.go` must pass **unmodified**

- [ ] **Step 1: Write the failing tests**

```go
func TestColdStartSendsCouchLayout(t *testing.T) {
	// Couch{Layout: Layout3} -> recorded argv {"pair","resume",<tag>,"--layout3"}
}

// #179 restated at the new mechanism: the warm branch sends no layout flag
// whatever Couch.Layout is.
func TestWarmReattachSendsNoLayoutEvenInLayout3(t *testing.T) {
	// Couch{Layout: Layout3}, warm -> argv exactly {"pair","resume",<tag>}
}

func TestColdResumeRewritesTheWitnessInOneTransaction(t *testing.T) {
	// parked thread recorded layout2, Couch{Layout: Layout3}, cold resume
	// -> record.Layout == Layout3, and Revision advanced exactly once
}

func TestDefaultCouchIsLayout2(t *testing.T) {
	// New(...) -> c.Layout == Layout2, so Flag() never yields "--"
}
```

- [ ] **Step 2: Run to verify they fail.**

- [ ] **Step 3: Implement.**
  - Add `Layout Layout` to `Couch` beside `RootAgent` (`couch.go:35`), set to
    `Layout2` in the result literal at `couch.go:99`.
  - `launch_existing.go:50`:

    ```go
    argv := []string{"pair", "resume", string(thread.Address.Tag), c.Layout.Flag()}
    if in.Warm {
        argv = []string{"pair", "resume", string(thread.Address.Tag)}
    }
    ```

  - Add `Layout Layout` to `StartEvent` (`starttransaction.go:30`, beside
    `Profile`), applied in the `StartRegistered` case (`:74-83`). Pass
    `Layout: c.Layout` on the `StartRegistered` `AdvanceStart` at
    `launch_existing.go:124` — **only when `!in.Warm`**, since a warm reattach
    did not choose a layout and must not overwrite the witness. Note this is
    **not** the `StartHelperRecorded` advance at `:82`: attaching the layout
    there while applying it in the `StartRegistered` case would leave the
    witness silently never written.
  - Order is **launch, then record** (ARCH-ORDER); the CAS already runs after
    the child is up, so no code moves.

- [ ] **Step 4: Verify green + the untouched #179 pin.**
      `go test ./cmd/internal/couchcore/ -v`, then confirm
      `git diff --stat cmd/internal/couchcore/warmresume_test.go` is empty.
- [ ] **Step 5: Commit** — `#198: couch: launch threads in the couch-wide layout`

## Task 5: The startup guard

**Files:**
- Modify: `cmd/internal/couchcore/startup.go`
- Test: `cmd/internal/couchcore/startup_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestStartInteractiveRefusesOnLayoutConflict(t *testing.T) {
	// fake seeded with a detached layout2 thread; Couch{Layout: Layout3}
	// -> error names the thread AND says to park it; NO child was started
}

func TestStartInteractiveProceedsWhenOnlyParkedThreadsDisagree(t *testing.T) {
	// parked layout2 thread; Couch{Layout: Layout3} -> starts normally
}

// The budget from ARCH-CONSTRAINTS, pinned: the guard must add no enumeration.
func TestGuardAddsNoSessionEnumeration(t *testing.T) {
	// count DetachedSessions calls on the fake with and without a conflict;
	// both equal the pre-change count
}
```

- [ ] **Step 2: Run to verify they fail.**

- [ ] **Step 3: Implement.** In `StartInteractive` (`startup.go:123`), directly
      after `rows, err := c.ActionableThreadInventoryContext(ctx, nil)` at `:132`
      and before `SelectResumableRoot`:

```go
	if conflicts := ResolveLayoutConflicts(c.Layout, rows); len(conflicts) > 0 {
		return StartResult{}, layoutConflictRefusal(c.Layout, conflicts)
	}
```

`layoutConflictRefusal` follows the shape `startupResumeRefusal` established at
`startup.go:152` — say which threads, what happened, and the next step:

```
couch: cannot start in layout3 -- 2 threads hold layout2 sessions:
  brain  (detached)
  pair   (live)
park them first, then start couch in layout3:
  couch --layout2     # park them from the switcher, then quit
```

- [ ] **Step 4: Verify green.** `go test ./cmd/internal/couchcore/ -v`
- [ ] **Step 5: Commit** — `#198: couch: refuse a startup that would mix layouts`

## Task 6: The CLI flag

**Files:**
- Modify: `cmd/internal/couchcmd/cli.go`, `run.go`
- Test: `cli_test.go`, `run_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func TestParseCLIAcceptsLayoutFlag(t *testing.T) {
	// "--layout3"              -> cliLaunch, path ".",          Layout3
	// "--layout3 /some/path"   -> cliLaunch, path "/some/path", Layout3
	// "/some/path --layout3"   -> cliLaunch, path "/some/path", Layout3
	// ""                       -> cliLaunch, path ".",          Layout2
}

func TestParseCLIRejectsBothLayouts(t *testing.T)        { /* "--layout2 --layout3" -> error */ }
func TestParseCLIRejectsLayoutOnNonLaunchForms(t *testing.T) { /* "--list --layout3" -> error */ }
func TestLayoutReachesTheCouch(t *testing.T)             { /* "--layout3" -> c.Layout == Layout3 */ }
```

- [ ] **Step 2: Run to verify they fail.**

- [ ] **Step 3: Implement.** In `ParseCLI`, strip layout flags **before** the
      existing shape checks — mirroring `launcher.ParseArgs`, which runs
      `extractLayoutRequest` first (`args.go:51`) — so the "path cannot be
      combined with other arguments" guard at `cli.go:87-88` still sees a bare path.
      Reject the flag on `--list` / `--archived` / `--show` / `--internal`.
      Carry it as `cliInvocation.layout Layout`, pass it as a typed parameter
      through `runTypedOperation` → `runTypedOperationWithConsole`, and set
      `c.Layout` immediately after `rt.NewCouchWith` at `run.go:252`. The
      `Runtime` interface is unchanged, so `testRT` (`run_test.go:96`) needs no
      edit. Update `usage()`.

- [ ] **Step 4: Verify green.** `go test ./cmd/internal/couchcmd/ ./cmd/couch/ -v`
- [ ] **Step 5: Commit** — `#198: couch: --layout3 flag`

## Task 7: Correct the test premises

**The enumeration (~18 references, counted in the issue):**

| file | refs | note |
|---|---|---|
| `cmd/internal/couchcore/couch_test.go` | 5 | incl. `:1666` "Layout pinned to layout2 (operator decision 2026-08-22)" |
| `cmd/internal/couchcmd/run_test.go` | 5 | |
| `cmd/couch/main_test.go` | 3 | incl. `:154` "want **exactly** `pair resume <tag> --layout2`" |
| `cmd/internal/couchcore/runner_test.go` | 3 | generic argv fixtures, not pins |
| `cmd/internal/couchcore/resume_launch_test.go` | 1 | |
| `cmd/internal/couchcore/warmresume_test.go` | 1 | **must not change** |

- [ ] **Step 1:** Rewrite each premise from "couch pins layout2" to "couch sends
      its process-wide layout at a cold boundary and none at a warm one". Where a
      test asserts `--layout2`, keep the assertion — it is still correct for a
      default-constructed couch; the *comment* is what is wrong.
- [ ] **Step 2:** `git diff --stat cmd/internal/couchcore/warmresume_test.go` is empty.
- [ ] **Step 3:** `grep -rn "pinned to layout2\|operator decision 2026-08-22" cmd/`
      returns nothing.
- [ ] **Step 4:** `env -u PAIR_SESSION_ID -u PAIR_TAG make test`
- [ ] **Step 5:** Commit.

## Task 8: Sweep the rationale and the atlas

- [ ] **Step 1:** Replace `cmd/internal/couchcore/couch.go:427-431`. Record the
      pin as **reversed on 2026-09-06 with its reason**: its rationale ("couch
      owns terminal switching, so layout3's third pane is the layer couch
      replaces") was an actor-cluster-era claim that #170's rescope to couch-lite
      invalidated — couch-lite switches agent sessions and never took over that
      layer. **Keep** the measured correction about `resume` accepting layout
      flags (`:433-439`); it is still true and now load-bearing.
- [ ] **Step 2:** Update `atlas/couch.md` at `:201`, `:815`, `:880`, `:903-904`
      to the new rule: couch-wide layout, cold-boundary only, mixed layouts
      refused at startup.
- [ ] **Step 3:** `grep -rn "layout2" atlas/` — every remaining hit describes the
      default, not a pin.
- [ ] **Step 4:** Commit.

## Task 9: Backfill the existing threads

- [ ] **Step 1:** `couch --list` and record the thread count and states in `## Log`.
- [ ] **Step 2:** No migration script: an absent field normalizes to layout2,
      which is true for all of them, and Task 4 rewrites each witness on its next
      cold resume. Confirm by starting bare `couch` and observing it does not refuse.

## Task 10: Manual verification (the Done-when the tests cannot reach)

- [ ] **Step 1:** `couch --layout3` from a clean state → a thread starts with
      pair's right-hand terminal present.
- [ ] **Step 2:** Park it, `couch --layout3`, resume → still layout3.
- [ ] **Step 3:** With that thread **detached**, `couch --layout2` → refused,
      naming the thread and the remedy.
- [ ] **Step 4:** Park it, `couch --layout2` → starts; the thread cold-resumes
      into layout2, and `couch --layout3` is now refused for the mirrored reason.
- [ ] **Step 5:** Throughout: pair **never** offers to delete a live session.
      That is the #179 invariant and the one destructive failure mode — if it
      appears, stop and re-plan.
- [ ] **Step 6:** Record in `## Log` what was actually observed, with the argv and
      the refusal text — not "it worked" (a prior round was flagged for exactly that).
