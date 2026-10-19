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

## Open findings

- **BR-9** [Minor] `deferral-recorded-at-receiver` (kind: defect) BR-4's row note and legacy-ledger row detail are deferred to #684 M2 only in #695's plan; #684's tracker does not mention them
- **BR-10** [Minor] `nested-unknown-preservation` (kind: defect) Record.Unknown comment still says nested objects stay strict after BR-7 made transcript tolerant
