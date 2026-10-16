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
| pair-spawned, untagged or unreaped | `changelog` viewer passes no `PAIR_NVIM_PID_FILE` (`opener/run.go:211-217`) and `nvim/changelog.lua` has no pidfile block; `review` writes one (`review.lua:791-792`) but `"review"` is absent from the reap/sweep kind lists (`osruntime.go:947,973`, `lifecycle.go:195,297`) and from `TagFromNvimEmbedArgv` (`artifactpath/paths.go:318`) | M1: close the kind set, tag the changelog viewer, reap all four kinds |
| not pair-spawned (a bare interactive `nvim` typed in a pane) | pair never spawned it, so it can carry no pair tag; argv is bare, so no scope path exists to match | M2: detect by ppid 1 + managed cwd + age, report, reap on confirmation |

pair cannot make nvim's forked embed die with its TUI — that is upstream behaviour. What pair owns is
(a) tagging and reaping everything it spawns, and (b) detecting and reporting the rest. This plan's
answer to Done-when #2 is that split. It is raised with the TL in the plan post; the reply is recorded
in the issue's `## Log` as a scope revision if it moves the design.

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `NvimKinds` / `Paths.NvimPIDs` / `ScopePaths.NvimPIDGlobs` | `cmd/internal/artifactpath/paths.go` | new |
| `TagFromNvimEmbedArgv` (changelog marker) | `cmd/internal/artifactpath/paths.go:318` | modified |
| `NvimReapRow` + `ClassifyNvimReap` | `cmd/internal/launcher/nvim_reap.go` | new |
| `SweepOrphanNvim` / `ReapNvim` / `ReapNvimContext` | `cmd/internal/launcher/osruntime.go:942-990`, `lifecycle_os.go:34-60` | modified |
| `editorPathsOf` + quit-path inline literal | `cmd/internal/launcher/lifecycle.go:193-196,294-299` | modified |
| `UnattributedCandidate` (extracted from #587) | `cmd/internal/couchcore/resource.go:301` | new |
| `OrphanNvim` + `ClassifyOrphanNvim` + `ManagedRoots` | `cmd/internal/couchcore/resource_orphan.go` | new |
| `ResourcePolicy.OrphanNvimAge` | `cmd/internal/couchcore/resource_policy.go:11-77` | modified |
| `ResourceView.OrphanNvim` (+ projection) | `cmd/internal/couchcore/resource.go`, `couchcmd/resources_projection.go` | modified |
| `ClassifyNvimReap` decision table | `cmd/internal/launcher/nvim_reap_test.go` | new |

- **`NvimKinds`** — the closed set `{draft, scrollback, review, changelog}` of embedded-nvim surfaces pair spawns.
  - **Relationships:** 1:N with the pid sidecar family `nvim-pid-<tag>-<kind>`; consumed by GC retention, the rename inventory, the quit path, the reaper and the sweep.
  - **DRY rationale:** today the set is spelled in six places with three different members (`gc.go:253` has review, the reaper does not; nothing has changelog). #583 is that drift: a kind nobody reaps is a kind that leaks. ARCH-DRY.
  - **Future extensions:** a new editor surface adds one string and derives everywhere; a drift pin fails if a consumer hardcodes a set again.
- **`ClassifyNvimReap`** — decides one pidfile-derived candidate: kill, or skip with a reason. No IO, no clock, no signals.
  - **Relationships:** 1:1 with a `NvimReapRow` built by the IO shell from one `procutil` snapshot; N:1 with a tag.
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
| `procutil.Source.Identity` | `cmd/internal/procutil/source.go` | new | existing `nativeIdentity` |
| `launcher.ProcessTable` (`OSProcessTable`) | `cmd/internal/launcher/session_reap.go:152-175` | reused | `syscall.Kill`, `procutil.Table/Identity` |
| `couchcore.ProcOps` / `FakeProcOps` | `cmd/internal/couchcore/procops.go:52-275` | reused | identity-fenced `Signal` |
| `resourceSampler.pass` candidate probe | `cmd/internal/couchcmd/resources.go:144-177` | modified | one bounded `Cwd` call per gated candidate |
| `reap-orphans` operation | `cmd/internal/couchcore/ops.go`, `couchcmd/cli.go`, `couchcmd/slot_operations.go`, `couchcmd/reap_orphans.go` | new | confirmation gate + `ProcOps` |
| `nvim/changelog.lua` pidfile block; VimLeave removal in the four lua configs | `nvim/changelog.lua`, `nvim/init.lua:24-34`, `nvim/scrollback.lua:47`, `nvim/review.lua:791` | new/modified | `PAIR_NVIM_PID_FILE` |

- **`procutil.Cwd`** — one `proc_pidinfo` call per pid, added to the existing `darwin && cgo` libproc block that already carries `resource_table`/`resource_identity`/`resource_usage`; unsupported builds return an explicit error like `identity_other.go`.
  - **Injected into:** the sampler's candidate loop (through `Reader`, so `samplerReader` fakes it) and nothing else — the classification stays pure.
  - **Cost:** charged to `Cost.NativeCalls` like every other native attempt, and only for rows that already pass ppid/name/argv/age, so the envelope stays inside #587's measured budget (ARCH-CONSTRAINTS).
- **`reap-orphans`** — follows the house confirmation pattern exactly: `slotOperationFlag` wiring, refusal `"%s requires --confirm"`, the server-side second gate `Response{Code:"confirmation-required"}`, and `couchcore.OperationConfirms` as the single source (`couchcmd/cli.go:364-401`, `slot_operations.go:124-126`, `ops.go:154`).

## Operating envelope (ARCH-CONSTRAINTS)

- Interaction path: a 60 s background sample plus an operator-invoked verb. No keystroke path is touched.
- Per pass: one `kern.proc.all` sysctl (existing), `Command` only for nvim candidates over the RSS floor (existing), **plus `Cwd` only for rows that are ppid 1, name nvim, argv `--embed`, and older than `OrphanNvimAge`** — typically 0-3 rows on this machine, hard-capped by `ResourceMaxProcesses`.
- Budget: #587 measured 0.12% of one core at 60 s against a <0.2% approval. One extra `proc_pidinfo` per gated candidate is ~1 µs; the plan re-measures at maximum cardinality and reports the number rather than assuming it.
- Overload: an unreadable cwd yields `CwdKnown == false` → the candidate is **not** classified (never reaped on missing evidence) and the pass records one bounded error string.
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
- `Cwd` returns a path from the kernel, not from a config file; containment is a cleaned path-boundary
  prefix match, so `/work/tree2` is never inside `/work/tree`.
- Nothing new reads credentials; no secret reaches a log line, an argv, or a fixture. The report names
  pid, age, cwd and RSS only.
- Blast radius: the confirmed verb signals only rows the pure classifier admitted, one pid at a time,
  identity-fenced; a process with a live parent is unreachable by construction because ppid 1 is required.
  Tests run against `FakeProcOps`/fake readers and never signal a real process, except the flag-gated
  live conformance test, which spawns and kills only its own child.

## Funeral (ARCH-FUNERAL)

- `nvim-pid-<tag>-changelog` joins the existing sidecar family: created by the changelog viewer's Lua at
  startup, last needed by the reaper, removed by the reaper after a kill and by VimLeave on a clean exit,
  and covered by `storagegc` retention through `gc.go:253`'s kind loop — so its growth bound is the
  family's existing one (one file per tag per kind), not a new family.
- No new durable store: the orphan classification is in-memory per pass, exactly like #587's episodes
  after its simplification. The report dies with the supervisor; the history file is unchanged.
- The killed process is the residue this issue removes; nothing is created that outlives its owner.

---

## Chunk 1: M1 — tagged embeds are reaped by exact identity

### Task 1: One closed kind set

**Files:**
- Modify: `cmd/internal/artifactpath/paths.go` (near `NvimPIDChecked:674`, `RenameArtifacts:552`, `NvimPIDGlob:286`)
- Modify: `cmd/internal/artifactpath/gc.go:253-255`
- Modify: `cmd/internal/launcher/lifecycle.go:193-196,294-299`, `cmd/internal/launcher/osruntime.go:947,973`
- Test: `cmd/internal/artifactpath/paths_test.go`, `cmd/internal/launcher/lifecycle_test.go`

- [ ] **Step 1: Write the failing drift pin.** In `paths_test.go`, assert `NvimKinds()` equals
  `{draft, scrollback, review, changelog}`, that `Paths.NvimPIDs()` resolves exactly one sidecar per kind,
  and that `gc.go`'s retention match, `RenameArtifacts`, `editorPathsOf` and both `osruntime` loops each
  derive from it (each consumer test builds its expectation from `NvimKinds()`, never from a literal).
- [ ] **Step 2: Run it, expect FAIL** (`undefined: NvimKinds`).
  `go test ./cmd/internal/artifactpath/ -run 'NvimKinds|NvimPID' -count=1`
- [ ] **Step 3: Implement** `func NvimKinds() []string`, `func (p Paths) NvimPIDs() []string`,
  `func (p ScopePaths) NvimPIDGlobs() ([]NvimPIDGlob, error)` (`NvimPIDGlob{Kind, Pattern string}`), and
  rewrite the six consumers to derive. Keep the quit-path literal at `lifecycle.go:193-196` spelled at its
  call site — `TestManagedIOInventoryAnchors` requires each scoped member to be consumed where resolved —
  but let it range over `NvimKinds()`.
- [ ] **Step 4: Run the artifactpath + launcher packages, expect PASS.**
  `go test ./cmd/internal/artifactpath/ ./cmd/internal/launcher/ -count=1`
- [ ] **Step 5: Commit** `#583: single-source the nvim pid sidecar kind set`

### Task 2: The reap decision, pure

**Files:**
- Create: `cmd/internal/launcher/nvim_reap.go`, `cmd/internal/launcher/nvim_reap_test.go`

- [ ] **Step 1: Write the failing table test** covering every branch, each asserting the verdict *and* the
  reason string: pid ≤ 0; empty identity; short name not `nvim`; argv without an exact `--embed` field;
  argv without the expected artifact path (draft/scrollback/changelog); expected path empty (review) so
  argv need not carry one; tag live **and** ppid ≠ 1 → skip `owner live`; tag not live → kill; tag live but
  ppid 1 → kill (the closed-pane case #583's 22 GB orphan is an instance of).
- [ ] **Step 2: Run, expect FAIL.** `go test ./cmd/internal/launcher/ -run ClassifyNvimReap -count=1`
- [ ] **Step 3: Implement**

```go
type NvimReapRow struct {
	Kind, Tag       string
	PID, PPID       int
	Identity        string
	Name            string
	Args            []string
	ArgsKnown       bool
	TagLive         bool
	ExpectedArgPath string // "" for kinds whose argv carries no scope path (review)
}

func ClassifyNvimReap(row NvimReapRow) (kill bool, reason string)
```

  Exact `--embed` matching uses a field comparison, not `strings.Contains` (the #587 classifier's rule,
  `couchcore/resource.go:326-332`).
- [ ] **Step 4: Run, expect PASS.**
- [ ] **Step 5: Commit** `#583: pure identity-exact nvim reap decision`

### Task 3: The reaper kills on verified identity, never by name

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go:937-990`, `cmd/internal/launcher/lifecycle_os.go:34-60`
- Modify: `cmd/internal/procutil/source.go` (expose `Identity`)
- Test: `cmd/internal/launcher/osruntime_test.go`, `cmd/internal/launcher/session_reap_test.go` fakes

- [ ] **Step 1: Write the failing tests** with the existing `fakeProcessTable` (`session_reap_test.go:28-60`)
  extended with argv: a pidfile naming a live `nvim --embed` whose tag is dead → exactly one `SIGKILL`
  recorded for that pid; a pidfile naming a recycled pid (identity differs at signal time) → **no** signal;
  a pidfile naming a non-nvim process → no signal; a pidfile naming an nvim serving a different draft path
  → no signal; two kinds for one tag → both reaped. Assert `pkill` is never invoked (the fake records
  subprocess attempts).
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement.** `ReapNvim`/`ReapNvimContext` build one `procutil` snapshot, one row per
  pidfile across `NvimKinds()`, decide with `ClassifyNvimReap`, then signal through the identity-fenced
  path (`Reaper.signalAll`'s rule) and remove the sidecar only after a decided outcome. **Delete `pkillF`**
  and both pattern kills (`osruntime.go:938,956-957`, `lifecycle_os.go:49-58`): the argv scan is the
  pidfile-less fallback and is now per-pid exact.
- [ ] **Step 4: Run launcher + procutil, expect PASS.** `go test ./cmd/internal/launcher/ ./cmd/internal/procutil/ -count=1`
- [ ] **Step 5: Commit** `#583: reap embedded nvim on verified identity, drop name-based pkill`

### Task 4: The sweep reaps untagged-by-sidecar embeds per pid

**Files:**
- Modify: `cmd/internal/launcher/osruntime.go:960-990` (`SweepOrphanNvim`)
- Test: `cmd/internal/launcher/lifecycle_test.go`

- [ ] **Step 1: Failing test:** a snapshot containing an `nvim --embed` whose argv carries a dead tag's
  draft path is reaped by pid; one whose tag is live and whose ppid is not 1 is untouched; one with no
  scope path in argv is left to M2's classifier (no signal from the sweep).
- [ ] **Step 2: Run, expect FAIL.** `go test ./cmd/internal/launcher/ -run SweepOrphanNvim -count=1`
- [ ] **Step 3: Implement:** replace `cands[tag] → ReapNvim(tag)` with per-row `ClassifyNvimReap` decisions
  (the row's tag comes from `TagFromNvimEmbedArgv`, its liveness from `liveTags`), keeping the sidecar
  branch as the other evidence source.
- [ ] **Step 4: Run, expect PASS.**
- [ ] **Step 5: Commit** `#583: sweep embedded nvim per pid from one snapshot`

### Task 5: Tag the changelog viewer; teach the argv parser its marker

**Files:**
- Modify: `cmd/internal/opener/run.go:211-217`, `nvim/changelog.lua`, `cmd/internal/artifactpath/paths.go:318-338`
- Test: `cmd/internal/opener/run_test.go`, `cmd/internal/artifactpath/paths_test.go:363-390`

- [ ] **Step 1: Failing tests:** `RunChangelog` passes `PAIR_NVIM_PID_FILE=<scope>/nvim-pid-<tag>-changelog`
  (mirroring `run_test.go:120`); `TagFromNvimEmbedArgv` returns the tag for a `changelog-<tag>-<agent>-<session>.log`
  argv and still returns `""` for a foreign path.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** the env entry, the parser marker, and the pidfile block in `nvim/changelog.lua`
  (a copy of `init.lua:24-34`'s VimEnter write — the same three lines, same `pcall`, same silence on failure).
- [ ] **Step 4: Run opener + artifactpath, expect PASS.**
- [ ] **Step 5: Commit** `#583: tag the changelog viewer's embedded nvim`

### Task 6: A clean exit removes its own sidecar

**Files:**
- Modify: `nvim/init.lua`, `nvim/scrollback.lua`, `nvim/review.lua`, `nvim/changelog.lua`
- Test: `tests/` lua harness for the pidfile lifecycle (the existing `nvim --headless` test shape)

- [ ] **Step 1: Failing test:** a headless nvim started with `PAIR_NVIM_PID_FILE` in a temp dir writes the
  sidecar, and after `qa!` the sidecar is gone. (A surviving embed keeps its sidecar — that is the reaper's
  evidence, so it must **not** be removed by anything but the embed's own VimLeave.)
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** a `VimLeave` autocmd in each config removing `vim.env.PAIR_NVIM_PID_FILE`.
- [ ] **Step 4: Run, expect PASS.**
- [ ] **Step 5: Commit** `#583: remove the nvim pid sidecar on clean exit`

### Task 7: Live conformance — Done-when #1

**Files:**
- Create: `cmd/internal/launcher/nvim_reap_live_test.go`

- [ ] **Step 1: Write the flag-gated live test** (`-nvim-reap-live`, skipping unless set, in the
  `peer_live_test.go` shape): spawn a real `nvim --embed --headless -u <pair init.lua> <tmp>/draft-<tag>.md`
  with `PAIR_NVIM_PID_FILE` set, wait for the sidecar, kill the owner (the spawning parent), run the reaper,
  assert the pid is gone and the sidecar removed. Second case: a real `nvim --embed` with a **live** parent
  and a live tag is still running afterwards. Third case: a non-nvim process named in a stale sidecar is
  untouched.
- [ ] **Step 2: Run it, expect FAIL before Task 3's behaviour is in place / PASS after.**
- [ ] **Step 3: Run the flag-gated live test on this machine and record the output in `## Log`.**
- [ ] **Step 4: Commit** `#583: live conformance for the tagged embed reaper`

### Task 8: M1 atlas + storage inventory

- [ ] **Step 1:** update `atlas/storage-retention-io.md`'s nvim-pid rows (read them first: the anchors are
  grep-checked by `storagegc.TestManagedIOInventoryAnchors`) and `atlas/architecture.md:1256`'s sidecar
  sentence to name all four kinds and identity-exact reaping.
- [ ] **Step 2:** `go test ./cmd/internal/storagegc/ -run TestManagedIOInventoryAnchors -count=1` → PASS.
- [ ] **Step 3:** `sdlc milestone-close --issue 583 --milestone M1`, then commit.

## Chunk 2: M2 — untagged orphans are reported and reaped on confirmation

### Task 9: Prove the identity stamp is a wall clock

**Files:**
- Create: `cmd/internal/procutil/identity_age_test.go`

- [ ] **Step 1: Failing test:** spawn `sleep 30` as an owned child, read `Identity(pid)`, parse
  `"<sec>.<usec>"`, assert the elapsed time since that stamp is between 0 and 5 s. This is the assumption
  every age threshold rests on, so it is measured, not read from a header (ARCH-MOCK: the fake models what
  a live check proves).
- [ ] **Step 2: Run, expect FAIL** until `ParseIdentityTime` exists.
- [ ] **Step 3: Implement** `func ParseIdentityTime(identity string) (time.Time, bool)` in `procutil`
  (rejecting a malformed or empty stamp) and use it wherever age is derived.
- [ ] **Step 4: Run, expect PASS.** Commit `#583: derive process age from the kernel identity stamp`

### Task 10: The cwd probe

**Files:**
- Modify: `cmd/internal/procutil/source.go` (`Reader`, `Source`), `resource_darwin.go` (C block), the
  unsupported-build file beside `identity_other.go`
- Test: `cmd/internal/procutil/resource_darwin_test.go`, `source_test.go`

- [ ] **Step 1: Failing live conformance test:** for an owned child started with a known `Dir`, `Cwd(pid)`
  returns that directory; for a dead pid it returns an error, never an empty success.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** `static int resource_cwd(int pid, char *buf, size_t len)` with
  `proc_pidinfo(pid, PROC_PIDVNODEPATHINFO, 0, &vpi, sizeof vpi)` copying `vpi.pvi_cdir.vip_path`, the Go
  wrapper, the `Reader`/`Source` passthrough and the unsupported-build error. Add `Cwd` to every existing
  fake reader (`couchcmd/resources_test.go:20-66`, `procutil/source_test.go`).
- [ ] **Step 4: Run procutil, expect PASS.** Commit `#583: read a process cwd through libproc`

### Task 11: One unattributed rule (DRY with #587)

**Files:**
- Modify: `cmd/internal/couchcore/resource.go:301-341`
- Test: `cmd/internal/couchcore/resource_test.go:90` (`TestResourceClassifyUnproven`) stays green unchanged

- [ ] **Step 1: Extract** `func UnattributedCandidate(p procutil.Process, claimed map[ResourceProcessKey]bool, args []string) bool`
  holding exactly today's predicates (ppid 1, not zombie, identity known, unclaimed, known kind name, and
  for `nvim` an exact `--embed` field). `ClassifyUnattributed` keeps its RSS floor and view construction and
  calls it.
- [ ] **Step 2: Run the existing #587 tests unchanged, expect PASS** — the extraction is behaviour-preserving,
  and a red existing test means the extraction changed the rule.
  `go test ./cmd/internal/couchcore/ -run 'Resource' -count=1`
- [ ] **Step 3: Commit** `#583: extract the shared unattributed-process predicate`

### Task 12: The orphan classification, pure

**Files:**
- Create: `cmd/internal/couchcore/resource_orphan.go`, `resource_orphan_test.go`

- [ ] **Step 1: Failing table test:** the full predicate matrix — ppid ≠ 1; zombie; empty identity; claimed
  by a live slot; name not nvim; argv without exact `--embed`; cwd unknown; cwd outside every root;
  cwd inside a root but a sibling prefix (`/work/tree2` vs root `/work/tree`); age below the threshold;
  age above → classified, carrying pid, identity, age, cwd, the matching root and RSS.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement**

```go
type OrphanNvim struct {
	Key      ResourceProcessKey
	Name     string
	Age      time.Duration
	Cwd      string
	Root     string
	RSSBytes *uint64
	Reason   string
}

func ManagedRoots(slots ResourceSlotInventory) []string
func WithinManagedRoot(cwd string, roots []string) (string, bool)
func ClassifyOrphanNvim(row procutil.Process, args []string, cwd string, cwdKnown bool,
	claimed map[ResourceProcessKey]bool, rss *uint64, roots []string, now time.Time, minAge time.Duration) (OrphanNvim, bool)
```

- [ ] **Step 4: Run, expect PASS.** Commit `#583: classify untagged embedded-nvim orphans`

### Task 13: Policy knob and sampler wiring

**Files:**
- Modify: `cmd/internal/couchcore/resource_policy.go:11-77`, `cmd/internal/couchcore/resource.go` (view),
  `resource_reduce.go:258`, `cmd/internal/couchcmd/resources.go:144-177`, `resources_inventory.go`
- Test: `resource_policy` tests, `couchcmd/resources_test.go`

- [ ] **Step 1: Failing tests:** `COUCH_RESOURCE_ORPHAN_NVIM_AGE=6h` parses; `0`, `-1`, `999d` and an
  unknown key refuse with the existing error shape; the default is 6h. A sampler pass over a fake reader
  with one qualifying row publishes exactly one `OrphanNvim` and charges one `Cwd` call to
  `Cost.NativeCalls`; a pass with a live-parent nvim charges none.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement:** add `OrphanNvimAge time.Duration` to the policy and its env case; in the
  candidate loop, gate `Cwd` behind ppid 1 + name nvim + age ≥ threshold (age comes from the identity stamp
  already in the row, so the gate costs no syscall), then classify with the roots from `s.slots()`.
- [ ] **Step 4: Run couchcore + couchcmd, expect PASS.** Commit `#583: sample untagged nvim orphans within the measured budget`

### Task 14: The health line

**Files:**
- Modify: `cmd/internal/couchcmd/resources_render.go:43-92`, `resources_projection.go:16-68`
- Test: `cmd/internal/couchcmd/resources_cli_test.go`, `resources_projection_test.go`

- [ ] **Step 1: Failing tests:** the text output gains an `Orphan nvim (untagged; reap needs --confirm):`
  section printing `pid=`, `age=`, `cwd=` and the matching root, and `none observed` when empty; `--json`
  carries the rows; the wire projection stays within `MaxFrameBytes` at maximum cardinality.
- [ ] **Step 2: Run, expect FAIL.** **Step 3: Implement.** **Step 4: Run, expect PASS.**
- [ ] **Step 5: Commit** `#583: report orphan embedded nvim in couch --resources`

### Task 15: One hub alert per episode

**Files:**
- Modify: `cmd/internal/couchcmd/resources_alert.go:35-59`, `cmd/internal/couchcore/resource_episode.go`
- Test: `resources_alert_test.go`, `resources_alert_broker_test.go`

- [ ] **Step 1: Failing tests:** an orphan crossing the threshold raises one hub notice naming pid, age,
  cwd and root with the existing body shape and `Inspect with couch --resources.`; a second pass in the same
  episode raises none; a real-broker test proves delivery and dedup by effect ID.
- [ ] **Step 2: Run, expect FAIL.** **Step 3: Implement** on the existing episode machinery (`pending →
  notified`, `resource-%x` effect ID). **Step 4: Run, expect PASS.**
- [ ] **Step 5: Commit** `#583: alert the hub once per orphan nvim episode`
- Note: #574's hub lifecycle notice does not exist yet (its Plan is empty), so this uses the #587 alert
  seam; when #574 lands, the notice moves there. Recorded in `## Log`.

### Task 16: The confirmed reap

**Files:**
- Create: `cmd/internal/couchcmd/reap_orphans.go`
- Modify: `cmd/internal/couchcore/ops.go:154` (registry + `ConfirmRequired`), `operationdispatch.go`,
  `cmd/internal/couchcmd/cli.go:364-401`, `slot_operations.go:124-126`
- Test: `cmd/internal/couchcmd/reap_orphans_test.go`, `slot_operations` tests

- [ ] **Step 1: Failing tests:** without `--confirm` the CLI refuses with `reap-orphans requires --confirm`
  and the server answers `confirmation-required`; with it, each admitted row is re-verified
  (`ProcOps.Identity(pid)` equals the classified identity) and signalled once, and `FakeProcOps.ReapedOnIdentity`
  proves a recycled identity is **not** signalled; a row whose ppid is no longer 1 at reap time is skipped;
  the output names every pid it ended and every pid it skipped with a reason.
- [ ] **Step 2: Run, expect FAIL.**
- [ ] **Step 3: Implement** the operation: re-classify from a fresh sample (never reap from a cached view),
  then signal `SIGTERM`, wait, `SIGKILL` — the `Reaper` cadence — each step identity-fenced.
- [ ] **Step 4: Run couchcmd, expect PASS.** Commit `#583: reap orphan embedded nvim only on confirmation`

### Task 17: Cost re-measurement and docs

- [ ] **Step 1:** re-run #587's sampler cost measurement at maximum cardinality with the cwd probe enabled
  and record mean/max CPU and the percentage of one core at 60 s in `## Log`; it must stay inside the
  approved <0.2%.
- [ ] **Step 2:** update `atlas/couch.md` (the resource monitor section) and `README.md` where the reaper or
  `couch --resources` is described; add the new env knob to the configuration list.
- [ ] **Step 3:** a live check on this machine: `couch --resources` shows the current bare `nvim --embed`
  rows as live-parent (not orphans), and `couch --reap-orphans` without `--confirm` refuses and names them.
  Record the output in `## Log`.
- [ ] **Step 4:** `sdlc local-check run --name suite`, then `sdlc close --issue 583`.

## Non-goals

- No automatic reaping of an untagged process: confirmation is required, always.
- No change to nvim's TUI/embed fork behaviour (upstream), and no attempt to keep an embed alive.
- No #574 lifecycle-notice implementation; the existing alert seam carries the report until #574 lands.
- #587's proven-orphan RSS floor and alert thresholds are untouched; a tagged leak below that floor is
  still only reaped at the next launch sweep.
