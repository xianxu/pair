# Boundary Review — pair#318 (whole-issue close)

| field | value |
|-------|-------|
| issue | 318 — Declare Pair post-move build |
| repo | pair |
| issue file | workshop/issues/000318-declare-pair-post-move-build.md |
| boundary | whole-issue close |
| milestone | — |
| window | c3acba714ca3596c92f4da4025bafb2bcc7962c2..defbbac5383de05e91b18f4f606c9e2d7c01d160 |
| command | sdlc close --issue 318 |
| reviewer | codex |
| timestamp | 2026-09-23T23:57:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Pair-specific instruction correctly names the post-move build, HEAD verification, fresh-session requirement, and shared procedure. One unrelated issue-317 plan and project-history update are included in this issue-318 range and should be separated or explicitly justified.

1. Strengths

- `AGENTS.local.md:8-11` fulfills the requested operational instruction.
- The generated `AGENTS.md:119-122` contains the composed instruction.
- The wording points to the existing slot-move procedure and preserves the Pair-only scope.
- `git diff --check` passes.
- No README or atlas update appears necessary; this adds no new user-facing command or architectural surface.

2. Critical findings

None.

3. Important findings

- `workshop/plans/000317-slot-quick-status-glyph-plan.md:1` — unrelated durable plan is included in the `pair#318` review window. Remove it from this branch/range or document why issue 318 owns it.
- `workshop/projects/couch-slots-v2.md:394,522,991,1008` — unrelated project archival/status changes are included in the same range. Enumerated family: all non-318 artifacts in this window consists of the issue-317 plan and these four project updates. Separate them or explicitly record the intended scope.

4. Minor findings

None.

5. Test coverage notes

This is documentation-only behavior. The authored fragment and current generated `AGENTS.md` were inspected; the issue log records composition through the real Weave pipeline. No runtime test is required.

6. Architectural notes for upcoming work

- ARCH-DRY: pass — the change reuses the shared slot-move procedure and does not alter build/bootstrap logic.
- ARCH-PURE: pass — no executable or side-effecting logic was added.
- ARCH-PURPOSE: pass for the Pair instruction itself; the unrelated plan/project artifacts weaken issue-range traceability.

7. Plan revision recommendations

- Add a `## Revisions` entry only if the unrelated issue-317/project changes are intentionally part of this boundary; otherwise remove them from the issue-318 range.
