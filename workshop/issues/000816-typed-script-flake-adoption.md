---
id: 000816
status: open
deps: [ariadne#556, ariadne#558]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '05ae38ee3265855470eb7732bf6a5d7db4b3fa34' # card fields mirrored from issue-cards; edit via sdlc
---

# Adopt typed script flake evidence

## Problem

The recorded Pair suite still invokes tests/nvim-route-poke-admission-test.sh directly. Pair#773 retained its 30-second run_headless timeout as an unattributable failure (tracked by #685), so it could not be registered or resumed. Ariadne's typed implementation and producer distribution must land before this local-gates activation.

## Spec

Adopt ariadne#556 after ariadne#558, with capable fleet binaries deployed first. Leave this issue unclaimed for Pair's owning TL to dispatch. This is a local-gates change: the operator holds verification and the local-gates-verified label; an agent must never self-apply it.

Exact adoption steps:

1. Upgrade .sdlc/local-checks.json from version 1 to version 3, preserving the suite's existing command [make, PAIR_GO_TRIMPATH=-trimpath, test], required flag, weight and timeout. Add suite flake_runners.script.command as [bash, {path}], where {path} is a complete argv element.
2. At Makefile.local's test-lua recipe, replace the managed invocation of bash tests/nvim-route-poke-admission-test.sh with sdlc flake run --unit script:tests/nvim-route-poke-admission-test.sh. Keep the existing direct bash recipe outside a typed-capable managed invocation. Change no other test-lua members, test selection, assertions or timeout budget.
3. The native script still calls tests/lib/run-headless.sh and nvim/route_poke_admission_test.lua. The wrapper observes the real script exit, including 124; stdout must never manufacture an identity. Preserve the existing headless test isolation and verify cancellation/descendant cleanup with this actual runner, not only a trivial shell fixture. The enclosing managed owner must remain responsible for cleanup.
4. Record a new typed-capable same-head failure and subsequent complete pass, then register script:tests/nvim-route-poke-admission-test.sh on the appropriate open flake issue (currently #685, subject to its state at adoption). Resume the exact original typed attempt and verify that only this whole script reruns, with a linked child/status and unchanged original evidence. Do not register from an unverified pass or force acceptance.
5. Historical Go-only/unattributable attempts, including #773's retained timeout, remain byte-for-byte unchanged and cannot be retroactively resumed. Registration requires NEW typed-capable evidence on the SAME exact head: a failure and a complete pass. Changes to the script between runs are not same-head proof.

## Done when

- Operator verification authorizes the version-3 local-gates declaration; capable binaries are deployed before activation.
- A real timeout is recorded as the exact script identity with its observed exit, and a new same-head fail/pass permits registration and exact saved resume.
- Missing completion, cancellation and unrelated passes cannot turn the attempt green; original logs stay unchanged and all owned descendants are joined.
- Ordinary direct make behavior, suite membership, assertions and timeout limits stay intact.

## Plan

Plan with Pair's TL after both dependencies land; implement the declaration and managed recipe together, add real production-runner conformance, obtain the operator's local-gates verification, then close/land through the normal gates. No activation is authorized by filing this issue.

## Log

### 2026-10-22

Filed unclaimed at ops:0's request from ariadne:3. Source inspected read-only: Makefile.local test-lua currently runs the script directly, and .sdlc/local-checks.json is version 1. Peer AGENTS.local.md read; root MEMORY.md is absent. No checkout files or existing branch work changed. See ariadne#556's native/private acceptance and atlas for the typed proof contract; distribution and peer activation remain separately governed by ariadne#558.
