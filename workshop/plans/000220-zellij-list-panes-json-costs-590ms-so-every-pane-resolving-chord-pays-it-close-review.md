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
