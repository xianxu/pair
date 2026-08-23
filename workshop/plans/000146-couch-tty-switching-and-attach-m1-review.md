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

---

## Re-review — 2026-08-22T18:54:31-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 146 — couch: tty switching and attach |
| repo | pair |
| issue file | workshop/issues/000146-couch-tty-switching-and-attach.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 3cdeb3f0a889ed610608784939e5c04f7f6e08c4^..3d8a2c0afac1c0de20e06a4a918544d96e0557d3 |
| command | sdlc milestone-close --issue 146 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-08-22T18:54:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've verified the claims independently — reverting fixes in a scratch worktree, re-measuring BR-1, and proving the hostty skip claim by construction. Cleanup done; the main tree is untouched.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1's stated purpose — extract the terminal plumbing once and migrate `pair term` onto it so the extraction is validated by an existing suite — is delivered, and I re-checked the load-bearing claims rather than reading them: `\x1b[r` exists exactly once as a value in the tree, three of the round's fixes go red when I revert them (BR-12's DECRSTR guard, BR-3's fake lifecycle, BR-13's `close(h.resized)`), and BR-4's false "bug found" claim is retracted in the code comment, in `Allocated()`'s doc, and in the issue Log with the measurement that falsified it. What holds this back from SHIP is one carried finding and two new ones, none Critical and none breaking `pair term` today. **BR-1 is not fixed, only moved:** the fix raised `maxPending` from 256 to 64 KiB and cures the everyday OSC 52 case (I confirmed a 100 KB clipboard write split into two reads now derives `bell=false`), but chunk-invariance is *not* restored — an unterminated prefix above 64 KiB still diverges, and the comment at `screen.go:104` asserts the opposite. Alongside it, the pins for BR-2/BR-11/BR-13 do not defend those fixes in this repo's own documented agent shell: `hostty` reports a clean `ok` while silently skipping three tests, two of which need no pty at all; and BR-2's defer ordering — a race this repo already wrote a lesson about — has no test anywhere. Finally, `probes/` arrived as a new top-level directory with no make target and no atlas entry, in the milestone whose docs gate covers exactly that.

## 1. Strengths

- **The retraction is the strongest thing in this diff.** BR-4 claimed a bug that did not exist; the fix corrects it in three places at once (`ring.go:48-52`, `Allocated()`'s doc at `ring.go:69-74`, and the issue Log with a strikethrough and the measured `cap=48 vs 64` that falsified it). A boundary that says "this claim of mine was false" in the artifact, the code, and the API doc is exactly what this gate exists to produce.
- **Three fixes verified load-bearing by revert, not by commit message.** Removing the `final != 'h' && final != 'l'` guard reddens `TestScreenRegionLost/DECRSTR_private_r`; `close(c.done)` at fake construction (the old doc's claim) panics `TestFakeChildExitReportsItsCode`; removing `defer close(h.resized)` from `OSHost.watch` reddens the consumer-release pin (via an ungated copy — see I1).
- **The single-source claim survives grep.** `\x1b[r` exists once as a value (`hostty/control.go:23`); the two other hits are comments. Same for `\x1b[1;1H\x1b[J`. `updateMouseMode`, `appendBuffer`, `readPTY` and `termcmd/queries.go` are gone rather than left as a second copy.
- **BR-2's ordering fix is right and says why.** `defer host.Close()` now registers after `defer mux.closeAll()`, so LIFO runs it first, and the comment names the scribecmd lesson it came from. The companion leak is closed properly: `resized` is closed by the watcher, its *only* sender, after its source is closed — one writer, no send-on-closed race.
- **BR-1's fix does cure the case it was raised for.** Measured: a 100 KB `\x1b]52;c;…\x07` fed as two reads split at 40000 bytes yields `bell=false`, matching the whole-feed result, because `frame` is consulted *before* the bound. That is the everyday trigger, and it is gone.

## 2. Critical findings

None.

## 3. Important findings

**I0 (carried, BR-1 → `not-addressed`) — the false bell is threshold-shifted, not fixed, and the comment claims otherwise.** `screen.go:104-112` says discarding "keeps Feed chunk-invariant — whole and split input abandon the same bytes and derive the same state." It does not. On the whole-feed path an unterminated run past the bound is dropped *along with the entire rest of the buffer* (`return`); on the split path only the first `maxPending+1` bytes are dropped, and everything after arrives in later `Feed` calls and is rescanned as plain text. Measured at HEAD, whole vs 4096-byte chunks:

```
"\x1b["  + 70000×';'  + "\x07"      whole bell=false   split bell=true
"\x1b]52;c;" + 100000×'A' + "\x07"  whole bell=false   split bell=true
"\x1b["  + 70000×';'  + "\x1b[?1049h"
                                    whole alt=false region=false
                                    split alt=true  region=true
```

The third case matters more than the first two: `AltScreen` and `TakeRegionLost` diverge too, and those are what M2's `Reserve` re-assertion reads. It also disposes of the Log's stated class fix — "the class fix is the fuzzer, it now covers both latches." `FuzzScreenFeed` already asserted `AltScreen`/`Mouse` invariance before this round and still cannot find this, because the threshold is 64 KiB and go-fuzz corpora are kilobytes; adding two more comparisons at the same input scale does not change that. The rule the fix needs is that **abandoning a run must be stateful**: once past the bound, keep skipping under the same framing rule until the sequence's terminator, so whole and split abandon identical bytes. `atlas/architecture.md`'s "BEL is likewise counted only outside a sequence" carries the same overstatement. Realistic trigger: an OSC 52 carrying >48 KiB of clipboard (base64 → >64 KiB) across pty reads. Latent today — `TakeBell`/`TakeRegionLost` have zero production callers until M4 — which is why this stays Important.

**I1 (new) — the pins for BR-2, BR-11 and BR-13 do not run in the environment this issue documents as its agent shell.**

> **This is the 2nd finding in family `fix-not-pinned-by-failing-test`.** Do NOT fix these three instances. The rule that covers them: **a fix is defended only by a test the default suite actually executes and that goes red without it. A `t.Skipf` on an unavailable dependency and a comment are both zero.**

Measured at HEAD in this shell (`pty.Open` → `operation not permitted`, the exact condition the issue Log records):

- `go test ./cmd/internal/hostty/` reports **`ok`** with three silent skips — `TestOSHostConformsToTheFakeOnSizeAndRawMode`, `TestCloseReleasesResizedConsumers` (BR-13's pin), `TestOSHostCoalescesRealSIGWINCH` (BR-11's pin). 3 of the package's 9 tests.
- Two of those three **do not need a pty at all.** I rewrote them against a plain `os.Create(t.TempDir()+"/notatty")` — `NewOSHost` only needs `in != nil` to start the watcher, and neither test measures a terminal. Both pass. Then, with `defer close(h.resized)` deleted, the ungated copy fails in 2s while the shipped `TestCloseReleasesResizedConsumers` skips and the package still reports `ok`.
- `ptychild` handles the identical constraint the opposite way — its pty tests `t.Fatalf`, so the package fails loudly. Two handlings of one constraint in one milestone. The Log's mitigation ("a sandboxed green on those packages is not evidence") is true for the loud one and false for the quiet one.
- **BR-2's defer ordering has no test at all** — `grep runShell cmd/internal/termcmd/*_test.go` is empty, so moving the defer back next to `NewOSHost` leaves the whole suite green. It is defended by a comment.
- Neither `test-race` nor `test-live` is in `make test`'s prerequisite list (`Makefile.local:81`), so both remain opt-in.

The sweep this implies: for every fix landed this round, revert it and require red *from the suite `make test` runs*. Where a test genuinely needs a real pty, gate it the way `ptychild` does (fail, don't skip) or make the pty an explicit prerequisite rather than a silent one.

**I2 (new) — `probes/` is new top-level surface with no target and no map entry.**

> **This is the 2nd finding in family `probe-hygiene`.** Do NOT just add an atlas line. The rule: **a committed probe is a first-class artifact — it has a make target, an entry in `atlas/`, and it cleans up after itself. Otherwise it is a scratch script with a commit hash.**

`probes/` was created in this window (`fd550be`, `git log --diff-filter=A -- probes/`). `grep -rn "termsmoke\|probes" Makefile Makefile.local tests/ .github/` returns nothing; `grep -rn "probes/" atlas/ README.md` returns nothing; `atlas/index.md` links every other file but not this directory. `probes/termsmoke/main.go`'s own header states the reason it is committed — "a probe whose output gets quoted in an issue Log has to be re-runnable against a later commit" — and its 8/8 result *is* quoted in the issue Log as M1 evidence. AGENTS.md §8 asks for atlas updates on new surface and file-tree locations at each milestone close. BR-14 (cleanup on the `os.Exit` path) was the first instance of this rule and is fixed; the enumeration is cheap now, while `probes/` has one member, and the plan schedules three more smokes (Tasks 2.7, 3.5, 4.6).

## 4. Minor findings

- **New, ledgered:** the plan's `Screen` description (`…-plan.md:99`) still declares `MarginsDirty bool` / `Bell bool`; the code has `regionLost`/`TakeRegionLost()` and `bell`/`TakeBell()`. Second instance of `plan-table-drift` on this issue, third counting `#145`'s BR-41 — see the findings block for the rule.
- **Not ledgered (cosmetic, noted for the next touch):** `termcmd/rename_input.go:5` wedges the `ansi` import inside the stdlib group (between `bytes` and `unicode/utf8`); gofmt tolerates it, goimports would not. `ptychild/replay.go:37` lost its wrapping when `appendBuffer` was rewritten to `Ring` — one ~150-char comment line in a file where every other line wraps at 80.
- `Child.Replay()` (`child.go:175`) documents itself as "what a repaint should write… Prefer Replay for repainting a screen", and the one repaint site in the tree does not use it: `redrawTab` (`termcmd/run.go:1053-1056`) hand-composes `HomeAndClear` + `StripQueries(Snapshot())`, which is `Replay()` plus the clear. Zero production callers for the helper; ARCH-DRY inside the diff whose thesis is one mechanism.
- Prior-round Minors still open and confirmed reproducing: `waitCode` byte-identical at `couchcore/runner.go:90` and `ptychild/child.go:130` (BR-6); three stale comment references (BR-7), plus a fourth now — `Makefile.local:73-77` still explains `test-race` as "scoped to packages where the suite has actual concurrent code… pair-wrap has translateStdin's goroutine" after the target was repointed at `./cmd/...`; `SetSink` unlocked with no `c.fake == nil` guard (BR-8); `newTab`'s snapshot-after-pump-start replay window (BR-9).

## 5. Test coverage notes

- **What I ran:** `go build ./...`, `go vet` over `ptychild`/`hostty`/`termcmd`/`probes` (clean), and the full `go test ./cmd/... -count=1`. Every failure is my shell's process/pty restriction (`operation not permitted` on `pty.Start`, `mktemp`, test-binary re-exec) in `ptychild`, `termcmd`, `keyscmd`, `wrapcmd` — the condition the issue Log records. Nothing failed for a code reason.
- **What I could not verify and am not assuming:** the Log's whole-tree green, `make test-race`, `make test-term-pane-shortcuts`, `probes/termsmoke` 8/8, and the 3.7M-exec fuzz run. `TestFakeChildConformsToRealChildLifecycle` — BR-3's class fix — is among the tests I could not execute; I verified the fake half separately instead.
- **Verified red on revert:** BR-12's introducer guard, BR-3's fake lifecycle, BR-13's `close(h.resized)` (via an ungated copy). **Verified green despite the fix being reverted:** BR-2's defer ordering (no test exists) — see I1.
- **Gaps that could ship the class of bug in this diff:** the split-read invariance property has no case above `maxPending` (I0); `hostty`'s real-side assertions vanish without a pty (I1); `Child`'s fake-mode branch (`Write`/`Resize`/`Signal`/`Close`/`Exit`) is exercised only through `termcmd`'s two `NewFakeChild` literals and the two fake-only tests.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with one flag.** The extraction is real and I checked it rather than reading the claim: `\x1b[r` and `\x1b[1;1H\x1b[J` each exist once as a value, and the four extracted symbols are deleted at their old sites rather than left behind. Flag: `waitCode` verbatim in two packages plus `asExitError` in two (BR-6, open), and `redrawTab` reimplementing `Replay()` (§4). Both are "the second copy" the milestone's own thesis is about.
- **ARCH-PURE — pass.** `Ring`, `Screen` and `StripQueries` are deterministic and their entire test files run with no process, no fs, no clock — confirmed, they are the packages that stayed green in a shell that cannot fork. `Child` and `OSHost` are the thin injected shell; `termcmd` keeps every policy it had, and `childSizeLocked`'s comment names where couch's one-row subtraction will diverge. Nothing to flag.
- **ARCH-PURPOSE — pass.** The shadow-sweep on M1's purpose: one consumer was named (`termcmd`) and it is migrated on both halves, with the extraction validated through the existing suite (neutering `StripQueries` still reddens termcmd's own `TestRedrawTabEmitsNoQueries`, so the caller-side pin survived the move). No surviving hand-maintained copy of what was extracted. Out of scope but worth naming for later: `scribecmd.go:92,143` and `wrapcmd/wrap.go:2392,2426` still hand-roll `MakeRaw`/`signal.Notify(SIGWINCH)`/`GetsizeFull`. `scribecmd` is the obvious third `hostty` consumer, and adopting it would retire the very lesson BR-2 re-earned rather than leaving it as prose.
- **ARCH-MOCK — flag.** Both doubles correctly live in non-test files so production and test flow share the boundary, and both packages ship a real-vs-fake conformance check driven through one shared scenario — `TestOSHostConformsToTheFakeOnSizeAndRawMode` and, new this round, `TestFakeChildConformsToRealChildLifecycle`. That closes BR-3's structural gap. The flag is reach, not shape: the `hostty` conformance check is one of the three that self-skips (I1), so the comparison that justifies the seam design runs only where a pty happens to exist. Before M2 wires `PtyRunner`, `Handle`'s contract changes — extend `FakeRunner`'s state model in the same commit, and give the conformance pair a prerequisite rather than a skip.
- **For M2 specifically:** `host.Close()` returns as soon as it stops the signal source; it does not join the consumer goroutine. So a SIGWINCH already in flight can still be running `inheritSize → resizeAll → Child.Resize` while `closeAll()` closes the ptys. That window is unchanged from pre-migration (`signal.Stop` had the same property), so it is not a regression and I am not raising it — but `couchtty.Console` will own the same shape with a reserved row on top of it, and a real join (a `sync.WaitGroup` the watcher and the consumer both hold) is the cheap fix to make once, there, rather than twice.

## 7. Plan revision recommendations

The `## Revisions` section landed and correctly records BR-5 and BR-4's retraction. Three entries still owed, two of them carried from the prior round unactioned:

1. **`Screen`'s field names.** The Core-concepts prose (`:99`) declares `MarginsDirty bool` and `Bell bool`; the code has `regionLost`/`TakeRegionLost()` and `bell`/`TakeBell()`. Per the rule `#145`'s BR-41 already stated: either stop restating type shapes in this table (as `atlas/couch.md` now correctly does for the operation set) or append the Revisions entry in the commit that changes the shape.
2. **`ptychild` shipped surface the tables do not list** *(carried from round 1, item 2)*: `Size`, `Options`, `DefaultRingBytes`, `NewFakeChild`, and `Child.Feed`/`SetSink`/`Writes`/`Resizes`/`Exit`. `NewFakeChild` in particular is the stateful double M2/M3 will build every switcher fixture on and belongs in the Integration-points table next to `FakeHost`.
3. **Tasks 1.1–1.6 are all still `- [ ]` while M1 closes** *(carried from round 1, item 4)*. Tick what ran and mark what did not — specifically that the pty-backed tests cannot run under the command sandbox, which the issue `## Log` already records and the plan does not. Same reconciliation `#145` needed for its Task 17.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Threshold moved 256 -> 64 KiB; the everyday OSC 52 case is genuinely cured, but chunk-invariance is not restored above the new bound. See I0 for the measurement.
  - id: BR-2
    disposition: addressed
    note: |
      defer host.Close() now registers after defer mux.closeAll(), so LIFO runs it first; resized is closed by its only sender. No test pins the ordering -- that is I1, not a re-raise.
  - id: BR-3
    disposition: addressed
    note: |
      Doc corrected to match the code. Verified by revert - closing done at construction panics TestFakeChildExitReportsItsCode; the real-vs-fake conformance test exists but could not run in my shell.
  - id: BR-4
    disposition: addressed
    note: |
      Claim retracted in ring.go's comment, Allocated()'s doc, and the issue Log, each carrying the cap=48-vs-64 measurement that falsified it.
  - id: BR-5
    disposition: addressed
    note: |
      Core-concepts row now reads "modified (now writes hostty.ResetRegion; the method stays, the constant moved)", recorded via a ## Revisions entry.
  - id: BR-6
    disposition: not-addressed
    note: |
      waitCode still byte-identical at couchcore/runner.go:90 and ptychild/child.go:130; asExitError still wrapped in two packages.
  - id: BR-7
    disposition: not-addressed
    note: |
      All three stale references remain (run.go:1052, run.go:1059, replay.go:37), and Makefile.local:73-77 is now a fourth -- it still explains test-race as scoped to pair-wrap after the target was repointed at ./cmd/...
  - id: BR-8
    disposition: not-addressed
    note: |
      fake.go:65 still writes c.sink unlocked with no `if c.fake == nil` guard.
  - id: BR-9
    disposition: not-addressed
    note: |
      newTab still calls bufferSnapshotLocked(tab) after ptychild.Start has launched the pump; a chunk landing in the gap is replayed and written live.
  - id: BR-10
    disposition: addressed
    note: |
      test-race is now `go test -count=1 -race ./cmd/...` and runnable. Residual, folded into I1 rather than re-raised - neither test-race nor test-live is in make test's prerequisites.
  - id: BR-11
    disposition: addressed
    note: |
      TestOSHostCoalescesRealSIGWINCH drives the real watcher against real signals. It skips without pty.Open, which it does not need -- that is I1, not a re-raise.
  - id: BR-12
    disposition: addressed
    note: |
      Verified by revert - removing the `final != 'h' && final != 'l'` guard reddens TestScreenRegionLost/DECRSTR_private_r.
  - id: BR-13
    disposition: addressed
    note: |
      OSHost.state deleted; resized closed by the watcher. Verified load-bearing - deleting `defer close(h.resized)` reddens an ungated copy of the pin in 2s.
  - id: BR-14
    disposition: addressed
    note: |
      cleanup() runs before os.Exit(1) at main.go:119-120; splitAny replaced by splitParams, which allocates no separator slice per byte.
findings:
  - id: new
    severity: Important
    family: fix-not-pinned-by-failing-test
    title: |
      the pins for three of this round's fixes skip themselves in the environment the issue documents as its agent shell
    detail: |
      2nd in this family. Do NOT fix the three instances -- the rule is that a fix is defended
      only by a test the default suite EXECUTES and that goes red without it; a t.Skipf on an
      unavailable dependency and a comment are both zero. Measured at HEAD: `go test
      ./cmd/internal/hostty/` reports ok with three silent skips (3 of 9 tests), including
      BR-11's and BR-13's pins. Two of the three need no pty at all -- rewritten against a
      plain temp file both pass, and with `defer close(h.resized)` deleted the ungated copy
      fails in 2s while the shipped test still skips and the package still reports ok.
      ptychild handles the identical constraint by failing loudly, so one milestone ships two
      handlings of one condition. BR-2's defer ordering has no test anywhere (grep runShell
      over termcmd's tests is empty), so reverting it leaves the suite green. Neither
      test-race nor test-live is in make test's prerequisites (Makefile.local:81).
  - id: new
    severity: Important
    family: probe-hygiene
    title: |
      probes/ is new top-level surface with no make target and no atlas entry
    detail: |
      2nd in this family. Do NOT just add an atlas line -- the rule is that a committed probe
      is a first-class artifact: a make target, an entry in atlas/, and self-cleanup.
      probes/ was created in this window (git log --diff-filter=A -- probes/ -> fd550be).
      grep over Makefile, Makefile.local, tests/ and .github/ for "termsmoke|probes" returns
      nothing; grep over atlas/ and README.md for "probes/" returns nothing; atlas/index.md
      links every other file. The probe's own header states why it is committed -- output
      quoted in an issue Log must be re-runnable against a later commit -- and its 8/8 result
      IS quoted as M1 evidence. AGENTS.md section 8 asks for atlas updates on new surface and
      file-tree locations at each milestone close. Cheap to sweep now, with one member; the
      plan schedules three more smokes at Tasks 2.7, 3.5 and 4.6.
  - id: new
    severity: Minor
    family: plan-table-drift
    title: |
      the plan's Screen description declares MarginsDirty and Bell; the code has regionLost and bell behind Take* readers
    detail: |
      2nd in this family on this issue, 3rd counting pair#145's BR-41. Do NOT just rename the
      two words -- the rule is that the plan's Core-concepts table is a hand-maintained
      restatement of code shapes, so either stop restating them (as atlas/couch.md now
      correctly does for the operation set) or append the ## Revisions entry in the same
      commit that changes the shape. Plan line 99 vs screen.go:39-42, 52-68.
```
