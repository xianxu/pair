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
