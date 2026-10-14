---
id: 000514
status: done
created: 2026-10-12
updated: 2026-10-13
estimate_hours:
github_issue:
started: 2026-10-12T18:20:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: 0.35
tracker:
    version: 1
    completion:
        token: close-32a136215aa4
        repository: github.com/xianxu/pair
        reviewed_head: 89d38a9738b732431c815845a2e6b4a3a5933ba5
        evidence_commit: 86bd8e8074a4d55f931b34b8aeb7b96da8a2c61b
        landed_commit: 40dc5b473c165f7a0ecdc908bfc05e2b3a962412
    actual_correction:
        version: 1
        token: actual-correction-a9b6e5584874
        completion_token: close-32a136215aa4
        repository: github.com/xianxu/pair
        evidence_commit: 86bd8e8074a4d55f931b34b8aeb7b96da8a2c61b
        reviewed_head: 89d38a9738b732431c815845a2e6b4a3a5933ba5
        previous_actual: "23.76"
        new_actual: "0.35"
        reason: Repair released-interval inflation and old child attribution using ownership-proven evidence (#335).
        intent_reason: Repair released-interval inflation and old child attribution using ownership-proven evidence (#335).
        measurement_digest: ead8ddb3fe2929f323135140ace04de254d22a9bef6289c937b365e8b200c6a4
        measurement_inputs: '{"version":1,"method":"active-time-v3-engagement-v1","repository":"github.com/xianxu/pair","code_head":"89d38a9738b732431c815845a2e6b4a3a5933ba5","tracker_head":"d6c76cb86c7d1ab461ddb0a578aa11c7a06651f4","branch_point":"1ef66ff23734f63b866852c147cbe78f4c356704","since":"2026-10-12T18:20:35-07:00","until":"2026-10-13T18:51:45-07:00","brain_root":"/Users/xianxu/workspace/brain","gap_minutes":15,"commit_weight":1,"include_assistant":true,"engagements":[{"start":"2026-10-12T18:20:35-07:00","end":"2026-10-12T18:21:14-07:00","began":"2026-10-12T18:20:35-07:00","roots":["/Users/xianxu/workspace/worktree/pair-slot6/pair"]},{"start":"2026-10-13T18:29:07-07:00","end":"2026-10-13T18:51:45-07:00","began":"2026-10-13T18:29:07-07:00","roots":["/Users/xianxu/workspace/worktree/pair-slot6/pair"]}],"sources":[{"harness":"claude","id":"9ec9e2de-c27e-4d23-988d-f5b646a40862","cwd":"/Users/xianxu/workspace/worktree/pair-slot6/pair","digest":"054d3000d2a7ba6ff86fcea3013cbc9e40d5748d6bece1b0c3b4cea23e5f74df","started":"2026-10-12T17:27:59.582Z","spawned":false,"bytes":10593066,"windows":[{"start":"2026-10-12T18:20:35-07:00","end":"2026-10-12T18:21:14-07:00"},{"start":"2026-10-13T18:29:07-07:00","end":"2026-10-13T18:51:45-07:00"}]},{"harness":"codex","id":"01a13646-a9b3-7443-a36e-43f301879fd6","cwd":"/Users/xianxu/workspace/worktree/pair-slot6/pair","digest":"2779f7a0ab55427be062ced2f78a67c9520fff320aa7ff4b466647d535a99122","started":"2026-10-13T17:19:50.728Z","spawned":false,"bytes":31277885,"windows":[{"start":"2026-10-13T18:29:07-07:00","end":"2026-10-13T18:51:45-07:00"}]}],"cuts":[{"dispatched":"2026-10-13T18:50:13-07:00","finalized":"2026-10-13T18:51:43-07:00","finalizing":true}]}'
        exclude: false
---

# test-perf-capture asserts a 6 s wall-clock budget on a shared loaded machine; fails under load on main

## Problem
