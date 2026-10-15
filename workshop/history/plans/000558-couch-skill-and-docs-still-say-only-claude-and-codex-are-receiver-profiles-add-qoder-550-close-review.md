# Boundary Review — pair#558 (whole-issue close)

| field | value |
|-------|-------|
| issue | 558 — couch skill and docs still say only Claude and Codex are receiver profiles; add Qoder (#550) |
| repo | pair |
| issue file | workshop/issues/000558-couch-skill-and-docs-still-say-only-claude-and-codex-are-receiver-profiles-add-qoder-550.md |
| boundary | whole-issue close |
| milestone | — |
| window | 28d9e6816fd5139ad895ea22d585d7cd6aac9ed9..3a8bf1fcb917a2246362dee4a3b482b03cf755db |
| command | sdlc close --issue 558 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-15T10:37:45-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This change meets the issue's purpose. The embedded couch skill (`SKILL.md:502`) now names Claude Code, Codex CLI and Qoder CLI as receiver profiles, and says Agy, Muse and Grok have no receiver profile and are explicitly disabled. The Spec preferred generating this list. Instead, the implementer kept the skill text static and added a drift test, `TestSkillNamesReceiverProfiles`. It builds the expected sentence from `wrapcmd.PeerReceiverAgents()` over `launcher.AgentInventory()`, so the inventory and policy drive the check and the hand-written sentence cannot drift silently. The Log gives the reason, and it holds up: `messages_test.go:224` requires `couch --skill` to print the embedded file exactly, so filling the list in at print time would make the output differ from the file operators save.

I checked the test is real: with the base-commit SKILL.md swapped in, it fails on both clauses; with the head version restored, it passes. The working tree is clean afterwards. My own re-grep for two-agent lists matches the Log: the hits left are accurate, or they aren't claims about which agents can receive. Nothing blocks shipping. Two Minor notes follow.

1. **Strengths**
   - `cmd/internal/couchcmd/skill_receivers_test.go:26-41` uses `t.Fatalf` if an inventory agent has no receiver policy or no display name. Adding an agent, removing one, or changing its policy all fail the test. I traced the removal case: drop `grok` and the expected disabled clause becomes "Agy and Muse have…", which the skill doesn't contain, so the test fails.
   - `peer_runtime.go:35-41`: `PeerReceiverAgents()` returns a copy of the policy map, so callers can't change the real policy.
   - Expected names follow `AgentInventory()` order (`claude, codex, agy, muse, qoder, grok`). The enabled and disabled lists come out deterministic and match the prose.
   - The Log records the sweep, including why the "claude or codex" text in `peer_live_test.go:19,73` stays. That text describes what the live harness can run (it refuses any other harness), not which agents receive, so it is accurate.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **ARCH-PURPOSE (receiver list still hand-written in docs):** `README.md:691-692` and `atlas/couch.md:593` repeat the receiver list, and no test checks either one. Both are correct today, and the issue's target was the skill, so this is a note for later, not a missing deliverable.
   - **Test edge case (empty disabled list):** `skill_receivers_test.go:46`. If every agent were enabled, `humanJoin` would return "" and the expected clause would start with a stray space. The test would still fail, so the skill sentence would get rewritten, but the failure message would be confusing. A fix is to add the disabled clause only when the disabled list is non-empty.

5. **Test coverage**
   - The regression test fails without the fix (checked by reverting `SKILL.md` to the base commit).
   - The two states in Done-when (enabled and disabled) are each checked by their own clause.
   - The existing `TestPeerReceiverInventoryHasExplicitPolicy` (`peer_qoder_test.go:19`) already checks that the policy and the inventory cover each other.
   - A stale entry left in `receiverDisplayNames` would not be flagged. That's harmless.

6. **Architectural notes**
   - **ARCH-DRY: pass.** `humanJoin` has no existing equivalent anywhere in `cmd`. The accessor reads the single policy map rather than copying it.
   - **ARCH-PURE: pass.** The test is pure: no file, network or process IO, since the skill is an embedded string.
   - **ARCH-PURPOSE: pass**, apart from the Minor note above. The display-name map lives only in the test. If a second consumer ever needs these names, it should move to `launcher`.

7. **Plan revisions:** none. The plan still matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: receiver-list-single-source
    title: |
      README and atlas receiver lists remain unpinned hand-maintained restatements
    detail: |
      README.md:691-692 and atlas/couch.md:593 restate the receiver set (Claude Code, Codex CLI, Qoder CLI) without the inventory-derived pin that now guards SKILL.md:502; both are correct today. Consider extending TestSkillNamesReceiverProfiles' expected-sentence check to these two sites.
  - id: new
    severity: Minor
    kind: defect
    family: test-expectation-edge-cases
    title: |
      Receiver drift test builds a malformed disabled clause when no agent is disabled
    detail: |
      skill_receivers_test.go:46 concatenates humanJoin(disabled) with " have no receiver profile..."; with an empty disabled set that yields a leading-space clause. It still fails loudly, but guard the clause on len(disabled) > 0 for a clear message.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"308edd2c-1f66-4ab5-be07-39765eea1e54","repository":"github.com/xianxu/pair","issue_id":"000558","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/06a5e1c74c178dbd","reviewed_head":"3a8bf1fcb917a2246362dee4a3b482b03cf755db","slot":"pair:4","requested_at":"2026-10-15T17:36:47.347031Z"},"phase":"finished","pid":62436,"started_at":"2026-10-15T17:36:47.454552Z","finished_at":"2026-10-15T17:37:44.653925Z","session_id":"7e5934c3-1039-4cf6-ad85-558d1613e578","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.716767417s
head: "3a8bf1fcb917a2246362dee4a3b482b03cf755db"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
