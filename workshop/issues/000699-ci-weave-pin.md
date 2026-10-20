---
id: 000699
status: codecomplete
deps: [ariadne#459]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'abdc72132efbef76efe1c452b864a2871d4f5d85' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T21:32:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.05
---

# CI: build weave from a pinned ariadne commit

## Problem

ariadne#459 reproduced the cold generated-vocabulary failure in published weave
0.1.0, before ariadne#422's shipped-output sidecar ownership. This repository's
active GitHub main merge-check installs the published formula without an exact
source pin. Verified 2026-10-19 against [the exact workflow blob](https://api.github.com/repos/xianxu/pair/git/blobs/91cfc2012e8f61ff8cfac85b2e7159e0c085711e).

## Spec

Repository-owned migration after ariadne#459's upstream compatibility guard
lands. Build weave from approved full Ariadne commit
63d6bfd86509d97649e10d7a5dd1e9a06db17d21 before bootstrap, using an authored
.sdlc/weave-compiler-ref outside weave's managed manifest. Follow the reviewed
upstream merge-check template. Install Go from that pinned source's go.mod;
verify fetched HEAD and binary vcs.revision/modified metadata, and prepend its
private output directory to PATH. No floating HEAD/formula or old PATH fallback
on pin acquisition/build failure. Preserve bootstrap, setup and check ordering.

## Done when

- The committed workflow builds the exact selected supported full SHA and uses
  that executable before bootstrap; tests reject malformed pins and fetch/build
  failure and prevent an older PATH weave from winning.
- A cold CI checkout with tracked old generated vocabulary and matching sidecars
  regenerates changed exports and sidecars without authored-replacement refusal;
  an authored/tampered output still refuses without losing its bytes.
- The consumer-authored pin survives compile and is not manifest-managed.
- Operator verifies the local-gates PR and applies local-gates-verified. Agents
  never apply that label. Future pin bumps require the same verification.
- The owner reports the PR, compiler identity and real CI acceptance to the hub.

## Plan

The payload is the weave refresh output and one authored pin, with no hand-written workflow:

- [x] `weave refresh --substrate` on this issue branch, with the slot's ariadne at 0956e5e1 (it contains #459's
  merge 401e1ebc0 and the approved 63d6bfd8). It regenerates `.github/workflows/merge-check.yml`, byte-identical to
  ariadne's template, and `.github/workflows/local-gates-guard.yml`, which adds `scripts/ci/*` to the guarded paths.
- [x] Author `.sdlc/weave-compiler-ref` = `63d6bfd86509d97649e10d7a5dd1e9a06db17d21` plus a newline. This is
  ariadne's sole supported compiler (`.sdlc/weave-compiler-refs`).
- [x] Prove the pin is consumer-owned: `weave compile` leaves its bytes unchanged, and
  `construct/generated/weave/ownership.json` does not list it.
- [x] Recorded suite: sdlc-local/suite success on 13aa8d4d.

Landing follows the close: the PR is a local-gates PR, so the hub gets the operator's `local-gates-verified` label.
Real CI acceptance, the compiler identity line from the build step, goes to the hub.

The tests that reject malformed pins, fetch or build failure, and an older weave earlier in PATH are upstream's
(ariadne `scripts/test/weave-compatibility.test.sh`, `portable-ci.test.sh`), run on ariadne's CI. This repository's
acceptance is its own merge-check run on the PR, building and reporting the pinned compiler.

## Log



- 2026-10-19: closed — sdlc-local/suite success on 13aa8d4d (docs-only commit since); merge-check.yml byte-identical to ariadne #459 template (weave refresh --substrate); pin 63d6bfd8 = ariadne sole supported compiler, unchanged by weave compile, absent from ownership.json; workflow validator refuses 5 malformed pins. Atlas: CI pin contract is documented upstream (ariadne atlas/workflow/ci-merge-check.md); pair adds no surface; review verdict: SHIP
- 2026-10-19: flow upgraded quick → full — 139 added lines in code files (limit 100)
- 2026-10-19: refresh and pin. The workflow's own pin validator accepts the committed bytes and refuses uppercase, a short SHA,
  a missing newline, a branch name and a trailing extra line. The pin's sha1 is unchanged across `weave compile`, and
  ownership.json does not name it.

Filed under explicit hub authorization from this repository's primary :0
checkout, unclaimed, depending on ariadne#459. Source class C; inventory:
[verified active GitHub consumers](https://github.com/xianxu/ariadne/blob/000459-weave-release-downstream-ci-still-runs-weave-0-1-0-pre-422-vocabulary-changes-break-every-downstream-merge-check/workshop/plans/000459-downstream-compiler-inventory.md).
The upstream owner files the hand-off; this repository's assigned slot and the
operator own implementation and acceptance.
