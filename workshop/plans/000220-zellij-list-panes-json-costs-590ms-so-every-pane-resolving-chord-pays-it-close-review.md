# Boundary Review — pair#220 (whole-issue close)

| field | value |
|-------|-------|
| issue | 220 — zellij list-panes --json costs 590ms, so every pane-resolving chord pays it |
| repo | pair |
| issue file | workshop/issues/000220-zellij-list-panes-json-costs-590ms-so-every-pane-resolving-chord-pays-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | 21cc7f7791587351601a8274043f62493d0edd98..ccc70fa16786ac19ead175f912c46475ac1727f2 |
| command | sdlc close --issue 220 |
| reviewer | claude |
| timestamp | 2026-09-09T15:07:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core of #220 is well built and the win is real: `resolveFromSidecars` (`layoutcmd.go:58`) is a genuinely pure decision function with a generated agreement oracle, the caller enumeration in the Plan matches the five `ListPanesJSON` call sites in the tree exactly, and the layoutcmd fast path is pinned by a test that counts `ListPanesJSON` calls so a regression to 590ms fails the suite. What blocks SHIP is the other two thirds of the change: the two `termcmd` fast paths (`run.go:165-169`, `run.go:604-606`) execute in **zero** tests — I confirmed this with a coverage profile, both blocks report count 0 — and the reason the first one is untestable is that it bypasses the `Runtime` seam it was handed, calling `draftroute.CachedDraftPaneIDFromEnv()` directly when `rt.CachedDraftPaneID()` exists and `OSRuntime` implements it as exactly that call. Same class: `procutil.Alive`'s two declared behavior changes (pid≤0 now false, EPERM now alive — I measured both against the old `kill -0`) are unpinned; `TestAlive` passes verbatim against either implementation. All findings are cheap; none is a correctness bug on the production path.

**Verification run:** `go build ./...` clean, `gofmt -l` clean, `go vet` clean. `go test ./cmd/internal/{layoutcmd,procutil,draftroute}` pass; `tests/term-pane-shortcuts-test.sh` passes end to end. Three `termcmd` failures (`TestTerminalMuxNewTab*`, `TestEveryStripModelMutationRepaintsTheRow`) are `ptychild: start /bin/sh: operation not permitted` — the documented pty-environment class, unrelated to this diff.

## 1. Strengths

- `resolveFromSidecars` (`layoutcmd.go:58-72`) is the right shape: no `Runtime`, four explicit branches, each with the reason it declines written next to it. ARCH-PURE pass — its tests run with no IO at all.
- `TestSidecarFastPathFallsBackWhenItCannotAnswer` (`layoutcmd_test.go:392`) asserts `listCalls` per case, so it pins both directions — the fast path answering *and* the fall-back still consulting the pane list. That is the assertion that stops a future "optimization" from silently eating the zellij-focus tie-break.
- The `## Log`'s honesty about the generated test finding two divergence classes, and scoping the property rather than weakening it, is exactly right. Asserting that two guessing paths guess alike would have been a coincidence test.
- `currentRightTerminalPane`'s fast path (`run.go:601`) gates on registry membership rather than on `ZELLIJ_PANE_ID` alone — the premise is checked against evidence. (Its sibling does not; see Important 3.)
- The ARCH-PURPOSE shadow-sweep holds: I enumerated `ListPanesJSON()` call sites independently and got the same five the Plan's table lists, with `RunToggleFocused` (`layoutcmd.go:239`) correctly kept on the pane list because it reads `Columns` for the resize planner and no sidecar carries geometry.

## 2. Critical findings

None.

## 3. Important findings

**I1 — both `termcmd` fast paths ship with zero test coverage** (`cmd/internal/termcmd/run.go:165`, `:604`).
Coverage profile over the package: block `165.4,169.1` count 0 and blocks `604.5,606.1` count 0. No test enters either. `TestSplitTerminalDownIsNativeTiledSplit` (`run_test.go:257`) sets `currentPaneID:"4"` but leaves `terminalPaneIDs` nil, so it falls through; nothing sets both. The only new `termcmd` test asserts a constant classifies correctly, not that the path taking it runs. Fix: `&fakeRuntime{currentPaneID:"4", terminalPaneIDs:[]string{"4"}, failList:true}` → `splitTerminalDown` must succeed with `listCalls == 0` (`failList` already makes a list call a hard failure, which is the strongest form of this assertion). Same for `focusedWorkbenchPanes` once I2 lands.

**I2 — the `focusedWorkbenchPanes` fast path bypasses the injected seam (ARCH-MOCK, ARCH-PURE)** (`run.go:164`).
It calls the package-level `draftroute.CachedDraftPaneIDFromEnv()`, reading `os.Getenv` + the real filesystem, while `termcmd.Runtime` already declares `CachedDraftPaneID() (string, bool)` (`run.go:29`) and `OSRuntime` implements it as literally `return draftroute.CachedDraftPaneIDFromEnv()` (`run.go:1669`). Production flow and test flow no longer share the boundary, which is what makes I1 unfixable as written, and it makes the branch taken depend on ambient env: inside a pair session `PAIR_DATA_DIR`/`PAIR_TAG`/`ZELLIJ_SESSION_NAME` are all set, so `go test` and `tests/term-pane-shortcuts-test.sh` can take a different path than they do on a clean shell. Fix: `if draftID, ok := rt.CachedDraftPaneID(); ok`.

**I3 — the fast path infers "I am a right terminal" from a bare `ZELLIJ_PANE_ID`, unchecked** (`run.go:163`).
`focusedWorkbenchPanes` synthesises `TerminalCommand: rightTerminalClassifier` for whatever pane id the env carries. The premise ("bytes on `pair term`'s stdin can only come from its own pane") holds for `pumpStdin`, but `handleChord` is also reachable from `--test-shortcut` (`run.go:74`), which `tests/term-pane-shortcuts-test.sh` deliberately drives from the *draft's* perspective (`:100-118` for the #216 tab chords, `:162-168` for the review-pane rows) inside a real zellij pane that exports `ZELLIJ_PANE_ID`. The Plan's justification — "the `--test-shortcut` path, which has no live pane" — is false there; what actually saves those rows today is the unrelated accident that the suite's temp `PAIR_DATA_DIR` holds no valid draft-pane cache. Fix: gate on `currentID ∈ rt.TerminalPaneIDs()`, the same predicate `currentRightTerminalPane` uses 440 lines down (ARCH-DRY). Registration happens at `run.go:269`, before `pumpStdin` at `:310`, so the live path still hits the fast branch; anything else correctly falls back.

**I4 — `procutil.Alive`'s claimed fixes are pinned by no test** (`procutil.go:35`, `procutil_test.go:11`).
The Spec commits to two behavior changes and PQ-4 demanded the `positivePID` guard, but `TestAlive` asserts only empty / self / bogus-high — all three pass identically against the old `exec.Command("kill","-0",pid)`. I measured the differences that are now unguarded: `kill -0 0` exits 0 (old `Alive("0")` was **true**, new is false) and `kill -0 1` exits 1 (old `Alive("1")` was **false**, new is true via EPERM). Fix: assert `Alive("0")`, `Alive("-1")`, `Alive("abc")` are false and `Alive("1")` is true. Without these, the guard PQ-4 asked for is a field set at zero assertion sites.

**I5 — the generated agreement oracle exercises only one of the two answering branches** (`layoutcmd_test.go:277-305`).
`subsets` is narrowed to `{"empty","full"}`, so registry is either nil (always declines) or `{"3","4"}`. Every one of the 54 answered cases therefore comes from the *record-hit* branch, and the record axis collapses too — with a full registry, `"unregistered-present"` and `"live-registered"` are the same case. The `len(liveIDs)==1` branch, the one that returns an id with no cross-check of any kind, is agreement-checked nowhere; it appears only as a single hand-written row in the fall-back test. The `case "subset"` / `case "disjoint"` arms (`:319`, `:323`) are now dead code that reads like live coverage. Fix: add a consistent single-half configuration (one right-terminal pane, registry containing exactly it) to the generated space, and delete or separately drive the dead arms.

## 4. Minor findings

- `resolveRightTerminal` (`layoutcmd.go:90`) now fabricates `zellijpane.Pane{ID: id}` — a struct whose other fields are zero, not unknown. Both callers read only `.ID`; collapse the wrapper and let them call `resolveRightTerminalID` directly, so no fabricated pane escapes for a future caller to read `.IsFocused` off.
- `procutil.go:32` — "the distinction the exit-status check silently got right before" is backwards; `kill -0 <other-user-pid>` exits 1, so the old code called an EPERM process dead. The new behavior is a fix, not a preservation, and the comment should say so.
- ARCH-SECURE, family left open from PQ-4: `LiveIDs` (`workbenchshortcut/shortcut.go:590`) validates the *pid* as numeric but never the *pane id*, and the fast path now hands that field straight to `focus-pane-id` / `write --pane-id` without the pane-report intersection that used to constrain it to a real pane. Low likelihood (zellij sets `ZELLIJ_PANE_ID` numeric), but the guard belongs next to the pid guard, one source for both.
- Plan-vs-code drift, two instances: the caller table says `currentRightTerminalPane` is "sidecar-first, same resolver" (it uses an inline registry-membership check, a different question, and reasonably so), and the Spec's "must produce the SAME id the slow path would in the cases it handles" is now scoped to registry-consistent states. Both are recorded only in the `## Log`. See §7.

## 5. Test coverage notes

The layoutcmd half is genuinely well covered — pure function, generated space, and call-count assertions on the IO shell. The termcmd half has none (I1), and the diff's most reachable production risk lives there: `focusedWorkbenchPanes` decides the *role* every terminal chord routes on, and it now decides it from synthesised data. The single new termcmd test guards the constant but not the path. `procutil` (I4) is the clearest instance of the "fix without a failing test" pattern this gate exists to catch — I verified by measuring the old semantics rather than reading the comment, and the two behaviors the Spec commits to are both invisible to the suite.

## 6. Architectural notes for upcoming work

- ARCH-DRY: pass with two small exceptions. `func(pid int) bool { return procutil.Alive(strconv.Itoa(pid)) }` is duplicated verbatim at `layoutcmd.go:314` and `run.go:1693` (pre-existing, but now sillier — `Alive` parses the string straight back to an int). A `procutil.AlivePID(int) bool` with `Alive` delegating would remove both adapters. Separately, "id ∈ ids" is written out three times (`layoutcmd.go:61`, `run.go:603`, `shortcut.go:608`).
- ARCH-PURE: pass for `resolveFromSidecars`; flagged for `focusedWorkbenchPanes` (I2).
- ARCH-PURPOSE: pass on the sweep — five callers enumerated, four converted or already cache-first, one dispositioned with a reason that survives the measurement (`--geometry` costs nothing on top of `--json`, so geometry callers cannot avoid the 590ms). Flagged once: PQ-4's class was "the registry is untrusted input"; the fix took the pid instance and left the pane-id sibling.
- ARCH-MOCK: flagged (I2) — the fake exists and is bypassed.
- ARCH-CONSTRAINTS: pass, and better than most. The envelope was measured before designing, re-measured after, and both numbers are in the `## Log` with the host's load. The layoutcmd path even has a machine-checkable guard against re-introducing the cost.
- ARCH-SECURE: mostly pass — the draft cache's session + liveness validation is reused rather than reimplemented. One gap (pane-id shape, Minor).
- ARCH-ORDER: pass on the modeled part — the registry's subset semantics and the startup race are enumerated in the Plan and asserted over a generated state space. Flagged where it matters most: I2 means the branch a termcmd test takes depends on ambient env, so a green run there is a sample of size one from an unstated environment.

## 7. Plan revision recommendations

Append a `## Revisions` entry to `workshop/issues/000220-…md` (timestamp + reason + delta) covering both drifts:

1. **Spec, agreement bullet.** "The fast path must produce the SAME id the slow path would in the cases it handles" is now scoped: agreement holds over registry-*consistent* pane sets; where the registry is a proper subset or disjoint, the guarantee is the weaker "never returns an id the registry did not list". The `## Log` records this; the Spec still states it absolutely, and the Spec is what the next reader greps.
2. **Plan, caller table row for `termcmd:568 currentRightTerminalPane`.** "sidecar-first, same resolver" is not what shipped — it uses an inline registry-membership check, because its question is "is the current pane a right terminal", not "which right terminal". The code is right; the row should say so, otherwise a future reader consolidating the two "duplicated" resolvers will merge two different questions.

```findings
findings:
  - id: new
    severity: Important
    family: fastpath-untested
    title: |
      Both termcmd fast paths execute in zero tests (coverage profile: count 0)
    detail: |
      run.go:165-169 and run.go:604-606 are never entered by any test; the coverage
      profile reports count 0 for both blocks. No fixture sets currentPaneID and a
      registry containing it. Add a fakeRuntime with failList:true asserting
      splitTerminalDown succeeds with listCalls == 0, and the same for
      focusedWorkbenchPanes once the seam bypass is fixed.
  - id: new
    severity: Important
    family: seam-bypass
    title: |
      focusedWorkbenchPanes calls draftroute.CachedDraftPaneIDFromEnv instead of rt.CachedDraftPaneID
    detail: |
      run.go:164 reads os.Getenv plus the real filesystem directly, though
      Runtime.CachedDraftPaneID exists (run.go:29) and OSRuntime implements it as
      exactly that call (run.go:1669). Production and test flow no longer share the
      boundary (ARCH-MOCK), the fast path becomes untestable with the fake, and which
      branch a test takes depends on ambient PAIR_DATA_DIR/PAIR_TAG/ZELLIJ_SESSION_NAME.
  - id: new
    severity: Important
    family: unchecked-fastpath-premise
    title: |
      Fast path synthesises role RightTerminal from a bare ZELLIJ_PANE_ID with no registry check
    detail: |
      run.go:163 trusts any non-empty current pane id, unlike its sibling at run.go:601
      which checks registry membership. handleChord is also reachable via --test-shortcut,
      which tests/term-pane-shortcuts-test.sh drives from the draft's and the review pane's
      perspective inside a real zellij pane that exports ZELLIJ_PANE_ID; the Plan's "the
      --test-shortcut path has no live pane" is false there. Gate on
      currentID in rt.TerminalPaneIDs() (ARCH-DRY with run.go:601).
  - id: new
    severity: Important
    family: claimed-fix-unpinned
    title: |
      procutil.Alive's two declared behavior changes are pinned by no test
    detail: |
      procutil_test.go:11 asserts only empty/self/bogus-high, all of which pass against the
      old exec-based implementation. Measured: kill -0 0 exits 0 (old Alive("0") was true,
      new false) and kill -0 1 exits 1 (old Alive("1") was false, new true via EPERM).
      Assert Alive("0"), Alive("-1"), Alive("abc") false and Alive("1") true, else PQ-4's
      positivePID guard is protection with zero assertion sites.
  - id: new
    severity: Important
    family: agreement-oracle-strength
    title: |
      Generated agreement space never exercises the single-live-id branch
    detail: |
      layoutcmd_test.go:304 narrows subsets to empty/full, so all 54 answered cases come from
      the record-hit branch and the record axis collapses (with a full registry
      "unregistered-present" == "live-registered"). The len(liveIDs)==1 branch — the one that
      answers with no cross-check — is agreement-checked nowhere. The case "subset" and
      case "disjoint" arms at :319/:323 are dead code that reads as coverage.
  - id: new
    severity: Minor
    family: fabricated-value-escapes-type
    title: |
      resolveRightTerminal fabricates a zellijpane.Pane with only ID populated
    detail: |
      layoutcmd.go:90 wraps the id in a Pane whose remaining fields are zero rather than
      unknown. Both callers read only .ID; drop the wrapper and call resolveRightTerminalID
      directly so no synthetic pane escapes for a later caller to read .IsFocused off.
  - id: new
    severity: Minor
    family: comment-vs-measured-behavior
    title: |
      Alive's doc comment misstates the old EPERM behavior
    detail: |
      procutil.go:32 says the exit-status check "silently got right" the EPERM distinction.
      Measured: kill -0 on another user's pid exits 1, so the old code reported it dead.
      The change is a fix, not preservation.
  - id: new
    severity: Minor
    family: untrusted-sidecar-parse
    title: |
      Registry pane id is never validated as a pane id, only the pid is
    detail: |
      shortcut.go:590 parses fields[1] as an int but takes fields[0] verbatim, and the fast
      path now hands it to focus-pane-id / write --pane-id without the pane-report
      intersection that previously constrained it to a real pane. PQ-4 named the registry as
      untrusted input; the fix took the pid instance and left this sibling (ARCH-SECURE).
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Spec's unscoped agreement claim and the Plan's "same resolver" row no longer match the code
    detail: |
      The Spec still states the fast path produces the same id as the slow path in every case
      it handles; the delivered guarantee is scoped to registry-consistent states. The Plan's
      caller table says currentRightTerminalPane uses the same resolver; it uses an inline
      registry-membership check, correctly, since its question is different. Record both in a
      "## Revisions" entry rather than leaving them only in the Log.
```

---

## Re-review — 2026-09-09T15:24:19-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 220 — zellij list-panes --json costs 590ms, so every pane-resolving chord pays it |
| repo | pair |
| issue file | workshop/issues/000220-zellij-list-panes-json-costs-590ms-so-every-pane-resolving-chord-pays-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | 21cc7f7791587351601a8274043f62493d0edd98..d9086ee20120ef7bd697014df5a6f30f870ba3fa |
| command | sdlc close --issue 220 |
| reviewer | claude |
| timestamp | 2026-09-09T15:24:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core of #220 is sound and round 2 genuinely closed the Important findings — I verified each by mutation in a scratch copy rather than by reading the commit message. `resolveFromSidecars` (`layoutcmd.go:58`) is a real pure decision function; both `termcmd` fast paths now execute (coverage count 1 on `run.go:188` and `run.go:625`, was 0); the BR-5 registry gate reddens two tests when removed; `Alive`'s `positivePID` guard and EPERM branch each redden a test when removed; and the rebuilt generator's single-live-id branch is genuinely agreement-checked — returning a bogus id from it reddens 18 cases. What keeps this off SHIP is one new Important finding and five Minor carry-overs: the reachability guard added to close BR-7 does not measure the branch it names (deleting the branch outright leaves the test green while still logging "36 via the single-live-id branch"), and BR-8/9/10/11 plus BR-1 are all still literally true in the tree. None is a correctness bug on the production path. Verification: `go vet ./cmd/...` clean; `go test ./cmd/...` — every failure is the documented `ptychild`/`operation not permitted` pty-environment class, none in `layoutcmd`/`procutil`/`termcmd`'s non-pty tests.

## 1. Strengths

- **The BR-5 fix is the right one, not the convenient one.** `registered()` (`run.go:159`) gates on registry membership rather than on `ZELLIJ_PANE_ID` alone, and it is shared by both fast paths — one predicate, two callers (ARCH-DRY). Mutation confirms: dropping `&& registered(rt, currentID)` from both sites reddens `TestFocusedWorkbenchPanesFallsBackWhenTheGateFails/pane_is_not_registered` and `TestCurrentRightTerminalPaneSkipsThePaneListWhenRegistered`.
- **BR-4's seam restoration made BR-3 testable, and the diff says so at the site** (`run.go:186-187`). `draftroute.CachedDraftPaneIDFromEnv` now appears only in `OSRuntime` (`run.go:1688`). Production flow and test flow share the `Runtime` boundary again (ARCH-MOCK).
- **`procutil.Alive`'s two declared behaviours are pinned by tests that fail without the fix.** Replacing `positivePID` with a bare `strconv.Atoi` reddens with `Alive("0") = true` and `Alive("-1") = true`; dropping `|| errors.Is(err, syscall.EPERM)` reddens with `Alive(1) = false`. That is the standard the claimed-fix check asks for.
- **The generator rebuild around *completeness* rather than *size* (BR-7) was the correct diagnosis** — `layoutcmd_test.go:311-322`. The `case "subset"` / `case "disjoint"` dead arms are gone, and 360 cases now reach the branch that previously could not be produced.
- **`TestSidecarFastPathFallsBackWhenItCannotAnswer` asserts `listCalls` in both directions** (`layoutcmd_test.go:409-441`), so it pins the 590ms saving *and* the fact that the tie-break is not eaten.

## 2. Critical findings

None.

## 3. Important findings

**The reachability guard for the single-live-id branch counts an input predicate, not the branch** — `layoutcmd_test.go:349-350`, guard at `:371`, log at `:374`.

> **This is the 2nd finding in family `agreement-oracle-strength`.** Round 1 fixed the instance (the branch was unreachable); do NOT just fix this instance. The rule that covers both: **a reachability/coverage guard in a generated test must be incremented at the site it names, not derived from an input predicate that correlates with it** — otherwise the guard survives the branch's deletion and reports coverage the space no longer has.

Measured prevalence in this diff: 5 such guards, 1 wrong. `checked`, `answered`, and the two `listCalls` assertions are all incremented at their real sites; `single` is incremented on `len(registry) == 1`, which the *record* branch also satisfies. Deleting the `len(liveIDs) == 1` branch from `resolveFromSidecars` entirely leaves `TestSidecarFastPathAgreesWithThePaneListWheneverItAnswers` **PASSING** and still logging `answered 90, of which 36 via the single-live-id branch`. The reported 54 is also wrong today: only 18 generated tuples actually enter that branch (record=`none`, size=1, complete=true), and because `kindB` is a dead axis when `size == 1`, those 18 are 2 distinct worlds repeated 9 times. Fix sketch: make the counter exact (`if lastTerminal == "" && len(registry) == 1`) — or better, per the rule, have `resolveFromSidecars` return which branch answered and count on that, so the guard cannot drift from the code again; and skip the `kindB` loop when `size == 1` so the case count stops overstating the space.

## 4. Minor findings

- **BR-1 not-addressed** (`run.go:1684` vs `layoutcmd.go:305`): termcmd's `OSRuntime.ListPanesJSON` still omits `--geometry`, and `handleTerminalChord` reaches `RunToggleFocused` with it at `run.go:588`. Traced to ground this round: `zellijpane` documents geometry as present only "when callers request `list-panes --geometry`", so `focused.Columns == 0` → `terminalToggleBurst(0, 0)` returns `false` (`resizeplan.go:27`) → `Alt+Shift+Enter` pressed *inside* the right terminal is silently inert. Pre-existing and outside #220's window; my call is that it is a separate issue, and it should be filed rather than carried further on this one.
- **BR-8 not-addressed, and the class grew** (`layoutcmd.go:95`): `resolveRightTerminal` still wraps the id in a `zellijpane.Pane` whose other fields are zero, not unknown. This diff added three more synthesised panes (`run.go:190`, `:191`, `:626`), so the family is now 4 sites, not 1. Both `layoutcmd` callers read only `.ID`; the `run.go:190` one is the one that matters, since it deliberately populates `TerminalCommand` to survive `RoleForPaneWith` while leaving `X`/`Columns`/`IsFocused` fabricated as zero for any future reader.
- **BR-9 not-addressed, now at two sites** (`procutil.go:37`, `procutil_test.go:129`): re-measured on this host — `/bin/kill -0 1` exits **1** ("Operation not permitted"), so the old `exec.Command("kill", "-0", ...)` reported PID 1 dead. Both the doc comment ("the distinction the exit-status check silently got right before") and the new test comment ("got this right by accident via its exit status") state the opposite of the measurement. The change is a fix, not a preservation.
- **BR-10 not-addressed** (`shortcut.go:588-595`): `fields[0]` is still taken verbatim while `fields[1]` is parsed. Sibling worth folding in: the registry line carries **no session field**, unlike `CachedPaneRecord.Session` which `ValidateCachedDraftPane` checks (`route.go:44-46`) — so a registry scoped only by `PAIR_DATA_DIR`+`PAIR_TAG` can name a pane from another zellij session, and the fast path now hands that id to `focus-pane-id`/`write --pane-id` without the pane-report intersection the slow path applied. Argv is an array, so there is no injection surface; the failure mode is a silently wrong or inert target (ARCH-SECURE).
- **BR-11 not-addressed**: the issue file has no `## Revisions` section (grep confirms), the Spec's unscoped "must produce the SAME id the slow path would in the cases it handles" is unchanged, and the Plan's `termcmd:568` row still says "same resolver" while the code uses the inline `registered()` check.
- `registered()` reads `rt.TerminalPaneIDs()` and then `handleChord` reads it again at `run.go:131`; two registry reads per fast-path keypress. Negligible against 590ms, noted only because the same list is now scanned by three near-identical membership loops (`layoutcmd.go:63-67`, `run.go:163-167`, `shortcut.go:608-612`).

## 5. Test coverage notes

The bug class this diff could ship — the two paths landing on different split halves — is now covered from both sides, and the coverage claim is checkable rather than asserted: both `termcmd` fast-path blocks report count 1 where round 1 measured 0. The one gap is the guard described in §3. Two environment dependencies in `procutil_test.go` are worth knowing about but not worth fixing: `Alive("1")` assumes PID 1 exists and is not ours (it passes either way — as root it takes the `err == nil` path, so the EPERM branch is only genuinely exercised as non-root), and `Alive("4194305")` assumes the platform pid maximum, which a Linux host with a raised `pid_max` could invalidate. Neither is flaky in this repo's actual environment.

## 6. Architectural notes

- **ARCH-DRY** — flag, Minor only: three copies of the id-membership loop (§4). The shared `registered()` helper across both `termcmd` fast paths is the right consolidation at the level that mattered.
- **ARCH-PURE** — pass. `resolveFromSidecars` takes two slices and returns `(string, bool)`; its tests run with zero IO, exactly as the plan's Core-concepts row claims. `resolveRightTerminalID` is the thin IO shell around it.
- **ARCH-PURPOSE** — pass on the shadow-sweep. All five `ListPanesJSON` consumers enumerated in the Plan exist at the stated paths and each derives its disposition from the stated rule; `grep -n list-panes` finds exactly the two `OSRuntime` implementations and no hand-rolled third. Partial flag: BR-2's *rule* ("fall back when ANY substituted field is unavailable") is implemented in code and pinned by the `no cached draft` case, but written down nowhere — the atlas states the ID-vs-geometry rule, not the disjunctive-fallback one.
- **ARCH-MOCK** — pass. `fakeRuntime` in both packages sits on the same `Runtime` seam production uses; BR-4's bypass is gone. `syscall.Kill` is a syscall, not an external binary, and liveness is still injected into `LiveIDs(alive func(int) bool)`.
- **ARCH-CONSTRAINTS** — pass. Keystroke-path work with a stated budget and before/after medians in the `## Log` (631→55ms, 671→27ms), and the residue is accounted for rather than hand-waved.
- **ARCH-SECURE** — flag, covered by BR-10 (§4). Everything else degrades visibly: a registry read error narrows to `nil` and falls back; a partial append-only line fails `len(fields) != 2` and is skipped; the draft cache is session- and pid-validated.
- **ARCH-ORDER** — pass, with the claim written out rather than left bare: each pane-resolving command is a fresh short-lived process holding no state between events; the only carried state is the two sidecars, read whole per invocation. The one ordering event that matters — a `pair term` that has not self-registered yet — is enumerated in the Plan with its chosen disposition (fall back) and its accepted residual risk.

## 7. Plan revision recommendations

The issue still has no `## Revisions` section. Two entries are owed (this is BR-11, re-raised as still open rather than as new):

- **Spec, agreement scope.** Replace "The fast path must produce the SAME id the slow path would in the cases it handles" with the guarantee the code and the generator actually deliver: agreement holds whenever the registry is *consistent* with the pane report (complete or empty), and the invariant that holds unconditionally is that the fast path never returns an id the registry did not list. Reason: the generator found two divergence classes (incomplete registry with no record; disjoint registry) and the response was to scope the property — that scoping currently lives only in the `## Log`.
- **Plan, `termcmd:568` caller row.** "sidecar-first, same resolver" → sidecar-first via an inline registry-membership check (`registered()`), not `resolveFromSidecars`. Reason: its question is "is *this* pane a right terminal?", not "which right terminal?", so a different predicate is correct — but the table currently claims a code-sharing that does not exist.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Still true; re-measured the consequence (Alt+Shift+Enter inert from the terminal pane). Pre-existing, outside #220's window — file as its own issue.
  - id: BR-2
    disposition: addressed
    note: |
      Fast path now gates on all three substituted fields; the "no cached draft" case pins the fallback. The general rule is implemented but not written down.
  - id: BR-3
    disposition: addressed
    note: |
      Coverage profile now reports count 1 on run.go:188 and run.go:625 (was 0), and the saving is asserted via listCalls.
  - id: BR-4
    disposition: addressed
    note: |
      draftroute.CachedDraftPaneIDFromEnv now appears only at run.go:1688 (the OSRuntime impl); the fast path uses rt.CachedDraftPaneID().
  - id: BR-5
    disposition: addressed
    note: |
      Verified by mutation — removing "&& registered(rt, currentID)" from both sites reddens two tests.
  - id: BR-6
    disposition: addressed
    note: |
      Verified by mutation — dropping positivePID reddens Alive("0")/Alive("-1"); dropping the EPERM branch reddens Alive("1").
  - id: BR-7
    disposition: addressed
    note: |
      Verified by mutation — a bogus return from the single-live-id branch reddens 18 generated cases. The branch is agreement-checked; its reachability GUARD is not (new finding).
  - id: BR-8
    disposition: not-addressed
    note: |
      layoutcmd.go:95 unchanged, and this diff added three more synthesised panes (run.go:190, :191, :626) — the class is now 4 sites.
  - id: BR-9
    disposition: not-addressed
    note: |
      Re-measured: /bin/kill -0 1 exits 1. procutil.go:37 still claims the exit-status check got it right, and procutil_test.go:129 now repeats the claim.
  - id: BR-10
    disposition: not-addressed
    note: |
      shortcut.go:588 unchanged. Sibling: the registry line carries no session field, unlike CachedPaneRecord, so a cross-session id can now reach focus-pane-id unintersected.
  - id: BR-11
    disposition: not-addressed
    note: |
      No "## Revisions" section exists in the issue file; the Spec's unscoped claim and the Plan's "same resolver" row are both unchanged.
findings:
  - id: new
    severity: Important
    family: agreement-oracle-strength
    title: |
      The single-live-id reachability guard counts an input predicate, so it survives the branch's deletion
    detail: |
      2nd in this family, so the deliverable is the RULE: a reachability/coverage guard
      in a generated test must be incremented at the site it names, not derived from an
      input predicate that correlates with it. layoutcmd_test.go:349 increments `single`
      on len(registry)==1, which the RECORD branch also satisfies. Measured: deleting
      the `len(liveIDs)==1` branch from resolveFromSidecars leaves the test PASSING and
      still logging "36 via the single-live-id branch". The reported 54 is wrong today
      too — only 18 tuples enter that branch, and since kindB is a dead axis when
      size==1, those 18 are 2 distinct worlds repeated 9 times. Prevalence in this diff:
      5 guards, 1 derived (checked/answered/listCalls x2 are all counted at their real
      sites). Fix the rule: return which branch answered and count on that, so the guard
      cannot drift from the code; and skip the kindB loop when size==1.
```

---

## Re-review — 2026-09-09T15:40:34-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 220 — zellij list-panes --json costs 590ms, so every pane-resolving chord pays it |
| repo | pair |
| issue file | workshop/issues/000220-zellij-list-panes-json-costs-590ms-so-every-pane-resolving-chord-pays-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | 21cc7f7791587351601a8274043f62493d0edd98..67f47b6afe012d9d6ab8a2eaabdff0d3d44772d5 |
| command | sdlc close --issue 220 |
| reviewer | claude |
| timestamp | 2026-09-09T15:40:34-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The one Important finding carried into this round (BR-12) is genuinely closed, and I confirmed it the way the rule demands rather than by reading the commit message: deleting the `len(liveIDs)==1` branch from `resolveFromSidecars` in a scratch worktree now fails the guard by name (`"no generated case was answered with NO recorded half … the single-live-id branch is unreached"`), where before it passed while still logging branch coverage. I also mutation-tested the record branch (dropping its membership loop reddens 108 generated cases) and confirmed by coverage profile that both `termcmd` fast paths now execute (`run.go:188`, `run.go:625`, count 1 each — they were 0 two rounds ago). `go build`, `go vet`, `gofmt` are clean; the affected packages pass; the only failures in `go test ./...` are the documented pty class (`ptychild: … operation not permitted`), which I reproduced at the base commit too. What keeps this off SHIP is one new Important — of the three fast paths this issue ships, only the `layoutcmd` one has a differential oracle against the slow path it replaces, and the Spec's third bullet applies to all three — plus five Minors that have now survived two rounds unaddressed. None blocks the gate.

## 1. Strengths

- `resolveFromSidecars` (`layoutcmd.go:58-79`) is the right shape and its purity is real, not claimed: the whole generated space runs with zero IO. ARCH-PURE passes cleanly.
- The reachability guard now discriminates. `layoutcmd_test.go:349` counts `lastTerminal == ""`, which under the current branch structure is an exact characterisation of the single-live-id branch — and the fix was proved by deletion, which is what `workshop/lessons.md` now requires of this class. I reproduced the proof independently.
- The oracle catches real divergence, not just its own shape. Replacing the record branch's membership loop with `return lastTerminal, true` reddens 108 cases with a message that names the actual hazard ("the two paths would land on different halves").
- `procutil.Alive`'s two declared behaviour changes are both pinned, and the test names the reason the `positivePID` guard exists (false positive from a `1 0` registry line, not a delivered signal) — a correction the implementor made against their own earlier comment.
- ARCH-PURPOSE shadow-sweep holds. I enumerated `list-panes --json` invocations across Go *and* the bundled Lua independently of the Plan: `launcher.ProbeLiveLayout` needs the full pane set, `opener.AgentPaneID` and `nvim/pair_poke.lua` resolve an id no sidecar carries, and `nvim/workbench_route.lua:91` is already cache-first. The Plan's five is the complete sweepable class; nothing that could derive from a sidecar was left as a "follow-up".

## 2. Critical findings

None.

## 3. Important findings

**N1 — two of the three fast paths pin their *saving* and their *gate*, but never their *answer*** (`cmd/internal/termcmd/run_test.go:1243`, `:1288`).

**This is the 2nd finding in family `fastpath-untested`.** BR-3 fixed the instance ("both `termcmd` fast paths execute in zero tests"), so the deliverable here is the rule, not the site:

> A fast path introduced as a substitute for an existing slow path is not tested by asserting that it skipped the expensive call (`listCalls == 0`) and that it declines when gated. It is tested when its **answer** is differentially pinned against the slow path's answer on a shared fixture. Asserting a hardcoded literal instead re-asserts the implementation.

Measured prevalence in this diff: 3 fast paths, 1 differentially pinned. `layoutcmd`'s has a 360-case generated agreement oracle. `focusedWorkbenchPanes` (`run.go:188`) and `currentRightTerminalPane` (`run.go:625`) have none — the new tests assert `listCalls`, the gate's three decline cases, and hardcoded IDs.

Concrete gap it leaves: `TestFocusedWorkbenchPanesSkipsThePaneListWhenRegistered` sets `cachedDraft: "2"` and the fall-back fixture's draft pane is *also* id 2, so `panes.draft.ID != "2"` cannot distinguish "reads the cache" from "agrees with the report". Set `cachedDraft: "7"` against the same report and the fast path returns 7 with the suite still green. That value is live: `Decide`'s right-terminal `Alt+K` targets `DraftPaneID` whenever `LastLeftPaneID` is empty (`shortcut.go:265-267`) — the first `Alt+k` of a session started in the terminal. Fix that implements the rule: for each fast path, run the *same* fixture twice — once with the gate satisfied, once with `terminalPaneIDs` nil — and assert the two results are equal, instead of comparing against a literal.

## 4. Minor findings

- ARCH-DRY: `registered` (`run.go:159`) is the third open-coded "is this pane id in the live registry" loop, alongside the one inside `resolveFromSidecars` (`layoutcmd.go:66`) and `RoleForPaneWith` (`shortcut.go:606`). `workbenchshortcut` owns the registry; a `Registered(ids []string, paneID string) bool` there would give all three one source.
- `currentRightTerminalPane`'s comment cites its caller as `run.go:545`; `splitTerminalDown` is at `run.go:595`.
- `if len(liveIDs) == 0 { return "", false }` (`layoutcmd.go:59`) is behaviourally redundant — both downstream branches already decline on an empty slice. Harmless as documentation; worth knowing no test can pin it.
- The `kindB` axis is dead when `size == 1`, so the 18 single-live-id cases the guard reports are 6 distinct worlds repeated three times. Cosmetic now that the guard itself discriminates, but the logged number still overstates coverage 3x. (Part of BR-12's ask; see its disposition.)

## 5. Test coverage notes

Coverage of the new surface is now real where it was absent: both `termcmd` fast paths execute, `procutil.Alive`'s two behaviour changes each redden under mutation, and the generated space reaches the branch it claims. The remaining hole is the one N1 names. Two structural observations for later: `TestSidecarFastPathNeverInventsAnID` is a hand-enumerated 25-tuple space, not a generated one, and it survives only because `resolveFromSidecars` returns elements of `liveIDs` — it is the weakest oracle in the file, though I confirmed it is not tautological. And the residual pty failures (`ptychild`, `hostty`, `couchcore`) are environmental and reproduce at the base commit; they are not evidence about this diff either way.

## 6. Architectural notes

Marker by marker, at-review lens:

- **ARCH-DRY** — flag (Minor above). `resolveRightTerminalID` correctly collapses both `layoutcmd` callers onto one shell, and `resolveFromSidecars` mirrors `ValidateCachedDraftPane` rather than inventing a shape. The membership predicate is the one duplication.
- **ARCH-PURE** — pass. The pure decision is genuinely pure; the IO shell is thin and named as such.
- **ARCH-PURPOSE** — pass. Shadow-sweep above; no consumer left as a hand-maintained restatement.
- **ARCH-MOCK** — pass. zellij stays behind the `Runtime` seam in both packages, both fakes count `listCalls` so production and test flow share the boundary, and `Alive`'s tests exercise real pids rather than a stub. Note for later: the 590ms figure that justifies the whole design is a `## Log` measurement, not a conformance check — a zellij upgrade that fixes `--json` would leave the fast path as pure complexity with nothing reporting it.
- **ARCH-CONSTRAINTS** — pass. Keystroke workload, budget stated, before/after measured on the same host (631→55ms, 671→27ms), and the regression guard is structural (`listCalls == 0`) rather than a timing assertion. The `termcmd` fast paths were not measured end to end, which the Log states honestly and the Done-when does not require.
- **ARCH-SECURE** — flag, and it is BR-10, still open: the registry's pid is parsed, its pane id is not, and the fast path now hands `fields[0]` (`shortcut.go:595`) straight to `focus-pane-id` / `write --pane-id` without the pane-report intersection that used to constrain it.
- **ARCH-ORDER** — pass with a note. `resolveFromSidecars` holds no state between events; the ordering event that matters is the registration race, and it is generated as an input class with the decline behaviour asserted. The interleaving *not* covered anywhere is a half closing between resolution and delivery — the atlas records it as fire-and-forget, which is a policy, not a test.

## 7. Plan revision recommendations

The issue file still has no `## Revisions` section. Three deltas belong in one entry:

1. **Spec, third bullet** — "The fast path must produce the SAME id the slow path would in the cases it handles" is unscoped; the delivered guarantee holds over registry-*consistent* states, and over inconsistent ones only the weaker "never returns an id the registry did not list". The Log says this; the Spec still promises the stronger thing.
2. **Plan, caller table, `termcmd:568` row** — "sidecar-first, **same resolver**" is wrong: `currentRightTerminalPane` uses an inline registry-membership check, correctly, because its question ("am *I* a right terminal?") differs from `resolveRightTerminal`'s.
3. **Plan, caller table, `termcmd:144` row** — the justification "falls back to the pane list when `CurrentPaneID()` is empty (the `--test-shortcut` path, which has no live pane)" was shown false by BR-5 and superseded by the registry gate. The row should say what the code does: gate on registry membership, fall back on any of the three misses.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      Verified by mutation in a scratch worktree — deleting the single-live-id branch now fails the guard by name; residue: the kindB axis is still dead when size==1, so the reported 18 is 6 distinct worlds.
  - id: BR-1
    disposition: not-addressed
    note: |
      Re-confirmed: termcmd's ListPanesJSON (run.go:1684) omits --geometry, layoutcmd's (layoutcmd.go:305) has it, so Alt+Shift+Enter from the terminal is inert — pre-existing, close it by filing its own issue rather than by more work in 220.
  - id: BR-8
    disposition: not-addressed
    note: |
      layoutcmd.go:95 unchanged; the class is 4 sites now (run.go:190, :191, :626).
  - id: BR-9
    disposition: not-addressed
    note: |
      Re-measured on this host: /bin/kill -0 1 exits 1, so the old code called an EPERM process dead; procutil.go:37 and procutil_test.go:129 both still claim otherwise.
  - id: BR-10
    disposition: not-addressed
    note: |
      shortcut.go:587-595 unchanged — fields[1] is parsed as an int, fields[0] is taken verbatim and now reaches focus-pane-id unintersected.
  - id: BR-11
    disposition: not-addressed
    note: |
      No "## Revisions" section exists; and a third instance has accumulated — the Plan's termcmd:144 row still carries the "--test-shortcut path has no live pane" justification BR-5 falsified.
findings:
  - id: new
    severity: Important
    family: fastpath-untested
    title: |
      Two of the three fast paths pin their saving and their gate but never their answer
    detail: |
      2nd in this family, so the deliverable is the RULE: a fast path substituting for an
      existing slow path is not tested by asserting listCalls==0 and that it declines when
      gated; it is tested when its ANSWER is differentially pinned against the slow path on
      a shared fixture. Measured prevalence in this diff: 3 fast paths, 1 pinned —
      layoutcmd's resolveFromSidecars has a 360-case agreement oracle, focusedWorkbenchPanes
      (run.go:188) and currentRightTerminalPane (run.go:625) have none. Concrete gap:
      run_test.go:1243 sets cachedDraft "2" while the fall-back fixture's draft is also id 2,
      so the assertion cannot distinguish "reads the cache" from "agrees with the report" —
      set cachedDraft to "7" and the fast path returns 7 with the suite green. DraftPaneID is
      live via shortcut.go:265-267 (right-terminal Alt+K with an empty LastLeftPaneID). Fix
      the rule: run each fixture twice, gate satisfied and gate unsatisfied, and assert the
      two results are equal instead of comparing to a literal.
  - id: new
    severity: Minor
    family: shared-predicate-not-extracted
    title: |
      Registry membership is open-coded a third time instead of living with the registry
    detail: |
      run.go:159 `registered` duplicates the same "is this pane id in the live registry" loop
      that resolveFromSidecars carries at layoutcmd.go:66 and RoleForPaneWith at
      shortcut.go:606. workbenchshortcut owns the registry; export
      Registered(ids []string, paneID string) bool there and have all three call it (ARCH-DRY).
```

---

## Re-review — 2026-09-09T15:59:35-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 220 — zellij list-panes --json costs 590ms, so every pane-resolving chord pays it |
| repo | pair |
| issue file | workshop/issues/000220-zellij-list-panes-json-costs-590ms-so-every-pane-resolving-chord-pays-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | 21cc7f7791587351601a8274043f62493d0edd98..51011ee2713e015e34e7dd8e8630011a828791cd |
| command | sdlc close --issue 220 |
| reviewer | claude |
| timestamp | 2026-09-09T15:59:35-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The two findings this round had to dispose — BR-13 (fast paths pin their saving, not their answer) and BR-14 (registry membership open-coded three times) — are both genuinely addressed, and I verified them the way the claimed-fix rule demands rather than by reading the commit messages: mutating `focusedWorkbenchPanes`' draft id to `draftID + "9"` reddens with *fast = focused "4" draft "79"; slow = ... draft "7" — the two paths disagree*, and mutating `currentRightTerminalPane` to `currentID + "9"` reddens the same way, so both termcmd fast paths are now differentially pinned against the slow path on a shared fixture with a draft id (7) distinct from every other id. `workbenchshortcut.Registered` has exactly three call sites and no fourth open-coded loop survives. The full suite shows no regression: the failing packages (`termcmd`, `hostty`, `ptychild`, `couch*`, `keyscmd`, `wrapcmd`, `pair-go`) fail identically at the base commit with `fork/exec: operation not permitted` — the documented pty/exec environment class — while `layoutcmd`, `workbenchshortcut`, `procutil` and `draftroute` are green. What keeps this off SHIP is five carried Minors that are all one-liners and keep getting deferred, one of which (BR-9) is a comment stating something I measured to be false, and one of which (BR-11) is now four accumulated Spec/Plan claims the code does not deliver.

## 1. Strengths

- **The generated agreement oracle is the real thing** (`cmd/internal/layoutcmd/layoutcmd_test.go:274-381`). 360 cases, 108 answered, 18 through the single-live-id branch, and the reachability guard now keys on *answered with no recorded half* — a signature only that branch can produce — rather than on `len(registry) == 1`. It logs its own coverage, so a future generator change that stops reaching the branch announces itself.
- **The scoping of the guarantee is honest rather than convenient.** `TestSidecarFastPathNeverInventsAnID` (layoutcmd_test.go:392) asserts the weaker invariant that actually holds on inconsistent registries instead of asserting a coincidence. That is the right call and the Log says why.
- **`resolveFromSidecars` (layoutcmd.go:61-73) is a genuinely pure decision** — four branches, each with the reason it declines written next to it, no `Runtime`, no IO, mirroring `draftroute.ValidateCachedDraftPane`'s established shape (ARCH-PURE, ARCH-DRY).
- **`procutil.Alive`'s EPERM branch is pinned against the real kernel** (`procutil_test.go:126`, `Alive("1")`), not a fake — a live conformance check for the one semantic the syscall port changed.
- **The atlas paragraph states the rule for the next caller**, not just what happened: *"if you need a pane ID, resolve sidecar-first; if you need geometry or the full pane set, you must list panes."* That is the durable form of the decision.

## 2. Critical findings

None.

## 3. Important findings

None newly raised. Both prior Important-class findings are disposed `addressed` and mutation-verified.

## 4. Minor findings

- **BR-9 not-addressed — and I re-measured it.** `procutil.go:35` still reads *"the distinction the exit-status check silently got right before."* `kill -0 1` on this machine exits 1 (`kill 1 failed: operation not permitted`), so the old code reported another user's process **dead**. The change is a behavior fix; the comment claims preservation. One-line edit.
- **BR-11 not-addressed — the owed `## Revisions` entry is now four deltas, not two.** No `## Revisions` section exists in `workshop/issues/000220-…md`. Beyond the Spec's unscoped agreement claim and the Plan's `termcmd:568` "same resolver" row, and the `termcmd:144` row's falsified `--test-shortcut` justification, the Plan's *"What the registry is, and is not"* section claims *"the slow path already trusts this registry … so this is not new trust, only earlier trust."* That is false as written: the slow path trusted the registry as a **classifier over panes zellij reported**, so it could never return an id absent from the report; the fast path trusts it as the **source of the id**, with no such intersection. Same sentence, different trust.
- **BR-10 not-addressed, with a stronger mechanism than the round that raised it had.** `shortcut.go:590` still takes `fields[0]` verbatim. Two facts sharpen it: `TerminalPaneRegistry` carries **no session field** (unlike the draft cache, which `ValidateCachedDraftPane` rejects on `record.Session != session`), the file is append-only and never compacted, and `LiveIDs` filters on **pid liveness alone** — while `procutil.Identity` exists in the same package and its doc comment says *"It changes even when the OS recycles the same numeric PID quickly."* So a stale line from a previous session whose pid has been recycled now yields an id that goes straight to `focus-pane-id`/`write --pane-id`. It degrades visibly (the action errors and exits 1) and self-heals on the next recorded half, which is why it stays Minor — but the cross-check that used to make it unreachable is the one this diff removed.
- **BR-8 not-addressed.** `layoutcmd.go:90` still returns `zellijpane.Pane{ID: id}`. Both callers read only `.ID`; dropping the wrapper and exporting `resolveRightTerminalID` to them stops a synthetic pane escaping for a later caller to read `.IsFocused` off.
- **BR-1 not-addressed, and I recommend it become its own issue.** Confirmed unchanged: `layoutcmd.OSRuntime.ListPanesJSON` (layoutcmd.go:303) passes `--geometry`; `termcmd.OSRuntime.ListPanesJSON` (run.go:1679) does not; `runDecision` reaches `RunToggleFocused` with the termcmd runtime at run.go:256 and run.go:583. `tiledScreenSize` reads `X`/`Columns`/`Rows`, so `Alt+Shift+Enter` **from the terminal pane** computes screen size from zeroed geometry. This is pre-existing and untouched by #220's window — it belongs to a separate issue, not to this close.
- **New (raised below): the round-3 review rule never reached `workshop/lessons.md`.** Round 2's rule is there (`lessons.md:4225`); BR-13's rule — *a fast path is pinned by its answer, not by its saving* — lives only in the issue Log and a test comment. AGENTS.md §4 makes lessons.md the durable form.
- Not raised, recorded for completeness: on the fast path `handleChord` reads `rt.TerminalPaneIDs()` twice per chord (directly at run.go:132, and via `registered` at run.go:160), and the `currentID != "" && registered(rt, currentID)` gate is open-coded at run.go:178 and run.go:620. Both are microseconds on a path that went 631ms → 55ms; below the bar for action.

## 5. Test coverage notes

- The kind of bug this diff could ship — the two paths landing on different split halves — is now covered at all three fast paths, which was the whole point of BR-13's rule and is the one thing I checked by reverting rather than by reading.
- `TestFocusedWorkbenchPanesFallsBackWhenTheGateFails` covers all three decline conditions (unregistered, empty current id, cache miss) and asserts the list call actually happens, so the gate cannot silently become a no-op.
- The one class no test can currently express: an inconsistent registry where the fast path answers. That is deliberate and documented — asserting agreement there would assert a coincidence — but it means the pid-recycling phantom in BR-10 has no fixture, and would need `LiveIDs` to carry an incarnation before one could be written.
- Environment caveat for whoever re-runs: `termcmd`'s three pty failures are the documented sandbox class and are identical at base; scrub `PAIR_SESSION_ID`/`PAIR_TAG` (`env -u`) or the review-target and changelog tests fail spuriously inside a pair session.

## 6. Architectural notes

- **ARCH-DRY — pass.** BR-14's extraction is real and complete; `Registered` is the registry's own predicate with three callers and no survivor. Residue noted above is below the bar.
- **ARCH-PURE — pass, with an asymmetry worth knowing.** layoutcmd extracted the decision (`resolveFromSidecars`, pure, generated tests); termcmd left the structurally identical decision inline inside the IO function. The differential tests now cover it, so this is a shape note for the next fast path, not a finding.
- **ARCH-PURPOSE — pass.** The shadow sweep holds: all five `ListPanesJSON()` call sites are enumerated in the Plan and each is disposed; the two chords that motivated the issue are fast, and the geometry caller is a declared exception with a stated reason. BR-13's class was swept in one round across all three fast paths rather than at the site the finding named.
- **ARCH-MOCK — pass.** zellij stays behind `Runtime`; no direct external call escapes the seam in this diff; `Alive`'s changed semantics are checked against the real kernel, not a model of it.
- **ARCH-CONSTRAINTS — pass.** Keystroke workload, budget declared and measured before/after with host conditions (26 sessions, load 3.3, median of 6), residue attributed to its two components. No unbounded work introduced.
- **ARCH-SECURE — flag, folded into BR-10.** The registry is durable state written by other processes across session boundaries and is now the sole provenance of an id handed to a subprocess. The pid is parsed into a typed value; the pane id is not, and there is no incarnation identity despite `procutil.Identity` existing for exactly this.
- **ARCH-ORDER — flag, folded into BR-10/BR-11.** The startup race is enumerated and the agreement property explicitly excludes it — good. The event *not* enumerated is process death plus pid reuse across sessions, which the append-only, never-compacted, non-session-scoped registry makes representable and the removed pane-report intersection makes reachable.

## 7. Plan revision recommendations

One `## Revisions` entry in `workshop/issues/000220-…md` (timestamp + reason + delta) covering four items:

1. **Spec** — "The fast path must produce the SAME id the slow path would in the cases it handles" is unscoped; the delivered guarantee holds over registry-**consistent** states, with "never returns an id the registry did not list" as the invariant on inconsistent ones.
2. **Plan, `termcmd:568` row** — "sidecar-first, same resolver" is wrong; `currentRightTerminalPane` uses an inline `registered()` membership check, correctly, because its question ("am I a right terminal?") is not `resolveRightTerminal`'s ("which right terminal?").
3. **Plan, `termcmd:144` row** — the "`--test-shortcut` path, which has no live pane" justification was falsified by BR-5; the fast path now also gates on registry membership and on a cache hit.
4. **Plan, "What the registry is, and is not"** — "not new trust, only earlier trust" is false. The slow path's trust was bounded by the pane report; the fast path's is not, and that is the delta BR-10 turns on.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Verified by mutation, not by the commit message: draftID+"9" and currentID+"9" each redden the new differential test.
  - id: BR-14
    disposition: addressed
    note: |
      workbenchshortcut.Registered has exactly 3 call sites; grep finds no fourth open-coded membership loop.
  - id: BR-11
    disposition: not-addressed
    note: |
      Still no "## Revisions" section, and a fourth delta has accumulated - the Plan's "not new trust, only earlier trust" claim is falsified by the removed pane-report intersection.
  - id: BR-10
    disposition: not-addressed
    note: |
      fields[0] still verbatim; sharpened - the registry carries no session field and no incarnation identity though procutil.Identity exists, so a recycled pid now yields an answer.
  - id: BR-9
    disposition: not-addressed
    note: |
      Comment unchanged at procutil.go:35; I re-measured, kill -0 1 exits 1 here, so the old code reported another user's process dead.
  - id: BR-8
    disposition: not-addressed
    note: |
      layoutcmd.go:90 still returns zellijpane.Pane{ID: id}; both callers read only .ID.
  - id: BR-1
    disposition: not-addressed
    note: |
      Confirmed unchanged and pre-existing (run.go:1679 omits --geometry, reached via run.go:256/583); recommend a separate issue rather than this close.
findings:
  - id: new
    severity: Minor
    family: review-rule-not-recorded
    title: |
      Round 3's review rule reached the issue Log and a test comment but not workshop/lessons.md
    detail: |
      AGENTS.md section 4 makes lessons.md the durable form of a rule found in review, and
      round 2's rule was recorded there (lessons.md:4225). BR-13's rule - a fast path
      substituting for an existing slow path is pinned by its ANSWER, not by asserting
      listCalls==0 and that it declines when gated - exists only in the issue Log and in a
      comment above run_test.go:1236. grep for "fast path" in lessons.md finds no entry. It
      is the more generalizable of the two rules and the one most likely to recur in a
      different file.
```
