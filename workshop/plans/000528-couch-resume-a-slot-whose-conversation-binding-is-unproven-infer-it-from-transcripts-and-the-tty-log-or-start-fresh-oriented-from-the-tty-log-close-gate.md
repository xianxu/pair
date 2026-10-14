---
gate: boundary-review
issue: 528
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T11:01:13-07:00"
      dispatched: "2026-10-13T10:58:01-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Unknown terminal observations become live or actionable relaunch verdicts
          detail: cmd/internal/couchtty/menu_reattach.go:373 promotes ReasonUnknown to ThreadLive whenever an attached-pass entry exists. cmd/internal/launcher/session_handoff_os.go:73-97 also collapses inspection errors into false usability, which cmd/internal/couchcore/agent_usability.go:7 converts to needs-relaunch. Preserve typed usable/unsafe/unknown outcomes through observers, classifiers, and overlays; test failed refresh after successful attachment and failed metadata inspection. ARCH-ORDER, ARCH-PURPOSE.
          family: preserve-observation-uncertainty
          round: 1
        - id: BR-2
          severity: Critical
          title: Agent usability accepts a recycled agent PID beneath the same server
          detail: cmd/internal/launcher/terminal_agent.go:7-34 compares only the server generation; cmd/internal/launcher/session_handoff_os.go:205 supplies a readiness PID without its expected start identity. A replacement descendant reusing that PID passes stale readiness. Persist and validate the agent generation, with a regression changing only the agent identity while preserving PID and ancestry. ARCH-SECURE.
          family: exact-process-generation-authority
          round: 1
        - id: BR-3
          severity: Important
          title: Policy and readiness reads can defeat the admission timeout
          detail: cmd/internal/launcher/session_handoff_os.go:127 opens policy files without regular-file/nonblocking safeguards, and readiness reads at lines 193 and 423 are unbounded. A FIFO can hang refresh or admission indefinitely because cancellation cannot interrupt these reads and the coordinator joins the observer. Share the cold-check branch's bounded regular-file reader across these consumers and test FIFO and oversized inputs. ARCH-CONSTRAINTS, ARCH-SECURE.
          family: bounded-untrusted-artifact-reads
          round: 1
        - id: BR-4
          severity: Important
          title: Per-terminal coordination files accumulate without an aggregate lifecycle bound
          detail: cmd/internal/artifactpath/session_policy.go:151-165 allocates one lock pathname per canonical socket, while cmd/internal/artifactpath/manifest.go:1460 permanently protects every such inode. Define a bounded coordination scheme or safe retirement protocol without unlinking active or awaited locks; document and measure any intentional permanent per-session cost. ARCH-FUNERAL.
          family: durable-artifact-lifecycle-bound
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T11:34:12-07:00"
      dispatched: "2026-10-13T11:30:13-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Typed uncertainty reaches core projection and the attached-pass overlay. Focused tests pass; removing the unknown-state overlay guard or converting inspection errors to unsafe makes the respective regressions fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Readiness records carry agent start identity and AgentBelongsToServer validates it. Removing the agent-identity comparison fails the unchanged-PID/ancestry regression.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Policy and readiness consumers use readBoundedRegularMetadata. Its regressions pass; replacing it with os.ReadFile fails FIFO, oversized-input, symlink and cancellation cases. Additional consumers missed by the family sweep are reported separately below.
          round: 2
        - id: BR-4
          disposition: addressed
          note: TerminalLease now selects among 256 permanent stripes. The 10,000-socket regression passes and fails when full-hash per-socket lock paths are restored.
          round: 2
      findings:
        - id: BR-5
          severity: Important
          title: Capture-witness and admission-config reads still permit indefinite blocking
          detail: 'cmd/internal/storagegc/capture_metadata.go:142 opens events with blocking os.Open before its byte limit; a FIFO without a writer blocks capture preservation indefinitely, reproduced by a disposable regression probe. cmd/internal/launcher/session_handoff_os.go:468 also reads config.kdl with unbounded, non-cancellable os.ReadFile, so admission can remain stuck beyond its timeout while the coordinator waits for the observer. This is the 2nd finding in family bounded-untrusted-artifact-reads. Do NOT fix only these instances: enumerate every persisted-input read added by M1 and enforce one bounded, regular-file, nonblocking, cancellation-aware rule through a shared IO owner, with consumer-level FIFO and oversized-input regressions. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-PURPOSE.'
          family: bounded-untrusted-artifact-reads
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-13T12:06:16-07:00"
      dispatched: "2026-10-13T12:03:38-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Prior disposition retained; unknown-observation and reattach-overlay regressions pass.
          round: 3
        - id: BR-2
          disposition: addressed
          note: Prior disposition retained; AgentBelongsToServer checks both server and agent process generations.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Prior disposition retained; policy and readiness consumers use the shared bounded reader, with passing FIFO regressions.
          round: 3
        - id: BR-4
          disposition: addressed
          note: Prior disposition retained; permanent coordination files remain bounded to 256 stripes, with passing lease tests.
          round: 3
        - id: BR-5
          disposition: addressed
          note: Enumerated metadata consumers use osfs.ReadRegularMetadata. Consumer FIFO, oversized-input and cancellation tests pass; substituting os.ReadFile in a disposable checkout makes the regressions fail. Both disposable checkouts were removed.
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: dfedd894650cbf1f3cc2dbdb5782b8724131f01d
      blocked: false
    - "n": 4
      timestamp: "2026-10-13T13:26:16-07:00"
      dispatched: "2026-10-13T13:20:56-07:00"
      agent: codex
      findings:
        - id: BR-6
          severity: Critical
          title: Native ownership substitutes the latest tag-level launch for the live wrapper's launch
          detail: 'cmd/internal/launcher/recovery_native_owner.go:85 uses CurrentLaunch(scope, tag, agent) without associating that launch with the observed wrapper PID/start identity. A regression retaining the old wrapper while appending a newer launch targeting another conversation incorrectly returns success. This is the 2nd finding in family `exact-process-generation-authority`. Earlier rounds fixed instances. Do NOT fix only this instance: state and enforce the rule that every native-ownership assertion must bind to the exact observed process generation; sweep preflight, Pair admission, source exemptions, and their doubles. Missing association must remain unresolved. ARCH-SECURE, ARCH-ORDER, ARCH-MOCK.'
          family: exact-process-generation-authority
          round: 4
        - id: BR-7
          severity: Critical
          title: The advertised primary Relaunch action never opens the new recovery preview
          detail: cmd/internal/couchtty/menu.go:864 handles only the new resume-candidates item, appended separately at menu_switchagent.go:204. Unsafe-row Enter/mouse still select relaunch at menu.go:803 and open the old confirmation through menu_slot.go:34; the old executor requires established binding. The primary-route regression fails with the confirmation frame and no preview effect. Route Enter, mouse, and the unsafe-row Relaunch item through the same evidence preview while preserving ordinary healthy relaunch behavior. ARCH-PURPOSE.
          family: recovery-primary-action-routing
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 5
      timestamp: "2026-10-13T13:39:34-07:00"
      dispatched: "2026-10-13T13:34:13-07:00"
      agent: codex
      dispose:
        - id: BR-6
          disposition: addressed
          note: Ownership now resolves the launch ordinal associated with the observed wrapper's live readiness generation. Checked preflight, final Couch check, Pair admission, source exemptions, and the stateful double. TestRecoveryNativeOwnerOldWrapperCannotBorrowNewLaunch passes at head and fails when CurrentLaunch receives the entire tag ledger again.
          round: 5
        - id: BR-7
          disposition: addressed
          note: Unsafe Enter, mouse, Relaunch item, and restart shortcut share the recovery preview route. Production reducer tests pass; disabling the shared routing predicate fails all four cases. Healthy relaunch retains its existing confirmation.
          round: 5
      findings:
        - id: BR-8
          severity: Important
          title: Recovery-specific native parsing is outside live conformance
          detail: 'ARCH-MOCK: cmd/internal/sessioninventory/recovery_candidate.go:192 introduces eligibility based on Claude entrypoint and related provenance, but conformance.go:52 checks only the existing scanners; conformance_live_test.go:21 never invokes the recovery parser or collector. Native changes can therefore leave conformance green while recovery eligibility or tail evidence stops working. Extend the existing read-only conformance path through the production recovery parser, covering provenance, cwd, activity and authored tails, with explicit unsupported/absent-sample outcomes and the existing contract-change/monthly cadence.'
          family: external-contract-conformance
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 6
      timestamp: "2026-10-13T13:49:02-07:00"
      dispatched: "2026-10-13T13:44:33-07:00"
      agent: codex
      dispose:
        - id: BR-8
          disposition: addressed
          note: conformance_live_test.go:24 invokes RunRecoveryConformance, which exercises bounded production parsing and checks provenance, cwd, activity, and authored tails. Removing the provenance rejection at conformance.go:281 made the Codex drift and Claude missing/changed-entrypoint regressions fail. Unsupported and absent samples report skip explicitly; atlas/session-identity.md preserves contract-change/monthly cadence.
          round: 6
        - id: BR-6
          disposition: addressed
          note: recovery_native_owner.go resolves the readiness-proved launch ordinal only after associating the exact live agent generation with its wrapper. Exact-generation and source-exemption regressions passed.
          round: 6
        - id: BR-7
          disposition: addressed
          note: Unsafe primary activation and relaunch routes use the recovery chooser. TestReviewNeedsRelaunchPrimaryOpensRecovery and the targeted UI routing tests passed.
          round: 6
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 81a4fc7d5a62180663d7b84245c0c985fedbec21
      blocked: false
    - "n": 7
      timestamp: "2026-10-13T21:42:34-07:00"
      dispatched: "2026-10-13T21:37:49-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: The approved replacement contract distinguishes positive listing from failed observation. TestListedLiveIsContinueAuthority and TestAttributedOwnerCannotResumeDisk pass; unknown observations do not authorize Continue.
          round: 7
        - id: BR-2
          disposition: addressed
          note: Exact agent-generation comparison remains in terminal_agent.go. TestTerminalAgentRequiresExactServerAndLiveDescendant passes and fails when that comparison is removed.
          round: 7
        - id: BR-3
          disposition: addressed
          note: Policy reads were removed from admission. Remaining readiness reads in session_handoff_os.go use contextual ReadRegularMetadata; bounded metadata regressions pass.
          round: 7
        - id: BR-4
          disposition: addressed
          note: session_lock.go preserves the 256-stripe bound and policy publication is removed. TestTerminalLeasePathsHaveFixedBound passes and fails when full socket hashes replace stripe keys.
          round: 7
        - id: BR-5
          disposition: addressed
          note: Admission-config reads were removed. Capture-witness reads still use ReadRegularMetadata in recovery_evidence_os.go:141; bounded metadata and wrong-witness regressions pass.
          round: 7
      findings:
        - id: BR-9
          severity: Critical
          title: Continue is offered where the resume executor still refuses attachment
          detail: 'actionableinventory.go:494 now projects any positively listed session as Continue, but ResumeContextWith calls DecideResume, whose resume.go:186 preconditions still reject missing/incomplete profiles, unsupported saved agents, and unavailable working paths. A temporary review regression reproduced all four refusals. This is the 2nd finding in family recovery-primary-action-routing: state and enforce the rule that every offered Continue reaches a warm attach using listed-session authority; sweep all downstream warm-path gates rather than patching one profile check. ARCH-PURPOSE.'
          family: recovery-primary-action-routing
          round: 7
        - id: BR-10
          severity: Important
          title: README update appears missing for changed Couch startup and action semantics
          detail: startup.go:73 removes automatic launch/resume on ordinary Couch startup, and menu.go:1446 changes visible action labels. README.md:389 remains unchanged throughout the pinned range. Document observation-only startup, explicit action selection, and the M1 recovery limitations alongside the existing usage instructions; atlas/couch.md already describes these changes.
          family: user-facing-contract-documentation
          round: 7
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 8
      timestamp: "2026-10-13T22:11:38-07:00"
      dispatched: "2026-10-13T22:08:52-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Retained disposition. Failed listing does not authorize Continue; the revised contract explicitly supersedes policy-based terminal usability.
          round: 8
        - id: BR-2
          disposition: addressed
          note: Retained disposition. Exact-generation helpers remain; policy/readiness-based warm admission was explicitly removed by the approved replacement design.
          round: 8
        - id: BR-3
          disposition: addressed
          note: Retained disposition. Policy reads were removed; remaining admission readiness reads use bounded ReadRegularMetadata.
          round: 8
        - id: BR-4
          disposition: addressed
          note: TerminalLease retains the aggregate bound of 256 permanent lock stripes in artifactpath/session_lock.go.
          round: 8
        - id: BR-5
          disposition: addressed
          note: Retained disposition. Shared bounded metadata reading remains; the obsolete admission-config reader was removed.
          round: 8
        - id: BR-9
          disposition: not-addressed
          note: ARCH-PURPOSE / ARCH-ORDER; family recovery-primary-action-routing remains open. Metadata regressions pass and fail with the old eligibility function restored, but resume.go:556 still calls clearLifecycleDebris, whose lifecycledebris.go:87 refuses an outstanding start owned by a live supervisor. A temporary regression constructed a valid StartClaim, positively listed its exact detached session, verified classification offers Continue, then observed ResumeContextWith(WarmOnly:true) fail with resume-starting before attachment. Separate listed-session attachment from lifecycle retirement; enumerate and test all accepted record shapes through the executor rather than fixing another individual gate.
          round: 8
        - id: BR-10
          disposition: addressed
          note: README.md:395 now documents observation-only startup and explicit actions; the recovery section documents Starting, restricted Fresh execution, legacy attachment race and unavailable messaging. These passages agree with startup.go:73, the menu projection and atlas/couch.md. This is a prose correction and requires no wording-presence regression.
          round: 8
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 9
      timestamp: "2026-10-13T23:03:06-07:00"
      dispatched: "2026-10-13T22:56:16-07:00"
      agent: codex
      dispose:
        - id: BR-9
          disposition: addressed
          note: Public Continue bypasses lifecycle adoption and cold prerequisites. The record-shape regression passes at HEAD and fails when the helper-only route is disabled, reproducing refusals and durable-record mutations.
          round: 9
        - id: BR-1
          disposition: addressed
          note: Prior disposition retained; failed listing does not authorize Continue, and uncertain ownership prevents executable native resume.
          round: 9
        - id: BR-2
          disposition: addressed
          note: Prior disposition retained; lifecycle operations retain exact process-identity checks. Listed-session attachment follows the explicitly revised authority contract.
          round: 9
        - id: BR-3
          disposition: addressed
          note: Prior disposition retained; removed policy admission is not reintroduced into Continue.
          round: 9
        - id: BR-4
          disposition: addressed
          note: TerminalLease retains the fixed 256-stripe namespace; its bound regression passes.
          round: 9
        - id: BR-5
          disposition: addressed
          note: Prior disposition retained; bounded regular-metadata regressions pass.
          round: 9
        - id: BR-10
          disposition: addressed
          note: README documents observation-only startup, listed-live Continue, untouched lifecycle records, recovery choices, and accepted compatibility limits; these agree with the inspected implementation.
          round: 9
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: false
    - "n": 10
      timestamp: "2026-10-13T23:12:54-07:00"
      dispatched: "2026-10-13T23:10:32-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Retained disposition: failed listing does not authorize Continue; current uncertainty and listing-budget regressions passed.'
          round: 10
        - id: BR-2
          disposition: addressed
          note: Retained disposition under the superseding listed-live attach contract; process-dependent cold-path verification remains blocked by denied /bin/ps.
          round: 10
        - id: BR-3
          disposition: addressed
          note: Policy/config admission readers were removed; retained readiness uses bounded regular-file reads. Shared reader regressions passed.
          round: 10
        - id: BR-4
          disposition: addressed
          note: TerminalLease retains 256 permanent stripes; canonical-path and fixed-bound tests passed.
          round: 10
        - id: BR-5
          disposition: addressed
          note: Capture-witness reads use osfs.ReadRegularMetadata; FIFO, oversize and cancellation regressions passed. The superseded admission-config reader is removed.
          round: 10
        - id: BR-9
          disposition: addressed
          note: Public Continue enters helper-only attachment before lifecycle adoption. Targeted shape, cleanup, listing and registration regressions passed; console/public-binary verification remains environment-blocked.
          round: 10
        - id: BR-10
          disposition: addressed
          note: README opening-Couch, attachment, recovery and retention passages match startup.go, warm_attach.go, listing admission and removal of Couch-started maintenance; atlas/couch.md documents the same boundaries.
          round: 10
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: false
    - actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 95a2b9fe9b8418fad792941b0f09337bd8f7d13d
        tracker_head: bd7678924fab40f33459c29db09b0129278b8e08
        branch_point: c07a31921d66b891e36ae300e266bd7794bede3e
        since: "2026-10-13T08:57:28-07:00"
        until: "2026-10-14T15:15:48-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-13T08:57:28-07:00"
              end: "2026-10-14T15:15:48-07:00"
              began: "2026-10-13T08:57:28-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 4315c85cb45b1acd24b3f2fa78a5d8c68861146b607031db85c7475363bf2a4e
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 44806115
              windows:
                - start: "2026-10-13T08:57:28-07:00"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a13604-7f67-7c71-8862-d652ff00507c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: b41b7a545560a75e22708b5d1399ed0c43c186a6cae81a5f61981dddcb7c3c26
              started: "2026-10-13T16:07:06.952Z"
              spawned: true
              bytes: 687525
              windows:
                - start: "2026-10-13T16:07:06.952Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a1361a-aa08-77d0-8a39-fcac18d7aec7
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 048dd253370c5d447ca62a2e36eea18b6389d43cdaf73fa89d7cfafd6434d5f5
              started: "2026-10-13T16:31:19.827Z"
              spawned: false
              bytes: 782141
              windows:
                - start: "2026-10-13T16:31:19.827Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a1361e-8991-7f03-8a7b-3d5973745b61
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 92b2c39dd8e4cb64ba521e35394d47eca35dfc839b69df5ad0f8afee6011c7f5
              started: "2026-10-13T16:35:33.749Z"
              spawned: false
              bytes: 538951
              windows:
                - start: "2026-10-13T16:35:33.749Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a13620-2019-7901-9606-4ef4c7d27b1c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2df34f8d5aa5d7b6c236ce7bc2e70e77e4c35f5180e6f7addab471e95be82791
              started: "2026-10-13T16:37:17.75Z"
              spawned: false
              bytes: 541231
              windows:
                - start: "2026-10-13T16:37:17.75Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a13620-de40-7632-a9af-061a59f0a52b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a3f94580afaea679dbf2cecb0237c4d6ec3d2ae1ef8542ff997cd18a35b3043f
              started: "2026-10-13T16:38:06.405Z"
              spawned: false
              bytes: 521917
              windows:
                - start: "2026-10-13T16:38:06.405Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a13883-ab89-7992-ab0c-79b093c13b95
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 750bd23d4285903b71f354129fa6927c5a9776752ffc18e2b00d5ee3b3a5ed3f
              started: "2026-10-14T03:45:15.863Z"
              spawned: false
              bytes: 831906
              windows:
                - start: "2026-10-14T03:45:15.863Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a13884-4286-73c2-b7b2-17540c9cac4a
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ee61a169ede4b6e959c5a65aee80abf76ddc3a35ff0c484a0eada3b4c77eca0a
              started: "2026-10-14T03:45:54.508Z"
              spawned: false
              bytes: 492551
              windows:
                - start: "2026-10-14T03:45:54.508Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a13b23-d0ac-7b81-a844-510e6c4484bd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: 889118e7c9e65c9280146d928c7c80e9b620848b85d4d58f22c63e11ae063f68
              started: "2026-10-14T15:59:25.5Z"
              spawned: true
              bytes: 5153763
              windows:
                - start: "2026-10-14T15:59:25.5Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a13b23-eb0b-7642-9e0f-496f67490264
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: cc6243fd7178ee354ca679c48f70fe4c573eb09eba800d5f94831a0a98f49efc
              started: "2026-10-14T15:59:32.235Z"
              spawned: true
              bytes: 2730708
              windows:
                - start: "2026-10-14T15:59:32.235Z"
                  end: "2026-10-14T15:15:48-07:00"
            - harness: codex
              id: 01a13b24-1965-7260-926a-536a068a83f7
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: cf54076e00d5ae43faa20b726f2b9ce7283462d08b0fd8a00dc7ec39cbe8e317
              started: "2026-10-14T15:59:44.113Z"
              spawned: true
              bytes: 3340012
              windows:
                - start: "2026-10-14T15:59:44.113Z"
                  end: "2026-10-14T15:15:48-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 75.37058333333333
        cuts:
            - boundary: M1
              dispatched: "2026-10-13T10:58:01-07:00"
              finalized: "2026-10-13T11:01:13-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T11:30:13-07:00"
              finalized: "2026-10-13T11:34:12-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T12:03:38-07:00"
              finalized: "2026-10-13T12:06:16-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-13T13:20:56-07:00"
              finalized: "2026-10-13T13:26:16-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-13T13:34:13-07:00"
              finalized: "2026-10-13T13:39:34-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-13T13:44:33-07:00"
              finalized: "2026-10-13T13:49:02-07:00"
              finalizing: true
            - boundary: M1
              dispatched: "2026-10-13T21:37:49-07:00"
              finalized: "2026-10-13T21:42:34-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T22:08:52-07:00"
              finalized: "2026-10-13T22:11:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T22:56:16-07:00"
              finalized: "2026-10-13T23:03:06-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T23:10:32-07:00"
              finalized: "2026-10-13T23:12:54-07:00"
              finalizing: false
      "n": 11
      timestamp: "2026-10-14T15:20:38-07:00"
      dispatched: "2026-10-14T15:16:46-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Positive-listing classification and failed-list tests preserve uncertainty; warm attachment requires fresh positive evidence.
          round: 11
        - id: BR-2
          disposition: addressed
          note: Agent-generation authority is no longer an attachment prerequisite under the revised contract; exact process checks remain on retirement and dead-client repair.
          round: 11
        - id: BR-3
          disposition: addressed
          note: Policy admission reads were removed; focused cold-launch, readiness and terminal-handoff checks passed.
          round: 11
        - id: BR-4
          disposition: addressed
          note: Terminal leases retain the fixed 256-stripe namespace; session-lock tests passed.
          round: 11
        - id: BR-5
          disposition: addressed
          note: Retained metadata consumers use bounded readers; the full osfs and artifactpath suites passed.
          round: 11
        - id: BR-9
          disposition: addressed
          note: Public warm attachment bypasses lifecycle adoption; executor tests cover listed record shapes, independent claims and helper-only cleanup.
          round: 11
        - id: BR-10
          disposition: addressed
          note: README.md:395 and README.md:804 document explicit hubs, background Continue attachment and failure retry, consistent with startup and menu wiring.
          round: 11
      findings:
        - kind: defect
          id: BR-11
          severity: Important
          title: New startup integration regression fails reproducibly
          detail: 'cmd/internal/couchcmd/startup_warm_pass_test.go:179 times out awaiting two failed attachment attempts and their visible Continue failure reason. It failed in the focused multi-package run and again with go test ./cmd/internal/couchcmd -run ''^TestStartupWarmPassPaintsAndAcceptsInputWhileMixedAttachmentsBlock$'' -count=1 -v -timeout=65s (31.542s, exit 1). Separate the retry-count and rendered-reason assertions to expose which invariant fails, correct the runtime or fixture accordingly, and rerun through the later repair and explicit-retry assertions. ARCH-ORDER: the claimed startup sequence currently lacks passing end-to-end evidence.'
          family: reproducible-boundary-acceptance
          round: 11
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: false
    - actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 9a167f1fa0667715bd585b87dcaf7d65111c5487
        tracker_head: df8113b8fae458b509e76e3fbd0c703442505e9c
        branch_point: c07a31921d66b891e36ae300e266bd7794bede3e
        since: "2026-10-13T08:57:28-07:00"
        until: "2026-10-14T16:21:58-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-13T08:57:28-07:00"
              end: "2026-10-14T16:21:58-07:00"
              began: "2026-10-13T08:57:28-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: c044872a0fafd62010915f4dd7cefd26a5c6e6c515e5f3f41283ebe8609e0989
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 45449047
              windows:
                - start: "2026-10-13T08:57:28-07:00"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a13604-7f67-7c71-8862-d652ff00507c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: b41b7a545560a75e22708b5d1399ed0c43c186a6cae81a5f61981dddcb7c3c26
              started: "2026-10-13T16:07:06.952Z"
              spawned: true
              bytes: 687525
              windows:
                - start: "2026-10-13T16:07:06.952Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a1361a-aa08-77d0-8a39-fcac18d7aec7
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 048dd253370c5d447ca62a2e36eea18b6389d43cdaf73fa89d7cfafd6434d5f5
              started: "2026-10-13T16:31:19.827Z"
              spawned: false
              bytes: 782141
              windows:
                - start: "2026-10-13T16:31:19.827Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a1361e-8991-7f03-8a7b-3d5973745b61
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 92b2c39dd8e4cb64ba521e35394d47eca35dfc839b69df5ad0f8afee6011c7f5
              started: "2026-10-13T16:35:33.749Z"
              spawned: false
              bytes: 538951
              windows:
                - start: "2026-10-13T16:35:33.749Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a13620-2019-7901-9606-4ef4c7d27b1c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2df34f8d5aa5d7b6c236ce7bc2e70e77e4c35f5180e6f7addab471e95be82791
              started: "2026-10-13T16:37:17.75Z"
              spawned: false
              bytes: 541231
              windows:
                - start: "2026-10-13T16:37:17.75Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a13620-de40-7632-a9af-061a59f0a52b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a3f94580afaea679dbf2cecb0237c4d6ec3d2ae1ef8542ff997cd18a35b3043f
              started: "2026-10-13T16:38:06.405Z"
              spawned: false
              bytes: 521917
              windows:
                - start: "2026-10-13T16:38:06.405Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a13883-ab89-7992-ab0c-79b093c13b95
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 750bd23d4285903b71f354129fa6927c5a9776752ffc18e2b00d5ee3b3a5ed3f
              started: "2026-10-14T03:45:15.863Z"
              spawned: false
              bytes: 831906
              windows:
                - start: "2026-10-14T03:45:15.863Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a13884-4286-73c2-b7b2-17540c9cac4a
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ee61a169ede4b6e959c5a65aee80abf76ddc3a35ff0c484a0eada3b4c77eca0a
              started: "2026-10-14T03:45:54.508Z"
              spawned: false
              bytes: 492551
              windows:
                - start: "2026-10-14T03:45:54.508Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a13b23-d0ac-7b81-a844-510e6c4484bd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: ef3b2239b591af4f1c5b3f9f7195aad53a8700812c6c47e60620fa7b92489f62
              started: "2026-10-14T15:59:25.5Z"
              spawned: true
              bytes: 5569518
              windows:
                - start: "2026-10-14T15:59:25.5Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a13b23-eb0b-7642-9e0f-496f67490264
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: cc6243fd7178ee354ca679c48f70fe4c573eb09eba800d5f94831a0a98f49efc
              started: "2026-10-14T15:59:32.235Z"
              spawned: true
              bytes: 2730708
              windows:
                - start: "2026-10-14T15:59:32.235Z"
                  end: "2026-10-14T16:21:58-07:00"
            - harness: codex
              id: 01a13b24-1965-7260-926a-536a068a83f7
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: cf54076e00d5ae43faa20b726f2b9ce7283462d08b0fd8a00dc7ec39cbe8e317
              started: "2026-10-14T15:59:44.113Z"
              spawned: true
              bytes: 3340012
              windows:
                - start: "2026-10-14T15:59:44.113Z"
                  end: "2026-10-14T16:21:58-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 75.37058333333333
        cuts:
            - boundary: M1
              dispatched: "2026-10-13T10:58:01-07:00"
              finalized: "2026-10-13T11:01:13-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T11:30:13-07:00"
              finalized: "2026-10-13T11:34:12-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T12:03:38-07:00"
              finalized: "2026-10-13T12:06:16-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-13T13:20:56-07:00"
              finalized: "2026-10-13T13:26:16-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-13T13:34:13-07:00"
              finalized: "2026-10-13T13:39:34-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-13T13:44:33-07:00"
              finalized: "2026-10-13T13:49:02-07:00"
              finalizing: true
            - boundary: M1
              dispatched: "2026-10-13T21:37:49-07:00"
              finalized: "2026-10-13T21:42:34-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T22:08:52-07:00"
              finalized: "2026-10-13T22:11:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T22:56:16-07:00"
              finalized: "2026-10-13T23:03:06-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T23:10:32-07:00"
              finalized: "2026-10-13T23:12:54-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-14T15:16:46-07:00"
              finalized: "2026-10-14T15:20:38-07:00"
              finalizing: false
      "n": 12
      timestamp: "2026-10-14T16:27:24-07:00"
      dispatched: "2026-10-14T16:23:08-07:00"
      agent: codex
      dispose:
        - id: BR-11
          disposition: addressed
          note: The integration and delta-repaint regression passed ten repetitions (17.934s), including repair and explicit retry. Replacing VT screen projection with raw writes makes TestStartupWarmScreenReadsFailureAcrossDeltaRepaint fail; deadlines remain unchanged.
          round: 12
      findings:
        - kind: defect
          id: BR-12
          severity: Critical
          title: Automatic wrapper admission replays pending thread-store journals
          detail: 'cmd/internal/couchcmd/message_service.go:1224 calls writable GetThread during automatic wrapper admission; GetThread reaches withStoreLock and journal recovery (threadstore.go:205). A temporary overlay regression demonstrated recordChanged=true, journalRemoved=true, resolveErr=<nil>, without any requested lifecycle action. ARCH-PURPOSE / ARCH-ORDER: enforce preview-only storage access across automatic admission and use-time authority reads, sweeping the thread resolver and record/agent callbacks at lines 329 and 338; add pending-journal regressions that assert unchanged records, journals and retention enrollment.'
          family: observation-only-store-access
          round: 12
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 99035c559e3dd445a7ac568ebef91815a61079dd
        tracker_head: bfa42af5ea5d8cdf04e7986af0fda5bed5b6e5d7
        branch_point: c07a31921d66b891e36ae300e266bd7794bede3e
        since: "2026-10-13T08:57:28-07:00"
        until: "2026-10-14T16:39:58-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-13T08:57:28-07:00"
              end: "2026-10-14T16:39:58-07:00"
              began: "2026-10-13T08:57:28-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot2/pair
        sources:
            - harness: codex
              id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 07db6e12c729840dfc5e4c47bb41f9a1f24d03d107a8237af5087c244e611b71
              started: "2026-10-13T05:05:19.479Z"
              spawned: false
              bytes: 46076858
              windows:
                - start: "2026-10-13T08:57:28-07:00"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a13604-7f67-7c71-8862-d652ff00507c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: b41b7a545560a75e22708b5d1399ed0c43c186a6cae81a5f61981dddcb7c3c26
              started: "2026-10-13T16:07:06.952Z"
              spawned: true
              bytes: 687525
              windows:
                - start: "2026-10-13T16:07:06.952Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a1361a-aa08-77d0-8a39-fcac18d7aec7
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 048dd253370c5d447ca62a2e36eea18b6389d43cdaf73fa89d7cfafd6434d5f5
              started: "2026-10-13T16:31:19.827Z"
              spawned: false
              bytes: 782141
              windows:
                - start: "2026-10-13T16:31:19.827Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a1361e-8991-7f03-8a7b-3d5973745b61
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 92b2c39dd8e4cb64ba521e35394d47eca35dfc839b69df5ad0f8afee6011c7f5
              started: "2026-10-13T16:35:33.749Z"
              spawned: false
              bytes: 538951
              windows:
                - start: "2026-10-13T16:35:33.749Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a13620-2019-7901-9606-4ef4c7d27b1c
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 2df34f8d5aa5d7b6c236ce7bc2e70e77e4c35f5180e6f7addab471e95be82791
              started: "2026-10-13T16:37:17.75Z"
              spawned: false
              bytes: 541231
              windows:
                - start: "2026-10-13T16:37:17.75Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a13620-de40-7632-a9af-061a59f0a52b
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: a3f94580afaea679dbf2cecb0237c4d6ec3d2ae1ef8542ff997cd18a35b3043f
              started: "2026-10-13T16:38:06.405Z"
              spawned: false
              bytes: 521917
              windows:
                - start: "2026-10-13T16:38:06.405Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a13883-ab89-7992-ab0c-79b093c13b95
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: 750bd23d4285903b71f354129fa6927c5a9776752ffc18e2b00d5ee3b3a5ed3f
              started: "2026-10-14T03:45:15.863Z"
              spawned: false
              bytes: 831906
              windows:
                - start: "2026-10-14T03:45:15.863Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a13884-4286-73c2-b7b2-17540c9cac4a
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              digest: ee61a169ede4b6e959c5a65aee80abf76ddc3a35ff0c484a0eada3b4c77eca0a
              started: "2026-10-14T03:45:54.508Z"
              spawned: false
              bytes: 492551
              windows:
                - start: "2026-10-14T03:45:54.508Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a13b23-d0ac-7b81-a844-510e6c4484bd
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: ef3b2239b591af4f1c5b3f9f7195aad53a8700812c6c47e60620fa7b92489f62
              started: "2026-10-14T15:59:25.5Z"
              spawned: true
              bytes: 5569518
              windows:
                - start: "2026-10-14T15:59:25.5Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a13b23-eb0b-7642-9e0f-496f67490264
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: cc6243fd7178ee354ca679c48f70fe4c573eb09eba800d5f94831a0a98f49efc
              started: "2026-10-14T15:59:32.235Z"
              spawned: true
              bytes: 2730708
              windows:
                - start: "2026-10-14T15:59:32.235Z"
                  end: "2026-10-14T16:39:58-07:00"
            - harness: codex
              id: 01a13b24-1965-7260-926a-536a068a83f7
              cwd: /Users/xianxu/workspace/worktree/pair-slot2/pair
              parent_id: 01a133a3-eaec-77b3-8c1e-dc4ac6b41fbd
              digest: dcf26ef7ba000949ac4c87be468b24a1f030923e3e098845ec7160668034561f
              started: "2026-10-14T15:59:44.113Z"
              spawned: true
              bytes: 4064482
              windows:
                - start: "2026-10-14T15:59:44.113Z"
                  end: "2026-10-14T16:39:58-07:00"
        excluded_brain_sessions: 1
        excluded_brain_minutes: 75.37058333333333
        cuts:
            - boundary: M1
              dispatched: "2026-10-13T10:58:01-07:00"
              finalized: "2026-10-13T11:01:13-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T11:30:13-07:00"
              finalized: "2026-10-13T11:34:12-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T12:03:38-07:00"
              finalized: "2026-10-13T12:06:16-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-13T13:20:56-07:00"
              finalized: "2026-10-13T13:26:16-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-13T13:34:13-07:00"
              finalized: "2026-10-13T13:39:34-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-13T13:44:33-07:00"
              finalized: "2026-10-13T13:49:02-07:00"
              finalizing: true
            - boundary: M1
              dispatched: "2026-10-13T21:37:49-07:00"
              finalized: "2026-10-13T21:42:34-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T22:08:52-07:00"
              finalized: "2026-10-13T22:11:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T22:56:16-07:00"
              finalized: "2026-10-13T23:03:06-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-13T23:10:32-07:00"
              finalized: "2026-10-13T23:12:54-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-14T15:16:46-07:00"
              finalized: "2026-10-14T15:20:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-14T16:23:08-07:00"
              finalized: "2026-10-14T16:27:24-07:00"
              finalizing: false
      "n": 13
      timestamp: "2026-10-14T16:44:45-07:00"
      dispatched: "2026-10-14T16:41:22-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Remains disposed under the revised listing-based contract; uncertainty is retained in recovery admission.
          round: 13
        - id: BR-2
          disposition: addressed
          note: Exact process-generation checks remain in readiness and native-owner authority.
          round: 13
        - id: BR-3
          disposition: addressed
          note: Policy admission was removed; the previously corrected readiness reads remain bounded.
          round: 13
        - id: BR-4
          disposition: addressed
          note: Permanent terminal coordination remains bounded to 256 stripes.
          round: 13
        - id: BR-5
          disposition: addressed
          note: The corrected capture/config reader contracts remain intact; the new finding concerns newly contextualized ThreadStore authority reads.
          round: 13
        - id: BR-9
          disposition: addressed
          note: Listed-live Continue uses the shared warm attachment route; focused attachment and reattach tests pass.
          round: 13
        - id: BR-10
          disposition: addressed
          note: README.md documents startup attachment, retry, recovery actions and compatibility limits, consistent with the revised implementation.
          round: 13
        - id: BR-11
          disposition: addressed
          note: The rendered-cell startup oracle and mixed startup integration pass in the focused couchcmd run.
          round: 13
        - id: BR-12
          disposition: addressed
          note: All three broker callbacks use PreviewThreadContext. Global/local/routed journal and retention tests pass; restoring GetThread makes TestMessageThreadResolverLeavesPendingJournalUntouched fail.
          round: 13
      findings:
        - kind: defect
          id: BR-13
          severity: Critical
          title: Contextual broker previews still block indefinitely on persisted-file reads
          detail: 'PreviewThreadContext reaches os.ReadFile in threadstore_location.go:217 and global readPayload/readOptionalPayload in threadstore_layout.go:163/136. Context bounds lock acquisition but cannot interrupt these reads. Two controlled disposable-checkout regressions, covering routed legacy lookup and direct global record lookup, remained blocked beyond a 25ms deadline until a FIFO writer released them after 200ms; without that writer they cannot finish. This defeats automatic admission/use-time cancellation while holding the store lock (ARCH-CONSTRAINTS, ARCH-SECURE). This is the 3rd finding in family bounded-untrusted-artifact-reads: state and enforce one rule for every persisted byte read reachable from broker preview, enumerating manifest, routing record/archive and final record readers. Use bounded regular-file/no-follow/nonblocking reads with contextual cancellation, and add consumer regressions across global, local and routed backends.'
          family: bounded-untrusted-artifact-reads
          round: 13
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
---

# Gate ledger — pair#528 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T11:01:13-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `preserve-observation-uncertainty` Unknown terminal observations become live or actionable relaunch verdicts
  cmd/internal/couchtty/menu_reattach.go:373 promotes ReasonUnknown to ThreadLive whenever an attached-pass entry exists. cmd/internal/launcher/session_handoff_os.go:73-97 also collapses inspection errors into false usability, which cmd/internal/couchcore/agent_usability.go:7 converts to needs-relaunch. Preserve typed usable/unsafe/unknown outcomes through observers, classifiers, and overlays; test failed refresh after successful attachment and failed metadata inspection. ARCH-ORDER, ARCH-PURPOSE.
- **BR-2** [Critical] `exact-process-generation-authority` Agent usability accepts a recycled agent PID beneath the same server
  cmd/internal/launcher/terminal_agent.go:7-34 compares only the server generation; cmd/internal/launcher/session_handoff_os.go:205 supplies a readiness PID without its expected start identity. A replacement descendant reusing that PID passes stale readiness. Persist and validate the agent generation, with a regression changing only the agent identity while preserving PID and ancestry. ARCH-SECURE.
- **BR-3** [Important] `bounded-untrusted-artifact-reads` Policy and readiness reads can defeat the admission timeout
  cmd/internal/launcher/session_handoff_os.go:127 opens policy files without regular-file/nonblocking safeguards, and readiness reads at lines 193 and 423 are unbounded. A FIFO can hang refresh or admission indefinitely because cancellation cannot interrupt these reads and the coordinator joins the observer. Share the cold-check branch's bounded regular-file reader across these consumers and test FIFO and oversized inputs. ARCH-CONSTRAINTS, ARCH-SECURE.
- **BR-4** [Important] `durable-artifact-lifecycle-bound` Per-terminal coordination files accumulate without an aggregate lifecycle bound
  cmd/internal/artifactpath/session_policy.go:151-165 allocates one lock pathname per canonical socket, while cmd/internal/artifactpath/manifest.go:1460 permanently protects every such inode. Define a bounded coordination scheme or safe retirement protocol without unlinking active or awaited locks; document and measure any intentional permanent per-session cost. ARCH-FUNERAL.

## Round 2 — 2026-10-13T11:34:12-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Typed uncertainty reaches core projection and the attached-pass overlay. Focused tests pass; removing the unknown-state overlay guard or converting inspection errors to unsafe makes the respective regressions fail.
- BR-2 — addressed — Readiness records carry agent start identity and AgentBelongsToServer validates it. Removing the agent-identity comparison fails the unchanged-PID/ancestry regression.
- BR-3 — addressed — Policy and readiness consumers use readBoundedRegularMetadata. Its regressions pass; replacing it with os.ReadFile fails FIFO, oversized-input, symlink and cancellation cases. Additional consumers missed by the family sweep are reported separately below.
- BR-4 — addressed — TerminalLease now selects among 256 permanent stripes. The 10,000-socket regression passes and fails when full-hash per-socket lock paths are restored.

### Raised

- **BR-5** [Important] `bounded-untrusted-artifact-reads` Capture-witness and admission-config reads still permit indefinite blocking
  cmd/internal/storagegc/capture_metadata.go:142 opens events with blocking os.Open before its byte limit; a FIFO without a writer blocks capture preservation indefinitely, reproduced by a disposable regression probe. cmd/internal/launcher/session_handoff_os.go:468 also reads config.kdl with unbounded, non-cancellable os.ReadFile, so admission can remain stuck beyond its timeout while the coordinator waits for the observer. This is the 2nd finding in family bounded-untrusted-artifact-reads. Do NOT fix only these instances: enumerate every persisted-input read added by M1 and enforce one bounded, regular-file, nonblocking, cancellation-aware rule through a shared IO owner, with consumer-level FIFO and oversized-input regressions. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-PURPOSE.

## Round 3 — 2026-10-13T12:06:16-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Prior disposition retained; unknown-observation and reattach-overlay regressions pass.
- BR-2 — addressed — Prior disposition retained; AgentBelongsToServer checks both server and agent process generations.
- BR-3 — addressed — Prior disposition retained; policy and readiness consumers use the shared bounded reader, with passing FIFO regressions.
- BR-4 — addressed — Prior disposition retained; permanent coordination files remain bounded to 256 stripes, with passing lease tests.
- BR-5 — addressed — Enumerated metadata consumers use osfs.ReadRegularMetadata. Consumer FIFO, oversized-input and cancellation tests pass; substituting os.ReadFile in a disposable checkout makes the regressions fail. Both disposable checkouts were removed.

## Round 4 — 2026-10-13T13:26:16-07:00 (codex) — BLOCKED

### Raised

- **BR-6** [Critical] `exact-process-generation-authority` Native ownership substitutes the latest tag-level launch for the live wrapper's launch
  cmd/internal/launcher/recovery_native_owner.go:85 uses CurrentLaunch(scope, tag, agent) without associating that launch with the observed wrapper PID/start identity. A regression retaining the old wrapper while appending a newer launch targeting another conversation incorrectly returns success. This is the 2nd finding in family `exact-process-generation-authority`. Earlier rounds fixed instances. Do NOT fix only this instance: state and enforce the rule that every native-ownership assertion must bind to the exact observed process generation; sweep preflight, Pair admission, source exemptions, and their doubles. Missing association must remain unresolved. ARCH-SECURE, ARCH-ORDER, ARCH-MOCK.
- **BR-7** [Critical] `recovery-primary-action-routing` The advertised primary Relaunch action never opens the new recovery preview
  cmd/internal/couchtty/menu.go:864 handles only the new resume-candidates item, appended separately at menu_switchagent.go:204. Unsafe-row Enter/mouse still select relaunch at menu.go:803 and open the old confirmation through menu_slot.go:34; the old executor requires established binding. The primary-route regression fails with the confirmation frame and no preview effect. Route Enter, mouse, and the unsafe-row Relaunch item through the same evidence preview while preserving ordinary healthy relaunch behavior. ARCH-PURPOSE.

## Round 5 — 2026-10-13T13:39:34-07:00 (codex) — BLOCKED

### Disposed

- BR-6 — addressed — Ownership now resolves the launch ordinal associated with the observed wrapper's live readiness generation. Checked preflight, final Couch check, Pair admission, source exemptions, and the stateful double. TestRecoveryNativeOwnerOldWrapperCannotBorrowNewLaunch passes at head and fails when CurrentLaunch receives the entire tag ledger again.
- BR-7 — addressed — Unsafe Enter, mouse, Relaunch item, and restart shortcut share the recovery preview route. Production reducer tests pass; disabling the shared routing predicate fails all four cases. Healthy relaunch retains its existing confirmation.

### Raised

- **BR-8** [Important] `external-contract-conformance` Recovery-specific native parsing is outside live conformance
  ARCH-MOCK: cmd/internal/sessioninventory/recovery_candidate.go:192 introduces eligibility based on Claude entrypoint and related provenance, but conformance.go:52 checks only the existing scanners; conformance_live_test.go:21 never invokes the recovery parser or collector. Native changes can therefore leave conformance green while recovery eligibility or tail evidence stops working. Extend the existing read-only conformance path through the production recovery parser, covering provenance, cwd, activity and authored tails, with explicit unsupported/absent-sample outcomes and the existing contract-change/monthly cadence.

## Round 6 — 2026-10-13T13:49:02-07:00 (codex) — passed

### Disposed

- BR-8 — addressed — conformance_live_test.go:24 invokes RunRecoveryConformance, which exercises bounded production parsing and checks provenance, cwd, activity, and authored tails. Removing the provenance rejection at conformance.go:281 made the Codex drift and Claude missing/changed-entrypoint regressions fail. Unsupported and absent samples report skip explicitly; atlas/session-identity.md preserves contract-change/monthly cadence.
- BR-6 — addressed — recovery_native_owner.go resolves the readiness-proved launch ordinal only after associating the exact live agent generation with its wrapper. Exact-generation and source-exemption regressions passed.
- BR-7 — addressed — Unsafe primary activation and relaunch routes use the recovery chooser. TestReviewNeedsRelaunchPrimaryOpensRecovery and the targeted UI routing tests passed.

## Round 7 — 2026-10-13T21:42:34-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — The approved replacement contract distinguishes positive listing from failed observation. TestListedLiveIsContinueAuthority and TestAttributedOwnerCannotResumeDisk pass; unknown observations do not authorize Continue.
- BR-2 — addressed — Exact agent-generation comparison remains in terminal_agent.go. TestTerminalAgentRequiresExactServerAndLiveDescendant passes and fails when that comparison is removed.
- BR-3 — addressed — Policy reads were removed from admission. Remaining readiness reads in session_handoff_os.go use contextual ReadRegularMetadata; bounded metadata regressions pass.
- BR-4 — addressed — session_lock.go preserves the 256-stripe bound and policy publication is removed. TestTerminalLeasePathsHaveFixedBound passes and fails when full socket hashes replace stripe keys.
- BR-5 — addressed — Admission-config reads were removed. Capture-witness reads still use ReadRegularMetadata in recovery_evidence_os.go:141; bounded metadata and wrong-witness regressions pass.

### Raised

- **BR-9** [Critical] `recovery-primary-action-routing` Continue is offered where the resume executor still refuses attachment
  actionableinventory.go:494 now projects any positively listed session as Continue, but ResumeContextWith calls DecideResume, whose resume.go:186 preconditions still reject missing/incomplete profiles, unsupported saved agents, and unavailable working paths. A temporary review regression reproduced all four refusals. This is the 2nd finding in family recovery-primary-action-routing: state and enforce the rule that every offered Continue reaches a warm attach using listed-session authority; sweep all downstream warm-path gates rather than patching one profile check. ARCH-PURPOSE.
- **BR-10** [Important] `user-facing-contract-documentation` README update appears missing for changed Couch startup and action semantics
  startup.go:73 removes automatic launch/resume on ordinary Couch startup, and menu.go:1446 changes visible action labels. README.md:389 remains unchanged throughout the pinned range. Document observation-only startup, explicit action selection, and the M1 recovery limitations alongside the existing usage instructions; atlas/couch.md already describes these changes.

## Round 8 — 2026-10-13T22:11:38-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Retained disposition. Failed listing does not authorize Continue; the revised contract explicitly supersedes policy-based terminal usability.
- BR-2 — addressed — Retained disposition. Exact-generation helpers remain; policy/readiness-based warm admission was explicitly removed by the approved replacement design.
- BR-3 — addressed — Retained disposition. Policy reads were removed; remaining admission readiness reads use bounded ReadRegularMetadata.
- BR-4 — addressed — TerminalLease retains the aggregate bound of 256 permanent lock stripes in artifactpath/session_lock.go.
- BR-5 — addressed — Retained disposition. Shared bounded metadata reading remains; the obsolete admission-config reader was removed.
- BR-9 — not-addressed — ARCH-PURPOSE / ARCH-ORDER; family recovery-primary-action-routing remains open. Metadata regressions pass and fail with the old eligibility function restored, but resume.go:556 still calls clearLifecycleDebris, whose lifecycledebris.go:87 refuses an outstanding start owned by a live supervisor. A temporary regression constructed a valid StartClaim, positively listed its exact detached session, verified classification offers Continue, then observed ResumeContextWith(WarmOnly:true) fail with resume-starting before attachment. Separate listed-session attachment from lifecycle retirement; enumerate and test all accepted record shapes through the executor rather than fixing another individual gate.
- BR-10 — addressed — README.md:395 now documents observation-only startup and explicit actions; the recovery section documents Starting, restricted Fresh execution, legacy attachment race and unavailable messaging. These passages agree with startup.go:73, the menu projection and atlas/couch.md. This is a prose correction and requires no wording-presence regression.

## Round 9 — 2026-10-13T23:03:06-07:00 (codex) — passed

### Disposed

- BR-9 — addressed — Public Continue bypasses lifecycle adoption and cold prerequisites. The record-shape regression passes at HEAD and fails when the helper-only route is disabled, reproducing refusals and durable-record mutations.
- BR-1 — addressed — Prior disposition retained; failed listing does not authorize Continue, and uncertain ownership prevents executable native resume.
- BR-2 — addressed — Prior disposition retained; lifecycle operations retain exact process-identity checks. Listed-session attachment follows the explicitly revised authority contract.
- BR-3 — addressed — Prior disposition retained; removed policy admission is not reintroduced into Continue.
- BR-4 — addressed — TerminalLease retains the fixed 256-stripe namespace; its bound regression passes.
- BR-5 — addressed — Prior disposition retained; bounded regular-metadata regressions pass.
- BR-10 — addressed — README documents observation-only startup, listed-live Continue, untouched lifecycle records, recovery choices, and accepted compatibility limits; these agree with the inspected implementation.

## Round 10 — 2026-10-13T23:12:54-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Retained disposition: failed listing does not authorize Continue; current uncertainty and listing-budget regressions passed.
- BR-2 — addressed — Retained disposition under the superseding listed-live attach contract; process-dependent cold-path verification remains blocked by denied /bin/ps.
- BR-3 — addressed — Policy/config admission readers were removed; retained readiness uses bounded regular-file reads. Shared reader regressions passed.
- BR-4 — addressed — TerminalLease retains 256 permanent stripes; canonical-path and fixed-bound tests passed.
- BR-5 — addressed — Capture-witness reads use osfs.ReadRegularMetadata; FIFO, oversize and cancellation regressions passed. The superseded admission-config reader is removed.
- BR-9 — addressed — Public Continue enters helper-only attachment before lifecycle adoption. Targeted shape, cleanup, listing and registration regressions passed; console/public-binary verification remains environment-blocked.
- BR-10 — addressed — README opening-Couch, attachment, recovery and retention passages match startup.go, warm_attach.go, listing admission and removal of Couch-started maintenance; atlas/couch.md documents the same boundaries.

## Round 11 — 2026-10-14T15:20:38-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Positive-listing classification and failed-list tests preserve uncertainty; warm attachment requires fresh positive evidence.
- BR-2 — addressed — Agent-generation authority is no longer an attachment prerequisite under the revised contract; exact process checks remain on retirement and dead-client repair.
- BR-3 — addressed — Policy admission reads were removed; focused cold-launch, readiness and terminal-handoff checks passed.
- BR-4 — addressed — Terminal leases retain the fixed 256-stripe namespace; session-lock tests passed.
- BR-5 — addressed — Retained metadata consumers use bounded readers; the full osfs and artifactpath suites passed.
- BR-9 — addressed — Public warm attachment bypasses lifecycle adoption; executor tests cover listed record shapes, independent claims and helper-only cleanup.
- BR-10 — addressed — README.md:395 and README.md:804 document explicit hubs, background Continue attachment and failure retry, consistent with startup and menu wiring.

### Raised

- **BR-11** [Important] `reproducible-boundary-acceptance` (kind: defect) New startup integration regression fails reproducibly
  cmd/internal/couchcmd/startup_warm_pass_test.go:179 times out awaiting two failed attachment attempts and their visible Continue failure reason. It failed in the focused multi-package run and again with go test ./cmd/internal/couchcmd -run '^TestStartupWarmPassPaintsAndAcceptsInputWhileMixedAttachmentsBlock$' -count=1 -v -timeout=65s (31.542s, exit 1). Separate the retry-count and rendered-reason assertions to expose which invariant fails, correct the runtime or fixture accordingly, and rerun through the later repair and explicit-retry assertions. ARCH-ORDER: the claimed startup sequence currently lacks passing end-to-end evidence.

## Round 12 — 2026-10-14T16:27:24-07:00 (codex) — BLOCKED

### Disposed

- BR-11 — addressed — The integration and delta-repaint regression passed ten repetitions (17.934s), including repair and explicit retry. Replacing VT screen projection with raw writes makes TestStartupWarmScreenReadsFailureAcrossDeltaRepaint fail; deadlines remain unchanged.

### Raised

- **BR-12** [Critical] `observation-only-store-access` (kind: defect) Automatic wrapper admission replays pending thread-store journals
  cmd/internal/couchcmd/message_service.go:1224 calls writable GetThread during automatic wrapper admission; GetThread reaches withStoreLock and journal recovery (threadstore.go:205). A temporary overlay regression demonstrated recordChanged=true, journalRemoved=true, resolveErr=<nil>, without any requested lifecycle action. ARCH-PURPOSE / ARCH-ORDER: enforce preview-only storage access across automatic admission and use-time authority reads, sweeping the thread resolver and record/agent callbacks at lines 329 and 338; add pending-journal regressions that assert unchanged records, journals and retention enrollment.

## Round 13 — 2026-10-14T16:44:45-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Remains disposed under the revised listing-based contract; uncertainty is retained in recovery admission.
- BR-2 — addressed — Exact process-generation checks remain in readiness and native-owner authority.
- BR-3 — addressed — Policy admission was removed; the previously corrected readiness reads remain bounded.
- BR-4 — addressed — Permanent terminal coordination remains bounded to 256 stripes.
- BR-5 — addressed — The corrected capture/config reader contracts remain intact; the new finding concerns newly contextualized ThreadStore authority reads.
- BR-9 — addressed — Listed-live Continue uses the shared warm attachment route; focused attachment and reattach tests pass.
- BR-10 — addressed — README.md documents startup attachment, retry, recovery actions and compatibility limits, consistent with the revised implementation.
- BR-11 — addressed — The rendered-cell startup oracle and mixed startup integration pass in the focused couchcmd run.
- BR-12 — addressed — All three broker callbacks use PreviewThreadContext. Global/local/routed journal and retention tests pass; restoring GetThread makes TestMessageThreadResolverLeavesPendingJournalUntouched fail.

### Raised

- **BR-13** [Critical] `bounded-untrusted-artifact-reads` (kind: defect) Contextual broker previews still block indefinitely on persisted-file reads
  PreviewThreadContext reaches os.ReadFile in threadstore_location.go:217 and global readPayload/readOptionalPayload in threadstore_layout.go:163/136. Context bounds lock acquisition but cannot interrupt these reads. Two controlled disposable-checkout regressions, covering routed legacy lookup and direct global record lookup, remained blocked beyond a 25ms deadline until a FIFO writer released them after 200ms; without that writer they cannot finish. This defeats automatic admission/use-time cancellation while holding the store lock (ARCH-CONSTRAINTS, ARCH-SECURE). This is the 3rd finding in family bounded-untrusted-artifact-reads: state and enforce one rule for every persisted byte read reachable from broker preview, enumerating manifest, routing record/archive and final record readers. Use bounded regular-file/no-follow/nonblocking reads with contextual cancellation, and add consumer regressions across global, local and routed backends.

## Open findings

- **BR-13** [Critical] `bounded-untrusted-artifact-reads` (kind: defect) Contextual broker previews still block indefinitely on persisted-file reads
