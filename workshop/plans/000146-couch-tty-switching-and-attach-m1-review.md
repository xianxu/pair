# Boundary Review — pair#146 (milestone M1)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3cdeb3f0a889ed610608784939e5c04f7f6e08c4^..a0fa977bb53b50c03b9185f3069bbd6bc662b172 |
| command | sdlc milestone-close --issue 146 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-22T18:31:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1's stated purpose — extract the terminal plumbing once and migrate `pair term` onto it so the extraction is validated by an existing suite rather than by new tests — is genuinely delivered, and I verified the single-source claim rather than taking it: `\x1b[r` now exists in exactly one place in the tree (`hostty/control.go:23`), `updateMouseMode` and `termcmd/queries.go` are gone, and neutering `StripQueries` reddens termcmd's own `TestRedrawTabEmitsNoQueries` — so the old suite really is pinning the new code through the new call, which was the whole argument for migrating in M1 rather than deferring it. Four of the six deletion checks the issue Log lists came back red when I re-ran them. What holds this back from SHIP is four Importants, three of them inside the newly-introduced API M2 consumes next: `Screen` raises a **false bell** for any escape sequence longer than `maxPending` split across two reads (nvim's OSC 52 clipboard is the everyday trigger), which is precisely the invariant the type's own doc and the atlas say it exists to protect; `runShell`'s defer ordering now runs `host.Close()` (and its `signal.Stop`) *after* `mux.closeAll()`, re-opening the SIGWINCH-goroutine-vs-deferred-Close race this repo already wrote a lesson about; `NewFakeChild`'s documented `Wait`/`Done` contract is false in a way that makes a downstream test hang rather than fail; and the "Ring's trim now COPIES instead of re-slicing" fix is claimed as one of three bugs the extraction found, but the test named as its pin passes with the fix reverted and the stated bug does not reproduce. None of these breaks `pair term` today, which is why this is not REWORK.

## 1. Strengths

- **The migration is pinned by the suite it was migrated to validate.** I neutered `StripQueries` to `return buf` and `termcmd`'s `TestRedrawTabEmitsNoQueries` (`replay_path_test.go:25`) went red. That is the M1 thesis holding: the extracted policy is still defended from the caller's side, not only by its own moved tests.
- **`Screen` genuinely beats the scanner it replaces, and the improvement is pinned.** `TestScreenSplitReadsReachTheSameState` and `FuzzScreenFeed` assert chunk-invariance, which `updateMouseMode` structurally could not satisfy. Dropping `"1049"` from the alt-screen list (`screen.go:145`) reddens two subtests; moving the BEL check to grep the whole chunk reddens `TestScreenBellIgnoresOSCTerminators/OSC_title_BEL-terminated`.
- **`TestChildResizeIsObservedByTheChild` (`child_test.go:53`) asserts the child *sees* the resize** via `stty size`, not that the ioctl returned nil — the comment says exactly why, and that distinction is the difference between a test and a formality for M2's one-row-shorter pty.
- **`hostty` ships a real, ungated conformance comparison** (`hostty_test.go:118`, `TestOSHostConformsToTheFakeOnSizeAndRawMode`) driving `OSHost` and `FakeHost` through one shared scenario against a live pty. That is the ARCH-MOCK shape done right, and it is the template `ptychild` should copy (see Important 3).
- **`TestRingSnapshotDoesNotAliasTheRing` documents why its first version could not fail** and constructs the capacity so the next `Append` overwrites in place. Confirmed load-bearing: `return r.data` reddens it.
- **The issue Log records what the automated probe *cannot* prove** and diagnoses the "no visible tabs" report as a smoke-instruction fault (the tab strip renders as a zellij pane title) rather than arguing it away — with a differential against `7187b22` as the evidence.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `Screen` raises a false bell for any sequence longer than `maxPending` split across two reads (`cmd/internal/ptychild/screen.go:104`).** When `frame` reports incomplete and `len(buf) > 256`, the code drops the ESC and rescans the remainder as text — so the sequence's payload, *and eventually its terminating BEL*, go through the plain-run scan at `screen.go:90`. Reproduced:

```
split long OSC (\x1b]0; + 300 bytes, then "more\x07plain"):  bell=true
same bytes fed whole:                                        bell=false
```

The realistic trigger is not a long title, it is **OSC 52** — nvim's clipboard write is `\x1b]52;c;<base64>\x07`, routinely kilobytes, so it is always unterminated at a 4096-byte read boundary. Every clipboard yank would fire the status row's one activity signal. This contradicts `TakeBell`'s own doc ("BEL is only counted outside a sequence") and `atlas/architecture.md:462`. It survives the tests because `FuzzScreenFeed` (`screen_test.go:198`) compares `AltScreen`/`Mouse`/`Pending` but **not** the two latches, and `TestScreenSplitReadsReachTheSameState` (`screen_test.go:140`) does compare them but only over three streams all well under 256 bytes. Fix: on the over-bound path, remember that the abandoned run is a sequence payload and suppress BEL until a terminator is seen (or bound `pending` by its own length and keep framing state) — and extend both split-read assertions to `TakeBell`/`TakeRegionLost`, which is what would have caught it. Zero operator impact today because nothing consumes `TakeBell`; I would have rated this Critical if M4's status row were already wired.

**I2 — `runShell` now stops SIGWINCH delivery *after* closing every child's pty (`cmd/internal/termcmd/run.go:220`).** Defers register in the order `host.Close()` → `restore()` → `mux.closeAll()` → `mux.restoreTerminal()`, so LIFO execution is `restoreTerminal, closeAll, restore, host.Close`. `host.Close()` is what calls `signal.Stop`, so during `closeAll()` the watcher goroutine at `run.go:246` is still live: a SIGWINCH there runs `inheritSize → resizeAll → child.Resize → pty.Setsize(c.ptmx, …) → c.ptmx.Fd()` concurrently with `c.ptmx.Close()`. The pre-migration code registered `defer signal.Stop(winch)` last (`7187b22:run.go:249`), so it ran *first* — this is a strict widening. `workshop/lessons.md` prescribes exactly this ordering ("register that cleanup *after* the close defer so it runs *before* it, since defers are LIFO"), from the `scribecmd` bug that could "ioctl a recycled descriptor and resize an unrelated file". Fix: move the `defer func() { _ = host.Close() }()` registration to immediately after `defer mux.closeAll()`.

**I3 — `NewFakeChild`'s documented contract is false, and nothing pins it (`cmd/internal/ptychild/fake.go:20`).** The doc says "Wait returns ExitCode (default 0) immediately; Done is true once Exit is called, **or immediately if the child was never 'running'**". Measured:

```
Done() right after construction = false   (doc implies true)
Wait() BLOCKED for 300ms                  (doc says immediate)
```

`done` is created open at `fake.go:26` and only `Exit` closes it. No test calls `Wait()` or `Done()` on a fake. This is the ARCH-MOCK divergence that matters: the fake and the real `Child` have opposite `Wait` semantics, and a downstream M2/M3 test written from the doc **hangs** instead of failing — the failure mode `workshop/lessons.md` already records ("A fixture that fights the policy it sits on deadlocks rather than fails"). Fix: either close `done` in `NewFakeChild` (matching the doc) or correct the doc to "Wait blocks until Exit", and add the two assertions. A `ptychild` conformance test comparing `NewFakeChild` and a real `Child` over one shared scenario — the shape `hostty_test.go:118` already uses — is the structural fix.

**I4 — the "Ring's trim now COPIES instead of re-slicing" fix is not a fix, and the test named as its pin passes without it (`cmd/internal/ptychild/ring.go:48-52`).** The comment claims re-slicing is "unbounded growth hiding behind a bounded Snapshot", `Allocated()` (`ring.go:67`) is exported solely to pin it, and the issue Log lists "ring trim" among deletion checks confirmed red. I reverted the trim to `r.data = r.data[len(r.data)-r.capacity:]` and `TestRingDoesNotGrowWithoutBound` (`ring_test.go:63`) **stayed green**. Measuring the actual backing array over 2000 appends into a 32-byte ring:

```
reslice variant: final cap=32, max cap=48
copy    variant: final cap=64, max cap=64
```

Re-slicing monotonically shrinks the remaining capacity, so `append` is *guaranteed* to reallocate — the array is bounded, and in fact smaller than the copy version's. The copy is a reasonable cleanup; it is not a bug fix, the comment asserts a defect that does not exist, and `Allocated()` protects nothing. Fix: either drop the claim (comment, issue Log, `Allocated()`) or write the test that actually distinguishes the two.

**I5 — the plan's Core-concepts table says `termcmd.restoreTerminal` is "deleted (folded into `hostty`)"; it still exists** (`workshop/plans/000146-couch-tty-switching-and-attach-plan.md:89` vs `cmd/internal/termcmd/run.go:1061`). Task 1.4a repeats it: "`restoreTerminal` is deleted and its `\x1b[r` becomes `hostty`'s constant" (`:213`). Only the literal moved; the method survives and writes `hostty.ResetRegion`. The *behaviour* the row is about (one constant, one site) is delivered and I verified it, which is why I am not rating this Critical despite the checklist's default — but the table is a claim about the code and it is wrong at one row, so it needs a `## Revisions` entry rather than a silent edit.

## 4. Minor findings

- `waitCode` is byte-identical in `cmd/internal/couchcore/runner.go:90` and `cmd/internal/ptychild/child.go:130`, and `asExitError` now exists in three packages (`couchcmd`, `couchcore`, `ptychild`) — the third one's own comment says "mirroring couchcore's errors.go" (ARCH-DRY, in the diff whose thesis is "don't write the second copy").
- Stale comment references to deleted symbols: `run.go:1046` "see queries.go", `run.go:1053` "appendBuffer re-slices tab.buffer", `replay.go:37` "returns through readPTY".
- `Child.SetSink` (`fake.go:57`) writes `c.sink` unlocked with no `if c.fake == nil` guard, while `pump` reads it from the read goroutine — a data race if it is ever called on a real child. No caller today.
- `newTab` widens the duplicate-first-output window: the ring is now filled inside the pump, before `bufferSnapshotLocked(tab)` at `run.go:707`, so a chunk that lands in that gap is both replayed by `redrawTab` and written live by `copyActiveOutput`. Passing `nil` for a brand-new tab is the honest fix (there is nothing to replay).
- `make test-race` still targets `./cmd/pair-wrap/`, which does not exist, while plan Task 1.6 requires `go test ./cmd/... -race`. `make test-live` landed; the `-race` half of the same rule did not.
- `OSHost.watch()`'s coalescing has no test — only `FakeHost`'s does (`TestFakeHostResizesCoalesce`). The seam whose stated purpose was making the SIGWINCH path testable is untested on the real side.
- The "an `r` final behind the private introducer is DECRSTR, not a margin change" negative is asserted in two comments (`screen.go:121`, `screen_test.go:105`) and covered by no test; removing the `?`-branch's early return leaves the suite green. The issue Log lists it as a deletion check that went red.
- `OSHost.state` (`os.go:20`) is written under the mutex and never read; `h.resized` is never closed, so a `for range host.Resized()` consumer leaks at teardown.
- `splitAny` (`screen.go:166`) allocates `[]byte(seps)` once per byte scanned.
- `probes/termsmoke/main.go` ends on `os.Exit(1)`, which skips the deferred `cmd.Process.Kill()` — a failing probe run leaks the `pair` child.

## 5. Test coverage notes

- **What I could not run:** every pty-spawning test fails in this shell with `ptychild: start sh: operation not permitted` (fork-with-pty is blocked; `pty.Open` works, which is why `hostty` passes). That is 9 of `ptychild`'s tests and `termcmd`'s `TestTerminalMuxNewTabClearsPreviousTabViewport`, and it matches the issue Log's own environment note. I am reporting the Log's whole-tree green and the `probes/termsmoke` 8/8 as **unverified by me**, not as assumed-good.
- **What passed here:** `go build ./...`, `go vet ./cmd/... ./probes/...`, and every non-pty test in `ptychild`, `hostty` and `termcmd`, including under `-race`. Fuzz seed corpora clean.
- **Deletion checks I re-ran:** red as claimed — Snapshot aliasing, `StripQueries` neutered (via termcmd), `?1049`, BEL-grepped-not-framed. Green despite the claim — ring trim copy (I4), private-introducer `r` (Minor).
- **Gaps that could ship the class of bug in this diff:** the split-read invariance property is asserted for two of `Screen`'s four outputs (I1); `NewFakeChild` has no conformance check against a real `Child`, and its documented contract is already wrong (I3); `ptychild.Child`'s fake branch (`Write`/`Resize`/`Signal`/`Close`) is exercised only through `termcmd`'s two `NewFakeChild` literals.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with one flag.** The extraction is the DRY win and it is real, not asserted: I grepped the tree and `\x1b[r` exists only at `hostty/control.go:23`; `updateMouseMode`, `appendBuffer`, `readPTY` and `queries.go` are all gone rather than left as a second copy. Flag: `waitCode` verbatim twice plus a third `asExitError` wrapper (Minor above) — the same "reap a child, map to an exit code, expose liveness as a closed channel" shape now exists in `couchcore` and `ptychild` independently, and it is the natural next extraction if a third pty owner appears.
- **ARCH-PURE — pass.** `Ring`, `Screen` and `StripQueries` are deterministic and their entire test files run with no process, no fs, no clock; `Child` and `OSHost` are the thin injected shell; `termcmd` keeps every policy it had (numbered tabs, rename, pane title, exit-when-empty) and `childSizeLocked`'s comment names where couch's one-row subtraction will differ. Nothing to flag.
- **ARCH-PURPOSE — pass.** M1's purpose was the shared mechanism plus the migration that validates it, and both halves shipped; the shadow-sweep found no surviving hand-maintained copy of what was extracted. Note for later, not a finding: `scribecmd.go:90-177` still hand-rolls the same host half (`MakeRaw`/`Restore`/`SIGWINCH`→`GetsizeFull`) and carries the very race I2 re-introduces — it is the obvious third `hostty` consumer, and adopting it would retire that lesson's instance rather than just documenting it.
- **ARCH-MOCK — flag.** Both doubles correctly live in non-test files so production and test flow share the boundary, and `hostty` ships the live conformance check ARCH-MOCK asks for. `ptychild` does not, and I3 is exactly the drift such a check exists to catch — the fake's `Wait` blocks forever where the real one returns an exit code. Before M2 wires `PtyRunner`, give `ptychild` the `hostty_test.go:118` treatment: one scenario, both implementations, same assertions.
- **For M2 specifically:** `Screen`'s over-bound discard (I1) is on the path that M4's status row reads and M2's `Reserve` re-assertion reads (`TakeRegionLost` has the same latch shape and the same exposure). Fixing the framing state once fixes both.

## 7. Plan revision recommendations

Append a `## Revisions` entry to `workshop/plans/000146-couch-tty-switching-and-attach-plan.md` recording:

1. **`termcmd.restoreTerminal` was not deleted.** The Core-concepts row (`:89`) and Task 1.4a (`:213`) both say deleted/folded; the method survives at `run.go:1061` and writes `hostty.ResetRegion`. Correct the row to `modified` and state that what moved is the constant, not the method (I5).
2. **`ptychild` shipped surface the table does not list:** `Size`, `Options`, `DefaultRingBytes`, `NewFakeChild`, `Child.Feed`/`SetSink`/`Writes`/`Resizes`/`Exit`. `NewFakeChild` in particular is a stateful double M2/M3 will build fixtures on and belongs in the Integration-points table alongside `FakeHost`.
3. **Task 1.6's `-race` requirement has no target.** `make test-race` points at `./cmd/pair-wrap/`; either repoint it at `./cmd/...` or record that the milestone's race run is manual and unrepeatable.
4. **Tasks 1.1–1.6 are all unticked while M1 closes.** Tick what ran and mark what did not (the pty-backed tests cannot run under the command sandbox), so the plan file and the issue `## Log` stop disagreeing — the same reconciliation `#145` needed for its Task 17.
5. **Record that Decision 6's "Ring trim" claim did not survive verification** (I4), so the next reader does not re-derive the nonexistent unbounded-growth bug from the comment.

```findings
findings:
  - id: new
    severity: Important
    family: chunking-invariance
    title: |
      Screen raises a false bell for any sequence longer than maxPending split across two reads
    detail: |
      screen.go:104 drops the ESC and rescans the payload as text when an incomplete
      sequence exceeds 256 bytes, so the sequence's terminating BEL is later counted by
      the plain-run scan at screen.go:90. Reproduced: a 300-byte OSC split across two
      Feed calls yields bell=true where the same bytes fed whole yield bell=false. The
      everyday trigger is OSC 52 clipboard writes, which are kilobytes and always cross
      a 4096-byte pty read boundary. Contradicts TakeBell's own doc and
      atlas/architecture.md:462. FuzzScreenFeed asserts chunk-invariance for AltScreen,
      Mouse and Pending but not for the two latches, which is why it cannot find this.
  - id: new
    severity: Important
    family: signal-goroutine-outlives-close
    title: |
      runShell now stops SIGWINCH delivery after closing every child pty, re-opening the scribecmd race
    detail: |
      termcmd/run.go:220 registers the host.Close() defer first, so LIFO runs it LAST --
      after mux.closeAll(). The watcher goroutine at run.go:246 is therefore still live
      during closeAll, and a SIGWINCH there runs resizeAll -> child.Resize ->
      pty.Setsize -> ptmx.Fd() concurrently with ptmx.Close(). The pre-migration code
      registered defer signal.Stop(winch) last so it ran first. workshop/lessons.md
      prescribes exactly this ordering from the scribecmd bug. Fix: register the
      host.Close() defer immediately after defer mux.closeAll().
  - id: new
    severity: Important
    family: fake-diverges-from-production
    title: |
      NewFakeChild's documented Wait/Done contract is false and no test pins either
    detail: |
      fake.go:20 documents "Wait returns ExitCode (default 0) immediately; Done is true
      once Exit is called, or immediately if the child was never running". Measured:
      Done() is false at construction and Wait() blocks indefinitely, because done is
      created open at fake.go:26 and only Exit closes it. The fake and the real Child
      have opposite Wait semantics, so an M2/M3 test written from the doc hangs rather
      than fails. No conformance check exists between the fake and a real Child, which is
      the ARCH-MOCK gap that would have caught it -- hostty_test.go:118 is the template.
  - id: new
    severity: Important
    family: fix-not-pinned-by-failing-test
    title: |
      the Ring copy-vs-reslice fix is claimed as a bug fix, but reverting it leaves the named test green
    detail: |
      ring.go:48-52 asserts re-slicing is "unbounded growth hiding behind a bounded
      Snapshot", Allocated() is exported solely to pin it, and the issue Log lists "ring
      trim" as a deletion check confirmed red. Reverting to the re-slice leaves
      TestRingDoesNotGrowWithoutBound green. Measured over 2000 appends into a 32-byte
      ring: reslice peaks at cap=48, copy sits at cap=64 -- re-slicing monotonically
      shrinks remaining capacity so append is guaranteed to reallocate. The copy is a
      reasonable cleanup; the defect it claims to fix does not exist.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      the plan's Core-concepts table says termcmd.restoreTerminal is deleted; it still exists
    detail: |
      Plan row :89 and Task 1.4a :213 both state restoreTerminal is deleted and folded
      into hostty. The method survives at termcmd/run.go:1061 and writes
      hostty.ResetRegion. The behaviour the row is about -- one escape constant, one site
      -- is delivered and verified, so this is a table-accuracy defect rather than missing
      work, but it needs a "## Revisions" entry rather than a silent edit.
  - id: new
    severity: Minor
    family: needless-indirection
    title: |
      waitCode is duplicated verbatim across couchcore and ptychild, and asExitError now exists in three packages
    detail: |
      couchcore/runner.go:90 and ptychild/child.go:130 are byte-identical; ptychild's
      errors.go says it mirrors couchcore's. ARCH-DRY, in the diff whose stated thesis is
      that the second copy is the thing to avoid.
  - id: new
    severity: Minor
    family: stale-comment-reference
    title: |
      comments still cite queries.go, appendBuffer, tab.buffer and readPTY, all deleted by this diff
    detail: |
      termcmd/run.go:1046 ("see queries.go"), run.go:1053 ("appendBuffer re-slices
      tab.buffer from the output goroutine"), ptychild/replay.go:37 ("returns through
      readPTY").
  - id: new
    severity: Minor
    family: unsynchronised-shared-state
    title: |
      Child.SetSink writes c.sink unlocked with no fake-only guard while the pump reads it
    detail: |
      fake.go:57. Documented as being for fakes, but it is a method on *Child with no
      `if c.fake == nil` guard, so calling it on a real child races the read pump. No
      caller today.
  - id: new
    severity: Minor
    family: replay-duplicates-live-output
    title: |
      newTab widens the window where a chunk is both replayed and written live
    detail: |
      The ring is now filled inside the pump, before bufferSnapshotLocked(tab) at
      run.go:707, so a chunk landing in that gap is replayed by redrawTab and then
      written again by copyActiveOutput. A brand-new tab has nothing to replay; passing
      nil is the honest fix. The same shape recurs in M2 when the console attaches to a
      freshly started child.
  - id: new
    severity: Minor
    family: stale-build-target
    title: |
      make test-race still targets ./cmd/pair-wrap/, so Task 1.6's -race requirement has no runnable target
    detail: |
      make test-live landed this window; the test-race half of the same rule did not, and
      the target fails outright with "directory not found". Plan Task 1.6 requires
      go test ./cmd/... -race over the whole tree.
  - id: new
    severity: Minor
    family: seam-untested-on-the-real-side
    title: |
      OSHost's SIGWINCH coalescing has no test; only FakeHost's does
    detail: |
      TestFakeHostResizesCoalesce drives the fake. OSHost.watch()'s non-blocking send --
      the actual coalescing production depends on -- has no coverage, in the package whose
      stated purpose was making the SIGWINCH path testable.
  - id: new
    severity: Minor
    family: uncovered-negative-assertion
    title: |
      the "an r final behind the private introducer is not DECSTBM" negative is asserted twice in comments and covered by no test
    detail: |
      screen.go:121 and screen_test.go:105 both state the rule; TestScreenRegionLost has
      no \x1b[?<n>r case, and removing the ? branch's early return leaves the suite green.
      The issue Log lists this among deletion checks confirmed red.
  - id: new
    severity: Minor
    family: dead-field-and-leaked-consumer
    title: |
      OSHost.state is written and never read, and h.resized is never closed so a range consumer leaks
    detail: |
      hostty/os.go:20 stores the term.State under the mutex; only the closure's captured
      copy is used by Restore. Close() stops and closes h.sigs but leaves h.resized open,
      so termcmd's `for range host.Resized()` goroutine never returns.
  - id: new
    severity: Minor
    family: probe-hygiene
    title: |
      probes/termsmoke exits via os.Exit(1) on failure, skipping the deferred kill of the pair child
    detail: |
      probes/termsmoke/main.go. A failing probe run leaves the spawned pair process alive.
      Also splitAny (screen.go:166) allocates []byte(seps) once per scanned byte.
```
