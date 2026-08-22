---
gate: boundary-review
issue: 145
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-21T16:52:24-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: the one-agent-per-tree guard never consults IsLive, so a dead actor blocks its tree forever
          detail: |-
            registry.go:70 and :85 refuse on registry membership alone; Couch.IsLive
            (couch.go:94) is called only from Views/Summarize. Since `couch start`
            blocks and nothing unregisters on child exit, the normal end of a session
            leaves a dead record that permanently refuses the tree. Reproduced with
            the built binary: the refusal names a dead PID and offers "switch to it"
            (unimplemented, #146) or --same-tree (records a false co-tenancy).
          family: guard-ignores-recomputed-liveness
          round: 1
        - id: BR-2
          severity: Critical
          title: couch stop never signals its child, and forgetting the record opens the collision hazard
          detail: |-
            ops.go:84 declares "Signal an actor's child and forget it"; ops.go:94 only
            calls c.Forget, and no production code calls Handle.Signal or any kill
            path. Stopping a live actor leaves the agent running with no registry
            entry, so a subsequent `couch start` on that tree is allowed -- two agents
            on one index lock and one branch. The summary is also the machine-facing
            contract for the advisor in pair#148.
          family: declared-contract-not-implemented
          round: 1
        - id: BR-3
          severity: Critical
          title: couch show <ref> prints every tree with a registered actor, not the requested one
          detail: |-
            Summarize (couch.go:213) takes a tree filter, then unconditionally folds in
            every record in the registry (couch.go:231), so the filter can only add.
            Verified in-package (Summarize([/repo]) -> [/other /repo]) and with the
            built binary, where `couch show <pair tree>` printed output identical to
            `couch list`. Contradicts the declared summary "Show the actors on one
            tree". The existing test passes only because its fixture has one tree.
          family: filter-argument-only-adds
          round: 1
        - id: BR-4
          severity: Important
          title: Store.Load fabricates same_tree=true on every record and the next Save persists it
          detail: |-
            store.go:69 sets a.Args.SameTree = true on replay to dodge its own
            re-register refusal. Verified: save with SameTree=false, Load, Save, and
            the on-disk snapshot reads "same_tree": true. StartArgs documents this
            field as the single record of the escape hatch, so afterwards no reader can
            tell which actors actually used it. Give Registry an unchecked insert for
            Load instead.
          family: replay-mutates-persisted-record
          round: 1
        - id: BR-5
          severity: Important
          title: an unreadable registry reads as a first run, and the next Save destroys it
          detail: |-
            store.go:61 discards every ReadFile error, not just not-exist. Verified
            with registry.json at mode 000: Load returns err=nil with zero records, and
            the next spawn writes a one-actor snapshot over the old file. Distinguish
            fs.ErrNotExist from real IO/permission failures and return the latter.
          family: silent-error-swallowing
          round: 1
        - id: BR-6
          severity: Important
          title: the operation-set audit compares two views of the same source and cannot fail
          detail: |-
            run_test.go:31 asserts Dispatch()'s keys equal OperationNames(); both derive
            from Operations(), so the identity is structural. I added an undeclared
            `couch nuke` branch ahead of the table lookup in RunWithRuntime and the
            suite stayed green -- the exact hazard the comment claims it catches, and
            what Done-when 6 asks to be audited rather than spot-checked.
          family: test-cannot-fail-for-claimed-reason
          round: 1
        - id: BR-7
          severity: Important
          title: couchcmd constructs production seams inline, so start/stop/refusal have no reachable tests
          detail: |-
            run.go:87-91 builds ExecRunner/OSPathOps/ExecGit/OSProcOps directly;
            Runtime injects only env and store dir. No test exercises render's
            StartResult path or renderError's worktree-or-switch offer (Done-when 2),
            and existing CLI tests shell out to real git instead of FakeGit. ARCH-MOCK:
            production flow and test flow do not share the boundary at this layer,
            which is why the three Critical findings shipped.
          family: cli-shell-not-injectable
          round: 1
        - id: BR-8
          severity: Important
          title: the ExecRunner liveness fix is pinned only by a gate nothing runs
          detail: |-
            Restoring the full pre-fix shape (no reaper, Alive = kill -0, Wait =
            cmd.Wait) leaves `go test ./cmd/internal/couchcore/` green in 0.35s; only
            PAIR_LIVE_COUCH=1 fails. No Makefile target sets any PAIR_LIVE_* gate, and
            test-race still points at ./cmd/pair-wrap/, which no longer exists.
            Separately, swapping Alive() back to procutil.Alive while keeping the
            reaper leaves BOTH suites green, so runner.go:120's "deliberately does NOT
            consult procutil.Alive" is undefended. A default-suite test that polls
            Alive() after `sh -c 'exit 0'` without calling Wait() would pin it.
          family: fix-pinned-only-by-opt-in-test
          round: 1
        - id: BR-9
          severity: Important
          title: the agent half of the agent-supplied description was not built
          detail: |-
            Store.WriteDescription (store.go:101) has zero callers and zero tests;
            Couch.Describe prefers a sidecar nothing writes. What ships is an operator
            typing `couch describe`, landing in the naming table. The Spec is explicit
            that descriptors come from the agent with a cached fallback -- the cache
            exists with no source to cache from (ARCH-PURPOSE).
          family: deferred-purpose
          round: 1
        - id: BR-10
          severity: Important
          title: atlas/couch.md describes an actor loop that no command ever starts
          detail: |-
            atlas/couch.md:67-77 says "One goroutine per actor, holding a bounded
            mailbox" in the present tense, but NewActor (actor.go:36) has no production
            call site -- Couch.Spawn starts a child and returns. Fine as pair#147
            groundwork; wrong as a map of what exists. Move it under "Planned, not
            built" or state that the loop is unit-tested but not instantiated.
          family: docs-claim-unbuilt-behavior
          round: 1
        - id: BR-11
          severity: Important
          title: README not updated for a second installed binary
          detail: |-
            GO_BINS := pair couch means `make install` now puts a second executable on
            PATH, while README's Install section still says pair is "a single Go
            binary" and Command Usage lists only `pair ...`. One line pointing at
            atlas/couch.md clears the gate.
          family: readme-gate
          round: 1
        - id: BR-12
          severity: Important
          title: the issue ticks the operator smoke while four of its five steps are recorded unrun
          detail: |-
            The issue Plan has "[x] Operator smoke: host one real pair child"; plan
            Task 17's five checkboxes are all unchecked, and the issue Log says the
            second-shell read path, the refusal offer and the kbench-subdirectory case
            were not exercised. Two of those unrun steps are exactly where the
            dead-actor refusal and the show-leak live.
          family: checklist-ticked-beyond-evidence
          round: 1
        - id: BR-13
          severity: Minor
          title: Enqueue's collapse can silently downgrade a queued Control message
          detail: |-
            mailbox.go:35 matches on Kind alone, so a non-control message replaces a
            queued Control one of the same kind: Enqueue([stop{Control:true}],
            stop{}, 8) yields one entry with Control=false and ok=true. Contradicts
            "never drop a Control message"; unreachable today only because Actor has no
            production caller.
          family: control-message-invariant
          round: 1
        - id: BR-14
          severity: Minor
          title: CheckAvailable and RegisterWithPolicy duplicate the occupancy test verbatim
          detail: |-
            registry.go:70-82 and :85-101 are the same block; ARCH-DRY, and it means
            the liveness fix for the Critical finding has to be written twice.
          family: duplicated-guard-block
          round: 1
        - id: BR-15
          severity: Minor
          title: the bare-named couch binary contradicts the Makefile's own stated pair- prefix rule
          detail: |-
            Makefile.local:6-8 says every Go binary ships with the pair- prefix to avoid
            PATH collisions (citing the bare `scribe` that was renamed). `couch` is
            bare and the comment was not amended either way.
          family: binary-naming-convention
          round: 1
        - id: BR-16
          severity: Minor
          title: two new errors.go files wrap errors.As at a single call site each
          detail: |-
            couchcmd/errors.go and couchcore/errors.go add a one-line helper where the
            rest of the repo calls errors.As directly (wrap.go:2206, launcher tests).
          family: needless-indirection
          round: 1
        - id: BR-17
          severity: Minor
          title: dead exported API and never-populated StartArgs fields
          detail: |-
            Couch.List, Couch.Policy, Registry.Unregister, FakeRunner.Signals and
            StartArgs.AgentStack have zero non-test callers; the CLI never populates
            Stack, Issue or ExtraArgs, so the "structured start-args" record is
            effectively {Worktree, Cwd, SameTree}.
          family: unused-public-surface
          round: 1
        - id: BR-18
          severity: Minor
          title: bindArgs accepts any --flag silently
          detail: |-
            run.go:110 stores every --flag it sees, so `--same-tre` leaves the guard in
            force with no diagnostic -- unhelpful for the one loud escape hatch.
          family: silent-flag-acceptance
          round: 1
        - id: BR-19
          severity: Minor
          title: make test-race targets ./cmd/pair-wrap/, a directory that no longer exists
          detail: |-
            Makefile.local:131. This diff's own lesson prescribes running the whole tree
            under -race; pointing the target at ./cmd/... would encode it instead of
            leaving it as a habit.
          family: stale-build-target
          round: 1
        - id: BR-20
          severity: Minor
          title: trimTrailingNewline is TrimSpace, and sanitizeKey can collide two trees
          detail: |-
            strings.go:7 trims all surrounding whitespace, not a trailing newline;
            sanitizeKey maps /a/b and /a_b onto the same description sidecar file.
          family: misleading-helper-names
          round: 1
        - id: BR-21
          severity: Minor
          title: Save writes actors in Go map order, so registry.json churns between identical saves
          detail: |-
            store.go:47 uses reg.Records(), which iterates the map. Sorting by
            (worktree, id) would make the snapshot diffable.
          family: nondeterministic-snapshot
          round: 1
        - id: BR-22
          severity: Minor
          title: no locking on registry.json across couch processes
          detail: |-
            Concurrent invocations are last-writer-wins over the whole snapshot. Narrow
            today because `couch start` saves once and then blocks, but it widens as
            soon as any command saves more than once.
          family: unsynchronised-shared-state
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-22T09:51:25-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified: deleting Spawn''s PruneDead call reddens TestDeadActorDoesNotBlockItsTreeForever and TestKnownDeadIsStillPruned.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Verified: deleting Stop''s signal block reddens 2 couchcore tests and 1 couchcmd test.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Verified: making Summarize additive again reddens TestShowFilterRestrictsRatherThanAdds ([/other /repo]).'
          round: 2
        - id: BR-4
          disposition: addressed
          note: 'Verified: restoring SameTree=true + Register on replay reddens TestReplayPreservesSameTreeExactly.'
          round: 2
        - id: BR-5
          disposition: addressed
          note: 'Verified: swallowing every ReadFile error reddens TestUnreadableRegistryErrorsRatherThanReadingAsFirstRun.'
          round: 2
        - id: BR-6
          disposition: addressed
          note: 'Verified: an injected `couch nuke` branch ahead of Resolve now reddens TestCLIAcceptsExactlyTheDeclaredOperations.'
          round: 2
        - id: BR-7
          disposition: addressed
          note: Runtime.NewCouch makes start/stop/refusal reachable; the residual real-git tests are raised separately as a family repeat.
          round: 2
        - id: BR-8
          disposition: addressed
          note: 'Verified: the full pre-fix ExecRunner shape now reddens TestAliveIsFalseForAnExitedChildWithoutCallingWait in the default suite. Residual, not re-raised - no target or CI sets PAIR_LIVE_COUCH, so conformance still has no cadence.'
          round: 2
        - id: BR-9
          disposition: addressed
          note: WriteDescription now has a caller, a CLI operation and tests; the unresolvable-description half is raised separately as a family repeat.
          round: 2
        - id: BR-10
          disposition: addressed
          note: atlas section is now titled "built, unit-tested, not yet instantiated" and says no command starts one.
          round: 2
        - id: BR-11
          disposition: addressed
          note: README.md:260-264 names the second binary and points at atlas/couch.md.
          round: 2
        - id: BR-12
          disposition: addressed
          note: The issue Plan bullet now enumerates which smoke steps ran and states the kbench case is unrun; the plan file's own Task 17 checkboxes are stale in the opposite direction (see plan revisions).
          round: 2
        - id: BR-13
          disposition: not-addressed
          note: Still reproduces - Enqueue([stop{Control:true}], stop{}, 8) yields one entry with Control=false, ok=true.
          round: 2
        - id: BR-14
          disposition: not-addressed
          note: registry.go:70-80 and :85-94 still hold the identical occupancy block.
          round: 2
        - id: BR-15
          disposition: not-addressed
          note: Makefile.local:6-8 comment unamended; the binary is still bare-named `couch`.
          round: 2
        - id: BR-16
          disposition: not-addressed
          note: Both couchcmd/errors.go and couchcore/errors.go still wrap errors.As at one call site each.
          round: 2
        - id: BR-17
          disposition: not-addressed
          note: Couch.Policy, Registry.Unregister, FakeRunner.Signals and StartArgs.AgentStack still have zero non-test callers.
          round: 2
        - id: BR-18
          disposition: not-addressed
          note: bindArgs still stores every --flag without validating it against the operation's ArgSpecs.
          round: 2
        - id: BR-19
          disposition: not-addressed
          note: Makefile.local:131-132 still targets ./cmd/pair-wrap/; the target now fails outright with "directory not found / setup failed".
          round: 2
        - id: BR-20
          disposition: not-addressed
          note: strings.go unchanged - trimTrailingNewline is still TrimSpace and sanitizeKey still collides /a/b with /a_b.
          round: 2
        - id: BR-21
          disposition: not-addressed
          note: store.go Save still marshals reg.Records() in Go map order.
          round: 2
        - id: BR-22
          disposition: not-addressed
          note: No locking on registry.json; see the new partial-mutex finding for the same rule inside the process.
          round: 2
      findings:
        - id: BR-23
          severity: Important
          title: an agent-published description is displayed but does not resolve, so Done-when 3 is half delivered
          detail: |-
            PublishDescription writes the sidecar and Describe prefers it, but ResolveRef goes
            through NamingTable.Lookup, which searches only the operator-typed Name and
            Description. Verified in-package - publish "reworking the composer gate", then
            ResolveRef("composer") returns `no actor matches "composer"`. Done-when 3 requires
            the agent-supplied description to resolve to the right actor; only the operator's
            does. 2nd in this family - the rule is that every consumer of a description must
            derive from the agent's source, and display now derives while resolution does not.
          family: deferred-purpose
          round: 2
        - id: BR-24
          severity: Important
          title: --same-tree co-tenants cannot be stopped, and the error names a remedy that does not exist
          detail: |-
            ops.go's stop requires ResolveRef to return exactly one actor, but ResolveRef matches
            a name or a path and returns every actor on the tree; it has no ActorID branch.
            Verified through RunWithRuntime with two live co-tenants: stop "/repo" fails with
            `"/repo" matches 2 actors; be specific`, and stop "couch-ah8d" fails with
            `no actor matches "couch-ah8d"`. The escape hatch Done-when 2's refusal offers thus
            creates a state couch cannot exit. The same message also fires for a parked tree
            with zero actors, reading as ambiguity when it is absence.
          family: unaddressable-state
          round: 2
        - id: BR-25
          severity: Important
          title: three couchcmd tests drive real git against the ambient checkout, and one asserts on the checkout's directory name
          detail: |-
            run_test.go's run() helper uses testRT{fakes:false}, so NewCouch builds ExecGit and
            OSPathOps. TestShowResolvesANameToItsTreePath then asserts strings.Contains(out,
            "/pair"). Verified: in a pristine git worktree of the same commit it fails with
            out = "pairtree  /Users/xianxu/.cache/couchrev...", and identically under -race.
            2nd in this family - the rule is that every couchcmd test drives the CLI through
            Runtime's fakes. Measured prevalence: 7 of 9 test functions use the non-fake run();
            3 of those actually invoke git. Remove the fakes bool fork so the production path is
            unreachable from a test.
          family: cli-shell-not-injectable
          round: 2
        - id: BR-26
          severity: Important
          title: the launcher fake's new mutex guards 2 of 9 accessors of the map it protects
          detail: |-
            createflow_test.go locks WriteAtomic and Remove, but ReadAgentDefault, ReadFile,
            FileSize, Touch, Rename and ReadDir all touch f.files unguarded - and Touch and
            Rename write to it. A concurrent WriteAtomic and Touch is a concurrent map write,
            which Go turns into an unrecoverable fatal error rather than a test failure. 2nd in
            this family - the rule is that adding synchronisation to a shared structure means
            auditing every accessor, not only the one the race detector flagged. Measured
            prevalence: 2 of 9 locked.
          family: unsynchronised-shared-state
          round: 2
        - id: BR-27
          severity: Important
          title: atlas and plan hand-restate the operation set and seam list, and three restatements have drifted from the code
          detail: |-
            atlas/couch.md:13 and plan Task 15 both list six operations; seven ship
            (publish-description is documented nowhere a reader would look). The plan also
            states NormalizePath is Abs+Clean and prescribes a deletion check on a Clean the
            code deliberately omits, declares Couch without its Proc field, omits ProcOps and
            Liveness from the seam tables entirely, and leaves Task 17's checkboxes unticked
            with "operator, unrun" while the issue records steps 1-4 as run. No Revisions
            section exists, so round 1's five plan-revision recommendations never landed. 2nd in
            this family - the rule is that any prose restating the operation set or seam list is
            a consumer that must be re-derived at every boundary. Measured prevalence: 3
            restatements, 3 inconsistent.
          family: docs-claim-unbuilt-behavior
          round: 2
        - id: BR-28
          severity: Minor
          title: the new binary name is not gitignored and a built couch executable is sitting in the working tree
          detail: |-
            .gitignore covers bin/* but not a root-level binary; `git check-ignore couch`
            reports it unignored and a Mach-O executable is untracked at the repo root right
            now, so a `git add -A` would commit it.
          family: untracked-build-artifact
          round: 2
        - id: BR-29
          severity: Minor
          title: Views computes Liveness twice per record, so every list issues two probes per actor
          detail: |-
            couch.go's Views sets Live from c.Liveness(r) and State from a second c.Liveness(r)
            call. Each is a syscall plus a kernel-token read.
          family: redundant-recomputation
          round: 2
        - id: BR-30
          severity: Minor
          title: bindArgs's error is checked after its result is read
          detail: |-
            run.go reads parsed["tree"] and writes into parsed for the $COUCH_TREE default
            before the `if err != nil` check below it. Safe today only because of the
            `parsed != nil` guard; check err first.
          family: error-checked-after-use
          round: 2
      blocked: true
---

# Gate ledger — pair#145 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-21T16:52:24-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `guard-ignores-recomputed-liveness` the one-agent-per-tree guard never consults IsLive, so a dead actor blocks its tree forever
  registry.go:70 and :85 refuse on registry membership alone; Couch.IsLive
  (couch.go:94) is called only from Views/Summarize. Since `couch start`
  blocks and nothing unregisters on child exit, the normal end of a session
  leaves a dead record that permanently refuses the tree. Reproduced with
  the built binary: the refusal names a dead PID and offers "switch to it"
  (unimplemented, #146) or --same-tree (records a false co-tenancy).
- **BR-2** [Critical] `declared-contract-not-implemented` couch stop never signals its child, and forgetting the record opens the collision hazard
  ops.go:84 declares "Signal an actor's child and forget it"; ops.go:94 only
  calls c.Forget, and no production code calls Handle.Signal or any kill
  path. Stopping a live actor leaves the agent running with no registry
  entry, so a subsequent `couch start` on that tree is allowed -- two agents
  on one index lock and one branch. The summary is also the machine-facing
  contract for the advisor in pair#148.
- **BR-3** [Critical] `filter-argument-only-adds` couch show <ref> prints every tree with a registered actor, not the requested one
  Summarize (couch.go:213) takes a tree filter, then unconditionally folds in
  every record in the registry (couch.go:231), so the filter can only add.
  Verified in-package (Summarize([/repo]) -> [/other /repo]) and with the
  built binary, where `couch show <pair tree>` printed output identical to
  `couch list`. Contradicts the declared summary "Show the actors on one
  tree". The existing test passes only because its fixture has one tree.
- **BR-4** [Important] `replay-mutates-persisted-record` Store.Load fabricates same_tree=true on every record and the next Save persists it
  store.go:69 sets a.Args.SameTree = true on replay to dodge its own
  re-register refusal. Verified: save with SameTree=false, Load, Save, and
  the on-disk snapshot reads "same_tree": true. StartArgs documents this
  field as the single record of the escape hatch, so afterwards no reader can
  tell which actors actually used it. Give Registry an unchecked insert for
  Load instead.
- **BR-5** [Important] `silent-error-swallowing` an unreadable registry reads as a first run, and the next Save destroys it
  store.go:61 discards every ReadFile error, not just not-exist. Verified
  with registry.json at mode 000: Load returns err=nil with zero records, and
  the next spawn writes a one-actor snapshot over the old file. Distinguish
  fs.ErrNotExist from real IO/permission failures and return the latter.
- **BR-6** [Important] `test-cannot-fail-for-claimed-reason` the operation-set audit compares two views of the same source and cannot fail
  run_test.go:31 asserts Dispatch()'s keys equal OperationNames(); both derive
  from Operations(), so the identity is structural. I added an undeclared
  `couch nuke` branch ahead of the table lookup in RunWithRuntime and the
  suite stayed green -- the exact hazard the comment claims it catches, and
  what Done-when 6 asks to be audited rather than spot-checked.
- **BR-7** [Important] `cli-shell-not-injectable` couchcmd constructs production seams inline, so start/stop/refusal have no reachable tests
  run.go:87-91 builds ExecRunner/OSPathOps/ExecGit/OSProcOps directly;
  Runtime injects only env and store dir. No test exercises render's
  StartResult path or renderError's worktree-or-switch offer (Done-when 2),
  and existing CLI tests shell out to real git instead of FakeGit. ARCH-MOCK:
  production flow and test flow do not share the boundary at this layer,
  which is why the three Critical findings shipped.
- **BR-8** [Important] `fix-pinned-only-by-opt-in-test` the ExecRunner liveness fix is pinned only by a gate nothing runs
  Restoring the full pre-fix shape (no reaper, Alive = kill -0, Wait =
  cmd.Wait) leaves `go test ./cmd/internal/couchcore/` green in 0.35s; only
  PAIR_LIVE_COUCH=1 fails. No Makefile target sets any PAIR_LIVE_* gate, and
  test-race still points at ./cmd/pair-wrap/, which no longer exists.
  Separately, swapping Alive() back to procutil.Alive while keeping the
  reaper leaves BOTH suites green, so runner.go:120's "deliberately does NOT
  consult procutil.Alive" is undefended. A default-suite test that polls
  Alive() after `sh -c 'exit 0'` without calling Wait() would pin it.
- **BR-9** [Important] `deferred-purpose` the agent half of the agent-supplied description was not built
  Store.WriteDescription (store.go:101) has zero callers and zero tests;
  Couch.Describe prefers a sidecar nothing writes. What ships is an operator
  typing `couch describe`, landing in the naming table. The Spec is explicit
  that descriptors come from the agent with a cached fallback -- the cache
  exists with no source to cache from (ARCH-PURPOSE).
- **BR-10** [Important] `docs-claim-unbuilt-behavior` atlas/couch.md describes an actor loop that no command ever starts
  atlas/couch.md:67-77 says "One goroutine per actor, holding a bounded
  mailbox" in the present tense, but NewActor (actor.go:36) has no production
  call site -- Couch.Spawn starts a child and returns. Fine as pair#147
  groundwork; wrong as a map of what exists. Move it under "Planned, not
  built" or state that the loop is unit-tested but not instantiated.
- **BR-11** [Important] `readme-gate` README not updated for a second installed binary
  GO_BINS := pair couch means `make install` now puts a second executable on
  PATH, while README's Install section still says pair is "a single Go
  binary" and Command Usage lists only `pair ...`. One line pointing at
  atlas/couch.md clears the gate.
- **BR-12** [Important] `checklist-ticked-beyond-evidence` the issue ticks the operator smoke while four of its five steps are recorded unrun
  The issue Plan has "[x] Operator smoke: host one real pair child"; plan
  Task 17's five checkboxes are all unchecked, and the issue Log says the
  second-shell read path, the refusal offer and the kbench-subdirectory case
  were not exercised. Two of those unrun steps are exactly where the
  dead-actor refusal and the show-leak live.
- **BR-13** [Minor] `control-message-invariant` Enqueue's collapse can silently downgrade a queued Control message
  mailbox.go:35 matches on Kind alone, so a non-control message replaces a
  queued Control one of the same kind: Enqueue([stop{Control:true}],
  stop{}, 8) yields one entry with Control=false and ok=true. Contradicts
  "never drop a Control message"; unreachable today only because Actor has no
  production caller.
- **BR-14** [Minor] `duplicated-guard-block` CheckAvailable and RegisterWithPolicy duplicate the occupancy test verbatim
  registry.go:70-82 and :85-101 are the same block; ARCH-DRY, and it means
  the liveness fix for the Critical finding has to be written twice.
- **BR-15** [Minor] `binary-naming-convention` the bare-named couch binary contradicts the Makefile's own stated pair- prefix rule
  Makefile.local:6-8 says every Go binary ships with the pair- prefix to avoid
  PATH collisions (citing the bare `scribe` that was renamed). `couch` is
  bare and the comment was not amended either way.
- **BR-16** [Minor] `needless-indirection` two new errors.go files wrap errors.As at a single call site each
  couchcmd/errors.go and couchcore/errors.go add a one-line helper where the
  rest of the repo calls errors.As directly (wrap.go:2206, launcher tests).
- **BR-17** [Minor] `unused-public-surface` dead exported API and never-populated StartArgs fields
  Couch.List, Couch.Policy, Registry.Unregister, FakeRunner.Signals and
  StartArgs.AgentStack have zero non-test callers; the CLI never populates
  Stack, Issue or ExtraArgs, so the "structured start-args" record is
  effectively {Worktree, Cwd, SameTree}.
- **BR-18** [Minor] `silent-flag-acceptance` bindArgs accepts any --flag silently
  run.go:110 stores every --flag it sees, so `--same-tre` leaves the guard in
  force with no diagnostic -- unhelpful for the one loud escape hatch.
- **BR-19** [Minor] `stale-build-target` make test-race targets ./cmd/pair-wrap/, a directory that no longer exists
  Makefile.local:131. This diff's own lesson prescribes running the whole tree
  under -race; pointing the target at ./cmd/... would encode it instead of
  leaving it as a habit.
- **BR-20** [Minor] `misleading-helper-names` trimTrailingNewline is TrimSpace, and sanitizeKey can collide two trees
  strings.go:7 trims all surrounding whitespace, not a trailing newline;
  sanitizeKey maps /a/b and /a_b onto the same description sidecar file.
- **BR-21** [Minor] `nondeterministic-snapshot` Save writes actors in Go map order, so registry.json churns between identical saves
  store.go:47 uses reg.Records(), which iterates the map. Sorting by
  (worktree, id) would make the snapshot diffable.
- **BR-22** [Minor] `unsynchronised-shared-state` no locking on registry.json across couch processes
  Concurrent invocations are last-writer-wins over the whole snapshot. Narrow
  today because `couch start` saves once and then blocks, but it widens as
  soon as any command saves more than once.

## Round 2 — 2026-08-22T09:51:25-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Verified: deleting Spawn's PruneDead call reddens TestDeadActorDoesNotBlockItsTreeForever and TestKnownDeadIsStillPruned.
- BR-2 — addressed — Verified: deleting Stop's signal block reddens 2 couchcore tests and 1 couchcmd test.
- BR-3 — addressed — Verified: making Summarize additive again reddens TestShowFilterRestrictsRatherThanAdds ([/other /repo]).
- BR-4 — addressed — Verified: restoring SameTree=true + Register on replay reddens TestReplayPreservesSameTreeExactly.
- BR-5 — addressed — Verified: swallowing every ReadFile error reddens TestUnreadableRegistryErrorsRatherThanReadingAsFirstRun.
- BR-6 — addressed — Verified: an injected `couch nuke` branch ahead of Resolve now reddens TestCLIAcceptsExactlyTheDeclaredOperations.
- BR-7 — addressed — Runtime.NewCouch makes start/stop/refusal reachable; the residual real-git tests are raised separately as a family repeat.
- BR-8 — addressed — Verified: the full pre-fix ExecRunner shape now reddens TestAliveIsFalseForAnExitedChildWithoutCallingWait in the default suite. Residual, not re-raised - no target or CI sets PAIR_LIVE_COUCH, so conformance still has no cadence.
- BR-9 — addressed — WriteDescription now has a caller, a CLI operation and tests; the unresolvable-description half is raised separately as a family repeat.
- BR-10 — addressed — atlas section is now titled "built, unit-tested, not yet instantiated" and says no command starts one.
- BR-11 — addressed — README.md:260-264 names the second binary and points at atlas/couch.md.
- BR-12 — addressed — The issue Plan bullet now enumerates which smoke steps ran and states the kbench case is unrun; the plan file's own Task 17 checkboxes are stale in the opposite direction (see plan revisions).
- BR-13 — not-addressed — Still reproduces - Enqueue([stop{Control:true}], stop{}, 8) yields one entry with Control=false, ok=true.
- BR-14 — not-addressed — registry.go:70-80 and :85-94 still hold the identical occupancy block.
- BR-15 — not-addressed — Makefile.local:6-8 comment unamended; the binary is still bare-named `couch`.
- BR-16 — not-addressed — Both couchcmd/errors.go and couchcore/errors.go still wrap errors.As at one call site each.
- BR-17 — not-addressed — Couch.Policy, Registry.Unregister, FakeRunner.Signals and StartArgs.AgentStack still have zero non-test callers.
- BR-18 — not-addressed — bindArgs still stores every --flag without validating it against the operation's ArgSpecs.
- BR-19 — not-addressed — Makefile.local:131-132 still targets ./cmd/pair-wrap/; the target now fails outright with "directory not found / setup failed".
- BR-20 — not-addressed — strings.go unchanged - trimTrailingNewline is still TrimSpace and sanitizeKey still collides /a/b with /a_b.
- BR-21 — not-addressed — store.go Save still marshals reg.Records() in Go map order.
- BR-22 — not-addressed — No locking on registry.json; see the new partial-mutex finding for the same rule inside the process.

### Raised

- **BR-23** [Important] `deferred-purpose` an agent-published description is displayed but does not resolve, so Done-when 3 is half delivered
  PublishDescription writes the sidecar and Describe prefers it, but ResolveRef goes
  through NamingTable.Lookup, which searches only the operator-typed Name and
  Description. Verified in-package - publish "reworking the composer gate", then
  ResolveRef("composer") returns `no actor matches "composer"`. Done-when 3 requires
  the agent-supplied description to resolve to the right actor; only the operator's
  does. 2nd in this family - the rule is that every consumer of a description must
  derive from the agent's source, and display now derives while resolution does not.
- **BR-24** [Important] `unaddressable-state` --same-tree co-tenants cannot be stopped, and the error names a remedy that does not exist
  ops.go's stop requires ResolveRef to return exactly one actor, but ResolveRef matches
  a name or a path and returns every actor on the tree; it has no ActorID branch.
  Verified through RunWithRuntime with two live co-tenants: stop "/repo" fails with
  `"/repo" matches 2 actors; be specific`, and stop "couch-ah8d" fails with
  `no actor matches "couch-ah8d"`. The escape hatch Done-when 2's refusal offers thus
  creates a state couch cannot exit. The same message also fires for a parked tree
  with zero actors, reading as ambiguity when it is absence.
- **BR-25** [Important] `cli-shell-not-injectable` three couchcmd tests drive real git against the ambient checkout, and one asserts on the checkout's directory name
  run_test.go's run() helper uses testRT{fakes:false}, so NewCouch builds ExecGit and
  OSPathOps. TestShowResolvesANameToItsTreePath then asserts strings.Contains(out,
  "/pair"). Verified: in a pristine git worktree of the same commit it fails with
  out = "pairtree  /Users/xianxu/.cache/couchrev...", and identically under -race.
  2nd in this family - the rule is that every couchcmd test drives the CLI through
  Runtime's fakes. Measured prevalence: 7 of 9 test functions use the non-fake run();
  3 of those actually invoke git. Remove the fakes bool fork so the production path is
  unreachable from a test.
- **BR-26** [Important] `unsynchronised-shared-state` the launcher fake's new mutex guards 2 of 9 accessors of the map it protects
  createflow_test.go locks WriteAtomic and Remove, but ReadAgentDefault, ReadFile,
  FileSize, Touch, Rename and ReadDir all touch f.files unguarded - and Touch and
  Rename write to it. A concurrent WriteAtomic and Touch is a concurrent map write,
  which Go turns into an unrecoverable fatal error rather than a test failure. 2nd in
  this family - the rule is that adding synchronisation to a shared structure means
  auditing every accessor, not only the one the race detector flagged. Measured
  prevalence: 2 of 9 locked.
- **BR-27** [Important] `docs-claim-unbuilt-behavior` atlas and plan hand-restate the operation set and seam list, and three restatements have drifted from the code
  atlas/couch.md:13 and plan Task 15 both list six operations; seven ship
  (publish-description is documented nowhere a reader would look). The plan also
  states NormalizePath is Abs+Clean and prescribes a deletion check on a Clean the
  code deliberately omits, declares Couch without its Proc field, omits ProcOps and
  Liveness from the seam tables entirely, and leaves Task 17's checkboxes unticked
  with "operator, unrun" while the issue records steps 1-4 as run. No Revisions
  section exists, so round 1's five plan-revision recommendations never landed. 2nd in
  this family - the rule is that any prose restating the operation set or seam list is
  a consumer that must be re-derived at every boundary. Measured prevalence: 3
  restatements, 3 inconsistent.
- **BR-28** [Minor] `untracked-build-artifact` the new binary name is not gitignored and a built couch executable is sitting in the working tree
  .gitignore covers bin/* but not a root-level binary; `git check-ignore couch`
  reports it unignored and a Mach-O executable is untracked at the repo root right
  now, so a `git add -A` would commit it.
- **BR-29** [Minor] `redundant-recomputation` Views computes Liveness twice per record, so every list issues two probes per actor
  couch.go's Views sets Live from c.Liveness(r) and State from a second c.Liveness(r)
  call. Each is a syscall plus a kernel-token read.
- **BR-30** [Minor] `error-checked-after-use` bindArgs's error is checked after its result is read
  run.go reads parsed["tree"] and writes into parsed for the $COUCH_TREE default
  before the `if err != nil` check below it. Safe today only because of the
  `parsed != nil` guard; check err first.

## Open findings

- **BR-13** [Minor] `control-message-invariant` Enqueue's collapse can silently downgrade a queued Control message
- **BR-14** [Minor] `duplicated-guard-block` CheckAvailable and RegisterWithPolicy duplicate the occupancy test verbatim
- **BR-15** [Minor] `binary-naming-convention` the bare-named couch binary contradicts the Makefile's own stated pair- prefix rule
- **BR-16** [Minor] `needless-indirection` two new errors.go files wrap errors.As at a single call site each
- **BR-17** [Minor] `unused-public-surface` dead exported API and never-populated StartArgs fields
- **BR-18** [Minor] `silent-flag-acceptance` bindArgs accepts any --flag silently
- **BR-19** [Minor] `stale-build-target` make test-race targets ./cmd/pair-wrap/, a directory that no longer exists
- **BR-20** [Minor] `misleading-helper-names` trimTrailingNewline is TrimSpace, and sanitizeKey can collide two trees
- **BR-21** [Minor] `nondeterministic-snapshot` Save writes actors in Go map order, so registry.json churns between identical saves
- **BR-22** [Minor] `unsynchronised-shared-state` no locking on registry.json across couch processes
- **BR-23** [Important] `deferred-purpose` an agent-published description is displayed but does not resolve, so Done-when 3 is half delivered
- **BR-24** [Important] `unaddressable-state` --same-tree co-tenants cannot be stopped, and the error names a remedy that does not exist
- **BR-25** [Important] `cli-shell-not-injectable` three couchcmd tests drive real git against the ambient checkout, and one asserts on the checkout's directory name
- **BR-26** [Important] `unsynchronised-shared-state` the launcher fake's new mutex guards 2 of 9 accessors of the map it protects
- **BR-27** [Important] `docs-claim-unbuilt-behavior` atlas and plan hand-restate the operation set and seam list, and three restatements have drifted from the code
- **BR-28** [Minor] `untracked-build-artifact` the new binary name is not gitignored and a built couch executable is sitting in the working tree
- **BR-29** [Minor] `redundant-recomputation` Views computes Liveness twice per record, so every list issues two probes per actor
- **BR-30** [Minor] `error-checked-after-use` bindArgs's error is checked after its result is read
