# Boundary Review — pair#410 (milestone M1)

| field | value |
|-------|-------|
| issue | 410 — Grok harness integration for pair and couch |
| repo | pair |
| issue file | workshop/issues/000410-grok-harness-integration-for-pair-and-couch.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | fcce4b21c9cbfc3c6203df98259d34c1d4c5f191..0f0597a638816c492d6f7a041191678e21bb4821 |
| command | sdlc milestone-close --issue 410 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-08T00:24:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Summary:** M1 is ready to ship after one cheap docs fix. It delivers the re-cut boundary: grok joins the registry; the resume and session-id forms, the fresh-launch table and the shared inline-mode flag land; the composer recognizer, overlay markers and orientation glyph are built from the live captures; and the full session side lands (scanner, event normalizer, every dispatch site, watcher and ledger membership). `sessionInventoryKnownGaps` is empty, and the parity test enforces that. The design reuses existing seams and fixes a real codex bug: codex's `--no-alt-screen` used to land after `--`. Nothing Critical turned up. The one Important item is that README still lists five agents while `pair grok` now works. The rest are edge cases in how args are parsed.

**Tests I ran:** the `sessioninventory`, `resumeform`, `sessionledger` and `sessionwatch` packages all pass. The `launcher` and `wrapcmd` failures I saw are environment noise, not this diff: "operation not permitted" from PTY-spawning tests in the sandbox, and the `PAIR_DATA_DIR` scope conflict. The issue log records the same thing.

### 1. Strengths
- **One inline-mode table for every harness** (`agentargs.go:204-241`): `inlineModes` plus `insertBeforeDoubleDash` replaces `codexAltScreenArgs`, so the inline flag, the minted id and the resume token all go through one rule for placing tokens before `--`. `TestPairInsertedTokensPrecedeDoubleDash` checks this for every agent.
- **Scanner identity is strict** (`scan_grok.go:131-152`): every record's `sessionId` must match the session id in the path. Timestamps outside the 2000–9999 window dispute the record instead of inventing a time. Junk paths (subagents, non-UUID, a relative cwd) are refused, and tests pin each case.
- **The normalizer's tests are built from its own kind tables** (`event_grok_internal_test.go`): any kind added to `grokMappedKinds` or `grokIgnoredKinds` is tested automatically. Unknown kinds and unknown methods stay flagged as possible format drift.
- **Shared code instead of copies:** `promptGlyphAuthorities` (`orientation.go:224`) replaces the per-agent if-chain, and `detectRawCarryOverlay` (`wrap.go:933`) is shared by qoder and grok. The claude/qoder special cases for optional-value flags became data (`optionalValue`).
- **The recognizer generalizes cleanly:** `ruleCol` and `sideGlyph` extend `ruledBoxComposerSpec` without changing the existing specs, whose zero values keep column 0 and no side glyph.

### 2. Critical findings
None.

### 3. Important findings
- **README update appears missing for the grok harness** (`README.md:44, 304, 354, 945, 1137`). The agent rosters, the `pair <agent>` usage line and the resume-spelling paragraph still list five agents. `PAIR_GROK_ALT_SCREEN` (`agentargs.go:209`) is a new user-facing env var and appears nowhere in the docs. The plan puts the docs sweep in M2 Task 12, but the usable surface ships at this boundary.
  - **Fix:** add `grok` to the rosters now (a few one-line edits), or record why it's deferred in a `## Revisions` entry.

### 4. Minor findings
- **A glued `-s<uuid>` is not recognized** (`resumeform.go:118-125`, `:184`). `HasSessionID` and `Strip` only handle the space and `=` forms. `pair grok -s<uuid>` would still get a minted `--session-id`, which probably makes grok's argument parser reject the launch, and the glued token would survive into saved args. The fresh-launch validator does refuse it, through the cluster-letter check.
- **`inlineModeArgs` strips user text after `--`** (`agentargs.go:228`): `stripValuelessFlag` removes `--no-alt-screen` anywhere, including from the prompt after `--`. That clashes with the new rule that the inserted flag goes before `--`. The behavior predates this diff, but this diff made the function `--`-aware.
- **The property test is weaker than the plan says.** `TestPairInsertedTokensPrecedeDoubleDash` composes the helpers by hand instead of running `runCreate`, and it never checks the plan's "no flag appears twice" invariant. The createflow tests do cover the real composition for grok and codex.
- **Provider contract naming drift:** the plan says `ProviderGrokACPV1 = "grok-acp-v1"`; the code has `ProviderGrokACPJSONLV1 = "grok-acp-jsonl-v1"`. The code name matches the sibling contracts, so the plan text is what's stale.
- `detectGrokOverlayText` appears to be used only by tests (fine; it mirrors `detectQoderOverlayText`).

### 5. Test coverage notes
- **Scanner:** the fixture is a real session (sanitized) with a stub session and sibling files, and the test asserts zero diagnostics. Delta validation covers a foreign session id, malformed records, a missing session id and timestamp bounds. The issue log reports a live inventory of 12 real roots with zero grok diagnostics.
- **Launcher:** the grok mint retries on a collision, scoped to the agent (`TestRunLaunchForcedCreateGrokMintProbesGrokSessions`). Grok resume composition and saved-args stripping are pinned too.
- **TTY:** the captured fixtures `composer.raw`, `overlay.raw` and `selection.raw` are registered in the fixture, overlay and picker tests.
- **Gap:** no test covers the glued short form `-s<id>` / `-c` inside a cluster for `Strip` or `HasSessionID`.

### 6. Architectural notes
- **ARCH-DRY: pass.** Inline mode, overlay carry and glyph authority are each consolidated. The per-agent `switch` lists in `sessionwatch.SupportsAgent` and `sessionledger.isSupportedAgent` are still restated by hand. That predates this diff, and the parity test guards it, but they could derive from one inventory later.
- **ARCH-PURE: pass.** The normalizer, the path fact, the delta validator and the arg helpers are all pure. Only `scanGrokFile` reads files, through `Runtime`.
- **ARCH-PURPOSE: pass.** The re-cut brings the session side into this boundary rather than leaving a known gap, which is the purpose being delivered rather than deferred.
- **ARCH-MOCK: pass.** The scanner tests run on `sessioninventorytest.NewFakeRuntime`, and the TTY live conformance test is gated on `PAIR_LIVE_HARNESS`. M2's `runGrok` needs its own live conformance check, as planned.
- **ARCH-CONSTRAINTS: pass.** The record budget (`unlimitedRecordSize`) is the same as for the sibling transcripts, and reads are incremental from the tail.
- **ARCH-SECURE: pass.** Untrusted transcript records are parsed into typed envelopes, and failures dispute the record with a visible diagnostic rather than fabricating a value. The url-decoded cwd is only validated, never used as a filesystem path.
- **ARCH-ORDER: pass.** The normalizer holds no state between records, because grok writes a whole prompt as one `user_message_chunk` (measured live), so no chunk joining is needed.
- **ARCH-FUNERAL: pass.** M1 creates no new durable family; the grok rows ride existing retention. The slug-residue cleanup is M2's problem, and the plan already covers it with confinement rules.

### 7. Plan revision recommendations
- A `## Revisions` entry saying Task 6's "Couch interim check" (`ledgerRejectsAgent` reads grok as malformed) became moot after the re-cut, since ledger membership landed in M1.
- Correct `ProviderGrokACPV1` / `"grok-acp-v1"` to `ProviderGrokACPJSONLV1` / `"grok-acp-jsonl-v1"` in the Integration points table and in Task 7.
- If README stays deferred, note that M2 Task 12 owns the README roster and `PAIR_GROK_ALT_SCREEN`.

```findings
findings:
  - id: new
    severity: Important
    family: docs-lag-new-surface
    title: |
      README rosters and env docs omit grok and PAIR_GROK_ALT_SCREEN though grok is usable at M1
    detail: |
      README.md:44,304,354,945,1137 still list five agents; the new per-harness opt-out env is undocumented. Add the roster entries now (cheap) or record the M2 deferral in Revisions.
  - id: new
    severity: Minor
    family: resumeform-spelling-coverage
    title: |
      Glued short form -s<uuid> is not seen by HasSessionID or Strip
    detail: |
      pair grok -s<uuid> would also get a minted --session-id (likely a duplicate-argument error from grok) and the glued token would persist into saved args; the fresh-launch validator does refuse it.
  - id: new
    severity: Minor
    family: double-dash-boundary
    title: |
      inlineModeArgs strips --no-alt-screen from user prompt text after --
    detail: |
      stripValuelessFlag ignores the -- boundary that insertBeforeDoubleDash now honors; the strip should stop at the first --.
  - id: new
    severity: Minor
    family: test-restates-composition
    title: |
      TestPairInsertedTokensPrecedeDoubleDash composes helpers by hand and skips the no-duplicate invariant
    detail: |
      The plan promised a no-flag-twice assertion across fresh/resume/restart; the test checks only the tail after --.
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Plan names ProviderGrokACPV1 grok-acp-v1; code ships ProviderGrokACPJSONLV1 grok-acp-jsonl-v1
```

---

## Re-review — 2026-10-08T00:32:11-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 410 — Grok harness integration for pair and couch |
| repo | pair |
| issue file | workshop/issues/000410-grok-harness-integration-for-pair-and-couch.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | fcce4b21c9cbfc3c6203df98259d34c1d4c5f191..6ecf7d27870c81bdeb47a6952c6b003c7468af8b |
| command | sdlc milestone-close --issue 410 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-08T00:32:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

**Verdict: fix then ship.** One new Minor finding; all five round-1 findings are resolved. The round-1 fix commit `6ecf7d27` adds Grok to every README roster. It also documents `PAIR_CODEX_ALT_SCREEN`/`PAIR_GROK_ALT_SCREEN`, which had never been documented, even for codex. It stops every argv *editor* at the first `--`, and teaches the session-id check to recognise glued `-s<uuid>`. The new tests target the actual mistakes, and I checked that the strip still removes the bare space-form `--session-id <id>` / `-s <id>` value correctly. The one gap: the round-1 rule stops argv *editors* at `--`, but one argv *reader* in the launcher still looks past it. That is the second finding in family `double-dash-boundary`, so the fix should be the rule, not this one site. It is Minor and does not block the gate.

**1. Strengths**
- `resumeform.sessionIDToken` (`resumeform.go:128`) is now the single source for session-id spellings. `HasSessionID`, `Strip` and `ContextSelector` all read it, and the separate `sessionIDInline` helper is gone (ARCH-DRY).
- `TestDoubleDashTailIsNeverABinding` builds its cases from `resumeform.Forms()`. It covers every spelling of every agent rather than a hand-picked list, so a new agent or spelling is covered automatically.
- The live-conformance agent lists in `provider_live_fake_test.go:25` and `conformance_live_test.go:21` now come from `SupportedAgents()` instead of being typed out by hand. That fixes the class (lists drifting from the registry), not just the one site.
- I probed `persistedConfigArgs` with `--session-id U` and `-s U` for grok, and `--session-id U` for claude. All three strip cleanly with no stray value left behind.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- **`hasFlag` reads past `--` (second finding in family `double-dash-boundary`).** `shouldMintSessionID` (`agentargs.go:259-260`) checks `hasFlag(agentExtra, "--session-id")` and `hasFlag(agentExtra, "--fork-session")`, and `hasFlag` (`agentargs.go:17`) scans the whole argv, prompt text included.
  - Measured: `shouldMintSessionID("claude", "", ["--", "--fork-session"])` returns `false`. Prompt text therefore stops pair minting a session id.
  - The rule: every argv reader *or* editor looks only at the flag region before the first `--`. Today that boundary is written five separate ways: `resumeform.beforeDoubleDash`, the inline checks in `Strip`, `stripValuelessFlag` and `stripFlagAllForms`, and `slices.Index` in `insertBeforeDoubleDash`.
  - Fix: make one exported helper the single source, e.g. `resumeform.FlagRegion(args) (flags, tail)`. Route every helper, including `hasFlag`, through it, and add `hasFlag`/`shouldMintSessionID` to `TestStripHelpersStopAtDoubleDash`.
- **The composition test still assembles the helpers by hand.** `TestPairInsertedTokensPrecedeDoubleDash` builds the argv itself instead of driving the `createflow.go:620-658` sequence. It also adds `--session-id` on resume, which production never does. The new no-duplicates assertion is real: a mutation that removes the strip inside `inlineModeArgs` turns it red. So I record BR-4 as addressed and leave this as a note, not a new finding.

**5. Test coverage notes**
- `resumeform`, `sessioninventory`, `sessionledger` and `sessionwatch` pass in the sandbox. `wrapcmd` passes unsandboxed; inside the sandbox its PTY and `/tmp` tests fail with "operation not permitted", which is the sandbox, as already known.
- `launcher` showed five failures in my shell. All of them are `PAIR_DATA_DIR … conflicts with selected repository scope`, which comes from the host pair session's environment, not from grok code. The issue Log records `go test ./...` under `env -i` as passing for `launcher`.
- The `TestGluedSessionIDShortForm` and `TestStripHelpersStopAtDoubleDash` cases each go red if their fix is removed (BR-2 and BR-3).

**6. Architectural notes**
- **ARCH-DRY:** flag. The `--` boundary should have a single source (finding above).
- **ARCH-PURE:** pass. Every argv decision is a pure function, and the IO (mint, collision check) stays in `createflow`.
- **ARCH-PURPOSE:** pass, with one gap. Round 1 fixed the README class and the conformance-list class. The `--` class is fixed for editors but not readers (finding above).
- **ARCH-MOCK:** pass. The grok scanner runs against native fixtures plus the stateful live fake, and the conformance run now covers every registry agent.
- **ARCH-CONSTRAINTS:** pass. Nothing in this window touches a hot path.
- **ARCH-SECURE:** pass. Argv is user input and is parsed per agent's own table. The M2 cleanup design in the plan's Revisions derives the deletion target only from a temp dir pair just created, not from subprocess output, which is the right call.
- **ARCH-ORDER:** N/A for this window. It holds no state between events; the argv transforms are single-shot.
- **ARCH-FUNERAL:** nothing new is created in this window. The M2 temp-dir and Grok session-entry removal is designed in Revisions and should be checked at M2.

**7. Plan revision recommendations**
- Append to the M1 round-1 Revisions entry: the `--` boundary is a single shared helper that covers readers (`hasFlag`, `shouldMintSessionID`) as well as editors.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README lines 3/44/146/304/354/945/1008/1142 now list grok; PAIR_CODEX_ALT_SCREEN/PAIR_GROK_ALT_SCREEN documented under Command Usage (matches inlineModes in agentargs.go).
  - id: BR-2
    disposition: addressed
    note: |
      sessionIDToken reads glued single-letter spellings in HasSessionID/Strip/ContextSelector; TestGluedSessionIDShortForm goes red without it.
  - id: BR-3
    disposition: addressed
    note: |
      stripValuelessFlag stops at --; the opted-out inlineModeArgs case in TestStripHelpersStopAtDoubleDash goes red without it. Reader-side sibling raised as a new family finding.
  - id: BR-4
    disposition: addressed
    note: |
      No-duplicates assertion added before --; removing the strip in inlineModeArgs turns it red. The test still composes helpers by hand (noted, not re-raised).
  - id: BR-5
    disposition: addressed
    note: |
      Plan Core concepts and Task 7 now name ProviderGrokACPJSONLV1 grok-acp-jsonl-v1.
findings:
  - id: new
    severity: Minor
    family: double-dash-boundary
    title: |
      hasFlag in shouldMintSessionID reads prompt text after --, so the -- boundary has no single source
    detail: |
      2nd finding in this family. agentargs.go:17,259-260 scan the whole argv; measured shouldMintSessionID claude with args [--, --fork-session] returns false, so prompt text suppresses the mint. Rule: every argv reader or editor sees only the flag region before the first --. Fix by exporting one FlagRegion helper (today written five ways across resumeform.beforeDoubleDash, Strip, stripValuelessFlag, stripFlagAllForms, insertBeforeDoubleDash), route hasFlag through it, and extend TestStripHelpersStopAtDoubleDash to readers.
```
