---
gate: boundary-review
issue: 170
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-02T13:01:40-07:00"
      agent: claude
      boundary: M1
      blocked: false
      protocol_error: no valid findings block
    - "n": 2
      timestamp: "2026-09-02T16:05:44-07:00"
      agent: claude
      boundary: M2
      blocked: false
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-02T17:00:01-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: Detached rows are auto-selected at startup without the binding proof resume requires, so `couch` exits 1 with no way through
          detail: |-
            actionableinventory.go:238 appends detached candidates before any ResolveEstablished
            gate, while the parked branch at :243 filters on it and DecideResume/ResumeContext
            require it for both. Reproduced against HEAD: a detached row with an unbound binding
            is listed, selected by SelectUniqueResumableRoot, and refused with
            resume-binding-unbound; couchcmd renders the error and returns 1, and `couch` has no
            new-thread escape hatch. Before M3 the parked-only selector skipped it and startup
            created a new thread.
          family: listed-implies-resumable
          round: 3
        - id: BR-2
          severity: Important
          title: No couchcore test pins StartInteractive's detached wiring; only the pty-gated couchcmd acceptance test covers it
          detail: |-
            Filtering rows to parked before the SelectUniqueResumableRoot call in a scratch copy
            leaves the entire couchcore suite green. The two acceptance tests hard-fail at
            pty.Open() in any environment without pty access, so the commit's mutation claim for
            the reattach test is unconfirmable there. A ~15-line twin of
            TestStartInteractiveResumesUniqueExactParkedRoot closes it and would also catch the
            Critical.
          family: seam-untested-at-runnable-level
          round: 3
        - id: BR-3
          severity: Important
          title: Plan Step 3b, the project file and the commit credit M3 with a physicalization fix that shipped at M2
          detail: |-
            actionableinventory.go's parked-AND-detached physicalization is unchanged in this
            window; git log -L attributes it to fac153c9 (#170 M2), and M3's new test passes
            against the M2 base production source. M3 delivered the proof, not the fix. Correct
            the step wording and workshop/projects/couch.md:921-926.
          family: record-claims-unverified-delivery
          round: 3
        - id: BR-4
          severity: Important
          title: M3's stated startup envelope contradicts the M2-corrected cost model and is unmeasured
          detail: |-
            plan.md:402 says "unchanged from #167 -- one local actionable snapshot ... no
            fan-out", but the snapshot spawns 2 + N zellij CLI processes (N = every pair session
            on the host, 5s timeout each) whenever a detach candidate exists, and M3 makes that
            the normal startup case. atlas/couch.md:401-406 frames the cost as refresh-worker
            only. Measure a startup with K detached threads and correct both sentences.
          family: envelope-claim-unmeasured
          round: 3
        - id: BR-5
          severity: Important
          title: README still describes the switcher's row states and unique resume as parked-only
          detail: |-
            README.md:360 "Rows expose only proven live and exact verified parked states"
            contradicts M2's detached rows, which M3's Done-when depends on; README.md:308
            "automatic unique resume reuses the parked thread instead" is the same residue one
            paragraph below the sentence M2 correctly widened to "the sole exact resumable
            thread".
          family: readme-stale-for-shipped-surface
          round: 3
        - id: BR-6
          severity: Minor
          title: The selector's rationale is restated near-verbatim in five artifacts
          detail: |-
            startup.go:11-24, startup_test.go:13-19, atlas/couch.md:439-446,
            workshop/projects/couch.md:908-919 and the archived #167 plan all carry the same
            three-paragraph argument. Correct today, five copies to keep in sync.
          family: prose-duplication
          round: 3
        - id: BR-7
          severity: Minor
          title: The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
          detail: |-
            startup_test.go:37-38 duplicates one row; ProjectActionableThreads emits one row per
            record. The realistic distinct-address case exists two rows below, so this is
            fixture realism only.
          family: fixture-realism
          round: 3
      boundary: M3
      blocked: true
    - "n": 4
      timestamp: "2026-09-02T17:22:10-07:00"
      agent: claude
      boundary: M3
      blocked: true
      protocol_error: no valid findings block
    - "n": 5
      timestamp: "2026-09-02T17:36:29-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Gate reordered ahead of both appends; reverting it in a scratch copy reddens 3 subtests.
          round: 5
        - id: BR-2
          disposition: addressed
          note: New StartInteractive twin; the reviewer's exact parked-only mutation now reddens it.
          round: 5
        - id: BR-3
          disposition: addressed
          note: Plan Step 3b and workshop/projects/couch.md both credit M2 for the physicalization.
          round: 5
        - id: BR-4
          disposition: not-addressed
          note: Plan half corrected and measured; atlas/couch.md:399-406 still says refresh-worker-only, and "periodic" has no ticker.
          round: 5
        - id: BR-5
          disposition: addressed
          note: README.md:308 and :361 both rewritten for detached rows and resumable resume.
          round: 5
        - id: BR-6
          disposition: not-addressed
          note: All five copies of the selector rationale are unchanged in this window.
          round: 5
        - id: BR-7
          disposition: not-addressed
          note: startup_test.go:37-38 still duplicates one address for both ambiguity cases.
          round: 5
      findings:
        - id: BR-8
          severity: Important
          title: The detached row's binding proof is enforced in the IO shell, not the pure projector, and two comments claim the opposite
          detail: |-
            2nd finding in this family -- do NOT patch the instance. Rule: every proof a
            row's Enter requires must travel to ProjectActionableThreads as a field on that
            row's observation type and be enforced inside actionableThreadState; a proof
            enforced only in ActionableThreadInventoryContext's candidate loop is not part of
            the row's contract. Swept enumeration: Live needs a TTY observation (in projector),
            Parked needs NativeID (in projector, parkedResumeProofMatches at
            actionableinventory.go:181-187), Detached needs SessionName (in projector) AND the
            native binding (only at actionableinventory.go:250-253). That asymmetry is how BR-1
            shipped. actionableinventory.go:155-158 asserts the function "fails closed on its
            own, so it does not rely on the caller having filtered candidates" and :44-46 says
            "proof arrives as observations" -- both false for the binding. Class fix: add
            NativeID to DetachedSessionObservation and require it in the detached branch; the
            loop already holds binding at line 255, and ProjectActionableThreads is exported so
            a second caller is a public-API possibility. No live defect today
            (ScopedThreadArtifactCollisionChecker is the only production Artifacts, and it is
            gated). Minimum if the field threading is not cheap here: correct the two comments.
          family: listed-implies-resumable
          round: 5
        - id: BR-9
          severity: Minor
          title: The two new StartInteractive tests hand-rebuild the record setup seedStartupParked already encapsulates
          detail: |-
            startup_test.go:262-277 and :290-304 each repeat ~10 lines that
            seedStartupParked (startup_test.go:182-195) covers, differing only in
            markActionableParked vs SetDetachedSession. A
            seedStartupResumable(t, env, tag, path, kind) covers all three sites (ARCH-DRY).
          family: shared-helper-not-extracted
          round: 5
        - id: BR-10
          severity: Minor
          title: The no-surviving-session negative asserts only inequality, which a zero-valued record also satisfies
          detail: |-
            startup_test.go:309 checks start.Record.Thread != stale.Address, so a zero
            ThreadAddress would pass. TestStartInteractiveCreatesNewRootWithoutExactCandidate:109
            shows the stronger form; one line asserting a real new thread closes it.
          family: assertion-admits-vacuous-pass
          round: 5
      boundary: M3
      blocked: true
    - "n": 6
      timestamp: "2026-09-02T17:52:49-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: atlas:400-420 now names startup as the blocking caller, carries the 1.49 s measurement, and drops "periodic" (no refresh ticker exists in cmd/).
          round: 6
        - id: BR-6
          disposition: not-addressed
          note: 'Unchanged this window: startup.go:9-24, startup_test.go:12-18, atlas/couch.md, projects/couch.md all still carry the same rationale.'
          round: 6
        - id: BR-7
          disposition: not-addressed
          note: startup_test.go:36-38 still passes the same ThreadAddress twice for both ambiguity cases.
          round: 6
        - id: BR-8
          disposition: addressed
          note: Agent+NativeID added and enforced in detachedResumeProofMatches; mutation-verified in both directions (enforcement and reachability).
          round: 6
        - id: BR-9
          disposition: not-addressed
          note: startup_test.go:262-289 and :291-311 still hand-rebuild what seedStartupParked (:172-186) encapsulates.
          round: 6
        - id: BR-10
          disposition: not-addressed
          note: startup_test.go:309 still asserts only inequality; a zero StartResult would pass.
          round: 6
      findings:
        - id: BR-11
          severity: Important
          title: No durable record describes the layer the binding proof moved to; four records still describe the pre-fix shape
          detail: |-
            2nd finding in this family -- do NOT patch the four lines. Rule: a record that
            restates a code contract (a struct's fields, a projector's admission predicate,
            which layer enforces an invariant) is a hand-maintained restatement with no
            derivation, so a contract change must sweep the enumerated set in the same
            commit, and the enumeration belongs in the plan. Measured prevalence over three
            rounds: BR-3 (1 site), round 4's unrecorded candidate-rule item (3 sites, 2
            drifted), and this window (4 sites) -- atlas/couch.md:470-472, projects/couch.md:939-941,
            plan.md:238 (two-field struct, now four), plan.md:234 (admission predicate missing
            the agent and NativeID conjuncts). Durable fix: name the enumeration once in the
            plan's Core concepts preamble and stop transcribing field lists there.
          family: record-claims-unverified-delivery
          round: 6
        - id: BR-12
          severity: Important
          title: ProjectDetachedSessions emits a DetachedSessionObservation that ProjectActionableThreads now always rejects
          detail: |-
            detachedsessions.go:63 sets only Address and SessionName; actionableinventory.go:186-193
            now additionally requires Agent and NativeID. Both are exported pure functions in one
            package, so composing them directly yields zero detached rows -- silently, with no test
            covering it, and the operator-visible effect is "startup stops reattaching". Harmless
            today only because ActionableThreadInventoryContext:288-294 decorates in between and
            resume.go:222-226 reads only Address. Fix: split the type -- DetachedSessionObservation
            {Address, SessionName} for the session fact, DetachedResumeObservation adding the proof,
            assembled at the shell boundary (ARCH-SECURE: make the invalid state unrepresentable).
          family: producer-emits-value-its-consumer-rejects
          round: 6
        - id: BR-13
          severity: Important
          title: M3 produced a Critical and a three-round family and added no rule to workshop/lessons.md
          detail: |-
            lessons.md was last touched at M2 (9f7d4245); M1 has its own lessons commit (dec5928a),
            so per-milestone is this issue's own precedent and AGENTS.md section 4 asks for it. Two
            entries are owed, both two lines: widening an equivalence class widens its gates (gating
            one member of Resumable() and not the other is how BR-1 shipped); and a proof enforced
            in the IO shell is not part of the row's contract (BR-8's rule).
          family: lesson-not-recorded-for-boundary-defect
          round: 6
        - id: BR-14
          severity: Minor
          title: detachedResumeProofMatches repeats parkedResumeProofMatches' four-condition profile guard verbatim
          detail: |-
            2nd finding in this family -- do NOT patch the instance. Rule: when a second variant of
            an existing predicate is added, the invariant part is extracted into a shared helper in
            the same commit; a doc comment calling the two "twins" documents the duplication rather
            than removing it. Measured prevalence: 2 production sites (actionableinventory.go:187,
            :196) plus BR-9's 3 test sites. A resumableProfile(record) (*LaunchProfile, bool) helper
            covers both production sites and preserves the symmetry the comment defends (ARCH-DRY).
          family: shared-helper-not-extracted
          round: 6
      boundary: M3
      blocked: false
    - "n": 7
      timestamp: "2026-09-02T21:53:25-07:00"
      agent: claude
      findings:
        - id: BR-15
          severity: Critical
          title: Pristine-start rollback is gone; the call left in its place can never fire
          detail: |-
            AllocateThreadTag persists a reserved record and claims the artifact, so
            releaseClaimIfThreadAbsent at couch.go:297,301,313 always finds the record
            present and returns nil. A Proc.Current, entropy, or CommitStartClaim
            failure therefore leaks a reserved record that ProjectActionableThreads
            hides and reconcileInterruptedStarts never visits, plus its artifact claim.
            DeletePristineThread and rollbackUnforkedStart now have zero callers, and
            the two tests that pinned the property were deleted with admission. Plan
            Task 13 Step 4 required this rollback explicitly. A test setting
            FakeProcOps.CurrentErr and asserting zero records plus one release fails
            today.
          family: deleted-subsystem-drops-its-invariant
          round: 7
        - id: BR-16
          severity: Important
          title: Tombstone keeps old records decodable but drops the value they carry
          detail: |-
            A pre-M4 incarnation holds policy.repo_identity and no top-level
            repo_identity; fromPersistedThreadRecord (thread.go:184) never reads the
            tombstone, so RepoIdentity is empty and advanceSuccessfulStart
            (threadstore.go:614) refuses. Reached when an interrupted start written by
            the pre-M4 binary is promoted after the upgrade: reconcileInterruptedStarts
            fails, New returns an error, and couch refuses to start at all -- the same
            whole-store blast radius the manifest tombstone was added to prevent. The
            operator's store has 5 records with policy and 0 open starts today, so it
            does not fire on this host. Fix: read repo_identity out of DeprecatedPolicy
            when the new field is empty, and extend the fixture with a start block.
          family: compat-shim-preserves-shape-not-value
          round: 7
        - id: BR-17
          severity: Important
          title: Seven orphans survive the sweep, including a conformance target that now reports green
          detail: |-
            Each has exactly one reference, its own definition: threadrecord.PolicyCapacity
            (record.go:34), ThreadStore.DeletePristineThread (threadstore.go:526),
            Couch.rollbackUnforkedStart (couch.go:521), reflectBytesEqual
            (threadstore.go:905), ThreadSnapshotConflictError (threadstore.go:30),
            ThreadSnapshot.manifest/.raw (threadstore.go:39, write-only and copied per
            Snapshot call), and Makefile.local:73 test-couch-policy-live. That last one
            runs `go test -run TestFleetPolicyResolverConformance`, which now prints
            "ok [no tests to run]" and exits 0 -- verified. This is also the 3rd
            finding in family record-claims-unverified-delivery, since atlas/couch.md
            and the issue Log both state the target was deleted. Do not fix only these
            instances. The rule: a deletion is complete when the identifier set is
            swept, not the package -- grep every removed symbol and target name across
            Go, Makefile*, .github/, atlas/ and README.md before the commit, and let no
            doc or log sentence claim a removal grep has not confirmed. lessons.md
            states this rule in this very range and the range violates it, which argues
            for making it mechanical.
          family: deletion-leaves-orphaned-surface
          round: 7
        - id: BR-18
          severity: Important
          title: 4.5 MB Mach-O arm64 binary `couchstartrecovery` committed at the repo root
          detail: |-
            Landed in c11f61ea, almost certainly from `go build ./cmd/probes/couchstartrecovery`
            during the Task 14 Step 6 probe repair; the Makefile target uses `go run`, so
            nothing wants the file. No .gitignore rule covers the repo root. This is the
            ariadne base layer, so the blob propagates to dependent repos and becomes
            permanent history once merged -- the cost of removing it rises sharply at
            this boundary. Drop it from the branch and add a root ignore rule.
          family: build-artifact-committed
          round: 7
        - id: BR-19
          severity: Important
          title: README still documents fleet-policy admission, capacity refusal and provision-worktree
          detail: |-
            README.md:307-314 describes behavior D1 deleted. Plan Task 14's own Files
            list names README and it was not touched. This is the 2nd finding in this
            family, so do not fix only the paragraph. The rule: the docs sweep is
            driven by an enumeration derived from the diff -- deleted identifiers plus
            changed user-facing behavior -- grepped against README.md and atlas/
            together, not from recall of which files felt relevant. The atlas half was
            done well here and README was simply forgotten, which is exactly what a
            shared enumeration removes. A grep-based check over the deleted vocabulary
            would close the class.
          family: readme-stale-for-shipped-surface
          round: 7
        - id: BR-20
          severity: Important
          title: Overloading `absent` silently disables the pinned-declaration check for two entries
          detail: |-
            plan_contract_test.go:1481 skips issue151M3PinnedDeclarationExists whenever
            len(declaration.absent) != 0. Adding startgrant.go to the absent list of
            Couch.PrepareStart and Couch.SpawnPrepared therefore turned off a check
            both would still pass, since it reads the frozen git object rather than the
            worktree. Only StartGrantStore, whose source IS the absent file, needs the
            skip. Plan Task 14 Step 5 warned against loosening the pinned contract.
            Gate line 1481 on whether declaration.source itself appears in absent.
          family: guard-weakened-not-repaired
          round: 7
        - id: BR-21
          severity: Important
          title: The new git-common-dir seam has no live conformance case and one modeled answer shape
          detail: |-
            resolveRepoIdentity (couch.go:255) keys every saved launch preference off
            `git rev-parse --git-common-dir`, the one value D1 had to preserve
            byte-for-byte. Every fake reply in the tree is ".git" -- the repo-root case.
            Real git answers relative to the CURRENT directory from a subdirectory
            (measured: "../../../.git"), which is what production hits when the
            operator runs couch in a subdir; filepath.Join handles it, but nothing
            tests it and the function's doc comment states the model inaccurately
            ("absolute in a linked worktree"). conformance_live_test.go has no
            git-common-dir case. This is the 2nd finding in this family. The rule: a
            fake's canned replies must cover every input shape production can pass the
            seam -- enumerate the call sites' argument shapes and add a case per shape
            -- and any answer the code reasons about earns a live conformance case
            rather than a comment.
          family: fixture-realism
          round: 7
        - id: BR-22
          severity: Minor
          title: RetireIncarnation's doc comment is now attached to CommitStartClaim
          detail: |-
            threadstore.go:405-419 -- the new function was inserted between the comment
            and the function it documents, so godoc attributes RetireIncarnation's
            detach-vs-park rationale to CommitStartClaim and RetireIncarnation
            (threadstore.go:457) has no doc at all.
          family: doc-comment-misattached
          round: 7
        - id: BR-23
          severity: Minor
          title: GitRunner still documents exactly one call; there are two, and its unused ctx went with the policy seam
          detail: |-
            git.go:8-9 says "couch needs exactly one call: rev-parse --show-toplevel".
            Separately, resolveStartResolution's ctx parameter is now unused
            (ResolvePolicy was its only consumer), and the bounded 5 s policy
            subprocess it replaced is now an unbounded exec.Command git call on the
            preview worker. Pre-existing for ResolveTree, so not new exposure, but the
            last bounded IO on that path is gone. Couch.Spawn's doc (couch.go:128) also
            still reads as a production entry point where the plan promised "test seam".
          family: stale-doc-after-new-consumer
          round: 7
        - id: BR-24
          severity: Minor
          title: '`start` declares path/worktree/fingerprint Required but not ValueRequired, and worktree is never read'
          detail: |-
            ops.go:135-139 -- an empty fingerprint passes schema validation and is
            caught only by the later compare. resolveStartResolution uses
            WorkingDir(), which prefers Cwd, so the worktree arg CommitArgs emits is
            inert. Also: unrelated gofmt churn in entrypoint/alias_test.go,
            runtimebundle/store_test.go and two wrapcmd tests adds noise to the
            deletion diff.
          family: schema-looser-than-contract
          round: 7
      boundary: M4
      blocked: true
    - "n": 8
      timestamp: "2026-09-02T22:26:20-07:00"
      agent: claude
      boundary: M4
      blocked: true
      protocol_error: no valid findings block
    - "n": 9
      timestamp: "2026-09-02T22:41:43-07:00"
      agent: claude
      boundary: M4
      blocked: true
      protocol_error: no valid findings block
    - "n": 10
      timestamp: "2026-09-02T23:17:23-07:00"
      agent: claude
      dispose:
        - id: BR-15
          disposition: addressed
          note: rollbackPristineStart at couch.go:573 called from all three post-allocation sites; pinned by TestStartFailuresAfterTagAllocationRollBackTheReservation, whose budgetedReader puts the entropy failure after allocation rather than before.
          round: 10
        - id: BR-16
          disposition: addressed
          note: 'Measured: a pre-M4 record with an open start loads with RepoIdentity "/repo/.git" recovered from the policy tombstone and persists it as top-level repo_identity on the next write.'
          round: 10
        - id: BR-17
          disposition: addressed
          note: All seven orphans gone, test-couch-policy-live removed, and both guards added; I mutation-checked the dead-symbol guard and it fired. The six symbols it now allowlists are raised separately.
          round: 10
        - id: BR-18
          disposition: addressed
          note: The binary is untracked and root-anchored ignore rules exist; the guard meant to keep the list honest is raised separately.
          round: 10
        - id: BR-19
          disposition: addressed
          note: README.md:307-313 now describes couch-lite's no-gatekeeper behaviour and historicises the fleet policy.
          round: 10
        - id: BR-20
          disposition: addressed
          note: plan_contract_test.go:1489 now gates on whether declaration.source itself is absent, so PrepareStart and SpawnPrepared are checked again.
          round: 10
        - id: BR-21
          disposition: addressed
          note: 'I ran PAIR_LIVE_COUCH=1 TestGitConformance_LinkedWorktree: real git answered ".git", "../.git" and an absolute path, and production resolveRepoIdentity reduced all three to one identity. The doc comment now states the three-shape model.'
          round: 10
        - id: BR-22
          disposition: addressed
          note: CommitStartClaim and RetireIncarnation each carry their own doc comment.
          round: 10
        - id: BR-23
          disposition: addressed
          note: GitRunner documents both calls, resolveStartResolution's ctx is used, Spawn is documented as the test seam, and the 5s bound now lives at the seam pinned by a test that takes 5.00s and fails without it.
          round: 10
        - id: BR-24
          disposition: addressed
          note: The inert worktree argument is gone from ops.go and CommitArgs; an empty fingerprint is refused as such at SpawnPrepared with its own test. The tree is gofmt-clean.
          round: 10
      findings:
        - id: BR-25
          severity: Important
          title: The gitignore guard detects main packages by a byte prefix 5 of 8 fail, including the one that motivated it
          detail: |-
            3rd in this family, so the rule, not the instance. mainPackageDirs
            (maketargets_test.go:152) requires a file to begin with "package main\n";
            cmd/couch, cmd/pair-go, cmd/probes/couchstartrecovery, probes/termsmoke and
            probes/zellijpark all open with a doc comment. Measured in a scratch copy:
            deleting /couchstartrecovery, /couch, /pair-go, /termsmoke and /zellijpark
            from .gitignore leaves the test green; it only reddens on
            /pair-launch-helper. The plan Revision calls the list "Derived", and
            .gitignore:44 says "named after the main packages" -- both describe 3/8.
            Corroborating instance, same rule: deletedvocabulary_test.go derives its
            file set from git ls-files but hand-lists 5 terms against the 53 unique
            production declarations this window deletes, and one live-voice residue
            survives (resume_launch_test.go:183 names ReconcileResumeAdmission in the
            present tense). RULE: every axis of a mechanical guard's input must come
            from an oracle that already owns the answer (go list / go-parser / the
            boundary's deleted declarations), and the commit that adds the guard must
            mutation-check it against the artifact that motivated it, not an arbitrary
            member. This round's own lessons.md entry states that rule and this guard
            breaks it.
          family: assertion-admits-vacuous-pass
          round: 10
        - id: BR-26
          severity: Important
          title: Manifest tombstones are rewritten on every manifest write; two comments say they are never written
          detail: |-
            3rd in this family, so the rule, not the instance. nextManifest := manifest
            (threadstore.go:143, :789) copies the decoded envelope including its
            json.RawMessage tombstones and marshals it. Measured: after loading the
            liveManifest fixture and creating a thread, the manifest on disk still
            carries "legacy_cutover": true and "legacy_migration_version": 1.
            threadstore.go:52-54 and manifestcompat_test.go:26-27 both state they are
            "never written -- the manifest sheds them on its next write". Records DO
            shed (toPersistedThreadRecord builds a fresh struct; verified), so the
            asymmetry is silent. Harmless today, but a later milestone that deletes
            these fields trusting the claim takes the whole store down -- the exact
            blast radius the tombstone exists to prevent. RULE: a compat shim's stated
            lifecycle (decoded / read / written) is part of its contract, and the test
            that pins decodability must pin the rest of the sentence. The enumeration
            is five: Incarnation.DeprecatedPolicy, Incarnation.DeprecatedLegacyActorID,
            Record.DeprecatedClaimGeneration, threadManifest.DeprecatedLegacyCutover,
            threadManifest.DeprecatedLegacyMigrationVersion -- all pinned only on
            decode.
          family: record-claims-unverified-delivery
          round: 10
        - id: BR-27
          severity: Important
          title: Six production orphans are allowlisted against pair#173, an issue that does not exist
          detail: |-
            3rd in this family, so the rule, not the instance. deadsymbols_test.go:38-46
            admits PublishDescription, ReconcileActiveParks, OperationNames, Unregister,
            ResumeDiagnosticOf and ClassifyThreadReferenceFields with the reason
            "pair#173: ...", and the plan Revision repeats it. workshop/issues/ stops at
            000172, and pair#173 appears in exactly two files in the repo: that test and
            the plan. The allowlist's stated purpose -- "listed rather than silently
            tolerated so the debt is countable" -- fails when the citation resolves to
            nothing; as written it is six permanent exemptions. RULE: a deferral is only
            a deferral once the thing deferred to exists. An allowlist entry, plan step
            or comment naming an issue must name one that resolves, and the guard that
            reads the allowlist should fail on a reason citing an absent issue file.
            Cheapest fix: file the issue before the boundary and record its id.
          family: deletion-leaves-orphaned-surface
          round: 10
        - id: BR-28
          severity: Minor
          title: The bound's regression test costs 5.00s of wall clock because repoIdentityTimeout is a const
          detail: |-
            Measured: TestRepoIdentityResolutionIsBoundedEvenWithAnUncancelledContext
            takes 5.00s, since blockingGit waits for the real deadline. Making
            repoIdentityTimeout a package var or a Couch field lets the test assert the
            same property in 50 ms without weakening it.
          family: envelope-claim-unmeasured
          round: 10
        - id: BR-29
          severity: Minor
          title: The vocabulary guard t.Skips when git ls-files fails, while its sibling scan-broken case t.Fatals
          detail: |-
            deletedvocabulary_test.go:45 skips on git failure, so the guard silently
            vanishes wherever git is absent; the "found no tracked text files" case ten
            lines below correctly fatals. Same class of broken scan, opposite posture.
          family: assertion-admits-vacuous-pass
          round: 10
        - id: BR-30
          severity: Minor
          title: Two one-off README guards are now instances of what deletedVocabulary generalizes
          detail: |-
            TestReadmeDoesNotAdvertiseRemovedAdmissionFlags and
            TestReadmeDoesNotAdvertiseOwnerRequiredStopAsExternalCommand
            (couchcmd/run_test.go:741,752) each hand-roll one term against README.md.
            Folding "--same-tree" and "couch stop <ref>" into the deletedVocabulary
            table retires both (ARCH-DRY).
          family: prose-duplication
          round: 10
        - id: BR-31
          severity: Minor
          title: Two stale live-voice mentions of deleted admission surface
          detail: |-
            resume_launch_test.go:183 states "DecideResume and ReconcileResumeAdmission
            are tested with Detached hand-fed" for a function M4 deleted, and
            threadstore_test.go:550 is still named
            TestThreadStoreParkConflictsAndAbandonNeverReleaseAdmission. These are the
            measured residue behind the vocabulary half of the guard finding above; fix
            them via that rule rather than individually.
          family: deletion-leaves-orphaned-surface
          round: 10
      boundary: M4
      blocked: false
---

# Gate ledger — pair#170 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-02T13:01:40-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 2 — 2026-09-02T16:05:44-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-02T17:00:01-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `listed-implies-resumable` Detached rows are auto-selected at startup without the binding proof resume requires, so `couch` exits 1 with no way through
  actionableinventory.go:238 appends detached candidates before any ResolveEstablished
  gate, while the parked branch at :243 filters on it and DecideResume/ResumeContext
  require it for both. Reproduced against HEAD: a detached row with an unbound binding
  is listed, selected by SelectUniqueResumableRoot, and refused with
  resume-binding-unbound; couchcmd renders the error and returns 1, and `couch` has no
  new-thread escape hatch. Before M3 the parked-only selector skipped it and startup
  created a new thread.
- **BR-2** [Important] `seam-untested-at-runnable-level` No couchcore test pins StartInteractive's detached wiring; only the pty-gated couchcmd acceptance test covers it
  Filtering rows to parked before the SelectUniqueResumableRoot call in a scratch copy
  leaves the entire couchcore suite green. The two acceptance tests hard-fail at
  pty.Open() in any environment without pty access, so the commit's mutation claim for
  the reattach test is unconfirmable there. A ~15-line twin of
  TestStartInteractiveResumesUniqueExactParkedRoot closes it and would also catch the
  Critical.
- **BR-3** [Important] `record-claims-unverified-delivery` Plan Step 3b, the project file and the commit credit M3 with a physicalization fix that shipped at M2
  actionableinventory.go's parked-AND-detached physicalization is unchanged in this
  window; git log -L attributes it to fac153c9 (#170 M2), and M3's new test passes
  against the M2 base production source. M3 delivered the proof, not the fix. Correct
  the step wording and workshop/projects/couch.md:921-926.
- **BR-4** [Important] `envelope-claim-unmeasured` M3's stated startup envelope contradicts the M2-corrected cost model and is unmeasured
  plan.md:402 says "unchanged from #167 -- one local actionable snapshot ... no
  fan-out", but the snapshot spawns 2 + N zellij CLI processes (N = every pair session
  on the host, 5s timeout each) whenever a detach candidate exists, and M3 makes that
  the normal startup case. atlas/couch.md:401-406 frames the cost as refresh-worker
  only. Measure a startup with K detached threads and correct both sentences.
- **BR-5** [Important] `readme-stale-for-shipped-surface` README still describes the switcher's row states and unique resume as parked-only
  README.md:360 "Rows expose only proven live and exact verified parked states"
  contradicts M2's detached rows, which M3's Done-when depends on; README.md:308
  "automatic unique resume reuses the parked thread instead" is the same residue one
  paragraph below the sentence M2 correctly widened to "the sole exact resumable
  thread".
- **BR-6** [Minor] `prose-duplication` The selector's rationale is restated near-verbatim in five artifacts
  startup.go:11-24, startup_test.go:13-19, atlas/couch.md:439-446,
  workshop/projects/couch.md:908-919 and the archived #167 plan all carry the same
  three-paragraph argument. Correct today, five copies to keep in sync.
- **BR-7** [Minor] `fixture-realism` The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
  startup_test.go:37-38 duplicates one row; ProjectActionableThreads emits one row per
  record. The realistic distinct-address case exists two rows below, so this is
  fixture realism only.

## Round 4 — 2026-09-02T17:22:10-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 5 — 2026-09-02T17:36:29-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Gate reordered ahead of both appends; reverting it in a scratch copy reddens 3 subtests.
- BR-2 — addressed — New StartInteractive twin; the reviewer's exact parked-only mutation now reddens it.
- BR-3 — addressed — Plan Step 3b and workshop/projects/couch.md both credit M2 for the physicalization.
- BR-4 — not-addressed — Plan half corrected and measured; atlas/couch.md:399-406 still says refresh-worker-only, and "periodic" has no ticker.
- BR-5 — addressed — README.md:308 and :361 both rewritten for detached rows and resumable resume.
- BR-6 — not-addressed — All five copies of the selector rationale are unchanged in this window.
- BR-7 — not-addressed — startup_test.go:37-38 still duplicates one address for both ambiguity cases.

### Raised

- **BR-8** [Important] `listed-implies-resumable` The detached row's binding proof is enforced in the IO shell, not the pure projector, and two comments claim the opposite
  2nd finding in this family -- do NOT patch the instance. Rule: every proof a
  row's Enter requires must travel to ProjectActionableThreads as a field on that
  row's observation type and be enforced inside actionableThreadState; a proof
  enforced only in ActionableThreadInventoryContext's candidate loop is not part of
  the row's contract. Swept enumeration: Live needs a TTY observation (in projector),
  Parked needs NativeID (in projector, parkedResumeProofMatches at
  actionableinventory.go:181-187), Detached needs SessionName (in projector) AND the
  native binding (only at actionableinventory.go:250-253). That asymmetry is how BR-1
  shipped. actionableinventory.go:155-158 asserts the function "fails closed on its
  own, so it does not rely on the caller having filtered candidates" and :44-46 says
  "proof arrives as observations" -- both false for the binding. Class fix: add
  NativeID to DetachedSessionObservation and require it in the detached branch; the
  loop already holds binding at line 255, and ProjectActionableThreads is exported so
  a second caller is a public-API possibility. No live defect today
  (ScopedThreadArtifactCollisionChecker is the only production Artifacts, and it is
  gated). Minimum if the field threading is not cheap here: correct the two comments.
- **BR-9** [Minor] `shared-helper-not-extracted` The two new StartInteractive tests hand-rebuild the record setup seedStartupParked already encapsulates
  startup_test.go:262-277 and :290-304 each repeat ~10 lines that
  seedStartupParked (startup_test.go:182-195) covers, differing only in
  markActionableParked vs SetDetachedSession. A
  seedStartupResumable(t, env, tag, path, kind) covers all three sites (ARCH-DRY).
- **BR-10** [Minor] `assertion-admits-vacuous-pass` The no-surviving-session negative asserts only inequality, which a zero-valued record also satisfies
  startup_test.go:309 checks start.Record.Thread != stale.Address, so a zero
  ThreadAddress would pass. TestStartInteractiveCreatesNewRootWithoutExactCandidate:109
  shows the stronger form; one line asserting a real new thread closes it.

## Round 6 — 2026-09-02T17:52:49-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — atlas:400-420 now names startup as the blocking caller, carries the 1.49 s measurement, and drops "periodic" (no refresh ticker exists in cmd/).
- BR-6 — not-addressed — Unchanged this window: startup.go:9-24, startup_test.go:12-18, atlas/couch.md, projects/couch.md all still carry the same rationale.
- BR-7 — not-addressed — startup_test.go:36-38 still passes the same ThreadAddress twice for both ambiguity cases.
- BR-8 — addressed — Agent+NativeID added and enforced in detachedResumeProofMatches; mutation-verified in both directions (enforcement and reachability).
- BR-9 — not-addressed — startup_test.go:262-289 and :291-311 still hand-rebuild what seedStartupParked (:172-186) encapsulates.
- BR-10 — not-addressed — startup_test.go:309 still asserts only inequality; a zero StartResult would pass.

### Raised

- **BR-11** [Important] `record-claims-unverified-delivery` No durable record describes the layer the binding proof moved to; four records still describe the pre-fix shape
  2nd finding in this family -- do NOT patch the four lines. Rule: a record that
  restates a code contract (a struct's fields, a projector's admission predicate,
  which layer enforces an invariant) is a hand-maintained restatement with no
  derivation, so a contract change must sweep the enumerated set in the same
  commit, and the enumeration belongs in the plan. Measured prevalence over three
  rounds: BR-3 (1 site), round 4's unrecorded candidate-rule item (3 sites, 2
  drifted), and this window (4 sites) -- atlas/couch.md:470-472, projects/couch.md:939-941,
  plan.md:238 (two-field struct, now four), plan.md:234 (admission predicate missing
  the agent and NativeID conjuncts). Durable fix: name the enumeration once in the
  plan's Core concepts preamble and stop transcribing field lists there.
- **BR-12** [Important] `producer-emits-value-its-consumer-rejects` ProjectDetachedSessions emits a DetachedSessionObservation that ProjectActionableThreads now always rejects
  detachedsessions.go:63 sets only Address and SessionName; actionableinventory.go:186-193
  now additionally requires Agent and NativeID. Both are exported pure functions in one
  package, so composing them directly yields zero detached rows -- silently, with no test
  covering it, and the operator-visible effect is "startup stops reattaching". Harmless
  today only because ActionableThreadInventoryContext:288-294 decorates in between and
  resume.go:222-226 reads only Address. Fix: split the type -- DetachedSessionObservation
  {Address, SessionName} for the session fact, DetachedResumeObservation adding the proof,
  assembled at the shell boundary (ARCH-SECURE: make the invalid state unrepresentable).
- **BR-13** [Important] `lesson-not-recorded-for-boundary-defect` M3 produced a Critical and a three-round family and added no rule to workshop/lessons.md
  lessons.md was last touched at M2 (9f7d4245); M1 has its own lessons commit (dec5928a),
  so per-milestone is this issue's own precedent and AGENTS.md section 4 asks for it. Two
  entries are owed, both two lines: widening an equivalence class widens its gates (gating
  one member of Resumable() and not the other is how BR-1 shipped); and a proof enforced
  in the IO shell is not part of the row's contract (BR-8's rule).
- **BR-14** [Minor] `shared-helper-not-extracted` detachedResumeProofMatches repeats parkedResumeProofMatches' four-condition profile guard verbatim
  2nd finding in this family -- do NOT patch the instance. Rule: when a second variant of
  an existing predicate is added, the invariant part is extracted into a shared helper in
  the same commit; a doc comment calling the two "twins" documents the duplication rather
  than removing it. Measured prevalence: 2 production sites (actionableinventory.go:187,
  :196) plus BR-9's 3 test sites. A resumableProfile(record) (*LaunchProfile, bool) helper
  covers both production sites and preserves the symmetry the comment defends (ARCH-DRY).

## Round 7 — 2026-09-02T21:53:25-07:00 (claude) — BLOCKED

### Raised

- **BR-15** [Critical] `deleted-subsystem-drops-its-invariant` Pristine-start rollback is gone; the call left in its place can never fire
  AllocateThreadTag persists a reserved record and claims the artifact, so
  releaseClaimIfThreadAbsent at couch.go:297,301,313 always finds the record
  present and returns nil. A Proc.Current, entropy, or CommitStartClaim
  failure therefore leaks a reserved record that ProjectActionableThreads
  hides and reconcileInterruptedStarts never visits, plus its artifact claim.
  DeletePristineThread and rollbackUnforkedStart now have zero callers, and
  the two tests that pinned the property were deleted with admission. Plan
  Task 13 Step 4 required this rollback explicitly. A test setting
  FakeProcOps.CurrentErr and asserting zero records plus one release fails
  today.
- **BR-16** [Important] `compat-shim-preserves-shape-not-value` Tombstone keeps old records decodable but drops the value they carry
  A pre-M4 incarnation holds policy.repo_identity and no top-level
  repo_identity; fromPersistedThreadRecord (thread.go:184) never reads the
  tombstone, so RepoIdentity is empty and advanceSuccessfulStart
  (threadstore.go:614) refuses. Reached when an interrupted start written by
  the pre-M4 binary is promoted after the upgrade: reconcileInterruptedStarts
  fails, New returns an error, and couch refuses to start at all -- the same
  whole-store blast radius the manifest tombstone was added to prevent. The
  operator's store has 5 records with policy and 0 open starts today, so it
  does not fire on this host. Fix: read repo_identity out of DeprecatedPolicy
  when the new field is empty, and extend the fixture with a start block.
- **BR-17** [Important] `deletion-leaves-orphaned-surface` Seven orphans survive the sweep, including a conformance target that now reports green
  Each has exactly one reference, its own definition: threadrecord.PolicyCapacity
  (record.go:34), ThreadStore.DeletePristineThread (threadstore.go:526),
  Couch.rollbackUnforkedStart (couch.go:521), reflectBytesEqual
  (threadstore.go:905), ThreadSnapshotConflictError (threadstore.go:30),
  ThreadSnapshot.manifest/.raw (threadstore.go:39, write-only and copied per
  Snapshot call), and Makefile.local:73 test-couch-policy-live. That last one
  runs `go test -run TestFleetPolicyResolverConformance`, which now prints
  "ok [no tests to run]" and exits 0 -- verified. This is also the 3rd
  finding in family record-claims-unverified-delivery, since atlas/couch.md
  and the issue Log both state the target was deleted. Do not fix only these
  instances. The rule: a deletion is complete when the identifier set is
  swept, not the package -- grep every removed symbol and target name across
  Go, Makefile*, .github/, atlas/ and README.md before the commit, and let no
  doc or log sentence claim a removal grep has not confirmed. lessons.md
  states this rule in this very range and the range violates it, which argues
  for making it mechanical.
- **BR-18** [Important] `build-artifact-committed` 4.5 MB Mach-O arm64 binary `couchstartrecovery` committed at the repo root
  Landed in c11f61ea, almost certainly from `go build ./cmd/probes/couchstartrecovery`
  during the Task 14 Step 6 probe repair; the Makefile target uses `go run`, so
  nothing wants the file. No .gitignore rule covers the repo root. This is the
  ariadne base layer, so the blob propagates to dependent repos and becomes
  permanent history once merged -- the cost of removing it rises sharply at
  this boundary. Drop it from the branch and add a root ignore rule.
- **BR-19** [Important] `readme-stale-for-shipped-surface` README still documents fleet-policy admission, capacity refusal and provision-worktree
  README.md:307-314 describes behavior D1 deleted. Plan Task 14's own Files
  list names README and it was not touched. This is the 2nd finding in this
  family, so do not fix only the paragraph. The rule: the docs sweep is
  driven by an enumeration derived from the diff -- deleted identifiers plus
  changed user-facing behavior -- grepped against README.md and atlas/
  together, not from recall of which files felt relevant. The atlas half was
  done well here and README was simply forgotten, which is exactly what a
  shared enumeration removes. A grep-based check over the deleted vocabulary
  would close the class.
- **BR-20** [Important] `guard-weakened-not-repaired` Overloading `absent` silently disables the pinned-declaration check for two entries
  plan_contract_test.go:1481 skips issue151M3PinnedDeclarationExists whenever
  len(declaration.absent) != 0. Adding startgrant.go to the absent list of
  Couch.PrepareStart and Couch.SpawnPrepared therefore turned off a check
  both would still pass, since it reads the frozen git object rather than the
  worktree. Only StartGrantStore, whose source IS the absent file, needs the
  skip. Plan Task 14 Step 5 warned against loosening the pinned contract.
  Gate line 1481 on whether declaration.source itself appears in absent.
- **BR-21** [Important] `fixture-realism` The new git-common-dir seam has no live conformance case and one modeled answer shape
  resolveRepoIdentity (couch.go:255) keys every saved launch preference off
  `git rev-parse --git-common-dir`, the one value D1 had to preserve
  byte-for-byte. Every fake reply in the tree is ".git" -- the repo-root case.
  Real git answers relative to the CURRENT directory from a subdirectory
  (measured: "../../../.git"), which is what production hits when the
  operator runs couch in a subdir; filepath.Join handles it, but nothing
  tests it and the function's doc comment states the model inaccurately
  ("absolute in a linked worktree"). conformance_live_test.go has no
  git-common-dir case. This is the 2nd finding in this family. The rule: a
  fake's canned replies must cover every input shape production can pass the
  seam -- enumerate the call sites' argument shapes and add a case per shape
  -- and any answer the code reasons about earns a live conformance case
  rather than a comment.
- **BR-22** [Minor] `doc-comment-misattached` RetireIncarnation's doc comment is now attached to CommitStartClaim
  threadstore.go:405-419 -- the new function was inserted between the comment
  and the function it documents, so godoc attributes RetireIncarnation's
  detach-vs-park rationale to CommitStartClaim and RetireIncarnation
  (threadstore.go:457) has no doc at all.
- **BR-23** [Minor] `stale-doc-after-new-consumer` GitRunner still documents exactly one call; there are two, and its unused ctx went with the policy seam
  git.go:8-9 says "couch needs exactly one call: rev-parse --show-toplevel".
  Separately, resolveStartResolution's ctx parameter is now unused
  (ResolvePolicy was its only consumer), and the bounded 5 s policy
  subprocess it replaced is now an unbounded exec.Command git call on the
  preview worker. Pre-existing for ResolveTree, so not new exposure, but the
  last bounded IO on that path is gone. Couch.Spawn's doc (couch.go:128) also
  still reads as a production entry point where the plan promised "test seam".
- **BR-24** [Minor] `schema-looser-than-contract` `start` declares path/worktree/fingerprint Required but not ValueRequired, and worktree is never read
  ops.go:135-139 -- an empty fingerprint passes schema validation and is
  caught only by the later compare. resolveStartResolution uses
  WorkingDir(), which prefers Cwd, so the worktree arg CommitArgs emits is
  inert. Also: unrelated gofmt churn in entrypoint/alias_test.go,
  runtimebundle/store_test.go and two wrapcmd tests adds noise to the
  deletion diff.

## Round 8 — 2026-09-02T22:26:20-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 9 — 2026-09-02T22:41:43-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 10 — 2026-09-02T23:17:23-07:00 (claude) — passed

### Disposed

- BR-15 — addressed — rollbackPristineStart at couch.go:573 called from all three post-allocation sites; pinned by TestStartFailuresAfterTagAllocationRollBackTheReservation, whose budgetedReader puts the entropy failure after allocation rather than before.
- BR-16 — addressed — Measured: a pre-M4 record with an open start loads with RepoIdentity "/repo/.git" recovered from the policy tombstone and persists it as top-level repo_identity on the next write.
- BR-17 — addressed — All seven orphans gone, test-couch-policy-live removed, and both guards added; I mutation-checked the dead-symbol guard and it fired. The six symbols it now allowlists are raised separately.
- BR-18 — addressed — The binary is untracked and root-anchored ignore rules exist; the guard meant to keep the list honest is raised separately.
- BR-19 — addressed — README.md:307-313 now describes couch-lite's no-gatekeeper behaviour and historicises the fleet policy.
- BR-20 — addressed — plan_contract_test.go:1489 now gates on whether declaration.source itself is absent, so PrepareStart and SpawnPrepared are checked again.
- BR-21 — addressed — I ran PAIR_LIVE_COUCH=1 TestGitConformance_LinkedWorktree: real git answered ".git", "../.git" and an absolute path, and production resolveRepoIdentity reduced all three to one identity. The doc comment now states the three-shape model.
- BR-22 — addressed — CommitStartClaim and RetireIncarnation each carry their own doc comment.
- BR-23 — addressed — GitRunner documents both calls, resolveStartResolution's ctx is used, Spawn is documented as the test seam, and the 5s bound now lives at the seam pinned by a test that takes 5.00s and fails without it.
- BR-24 — addressed — The inert worktree argument is gone from ops.go and CommitArgs; an empty fingerprint is refused as such at SpawnPrepared with its own test. The tree is gofmt-clean.

### Raised

- **BR-25** [Important] `assertion-admits-vacuous-pass` The gitignore guard detects main packages by a byte prefix 5 of 8 fail, including the one that motivated it
  3rd in this family, so the rule, not the instance. mainPackageDirs
  (maketargets_test.go:152) requires a file to begin with "package main\n";
  cmd/couch, cmd/pair-go, cmd/probes/couchstartrecovery, probes/termsmoke and
  probes/zellijpark all open with a doc comment. Measured in a scratch copy:
  deleting /couchstartrecovery, /couch, /pair-go, /termsmoke and /zellijpark
  from .gitignore leaves the test green; it only reddens on
  /pair-launch-helper. The plan Revision calls the list "Derived", and
  .gitignore:44 says "named after the main packages" -- both describe 3/8.
  Corroborating instance, same rule: deletedvocabulary_test.go derives its
  file set from git ls-files but hand-lists 5 terms against the 53 unique
  production declarations this window deletes, and one live-voice residue
  survives (resume_launch_test.go:183 names ReconcileResumeAdmission in the
  present tense). RULE: every axis of a mechanical guard's input must come
  from an oracle that already owns the answer (go list / go-parser / the
  boundary's deleted declarations), and the commit that adds the guard must
  mutation-check it against the artifact that motivated it, not an arbitrary
  member. This round's own lessons.md entry states that rule and this guard
  breaks it.
- **BR-26** [Important] `record-claims-unverified-delivery` Manifest tombstones are rewritten on every manifest write; two comments say they are never written
  3rd in this family, so the rule, not the instance. nextManifest := manifest
  (threadstore.go:143, :789) copies the decoded envelope including its
  json.RawMessage tombstones and marshals it. Measured: after loading the
  liveManifest fixture and creating a thread, the manifest on disk still
  carries "legacy_cutover": true and "legacy_migration_version": 1.
  threadstore.go:52-54 and manifestcompat_test.go:26-27 both state they are
  "never written -- the manifest sheds them on its next write". Records DO
  shed (toPersistedThreadRecord builds a fresh struct; verified), so the
  asymmetry is silent. Harmless today, but a later milestone that deletes
  these fields trusting the claim takes the whole store down -- the exact
  blast radius the tombstone exists to prevent. RULE: a compat shim's stated
  lifecycle (decoded / read / written) is part of its contract, and the test
  that pins decodability must pin the rest of the sentence. The enumeration
  is five: Incarnation.DeprecatedPolicy, Incarnation.DeprecatedLegacyActorID,
  Record.DeprecatedClaimGeneration, threadManifest.DeprecatedLegacyCutover,
  threadManifest.DeprecatedLegacyMigrationVersion -- all pinned only on
  decode.
- **BR-27** [Important] `deletion-leaves-orphaned-surface` Six production orphans are allowlisted against pair#173, an issue that does not exist
  3rd in this family, so the rule, not the instance. deadsymbols_test.go:38-46
  admits PublishDescription, ReconcileActiveParks, OperationNames, Unregister,
  ResumeDiagnosticOf and ClassifyThreadReferenceFields with the reason
  "pair#173: ...", and the plan Revision repeats it. workshop/issues/ stops at
  000172, and pair#173 appears in exactly two files in the repo: that test and
  the plan. The allowlist's stated purpose -- "listed rather than silently
  tolerated so the debt is countable" -- fails when the citation resolves to
  nothing; as written it is six permanent exemptions. RULE: a deferral is only
  a deferral once the thing deferred to exists. An allowlist entry, plan step
  or comment naming an issue must name one that resolves, and the guard that
  reads the allowlist should fail on a reason citing an absent issue file.
  Cheapest fix: file the issue before the boundary and record its id.
- **BR-28** [Minor] `envelope-claim-unmeasured` The bound's regression test costs 5.00s of wall clock because repoIdentityTimeout is a const
  Measured: TestRepoIdentityResolutionIsBoundedEvenWithAnUncancelledContext
  takes 5.00s, since blockingGit waits for the real deadline. Making
  repoIdentityTimeout a package var or a Couch field lets the test assert the
  same property in 50 ms without weakening it.
- **BR-29** [Minor] `assertion-admits-vacuous-pass` The vocabulary guard t.Skips when git ls-files fails, while its sibling scan-broken case t.Fatals
  deletedvocabulary_test.go:45 skips on git failure, so the guard silently
  vanishes wherever git is absent; the "found no tracked text files" case ten
  lines below correctly fatals. Same class of broken scan, opposite posture.
- **BR-30** [Minor] `prose-duplication` Two one-off README guards are now instances of what deletedVocabulary generalizes
  TestReadmeDoesNotAdvertiseRemovedAdmissionFlags and
  TestReadmeDoesNotAdvertiseOwnerRequiredStopAsExternalCommand
  (couchcmd/run_test.go:741,752) each hand-roll one term against README.md.
  Folding "--same-tree" and "couch stop <ref>" into the deletedVocabulary
  table retires both (ARCH-DRY).
- **BR-31** [Minor] `deletion-leaves-orphaned-surface` Two stale live-voice mentions of deleted admission surface
  resume_launch_test.go:183 states "DecideResume and ReconcileResumeAdmission
  are tested with Detached hand-fed" for a function M4 deleted, and
  threadstore_test.go:550 is still named
  TestThreadStoreParkConflictsAndAbandonNeverReleaseAdmission. These are the
  measured residue behind the vocabulary half of the guard finding above; fix
  them via that rule rather than individually.

## Open findings

- **BR-6** [Minor] `prose-duplication` The selector's rationale is restated near-verbatim in five artifacts
- **BR-7** [Minor] `fixture-realism` The selector's ambiguity fixtures pass the same ThreadAddress twice, a shape the projector cannot emit
- **BR-9** [Minor] `shared-helper-not-extracted` The two new StartInteractive tests hand-rebuild the record setup seedStartupParked already encapsulates
- **BR-10** [Minor] `assertion-admits-vacuous-pass` The no-surviving-session negative asserts only inequality, which a zero-valued record also satisfies
- **BR-11** [Important] `record-claims-unverified-delivery` No durable record describes the layer the binding proof moved to; four records still describe the pre-fix shape
- **BR-12** [Important] `producer-emits-value-its-consumer-rejects` ProjectDetachedSessions emits a DetachedSessionObservation that ProjectActionableThreads now always rejects
- **BR-13** [Important] `lesson-not-recorded-for-boundary-defect` M3 produced a Critical and a three-round family and added no rule to workshop/lessons.md
- **BR-14** [Minor] `shared-helper-not-extracted` detachedResumeProofMatches repeats parkedResumeProofMatches' four-condition profile guard verbatim
- **BR-25** [Important] `assertion-admits-vacuous-pass` The gitignore guard detects main packages by a byte prefix 5 of 8 fail, including the one that motivated it
- **BR-26** [Important] `record-claims-unverified-delivery` Manifest tombstones are rewritten on every manifest write; two comments say they are never written
- **BR-27** [Important] `deletion-leaves-orphaned-surface` Six production orphans are allowlisted against pair#173, an issue that does not exist
- **BR-28** [Minor] `envelope-claim-unmeasured` The bound's regression test costs 5.00s of wall clock because repoIdentityTimeout is a const
- **BR-29** [Minor] `assertion-admits-vacuous-pass` The vocabulary guard t.Skips when git ls-files fails, while its sibling scan-broken case t.Fatals
- **BR-30** [Minor] `prose-duplication` Two one-off README guards are now instances of what deletedVocabulary generalizes
- **BR-31** [Minor] `deletion-leaves-orphaned-surface` Two stale live-voice mentions of deleted admission surface
