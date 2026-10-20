---
gate: boundary-review
issue: 695
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-19T07:40:57-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Validate() is said to fill TeamIdentity.Scope, but §3 declares it with a value receiver returning only error
          detail: |-
            This is the 3rd finding in this family. The rule: a Revision that changes a contract must also edit its signature in §2/§3, not just the prose. Here, use a constructor (for example NewTeamAddress) or have Validate return the resolved LifecycleAddress, so that Key() stays pure over a resolved Scope.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: contract-shape-consistency
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 0f2ec6a2-c6fa-4c46-b43e-34ad69f5e40d
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: milestone-review
            round: 1
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/a92d2faa5f12cfb7
            reviewed_head: 2649a9775a4f0163d8aeb5c41450f8865dc4210e
            slot: pair:1
            requested_at: "2026-10-19T14:20:40.378919Z"
          phase: finished
          pid: 8399
          started_at: "2026-10-19T14:20:40.604433Z"
          finished_at: "2026-10-19T14:40:49.656073Z"
          session_id: e1eea7b2-f8df-41e6-8cb7-94f6cc4cdb0f
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 2649a9775a4f0163d8aeb5c41450f8865dc4210e
        tracker_head: 133ed1d461a510e666d449427646fe8bf0f28b0c
        branch_point: 26df2fe40f70ab2b9e3d1bfd88c3b243ee42e02e
        since: "2026-10-18T22:57:15-07:00"
        until: "2026-10-19T07:19:42-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T22:57:15-07:00"
              end: "2026-10-19T07:19:42-07:00"
              began: "2026-10-18T22:57:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: claude
              id: 0f8622e1-c463-4612-8427-0339d0387f2b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: ca3da0965e8961d1c61c665731c4cd42b4e97772c869410aef3541505e4f55ce
              started: "2026-10-19T13:36:29.283Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-19T13:36:29.283Z"
                  end: "2026-10-19T07:19:42-07:00"
            - harness: claude
              id: 507d0585-c54a-4834-8abd-9a52d7f9da76
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 2abf80a28db0c2e0a5d5c81014f75e529d37539aa50bd3e42d6cbbb0a611eb2b
              started: "2026-10-19T02:49:17.103Z"
              spawned: false
              bytes: 8488448
              windows:
                - start: "2026-10-18T22:57:15-07:00"
                  end: "2026-10-19T07:19:42-07:00"
            - harness: claude
              id: 6fa6fea2-8902-4f5c-9b2a-6548cf879f30
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: faaf9c38d70a8a84ae00d705b1eef74c2570dd281bc35df426c3f6a837398123
              started: "2026-10-19T13:30:41.221Z"
              spawned: false
              bytes: 614766
              windows:
                - start: "2026-10-19T13:30:41.221Z"
                  end: "2026-10-19T07:19:42-07:00"
            - harness: claude
              id: 84fed9bb-2c33-495b-a2ac-b1c365c8f242
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 45246414c4d539667e82f0898e93805a8f3c200e359a018f9d53d1c10c45e757
              started: "2026-10-19T13:36:13.159Z"
              spawned: false
              bytes: 572779
              windows:
                - start: "2026-10-19T13:36:13.159Z"
                  end: "2026-10-19T07:19:42-07:00"
            - harness: claude
              id: 98afc721-0259-4ecb-a3aa-286966a8fa6b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a6426bb7d4eda2cbe641637263404af0ffc0e861b43796863689e58ebf3b5ab
              started: "2026-10-19T13:35:01.399Z"
              spawned: false
              bytes: 418152
              windows:
                - start: "2026-10-19T13:35:01.399Z"
                  end: "2026-10-19T07:19:42-07:00"
            - harness: claude
              id: b5833248-4bc3-4582-a4e6-6c13f3737a1f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 00e83c9ffe05ea6321734652c9f1f96a46f491047bfc59de33ed5ac5eff504d4
              started: "2026-10-19T13:32:47.197Z"
              spawned: false
              bytes: 604367
              windows:
                - start: "2026-10-19T13:32:47.197Z"
                  end: "2026-10-19T07:19:42-07:00"
            - harness: claude
              id: ccfc2441-0f65-4249-8017-65b2eb08cc24
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5b9b248858b400579af6244ff35f2baa03f8f0e3b41b50d9804fd87bf8a87314
              started: "2026-10-19T13:34:27.337Z"
              spawned: false
              bytes: 574614
              windows:
                - start: "2026-10-19T13:34:27.337Z"
                  end: "2026-10-19T07:19:42-07:00"
      "n": 2
      timestamp: "2026-10-19T07:40:57-07:00"
      dispatched: "2026-10-19T07:20:37-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      findings:
        - kind: defect
          id: BR-2
          severity: Critical
          title: 'artifactpath enforcement tests fail at head (pass at base): unclassified artifactpath import and dead constructors'
          detail: TestProductionArtifactReferencesAreExactlyClassified flags couchcore/transcript_bind.go importing artifactpath; TestNoProductionSymbolIsReferencedOnlyByTests flags NewSlotAddress and NewTeamAddress (lifecycleaddress.go) as test-only. Classify transcript_bind.go correctly, and wire or allowlist the constructors with a reason.
          family: repo-invariant-tests-red
          round: 2
        - kind: defect
          id: BR-3
          severity: Important
          title: replaceSlotCurrent marshals ThreadRecord directly and drops Unknown members on recovery rewrites
          detail: slotrecovery.go:148 uses json.MarshalIndent(next), and Unknown is tagged json:"-". Recovered survivors (slotrecovery.go:524/594, resume_route.go:292, recovery_prepared.go:497) lose a newer build's fields; a probe test reproduced it. The guard TestThreadRecordWritersUseThePreservingEncoder only matches toPersistedThreadRecord, so it misses this writer. Use encodeThreadRecord, widen the guard, and add a recovery round-trip test.
          family: preserving-encoder-single-writer
          round: 2
        - kind: defect
          id: BR-4
          severity: Important
          title: TranscriptBind.Note is never read in production; the spec's row note never reaches the operator
          detail: resumeEvidence and resumeContextWith keep only Resolution and Store. The "ledger unreadable/missing; resumed the stored binding" note required by P2 is dropped. Surface it at the row or resume result and test it there, or record a deferral Revision.
          family: field-without-consumer
          round: 2
        - kind: defect
          id: BR-5
          severity: Important
          title: Decode-time Transcript.Validate refuses the whole record instead of "Start fresh, invalid stored binding"
          detail: threadrecord.Validate (record.go:149) runs the semantic location check (NativeRootNames and RootTranscriptID, this build's catalogues) during DecodePersisted. A hand-edited binding, or a newer build's binding with a new root or agent, makes the whole record undecodable, which contradicts the P2 spec and §2a tolerance. ResumeTranscriptInvalid becomes nearly unreachable. Move the location check into Bind.
          family: decode-strictness-beyond-types
          round: 2
        - kind: defect
          id: BR-6
          severity: Minor
          title: lifecycleFailureTargetKey does not delegate to LifecycleAddress.Key() as the plan claims; NewSlotAddress does not canonicalize its root
          detail: The code shares slotLifecycleKey instead, which is fine in substance. Update the plan, or wire Key(), and document that the caller must canonicalize the root.
          family: plan-code-drift
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 432b3500-2296-4c6e-8f70-ae2ce14871da
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/eb4c71944b5a22f4
            reviewed_head: b5c0b8135c7873cf1a1ff198227c29bb142d34db
            slot: pair:1
            requested_at: "2026-10-19T14:55:36.449084Z"
          phase: finished
          pid: 2534
          started_at: "2026-10-19T14:55:36.560935Z"
          finished_at: "2026-10-19T15:17:36.530325Z"
          session_id: 6059e620-ff97-4232-b53f-d86c71cc6d68
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: b5c0b8135c7873cf1a1ff198227c29bb142d34db
        tracker_head: 0bb0f8146085db074347d7a650083a0f9fd8ec18
        branch_point: 26df2fe40f70ab2b9e3d1bfd88c3b243ee42e02e
        since: "2026-10-18T22:57:15-07:00"
        until: "2026-10-19T07:55:04-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T22:57:15-07:00"
              end: "2026-10-19T07:55:04-07:00"
              began: "2026-10-18T22:57:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: claude
              id: 0f8622e1-c463-4612-8427-0339d0387f2b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: ca3da0965e8961d1c61c665731c4cd42b4e97772c869410aef3541505e4f55ce
              started: "2026-10-19T13:36:29.283Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-19T13:36:29.283Z"
                  end: "2026-10-19T07:55:04-07:00"
            - harness: claude
              id: 507d0585-c54a-4834-8abd-9a52d7f9da76
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 148122ff865cc05fced695f54b93fe62e30deb63f2f9dd193b5c6af44849eca4
              started: "2026-10-19T02:49:17.103Z"
              spawned: false
              bytes: 9032267
              windows:
                - start: "2026-10-18T22:57:15-07:00"
                  end: "2026-10-19T07:55:04-07:00"
            - harness: claude
              id: 6fa6fea2-8902-4f5c-9b2a-6548cf879f30
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: faaf9c38d70a8a84ae00d705b1eef74c2570dd281bc35df426c3f6a837398123
              started: "2026-10-19T13:30:41.221Z"
              spawned: false
              bytes: 614766
              windows:
                - start: "2026-10-19T13:30:41.221Z"
                  end: "2026-10-19T07:55:04-07:00"
            - harness: claude
              id: 84fed9bb-2c33-495b-a2ac-b1c365c8f242
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 45246414c4d539667e82f0898e93805a8f3c200e359a018f9d53d1c10c45e757
              started: "2026-10-19T13:36:13.159Z"
              spawned: false
              bytes: 572779
              windows:
                - start: "2026-10-19T13:36:13.159Z"
                  end: "2026-10-19T07:55:04-07:00"
            - harness: claude
              id: 98afc721-0259-4ecb-a3aa-286966a8fa6b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a6426bb7d4eda2cbe641637263404af0ffc0e861b43796863689e58ebf3b5ab
              started: "2026-10-19T13:35:01.399Z"
              spawned: false
              bytes: 418152
              windows:
                - start: "2026-10-19T13:35:01.399Z"
                  end: "2026-10-19T07:55:04-07:00"
            - harness: claude
              id: b5833248-4bc3-4582-a4e6-6c13f3737a1f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 00e83c9ffe05ea6321734652c9f1f96a46f491047bfc59de33ed5ac5eff504d4
              started: "2026-10-19T13:32:47.197Z"
              spawned: false
              bytes: 604367
              windows:
                - start: "2026-10-19T13:32:47.197Z"
                  end: "2026-10-19T07:55:04-07:00"
            - harness: claude
              id: ccfc2441-0f65-4249-8017-65b2eb08cc24
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5b9b248858b400579af6244ff35f2baa03f8f0e3b41b50d9804fd87bf8a87314
              started: "2026-10-19T13:34:27.337Z"
              spawned: false
              bytes: 574614
              windows:
                - start: "2026-10-19T13:34:27.337Z"
                  end: "2026-10-19T07:55:04-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-19T07:20:37-07:00"
              finalized: "2026-10-19T07:40:57-07:00"
              finalizing: false
      "n": 3
      timestamp: "2026-10-19T08:17:38-07:00"
      dispatched: "2026-10-19T07:55:34-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: PQ-7 Revision gives the constructor signatures (NewSlotAddress/NewTeamAddress); code Validate is a pure shape check, Scope filled by NewTeamAddress (lifecycleaddress.go:55-79).
          round: 3
        - id: BR-2
          disposition: addressed
          note: artifactpath tests pass at head; transcript_bind.go classified scoped-ledger ResolvedConsumer (manifest.go), NewSlotAddress now used by LifecycleFailureSlotKey, NewTeamAddress allowlisted with pair#694 reason.
          round: 3
        - id: BR-3
          disposition: addressed
          note: ThreadRecord.MarshalJSON = encodeThreadRecord; reverting it makes TestThreadRecordJSONKeepsUnknownMembers fail (verified in checkout).
          round: 3
        - id: BR-4
          disposition: addressed
          note: 'Note field removed and deferral Revision recorded (plan M1 review round 1, BR-4) as the finding allowed; receiver #684 tracker does not yet carry it (new Minor).'
          round: 3
        - id: BR-5
          disposition: addressed
          note: Decode no longer runs Transcript.Validate; restoring it makes TestInvalidStoredBindingKeepsTheRecordReadable fail (verified); Bind refuses or replaces via ValidateLocation.
          round: 3
        - id: BR-6
          disposition: addressed
          note: LifecycleFailureSlotKey delegates to NewSlotAddress(...).Key(); NewSlotAddress documents caller canonicalizes; pinned by TestLifecycleAddressSlotKeyMatchesFailureSlotKey.
          round: 3
      findings:
        - kind: defect
          id: BR-7
          severity: Important
          title: 'Nested transcript object is strict: a newer build''s member inside "transcript" makes the whole record unreadable'
          detail: 'Probe: CreateThread with a binding, add "minted_at" inside transcript on disk, GetThread fails with json: unknown field "minted_at". record.go:113 keeps nested objects strict, contrary to ARCH-EVOLVE "including nested members"; the result is the same whole-record refusal BR-5 removed. Rule: every nested object this issue introduces or will extend preserves unknown members. Give TranscriptBinding Unknown plus UnmarshalJSON/MarshalJSON via strictjson.SplitObject/EncodeObject, add a nested round-trip test, and state the nested policy in plan §2a.'
          family: nested-unknown-preservation
          round: 3
        - kind: defect
          id: BR-8
          severity: Minor
          title: 'LifecycleFailureSlotKey now requires a clean absolute path; a non-clean absolute StartingPath silently moves from slot: to thread: key'
          detail: 'This is the 2nd finding in family plan-code-drift. Rule: a Revision that routes an existing #683 contract through a new constructor records any narrowing of its accepted inputs. Previously filepath.IsAbs; now workspaceAbsolute (Clean==p, valid UTF-8, no NUL). Record the narrowing in the plan, or pin it with a test.'
          family: plan-code-drift
          round: 3
        - kind: defect
          id: BR-9
          severity: Minor
          title: 'BR-4''s row note and legacy-ledger row detail are deferred to #684 M2 only in #695''s plan; #684''s tracker does not mention them'
          detail: 'Add the deferred items to the #684 issue (or its plan) so the receiving issue owns them. Also, transcript_bind.go:134 says "stored transcript binding is invalid" when the invalid binding came from the ledger proof.'
          family: deferral-recorded-at-receiver
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 583522e8-c04d-4833-80be-4c648b6a4cb7
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/8c093c960c259dd5
            reviewed_head: 0f1c1bed112bd36f3f7d7936886fed3887d46d4c
            slot: pair:1
            requested_at: "2026-10-19T15:27:52.857883Z"
          phase: finished
          pid: 37722
          started_at: "2026-10-19T15:27:52.984179Z"
          finished_at: "2026-10-19T15:44:30.770286Z"
          session_id: 4a19ae9d-ac38-4a0e-b456-0afbb16d3789
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 0f1c1bed112bd36f3f7d7936886fed3887d46d4c
        tracker_head: 55e9c5834590aaf92cec2bd5d4b4a2fd6a17b9b2
        branch_point: 26df2fe40f70ab2b9e3d1bfd88c3b243ee42e02e
        since: "2026-10-18T22:57:15-07:00"
        until: "2026-10-19T08:27:03-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T22:57:15-07:00"
              end: "2026-10-19T08:27:03-07:00"
              began: "2026-10-18T22:57:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: claude
              id: 0f8622e1-c463-4612-8427-0339d0387f2b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: ca3da0965e8961d1c61c665731c4cd42b4e97772c869410aef3541505e4f55ce
              started: "2026-10-19T13:36:29.283Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-19T13:36:29.283Z"
                  end: "2026-10-19T08:27:03-07:00"
            - harness: claude
              id: 507d0585-c54a-4834-8abd-9a52d7f9da76
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 9241864b05c11a745ab97438345dd81758809ee7a1b5f36624c6d53ad60d32ca
              started: "2026-10-19T02:49:17.103Z"
              spawned: false
              bytes: 9217659
              windows:
                - start: "2026-10-18T22:57:15-07:00"
                  end: "2026-10-19T08:27:03-07:00"
            - harness: claude
              id: 6fa6fea2-8902-4f5c-9b2a-6548cf879f30
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: faaf9c38d70a8a84ae00d705b1eef74c2570dd281bc35df426c3f6a837398123
              started: "2026-10-19T13:30:41.221Z"
              spawned: false
              bytes: 614766
              windows:
                - start: "2026-10-19T13:30:41.221Z"
                  end: "2026-10-19T08:27:03-07:00"
            - harness: claude
              id: 84fed9bb-2c33-495b-a2ac-b1c365c8f242
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 45246414c4d539667e82f0898e93805a8f3c200e359a018f9d53d1c10c45e757
              started: "2026-10-19T13:36:13.159Z"
              spawned: false
              bytes: 572779
              windows:
                - start: "2026-10-19T13:36:13.159Z"
                  end: "2026-10-19T08:27:03-07:00"
            - harness: claude
              id: 98afc721-0259-4ecb-a3aa-286966a8fa6b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a6426bb7d4eda2cbe641637263404af0ffc0e861b43796863689e58ebf3b5ab
              started: "2026-10-19T13:35:01.399Z"
              spawned: false
              bytes: 418152
              windows:
                - start: "2026-10-19T13:35:01.399Z"
                  end: "2026-10-19T08:27:03-07:00"
            - harness: claude
              id: b5833248-4bc3-4582-a4e6-6c13f3737a1f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 00e83c9ffe05ea6321734652c9f1f96a46f491047bfc59de33ed5ac5eff504d4
              started: "2026-10-19T13:32:47.197Z"
              spawned: false
              bytes: 604367
              windows:
                - start: "2026-10-19T13:32:47.197Z"
                  end: "2026-10-19T08:27:03-07:00"
            - harness: claude
              id: ccfc2441-0f65-4249-8017-65b2eb08cc24
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5b9b248858b400579af6244ff35f2baa03f8f0e3b41b50d9804fd87bf8a87314
              started: "2026-10-19T13:34:27.337Z"
              spawned: false
              bytes: 574614
              windows:
                - start: "2026-10-19T13:34:27.337Z"
                  end: "2026-10-19T08:27:03-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-19T07:20:37-07:00"
              finalized: "2026-10-19T07:40:57-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T07:55:34-07:00"
              finalized: "2026-10-19T08:17:38-07:00"
              finalizing: false
      "n": 4
      timestamp: "2026-10-19T08:44:32-07:00"
      dispatched: "2026-10-19T08:27:51-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-7
          disposition: addressed
          note: TranscriptBinding has its own JSON methods on strictjson (transcript.go:34-73). With them reverted, TestTranscriptBindingNestedUnknownMembersSurvive fails with unknown field future_tty; it is the only new nested persisted type.
          round: 4
        - id: BR-8
          disposition: addressed
          note: Narrowing recorded in the plan's round-2 Revision and pinned in TestLifecycleAddressSlotKeyMatchesFailureSlotKey (lifecycleaddress_test.go:157-168), which passes at head.
          round: 4
        - id: BR-9
          disposition: not-addressed
          note: 'The refusal text is fixed (transcript_bind.go:125,134). But no #684 plan exists in any ref here, and the #684 issue file is unchanged since f56df570d, so the receiver still does not own the deferred items.'
          round: 4
      findings:
        - kind: defect
          id: BR-10
          severity: Minor
          title: Record.Unknown comment still says nested objects stay strict after BR-7 made transcript tolerant
          detail: 'This is the 2nd finding in family nested-unknown-preservation. The rule is already in the round-2 Revision: a persisted type born in this issue is tolerant and preserving from birth; older nested types stay strict. record.go:110-114 still states the old policy. Reword it to that rule, so the code no longer contradicts the plan.'
          family: nested-unknown-preservation
          round: 4
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 0f1c1bed112bd36f3f7d7936886fed3887d46d4c
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: e4a1a249-8385-41a6-bf6d-88bc57f994b0
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: milestone-review
            round: 3
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/f5e1d9ddb9d3bb4a
            reviewed_head: 60c4f86e3a0f2ebeda728b969707661c3150af4d
            slot: pair:1
            requested_at: "2026-10-19T19:16:58.088808Z"
          phase: finished
          pid: 74503
          started_at: "2026-10-19T19:16:58.236453Z"
          finished_at: "2026-10-19T19:31:52.234804Z"
          session_id: 080e2bd5-8f0a-4e26-b94d-e2fe09deb11b
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 60c4f86e3a0f2ebeda728b969707661c3150af4d
        tracker_head: e38e5541b790e4577b04580ea64b9d14dd9e525f
        branch_point: fffbf9c0d921bfc1df6ffb1fc0fe18628e53d812
        since: "2026-10-18T22:57:15-07:00"
        until: "2026-10-19T12:10:42-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T22:57:15-07:00"
              end: "2026-10-19T12:10:42-07:00"
              began: "2026-10-18T22:57:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: claude
              id: 0f8622e1-c463-4612-8427-0339d0387f2b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: ca3da0965e8961d1c61c665731c4cd42b4e97772c869410aef3541505e4f55ce
              started: "2026-10-19T13:36:29.283Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-19T13:36:29.283Z"
                  end: "2026-10-19T12:10:42-07:00"
            - harness: claude
              id: 507d0585-c54a-4834-8abd-9a52d7f9da76
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 6c51048f44a15cd8557f959f3afcdc73296b5573ed275dbc7a0448e31dc28ee8
              started: "2026-10-19T02:49:17.103Z"
              spawned: false
              bytes: 15022758
              windows:
                - start: "2026-10-18T22:57:15-07:00"
                  end: "2026-10-19T12:10:42-07:00"
            - harness: claude
              id: 6fa6fea2-8902-4f5c-9b2a-6548cf879f30
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: faaf9c38d70a8a84ae00d705b1eef74c2570dd281bc35df426c3f6a837398123
              started: "2026-10-19T13:30:41.221Z"
              spawned: false
              bytes: 614766
              windows:
                - start: "2026-10-19T13:30:41.221Z"
                  end: "2026-10-19T12:10:42-07:00"
            - harness: claude
              id: 84fed9bb-2c33-495b-a2ac-b1c365c8f242
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 45246414c4d539667e82f0898e93805a8f3c200e359a018f9d53d1c10c45e757
              started: "2026-10-19T13:36:13.159Z"
              spawned: false
              bytes: 572779
              windows:
                - start: "2026-10-19T13:36:13.159Z"
                  end: "2026-10-19T12:10:42-07:00"
            - harness: claude
              id: 98afc721-0259-4ecb-a3aa-286966a8fa6b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a6426bb7d4eda2cbe641637263404af0ffc0e861b43796863689e58ebf3b5ab
              started: "2026-10-19T13:35:01.399Z"
              spawned: false
              bytes: 418152
              windows:
                - start: "2026-10-19T13:35:01.399Z"
                  end: "2026-10-19T12:10:42-07:00"
            - harness: claude
              id: b5833248-4bc3-4582-a4e6-6c13f3737a1f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 00e83c9ffe05ea6321734652c9f1f96a46f491047bfc59de33ed5ac5eff504d4
              started: "2026-10-19T13:32:47.197Z"
              spawned: false
              bytes: 604367
              windows:
                - start: "2026-10-19T13:32:47.197Z"
                  end: "2026-10-19T12:10:42-07:00"
            - harness: claude
              id: ccfc2441-0f65-4249-8017-65b2eb08cc24
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5b9b248858b400579af6244ff35f2baa03f8f0e3b41b50d9804fd87bf8a87314
              started: "2026-10-19T13:34:27.337Z"
              spawned: false
              bytes: 574614
              windows:
                - start: "2026-10-19T13:34:27.337Z"
                  end: "2026-10-19T12:10:42-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-19T07:20:37-07:00"
              finalized: "2026-10-19T07:40:57-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T07:55:34-07:00"
              finalized: "2026-10-19T08:17:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T08:27:51-07:00"
              finalized: "2026-10-19T08:44:32-07:00"
              finalizing: true
      "n": 5
      timestamp: "2026-10-19T12:31:56-07:00"
      dispatched: "2026-10-19T12:16:55-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Overtaken: the team arm and TeamIdentity were removed in M2 (009030459); Validate/Key now have one slot kind.'
          round: 5
      findings:
        - kind: defect
          id: BR-11
          severity: Critical
          title: Teardown proves a tree down while the published agent is still alive when agent-ready carries no tty
          detail: 'teardown.go teardownHolders and proveDown witness the agent only through TTYProcesses(table, tty), and TTYProcesses with "" matches nothing. So for any pre-M2 wrapper, or a wrapper whose agentTerminal table read failed, the proof has no agent witness. The plan''s fallback (proven as today by pid and session, ObserveSlotSessions) was dropped. A reviewer probe (agent published with no tty that ignores SIGTERM) returned err=nil with the agent Live and LeftRunning empty, and ArchiveThread would archive with that proof. Fix: when the published agent is listed, sweep and require its exact identity Dead whatever the tty (holder kind agent), and add a regression test.'
          family: proof-witness-per-process-kind
          round: 5
        - kind: defect
          id: BR-12
          severity: Important
          title: The tty-trusted-only-while-the-tree-is-up guard has no test; a mutation that always trusts the tty passes the suite
          detail: 'Changing teardown.go:161 to `if true || agentListed(...) || !free` passed every couchcore Teardown, Retire, Archive, Fresh, Reboot and Recover test (183s). This guard is what stops teardown from SIGKILLing an unrelated terminal''s processes on a reused device. Add a test: agent dead, lease free, an unrelated live process on the same tty number; expect no signal and the note.'
          family: kill-guard-regression-test
          round: 5
        - kind: defect
          id: BR-13
          severity: Important
          title: Per-thread lifetime lease files are removed only by Teardown(stop); every other retirement leaks one file forever
          detail: Lease keys include scope/tag (lifetime_lease.go:20), and only teardown with stop removes the file (teardown.go:208). Retirements through ProveDown (startFreshSlotWithPrompt, reboot archive-only), proveDownAndFold (Fresh recovery), AdoptionAuthority and UnreadableRecordEscape leave <namespace>/leases/<digest>.lease with no sweep, so the directory grows with every Fresh, reboot or adoption (ARCH-FUNERAL). Remove the retired thread's lease once its proof stands, or sweep leases whose thread is gone, and state the lifetime in the atlas.
          family: artifact-removal-path
          round: 5
        - kind: defect
          id: BR-14
          severity: Important
          title: TeardownProof is forgeable (all fields exported), unlike the three other constructor-only retire authorities
          detail: RetireAuthority is sealed at the interface, but any package can build couchcore.TeardownProof{Address, Thread, Revision, ProvenAt}, and couchcoretest/retire.go:25 (a non-test package) does. That undercuts ARCH-ORDER's structural enforcement that the archive and replace funnels cannot be reached without a real proof. Unexport the authority-bearing fields (with accessors), so only proveDown mints one, and give fixtures an explicitly test-only minting path.
          family: authority-unforgeable
          round: 5
        - kind: defect
          id: BR-15
          severity: Important
          title: 'README update missing: couch --reap-orphans --confirm now also ends teardown''s left-running processes'
          detail: reap_orphans.go adds reapLeftRunning to the destructive verb, and --resources gains the "Left running by teardown" section. README.md:477-478 and :822-828 still describe only unattributed embedded nvim, so an operator who uses --confirm for nvim cleanup will now also kill setsid'd runners and background merges without the docs saying so.
          family: readme-surface-gate
          round: 5
        - kind: defect
          id: BR-16
          severity: Important
          title: M2 plan contracts and named tests not delivered as specified (plain-pair test cases, both-env activation, sweep test, recorder path, holder kinds, witnesses)
          detail: 'This is the 3rd finding in family plan-code-drift. Rule: at milestone close, every test name and contract in the plan''s governing revision is checked mechanically against the tree (grep the func names; diff each contract against the code), and each mismatch gets either code or a Revisions line in the same round. Measured prevalence in M2 is 7 instances. (1) TestPlainPairUnchangedWithoutCouch checks only HoldFromEnv with an empty env, not two sessions, a sidecar without a tty, no extra fd, or the lease-without-tag case. (2) Activation needs only COUCH_LIFETIME_LEASE (lease.go:60, wrap.go:652), not both env vars. (3) TestTeardownSweepsReparentedHolders is absent. (4) TestTeardownUnprovenRecordsPhaseAndHolders does not go through the real recorder. (5) ReadyRecord is tolerant but not preserving, unrecorded. (6) The agent and session-watch holder kinds were not added (pair/process used). (7) The plan''s four witnesses became three (ObserveSlotSessions dropped).'
          family: plan-code-drift
          round: 5
        - kind: defect
          id: BR-17
          severity: Important
          title: orphans.json accepts any schema_version and rewrites it as 1; entries with pid <= 0 or an empty identity decode
          detail: 'This is the 2nd finding in family decode-strictness-beyond-types. Rule: every persisted envelope validates its known-field values (supported schema_version, positive pid, non-empty identity) at decode, not only their JSON types, and a rewrite never changes an existing field''s value. orphan_registry.go:201 decodes with no value checks, and :282 sets SchemaVersion=1 over whatever was read, which changes a newer writer''s field (ARCH-EVOLVE). Refuse an unsupported version, validate entries, and add round-trip and refusal tests.'
          family: decode-strictness-beyond-types
          round: 5
        - kind: defect
          id: BR-18
          severity: Minor
          title: The split, encode, strict-decode, keep-unknown sequence is copy-pasted at five decode sites
          detail: transcript.go:47, record.go:324, readiness/record.go:61 and orphan_registry.go:81/114/142 repeat the same four steps. A strictjson helper (decode the known members into a wire struct, return the unknown map) would make each type's UnmarshalJSON a single call (ARCH-DRY).
          family: preserving-codec-boilerplate
          round: 5
        - kind: defect
          id: BR-19
          severity: Minor
          title: teardown(stop, fold bool) encodes three legal modes as two booleans; stop without fold is representable but unused
          detail: Replace the two booleans with a mode enum (teardown, prove-down, prove-and-fold) so the legal set can be read off the type (ARCH-ORDER).
          family: mode-as-bool-flags
          round: 5
        - kind: defect
          id: BR-20
          severity: Minor
          title: ReadyRecord.TTY comment says names like ttys183 or pts/3, but the value is a decimal device number
          detail: procutil.Process.TTY stores e_tdev or tty_nr in decimal, and the wrapper publishes that same value. Fix the comment at readiness/record.go:30.
          family: comment-code-drift
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 9016333d-ec91-4480-af66-dc2138dc2bec
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: milestone-review
            round: 4
            attempt: 1
            agent: claude
            root: /Users/xianxu/Library/Caches/sdlc/review-checkouts/812ed72f3c403142
            reviewed_head: 99ca957e2b74d77402d35ca9eb40a408a295cb11
            slot: pair:1
            requested_at: "2026-10-19T19:46:12.850059Z"
          phase: finished
          pid: 18007
          started_at: "2026-10-19T19:46:13.015702Z"
          finished_at: "2026-10-19T20:16:10.12661Z"
          session_id: f1831f85-10bb-44ea-8d61-ddd0b630abfe
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-v1
        repository: github.com/xianxu/pair
        code_head: 99ca957e2b74d77402d35ca9eb40a408a295cb11
        tracker_head: 9e1fcff2b38cf616784e99d01160802c4f709428
        branch_point: fffbf9c0d921bfc1df6ffb1fc0fe18628e53d812
        since: "2026-10-18T22:57:15-07:00"
        until: "2026-10-19T12:44:58-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T22:57:15-07:00"
              end: "2026-10-19T12:44:58-07:00"
              began: "2026-10-18T22:57:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: claude
              id: 0f8622e1-c463-4612-8427-0339d0387f2b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: ca3da0965e8961d1c61c665731c4cd42b4e97772c869410aef3541505e4f55ce
              started: "2026-10-19T13:36:29.283Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-19T13:36:29.283Z"
                  end: "2026-10-19T12:44:58-07:00"
            - harness: claude
              id: 507d0585-c54a-4834-8abd-9a52d7f9da76
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: cee3b4a311375517db078d115a92caf62758255fc422d8ac956fd9af34de4f72
              started: "2026-10-19T02:49:17.103Z"
              spawned: false
              bytes: 15758559
              windows:
                - start: "2026-10-18T22:57:15-07:00"
                  end: "2026-10-19T12:44:58-07:00"
            - harness: claude
              id: 6fa6fea2-8902-4f5c-9b2a-6548cf879f30
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: faaf9c38d70a8a84ae00d705b1eef74c2570dd281bc35df426c3f6a837398123
              started: "2026-10-19T13:30:41.221Z"
              spawned: false
              bytes: 614766
              windows:
                - start: "2026-10-19T13:30:41.221Z"
                  end: "2026-10-19T12:44:58-07:00"
            - harness: claude
              id: 84fed9bb-2c33-495b-a2ac-b1c365c8f242
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 45246414c4d539667e82f0898e93805a8f3c200e359a018f9d53d1c10c45e757
              started: "2026-10-19T13:36:13.159Z"
              spawned: false
              bytes: 572779
              windows:
                - start: "2026-10-19T13:36:13.159Z"
                  end: "2026-10-19T12:44:58-07:00"
            - harness: claude
              id: 98afc721-0259-4ecb-a3aa-286966a8fa6b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a6426bb7d4eda2cbe641637263404af0ffc0e861b43796863689e58ebf3b5ab
              started: "2026-10-19T13:35:01.399Z"
              spawned: false
              bytes: 418152
              windows:
                - start: "2026-10-19T13:35:01.399Z"
                  end: "2026-10-19T12:44:58-07:00"
            - harness: claude
              id: b5833248-4bc3-4582-a4e6-6c13f3737a1f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 00e83c9ffe05ea6321734652c9f1f96a46f491047bfc59de33ed5ac5eff504d4
              started: "2026-10-19T13:32:47.197Z"
              spawned: false
              bytes: 604367
              windows:
                - start: "2026-10-19T13:32:47.197Z"
                  end: "2026-10-19T12:44:58-07:00"
            - harness: claude
              id: ccfc2441-0f65-4249-8017-65b2eb08cc24
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5b9b248858b400579af6244ff35f2baa03f8f0e3b41b50d9804fd87bf8a87314
              started: "2026-10-19T13:34:27.337Z"
              spawned: false
              bytes: 574614
              windows:
                - start: "2026-10-19T13:34:27.337Z"
                  end: "2026-10-19T12:44:58-07:00"
        cuts:
            - boundary: M1
              dispatched: "2026-10-19T07:20:37-07:00"
              finalized: "2026-10-19T07:40:57-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T07:55:34-07:00"
              finalized: "2026-10-19T08:17:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T08:27:51-07:00"
              finalized: "2026-10-19T08:44:32-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-19T12:16:55-07:00"
              finalized: "2026-10-19T12:31:56-07:00"
              finalizing: false
      "n": 6
      timestamp: "2026-10-19T13:16:10-07:00"
      dispatched: "2026-10-19T12:46:10-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-11
          disposition: addressed
          note: teardown.go:365-369 adds the published agent by exact pid+identity; mutation (drop add) turns TestTeardownWitnessesAnAgentWithoutATerminal red.
          round: 6
        - id: BR-12
          disposition: addressed
          note: TestTeardownNeverSweepsAReusedTerminal fails under the `if true ||` mutation (pid 900 terminated), verified in this checkout.
          round: 6
        - id: BR-13
          disposition: addressed
          note: Lease removed after a proof in every mode (all-mode test fails when limited to stopping mode); SweepFree at Couch start with TestSweepFreeRemovesOnlyFreeLeases; atlas states the lifetime.
          round: 6
        - id: BR-14
          disposition: addressed
          note: Authority fields unexported with accessors; fixtures use TestOnlyTeardownProof plus a scan test (scan-scope gap raised as a new Minor).
          round: 6
        - id: BR-15
          disposition: addressed
          note: README.md:477-481 and :823-829 now name the left-running processes that --reap-orphans --confirm ends and the --resources section.
          round: 6
        - id: BR-16
          disposition: addressed
          note: Each of the 7 mismatches got code or a Revisions entry (plan round-4 log); sweep and recorder tests exist, and Active is the single activation predicate.
          round: 6
        - id: BR-17
          disposition: addressed
          note: orphan_registry.go refuses schema_version != 1 and pid<=0 or empty identity at decode, and the rewrite keeps the version; refusal + byte-unchanged test present.
          round: 6
        - id: BR-18
          disposition: addressed
          note: strictjson.DecodeKnown replaces the five sites; no remaining copy of the split/encode/decode sequence in the diff.
          round: 6
        - id: BR-19
          disposition: addressed
          note: teardownMode enum with stops()/folds() replaces (stop, fold bool).
          round: 6
        - id: BR-20
          disposition: addressed
          note: readiness/record.go:30 comment now says decimal device number (Darwin e_tdev, Linux tty_nr).
          round: 6
      findings:
        - kind: defect
          id: BR-21
          severity: Minor
          title: Teardown unlinks the proven-free lease without holding its exclusive lock, unlike SweepFree
          detail: 'teardown.go:245 calls os.Remove after proveDown''s Free released its probe lock, so a Hold that locks between the two keeps an unlinked inode that every later witness misses. Rule: a lock file is unlinked only under LOCK_EX with a same-file check; share one lifetimelease removeIfFree helper used by SweepFree and teardown.'
          family: unlink-lock-file-under-lock
          round: 6
        - kind: defect
          id: BR-22
          severity: Minor
          title: The production-caller scan for TestOnlyTeardownProof covers only cmd/internal and exempts couchcoretest
          detail: 'This is the 2nd finding in family authority-unforgeable. Rule: an authority''s test-only minting path is unreachable from every production package in the module, so the guard walks the module root and also refuses non-test imports of couchcoretest. retire_test.go walks ".." (cmd/internal) only, so cmd/couch and cmd/pair-go are unscanned.'
          family: authority-unforgeable
          round: 6
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 99ca957e2b74d77402d35ca9eb40a408a295cb11
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: b8c7408c-8a96-423f-9fef-ff9b5c780d65
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: milestone-review
            round: 7
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/bf781e2264670ba0
            reviewed_head: f9d4f031fe85f8fb6d525c1b8a5b5a5a13b6d17e
            slot: pair:1
            requested_at: "2026-10-20T00:26:13.755908Z"
          phase: finished
          pid: 4682
          started_at: "2026-10-20T00:26:13.932818Z"
          finished_at: "2026-10-20T00:35:29.038101Z"
          session_id: a53bc65f-e1da-4ce1-844c-b4beefbb2c8d
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: f9d4f031fe85f8fb6d525c1b8a5b5a5a13b6d17e
        tracker_head: abb871671e94a94ba410e62e0cd12c71775f95b9
        branch_point: fffbf9c0d921bfc1df6ffb1fc0fe18628e53d812
        since: "2026-10-18T22:57:15-07:00"
        until: "2026-10-19T16:47:18-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T22:57:15-07:00"
              end: "2026-10-19T16:47:18-07:00"
              began: "2026-10-18T22:57:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: claude
              id: 0f8622e1-c463-4612-8427-0339d0387f2b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: ca3da0965e8961d1c61c665731c4cd42b4e97772c869410aef3541505e4f55ce
              started: "2026-10-19T13:36:29.283Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-19T13:36:29.283Z"
                  end: "2026-10-19T16:47:18-07:00"
            - harness: claude
              id: 507d0585-c54a-4834-8abd-9a52d7f9da76
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 7a9fef517c978e06247002f5f9fdab5df82f006d847b0f5ceea1f4fb8f99bb3a
              started: "2026-10-19T02:49:17.103Z"
              spawned: false
              bytes: 16587613
              windows:
                - start: "2026-10-18T22:57:15-07:00"
                  end: "2026-10-19T16:47:18-07:00"
            - harness: claude
              id: 6fa6fea2-8902-4f5c-9b2a-6548cf879f30
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: faaf9c38d70a8a84ae00d705b1eef74c2570dd281bc35df426c3f6a837398123
              started: "2026-10-19T13:30:41.221Z"
              spawned: false
              bytes: 614766
              windows:
                - start: "2026-10-19T13:30:41.221Z"
                  end: "2026-10-19T16:47:18-07:00"
            - harness: claude
              id: 84fed9bb-2c33-495b-a2ac-b1c365c8f242
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 45246414c4d539667e82f0898e93805a8f3c200e359a018f9d53d1c10c45e757
              started: "2026-10-19T13:36:13.159Z"
              spawned: false
              bytes: 572779
              windows:
                - start: "2026-10-19T13:36:13.159Z"
                  end: "2026-10-19T16:47:18-07:00"
            - harness: claude
              id: 98afc721-0259-4ecb-a3aa-286966a8fa6b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a6426bb7d4eda2cbe641637263404af0ffc0e861b43796863689e58ebf3b5ab
              started: "2026-10-19T13:35:01.399Z"
              spawned: false
              bytes: 418152
              windows:
                - start: "2026-10-19T13:35:01.399Z"
                  end: "2026-10-19T16:47:18-07:00"
            - harness: claude
              id: b5833248-4bc3-4582-a4e6-6c13f3737a1f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 00e83c9ffe05ea6321734652c9f1f96a46f491047bfc59de33ed5ac5eff504d4
              started: "2026-10-19T13:32:47.197Z"
              spawned: false
              bytes: 604367
              windows:
                - start: "2026-10-19T13:32:47.197Z"
                  end: "2026-10-19T16:47:18-07:00"
            - harness: claude
              id: ccfc2441-0f65-4249-8017-65b2eb08cc24
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5b9b248858b400579af6244ff35f2baa03f8f0e3b41b50d9804fd87bf8a87314
              started: "2026-10-19T13:34:27.337Z"
              spawned: false
              bytes: 574614
              windows:
                - start: "2026-10-19T13:34:27.337Z"
                  end: "2026-10-19T16:47:18-07:00"
        queue_waits:
            - attempt_id: 1445b62cb2dd5a17cb7c0288dec9e972
              slot: pair:1
              record_digest: 723c5097e3005e039c48f91b9ebc5f795800271e11508eb250d4bad58b05d28e
              start: "2026-10-19T20:17:24.689345Z"
              end: "2026-10-19T21:06:57.319288Z"
              queue_seconds: 2972.629943
              raw_seconds: 2972.629943
              exact: true
            - attempt_id: 98a0a59be0897c1c1859129ba9f7443e
              slot: pair:1
              record_digest: 344451b4b4f239f83ef1180c373ebb01ae733bf2f1fab31e157a88bd399a0a75
              start: "2026-10-19T21:20:26.111822Z"
              end: "2026-10-19T22:28:42.531888Z"
              queue_seconds: 4096.420066
              raw_seconds: 4096.420066
              exact: true
            - attempt_id: 47b2fef6de776a0b9f4a3723781e4946
              slot: pair:1
              record_digest: 553d78160fddadeb4f496831499fe10b6ac9390c8ccc46ee45f47d6614d6bc7a
              start: "2026-10-19T23:19:39.509446Z"
              end: "2026-10-19T23:38:10.154435Z"
              queue_seconds: 1110.644989
              raw_seconds: 1110.644989
              exact: true
        cuts:
            - boundary: M1
              dispatched: "2026-10-19T07:20:37-07:00"
              finalized: "2026-10-19T07:40:57-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T07:55:34-07:00"
              finalized: "2026-10-19T08:17:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T08:27:51-07:00"
              finalized: "2026-10-19T08:44:32-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-19T12:16:55-07:00"
              finalized: "2026-10-19T12:31:56-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-19T12:46:10-07:00"
              finalized: "2026-10-19T13:16:10-07:00"
              finalizing: true
      "n": 7
      timestamp: "2026-10-19T17:35:29-07:00"
      dispatched: "2026-10-19T17:26:10-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      dispose:
        - id: BR-9
          disposition: addressed
          note: 'The #684 issue''s M2 checklist now owns both items from pair#695 M1 (BR-4), the stored-binding row note and the legacy-ledger row detail, and its log records receiving them. The transcript_bind.go:134 message now reads "its transcript binding is invalid", which is neutral about where the binding came from.'
          round: 7
        - id: BR-10
          disposition: addressed
          note: record.go:113-118 now says nested types older than pair#695 stay strict and a nested type born additive (TranscriptBinding) keeps its own unknown members, matching the round-2 Revision.
          round: 7
        - id: BR-21
          disposition: addressed
          note: 'teardown.go:245 now calls lifetimelease.RemoveFree, the same helper SweepFree uses: it takes LOCK_EX|LOCK_NB and checks the path still names the same file. TestRemoveFreeLeavesAHeldLease fails at lease_test.go:242 when the lock check is removed.'
          round: 7
        - id: BR-22
          disposition: addressed
          note: retire_test.go now walks the module root and also refuses non-test imports of couchcoretest. A planted cmd/zzprobe/p.go was flagged at retire_test.go:326 and :329.
          round: 7
      recipe: milestone-review
      isolation: fenced
      reviewed: f9d4f031fe85f8fb6d525c1b8a5b5a5a13b6d17e
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: be05aaf8-1a97-4576-a4bb-d6b88b2877fe
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: fix-review
            round: 8
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/3a358c9e6d57df36
            reviewed_head: cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b
            slot: pair:1
            requested_at: "2026-10-20T02:10:26.698876Z"
          phase: finished
          pid: 82054
          started_at: "2026-10-20T02:10:26.890295Z"
          finished_at: "2026-10-20T02:18:10.438264Z"
          session_id: 8f63c762-61e8-4e14-8011-9d2724e7fea7
          outcome: returned
        - version: 1
          intent:
            disposable_root: true
            id: c8bcda53-d77e-4a1e-b5c5-289287af1150
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: fix-review
            round: 8
            attempt: 2
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/3a358c9e6d57df36
            reviewed_head: cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b
            slot: pair:1
            requested_at: "2026-10-20T02:18:16.517387Z"
          phase: finished
          pid: 62311
          started_at: "2026-10-20T02:18:16.732709Z"
          finished_at: "2026-10-20T02:25:33.271074Z"
          session_id: 969a7df4-c309-47cf-949f-53a7fea690d3
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b
        tracker_head: 49c1af72fed3994e52e68fed20962247c9d3f021
        branch_point: 88540c4dbdbf68d1b63de8f6c05840a11b82deb3
        since: "2026-10-18T22:57:15-07:00"
        until: "2026-10-19T18:29:50-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T22:57:15-07:00"
              end: "2026-10-19T18:29:50-07:00"
              began: "2026-10-18T22:57:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: claude
              id: 0f8622e1-c463-4612-8427-0339d0387f2b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: ca3da0965e8961d1c61c665731c4cd42b4e97772c869410aef3541505e4f55ce
              started: "2026-10-19T13:36:29.283Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-19T13:36:29.283Z"
                  end: "2026-10-19T18:29:50-07:00"
            - harness: claude
              id: 507d0585-c54a-4834-8abd-9a52d7f9da76
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 543f244f1abe99f49999b94a19bf7b87b5736c356774801254f979352781fa00
              started: "2026-10-19T02:49:17.103Z"
              spawned: false
              bytes: 17106707
              windows:
                - start: "2026-10-18T22:57:15-07:00"
                  end: "2026-10-19T18:29:50-07:00"
            - harness: claude
              id: 6fa6fea2-8902-4f5c-9b2a-6548cf879f30
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: faaf9c38d70a8a84ae00d705b1eef74c2570dd281bc35df426c3f6a837398123
              started: "2026-10-19T13:30:41.221Z"
              spawned: false
              bytes: 614766
              windows:
                - start: "2026-10-19T13:30:41.221Z"
                  end: "2026-10-19T18:29:50-07:00"
            - harness: claude
              id: 84fed9bb-2c33-495b-a2ac-b1c365c8f242
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 45246414c4d539667e82f0898e93805a8f3c200e359a018f9d53d1c10c45e757
              started: "2026-10-19T13:36:13.159Z"
              spawned: false
              bytes: 572779
              windows:
                - start: "2026-10-19T13:36:13.159Z"
                  end: "2026-10-19T18:29:50-07:00"
            - harness: claude
              id: 98afc721-0259-4ecb-a3aa-286966a8fa6b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a6426bb7d4eda2cbe641637263404af0ffc0e861b43796863689e58ebf3b5ab
              started: "2026-10-19T13:35:01.399Z"
              spawned: false
              bytes: 418152
              windows:
                - start: "2026-10-19T13:35:01.399Z"
                  end: "2026-10-19T18:29:50-07:00"
            - harness: claude
              id: b5833248-4bc3-4582-a4e6-6c13f3737a1f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 00e83c9ffe05ea6321734652c9f1f96a46f491047bfc59de33ed5ac5eff504d4
              started: "2026-10-19T13:32:47.197Z"
              spawned: false
              bytes: 604367
              windows:
                - start: "2026-10-19T13:32:47.197Z"
                  end: "2026-10-19T18:29:50-07:00"
            - harness: claude
              id: ccfc2441-0f65-4249-8017-65b2eb08cc24
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5b9b248858b400579af6244ff35f2baa03f8f0e3b41b50d9804fd87bf8a87314
              started: "2026-10-19T13:34:27.337Z"
              spawned: false
              bytes: 574614
              windows:
                - start: "2026-10-19T13:34:27.337Z"
                  end: "2026-10-19T18:29:50-07:00"
        queue_waits:
            - attempt_id: 1445b62cb2dd5a17cb7c0288dec9e972
              slot: pair:1
              record_digest: 723c5097e3005e039c48f91b9ebc5f795800271e11508eb250d4bad58b05d28e
              start: "2026-10-19T20:17:24.689345Z"
              end: "2026-10-19T21:06:57.319288Z"
              queue_seconds: 2972.629943
              raw_seconds: 2972.629943
              exact: true
            - attempt_id: 98a0a59be0897c1c1859129ba9f7443e
              slot: pair:1
              record_digest: 344451b4b4f239f83ef1180c373ebb01ae733bf2f1fab31e157a88bd399a0a75
              start: "2026-10-19T21:20:26.111822Z"
              end: "2026-10-19T22:28:42.531888Z"
              queue_seconds: 4096.420066
              raw_seconds: 4096.420066
              exact: true
            - attempt_id: 47b2fef6de776a0b9f4a3723781e4946
              slot: pair:1
              record_digest: 553d78160fddadeb4f496831499fe10b6ac9390c8ccc46ee45f47d6614d6bc7a
              start: "2026-10-19T23:19:39.509446Z"
              end: "2026-10-19T23:38:10.154435Z"
              queue_seconds: 1110.644989
              raw_seconds: 1110.644989
              exact: true
            - attempt_id: acbd0650b53472438bdf65712a5d3b0b
              slot: pair:1
              record_digest: a87d153597baad9379ffc7d3e4c92f6c3c405174d6bd6494430ed01317890152
              start: "2026-10-19T23:47:22.272478Z"
              end: "2026-10-20T00:17:10.102229Z"
              queue_seconds: 1787.829751
              raw_seconds: 1787.829751
              exact: true
            - attempt_id: 13c132ae04987e65b18fe6a033603a50
              slot: pair:1
              record_digest: edd274eeba1202e3247ce23a5f5d5352abb7e6b58ef0118d604cc897e609140a
              start: "2026-10-20T00:37:13.172268Z"
              end: "2026-10-20T00:51:05.960988Z"
              queue_seconds: 832.78872
              raw_seconds: 832.78872
              exact: true
            - attempt_id: 905409f564fe231a47fc99ebbacb7d61
              slot: pair:1
              record_digest: ab0ac002ea65a4282e4f7b955c5452981d2ec8f8c870c6e25752602f7895f9bd
              start: "2026-10-20T01:01:31.08107Z"
              end: "2026-10-20T01:20:27.725666Z"
              queue_seconds: 1136.644596
              raw_seconds: 1136.644596
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 28.593
        cuts:
            - boundary: M1
              dispatched: "2026-10-19T07:20:37-07:00"
              finalized: "2026-10-19T07:40:57-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T07:55:34-07:00"
              finalized: "2026-10-19T08:17:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T08:27:51-07:00"
              finalized: "2026-10-19T08:44:32-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-19T12:16:55-07:00"
              finalized: "2026-10-19T12:31:56-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-19T12:46:10-07:00"
              finalized: "2026-10-19T13:16:10-07:00"
              finalizing: true
            - dispatched: "2026-10-19T17:26:10-07:00"
              finalized: "2026-10-19T17:35:29-07:00"
              finalizing: true
      "n": 8
      timestamp: "2026-10-19T19:25:33-07:00"
      dispatched: "2026-10-19T19:10:23-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      recipe: fix-review
      isolation: fenced
      reviewed: cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b
      blocked: false
    - reviewer_runs:
        - version: 1
          intent:
            disposable_root: true
            id: 5d87e073-d9ae-4d91-9203-8dcdb31d7121
            repository: github.com/xianxu/pair
            issue_id: "000695"
            recipe: fix-review
            round: 9
            attempt: 1
            agent: claude
            root: /Volumes/TestScratch/review-checkouts/fe3411754d32507e
            reviewed_head: 2ac8c028e16bae89de44f0749effb18805650f1a
            slot: pair:1
            requested_at: "2026-10-20T03:43:03.37952Z"
          phase: finished
          pid: 52158
          started_at: "2026-10-20T03:43:03.624537Z"
          finished_at: "2026-10-20T03:43:45.64324Z"
          session_id: 709ef70f-83b7-497e-89a8-827b8de30031
          outcome: returned
      actual_inputs:
        version: 1
        method: active-time-v3-engagement-queue-v1
        repository: github.com/xianxu/pair
        code_head: 2ac8c028e16bae89de44f0749effb18805650f1a
        tracker_head: c5c63325f6b6967788353bff80c72542680b4401
        branch_point: 2106313091e8a16aa8ea5bcc2c19c88856bbdd0c
        since: "2026-10-18T22:57:15-07:00"
        until: "2026-10-19T20:18:01-07:00"
        brain_root: /Users/xianxu/workspace/brain
        gap_minutes: 15
        commit_weight: 1
        include_assistant: true
        engagements:
            - start: "2026-10-18T22:57:15-07:00"
              end: "2026-10-19T20:18:01-07:00"
              began: "2026-10-18T22:57:15-07:00"
              roots:
                - /Users/xianxu/workspace/worktree/pair-slot1/pair
        sources:
            - harness: claude
              id: 0f8622e1-c463-4612-8427-0339d0387f2b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: ca3da0965e8961d1c61c665731c4cd42b4e97772c869410aef3541505e4f55ce
              started: "2026-10-19T13:36:29.283Z"
              spawned: false
              bytes: 393181
              windows:
                - start: "2026-10-19T13:36:29.283Z"
                  end: "2026-10-19T20:18:01-07:00"
            - harness: claude
              id: 507d0585-c54a-4834-8abd-9a52d7f9da76
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 564b7a5df01d853f261e688e16fbe0f02a2edeb0bbb00b5fc85521818eb872c9
              started: "2026-10-19T02:49:17.103Z"
              spawned: false
              bytes: 17308802
              windows:
                - start: "2026-10-18T22:57:15-07:00"
                  end: "2026-10-19T20:18:01-07:00"
            - harness: claude
              id: 6fa6fea2-8902-4f5c-9b2a-6548cf879f30
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: faaf9c38d70a8a84ae00d705b1eef74c2570dd281bc35df426c3f6a837398123
              started: "2026-10-19T13:30:41.221Z"
              spawned: false
              bytes: 614766
              windows:
                - start: "2026-10-19T13:30:41.221Z"
                  end: "2026-10-19T20:18:01-07:00"
            - harness: claude
              id: 84fed9bb-2c33-495b-a2ac-b1c365c8f242
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 45246414c4d539667e82f0898e93805a8f3c200e359a018f9d53d1c10c45e757
              started: "2026-10-19T13:36:13.159Z"
              spawned: false
              bytes: 572779
              windows:
                - start: "2026-10-19T13:36:13.159Z"
                  end: "2026-10-19T20:18:01-07:00"
            - harness: claude
              id: 98afc721-0259-4ecb-a3aa-286966a8fa6b
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 0a6426bb7d4eda2cbe641637263404af0ffc0e861b43796863689e58ebf3b5ab
              started: "2026-10-19T13:35:01.399Z"
              spawned: false
              bytes: 418152
              windows:
                - start: "2026-10-19T13:35:01.399Z"
                  end: "2026-10-19T20:18:01-07:00"
            - harness: claude
              id: b5833248-4bc3-4582-a4e6-6c13f3737a1f
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 00e83c9ffe05ea6321734652c9f1f96a46f491047bfc59de33ed5ac5eff504d4
              started: "2026-10-19T13:32:47.197Z"
              spawned: false
              bytes: 604367
              windows:
                - start: "2026-10-19T13:32:47.197Z"
                  end: "2026-10-19T20:18:01-07:00"
            - harness: claude
              id: ccfc2441-0f65-4249-8017-65b2eb08cc24
              slot: pair:1
              cwd: /Users/xianxu/workspace/worktree/pair-slot1/pair
              digest: 5b9b248858b400579af6244ff35f2baa03f8f0e3b41b50d9804fd87bf8a87314
              started: "2026-10-19T13:34:27.337Z"
              spawned: false
              bytes: 574614
              windows:
                - start: "2026-10-19T13:34:27.337Z"
                  end: "2026-10-19T20:18:01-07:00"
        queue_waits:
            - attempt_id: 1445b62cb2dd5a17cb7c0288dec9e972
              slot: pair:1
              record_digest: 723c5097e3005e039c48f91b9ebc5f795800271e11508eb250d4bad58b05d28e
              start: "2026-10-19T20:17:24.689345Z"
              end: "2026-10-19T21:06:57.319288Z"
              queue_seconds: 2972.629943
              raw_seconds: 2972.629943
              exact: true
            - attempt_id: 98a0a59be0897c1c1859129ba9f7443e
              slot: pair:1
              record_digest: 344451b4b4f239f83ef1180c373ebb01ae733bf2f1fab31e157a88bd399a0a75
              start: "2026-10-19T21:20:26.111822Z"
              end: "2026-10-19T22:28:42.531888Z"
              queue_seconds: 4096.420066
              raw_seconds: 4096.420066
              exact: true
            - attempt_id: 47b2fef6de776a0b9f4a3723781e4946
              slot: pair:1
              record_digest: 553d78160fddadeb4f496831499fe10b6ac9390c8ccc46ee45f47d6614d6bc7a
              start: "2026-10-19T23:19:39.509446Z"
              end: "2026-10-19T23:38:10.154435Z"
              queue_seconds: 1110.644989
              raw_seconds: 1110.644989
              exact: true
            - attempt_id: acbd0650b53472438bdf65712a5d3b0b
              slot: pair:1
              record_digest: a87d153597baad9379ffc7d3e4c92f6c3c405174d6bd6494430ed01317890152
              start: "2026-10-19T23:47:22.272478Z"
              end: "2026-10-20T00:17:10.102229Z"
              queue_seconds: 1787.829751
              raw_seconds: 1787.829751
              exact: true
            - attempt_id: 13c132ae04987e65b18fe6a033603a50
              slot: pair:1
              record_digest: edd274eeba1202e3247ce23a5f5d5352abb7e6b58ef0118d604cc897e609140a
              start: "2026-10-20T00:37:13.172268Z"
              end: "2026-10-20T00:51:05.960988Z"
              queue_seconds: 832.78872
              raw_seconds: 832.78872
              exact: true
            - attempt_id: 905409f564fe231a47fc99ebbacb7d61
              slot: pair:1
              record_digest: ab0ac002ea65a4282e4f7b955c5452981d2ec8f8c870c6e25752602f7895f9bd
              start: "2026-10-20T01:01:31.08107Z"
              end: "2026-10-20T01:20:27.725666Z"
              queue_seconds: 1136.644596
              raw_seconds: 1136.644596
              exact: true
            - attempt_id: 59ae7e44a64bc65cf7b57726c42ef2f0
              slot: pair:1
              record_digest: 861db3d1e89872b380db6c40e4f8260b89687b42b434427fb000a3c18086215f
              start: "2026-10-20T01:29:53.75337Z"
              end: "2026-10-20T02:00:01.349576Z"
              queue_seconds: 1807.596206
              raw_seconds: 1807.596206
              exact: true
        excluded_brain_sessions: 1
        excluded_brain_minutes: 83.18911666666668
        cuts:
            - boundary: M1
              dispatched: "2026-10-19T07:20:37-07:00"
              finalized: "2026-10-19T07:40:57-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T07:55:34-07:00"
              finalized: "2026-10-19T08:17:38-07:00"
              finalizing: false
            - boundary: M1
              dispatched: "2026-10-19T08:27:51-07:00"
              finalized: "2026-10-19T08:44:32-07:00"
              finalizing: true
            - boundary: M2
              dispatched: "2026-10-19T12:16:55-07:00"
              finalized: "2026-10-19T12:31:56-07:00"
              finalizing: false
            - boundary: M2
              dispatched: "2026-10-19T12:46:10-07:00"
              finalized: "2026-10-19T13:16:10-07:00"
              finalizing: true
            - dispatched: "2026-10-19T17:26:10-07:00"
              finalized: "2026-10-19T17:35:29-07:00"
              finalizing: true
            - dispatched: "2026-10-19T19:10:23-07:00"
              finalized: "2026-10-19T19:25:33-07:00"
              finalizing: true
      "n": 9
      timestamp: "2026-10-19T20:43:47-07:00"
      dispatched: "2026-10-19T20:43:00-07:00"
      agent: claude
      reviewer: claude (anthropic; author anthropic)
      recipe: fix-review
      isolation: fenced
      reviewed: 2ac8c028e16bae89de44f0749effb18805650f1a
      blocked: false
---

# Gate ledger — pair#695 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-19T07:40:57-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `contract-shape-consistency` Validate() is said to fill TeamIdentity.Scope, but §3 declares it with a value receiver returning only error
  This is the 3rd finding in this family. The rule: a Revision that changes a contract must also edit its signature in §2/§3, not just the prose. Here, use a constructor (for example NewTeamAddress) or have Validate return the resolved LifecycleAddress, so that Key() stays pure over a resolved Scope.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-10-19T07:40:57-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Critical] `repo-invariant-tests-red` (kind: defect) artifactpath enforcement tests fail at head (pass at base): unclassified artifactpath import and dead constructors
  TestProductionArtifactReferencesAreExactlyClassified flags couchcore/transcript_bind.go importing artifactpath; TestNoProductionSymbolIsReferencedOnlyByTests flags NewSlotAddress and NewTeamAddress (lifecycleaddress.go) as test-only. Classify transcript_bind.go correctly, and wire or allowlist the constructors with a reason.
- **BR-3** [Important] `preserving-encoder-single-writer` (kind: defect) replaceSlotCurrent marshals ThreadRecord directly and drops Unknown members on recovery rewrites
  slotrecovery.go:148 uses json.MarshalIndent(next), and Unknown is tagged json:"-". Recovered survivors (slotrecovery.go:524/594, resume_route.go:292, recovery_prepared.go:497) lose a newer build's fields; a probe test reproduced it. The guard TestThreadRecordWritersUseThePreservingEncoder only matches toPersistedThreadRecord, so it misses this writer. Use encodeThreadRecord, widen the guard, and add a recovery round-trip test.
- **BR-4** [Important] `field-without-consumer` (kind: defect) TranscriptBind.Note is never read in production; the spec's row note never reaches the operator
  resumeEvidence and resumeContextWith keep only Resolution and Store. The "ledger unreadable/missing; resumed the stored binding" note required by P2 is dropped. Surface it at the row or resume result and test it there, or record a deferral Revision.
- **BR-5** [Important] `decode-strictness-beyond-types` (kind: defect) Decode-time Transcript.Validate refuses the whole record instead of "Start fresh, invalid stored binding"
  threadrecord.Validate (record.go:149) runs the semantic location check (NativeRootNames and RootTranscriptID, this build's catalogues) during DecodePersisted. A hand-edited binding, or a newer build's binding with a new root or agent, makes the whole record undecodable, which contradicts the P2 spec and §2a tolerance. ResumeTranscriptInvalid becomes nearly unreachable. Move the location check into Bind.
- **BR-6** [Minor] `plan-code-drift` (kind: defect) lifecycleFailureTargetKey does not delegate to LifecycleAddress.Key() as the plan claims; NewSlotAddress does not canonicalize its root
  The code shares slotLifecycleKey instead, which is fine in substance. Update the plan, or wire Key(), and document that the caller must canonicalize the root.

## Round 3 — 2026-10-19T08:17:38-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — PQ-7 Revision gives the constructor signatures (NewSlotAddress/NewTeamAddress); code Validate is a pure shape check, Scope filled by NewTeamAddress (lifecycleaddress.go:55-79).
- BR-2 — addressed — artifactpath tests pass at head; transcript_bind.go classified scoped-ledger ResolvedConsumer (manifest.go), NewSlotAddress now used by LifecycleFailureSlotKey, NewTeamAddress allowlisted with pair#694 reason.
- BR-3 — addressed — ThreadRecord.MarshalJSON = encodeThreadRecord; reverting it makes TestThreadRecordJSONKeepsUnknownMembers fail (verified in checkout).
- BR-4 — addressed — Note field removed and deferral Revision recorded (plan M1 review round 1, BR-4) as the finding allowed; receiver #684 tracker does not yet carry it (new Minor).
- BR-5 — addressed — Decode no longer runs Transcript.Validate; restoring it makes TestInvalidStoredBindingKeepsTheRecordReadable fail (verified); Bind refuses or replaces via ValidateLocation.
- BR-6 — addressed — LifecycleFailureSlotKey delegates to NewSlotAddress(...).Key(); NewSlotAddress documents caller canonicalizes; pinned by TestLifecycleAddressSlotKeyMatchesFailureSlotKey.

### Raised

- **BR-7** [Important] `nested-unknown-preservation` (kind: defect) Nested transcript object is strict: a newer build's member inside "transcript" makes the whole record unreadable
  Probe: CreateThread with a binding, add "minted_at" inside transcript on disk, GetThread fails with json: unknown field "minted_at". record.go:113 keeps nested objects strict, contrary to ARCH-EVOLVE "including nested members"; the result is the same whole-record refusal BR-5 removed. Rule: every nested object this issue introduces or will extend preserves unknown members. Give TranscriptBinding Unknown plus UnmarshalJSON/MarshalJSON via strictjson.SplitObject/EncodeObject, add a nested round-trip test, and state the nested policy in plan §2a.
- **BR-8** [Minor] `plan-code-drift` (kind: defect) LifecycleFailureSlotKey now requires a clean absolute path; a non-clean absolute StartingPath silently moves from slot: to thread: key
  This is the 2nd finding in family plan-code-drift. Rule: a Revision that routes an existing #683 contract through a new constructor records any narrowing of its accepted inputs. Previously filepath.IsAbs; now workspaceAbsolute (Clean==p, valid UTF-8, no NUL). Record the narrowing in the plan, or pin it with a test.
- **BR-9** [Minor] `deferral-recorded-at-receiver` (kind: defect) BR-4's row note and legacy-ledger row detail are deferred to #684 M2 only in #695's plan; #684's tracker does not mention them
  Add the deferred items to the #684 issue (or its plan) so the receiving issue owns them. Also, transcript_bind.go:134 says "stored transcript binding is invalid" when the invalid binding came from the ledger proof.

## Round 4 — 2026-10-19T08:44:32-07:00 (claude) — passed

### Disposed

- BR-7 — addressed — TranscriptBinding has its own JSON methods on strictjson (transcript.go:34-73). With them reverted, TestTranscriptBindingNestedUnknownMembersSurvive fails with unknown field future_tty; it is the only new nested persisted type.
- BR-8 — addressed — Narrowing recorded in the plan's round-2 Revision and pinned in TestLifecycleAddressSlotKeyMatchesFailureSlotKey (lifecycleaddress_test.go:157-168), which passes at head.
- BR-9 — not-addressed — The refusal text is fixed (transcript_bind.go:125,134). But no #684 plan exists in any ref here, and the #684 issue file is unchanged since f56df570d, so the receiver still does not own the deferred items.

### Raised

- **BR-10** [Minor] `nested-unknown-preservation` (kind: defect) Record.Unknown comment still says nested objects stay strict after BR-7 made transcript tolerant
  This is the 2nd finding in family nested-unknown-preservation. The rule is already in the round-2 Revision: a persisted type born in this issue is tolerant and preserving from birth; older nested types stay strict. record.go:110-114 still states the old policy. Reword it to that rule, so the code no longer contradicts the plan.

## Round 5 — 2026-10-19T12:31:56-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Overtaken: the team arm and TeamIdentity were removed in M2 (009030459); Validate/Key now have one slot kind.

### Raised

- **BR-11** [Critical] `proof-witness-per-process-kind` (kind: defect) Teardown proves a tree down while the published agent is still alive when agent-ready carries no tty
  teardown.go teardownHolders and proveDown witness the agent only through TTYProcesses(table, tty), and TTYProcesses with "" matches nothing. So for any pre-M2 wrapper, or a wrapper whose agentTerminal table read failed, the proof has no agent witness. The plan's fallback (proven as today by pid and session, ObserveSlotSessions) was dropped. A reviewer probe (agent published with no tty that ignores SIGTERM) returned err=nil with the agent Live and LeftRunning empty, and ArchiveThread would archive with that proof. Fix: when the published agent is listed, sweep and require its exact identity Dead whatever the tty (holder kind agent), and add a regression test.
- **BR-12** [Important] `kill-guard-regression-test` (kind: defect) The tty-trusted-only-while-the-tree-is-up guard has no test; a mutation that always trusts the tty passes the suite
  Changing teardown.go:161 to `if true || agentListed(...) || !free` passed every couchcore Teardown, Retire, Archive, Fresh, Reboot and Recover test (183s). This guard is what stops teardown from SIGKILLing an unrelated terminal's processes on a reused device. Add a test: agent dead, lease free, an unrelated live process on the same tty number; expect no signal and the note.
- **BR-13** [Important] `artifact-removal-path` (kind: defect) Per-thread lifetime lease files are removed only by Teardown(stop); every other retirement leaks one file forever
  Lease keys include scope/tag (lifetime_lease.go:20), and only teardown with stop removes the file (teardown.go:208). Retirements through ProveDown (startFreshSlotWithPrompt, reboot archive-only), proveDownAndFold (Fresh recovery), AdoptionAuthority and UnreadableRecordEscape leave <namespace>/leases/<digest>.lease with no sweep, so the directory grows with every Fresh, reboot or adoption (ARCH-FUNERAL). Remove the retired thread's lease once its proof stands, or sweep leases whose thread is gone, and state the lifetime in the atlas.
- **BR-14** [Important] `authority-unforgeable` (kind: defect) TeardownProof is forgeable (all fields exported), unlike the three other constructor-only retire authorities
  RetireAuthority is sealed at the interface, but any package can build couchcore.TeardownProof{Address, Thread, Revision, ProvenAt}, and couchcoretest/retire.go:25 (a non-test package) does. That undercuts ARCH-ORDER's structural enforcement that the archive and replace funnels cannot be reached without a real proof. Unexport the authority-bearing fields (with accessors), so only proveDown mints one, and give fixtures an explicitly test-only minting path.
- **BR-15** [Important] `readme-surface-gate` (kind: defect) README update missing: couch --reap-orphans --confirm now also ends teardown's left-running processes
  reap_orphans.go adds reapLeftRunning to the destructive verb, and --resources gains the "Left running by teardown" section. README.md:477-478 and :822-828 still describe only unattributed embedded nvim, so an operator who uses --confirm for nvim cleanup will now also kill setsid'd runners and background merges without the docs saying so.
- **BR-16** [Important] `plan-code-drift` (kind: defect) M2 plan contracts and named tests not delivered as specified (plain-pair test cases, both-env activation, sweep test, recorder path, holder kinds, witnesses)
  This is the 3rd finding in family plan-code-drift. Rule: at milestone close, every test name and contract in the plan's governing revision is checked mechanically against the tree (grep the func names; diff each contract against the code), and each mismatch gets either code or a Revisions line in the same round. Measured prevalence in M2 is 7 instances. (1) TestPlainPairUnchangedWithoutCouch checks only HoldFromEnv with an empty env, not two sessions, a sidecar without a tty, no extra fd, or the lease-without-tag case. (2) Activation needs only COUCH_LIFETIME_LEASE (lease.go:60, wrap.go:652), not both env vars. (3) TestTeardownSweepsReparentedHolders is absent. (4) TestTeardownUnprovenRecordsPhaseAndHolders does not go through the real recorder. (5) ReadyRecord is tolerant but not preserving, unrecorded. (6) The agent and session-watch holder kinds were not added (pair/process used). (7) The plan's four witnesses became three (ObserveSlotSessions dropped).
- **BR-17** [Important] `decode-strictness-beyond-types` (kind: defect) orphans.json accepts any schema_version and rewrites it as 1; entries with pid <= 0 or an empty identity decode
  This is the 2nd finding in family decode-strictness-beyond-types. Rule: every persisted envelope validates its known-field values (supported schema_version, positive pid, non-empty identity) at decode, not only their JSON types, and a rewrite never changes an existing field's value. orphan_registry.go:201 decodes with no value checks, and :282 sets SchemaVersion=1 over whatever was read, which changes a newer writer's field (ARCH-EVOLVE). Refuse an unsupported version, validate entries, and add round-trip and refusal tests.
- **BR-18** [Minor] `preserving-codec-boilerplate` (kind: defect) The split, encode, strict-decode, keep-unknown sequence is copy-pasted at five decode sites
  transcript.go:47, record.go:324, readiness/record.go:61 and orphan_registry.go:81/114/142 repeat the same four steps. A strictjson helper (decode the known members into a wire struct, return the unknown map) would make each type's UnmarshalJSON a single call (ARCH-DRY).
- **BR-19** [Minor] `mode-as-bool-flags` (kind: defect) teardown(stop, fold bool) encodes three legal modes as two booleans; stop without fold is representable but unused
  Replace the two booleans with a mode enum (teardown, prove-down, prove-and-fold) so the legal set can be read off the type (ARCH-ORDER).
- **BR-20** [Minor] `comment-code-drift` (kind: defect) ReadyRecord.TTY comment says names like ttys183 or pts/3, but the value is a decimal device number
  procutil.Process.TTY stores e_tdev or tty_nr in decimal, and the wrapper publishes that same value. Fix the comment at readiness/record.go:30.

## Round 6 — 2026-10-19T13:16:10-07:00 (claude) — passed

### Disposed

- BR-11 — addressed — teardown.go:365-369 adds the published agent by exact pid+identity; mutation (drop add) turns TestTeardownWitnessesAnAgentWithoutATerminal red.
- BR-12 — addressed — TestTeardownNeverSweepsAReusedTerminal fails under the `if true ||` mutation (pid 900 terminated), verified in this checkout.
- BR-13 — addressed — Lease removed after a proof in every mode (all-mode test fails when limited to stopping mode); SweepFree at Couch start with TestSweepFreeRemovesOnlyFreeLeases; atlas states the lifetime.
- BR-14 — addressed — Authority fields unexported with accessors; fixtures use TestOnlyTeardownProof plus a scan test (scan-scope gap raised as a new Minor).
- BR-15 — addressed — README.md:477-481 and :823-829 now name the left-running processes that --reap-orphans --confirm ends and the --resources section.
- BR-16 — addressed — Each of the 7 mismatches got code or a Revisions entry (plan round-4 log); sweep and recorder tests exist, and Active is the single activation predicate.
- BR-17 — addressed — orphan_registry.go refuses schema_version != 1 and pid<=0 or empty identity at decode, and the rewrite keeps the version; refusal + byte-unchanged test present.
- BR-18 — addressed — strictjson.DecodeKnown replaces the five sites; no remaining copy of the split/encode/decode sequence in the diff.
- BR-19 — addressed — teardownMode enum with stops()/folds() replaces (stop, fold bool).
- BR-20 — addressed — readiness/record.go:30 comment now says decimal device number (Darwin e_tdev, Linux tty_nr).

### Raised

- **BR-21** [Minor] `unlink-lock-file-under-lock` (kind: defect) Teardown unlinks the proven-free lease without holding its exclusive lock, unlike SweepFree
  teardown.go:245 calls os.Remove after proveDown's Free released its probe lock, so a Hold that locks between the two keeps an unlinked inode that every later witness misses. Rule: a lock file is unlinked only under LOCK_EX with a same-file check; share one lifetimelease removeIfFree helper used by SweepFree and teardown.
- **BR-22** [Minor] `authority-unforgeable` (kind: defect) The production-caller scan for TestOnlyTeardownProof covers only cmd/internal and exempts couchcoretest
  This is the 2nd finding in family authority-unforgeable. Rule: an authority's test-only minting path is unreachable from every production package in the module, so the guard walks the module root and also refuses non-test imports of couchcoretest. retire_test.go walks ".." (cmd/internal) only, so cmd/couch and cmd/pair-go are unscanned.

## Round 7 — 2026-10-19T17:35:29-07:00 (claude) — passed

### Disposed

- BR-9 — addressed — The #684 issue's M2 checklist now owns both items from pair#695 M1 (BR-4), the stored-binding row note and the legacy-ledger row detail, and its log records receiving them. The transcript_bind.go:134 message now reads "its transcript binding is invalid", which is neutral about where the binding came from.
- BR-10 — addressed — record.go:113-118 now says nested types older than pair#695 stay strict and a nested type born additive (TranscriptBinding) keeps its own unknown members, matching the round-2 Revision.
- BR-21 — addressed — teardown.go:245 now calls lifetimelease.RemoveFree, the same helper SweepFree uses: it takes LOCK_EX|LOCK_NB and checks the path still names the same file. TestRemoveFreeLeavesAHeldLease fails at lease_test.go:242 when the lock check is removed.
- BR-22 — addressed — retire_test.go now walks the module root and also refuses non-test imports of couchcoretest. A planted cmd/zzprobe/p.go was flagged at retire_test.go:326 and :329.

## Round 8 — 2026-10-19T19:25:33-07:00 (claude) — passed

## Round 9 — 2026-10-19T20:43:47-07:00 (claude) — passed

## Open findings

(none — every finding has been disposed)
