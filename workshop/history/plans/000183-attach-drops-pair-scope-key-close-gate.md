---
gate: boundary-review
issue: 183
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-10T18:56:26-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: optionsFromCLI is extracted from an untested RunCLI; its refusal path stays unpinned
          detail: |-
            git grep RunCLI over cmd/internal/titlepoller/*_test.go returns nothing, so
            "Behaviour is unchanged" rests on reading alone, and the len(args) < 2 path
            makes the poller exit 0 without starting — the same silent failure class as
            pair#183. Add one assertion that optionsFromCLI returns ok == false on a
            short argv; no case list needed.
            (carried from plan-quality PQ-1, deferred to the boundary review)
          family: silent-degradation-untested
          round: 1
        - id: BR-2
          severity: Minor
          title: Plan does not say why the poller's contract travels by env when its sibling sidecar travels by argv
          detail: |-
            SpawnSessionWatcher hands scopeKey positionally on argv via
            sessionwatch.CommandArgs (osruntime.go:348); SessionEnv instead renders
            KEY=value for the child. The env choice is defensible (contextcmd reads
            os.Getenv, and last-duplicate-wins lets the contract override a stale
            inherited PAIR_SCOPE_KEY), but the rationale is unwritten next to the
            divergent precedent. One sentence in Core concepts.
            (carried from plan-quality PQ-2, deferred to the boundary review)
          family: sidecar-handoff-shape
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-10T18:56:26-07:00"
      agent: claude
      findings:
        - id: BR-3
          severity: Important
          title: Attach's unresolvable-scope branch renders an empty PAIR_SCOPE_KEY that overrides a correct inherited key, and the comment defending it is circular
          detail: |-
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
          family: empty-override-beats-inherited
          round: 2
        - id: BR-4
          severity: Minor
          title: The "exec keeps the last duplicate key" guarantee is load-bearing but has no conformance check
          detail: |-
            osruntime.go:378-381 and runcli.go:63-68 both rest on it, and the fake's overlay
            (createflow_test.go:243-249) encodes it, yet nothing execs a real child with a
            duplicate key. It holds today only via Go's os/exec dedupEnv (verified empirically
            on darwin during this review). A small test spawning through spawnDetached would
            pin it at the seam (ARCH-MOCK).
          family: unverified-dependency-assumption
          round: 2
        - id: BR-5
          severity: Minor
          title: The new env-name constants are the poller contract's single spelling, not the repo's
          detail: |-
            opener/runcli.go:30, reviewcmd/runcli.go:23,46,87, sessioninventory/runcli.go:27,50,
            slugcmd.go:133 and wrapcmd/wrap.go:2249,2278 still use the "PAIR_SCOPE_KEY" string
            literal, so a rename of the variable would still break them silently (ARCH-DRY).
            Scoped claim is fine; the plan's ARCH note reads wider than the code delivers.
          family: env-name-literals
          round: 2
        - id: BR-6
          severity: Minor
          title: The one production consumer of the new ExitNoScopeKey distinction discards it
          detail: |-
            titlepoller/runtime.go:105 passes io.Discard, so in production a missing scope key
            is still silent; only a hand-run `pair context` shows the reason, which is how #183
            was diagnosed in the first place. Accepted by the issue's Done-when (doctor surface
            optional) — worth carrying into the `pair doctor` check it names.
          family: silent-degradation-untested
          round: 2
        - id: BR-7
          severity: Minor
          title: Rewritten atlas paragraph still names cmd/pair-title, and a new comment cites the wrong line
          detail: |-
            atlas/architecture.md:311 opens with `cmd/pair-title`, a directory that no longer
            exists (#104 M2 folded it into `pair title`); the line was rewritten in this window,
            so the stale clause was re-committed. lifecycle.go:71 cites createflow.go:386 where
            the ResolveRepoScope call is at :387.
          family: stale-doc-reference
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-10T19:17:00-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestOptionsFromCLIRefusesAnArgvWithoutTagAndAgent; verified red on revert.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Rationale now in SessionEnv's doc comment, citing sessionwatch.CommandArgs.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Three-step order, three subtests; each reverted independently and went red.
          round: 3
        - id: BR-4
          disposition: addressed
          note: childEnviron extracted and pinned against a real /bin/sh child; red on reorder.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Plan Revisions narrows the ARCH-DRY claim to the poller contract; five literal sites named.
          round: 3
        - id: BR-6
          disposition: addressed
          note: Accepted per Done-when; the degraded path it covers is now tested via BR-3.
          round: 3
        - id: BR-7
          disposition: addressed
          note: Atlas says `pair title`; the comment cites runCreate by symbol, not a line.
          round: 3
      findings:
        - id: BR-8
          severity: Minor
          title: Attach's couch-fallback validates the key in a compound condition no fixture enters
          detail: |-
            This is the 3rd finding in family silent-degradation-untested (BR-1, BR-6 preceded it).
            Do NOT just add a case for this guard. The rule that covers all three: enumerate branch
            CONDITIONS, not fallback STEPS -- a compound condition is two branches, and the arm
            nobody runs is where an unverified claim survives. Measured: deleting
            "and ValidateRepoScopeKey(env.CouchThreadScope) == nil" from lifecycle.go:83 leaves the
            entire ./cmd/internal/launcher suite green. The enumeration the rule implies here is
            ResolveRepoScope ok/fails, couch tag matches/does not, couch key validates/does not --
            five of six cells are pinned by TestTitlePollerScopeWhenAttachCannotResolveARoot; write
            the sixth into that same subtest table rather than patching the guard alone.
          family: silent-degradation-untested
          round: 3
        - id: BR-9
          severity: Minor
          title: SessionEnv's scope key is authoritative but its data dir is still inherited, so the halves can name different repos
          detail: |-
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
          family: contract-fields-one-authority
          round: 3
      blocked: false
---

# Gate ledger — pair#183 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-10T18:56:26-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `silent-degradation-untested` optionsFromCLI is extracted from an untested RunCLI; its refusal path stays unpinned
  git grep RunCLI over cmd/internal/titlepoller/*_test.go returns nothing, so
  "Behaviour is unchanged" rests on reading alone, and the len(args) < 2 path
  makes the poller exit 0 without starting — the same silent failure class as
  pair#183. Add one assertion that optionsFromCLI returns ok == false on a
  short argv; no case list needed.
  (carried from plan-quality PQ-1, deferred to the boundary review)
- **BR-2** [Minor] `sidecar-handoff-shape` Plan does not say why the poller's contract travels by env when its sibling sidecar travels by argv
  SpawnSessionWatcher hands scopeKey positionally on argv via
  sessionwatch.CommandArgs (osruntime.go:348); SessionEnv instead renders
  KEY=value for the child. The env choice is defensible (contextcmd reads
  os.Getenv, and last-duplicate-wins lets the contract override a stale
  inherited PAIR_SCOPE_KEY), but the rationale is unwritten next to the
  divergent precedent. One sentence in Core concepts.
  (carried from plan-quality PQ-2, deferred to the boundary review)

## Round 2 — 2026-09-10T18:56:26-07:00 (claude) — BLOCKED

### Raised

- **BR-3** [Important] `empty-override-beats-inherited` Attach's unresolvable-scope branch renders an empty PAIR_SCOPE_KEY that overrides a correct inherited key, and the comment defending it is circular
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
- **BR-4** [Minor] `unverified-dependency-assumption` The "exec keeps the last duplicate key" guarantee is load-bearing but has no conformance check
  osruntime.go:378-381 and runcli.go:63-68 both rest on it, and the fake's overlay
  (createflow_test.go:243-249) encodes it, yet nothing execs a real child with a
  duplicate key. It holds today only via Go's os/exec dedupEnv (verified empirically
  on darwin during this review). A small test spawning through spawnDetached would
  pin it at the seam (ARCH-MOCK).
- **BR-5** [Minor] `env-name-literals` The new env-name constants are the poller contract's single spelling, not the repo's
  opener/runcli.go:30, reviewcmd/runcli.go:23,46,87, sessioninventory/runcli.go:27,50,
  slugcmd.go:133 and wrapcmd/wrap.go:2249,2278 still use the "PAIR_SCOPE_KEY" string
  literal, so a rename of the variable would still break them silently (ARCH-DRY).
  Scoped claim is fine; the plan's ARCH note reads wider than the code delivers.
- **BR-6** [Minor] `silent-degradation-untested` The one production consumer of the new ExitNoScopeKey distinction discards it
  titlepoller/runtime.go:105 passes io.Discard, so in production a missing scope key
  is still silent; only a hand-run `pair context` shows the reason, which is how #183
  was diagnosed in the first place. Accepted by the issue's Done-when (doctor surface
  optional) — worth carrying into the `pair doctor` check it names.
- **BR-7** [Minor] `stale-doc-reference` Rewritten atlas paragraph still names cmd/pair-title, and a new comment cites the wrong line
  atlas/architecture.md:311 opens with `cmd/pair-title`, a directory that no longer
  exists (#104 M2 folded it into `pair title`); the line was rewritten in this window,
  so the stale clause was re-committed. lifecycle.go:71 cites createflow.go:386 where
  the ResolveRepoScope call is at :387.

## Round 3 — 2026-09-10T19:17:00-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TestOptionsFromCLIRefusesAnArgvWithoutTagAndAgent; verified red on revert.
- BR-2 — addressed — Rationale now in SessionEnv's doc comment, citing sessionwatch.CommandArgs.
- BR-3 — addressed — Three-step order, three subtests; each reverted independently and went red.
- BR-4 — addressed — childEnviron extracted and pinned against a real /bin/sh child; red on reorder.
- BR-5 — addressed — Plan Revisions narrows the ARCH-DRY claim to the poller contract; five literal sites named.
- BR-6 — addressed — Accepted per Done-when; the degraded path it covers is now tested via BR-3.
- BR-7 — addressed — Atlas says `pair title`; the comment cites runCreate by symbol, not a line.

### Raised

- **BR-8** [Minor] `silent-degradation-untested` Attach's couch-fallback validates the key in a compound condition no fixture enters
  This is the 3rd finding in family silent-degradation-untested (BR-1, BR-6 preceded it).
  Do NOT just add a case for this guard. The rule that covers all three: enumerate branch
  CONDITIONS, not fallback STEPS -- a compound condition is two branches, and the arm
  nobody runs is where an unverified claim survives. Measured: deleting
  "and ValidateRepoScopeKey(env.CouchThreadScope) == nil" from lifecycle.go:83 leaves the
  entire ./cmd/internal/launcher suite green. The enumeration the rule implies here is
  ResolveRepoScope ok/fails, couch tag matches/does not, couch key validates/does not --
  five of six cells are pinned by TestTitlePollerScopeWhenAttachCannotResolveARoot; write
  the sixth into that same subtest table rather than patching the guard alone.
- **BR-9** [Minor] `contract-fields-one-authority` SessionEnv's scope key is authoritative but its data dir is still inherited, so the halves can name different repos
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

## Open findings

- **BR-8** [Minor] `silent-degradation-untested` Attach's couch-fallback validates the key in a compound condition no fixture enters
- **BR-9** [Minor] `contract-fields-one-authority` SessionEnv's scope key is authoritative but its data dir is still inherited, so the halves can name different repos
