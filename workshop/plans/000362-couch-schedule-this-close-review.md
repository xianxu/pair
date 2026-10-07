# Boundary Review — pair#362 (whole-issue close)

| field | value |
|-------|-------|
| issue | 362 — Teach Couch skill to schedule contextual work |
| repo | pair |
| issue file | workshop/issues/000362-couch-schedule-this.md |
| boundary | whole-issue close |
| milestone | — |
| window | d9cdc78ab8e0760422a51365251bcdff65cea452..713d2a984dfc061a2fd049c1528a946e46b3e8c3 |
| command | sdlc close --issue 362 |
| reviewer | claude |
| timestamp | 2026-10-06T20:07:54-07:00 |
| verdict | unknown |

## Review

The background test run has finished, and it changes nothing in my review: still **FIX-THEN-SHIP**. All four findings stand as written.

- **The #362 code passed its tests.** The targeted runs of `couchcore` (`Peek|OperationDeclarations`) and `couchcmd` (`Peek|ParseCLI|Arity|Summary|Skill`) passed, and so did all of `scrollbackcmd`.
- **One package failed, but not because of #362.** `TestProductionArtifactReferencesAreExactlyClassified` in `cmd/internal/artifactpath` fails. I re-ran that package alone: none of the files it flags were touched by #362. They include `wrapcmd/peer_composer.go`, `peer_delivery.go`, `automatic_input.go`, `peer_runtime.go` and the `nvim/review/*` files, which arrived with earlier work. `peek.go` is correctly listed in the inventory and is not flagged.
- **Not confirmed:** whether the full `couchcore` and `couchcmd` suites pass. The background run kept only its last 20 lines, which were all from `artifactpath`, so the other packages' results were cut off. Only the targeted subsets above are proven.

Before closing, check whether the `artifactpath` test also fails on main. It probably does, since the flagged files come from earlier work. Then run the full `make -k test`.

---

## Re-review — 2026-10-06T20:21:14-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 362 — Teach Couch skill to schedule contextual work |
| repo | pair |
| issue file | workshop/issues/000362-couch-schedule-this.md |
| boundary | whole-issue close |
| milestone | — |
| window | d9cdc78ab8e0760422a51365251bcdff65cea452..59b077bbe5340abe89db38958693bf0e062d6276 |
| command | sdlc close --issue 362 |
| reviewer | claude |
| timestamp | 2026-10-06T20:21:14-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

**Verdict: SHIP.** The diff does what the narrowed #362 asked for. `couch --peek repo:N` is a read-only look at another slot, and the Couch skill now teaches the scheduling evidence ladder. Peek gets its data from code that already existed: the scrollback replay, now pulled out as `RenderLines`; the switcher's `OSSwitchContextResolver`; the existing `resolveOperationThread` lookup; and the same retention lease that `pair scrollback render` takes. Plan revision (c) records honestly where the code differs from the planned Core concepts table: `PeekSlot` became `PeekThread`, and `RenderOwnedLines` and the `SlotTerminal` seam were added. I checked each row of that revision against the code and they all match. The issue Log records live exercises for all four Done-when bullets. The new tests pass: `-run 'Peek|RenderLines|RenderOwned|ParseCLI|OperationArity|OperationDeclarations'` is green in couchcore, couchcmd and scrollbackcmd. When I ran the whole of `artifactpath`, its inventory test failed. It names files outside this diff (`wrapcmd/peer_*.go`, `nvim/review/*`) and not `peek.go`, so the failure was already there before this change. Nothing blocks SHIP; the findings below are all Minor.

One process note: I accidentally ran `git checkout 59b077bb`, which detached HEAD in this worktree. I restored it straight away (`git checkout 000362-couch-schedule-this`, same commit, and the untracked file was untouched). Nothing was lost.

1. **Strengths**
   - `scrollbackcmd.go`: `render` became `replay` plus file writing. `TestRenderLinesMatchesTheRenderedFile` proves the in-memory lines match the written file byte for byte. ARCH-DRY: there is only one replay.
   - `peek.go:41-62`: when a source can't be read, peek names it in `Unavailable` with the reason instead of returning an empty answer. If the resolver fails, its partial result (the agent) is still used, so the terminal read can still go ahead. `TestPeekNamesEveryUnreadableSource` covers each failure branch.
   - `retention.go:RenderOwnedLines`: it finds the capture through `SelectedOwner` and the scoped directory, which is how the lease finds it. So peek and the lease agree on which file they mean, and the test reads real temporary files rather than mocks.
   - `cli.go` `--peek` parsing refuses repeated, unknown and layout flags. The rejection table in `cli_test.go` covers each of these.
   - `SKILL.md`: the evidence ladder separates accepted, submitted, claimed, progressing, complete and landed. It sends work-state questions to `sdlc issue show` and `sdlc help recovery` instead of restating SDLC's rules. It also corrects the receipt-`Detail` behavior found in live exercise 2.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `ops.go:443-447`: the `peek` declaration says the long-running console must not dispatch it, because each call parks a goroutine. Nothing enforces that. The console's dispatcher at `run.go:747-750` wires the same `DirectStoreExecutor`, which would run `peek` if the console or advisor ever named it. A declared field the dispatcher checks, or a console-side refusal plus a test, would make this a real rule rather than a comment (ARCH-CONSTRAINTS).
   - Done-when bullet 2 and live exercise 2: the exercise stopped once the message expired. Revision (b) said it would also show the other half — clear the draft so the message submits, or resend after the expiry — and the Log doesn't record either. The resolution itself (expired, so a resend is safe) is shown, but the retry is described rather than demonstrated.
   - The `run.go:507` branch (`parsed["json"] == "true"` → JSON encode) has no test through `RunWithRuntime`. `TestPeekJSONIsTheResult` only checks `json.Marshal` of the struct. The implementor already lists this as left; it is cheap to add next to the existing run tests.
   - The dispatcher's refusal of a non-positive or non-numeric `--lines` (`operationdispatch.go:159-165`) is untested. This was also already noted.

5. **Test coverage**
   - Pure pieces are tested without IO: `peekTail`, including that it doesn't alias its input, and `renderPeek`'s exact output.
   - `PeekThread` is tested through injected seams (`SwitchContext`, `SlotTerminal`) against a real thread store, and `TestPeekIsReadOnly` checks the store doesn't change.
   - `RenderOwnedLines` is tested with real files under `repos/<scope>`; a missing capture is an error.
   - The gap is the CLI → dispatch → JSON path from start to finish, which only the live exercise covers.

6. **Architecture, principle by principle**
   - **ARCH-DRY: pass.** One renderer, one resolver, one way of resolving a thread.
   - **ARCH-PURE: pass.** The IO sits behind `SlotTerminalReader` and `SwitchContextResolver`. `RenderLines` is correctly reclassified as IO in revision (c).
   - **ARCH-PURPOSE: pass.** All four Done-when bullets are addressed and the skill derives from SDLC's contracts.
   - **ARCH-MOCK: pass.** The seams are shared by production and tests, and the renderer test uses real capture files.
   - **ARCH-CONSTRAINTS: pass with the console note above.** Latency was measured (85 ms to render, about 2 s to resolve `repo:N`), and the follow-up is scoped sensibly.
   - **ARCH-SECURE: pass.** Peek reads only the same user's local Pair data. Transcripts are returned as paths and never parsed. Nothing touches credentials.
   - **ARCH-ORDER: pass.** Peek keeps no state between calls; it is a one-shot read, under a lease, of a file still being written.
   - **ARCH-FUNERAL: pass.** Peek creates nothing durable except the existing lease, which `defer lease.Close()` releases.
   - For upcoming work: the `repo:N` resolution cost (resolved twice per CLI call, `Slots.Discover` not cached) should become its own issue, as revision (a) says.

7. **Plan revisions recommended:** none. Revision (c) already matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: declared-invariant-unenforced
    title: |
      peek's "console must not dispatch it" rule is a comment; the console dispatcher would run it
    detail: |
      ops.go peek declaration vs run.go:747-750 console SetOperationDispatcher wiring DirectStoreExecutor; enforce via a declared field or console refusal plus a test (ARCH-CONSTRAINTS).
  - id: new
    severity: Minor
    family: done-when-evidence-partial
    title: |
      Exercise 2 ends at expiry; the planned clear-then-submit or resend-after-expiry outcome is not logged
    detail: |
      Revision (b) planned to show either a submit after clearing the draft or a safe resend after expiry; the Log stops at the expiry.
  - id: new
    severity: Minor
    family: cli-path-untested
    title: |
      The generic --json branch in runTypedOperationWithConsole and the --lines refusal have no run-path tests
    detail: |
      TestPeekJSONIsTheResult only marshals the struct; add a RunWithRuntime --peek --json test and a non-positive --lines dispatch test.
```
