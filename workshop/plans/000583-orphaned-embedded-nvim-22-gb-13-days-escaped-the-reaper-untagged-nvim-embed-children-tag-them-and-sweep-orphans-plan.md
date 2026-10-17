# Orphan nvim reaper Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every embedded nvim pair spawns is tagged and reaped when its owner dies, and an untagged `nvim --embed` orphan in a managed worktree is reported by Couch and reaped only on confirmation, matched by exact identity.

**Architecture:** Two layers. (1) pair's existing tag-scoped reaper stops killing by name: one closed kind set in `artifactpath` feeds every consumer, and each kill is decided by a pure function over evidence read in one process-table snapshot (pid + kernel start identity + argv) and re-fenced on identity immediately before the signal. (2) Couch's #587 sampler gains one more classification — an untagged `nvim --embed` whose parent is launchd, whose cwd is inside a root the supervisor already knows, and which is older than a threshold — reported through the existing resource health line and alert episode, and reaped only through a new `--confirm`-gated operation.

**Tech Stack:** Go (`cmd/internal/{artifactpath,launcher,procutil,couchcore,couchcmd,opener,reviewcmd}`), Lua (`nvim/*.lua`), Darwin `sysctl`/`libproc` through the existing cgo block, `zellij` layout KDL untouched.

---

## Root cause (identified, per Done-when #2)

`nvim FILE` forks into a TUI parent plus an `nvim --embed` server child — documented in pair's own
`nvim/init.lua:16-23`. When zellij kills the pane the TUI exits, but the embed can survive the RPC-pipe
EOF and reparents to launchd (ppid 1), inheriting the pane's cwd. Live evidence on this machine
(2026-10-16): `nvim --embed -u .../pair/nvim/init.lua .../draft-1-pair-6.md` (tagged, reaped) alongside
bare `nvim --embed` rows whose cwd is `worktree/ariadne-slot2/ariadne` — the 22 GB orphan's exact cwd —
with **live** parents, i.e. live editors that must never be signalled.

Two sub-classes, two different fixes:

| Class | Why it escaped | Fix |
|---|---|---|
| pair-spawned, untagged or unreaped | the `changelog` viewer passes no `PAIR_NVIM_PID_FILE` (`opener/run.go:211-217`) and `nvim/changelog.lua` has no pidfile block, so its embed leaves no identity at all; `review` writes one (`review.lua:791-792`) but `"review"` is absent from the reap/sweep kind lists (`osruntime.go:947,973`, `lifecycle.go:195,297`) | M1: close the sidecar kind set to `{draft, scrollback, review}`, give the changelog viewer an artifact-keyed pid record, and reap every one of them by verified identity |
| not pair-spawned (a bare interactive `nvim` typed in a pane) | pair never spawned it, so it can carry no pair tag; argv is bare (live pid 38555: `nvim --embed`, no arguments), so no scope path exists to match | M2: detect by ppid 1 + managed cwd + age, report, reap on confirmation |

**Why the changelog surface gets its own record shape.** `nvim-pid-<tag>-<kind>` is keyed by (tag, kind),
which is unique for draft, scrollback and review but not for changelog: a changelog viewer is per
(tag, agent, session) (`ChangelogSessionChecked`, `paths.go:738-749`), so two concurrent viewers of one
tag would overwrite one sidecar and leak the first viewer's pid. The tag also cannot be recovered from
the changelog *filename* or from argv: `componentPattern` admits `-` in tag, agent and session
(`paths.go:14`) and the session component is optional, so `changelog-<tag>-<agent>[-<session>]` has no
unambiguous split, and a `changelog-<tag>-*` glob crosses tags (`work` claims `work-x`'s file). Both of
the plan-quality gate's options fail on that. The changelog record is therefore keyed by the **artifact
base it already has** — `<base>.nvim-pid`, a sibling of `<base>.openlock` — which is unique by
construction and needs no parsing: the reaper enumerates the pair, reads the opener pid from the
openlock for owner liveness, and requires the embed's argv to carry `<base>.md` exactly.

pair cannot make nvim's forked embed die with its TUI — that is upstream behaviour. What pair owns is
(a) tagging and reaping everything it spawns, and (b) detecting and reporting the rest. This plan's
answer to Done-when #2 is that split. It is raised with the TL in the plan post; the reply is recorded
in the issue's `## Log` as a scope revision if it moves the design.

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `NvimKinds` / `Paths.NvimPIDs` / `ScopePaths.NvimPIDGlobs` | `cmd/internal/artifactpath/paths.go` | new |
| `ChangelogArtifactSet.NvimPID` | `cmd/internal/artifactpath/paths.go:106-113,759-772` | modified |
| `NvimReapRow` + `ClassifyNvimReap` | `cmd/internal/launcher/nvim_reap.go` | new |
| `ProcessRow` (gains `Name`) + `ProcessTable` (gains `Command`) | `cmd/internal/launcher/session_reap.go:29-40` | modified |
| `SweepOrphanNvim` / `ReapNvim` / `ReapNvimContext` | `cmd/internal/launcher/osruntime.go:942-1003`, `lifecycle_os.go:34-60` | modified |
| `editorPathsOf` + quit-path inline literal | `cmd/internal/launcher/lifecycle.go:193-196,294-299` | modified |
| `UnattributedCandidate` (extracted from #587) | `cmd/internal/couchcore/resource.go:301` | new |
| `OrphanNvim` + `ClassifyOrphanNvim` + `ManagedRoots` | `cmd/internal/couchcore/resource_orphan.go` | new |
| `ResourcePolicy.OrphanNvimAge` | `cmd/internal/couchcore/resource_policy.go:11-77` | modified |
| `ResourceView.OrphanNvim` (+ projection) | `cmd/internal/couchcore/resource.go`, `couchcmd/resources_projection.go` | modified |

- **`NvimKinds`** — the closed set `{draft, scrollback, review}` of embedded-nvim surfaces that own a
  `nvim-pid-<tag>-<kind>` sidecar. The changelog viewer is deliberately **not** a member: its identity is
  keyed per artifact base (see the root-cause note above), not per (tag, kind).
  - **Relationships:** 1:N with the pid sidecar family `nvim-pid-<tag>-<kind>`; consumed by GC retention, the rename inventory, the quit path, the reaper and the sweep.
  - **DRY rationale:** today the set is spelled in six places with two different members (`gc.go:253` has review, the reaper and the sweep do not). #583 is that drift: a kind nobody reaps is a kind that leaks. ARCH-DRY.
  - **Future extensions:** a new tag-keyed editor surface adds one string and derives everywhere; a drift pin fails if a consumer hardcodes a set again. A surface whose multiplicity is not (tag, kind) follows the changelog's artifact-keyed shape instead of joining this set.
- **`ClassifyNvimReap`** — decides one candidate: kill, or skip with a reason. No IO, no clock, no signals.
  - **Relationships:** 1:1 with a `NvimReapRow` built by the IO shell from one process-table snapshot plus one targeted `Command` read; N:1 with an owner (a tag, or a changelog artifact base).
  - **DRY rationale:** the reaper and the sweep ask the same question; today one answers it with `kill -9 <pid>` and the other with `pkill -9 -f <pattern>`.
- **`ClassifyOrphanNvim`** — decides one untagged candidate: report/reap-eligible, or not. Pure over row + argv + cwd + roots + now.
  - **Relationships:** 1:1 with a candidate row; N:N with `ManagedRoots` (a root may contain many candidates).
  - **DRY rationale:** shares `UnattributedCandidate` with #587's report so "launchd-owned, identity-known, unclaimed, a kind Couch launches" is one rule, not two. The hub's dispatch requires exactly this.
- **`ManagedRoots`** — the set of absolute paths the supervisor already knows: every inventory slot's `WorkspaceIdentity.WorktreeRoot`, `EnvironmentRoot`, `PrimaryRoot` and `FleetRoot`.
  - **DRY rationale:** no new discovery mechanism; `resourceInventory.snapshot()` already publishes them (`couchcmd/resources_inventory.go:239`). Containment is positive evidence, so an incomplete inventory narrows the sweep instead of widening it — absence is never inferred.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `procutil.Reader.Cwd` / `Source.Cwd` | `cmd/internal/procutil/source.go`, `resource_darwin.go`, `resource_unsupported.go` | new | `proc_pidinfo(PROC_PIDVNODEPATHINFO)` |
| `procutil.Source.Identity` / `Source.Command` | `cmd/internal/procutil/source.go` | new | existing `nativeIdentity` / `nativeCommand` |
| `launcher.ProcessTable` (`OSProcessTable`) | `cmd/internal/launcher/session_reap.go:29-40,152-176` | modified | `syscall.Kill`, `procutil.Table/Identity/Command` |
| `couchcore.ProcOps` / `FakeProcOps` | `cmd/internal/couchcore/procops.go:52-275` | reused | identity-fenced `Signal` |
| `resourceSampler.pass` candidate probe | `cmd/internal/couchcmd/resources.go:144-177` | modified | one bounded `Cwd` call per gated candidate |
| `reap-orphans` operation | `cmd/internal/couchcore/ops.go`, `couchcmd/cli.go`, `couchcmd/slot_operations.go`, `couchcmd/reap_orphans.go` | new | confirmation gate + `ProcOps` |
| changelog viewer pid record | `cmd/internal/opener/run.go:211-217`, `nvim/changelog.lua` | modified | `PAIR_NVIM_PID_FILE` |
| VimLeave sidecar removal | `nvim/init.lua:24-34`, `nvim/scrollback.lua:47`, `nvim/review.lua:791`, `nvim/changelog.lua` | modified | `PAIR_NVIM_PID_FILE` |

- **`launcher.ProcessTable`** — one seam for both reapers. `ProcessRow` gains `Name` (free: `procutil.Table()` already reads it) and the interface gains `Command(pid int) (args []string, ok bool)`, a **targeted** argv read for one candidate pid, so the session reaper's whole-host snapshot stays one cheap `kern.proc.all` sysctl. The nvim reaper signals through `Signal`, which makes `exec.Command("kill")` and `exec.Command("pkill")` unreachable from the reap path — that is the seam the "never by name alone" assertion is written against.
  - **Injected into:** `Reaper` (existing) and the new nvim reap path; `fakeProcessTable` (`session_reap_test.go:28-60`) grows a `name` and a `commands` map, and already records every signal as `"pid:SIG"`.
  - **Future extensions:** any further process evidence (cwd, rusage) joins as another targeted per-pid method rather than widening the snapshot.
- **`procutil.Cwd`** — one `proc_pidinfo` call per pid, added to the existing `darwin && cgo` libproc block that already carries `resource_table`/`resource_identity`/`resource_usage`; unsupported builds return an explicit error like `identity_other.go`.
  - **Injected into:** the sampler's candidate loop (through `Reader`, so `samplerReader` fakes it) and nothing else — the classification stays pure.
  - **Cost:** charged to `Cost.NativeCalls` like every other native attempt, and only for rows that already pass ppid/name/argv/age, so the envelope stays inside #587's measured budget (ARCH-CONSTRAINTS).
- **`reap-orphans`** — follows the house confirmation pattern exactly: `slotOperationFlag` wiring, refusal `"%s requires --confirm"`, the server-side second gate `Response{Code:"confirmation-required"}`, and `couchcore.OperationConfirms` as the single source (`couchcmd/cli.go:364-401`, `slot_operations.go:124-126`, `ops.go:154`).
- **The lua mirror is build output, not a diff.** `cmd/internal/runtimebundle/assets/runtime/` is gitignored
  (`.gitignore:133`; `git ls-files` returns nothing for it) and regenerated by `make runtimebundle-generate`
  (`Makefile.local:263-266`), a prerequisite of `build`, `test` and `$(RUNTIMEBUNDLE_ASSETS)`. Measured
  2026-10-16: generating into a temp dir reproduces `nvim/init.lua` byte-for-byte, so the generator copies lua
  verbatim and a differing local copy only means the checkout has not been rebuilt. M1/M2 therefore edit only
  `nvim/`, which is both the generator's source and what the running runtime loads
  (`opener/runtime.go:163-165` runs `PairHome + "/nvim/<config>.lua"`); the bundle picks the change up at the
  next build and there is nothing to commit.

## Operating envelope (ARCH-CONSTRAINTS)

- Interaction path: a 60 s background sample plus an operator-invoked verb. No keystroke path is touched.
- Per pass: one `kern.proc.all` sysctl (existing), `Command` only for nvim candidates over the RSS floor (existing), **plus `Cwd` only for rows that are ppid 1, name nvim, argv `--embed`, and older than `OrphanNvimAge`** — typically 0-3 rows on this machine, hard-capped by `ResourceMaxProcesses`.
- Budget: #587 measured 0.12% of one core at 60 s against a <0.2% approval. One extra `proc_pidinfo` per gated candidate is ~1 µs; the plan re-measures at maximum cardinality and reports the number rather than assuming it.
- Overload: an unreadable cwd yields `CwdKnown == false` → the candidate is **not** classified (never reaped on missing evidence) and the pass records one bounded error string.
- The M1 reaper is not periodic — it runs on quit and on attach. Its extra cost is one `Command(pid)` per candidate row named `nvim` (a handful) plus one `Identity(pid)` re-read per signal, replacing two `pkill` subprocesses and one `kill` subprocess per sidecar. Net: fewer process spawns than today.
- `N/A`: network, disk writes beyond the existing history file, concurrency (the sampler's ticks are already serial).

## State, ordering and uncertainty (ARCH-ORDER)

The reaper holds no state between events: each pass re-reads the table, so a pid observed a moment ago is
never trusted at signal time. The one transition that matters is **observation → signal**, and it is
fenced: `Identity(pid)` is re-read immediately before `Signal`, and a mismatch means "gone, or the pid
now belongs to someone else" (`session_reap.go:119-130` — the pattern already in the tree, reused, not
reinvented). The event most likely to be mishandled is **pid recycling between the pidfile write and the
reap**: a stale sidecar naming a recycled pid. It is governed by *ignore* — three independent predicates
must hold (kernel identity matches the same snapshot read, short name is `nvim`, argv carries an exact
`--embed` field and, where the kind has one, the expected artifact path) before any signal, so a recycled
pid is skipped with a reason rather than killed. Cancellation mid-pass drops the remaining candidates and
reaps nothing partially: a candidate is either signalled or not, and the next pass re-decides.

## Trust boundaries (ARCH-SECURE)

- The pidfile is **input pair did not produce** (Lua wrote it, possibly a pre-upgrade Lua, possibly a
  hand-edited or truncated file): parsed to an int at the boundary, and a non-positive or non-numeric
  value is skipped, never signalled. Its pid grants no authority by itself — authority comes from the
  kernel evidence read at reap time.
- The changelog `.openlock` is the same kind of input, and it decides **owner liveness**: a pid that
  reads as alive (even wrongly, through recycling) makes the reaper *skip*, so a mis-read is safe in the
  only direction that matters. Its pid is never signalled — only the `.nvim-pid` sibling's is, and only
  after the argv check against that sibling's exact `<base>.md`.
- `Cwd` returns a path from the kernel, not from a config file; containment is a cleaned path-boundary
  prefix match, so `/work/tree2` is never inside `/work/tree`.
- Nothing new reads credentials; no secret reaches a log line, an argv, or a fixture. The report names
  pid, age, cwd and RSS only.
- Blast radius: the confirmed verb signals only rows the pure classifier admitted, one pid at a time,
  identity-fenced; a process with a live parent is unreachable by construction because ppid 1 is required.
  Tests run against `FakeProcOps`/fake readers and never signal a real process, except the flag-gated
  live conformance test, which spawns and kills only its own child.

## Funeral (ARCH-FUNERAL)

- `<base>.nvim-pid` is a new suffix on the **existing** changelog artifact base, beside `.openlock`,
  `.status` and `.distill.lock`: created by the changelog viewer's Lua at VimEnter, last needed by the
  reaper, removed by the reaper after a decided outcome and by VimLeave on a clean exit, and covered by
  `storagegc` retention through `gc.go:265-275`'s changelog suffix loop (one string added). Its growth
  bound is therefore the changelog family's existing one — one file per (tag, agent, session), the same
  count as `.openlock` — not a new family and not a new cardinality.
- `nvim-pid-<tag>-review` is not new on disk (review.lua already writes it); it becomes *reaped*, so its
  end is now named: the reaper removes it after a decided outcome, VimLeave removes it on a clean exit,
  and `gc.go:253`'s kind loop already retains it.
- No new durable store: the orphan classification is in-memory per pass, exactly like #587's episodes
  after its simplification. The report dies with the supervisor; the history file is unchanged.
- The killed process is the residue this issue removes; nothing is created that outlives its owner.

---

## Chunk 1: M1 — tagged embeds are reaped by exact identity

### Task 1: One closed kind set

**Files:**
- Modify: `cmd/internal/artifactpath/paths.go` (near `NvimPIDChecked:673-676`, `RenameArtifacts:552-568`, `NvimPIDGlob:286`)
- Modify: `cmd/internal/artifactpath/gc.go:253-255`
- Modify: `cmd/internal/launcher/lifecycle.go:193-196,294-299`, `cmd/internal/launcher/osruntime.go:947,973`
- Test: `cmd/internal/artifactpath/paths_test.go`, `cmd/internal/launcher/lifecycle_test.go`

- [x] **Step 1: Write the failing drift pin.** Strategy: the kind set has exactly one spelling, so no
  consumer test may contain a literal `{draft, scrollback, …}` — each builds its expectation from
  `NvimKinds()`, and the pin asserts the set's members and that `Paths.NvimPIDs()` resolves one sidecar
  per kind. The consumers pinned: `gc.go`'s retention match, `RenameArtifacts`, `editorPathsOf`, and both
  `osruntime` loops.
- [x] **Step 2: Run it, expect FAIL** (`undefined: NvimKinds`).
  `go test ./cmd/internal/artifactpath/ -run 'NvimKinds|NvimPID' -count=1`
- [x] **Step 3: Implement** `func NvimKinds() []string` (returning `{draft, scrollback, review}`),
  `func (p Paths) NvimPIDs() []string`,
  `func (p ScopePaths) NvimPIDGlobs() ([]NvimPIDGlob, error)` (`NvimPIDGlob{Kind, Pattern string}`), and
  rewrite the six consumers to derive. Keep the quit-path literal at `lifecycle.go:193-196` spelled at its
  call site — `TestManagedIOInventoryAnchors` requires each scoped member to be consumed where resolved —
  but let it range over `NvimKinds()`. The changelog viewer is **not** a kind; Task 4 gives it an
  artifact-keyed record instead (see the root-cause note).
- [x] **Step 4: Run the artifactpath + launcher packages, expect PASS.**
  `go test ./cmd/internal/artifactpath/ ./cmd/internal/launcher/ -count=1`
- [x] **Step 5: Commit** `#583: single-source the nvim pid sidecar kind set`

### Task 2: The reap decision, pure

**Files:**
- Create: `cmd/internal/launcher/nvim_reap.go`, `cmd/internal/launcher/nvim_reap_test.go`

- [x] **Step 1: Write the failing table test.** Strategy: every field of a `NvimReapRow` is untrusted
  input, so the table drives one field at a time to its boundary — absent, wrong-valued, and
  right-valued — and asserts the verdict *and* the reason string for each, with the invariant that no
  row missing any one predicate is ever a kill. The two rows that must kill: owner dead, and owner live
  but ppid 1 — the closed pane, an editor that died while its session lived on. *(The 22 GB orphan was not
  an instance of this row: it was a **bare** embed whose argv carried no scope path, which is M2's class;
  and until the M1 boundary review no production source built such a row at all — see `## Revisions`,
  2026-10-16 "M1 boundary review".)*
- [x] **Step 2: Run, expect FAIL.** `go test ./cmd/internal/launcher/ -run ClassifyNvimReap -count=1`
- [x] **Step 3: Implement**

```go
type NvimReapRow struct {
	Kind, Owner     string // Owner is a tag, or a changelog artifact base
	PID, PPID       int
	Identity        string
	Name            string
	Args            []string
	ArgsKnown       bool
	OwnerLive       bool   // false on the explicit quit path: quitting authorizes the reap
	ExpectedArgPath string // "" for kinds whose argv carries no scope path (review)
}

// ClassifyNvimReap decides one candidate. It kills only when the pid is
// positive, the identity is known, the short name is nvim, argv is known and
// carries an exact "--embed" field, argv carries ExpectedArgPath exactly when
// one is expected, and the owner is dead or the process is reparented to 1.
// Every skip returns a reason; nothing is signalled on missing evidence.
func ClassifyNvimReap(row NvimReapRow) (kill bool, reason string)
```

  Exact `--embed` and exact `ExpectedArgPath` matching use field comparison, not `strings.Contains`
  (the #587 classifier's rule, `couchcore/resource.go:326-332`).
- [x] **Step 4: Run, expect PASS.**
- [x] **Step 5: Commit** `#583: pure identity-exact nvim reap decision`

### Task 3: The reaper kills on verified identity, never by name

**Files:**
- Modify: `cmd/internal/launcher/session_reap.go:29-40,152-176` (the `ProcessTable` seam)
- Modify: `cmd/internal/launcher/osruntime.go:937-958`, `cmd/internal/launcher/lifecycle_os.go:34-60`
- Modify: `cmd/internal/procutil/source.go` (expose `Identity` and `Command` on `Source`)
- Test: `cmd/internal/launcher/session_reap_test.go` (the fake), a new `nvim_reap_os_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: a sidecar names a pid, and a pid is not a process —
  the fake table makes each predicate independently false (recycled identity at signal time, non-nvim
  name, no `--embed` field, an argv serving a different draft path) and asserts that the recorded signal
  list stays empty, while a fully-evidenced dead-owner row records exactly one `SIGKILL`. The
  "never by name alone" assertion is structural: the reap path holds no subprocess seam at all, so the
  test asserts the fake's signal list and no test can observe a `pkill`.
- [x] **Step 2: Run, expect FAIL.**
- [x] **Step 3: Implement.** Extend the seam — `ProcessRow` gains `Name string`; `ProcessTable` gains
  `Command(pid int) (args []string, ok bool)`; `OSProcessTable.Snapshot` fills `Name` from the
  `procutil.Table()` row it already reads, and `OSProcessTable.Command` calls the new
  `procutil.Source.Command` (a thin export of `nativeCommand`, argv for one pid, `ok == false` when the
  read fails). Then `ReapNvim`/`ReapNvimContext` take one snapshot, build a row per sidecar across
  `NvimKinds()` with `OwnerLive: false` (the explicit path is authorized), decide with `ClassifyNvimReap`,
  re-read `Identity(pid)` immediately before `Signal` (`Reaper.signalAll`'s rule, `session_reap.go:119-130`),
  and remove the sidecar only after a decided outcome. **Delete `pkillF`** and both pattern kills
  (`osruntime.go:938,956-957`, `lifecycle_os.go:49-58`), and the `exec.Command("kill", "-9", pid)` calls
  with them: `regexp` may drop out of both files' imports.
- [x] **Step 4: Run launcher + procutil, expect PASS.** `go test ./cmd/internal/launcher/ ./cmd/internal/procutil/ -count=1`
- [x] **Step 5: Commit** `#583: reap embedded nvim on verified identity, drop name-based pkill`

### Task 4: Give the changelog viewer an artifact-keyed pid record

**Files:**
- Modify: `cmd/internal/artifactpath/paths.go:106-113` (`ChangelogArtifactSet`), `:759-772` (`ChangelogArtifacts`)
- Modify: `cmd/internal/artifactpath/gc.go:265,275` (the changelog suffix loop and `add("changelog", …)`)
- Modify: `cmd/internal/opener/run.go:211-217`
- Modify: `nvim/changelog.lua`
- Test: `cmd/internal/artifactpath/paths_test.go`, `cmd/internal/opener/run_test.go:120`

- [x] **Step 1: Write the failing tests.** Strategy: the record is a sibling of the artifacts it belongs
  to, so assert it derives from the same base (`.nvim-pid` beside `.openlock` for both the sessionless and
  the sessioned form), that GC's retention matcher recognises the new suffix as a changelog artifact, and
  that `RunChangelog` passes `PAIR_NVIM_PID_FILE=` pointing at exactly that path (the assertion shape at
  `run_test.go:120`).
- [x] **Step 2: Run, expect FAIL.** `go test ./cmd/internal/artifactpath/ ./cmd/internal/opener/ -count=1`
- [x] **Step 3: Implement** `ChangelogArtifactSet.NvimPID = base + ".nvim-pid"`, add `".nvim-pid"` to
  `gc.go:265`'s suffix list and `c.NvimPID` to `gc.go:275`'s `add`, add
  `"PAIR_NVIM_PID_FILE=" + changelog.NvimPID` to the viewer env at `opener/run.go:211-215`, and add the
  pidfile block to `nvim/changelog.lua` — a copy of `init.lua:24-34`'s VimEnter write (same three lines,
  same `pcall`, same silence on failure). The `runtimebundle` mirror is gitignored build output that
  `make runtimebundle-generate` refreshes from `nvim/` — edit the source only (see Integration points).
- [x] **Step 4: Run artifactpath + opener + storagegc, expect PASS.**
  `go test ./cmd/internal/artifactpath/ ./cmd/internal/opener/ ./cmd/internal/storagegc/ -count=1`
- [x] **Step 5: Commit** `#583: give the changelog viewer an artifact-keyed nvim pid record`

### Task 5: The sweep reaps per pid from one snapshot

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go:960-1003` (`SweepOrphanNvim`)
- Test: `cmd/internal/launcher/lifecycle_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: the sweep sees the whole scope at once, so the
  adversarial class is *mis-attribution* — a row whose tag is live, a row whose argv carries a tag that is
  a prefix of a live one, a changelog pair whose opener is still alive, and a bare `nvim --embed` with no
  scope path at all (which belongs to M2's classifier) must each leave the signal list empty, while a
  dead tag's draft embed, a dead tag's scrollback embed, an unreaped `review` sidecar, and a changelog
  pair with a dead opener are each signalled once.
- [x] **Step 2: Run, expect FAIL.** `go test ./cmd/internal/launcher/ -run SweepOrphanNvim -count=1`
- [x] **Step 3: Implement.** One snapshot; three evidence sources feeding `ClassifyNvimReap` rows:
  1. `nvim-pid-<tag>-<kind>` sidecars over `NvimPIDGlobs()` — `Owner` is the tag from `TagFromNvimPID`,
     `OwnerLive` is `live[tag]`, `ExpectedArgPath` is the kind's path for draft/scrollback and `""` for
     review. *(As built, every tag the globs find is a target, live ones included — `tagSidecars` hands
     `live[tag]` to `nvimSidecarsFor`, so a live tag's embed is decided on the ppid rule instead of being
     dropped; see `## Revisions`, 2026-10-16 "M1 boundary review".)*
  2. rows named `nvim` whose argv carries a scope path — `Owner` from `TagFromNvimEmbedArgv`
     (draft/scrollback only; it stays as it is), `OwnerLive` from `live`.
  3. `<base>.nvim-pid` siblings — `Owner` is the base, `ExpectedArgPath` is `<base>.md`, and `OwnerLive`
     comes from the paired `<base>.openlock`: **an absent openlock means the owner is dead**, because the
     opener removes it on a clean exit (`opener/run.go:189`) and only an abrupt death leaves it behind —
     so absent is the *reap* case, not a skip. A present openlock whose pid is alive skips; whose pid is
     gone reaps; whose content is not a positive integer is malformed and reaps (the opener that wrote it
     is not the process the row describes, and the row's own predicates still gate the kill).
  Every evidence source states its verdict for absent, present and malformed (the rule PQ-5 raised):
  a sidecar with no numeric pid is malformed and removed without a signal; a sidecar whose pid is absent
  from the snapshot is gone and removed; a `review` row has no `ExpectedArgPath`, so argv need not carry
  one and the sidecar plus identity are the whole authority.
  Replace the current `cands[tag] → ReapNvim(tag)` fan-out with per-row decisions and identity-fenced
  signals; a sidecar is removed only after its row is decided.
- [x] **Step 4: Run, expect PASS.** `go test ./cmd/internal/launcher/ -count=1`
- [x] **Step 5: Commit** `#583: sweep embedded nvim per pid from one snapshot`

### Task 6: A clean exit removes its own sidecar

**Files:**
- Modify: `nvim/init.lua`, `nvim/scrollback.lua`, `nvim/review.lua`, `nvim/changelog.lua`
- Test: `tests/` lua harness for the pidfile lifecycle (the existing `nvim --headless` test shape)

- [x] **Step 1: Failing test:** a headless nvim started with `PAIR_NVIM_PID_FILE` in a temp dir writes the
  sidecar, and after `qa!` the sidecar is gone. (A surviving embed keeps its sidecar — that is the reaper's
  evidence, so it must **not** be removed by anything but the embed's own VimLeave.)
- [x] **Step 2: Run, expect FAIL.**
- [x] **Step 3: Implement** a `VimLeave` autocmd in each config removing `vim.env.PAIR_NVIM_PID_FILE`.
- [x] **Step 4: Run, expect PASS.**
- [x] **Step 5: Commit** `#583: remove the nvim pid sidecar on clean exit`

### Task 7: Live conformance — Done-when #1

**Files:**
- Create: `cmd/internal/launcher/nvim_reap_live_test.go`

- [x] **Step 1: Write the flag-gated live test** (`-nvim-reap-live`, skipping unless set, in the
  `peer_live_test.go` shape): spawn a real `nvim --embed --headless -u <pair init.lua> <tmp>/draft-<tag>.md`
  with `PAIR_NVIM_PID_FILE` set, wait for the sidecar, kill the owner (the spawning parent), run the reaper,
  assert the pid is gone and the sidecar removed. Second case: a real `nvim --embed` with a **live** parent
  and a live tag is still running afterwards. Third case: a non-nvim process named in a stale sidecar is
  untouched. *(Landed with two measured deltas — an env gate instead of a flag, and owner death supplied as
  the tag's absence from `liveTags` instead of a killed parent; see `## Revisions`.)*
- [x] **Step 2: Run it, expect FAIL before Task 3's behaviour is in place / PASS after.** *(Task 3's behaviour
  was already in place, so the teeth were proven by mutation instead: `if row.OwnerLive ||` in
  `ClassifyNvimReap` turns the reap case into a clean four-assertion failure in 10 s.)*
- [x] **Step 3: Run the flag-gated live test on this machine and record the output in `## Log`.**
- [x] **Step 4: Commit** `#583: live conformance for the tagged embed reaper`

### Task 8: M1 atlas + storage inventory

- [x] **Step 1:** update `atlas/storage-retention-io.md`'s nvim-pid and changelog rows (read them first:
  the anchors are grep-checked by `storagegc.TestManagedIOInventoryAnchors`) and
  `atlas/architecture.md:1256`'s sidecar sentence to name the three tag-keyed kinds, the changelog's
  artifact-keyed `.nvim-pid` record, and identity-exact reaping (no name-based kill).
  *(Premise corrected on reading: `storage-retention-io.md` has **no** nvim-pid row — its rows are
  retention-IO anchors, and the pid records are written by lua `writefile` outside the retention guard, as
  they were before #583, so nothing there was stale and `TestManagedIOInventoryAnchors` passes untouched.
  `architecture.md` is the only atlas file that mentions `nvim-pid`, and it needed four edits, not one:
  `:383` still described the deleted `pkill -f` fallback and two publishing configs, `:385` still described
  tag-level collection and reaping, `:1218`'s tag-only family list still said `{draft,scrollback}`, and
  `:1256` is the sidecar row.)*
- [x] **Step 2:** `go test ./cmd/internal/storagegc/ -run TestManagedIOInventoryAnchors -count=1` → PASS.
- [x] **Step 3:** `sdlc milestone-close --issue 583 --milestone M1`, then commit.
  *(Closed at round 5: `Review-Verdict: SHIP` over `758c4b72..25ae5ed5`, 0 new findings, BR-7 disposed as
  addressed, all eight ARCH principles pass. The gate committed its own evidence as `9545171de` and pushed
  the branch, so there was nothing left to paste or commit here. Rounds 3, 4 and 5 were one family —
  `kill-attribution-must-name-owner` — which is the story `## Revisions` tells; the judge's own mutation
  check reddened 9 rows, a wider set than my 3, because it also dropped the rows that merely need the bound
  file.)*

## Chunk 2: M2 — untagged orphans are reported and reaped on confirmation

### Task 9: Prove the identity stamp is a wall clock

**Files:**
- Create: `cmd/internal/procutil/identity_age_test.go`

- [x] **Step 1: Failing test:** spawn `sleep 30` as an owned child, read `Identity(pid)`, parse
  `"<sec>.<usec>"`, assert the elapsed time since that stamp is between 0 and 5 s. This is the assumption
  every age threshold rests on, so it is measured, not read from a header (ARCH-MOCK: the fake models what
  a live check proves). *(Landed wider — see `## Revisions`: the live test also runs on linux, and three pure
  parser tables cover both token shapes on every platform.)*
- [x] **Step 2: Run, expect FAIL** until `ParseIdentityTime` exists. *(Red first, and red for the right
  reason: the live check caught the test passing a pid where the function takes a token.)*
- [x] **Step 3: Implement** `func ParseIdentityTime(identity string) (time.Time, bool)` in `procutil`
  (rejecting a malformed or empty stamp) and use it wherever age is derived. *(Landed as
  `StartTimeFromIdentity(identity string) (time.Time, bool)` plus the pure `ParseIdentityWallClock`,
  `ParseIdentityTicks` and `ParseBootTime` — see `## Revisions`.)*
- [x] **Step 4: Run, expect PASS.** Commit `#583: derive process age from the kernel identity stamp`

### Task 10: The cwd probe

**Files:**
- Modify: `cmd/internal/procutil/source.go` (`Reader`, `Source`), `resource_darwin.go` (C block), the
  unsupported-build file beside `identity_other.go`
- Test: `cmd/internal/procutil/resource_darwin_test.go`, `source_test.go`

- [x] **Step 1: Failing live conformance test:** for an owned child started with a known `Dir`, `Cwd(pid)`
  returns that directory; for a dead pid it returns an error, never an empty success. *(Landed as two tests
  beside #606's `Environment` conformance test, in its re-exec shape: `TestResourceCwdOfAChild` and
  `TestResourceCwdRefusesAGoneChild`. The known directory is `filepath.EvalSymlinks`'d first, because the
  kernel resolves `/var/folders/...` to `/private/var/...` and a caller comparing roots has to match.)*
- [x] **Step 2: Run, expect FAIL.** *(Red on the missing method, and red caught one real bug in the test
  itself — `cmd.Wait()` returns one value, not two.)*
- [x] **Step 3: Implement** `static int resource_cwd(int pid, char *buf, size_t len)` with
  `proc_pidinfo(pid, PROC_PIDVNODEPATHINFO, 0, &vpi, sizeof vpi)` copying `vpi.pvi_cdir.vip_path`, the Go
  wrapper, the `Reader`/`Source` passthrough and the unsupported-build error. Add `Cwd` to every existing
  fake reader (`couchcmd/resources_test.go:20-66`, `procutil/source_test.go`).
  *(Two deltas. The probe landed on the fenced `NativeResourceReader` seam — `resourceCalls.cwdRead` plus
  `Cwd(ctx, p) (string, int, error)` — and **not** on `source.go`'s `Reader`/`Source`: that seam feeds
  `Snapshot`, whose consumers want argv and identity, and widening an interface with no consumer forces every
  fake to grow a method nothing calls (ARCH-DRY cuts both ways). The fence is why: the cwd is evidence for a
  kill, so the read is checked against the same generation that produced the row, and a pid recycled
  mid-read answers an error rather than another process's directory. Consequently the `couchcmd` fake moves
  in Task 13, where `resourceReader` gains `Cwd`, not here.)*
- [x] **Step 4: Run procutil, expect PASS.** Commit `#583: read a process cwd through libproc`
  *(Also: a short `proc_pidinfo` read that leaves `errno` clean falls back to `ESRCH`; an empty path from a
  successful call is refused as `empty process cwd`, since no process has one; and because `cwdRead` joins
  `resourceCalls`, the probe inherits that interface's darwin+cgo boundary — `resource_unsupported.go` answers
  `ErrUnsupported`, which the sweep must read as *unknown*, never as *outside every root*. That is the same
  boundary #587's RSS and argv reads already have.)*

### Task 11: One unattributed rule (DRY with #587)

**Files:**
- Modify: `cmd/internal/couchcore/resource.go:301-341`
- Test: `cmd/internal/couchcore/resource_test.go:90` (`TestResourceClassifyUnproven`) stays green unchanged

- [x] **Step 1: Extract** `func UnattributedCandidate(p procutil.Process, claimed map[ResourceProcessKey]bool, args []string) bool`
  holding exactly today's predicates (ppid 1, not zombie, identity known, unclaimed, known kind name, and
  for `nvim` an exact `--embed` field). `ClassifyUnattributed` keeps its RSS floor and view construction and
  calls it.
  *(Landed as three named pieces rather than one, because there were **three** copies of the rule, not two:
  `UnattributedShape` (the row-level half), `UnattributedKind` (the name half, wide enough to run before argv
  is read) and `UnattributedCandidate` (both, plus nvim's `--embed` proof). The third copy was the sampler's
  own pre-filter, which built `{zellij, nvim} ∪ AgentInventory()` a second time and re-stated the shape checks
  inline; it now calls the shared two (`couchcmd/resources.go:193-205`), so a row the sampler admits and the
  classifier refuses is impossible rather than merely unlikely. The kind set also moved to a package-level
  `unattributedKinds`, built once: `ClassifyUnattributed` walks up to `ResourceMaxTableRows` rows per pass and
  was rebuilding the map for every call, inside a sampler with a <0.2% CPU budget.
  One deliberate tightening: `p.PID <= 0` is now refused. Neither platform's table can produce it
  (`table_darwin.go:24` and `parseLinuxStat` both filter), so `TestResourceClassifyUnproven` cannot tell the
  difference — but the predicate now gates a kill path, and a kill path should not inherit its pid validation
  from a caller's filter.)*
- [x] **Step 2: Run the existing #587 tests unchanged, expect PASS** — the extraction is behaviour-preserving,
  and a red existing test means the extraction changed the rule.
  `go test ./cmd/internal/couchcore/ -run 'Resource' -count=1`
  *(Green: `TestResourceClassifyUnproven` is byte-identical, all `Resource*` tests pass in 6.6 s, and the whole
  `couchcmd` package — which holds the sampler whose filter moved — passes in 79 s. The new
  `TestUnattributedCandidate` falsifies one claim per row across 14 rows: a plain nvim, an unread argv,
  `--embed` inside a longer field, an unknown kind, a name that merely contains `nvim`, a full path that still
  names its kind, a live parent, a zombie, no generation, no pid, and a claimed row.)*
- [x] **Step 3: Commit** `#583: extract the shared unattributed-process predicate`

### Task 12: The orphan classification, pure

**Files:**
- Create: `cmd/internal/couchcore/resource_orphan.go`, `resource_orphan_test.go`

- [x] **Step 1: Write the failing table test.** Strategy: every input to the classifier is a claim about
  the host that may be false or missing, so the table falsifies one claim at a time — including the two
  that look like success but are not (a cwd that is a *sibling prefix* of a managed root, and an age just
  under the threshold) — and asserts that only a row with every claim intact is classified, carrying pid,
  identity, age, cwd, the matching root and RSS.
  *Done: 20 rows in `TestClassifyOrphanNvimFalsifiesOneClaimPerRow` (including the two traps, plus "exactly
  at the threshold", "a start stamp in the future is not an age", "an unread RSS does not hide an orphan" and
  "the most specific checkout wins"), `TestManagedRootsCollectsEverySlotCheckout` and
  `TestWithinManagedRootIsSegmentSafe` (8 rows: the root itself, a descendant, a nested root, a shared string
  prefix, a shorter sibling, an ancestor of a root, a relative path and a non-canonical `..` path). Containment
  reuses `RelativeFamilyPath` (`repository_family.go:79`) rather than a new prefix rule — it already requires
  canonical absolute paths and refuses `..`, which is the segment safety the trap row attacks (ARCH-DRY).*
- [x] **Step 2: Run, expect FAIL.** *(Red for the right reason: 8 undefined symbols, no other failure.)*
- [x] **Step 3: Implement**

```go
type OrphanNvim struct {
	Key      ResourceProcessKey
	Age      time.Duration
	Cwd      string
	Root     string
	RSSBytes *uint64
	Reason   string
}

// ResourceOrphanProbe is what the sampler's IO proved about one candidate beyond
// its usage row and argv.
type ResourceOrphanProbe struct {
	Cwd      string
	CwdKnown bool
	Age      time.Duration
	AgeKnown bool
}

func ManagedRoots(slots []ResourceSlot) []string
func WithinManagedRoot(cwd string, roots []string) (string, bool)
func ClassifyOrphanNvim(row procutil.Process, args []string, claimed map[ResourceProcessKey]bool,
	probe ResourceOrphanProbe, rss *uint64, roots []string, minAge time.Duration) (OrphanNvim, bool)
func ClassifyOrphanNvims(table []procutil.Process, owned []ResourceProvenance,
	observations []ResourceUsageObservation, commands map[ResourceProcessKey][]string,
	probes map[ResourceProcessKey]ResourceOrphanProbe, roots []string, minAge time.Duration) []OrphanNvim
```

  *Delta (see `## Revisions`): the classifier takes a probe bundle instead of `(cwd, cwdKnown, now)` and
  derives no age itself, `Name` is dropped from `OrphanNvim` (it is provably `nvim`), `ManagedRoots` takes the
  slot slice, and the plural sweeper mirrors `ClassifyUnattributed`'s shape and caps.*
- [x] **Step 4: Run, expect PASS.** Commit `#583: classify untagged embedded-nvim orphans`
  *Green. Mutation check: replacing `RelativeFamilyPath` with a string-prefix match and longest-match with
  first-match reddened exactly `cwd shares a prefix with a managed checkout`, `the most specific checkout
  wins` and `TestWithinManagedRootIsSegmentSafe` — the three guards that carry the safety claim.*

### Task 13: Policy knob and sampler wiring

**Files:**
- Modify: `cmd/internal/couchcore/resource_policy.go:11-77`, `cmd/internal/couchcore/resource.go` (view),
  `resource_reduce.go:258`, `cmd/internal/couchcmd/resources.go:144-177`, `resources_inventory.go`
- Test: `resource_policy` tests, `couchcmd/resources_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: the knob is operator-supplied text and the probe is a
  cost, so the tests attack both — a malformed or out-of-range value must refuse with the policy's existing
  error shape rather than silently widen the sweep, and a sampler pass must pay for `Cwd` only on rows that
  already cleared the ppid/name/age gate (charged to `Cost.NativeCalls`), never on a live-parent nvim.
  *Done: `TestResourcePolicyStrict` grew a 6h default, a `90m` override and five refusals (`""`, `13 days`,
  `0s`, `-6h`, `25h`); `TestAdvanceResourcesReportsTheOrphanNvimSweep` proves the view carries pid/root/cwd/
  age/RSS, that an unprobed pass reports nothing (silence means "not proven", never "clean") and that a wider
  threshold suppresses the row; `TestResourceSamplerProbesCwdOnlyForAnOldOrphanNvim` runs five rows (an old
  orphan nvim, a one-second-old nvim, a live-parent nvim, an nvim whose stamp will not parse, an old zellij)
  and asserts `cwd` probes `[40]` only, argv reads `[40 41 43]`, `Cost.CwdCalls` charged, the generic
  `Unattributed` row for pid 40 suppressed while the other three survive, and the cached row is immutable.*
- [x] **Step 2: Run, expect FAIL.** *(Red on the undefined policy field, sample field and view field; then on
  the two expectations I had wrong — the darwin identity fraction is microseconds, not nanoseconds, so the
  expected age was off by 4.995µs, and `Unattributed` keeps every other big candidate rather than only the
  zellij row. Both were test bugs, fixed against measured behaviour.)*
- [x] **Step 3: Implement:** add `OrphanNvimAge time.Duration` to the policy and its env case; in the
  candidate loop, gate `Cwd` behind ppid 1 + name nvim + age ≥ threshold (age comes from the identity stamp
  already in the row, so the gate costs no syscall), then classify with the roots from `s.slots()`.
  *Delta: `COUCH_RESOURCE_ORPHAN_NVIM_AGE` joins the existing duration case, so it inherits the 1s..24h bound
  rather than adding a second range rule. The argv gate widened from "RSS above the floor" to "old OR above
  the floor", because a small untagged orphan is still an orphan; the floor still decides the argv read for
  #587's report. `ResourceCost` gained `CwdCalls`, `ResourceSample` gained `OrphanProbes`, `ResourceView`
  gained `OrphanNvim`, and `cloneResourceView` copies the new rows so a handler cannot mutate the cache.
  `resources_inventory.go` needed no change: `Workspace.WorktreeRoot` is already populated per slot there.*
- [x] **Step 4: Run couchcore + couchcmd, expect PASS.** Commit `#583: sample untagged nvim orphans within the measured budget`

### Task 14: The health line

**Files:**
- Modify: `cmd/internal/couchcmd/resources_render.go:43-92`, `resources_projection.go:16-68`,
  `cmd/internal/couchcore/resource_history.go:10-17,63-80`
- Test: `cmd/internal/couchcmd/resources_cli_test.go`, `resources_projection_test.go`,
  `cmd/internal/couchcore/resource_history_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: the section is an operator's only view of the sweep
  and the projection has a byte budget, so the tests attack silence and size — an empty result must say
  `none observed` rather than omit the section (an absent section reads as "not checked"), and the wire
  projection must stay inside `MaxFrameBytes` at maximum cardinality, with `--json` carrying the same rows.
  *Done: `TestResourceRenderReportsTheOrphanNvimSweep` pins both the empty section and one exact row
  (age rounded to the minute, RSS in bytes) plus the `orphan_nvim` JSON key; the sanitize test grew an
  orphan row carrying the terminal-injection string in `Cwd`, `Root` and `Reason`; the capacity test fills
  all 8192 orphan rows at maximum cardinality and asserts the wire keeps a bounded non-empty prefix and
  names the omission.*
- [x] **Step 2: Run, expect FAIL.** *(Red on `undefined: orphanNvimSection` in three places.)*
- [x] **Step 3: Implement:** an `Orphan nvim (untagged; reap needs --confirm):` section in the existing
  section list (`resources_render.go:66`), one row per orphan printing `pid=`, `age=`, `cwd=` and the
  matching root in the house row shape (`resources_render.go:72`), `none observed` when empty, and the
  same rows in `--json` and in the wire projection.
  *Delta: the header is a package constant so the tests cannot drift from it, and every string field goes
  through `rowtext.Sanitize`. The sampler cost line gained `cwd %d` beside `table`/`usage`/`command` — the
  health line's contract is that new IO is visible in it. The projection's budget divisor went 5 → 6 with
  an `orphan nvim omitted` notice, matching the existing per-section pattern.
  And `ResourceFrame` gained `OrphanNvim`: the reduce suppresses a named orphan from `Unattributed`
  (Task 13), so without this field the persisted ring would forget the very offender it exists to record
  — a 22 GB untagged nvim was in `frame.Unattributed` before #583 and would be nowhere after. Version stays
  1 and the field is additive, so an upgrade reads old history fine; a downgrade loses continuity into a
  fresh memory ring, the loud best-effort path `resources_persist.go` already has. That schema cost was
  already paid by `ResourceCost.CwdCalls`, which the frame persists. The per-frame bound check moved into
  `frameWithinBounds` rather than growing the 230-char condition; mutation-checked by deleting the
  `OrphanNvim` clause, which reddened exactly `TestResourceHistoryBoundsTheOrphanSweep`.*
- [x] **Step 4: Run, expect PASS.**
- [x] **Step 5: Commit** `#583: report orphan embedded nvim in couch --resources`

### Task 15: One hub alert per episode

**Files:**
- Modify: `cmd/internal/couchcmd/resources_alert.go:35-59`, `cmd/internal/couchcore/resource_episode.go`,
  `cmd/internal/couchcore/resource_reduce.go:89-112,155-161,227-246`
- Test: `resources_alert_test.go`, `resources_alert_broker_test.go`, `couchcore/resource_orphan_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: an alert that repeats every 60 s is noise and one
  that never fires is a silent leak, so the tests attack both edges of the episode — the first pass over
  the threshold notifies the hub once, in the existing body shape with `Inspect with couch --resources.`,
  and later passes inside the same episode notify nothing — proven end to end against the real broker
  (delivery and dedup by effect ID), not only against a fake.
  *Done: `TestOrphanNvimAlertsTheHubOncePerEpisode` walks the reduce — accumulate, fire once with the
  effect's pid/identity/name/root/age/workspace bound, three quiet samples, clear when the process goes,
  re-arm under a recycled pid as a different effect ID. `TestResourceAlertBrokerEndToEndFromTheSweep`
  runs the real sampler over a real broker with nothing stubbed between them and asserts the delivered
  body, the silence before and after, and that no `nvim/` episode survives the reap.
  `TestResourceAlertBrokerDeliversTheOrphanNvimOnce` pins the exact body and the hub-to-hub addressing;
  `TestResourceAlertSanitizesTheOrphanRoot` proves a hostile cwd cannot inject into the hub's terminal.*
- [x] **Step 2: Run, expect FAIL.** *(Red on the two new `ResourceEffect` fields.)*
- [x] **Step 3: Implement** on the existing episode machinery (`pending → notified`, the `resource-%x`
  effect ID at `resource_episode.go:112`, `sendGenerated` at `supervision.go:108-127`), with its own body
  template beside `resources_alert.go:48`.
  *Delta: `ResourceEffect` gained `Age` and `Root` (both `omitempty`, both bound when the effect fires and
  never re-read afterwards, per the BR-7 family rule), and `orphan-nvim` is the constant
  `ResourceMetricOrphanNvim` because couchcmd now selects a body on it. `advance`'s `value float64`
  parameter became a `ResourceEffect` so a metric carries its own evidence instead of a float whose
  meaning depends on the metric; the owner still supplies name/slot/process/workspace, so a new metric
  cannot forget to identify its process. The orphan classification moved above the episode-retention loop
  (it reads nothing the loop writes) so its episodes are `visited`, and `nvim/` joined the
  process-keyed prefixes — now `processEpisodeKey` — so a reaped orphan's episode clears instead of
  occupying `ResourceMaxEpisodes` forever.
  Routing was the open question: an orphan has no owning slot, and `hubBindingAt("")` resolves nothing, so
  a workspace-less effect would sit pending forever and the alert would be silently dead. The episode is
  addressed by the managed checkout the sweep matched, whose `WorkspaceIdentity` is copied whole from the
  live slot that owns that root — `ManagedRoots` only collects live slots, so the lookup always succeeds
  and the identity is as resolvable as any slot-attributed alert. No second routing rule was added.
  Pre-existing gap noted, not fixed here: nothing tests `newResourceAlertDispatcher`'s resolve
  (`hubBindingAt(authority.owner(id))`) — that needs a full `messageService` rig and is #587's seam.*
- [x] **Step 4: Run, expect PASS.** Mutation-checked: dropping `nvim/` from `processEpisodeKey`, firing on
  one sample instead of `OrphanSamples`, and removing the body's `rowtext.Sanitize` each reddened exactly
  their own guard; reverted, green.
- [x] **Step 5: Commit** `#583: alert the hub once per orphan nvim episode`
- Note: #574's hub lifecycle notice does not exist yet (its Plan is empty), so this uses the #587 alert
  seam; when #574 lands, the notice moves there. Recorded in `## Log`.

### Task 16: The confirmed reap

**Files:**
- Create: `cmd/internal/couchcmd/reap_orphans.go`
- Modify: `cmd/internal/couchcmd/cli.go` (the pre-scan list and `parseMessageCLI`), `messages.go` (the route),
  `message_service.go` (the handler and the sampler's construction), `resources.go` (the sweep seam),
  `resources_render.go` (the shared row renderer), `cost.go` + `run.go` + `skills/couch/SKILL.md` (the verb-cost
  triple), `cmd/internal/couchmessage/protocol.go` (the confirmation gate),
  `cmd/internal/launcher/session_reap.go` + `lifecycle.go` (`TreeReaper`)
- Test: `cmd/internal/couchcmd/reap_orphans_test.go`, `resources_test.go`

- [x] **Step 1: Write the failing tests.** Strategy: this verb ends processes, so the tests attack the gap
  between *deciding* and *signalling* — the confirmation gate at both the CLI and the server, and the
  world changing in between (a recycled identity, a row that has found a parent again) — against
  a `launcher.ProcessTable` fake, asserting that an unchanged world is signalled exactly once and a changed
  one not at all, with every skip named and reasoned in the output.
- [x] **Step 2: Run, expect FAIL.** `go vet` named it: `undefined: orphanReapOp`.
- [x] **Step 3: Implement** the operation: re-classify from a fresh sample (never reap from a cached view),
  then signal `SIGTERM`, wait, `SIGKILL` — the `Reaper` cadence — each step identity-fenced.
- [x] **Step 4: Run couchcmd, expect PASS.** Commit `#583: reap orphan embedded nvim only on confirmation`
- Mutation-checked, each reverted: dropping `protocol.go`'s confirmation gate (caught by both
  `TestReapOrphansConfirmationIsAProtocolGate` and the end-to-end `…SupervisorClassifiesFresh`, which reports
  "an unconfirmed reap was answered ok"); answering from `sampler.view()` instead of `sweep` (caught by
  `…SupervisorClassifiesFresh` on process count); removing `Reaper`'s identity fences (caught by
  `TestReapOrphanNvimFencesEveryRowOnItsIdentity` and by `TestReapOrphansCLIWithConfirmationReapsAndReports`,
  which names the unsafe signal: "a recycled pid was signalled: [terminated]", alongside #399's own
  `TestReapNeverSignalsARecycledPid` and `TestReapRefusesAChangedServer`). Passing an empty identity instead
  fails closed — every row is skipped, nothing is signalled — so the fence's dangerous direction is the one
  the launcher mutation measures.

### Task 17: Cost re-measurement and docs

- [x] **Step 1:** re-run #587's sampler cost measurement at maximum cardinality with the cwd probe enabled
  and record mean/max CPU and the percentage of one core at 60 s in `## Log`; it must stay inside the
  approved <0.2%.
- [x] **Step 2:** update `atlas/couch.md` (the resource monitor section) and `README.md` where the reaper or
  `couch --resources` is described; add the new env knob to the configuration list.
- [x] **Step 3:** a live check on this machine. **Substituted** — see the 2026-10-16 Task 17 revision:
  the running supervisor is a build from before this branch, so neither verb exists to invoke. An
  env-gated live conformance test (`cmd/internal/couchcmd/reap_orphans_live_test.go`) runs the same three
  claims against the real host instead: a real reparented `nvim --embed` at PPID 1 inside a managed
  checkout is named by the real sampler, an embed with a live parent in the same checkout under the same
  age gate is not, and the confirmed reap ends the real orphan with a real signal.
- [x] **Step 4:** `sdlc local-check run --name suite`, then `sdlc close --issue 583`.

## Non-goals

- No automatic reaping of an untagged process: confirmation is required, always.
- No change to nvim's TUI/embed fork behaviour (upstream), and no attempt to keep an embed alive.
- No #574 lifecycle-notice implementation; the existing alert seam carries the report until #574 lands.
- #587's proven-orphan RSS floor and alert thresholds are untouched; a tagged leak below that floor is
  still only reaped at the next launch sweep.
- No committed change to the `runtimebundle` lua mirror. It is gitignored build output that
  `make runtimebundle-generate` regenerates from `nvim/` on every build, so there is nothing to stage;
  the lua edits live in `nvim/` alone.
- No change to the scrollback viewer's one-sidecar-per-tag shape, though it has the same multiplicity as
  changelog (per agent). The class *was* considered: a scrollback embed's argv carries
  `scrollback-<tag>-<agent>.ansi`, which `TagFromNvimEmbedArgv` splits unambiguously at the last hyphen,
  so the argv fallback covers an overwritten sidecar. The changelog's argv has a variable trailing
  component count and no unambiguous split, which is exactly why only changelog moves to an
  artifact-keyed record.
- The quit path has no argv fallback. `ReapNvim` reads pid records only, so a second scrollback viewer
  whose per-tag sidecar the first one overwrote is not reaped at quit; the next launch sweep covers it —
  on its owner's death, or on the reparented rule if its pane closed first. The deleted `pkill -9 -f`
  pattern did catch that viewer, and also every other process whose argv mentioned the path, which is why
  #583 removed it. Recorded rather than fixed (M1 boundary review BR-3).

## Revisions

- **2026-10-16 — plan-quality gate round 1 (PQ-1 Important, PQ-2/3/4 Minor).** Reason: PQ-1 showed the
  changelog identity as designed cannot work — `ChangelogArtifactSet.Log` is `.md` not `.log`
  (`paths.go:765`), `componentPattern` admits `-` in tag/agent/session (`paths.go:14`) with the session
  optional (`paths.go:738-749`), so neither the filename nor argv splits unambiguously, and one
  `nvim-pid-<tag>-changelog` sidecar collides across a tag's per-(agent, session) viewers. Both options
  the finding offered were measured and rejected: keying the sidecar per agent/session still cannot
  recover the tag from the filename, and matching argv against a tag's expected paths crosses tags
  (`changelog-work-*` also claims `work-x`'s file), which is unsafe in a kill path.
  Delta: the changelog surface leaves the kind set and gets an **artifact-keyed** record
  (`<changelog-base>.nvim-pid`, a sibling of the existing `.openlock`), whose owner liveness comes from
  the openlock's opener pid rather than from a tag, and whose expected argv path is the base's exact
  `.md`; `NvimKinds()` is `{draft, scrollback, review}`. Task order changed (the changelog record is now
  Task 4, before the sweep that consumes it in Task 5). PQ-2: `TagLive` became `OwnerLive`, and the
  explicit quit path passes `false`. PQ-3: Tasks 2, 3, 5, 12, 13, 14, 15 and 16 now state one test
  strategy line naming the adversarial class and the guard. PQ-4: the seam is stated — `ProcessTable`
  gains `Command`, kills move onto `Signal`, and the reap path holds no subprocess seam, so `pkill` is
  unreachable rather than merely unasserted.
- **2026-10-16 — plan-quality rounds 2-3, and a false premise corrected.** Round 2 disposed PQ-1..PQ-4 and
  raised one Minor advisory (PQ-5, family `unstated-decision-input`): Task 5's changelog evidence source did
  not say what happens when `<base>.nvim-pid` exists but its `.openlock` sibling is gone. Delta: source 3 now
  states that an absent openlock means the owner exited cleanly (`opener/run.go:189`'s `defer rt.Remove`) and
  is therefore the *reap* case, a present openlock skips on a live pid and reaps on a dead one, and a
  non-integer pid is malformed and reaps behind the row's own predicates — and the rule is applied to every
  evidence source (absent / present / malformed). Round 3 returned CLEAN.
  The same pass corrected a premise this plan asserted twice: the `runtimebundle` lua mirror is **not** a
  tracked copy gone stale. `.gitignore:133` ignores `/cmd/internal/runtimebundle/assets/`, `git ls-files`
  returns nothing for it, `Makefile.local:263-266` regenerates it as a prerequisite of `build` and `test`,
  and a fresh generation reproduces `nvim/init.lua` byte-for-byte. Integration points, Task 4 Step 3 and
  Non-goals now say build output rather than drift; no separate issue is filed, because there is no defect.
- **2026-10-16 — Task 3 implemented; attribution widened from one predicate to three.** Reason: the plan's
  `ExpectedArgPath` ("the scope path argv must carry exactly") only fits the draft kind, and fitting the other
  two to it would have been unsafe rather than merely wrong. Measured on this host
  (`ps -eo pid,ppid,command`): the scrollback viewer's argv carries
  `<scopeDir>/scrollback-<tag>-<agent>.ansi`, and the obvious predicate `Paths.ScrollbackPrefix()`
  mis-attributes across tags (`scrollback-work-` also matches `scrollback-work-2-claude.ansi`, so quitting
  `work` would kill `work-2`'s viewer); a review pane's argv carries no pair path at all except the `-u`
  asset, and that asset's path is **checkout-relative** while a live embed outlives the checkout that spawned
  it (this reaper runs from a different worktree than the sessions it reaps).
  Delta: `NvimReapRow` carries `ExpectedArgPath` (exact argv field — draft), `ExpectArgvTag`+`ArgvTag`
  (attribution through `ScopePaths.TagFromNvimEmbedArgv`, the codebase's one parser — scrollback), and
  `ExpectedAssetBase` (base name of the pair asset in argv, from `reviewcmd.ReviewScript` — review).
  `nvimSidecar` mirrors the three, `nvimSidecarsFor` maps kind → attribution over `NvimKinds()`, and a kind
  that cannot be attributed is skipped (`argv serves another tag` / `argv runs another config`), never killed
  on `nvim --embed` alone; `TestNvimSidecarsGiveEveryKindArgvEvidence` fails a kind that arrives without a
  predicate. Task 5's three evidence sources build rows through the same three fields.
- **2026-10-16 — Task 3: stated behaviour the plan left open.** (a) *Record rule* — evidence merely missing
  (owner live, identity unreadable, argv unreadable) keeps the sidecar for the next pass; every decided
  outcome (kill, malformed record, pid gone, recycled identity, not-ours) removes it, so a zombie embed
  leaves its record to storage GC (`GCClassifications["nvim-pid"] = GCCollectable`). (b) *SIGKILL only* — no
  TERM→KILL escalation on this path: every caller has already established the owner is gone or quitting, so
  nothing is left to flush and quit is time-bounded. (c) *`Killed` is honest* — `nvimReapOutcome.Killed` is
  `Signal(...) == nil`, so a pid that races away between the identity re-read and the signal is not reported
  as reaped; Task 5's report and Task 14's health line read that field.
- **2026-10-16 — Task 3: one derivation replaced the two pinned spellings.** Reason: Task 1 pinned the quit
  path's inline spelling to `editorPathsOf`, and pinning two spellings to each other is the exact shape that
  let `review` drift out of both. Delta: the inline spelling in `runCleanupContext` is deleted;
  `lifecycleEditorPaths` is now `{sidecars []nvimSidecar}` and `editorPathsOf` →
  `nvimSidecarsFor(paths, paths.Tag(), paths.ScopeDir())` is the only place the kind→attribution mapping
  exists. Task 1's `TestNvimReapPathsDeriveEveryKindFromOneSource` is deleted as redundant against
  `TestNvimSidecarsForDerivesFromTheClosedKindSet` + `TestQuitPathAndReaperShareEditorPaths`. Artifact
  inventory follows: `lifecycle.go` drops family `nvim-pid` / binding `scoped-nvim-pid`, `osruntime.go` drops
  `draft` / `scoped-draft` / `scoped-nvim-pids`, `nvim_reap_os.go` is a ResolvedConsumer with no bindings (it
  consumes members on a `Paths` handed to it and calls no resolver of its own), and the pure `nvim_reap.go`
  joins `NonArtifactSources` — Task 2 had left it out of the inventory entirely, which
  `TestProductionArtifactReferencesAreExactlyClassified` reported as NEW.
- **2026-10-16 — Task 3 verified a premise Task 5 depends on.** `OSRuntime.DataDir` is the **scope** directory
  in a real launch (`runcli.go:110-139`: `env.DataDir = ScopedLaunchDataDir(dataDir, repoRoot)`), which is what
  `ResolveScoped`'s contract asks for ("an already-selected scope directory carried by a child process"), so
  both `ReapNvim`'s `ResolveScoped(r.DataDir, tag)` and `SweepOrphanNvim`'s `ResolveSelectedScope(r.DataDir)`
  are reading the composite layout, not the legacy flat root. No change needed; recorded because the two calls
  look like a data-dir/scope-dir mixup and Task 5 builds on both.
- **2026-10-16 — Task 4 landed; Task 6's "each config" is not one shape.** Reason: implementing the changelog
  pidfile meant reading the three existing spellings, and they are not copies of each other. `init.lua:24-34`
  and `scrollback.lua:46-56` are the identical `VimEnter` + `once` + `pcall(writefile)` block; `review.lua:791`
  is a **synchronous** `vim.fn.writefile` with no `pcall`, sitting inside the activation/render function so it
  re-publishes on every activation — which is load-bearing there, because the review embed can be restarted
  and rolled back, and a pid recorded once at `VimEnter` would go stale.
  Delta: Task 6 must not unify the four into one shared block on the assumption they are the same. The
  `VimLeave` removal belongs beside each config's own write (an autocmd for the two `VimEnter` configs and for
  the new changelog one; for `review.lua`, alongside its per-activation publish, and its removal must not
  delete a record a *later* activation just wrote). Extracting a shared `nvim/pidfile.lua` was considered and
  rejected for M1: the module would need two entry points to cover both shapes, it adds a non-Go production
  source to the artifact inventory, and the win is three deleted copies — worth revisiting only if a fifth
  config appears. Task 4 added the changelog block as a copy of the `init.lua`/`scrollback.lua` spelling, as
  planned, since that viewer is a plain read-only buffer with the same fork shape and no restart path.
- **2026-10-16 — Task 5 landed; the sweep decides per pid, and its changelog owner is the log it serves.**
  Reason: implementing the sweep turned up three decisions Step 3 had not stated, and one plan detail that
  could not survive contact with the artifact inventory.
  (a) *One snapshot, shared by all three sources.* `reapNvimSidecars` keeps its contract (ctx check +
  `Snapshot` + delegate) but its body is now `reapNvimRows(ctx, table, records, targets, rows)`, and
  `sweepOrphanNvim` takes the same snapshot once and hands it to the openlock liveness check, the argv
  attribution and the identity fence alike — three sources reading three moments would have compared a
  row against a process table the row had already moved past.
  (b) *Two records naming one pid are decided once.* The `decided` map is `pid → reason`: the first record
  to name a pid drives `decideOne` (and the only `Signal`), and a later record for that pid is removed
  when the first decision removes records, but emits no second outcome. They are two pieces of evidence
  about one process, not two processes; signalling twice would have made the report claim a reap it did
  not perform. Pinned by the "two sidecars naming the same embed" and "one embed named by both a sidecar
  and its argv" cases. This was found by two failing launcher tests, not by design: the first cut
  (`if decided[pid] { continue }`) skipped later targets outright and left their records on disk.
  (c) *Changelog `Owner` is `set.Log`, not the base.* Step 3 said "the base"; the base is a filename
  fragment (`changelog-work-codex-sid`) that names no process, while `set.Log` is the artifact the viewer
  holds open and the openlock it pairs with is a sibling of. `OwnerLive` comes from
  `openerLive(records, set.OpenLock, byPID)` — `ReadFile` error or a non-positive pid means dead, so an
  absent openlock reaps, as PQ-5 settled.
  (d) *The recognizer lives in `artifactpath`, and the inventory follows.* `ChangelogNvimPIDGlob()` and
  `ChangelogSetFromNvimPID(path)` join `ScopePaths` beside `NvimPIDGlob`/`TagFromNvimPID`, and
  `changelogSiblings(base)` is now the family's one sibling construction, so the constructor and the
  recognizer cannot drift on the `.nvim-pid` spelling (one `changelogNvimPIDSuffix` const). Enumeration
  moving out of `osruntime.go` into `nvim_reap_os.go` cost osruntime the `nvim-pid` family and the
  `selected-nvim-pid-globs` binding, and won nothing there: binding witnessing is per-file and
  one-directional, and a `Paths`/`ScopePaths` received as a *parameter* is never tracked by
  `resolverResults`, so `nvim_reap_os.go` stays a ResolvedConsumer with no bindings (its header comment
  now says so).
  (e) Tests landed in `nvim_reap_os_test.go`, not the `lifecycle_test.go` Step 3's Files block named: the
  sweep's own file already holds its harness, and a fake `ProcessTable` + a real temp scope directory are
  what the fifteen cases need. `fakeProcessTable` gained `snapshotErr` so a broken snapshot is provably
  refused rather than silently empty.
- **2026-10-16 — Tasks 6 and 7 landed; the live test cannot kill an owner, so it does not pretend to.**
  Reason: both tasks turned on measured host behaviour that the plan had assumed, and three of the plan's
  spellings could not survive it.
  (a) *Task 6 stayed four copies, pinned by behaviour instead of by each other.* The shared `nvim/pidfile.lua`
  Task 4's revision rejected stays rejected, but the reason is stronger now:
  `tests/nvim-pidfile-lifecycle-test.sh` runs **all four configs** through the same four properties
  (published at startup / removed on a clean exit / kept after a SIGKILL / never removed when a later embed
  republished it), 20 assertions. A copy that drifts fails its own case, which is the property the "one
  derivation" lesson actually wanted — a shared module would have proven the copies identical, not correct.
  The removal is guarded by "only a record that still names *this* pid", because `review.lua` republishes per
  activation and a restarted embed for one tag writes the same path first; that hazard is a tested case
  (`case_keeps_a_republished_record`, a `sleep 60` survivor's pid overwriting the record), not a comment.
  (b) *The lifecycle test quits nvim through an ack file, not `-c 'qa!'`.* Measured: `-c 'qa!'` runs before
  `VimEnter`, so the autocmd under test never registers and the test would pass by never publishing. A
  deferred `vim.wait` on `$PAIR_PIDFILE_ACK` inside `VimEnter` fires both `VimEnter` and `VimLeave` and is
  race-free. `--remote-send ':qa!<CR>'` does not quit a headless init.lua boot and `--remote-expr
  'execute("qa!")'` only works with the client already exiting 2 — both rejected as unreliable. The configs
  also refuse to boot without a `pair` binary on PATH (`retention.lua`: "cannot protect editor storage: …
  ENOENT … (cmd): 'pair'"), so the test builds one via `tests/lib/pair-test-binary.sh`.
  (c) *Task 7 is env-gated, not flag-gated.* `PAIR_LIVE_NVIM_REAP=1` + `t.Skip`, which is this package's live
  idiom (`session_owner_live_test.go`, `session_quiescence_live_test.go`); the plan's `-nvim-reap-live` flag
  belongs to `wrapcmd`'s `peer_live_test.go` shape, and adding a flag to `launcher` would have made it the
  only package with one.
  (d) *No owner is killed, because a dead owner takes the embed with it.* Step 1 said "kill the owner (the
  spawning parent)". Measured on this host: closing an `nvim --embed`'s stdin pipe ends it in **~31 ms**
  (exit status 1), and SIGKILLing a `nvim FILE` TUI parent removes its `nvim --embed` child within 3 s. The
  `sh -c '"$@" & exit 0'` owner this test first tried therefore never published a sidecar at all — the
  backgrounded nvim does not survive its shell. So an orphan is a process whose stdin is held open by
  something that outlived its slot, and the test supplies exactly that: it holds the pipe, and states owner
  death the way the sweep reads it — the tag absent from `liveTags`. That absence *is* the sweep's whole
  authorization for a tag kind (it never consults ppid there), so the case tests the rule that ships.
  The dropped third case (an orphaned embed of a **live** tag) became unreachable for the same reason; its
  meaning — a live tag is never decided — is the second case, and the `reapReasonReparented` branch stays
  pinned by the fake table's ppid-1 rows, which is where it is reachable from.
  (e) *The live run caught two things a fake cannot.* A process that has exited but not been waited on is a
  zombie whose argv the kernel will not report: the first gated run hit that (an embed killed by its own EOF),
  and the sweep said `argv unknown` and **kept the record** — real-kernel proof of the record rule, on the
  classifier's own restraint. And `waitSignal` blocked in `cmd.Wait()` for the full 5-minute go-test timeout
  when no kill landed; it is now deadline-bounded (10 s), which the mutation check
  (`if row.OwnerLive || row.Owner != ""`) proves turns a hang into four precise failures.
  Step 2's "expect FAIL before Task 3's behaviour" could not be run as written — Task 3 was already in place —
  so the teeth were proven by that mutation instead, and reverted (`git status` clean).
- **2026-10-16 — Task 8: the storage-inventory premise was wrong, and the atlas edits were four.** Reason:
  Step 1 named `atlas/storage-retention-io.md`'s "nvim-pid and changelog rows"; reading it first (as the step
  itself asked) shows there is no nvim-pid row. That file inventories **retention-IO anchors** — the paths that
  touch managed bytes under the retention guard, grep-checked by `storagegc.TestManagedIOInventoryAnchors` —
  and the pid records are not one: lua writes them with `writefile` outside the guard, exactly as it did
  before #583 (the changelog `.nvim-pid` follows its `.openlock` sibling's precedent), and `storagegc`
  classifies both (`GCClassifications["nvim-pid"]`, `gc.go`'s changelog suffix list). So the guard passes
  untouched and no row was invented to satisfy a plan sentence.
  Delta: `architecture.md` is the only atlas file mentioning `nvim-pid`, and it needed four edits —
  `:383` (the Alt+x reap still described the deleted `pkill -f` fallback and two publishing configs; now
  three kinds, four configs, `VimLeave` removal, and no subprocess seam), `:385` (the startup sweep still
  described tag-level collection; now per-pid over three evidence sources, the three attribution shapes, and
  the untagged class named as M2's, not this sweep's), `:1218` (the tag-only family list said
  `{draft,scrollback}`; `NvimKinds()` and the rename/GC enumeration that iterates it say three), and `:1256`
  (the sidecar row: three kinds, the review record resolved at its launch site rather than env-bound, the
  "only a record that still names this process" removal guard, identity-exact reaping, and the changelog's
  artifact-keyed record with the reason (tag, kind) cannot serve it).
- **2026-10-16 — M1 boundary review (FIX-THEN-SHIP): the closed pane was unreachable, so the sweep now
  decides live tags too.** Reason: BR-1 (Important) found that the rule Task 2 pins and Task 5 states —
  "owner live but reparented to launchd → kill" — could not fire for a tag-keyed kind: `deadTagSidecars`
  and `argvOnlyEmbeds` both dropped live tags and the quit path passes `OwnerLive: false`, so no
  draft/scrollback/review row was ever built with `OwnerLive: true`. The finding offered two fixes, emit
  live-tag rows or withdraw the claim. Withdrawal was drafted first and then rejected as unsound: the
  ~31 ms stdin-EOF measurement (Task 7 revision (d)) shows only that an embed *usually* dies with its
  parent, and the issue's own evidence is a 13-day survivor, so "a live tag's embed cannot linger at ppid
  1" would have generalized one measurement into a claim the 22 GB orphan contradicts — and the atlas
  already said as much ("the embed sometimes survives RPC-pipe EOF and gets reparented to launchd"). The
  leak was real: a session runs for days after its editor pane closes, and until now the embed that pane
  left behind waited for the session to end.
  (a) *Live tags are candidates.* `deadTagSidecars` → `tagSidecars` targets every tag whose record the
  globs find and passes `live[tag]` down; `nvimSidecarsFor` gained an `ownerLive` parameter (the quit path
  and `editorPathsOf` pass `false`, which is their own authorization); `argvOnlyEmbeds` no longer skips a
  live tag and states `OwnerLive: live[tag]`. `reapReasonReparented` is therefore reachable in production
  for all three tag kinds, and the guard did not move: ppid 1 **plus** the kind's full argv attribution
  plus the identity fence. Cost is near zero — source 2 already read argv for every nvim row before it
  checked liveness — plus one record read per live tag per kind, mostly a miss.
  (b) *Tests.* The two sweep cases that used the ppid-1 `embed()` helper for a live tag now give it a live
  parent, so each isolates what it names (a healthy live-tag embed is kept; a dead tag's record does not
  steal its live neighbour's viewer), and two new cases assert exactly one SIGKILL for a reparented
  live-tag embed — one reached through its sidecar, one through argv alone.
  `TestNvimSidecarsGiveEveryKindArgvEvidence` also pins that the liveness a caller states reaches every
  kind. The gated live test's live-tag case changed meaning with the code: it now asserts exactly one
  decision, reason `owner live`, no kill, record intact (PASS 3/3 in 0.176 s), and its header no longer
  calls the reparented branch unreachable. The live run still does not attempt a real reparenting — a
  disposable parent that leaves the embed alive at ppid 1 could not be built on this host (Task 7 (d)) —
  so that branch stays pinned by the fake table's ppid-1 rows.
  (c) *BR-2 (Minor, absent evidence read as owner death) is fixed at its root,* not with the suggested
  grandparent-pid check. `RunViewer` is a **held** child (`opener/runtime.go:161-176` ends in `cmd.Wait()`)
  and the openlock goes by `defer` on its return, so an absent lock really does mean the opener is gone
  and PQ-5's rule stands; the defect was that `opener/run.go` ignored the `WriteFile` error, leaving a
  viewer whose ownership was never recorded reapable by any other session's sweep. The write is now
  checked and the viewer refuses to open (exit 1, cause on stderr, no distiller launched, and a lock this
  opener never wrote is not removed), pinned by `TestRunChangelogRefusesAViewerItCannotClaim` — teeth
  proven by mutation, since reverting to `_ = rt.WriteFile` turns it RED on `code = 0, want 1`. A
  grandparent check would have added a second ownership model beside the openlock. The scrollback lock at
  `run.go:109` stays unchecked on purpose: it guards re-entrancy, not ownership, so a failed write costs a
  duplicate read-only viewer, not a live process.
  (d) *BR-3 (Minor, the quit path lost its argv fallback) is documented, not fixed* — see `## Non-goals`.
  (e) *BR-4 (Minor, repeated per-pass work) is half fixed and half disposed.* `byPID` is built once per
  pass (`indexByPID`) and handed to both the openlock liveness check and `reapNvimRows`. The second
  `Command(pid)` read stays: `argvOnlyEmbeds` reads argv to *attribute* and `decideNvimSidecar` reads it to
  *authorize*, and the authorizing read must sit beside the identity fence — carrying an earlier read into
  the decision would mix evidence from two moments, exactly what Task 5's revision (a) removed. It costs
  one argv read per argv-only target on a launch/quit path, and the reason is now a comment at the read.
  The atlas' startup-sweep paragraph (`:385`) was edited again for (a): its first two evidence sources
  described dead tags only, and it now states the live-tag rule — a dead owner is the whole authorization,
  a live tag's embed needs reparenting on top.
- **2026-10-16 — M1 boundary review round 2 (BR-5 Critical): the BR-2 fix tripped an inventory guard the
  verification never ran.** Reason: round 1's four findings were all addressed, and the fix for BR-2
  introduced a new one — `opener/run.go`'s new refusal message is a string literal in a `ResolvedConsumer`
  of the `changelog` family, which `artifactpath`'s `TestProductionArtifactReferencesAreExactlyClassified`
  rejects unless the exact literal is inventoried in `manifest.go`. My verification had covered
  `./cmd/internal/launcher/` and `./cmd/internal/opener/` only, so the red guard reached the review.
  Delta: the finding's suggested remedy was measured first and rejected — rewording to
  `"pair-changelog-open: claim viewer: %v\n"` still fails, because the token that fires is the `changelog`
  family token inside the *command prefix*, not the tail; no wording of a message from this file escapes
  the inventory. The fix is therefore the sanctioned shape: a terse `"pair-changelog-open: write %s: %v\n"`
  naming the record it could not write (parallel to `reviewcmd/run.go`'s already-listed
  `"pair-review-definition: write %s: %v\n"`) plus its entry in the file's existing changelog
  `goCallVocabularyValues` list, and `TestRunChangelogRefusesAViewerItCannotClaim` now asserts the refusal
  names that record path rather than a phrase. Verified with the whole `./cmd/internal/...` suite.
  Rule taken: a fix commit is verified against every package's tests — including grep/AST guards like
  `artifactpath`'s, which read production sources no test in the touched package imports — not only the
  new targeted test.
- **2026-10-16 — M1 boundary review round 3 (BR-6 Important): attribution has to name the owner, and the
  review kind's did not.** Reason: round 3 disposed BR-5 and found the class under it. `nvim_reap_os.go` set
  `ExpectedAssetBase: "review.lua"` as the review kind's whole attribution, and every tag's review pane runs
  that config. Pid records store no start stamp, so the identity fence only catches a reuse *after* the
  snapshot: a dead tag A's surviving `nvim-pid-A-review` naming a pid tag B's live review embed now holds
  passed every check — owner dead, name nvim, exact `--embed`, `review.lua` present — and SIGKILLed B's
  pane. A case in the classifier table pinned the unsafe behaviour outright ("review kind expects no scope
  path" → kill).
  Delta, stated once in the classifier: **a recycled pid is safe only for a kind whose argv names its
  owner.** `NvimReapRow` gained `ExpectArgvPath`, and `ClassifyNvimReap` now requires at least one
  owner-naming form (`ExpectArgvPath` / `ExpectArgvTag`) to be both stated and satisfied; a row stating
  neither returns the new `attribution unknown` and is never killed. `ExpectedAssetBase` is demoted to
  corroboration — it still refuses `argv runs another config`, but it names the kind, never the owner, so
  it cannot satisfy the requirement alone.
  *The review kind's owner check:* `nvimSidecarsFor` sets `ExpectArgvPath` plus
  `ReviewState = paths.ReviewTarget()`, and `decideOne` resolves that state before deciding
  (`reviewAttribution`), so `ExpectedArgPath` is the file **this tag's own** review state names. That state
  is the only place recording it: a review pane serves a file anywhere in the user's tree (`RunOpen` joins
  `LogicalDir(file)` to its base), so its argv carries no scope path. `reviewcmd.TargetFileOf` is the
  reader, exported beside `targetJSON` and pinned to it by `TestTargetFileOf`. A state that is gone,
  unreadable or names no file leaves the field empty — missing evidence, so the candidate skips and keeps
  its record (`reapKeepsRecord` gained `attribution unknown`), never a pass.
  *Residual gap, deliberate* (**wrong, and superseded by round 4 below** — the skip is not the safe
  direction, because the outcome that skips here also deletes the record): argv is fixed at spawn (`RunOpen` passes the file it was asked to open), while
  the state is rewritten per `:PairReview` and per prepare receipt, so an Alt+C **retarget** to a different
  file leaves the two naming different files and the embed unattributable — the sweep skips it and keeps the
  record. That is the safe direction (no wrong kill), the window is one retargeted pane, the embed normally
  dies with its pane (~31 ms on stdin EOF, Task 7 (d)), and the record is `GCCollectable`, so it does not
  accumulate. Closing it would need the record to carry the served file — the record-format change rejected
  below — and it is not worth a wrong-kill risk in the other direction.
  Two remedies were measured and rejected. Storing a start stamp in the record: `procutil.Identity` is a
  wall clock on darwin but **ticks since boot** on linux, so comparing one needs a per-OS parser and would
  either refuse every kill on linux or be inert there. Fencing on the record's mtime measures the wrong
  thing — a record is rewritten per activation, and mtime is not the process's start. Attribution through
  the owner's own state needs neither a new record format nor OS-specific time parsing.
  Tests: five review cases replace the one that pinned the kill (attributed by its tag's file → kill; argv
  serves another tag's document → `argv serves another path`; state names no file → `attribution unknown`;
  a kind-wide asset name attributes no owner → `attribution unknown`; argv runs another config →
  `argv runs another config`); the drop-invariant test gained an `attribution` mutation; the sweep gained
  the cross-tag regression at its own level (A's record and A's state, the pid holding a pane serving
  another file → no signal, stale record gone) and a "state names no file" case that keeps the record.
  `TestNvimSidecarsGiveEveryKindArgvEvidence`'s invariant is now "every kind carries owner-naming
  attribution", the drift guard for a fifth kind arriving with a kind-wide predicate. Atlas: the sweep
  paragraph (`:385`) and the sidecar row (`:1256`) now state the owner-naming rule and that the record
  alone is not the attribution.
  *Round 3's two Minor ARCH-DRY notes, disposed rather than silently dropped.* (i) The VimLeave
  "remove if it still names me" block in the four lua configs stays four copies, on the reasoning Task 6/7's
  revision (a) recorded and this round re-confirmed: `tests/nvim-pidfile-lifecycle-test.sh` runs all four
  through the same four properties (20 assertions), which proves each copy *correct* rather than identical,
  and a shared `nvim/pidfile.lua` would still need two entry points for the two publish shapes
  (`VimEnter`+`pcall` vs `review.lua`'s per-activation synchronous write) plus a non-Go production source in
  the artifact inventory. (ii) `gc.go`'s hand-listed changelog suffixes get a guard, not a refactor:
  `TestGCChangelogSiblingsAreAllRecognized` asserts every sibling of a session-carrying base — the viewer's
  `.nvim-pid` and the agent-keyed `.ready` included — yields a `changelog` candidate, so a suffix dropped
  from `ownerCandidates`' list turns it red (teeth proven by deleting `.nvim-pid` from the list: red on that
  one sibling, then reverted). Truly single-sourcing the literals in Go costs reflection, a sentinel probe,
  or a parallel field enumeration, each of which re-lists the seven members somewhere; guarding the
  behaviour removes the drift risk the finding names without the machinery.
- **2026-10-16 — M1 boundary review round 4 (BR-7 Critical): the review attribution is bound into the record
  at publish, because round 3's "safe direction" claim was false.** Reason: BR-7 found that attributing
  through `review-target-<tag>.json` — mutable state, rewritten by `:PairReview` and by every Alt+C
  activation of a live pane (`review/restore_client.lua:137`) while the pane's argv still names its spawn
  file — classifies the pane's own orphan as `argv serves another path`, and that reason is a *decided
  negative*: `reapKeepsRecord` drops the record on it. Review has no argv-only fallback (its argv carries no
  scope path, so `TagFromNvimEmbedArgv` attributes nothing), so deleting the record deletes the only name
  that process ever had, and no later pass can find it. That is the issue's own orphan class, reopened by
  the fix for the previous finding. Round 3 recorded this divergence as a deliberate residual and called it
  "the safe direction (no wrong kill)"; both halves were wrong — the divergence is the routine case rather
  than a narrow window, and the outcome deletes rather than merely skips. The paragraph above is marked
  superseded rather than deleted, so the false premise stays visible.
  Rule adopted (the finding's, and now the family's): **attribution evidence must be bound to the pid at the
  moment its record is written, and must not change for that pid's lifetime.** Mutable session state may
  corroborate a decision but cannot be its subject; a mismatch against state that moved after the record was
  written must keep the record, never drop it.
  Delta: `review.lua`'s publish writes two lines — the pid, then `vim.v.argv[#vim.v.argv]`, the file this
  embed was spawned with, read out of **its own argv** rather than from the state or from the activation's
  `file` parameter (which is precisely the value that drifts). `sidecarPID` → `sidecarRecord` (line 1 = pid,
  line 2 = served), with both call sites (`openerLive`, `reapNvimRows`); `nvimSidecar.ReviewState` →
  `ServedInRecord`, resolved in `reapNvimRows` beside the pid read instead of in `decideOne`;
  `reviewAttribution` deleted, and `reviewcmd.TargetFileOf` + `TestTargetFileOf` with it (dead once nothing
  reads the state). The other three configs keep writing one line, which `sidecarRecord` reads as "bound
  nothing".
  Why line 2 is sound where the state was not: it is written by the process it names, out of that process's
  own argv, which is fixed for the process's lifetime. A mismatch therefore *proves* the pid is not the
  process that wrote the record — a decided negative that may drop it — while an absent or empty line 2 (a
  legacy record, or a spawn shape that stops putting the file last) is missing evidence: `attribution
  unknown`, record kept. ARCH-ORDER: subject and evidence are now written at the same moment by the same
  actor. ARCH-FUNERAL: no negative outcome deletes a record that is still the only name for a process.
  Residual, narrower and stated: two tags serving the *same absolute file*, plus a pid wrap onto the other
  tag's record, could still collide. That is strictly narrower than BR-6's, which fired for any review pane
  whose state had moved at all. The closure path if it ever matters is a spawner-minted nonce carried in
  argv and bound into the record — rejected for M1 as a new spawn-shape contract bought for a case that also
  needs a pid reuse.
  Tests: the two rows the finding demanded (a reparented embed after a retarget must be KILLed, at reap and
  at sweep level; a cross-tag live pane serving a different file must be skipped) plus a legacy one-line
  record and an empty line 2, both kept. Teeth proven by mutation: restoring the state read (BR-6 semantics)
  fails exactly those three retarget rows and nothing else; reverted, the package is green. The lifecycle
  test reads line 1 through `record_pid` and asserts review's line 2 is byte-equal to the file it launched
  with, and that the other three configs write no second line — 21 assertions over the four configs.
  Atlas `:385` and `:1256` re-edited to name the record's second line as review's attribution and to state
  the binding rule; `workshop/lessons.md` gains the BR-7 rule (bind evidence at write time; check what a
  decided negative *deletes* before calling a skip safe).
- **2026-10-16 — M1 boundary review round 4 (BR-7 Critical): review attribution is bound into the record at
  publish, and round 3's "residual gap, deliberate" is withdrawn as wrong.** Reason: BR-7 measured what
  round 3 had asserted away. `:PairReview` and an Alt+C activation of a live pane
  (`nvim/review/restore_client.lua:137`) both rewrite `review-target-<tag>.json` while the pane's argv keeps
  naming its spawn file, so the divergence is the *routine* case for any pane retargeted after it opened, not
  a one-pane window. And the outcome is not "skips and keeps the record": `reapReasonOtherPath` is a decided
  negative, `reapKeepsRecord` drops the record on it, and review has no argv-only fallback (its argv carries
  no scope path, so `TagFromNvimEmbedArgv` attributes nothing) — the embed loses the only name it ever had.
  That is the issue's own orphan class, reopened by the fix to the previous finding.
  Rule adopted (the finding's, now the family's second): **attribution evidence must be bound to the pid at
  the moment its record is written and must not change for that pid's lifetime.** Mutable session state may
  corroborate a decision; a mismatch against it must keep the record, never drop it. ARCH-ORDER: subject and
  evidence are written at the same moment by the same actor. ARCH-FUNERAL: no negative outcome may delete the
  only record naming a live process.
  Delta: `nvim/review.lua`'s publish writes two lines — the pid, then `vim.v.argv[#vim.v.argv]`, the file this
  embed was spawned with, taken from its own argv rather than from the activation's `file` parameter (which is
  exactly the mutable value). `sidecarPID` → `sidecarRecord(raw) (pid, served, ok)` (line 1 = pid, line 2 =
  served), both call sites (`openerLive`, `reapNvimRows`) updated; `nvimSidecar.ReviewState` →
  `ServedInRecord`, bound where the record is read; `reviewAttribution` deleted; `reviewcmd.TargetFileOf` and
  `TestTargetFileOf` deleted as dead once nothing reads the state. The other three configs keep writing one
  line, which `sidecarRecord` reads as "bound nothing".
  Soundness: line 2 is written by the process it names, out of that process's own argv, which cannot change
  for its lifetime — so a mismatch *proves* the pid is not the process that wrote the record (a decided
  negative that may drop it), while a record with no line 2 (a legacy one, or an empty argv tail) is
  `attribution unknown` and is kept.
  Residual, strictly narrower than BR-6's: two tags serving the *same absolute file*, plus a pid wrap onto the
  other tag's record, could still collide. Closure path if it ever matters: the spawner mints a nonce into
  argv and the record binds it — rejected for M1 as a new spawn-shape contract bought for a case that also
  needs a pid reuse.
  Tests: the two rows the finding demanded plus two more. Reap level — a quit-path pane retargeted after
  spawn (kill), a live tag's retargeted embed reparented to launchd (kill, `reparented to launchd`), a
  cross-tag live pane serving a different file (skip, stale record dropped), a legacy one-line record and an
  empty line 2 (both `attribution unknown`, kept). Sweep level — the reparented-after-retarget kill with the
  drifted state present on disk, the cross-tag skip, the legacy record kept.
  `TestNvimSidecarsGiveEveryKindArgvEvidence`'s review branch now asserts the target states *where* the field
  comes from (`ServedInRecord`) and carries no value of its own, so a scope-derived path cannot come back.
  Teeth proven by mutation: restoring the state read (BR-6 semantics) reddens exactly the three retarget rows
  and nothing else; reverted, the package is green. `tests/nvim-pidfile-lifecycle-test.sh` reads line 1
  through `record_pid` and asserts review's line 2 is byte-equal to the file it launched with, and that the
  other three configs write no second line — 21 assertions across the four configs.
  Atlas `:385` and `:1256` re-edited (the record's second line is review's attribution; the binding rule;
  "the pid a record names is not itself the attribution"). `workshop/lessons.md` gains the BR-7 rule.
- **2026-10-16 — Task 9 implemented; the age derivation had to become platform-aware.** Reason: the task's
  title asserts "the identity stamp is a wall clock", and that is true only on darwin. Measured in the tree
  rather than assumed: `table_darwin.go:31` builds `"<sec>.<usec>"` with two `FormatInt` calls, so the
  fractional field is an *integer* microsecond count (".5" is 5us, not half a second — reading it as a
  fraction would date every birth up to 100ms late, an error an hours-wide threshold absorbs silently), while
  `command_parse.go:91` sets linux's token to field 22 of `/proc/pid/stat`, ticks since boot. The package
  already carries the precedent for handling exactly this asymmetry: `strict_identity.go` switches on
  `runtime.GOOS` and documents why linux adds its boot ID ("start ticks repeat across reboots"), so the age
  derivation follows it rather than inventing a second convention.
  Scope measured before deciding how far to go: `Table` and `Identity` exist on linux (`table_linux.go`,
  `identity_linux.go`), so a linux caller can legitimately hold a token, but the *sampler's* per-pid probes
  do not — `resource_unsupported.go:5` returns nil `resourceCalls` for `!darwin || !cgo`, so `Usage`,
  `Command` and `Environment` all answer `ErrUnsupported` there, and #587's sweep already reports nothing on
  linux for want of RSS and argv. M2's sweep inherits that boundary; converting linux's token does not enable
  it. What the conversion buys is narrower and still worth its ~15 pure lines: one function that is honest on
  both platforms instead of a trap for the next caller who holds a linux row, and both token shapes
  fixture-tested everywhere.
  Delta: `procutil.StartTimeFromIdentity(identity string) (time.Time, bool)` — the token in, the birth out,
  so a caller holding a snapshot row pays no syscall, which is what Task 13's cost gate assumes. Darwin parses
  the wall clock; linux converts ticks through `/proc/stat`'s `btime` and `USER_HZ`; any other platform
  reports false. Three pure parsers sit beside it (`ParseIdentityWallClock`, `ParseIdentityTicks`,
  `ParseBootTime`) so both token shapes are fixture-tested on every platform, and `false` is contracted to
  mean *age unknown*, never *age zero* — an unknown age is not an orphan, which is M1's family rule applied
  to the new evidence.
  Two assumptions are stated rather than hidden, and one of them CI cannot check. `USER_HZ` is hardcoded at
  100 (pure-Go linux cannot call `sysconf`; every linux target this module builds for reports 100, which is
  what `ps` and gopsutil assume too). The live birth test discriminates it — a wrong rate moves a
  just-started child's birth far outside the 5 s window — and it no longer skips on linux, but `merge-check`
  only compiles (`bootstrap.sh` execs `weave compile`) and runs no `go test`, so nothing in CI executes it
  today; on this host it measures darwin. `btime` is read per call, not cached: it is one small file, only
  for rows that already cleared the cheap gates, and a cache would survive a reboot and then date every
  process from the previous boot.
  The red step earned its keep: the first run failed because the test passed a pid where the function takes a
  token, which is why the name is `StartTimeFromIdentity` and not the plan's `ParseIdentityTime` — a function
  that does IO on one platform should not be called "parse", and the argument should not be mistakable for a
  pid.
- **2026-10-16 — Task 12/13 signatures corrected by measurement.** Reason: the drafted classifier derived
  its own age from `row.Identity`, but `procutil.StartTimeFromIdentity` switches on GOOS and reads
  `/proc/stat` on every linux call (`identity_age.go`), so Task 12's "pure" step would have done IO off
  darwin and its table test would have needed a platform-shaped identity token to stay deterministic.
  Delta: the sampler — which already owns the IO and the sample instant — derives the age from the stamp the
  table carries (still no extra syscall) and hands it over in a `ResourceOrphanProbe` (cwd, cwdKnown, age,
  ageKnown); `ClassifyOrphanNvim` consumes that bundle plus the argv and RSS the sample already holds, so no
  fact is stored twice (ARCH-DRY) and couchcore stays free of platform switches. `OrphanNvim.Name` was dropped
  (provably `nvim`; the section header says it), and `ManagedRoots` takes `[]ResourceSlot` rather than the
  inventory, whose `Complete`/`Omitted` are not this rule's business — an incomplete inventory narrows the
  report, which is the safe direction, and it is already an error line in the sample.
  Two additions the draft did not name: `ResourceCost.CwdCalls`, so Task 17's re-measurement attributes the
  new probe instead of inferring it from the total; and the sweep's rows are removed from `v.Unattributed`
  when reported, because one process under two section headings reads as two findings. Task 12 and Task 13
  landed as one commit pair rather than the drafted two messages, since the classifier has no consumer until
  the sampler wires it.
- **2026-10-16 — Task 14 keeps the named orphan in the persisted ring.** Reason: the suppression added in
  Task 13 removes a swept process from `v.Unattributed`, and `ResourceFrame` persists exactly that slice, so
  the 24 h history would have stopped recording the offender it exists to record — the ring is what an
  operator reads to answer "did we see the 22 GB nvim before anyone acted?". Delta: `ResourceFrame` gained
  `OrphanNvim []OrphanNvim` (additive, version stays 1; upgrade reads old history unchanged, downgrade loses
  continuity into a fresh memory ring on the loud best-effort path, and `ResourceCost.CwdCalls` had already
  moved the frame schema), `ValidateResourceHistory` bounds it at `ResourceMaxProcesses` via a new
  `frameWithinBounds` predicate, and the sampler cost line reports `cwd %d`. No consumer reads the frame's
  offender slices back — `historySummary` reads only `At` — so this is forensic value, not behaviour.
- **2026-10-16 — Task 15 had to solve alert routing, which the draft did not name.** Reason: every #587
  effect carries the owning slot's `WorkspaceIdentity`, and the dispatcher resolves the hub from it; an
  orphan's slot is gone by definition, so a workspace-less effect resolves to nothing and its alert would
  sit pending forever — silently dead, which is the failure mode this issue exists to end. Delta: the
  orphan episode is addressed by the managed checkout the sweep matched, taking that root's live slot's
  `WorkspaceIdentity` whole, so the existing `hubBindingAt(owner(id))` path resolves it and no second
  routing rule exists (ARCH-DRY). `ResourceEffect` gained `Age` and `Root` for the body's evidence, and
  `advance`'s `value float64` became a `ResourceEffect` so each metric carries its own evidence rather
  than a float whose unit depends on the metric. Two structural moves fell out: the orphan classification
  now runs above the episode-retention loop so its episodes are `visited`, and `nvim/` joined the
  process-keyed prefixes (extracted as `processEpisodeKey`) so a reaped orphan's episode clears.
- **2026-10-16 — Task 16's registry instruction was wrong, and its named fake cannot drive the reaper.**
  Reason (registry): the draft said to declare the verb in `couchcore/ops.go:154` with `ConfirmRequired` and
  route it through `operationdispatch.go` / `slot_operations.go`. `TestDispatchTableIsIdenticalToTheDeclaredOperationSet`
  (`run_test.go:578`) audits `Dispatch()` ≡ `couchcore.OperationNames()`, so declaring it obliges a dispatch
  entry — and the verb it is a sibling of, `--resources`, is declared nowhere: it is a `cliMessage` op parsed in
  `cli.go` and answered by the supervisor over the broker socket. Delta: `--reap-orphans` follows that shape
  exactly (parse in `cli.go`, route in `messages.go`, answer in `message_service.handle`), and `ops.go`,
  `operationdispatch.go` and `slot_operations.go` are untouched. The confirmation authority moved with it:
  `couchmessage.ValidateRequest` refuses an unconfirmed `reap-orphans` and refuses any second field, which is
  strictly stronger than a registry flag — a hand-written request cannot reach the classification at all, and
  the reflection audit the `resources` op already had now covers the new one.
  Reason (fake): the draft named `couchcore.FakeProcOps`, which implements `ProcOps` (`Current`, `Exists`,
  `Identity`, `Signal`, `SignalGroup`) and has no `Snapshot`. `launcher.Reaper` plans from a snapshot precisely
  because #399's close review BR-15 requires pid, parent and start identity to describe the same process at the
  same moment, so no `ProcOps` fake can drive it. Delta: the reap is tested against a `launcher.ProcessTable`
  fake (`fakeOrphanTable`: rows, per-pid current identity, `diesOn` for the escalation cases, recorded signals,
  injectable snapshot error), which is the seam `Reaper` actually holds.
  Three structural decisions the draft did not name, all forced by measurement:
  - **Who classifies vs who kills.** The supervisor classifies — whether a process is unattributed is a fact
    about the whole slot inventory, and a caller guessing it from a projected report would be guessing about a
    kill. The CLI signals, on `peek`'s precedent: seconds of escalation must not sit on the console's serial
    queue, and `couchmessage.AdmissionTimeout` is 2 s (`broker.go:13`), far short of a fresh pass plus a
    SIGTERM grace. Hence the 2-minute `orphanReapBudget` on the CLI side.
  - **The fresh pass belongs to the worker.** `resourceSampler`'s invariant is one worker owner with handlers
    reading only the cache, and `pass` reads `s.state.Provenance`, so a throwaway second sampler was not
    available. Delta: a `sweeps chan chan resourceSweep` served inside `run` — the request hands the worker a
    reply channel and the worker performs the pass, so the sweep *is* the loop's next pass rather than a second
    one. `pass` now returns whether it completed, because a pass the lifetime cut short leaves an older view in
    the cache and reporting that as fresh is how a reap would kill from a minute-old decision; `sweep` turns
    `fresh == false` into an error, and a sampler with no worker fails loudly rather than hanging to the
    deadline.
  - **The cadence gets one home.** `lifecycle.go:518`'s inline `Reaper` literal became `launcher.TreeReaper(table)`,
    so the second reaper inherits #399's measured 3 s / 2 s / 50 ms instead of quietly inventing a shorter fuse
    (ARCH-DRY).
  Two smaller ones: the whole tree is reaped, not the pid, because killing only the editor reparents its
  language servers to launchd and manufactures the next #583; and a skip exits 0 while a SIGKILL survivor or a
  refusal exits 1, since a skip is the world having moved and the leak the operator confirmed against is gone
  either way. The wire projection can truncate `OrphanNvim` at the detail limit, so the CLI prints
  `result.Resources.Errors` as `Coverage:` lines on stderr ahead of the outcomes — a row this verb did not
  consider is said before the fact, not after it.
- **2026-10-16 — Task 17 Step 3's manual live check was not performable, and one M1 measurement was wrong.**
  Reason (the check): Step 3 said to run `couch --resources` and `couch --reap-orphans` on this machine. The
  supervisor serving this slot's fleet was started from a build that predates this branch, so neither the
  `reap-orphans` op nor the orphan section exists in it, and restarting the fleet's Couch is the operator's
  call, not a worker slot's. Delta: the check became an env-gated live conformance test in the repo's
  established idiom (`launcher/nvim_reap_live_test.go`, `procutil/resource_darwin_test.go`),
  `couchcmd/reap_orphans_live_test.go` under `PAIR_LIVE_ORPHAN_REAP=1`. It asserts Step 3's three claims
  against the real host rather than against a running supervisor: a real reparented embed inside a managed
  checkout is named by the real sampler over the real table (identity, cwd, root and age all read natively,
  no synthetic field), an embed with a live parent in the same checkout under the same age gate is not named,
  and `reapOrphanNvim` over `launcher.TreeReaper(launcher.OSProcessTable{})` ends the real orphan — verified
  by the kernel no longer reporting that generation for the pid, not by a poll that went quiet. The refusal
  half of Step 3 is already covered where it lives: `TestReapOrphansCLIRefusesWithoutConfirmation`.
  Reason (the measurement): M1's `launcher/nvim_reap_live_test.go` header records that "a disposable parent
  which leaves the embed alive at ppid 1 … this host does not offer one". It does. Two constraints had to be
  satisfied together, and the first two attempts each satisfied one: the embed's stdin must be **held open**
  (M1's own ~31 ms measurement), and it must be **redirected explicitly** — bash sends an asynchronous
  command's stdin from `/dev/null` in a non-interactive shell, so an `os.Pipe` handed to `cmd.Stdin` and
  inherited reaches the embed as an immediate EOF. Measured 2026-10-16 with the write end open the whole
  time: the inherited-pipe embed was gone from the kernel table within 150 ms of its shell exiting; the same
  embed behind `< fifo` with this process holding the write end stayed at PPID 1 in state `S` for the whole
  poll. Delta: `spawnEmbed` builds the orphan that way (a `syscall.Mkfifo` in the test's own temp scope, the
  write end opened in a goroutine because a FIFO's two opens rendezvous), uses one script for both cases so
  PPID 1 is the only difference between them, and takes the pid from a sidecar file rather than the shell's
  stdout — in the live-parent case the shell never exits, so there is no `Wait` to close a stdout copy
  against. The launcher header's stale claim is corrected with the measurement and a pointer.
  Two ways the live-parent case could have passed for the wrong reason, both closed after its first green
  run: `exec.CommandContext` under the spawn function's own `defer cancel()` SIGKILLs the shell the moment
  that function returns, reparenting the embed to PID 1 *before the sweep runs*; and the verdict alone
  cannot distinguish "the rule held" from "the embed died or was reparented mid-run", so the test re-reads
  the kernel's table after the sweep and fails unless the embed is still there, still holding its spawn
  identity, and still not at PID 1.
  The live test was mutation-checked, not just green: relaxing `UnattributedShape`'s `p.PPID == 1` to
  `>= 1` makes the live-parent case fail on its own terms ("the sweep named 16284, whose parent was still
  alive"), and the same mutation raised that pass's native probes from 2 to 122 argv+cwd reads
  on a host running ~30 pair embeds — the PPID-1 gate is what bounds the probe Step 1 costs.
