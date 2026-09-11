# Boundary Review — pair#183 (whole-issue close)

| field | value |
|-------|-------|
| issue | 183 — Attach drops PAIR_SCOPE_KEY, so the context meter vanishes after reattach |
| repo | pair |
| issue file | workshop/issues/000183-attach-drops-pair-scope-key.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4b593920ef13f09da11d9fb52a5dcb2af2d32d28..53bf73e1a680e044b1dc74e1dc08dc5f37c182ce |
| command | sdlc close --issue 183 |
| reviewer | claude |
| timestamp | 2026-09-10T18:56:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have everything I need. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The diff delivers the issue's actual purpose, not a patch of the symptom: the title poller's launch contract is now one declaration (`titlepoller.SessionEnv`, positional `NewSessionEnv`), both launch paths hand it to the spawn, and the regression is measured against the contract's own names rather than a second hand-kept list. I independently re-ran all 8 mutation rows in a scratch copy of HEAD and every one dies as the plan claims — including the two that close the *class* (row 6, a new read through `EnvFrom`, fails the observed-reads test; row 7, a new `NewSessionEnv` parameter, breaks the build at exactly `createflow.go:609` and `lifecycle.go:76`). `go build ./...`, `gofmt -l cmd/` and `go vet` on launcher/couchcore/titlepoller/contextcmd/dispatcher are clean; the only suite failures in my run are the known `ptychild: operation not permitted` pty tests in `couch*`, untouched by this diff. What keeps it from SHIP is one thing, and it is the same failure mode this issue is about: the attach path's "scope won't resolve" branch now renders an explicit `PAIR_SCOPE_KEY=` that *overrides* a correct inherited value, and the comment justifying that ("no session could match regardless") asserts something the code itself causes — an untested branch defended by a claim that isn't true, which is exactly how the `lifecycle.go:42` comment hid #183.

## 1. Strengths

- **The contract is tied to reads, not to a list** (`cmd/internal/titlepoller/runcli_test.go:17`). Observing through an injected `getenv` across `contextcmd.EnvFrom` + `optionsFromCLI`, then asserting every `PAIR_`-prefixed read is supplied, is the one formulation that doesn't reproduce the bug it guards. Mutation row 6 confirms it fires on a *new* read.
- **Compile-time closure is real, not aspirational.** Positional `NewSessionEnv` with unexported fields: row 7 produced `not enough arguments in call to titlepoller.NewSessionEnv` at both call sites. That is the half of the class a test cannot cover, and it works.
- **Scope derivation on attach mirrors create for the right reason** (`lifecycle.go:72-75` vs `createflow.go:387`). I checked the alternative: `scopeKeyFromDataDir` (already in-package, used 3× in `createflow`) would derive the key from the data-dir path, which does *not* match the ledger owner key create wrote when `PAIR_DATA_DIR` is overridden. Re-resolving from the repo root is the consistent choice, and the comment saying so is correct.
- **The new status reaches the operator.** `dispatchContext` forwards stderr and `writeResult` (`cmd/pair-go/main.go:212-219`) writes it and returns the code, so `pair context` genuinely became diagnosable. I verified no shell/KDL/lua consumer of `pair context` exists, so `ExitNoScopeKey = 2` breaks nothing.
- **The fake models exec precedence rather than the implementation** (`createflow_test.go:241-250`): base env overlaid by `Environ()`, which is why row 2 (`NewSessionEnv(dataDir, "")` on create) dies even though create's ambient export still carries a good key.

## 2. Critical findings

None.

## 3. Important findings

**`cmd/internal/launcher/lifecycle.go:72-76` — the unresolvable-scope branch renders an empty key that beats a correct inherited one, justified by a claim the code makes false (`ARCH-ORDER`, `ARCH-PURPOSE`).**

Pre-fix, attach set no `PAIR_SCOPE_KEY`, so the poller inherited the parent's. Post-fix, when `ResolveRepoScope(envScopeRoot(env))` fails, `Environ()` emits `PAIR_SCOPE_KEY=` and — by the last-duplicate-wins precedence this change deliberately relies on — that empty value *wins over* an inherited correct one. `titlepoller/runtime.go:101-104` then defends discarding the diagnosis with "an empty key here means the launcher could not resolve a repo root at all, where no session could match regardless", which is circular: a match was possible via the inherited key; the contract's own empty render is what made it impossible. Reachable when `envScopeRoot` is `/` or empty (`ResolveRepoScope` rejects only `.` and `/`) — contrived for a terminal launch, but `couchcore/park_lifecycle_live_test.go:473` already passes an env with no `Cwd`, so the branch is live, and nothing tests it.

Fix sketch (cheap, pick one): in `AttachExistingSession`, fall back to `env.CouchThreadScope` when `env.CouchThreadTag == tag` (the guard `createflow.go:392` already uses for the same pair, and the value the issue's Spec notes couch already passes) before settling for `""`; add a subtest with `Cwd: "/"` pinning whichever behavior you choose; and correct the `ContextCount` comment so it states the contract's empty render is the cause rather than implying inevitability.

## 4. Minor findings

- `osruntime.go:378-381` / `runcli.go:63-68`: the "exec keeps the LAST duplicate" guarantee is load-bearing (it is PQ-2's stated reason for env over argv, and the fake's overlay encodes it) but has no conformance check — it holds only because Go's `os/exec` dedups last-wins. I verified it empirically on darwin today; a ~10-line test that spawns a real child through `spawnDetached` with a duplicate key would pin it (`ARCH-MOCK`).
- `contextcmd.go:24-27`: the new constants are the *poller contract's* single spelling, not the repo's — `opener/runcli.go:30`, `reviewcmd/runcli.go:23,46,87`, `sessioninventory/runcli.go:27,50`, `slugcmd.go:133`, `wrapcmd/wrap.go:2249,2278` still use the `"PAIR_SCOPE_KEY"` literal, so a rename still breaks them silently (`ARCH-DRY`).
- `titlepoller/runtime.go:105`: the one production consumer of the new distinction throws it away (`io.Discard`), so a recurrence is still silent in production — the operator diagnoses it the way they diagnosed #183, by running `pair context` by hand. The issue accepted this (doctor surface optional); worth carrying into the `pair doctor` check it names.
- `atlas/architecture.md:311`: the rewritten paragraph still opens with `cmd/pair-title`, which no longer exists (#104 M2 folded it into `pair title`); `lifecycle.go:71` cites `createflow.go:386` where the call is at `:387`.

## 5. Test coverage notes

Independently verified, in a `git archive` copy of HEAD (needle asserted to occur exactly once, restored from saved bytes, named `--- FAIL`/build error required — a non-zero exit alone was not accepted): row 1 attach-scope-unresolvable → `…/attach` red, `/create` green; row 2 create empty key → `/create` red; row 3 drop `EnvScopeKey` from `Environ` → observed-reads test *and* `…/attach` red, `/create` green by design; row 4 `env.Environ()`→`nil` → `TestSidecarSpawnArgvSelfExecsPair` red; row 5 `if false` → `TestRunWithoutAScopeKeyIsItsOwnStatus` red; row 6 new `getenv("PAIR_AGENT")` → observed-reads red; row 7 new parameter → build red at both call sites; row 8 dropped `Stderr` field → `TestDispatchContextWithoutAScopeKeySaysWhy` red (needs `runtimebundle/assets` present, which a bare archive lacks). Both plan-gate advisories are real code, not commit-message claims: PQ-1 is `TestOptionsFromCLIRefusesAnArgvWithoutTagAndAgent`, PQ-2 is the rationale paragraph at `runcli.go:63-68`. Untested: the `err != nil` half of the attach scope resolution (the Important above) and the one-line `OSRuntime.SpawnTitlePoller` glue.

## 6. Architecture

`ARCH-DRY` pass (one declaration; `titlePollerArgv` deleted, `git grep` clean) with the literals note above. `ARCH-PURE` pass — `SessionEnv`/`NewSessionEnv`/`Environ`/`optionsFromCLI`/`EnvFrom`/`titlePollerSpawn` are pure and tested with zero IO; the IO shell is one line. `ARCH-PURPOSE` pass — shadow-sweep: exactly two consumers (`createflow.go:609`, `lifecycle.go:76`), both derive; no hand-maintained restatement of the poller's env remains; the surviving ambient lists serve zellij panes, a different consumer, and now say so. `ARCH-MOCK` pass with the precedence-conformance note. `ARCH-CONSTRAINTS` pass — one SHA-256 added per attach, poll cadence untouched. `ARCH-SECURE` pass — the key is a hex digest of a local path, not a secret; no new untrusted parse; the new tests write only `t.TempDir` and use `t.Setenv`. `ARCH-ORDER` pass — no new cross-event state; the one ordering fact (the first-wins single-instance guard at `run.go:95-100`, which I confirmed keeps the *existing* poller) is handled by documentation plus the detach-then-reattach verification rather than by killing a live poller, which is the right call. For upcoming work: the next field added to the contract will exercise the empty-render hazard the plan review already identified — fixing the Important above is what makes that safe by default.

## 7. Plan revision recommendations

- A `## Revisions` entry scoping the ARCH-DRY note: the name constants are the single spelling *for the poller's contract* (contextcmd + titlepoller); five other packages still use `"PAIR_SCOPE_KEY"` literals, so no repo-wide env-name registry exists yet.
- If the Important is taken, a `## Revisions` entry recording the attach fallback order (resolved scope → `COUCH_THREAD_SCOPE` when the tag matches → empty) and the subtest that pins it, since Task 4's literal code in the plan no longer matches `lifecycle.go`.
- Optional: PQ-2 asked for one sentence in the plan's **Core concepts**; it landed in `runcli.go`'s doc comment and the Revisions entry instead. Fine as delivered, but Core concepts still reads as if the rationale were missing.

```findings
findings:
  - id: new
    severity: Important
    family: empty-override-beats-inherited
    title: |
      Attach's unresolvable-scope branch renders an empty PAIR_SCOPE_KEY that overrides a correct inherited key, and the comment defending it is circular
    detail: |
      lifecycle.go:72-76 falls back to scopeKey "" when ResolveRepoScope fails, and
      Environ() then emits an explicit PAIR_SCOPE_KEY= that wins over the value the
      child would have inherited (the same last-duplicate-wins precedence this change
      relies on). titlepoller/runtime.go:101-104 justifies discarding the diagnosis with
      "no session could match regardless", which is only true because of that empty
      render; pre-fix the inherited key could match. The branch has no test, and
      couchcore/park_lifecycle_live_test.go:473 already enters it via an env with no Cwd.
      Fix: fall back to env.CouchThreadScope when env.CouchThreadTag == tag (the guard
      createflow.go:392 uses), pin the degraded path with a Cwd "/" subtest, and correct
      the ContextCount comment.
  - id: new
    severity: Minor
    family: unverified-dependency-assumption
    title: |
      The "exec keeps the last duplicate key" guarantee is load-bearing but has no conformance check
    detail: |
      osruntime.go:378-381 and runcli.go:63-68 both rest on it, and the fake's overlay
      (createflow_test.go:243-249) encodes it, yet nothing execs a real child with a
      duplicate key. It holds today only via Go's os/exec dedupEnv (verified empirically
      on darwin during this review). A small test spawning through spawnDetached would
      pin it at the seam (ARCH-MOCK).
  - id: new
    severity: Minor
    family: env-name-literals
    title: |
      The new env-name constants are the poller contract's single spelling, not the repo's
    detail: |
      opener/runcli.go:30, reviewcmd/runcli.go:23,46,87, sessioninventory/runcli.go:27,50,
      slugcmd.go:133 and wrapcmd/wrap.go:2249,2278 still use the "PAIR_SCOPE_KEY" string
      literal, so a rename of the variable would still break them silently (ARCH-DRY).
      Scoped claim is fine; the plan's ARCH note reads wider than the code delivers.
  - id: new
    severity: Minor
    family: silent-degradation-untested
    title: |
      The one production consumer of the new ExitNoScopeKey distinction discards it
    detail: |
      titlepoller/runtime.go:105 passes io.Discard, so in production a missing scope key
      is still silent; only a hand-run `pair context` shows the reason, which is how #183
      was diagnosed in the first place. Accepted by the issue's Done-when (doctor surface
      optional) — worth carrying into the `pair doctor` check it names.
  - id: new
    severity: Minor
    family: stale-doc-reference
    title: |
      Rewritten atlas paragraph still names cmd/pair-title, and a new comment cites the wrong line
    detail: |
      atlas/architecture.md:311 opens with `cmd/pair-title`, a directory that no longer
      exists (#104 M2 folded it into `pair title`); the line was rewritten in this window,
      so the stale clause was re-committed. lifecycle.go:71 cites createflow.go:386 where
      the ResolveRepoScope call is at :387.
```

---

## Re-review — 2026-09-10T19:17:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 183 — Attach drops PAIR_SCOPE_KEY, so the context meter vanishes after reattach |
| repo | pair |
| issue file | workshop/issues/000183-attach-drops-pair-scope-key.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4b593920ef13f09da11d9fb52a5dcb2af2d32d28..fda117fb3b6415f477cd1f30fd7f9b2c40ed96b9 |
| command | sdlc close --issue 183 |
| reviewer | claude |
| timestamp | 2026-09-10T19:17:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All seven prior findings are genuinely disposed, and I verified each claimed fix by reverting it in the working tree and watching the named test go red — not by reading the commit messages. The core defect is fixed at the right level: the poller's environment is now one declaration (`titlepoller.SessionEnv`) that both launch paths hand to the spawn, the regression asserts against the contract's own names rather than a second hand-kept list, and BR-3's three-step authority order has one load-bearing subtest per step. Build, vet, and `go test ./cmd/...` are clean apart from the known pty-spawn-permission failures in `ptychild`/`couchcore`/`couchtty`/`hostty`/`keyscmd`, none of which touch this diff. Two Minor notes remain; neither blocks the gate.

## 1. Strengths

- **The regression is load-bearing, not decorative.** Reverting attach to `NewSessionEnv(env.DataDir, "")` turns `TestTitlePollerStartsWithItsWholeContractOnBothPaths/attach` red with the exact #183 sentence (`poller_env_test.go:25` → `the poller starts without PAIR_SCOPE_KEY`), while `/create` stays green. Measured, not assumed.
- **The contract is tied to observed reads, and both halves are pinned.** I dropped `PAIR_DATA_DIR` from `Environ()`: the launcher suite stayed green (the ambient `SetEnv` masks it in the fake) but `TestSessionEnvSuppliesEveryPairVariableThePollerReads` caught it at `runcli_test.go:38`. The two tests together close the hole either one leaves. The positional `NewSessionEnv` (`titlepoller/runcli.go`) makes a future field a compile error at both call sites rather than a silent empty override.
- **BR-3's fallback chain is three real branches.** Removing the couch arm fails only `couch's record of this thread`; making `Environ()` omit an empty key fails the other two (`poller_env_test.go:68-82`) with distinct messages. The rewritten justification argues from outside the branch, which is the point of the lesson it generated.
- **BR-4 is a genuine live-conformance check (ARCH-MOCK).** `TestChildEnvironLetsTheContractBeatAnInheritedKey` (`osruntime_test.go:508`) execs a real `/bin/sh`; reordering `childEnviron` to put the contract first turns it red. The fake's overlay in `createflow_test.go:243-249` now has a real counterpart for the `os/exec` dedup rule the whole design rests on.
- **`workshop/lessons.md` closes the loop** with three transferable rules, including the circular-justification one that produced BR-3 — written as rules, not as a changelog.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

- `cmd/internal/launcher/lifecycle.go:83` — the `ValidateRepoScopeKey` half of the couch fallback is a second branch inside a "one subtest per step" enumeration, and no fixture enters it (measured: dropping it leaves the whole launcher suite green).
- `cmd/internal/launcher/lifecycle.go:86` — the contract's scope key is authoritative but its data dir is still inherited, so the two halves can name different repos.
- (Not raised as findings, noted for the record: `lifecycle.go:83`'s comment says "the same guard `runCreate`'s `couchOwned` uses" where `couchOwned` is a conjunction including a scope equality attach cannot perform — the tag half only. And `RunCLI`'s `stderr` parameter is now unused, as it was before the extraction.)

## 5. Test coverage notes

Every claimed fix has a test that fails without it — I checked all five by revert, not by reading. The one gap is the compound condition above. `optionsFromCLI`'s `adapt.DataDir()` fallback reads the real process environment past the injected `getenv`; it is unreached on the green path, but when I deliberately broke the contract it surfaced the developer's real `~/.local/share/pair/repos/<key>` in the failure message — a reminder that the injected seam has one documented hole.

## 6. Architectural notes

ARCH-DRY **pass** — shadow-sweep clean: `SpawnTitlePoller` has exactly two production call sites (`createflow.go:609`, `lifecycle.go:86`), both deriving from `NewSessionEnv`; no third spawner of `pair title` exists. ARCH-PURE **pass** — `SessionEnv`/`Environ`/`EnvFrom`/`optionsFromCLI` are pure, IO stays in `OSRuntime.SpawnTitlePoller`/`spawnDetached`; the `adapt.DataDir()` read above is the only leak and is documented in the type's doc comment. ARCH-PURPOSE **pass** — all four Done-when bullets land, including the "one declaration both paths consume" that was the actual point rather than the cheap `SetEnv` one-liner. ARCH-MOCK **pass** (see BR-4). ARCH-CONSTRAINTS **pass** — the empty-scope branch short-circuits before the inventory listing, so the 60s poll path got strictly cheaper; no new concurrency. ARCH-SECURE **pass** — `COUCH_THREAD_SCOPE` is cross-process input and is now parsed through `ValidateRepoScopeKey` before becoming a query component; correct shape, untested arm. ARCH-ORDER **pass** — the one ordering hazard (first-wins single-instance guard letting a pre-fix scope-less poller outlive the upgrade) is enumerated in the plan's Transition note and the atlas, and the real-stack verification was chosen to match it (detach→reattach, not relaunch).

Forward: `pair doctor` is now named by two issues (#183, #226) and exists in neither — the `ExitNoScopeKey` reason has no production reader until it lands. Worth its own issue rather than a third reference.

## 7. Plan revision recommendations

- If N2 is accepted: a `## Revisions` entry recording that `SessionEnv`'s two fields derive from different authorities, and the decision taken (assert agreement, derive the key via the existing `scopeKeyFromDataDir`, or accept the divergence explicitly). The current doc comment claims protection for the pane-nesting case that only the scope key actually gets.
- Core concepts, Pure entities: the `optionsFromCLI` row reads unqualified `new (extracted)`; the `adapt.DataDir()` ambient read is described in a bullet and in the code but not in the table row a reader greps.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestOptionsFromCLIRefusesAnArgvWithoutTagAndAgent; verified red on revert.
  - id: BR-2
    disposition: addressed
    note: |
      Rationale now in SessionEnv's doc comment, citing sessionwatch.CommandArgs.
  - id: BR-3
    disposition: addressed
    note: |
      Three-step order, three subtests; each reverted independently and went red.
  - id: BR-4
    disposition: addressed
    note: |
      childEnviron extracted and pinned against a real /bin/sh child; red on reorder.
  - id: BR-5
    disposition: addressed
    note: |
      Plan Revisions narrows the ARCH-DRY claim to the poller contract; five literal sites named.
  - id: BR-6
    disposition: addressed
    note: |
      Accepted per Done-when; the degraded path it covers is now tested via BR-3.
  - id: BR-7
    disposition: addressed
    note: |
      Atlas says `pair title`; the comment cites runCreate by symbol, not a line.
findings:
  - id: new
    severity: Minor
    family: silent-degradation-untested
    title: |
      Attach's couch-fallback validates the key in a compound condition no fixture enters
    detail: |
      This is the 3rd finding in family silent-degradation-untested (BR-1, BR-6 preceded it).
      Do NOT just add a case for this guard. The rule that covers all three: enumerate branch
      CONDITIONS, not fallback STEPS -- a compound condition is two branches, and the arm
      nobody runs is where an unverified claim survives. Measured: deleting
      "and ValidateRepoScopeKey(env.CouchThreadScope) == nil" from lifecycle.go:83 leaves the
      entire ./cmd/internal/launcher suite green. The enumeration the rule implies here is
      ResolveRepoScope ok/fails, couch tag matches/does not, couch key validates/does not --
      five of six cells are pinned by TestTitlePollerScopeWhenAttachCannotResolveARoot; write
      the sixth into that same subtest table rather than patching the guard alone.
  - id: new
    severity: Minor
    family: contract-fields-one-authority
    title: |
      SessionEnv's scope key is authoritative but its data dir is still inherited, so the halves can name different repos
    detail: |
      lifecycle.go:86 pairs a freshly resolved scopeKey with env.DataDir, and
      launcher/runcli.go:89-91 lets an inherited PAIR_DATA_DIR override the derived scoped
      dir. Running `pair resume <tag>` from a Pair pane of repo A with cwd in repo B therefore
      hands the poller repo A's ledger dir and repo B's key; QuerySession matches nothing and
      the frame goes bare -- the pair#183 symptom, with the stderr reason discarded by
      io.Discard. SessionEnv's doc comment names exactly that scenario ("a Pair pane that
      `pair resume` was run inside") as one the contract defends, so the claim is wider than
      the code. Pre-fix both halves came from the same inherited pane, so the divergence is
      new here. Either derive the key from the data dir (scopeKeyFromDataDir already exists in
      the package) or assert the two agree at construction.
```
