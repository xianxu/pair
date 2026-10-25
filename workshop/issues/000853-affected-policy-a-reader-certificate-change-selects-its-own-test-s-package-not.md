---
id: 000853
status: codecomplete
deps: [ariadne#598]
github_issue:
created: 2026-10-23
updated: 2026-10-25
estimate_hours:
card_mirror: 'af3c48383cf12bb574c76e0e0ac9f1c1f8c4d4b6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T10:26:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "92f102e0", done: "ea157c89"}
actual_hours: 0.34
---

# Affected policy: a reader-certificate change selects its own test's package, not the full suite

## Problem

## Spec

**pair's affected-test policy falls back to the full suite whenever a reader certificate changes (found on the first
day of affected boarding, 10-23).** Several changes today ran a full suite under `--pre-close` only because they
touched a file under `tests/affected-readers/…` (the per-test reader certificates): pair#825 (its
`request_lifetime_test.go.json`), pair#679 and pair#739. Any test fix that changes what a test reads updates its
certificate, so most test changes get the full suite, which defeats the point of boarding on the affected check.

Fix: in pair's affected policy (`.sdlc/affected.json` and its selector), map a reader-certificate change to the
package (or test group) whose certificate it is, plus whatever that certificate's reads select, instead of selecting
everything. Keep the full-suite fallback for changes to the policy itself, the selector, or shared harness code.

## Done when

- A change to one test's reader certificate selects that test's package (and the reads it declares), not the full
  suite (test with a real certificate change, e.g. pair#825's).
- Changes to the policy, selector or shared harness still select the full suite (test).

## Plan

Consumer of ariadne#598 M1 (installed sdlc 2bc11af1 contains merge 2001041f0). ariadne owns selection; pair only
declares (ARCH-DRY). Quick flow: production change is a few lines of `.sdlc/affected.json`; the rest is tests.

**Local-gates:** `.sdlc/*` is on `local-gates-guard.yml`'s protected list, so the PR needs the operator's
`local-gates-verified`; I never apply it.

- [x] Repair the qualification harness (`tests/affected-config-test.py`, run by `make test`). On the current provider it
  fails to compile: `jsonstrict` moved to `cmd/internal/jsonstrict`, and `Plan` takes `(paths, g, cfg, m, isDoc)`.
  Copy the moved package, call `affected.Compile(cfg, sidecars)` with the repository's real certificates, and pass the
  mapping to every `Plan`. The provider gate requires #598 (`SidecarFile` in `mapping.go`), since an older planner's
  strict parse refuses the new members. Older providers make the test skip, not pass.
- [x] `.sdlc/affected.json`, additive members only:
  - `"sidecars": [{"prefix": "tests/affected-readers/", "suffix": ".json"}]`: a certificate path aliases its test,
    so it selects that test's package and its dependents.
  - `"tests/**"` becomes `{"glob": "tests/**", "except": ["tests/affected-readers/**"]}`. Everything else under
    `tests/` stays full: the selector's tests, the inventory tool, `tests/lib/**` and the shell/Lua tests.
  - `"cmd/couch/**"` becomes `{"glob": "cmd/couch/**", "except": ["cmd/couch/*_test.go"]}`. A test file is not
    compiled into the couch binary that native consumers build, so a test-only edit selects `cmd/couch` alone. Its
    production files stay full, and `cmd/couch/testdata/**` is still full through `cmd/**/testdata/**`.
- [x] Qualification cases against the real planner and real certificates:
  - `tests/affected-readers/cmd/internal/testcouch/agentfixture/request_lifetime_test.go.json` (#825) selects
    `…/testcouch/agentfixture` and its dependents, not full.
  - `cmd/couch/main_test.go` and `cmd/couch/editorial_acceptance_test.go` select `cmd/couch` exactly, not full;
    their certificates do the same.
  - Added, deleted and renamed certificates (path-only alias). A certificate whose test directory has no package goes
    full. A certificate path that does not end in `_test.go.json` goes full.
  - Still full: `.sdlc/affected.json`, `scripts/test-affected.py`, `tests/test-affected-test.py`,
    `tests/affected-go-readers-test.py`, `tests/affected-config-test.py`, `tests/lib/x`, `cmd/couch/main.go`,
    `cmd/couch/sub/x_test.go` (`*` does not cross `/`), and a certificate together with a full path.
  - A fixture certificate with `reads: ["nvim/x.lua"]` shows the declared read selecting the test's package. That
    input's own `full` rule still wins.
- [x] Recorder: `--record` keeps `reads` byte-for-byte when it refreshes a certificate's hash (a test beside the
  existing unknown-member round-trip test). Every affected run already runs `make test-affected-inventory`
  (`scripts/test-affected.py:500`), so a certificate-only change is still audited.
- [x] Docs: README's certificate paragraph and the atlas affected entry say what a certificate change selects, and
  that `reads` is optional and narrows nothing by itself.

**Out of scope (filed unclaimed as pair#893, TL-approved):** adding audited `reads` to the 589 real certificates. Every
non-Go input a reader uses is covered today by a `full` glob or a group. A `reads` entry only adds the reader's package
to an input's selection; it never removes the input's own `full` rule. So until those `full` rules narrow, populating
`reads` changes no selection, and it costs one audit per reader. The issue's "(and the reads it declares)" is
satisfied by the mechanism and the fixture case above.

- [x] Bump pair's `go.mod` pin of ariadne to main 9678ee653 (contains #598), so the suite qualifies against a #598
  planner instead of skipping (Revision 2026-10-25).

## Log


- 2026-10-25: unclaimed: TL-directed handoff to pair:6 (delivery 8574f573-f5e1-40b6-b447-be6ab09aaa92). Preserve plan-wait commit 2522724fa851d612fb4222da91b60b0c2d35a5fe and existing Log; no implementation or review was started. ariadne:5 reports ariadne#598 M1 landed as PR435 merge 2001041f06cd576094266cc4b562c09cec7d0858: declared input ownership (owners/full excepts/sidecars with reads, read from main) is available after rebuilt ariadne:0 and weave refresh. Next owner verifies current capability and dependency, resumes planning for reader certificates plus cmd/couch test-only changes, sends plan to hub before change-code. pair:1 remains on #880 waiting for #857 landing.
### 2026-10-23


### 2026-10-24 — provider prerequisite; paused before implementation

Claimed and entered planning on an independent branch from main. The blocker is structural: `.sdlc/affected.json` has full overrides `tests/**` and `cmd/couch/**`; removing the former leaves certificates unclassified. Ariadne `affected.Plan` unions native Graph.Owners with group paths, but Group.Go only triggers groups and cannot add packages. Current certificates contain disposition/hash, not declared reads. Reverse imports do not express runtime executable/fixture reads. ARCH-DRY: keep selection in the provider, rather than duplicating it in Pair or creating hundreds of test groups.

Ops approved the provider split (delivery e3fddf94-32aa-4044-99a5-c68e0cd84276). Filed **ariadne#598**, unclaimed, `workshop/issues/000598-affected-declared-input-ownership.md`; ariadne:5 takes it after #592 lands. Contract: declared non-Go path ownership adds Go packages through the existing union/closure, certificates record audited reads, malformed/unknown/deleted mappings fail conservatively, and policy/selector/shared harness fallbacks remain full. Set this issue's deps accordingly. No production or test code changed; no tests or passing measurement claimed.

**NEXT ACTION:** after ariadne#598 lands, resume this branch, integrate main and the provider, audit all certificate readers, finalize the consumer plan and obtain TL approval before change-code. Include #825's request_lifetime certificate and cmd/couch main_test.go/editorial_acceptance_test.go exact selections, unknown/deleted/renamed cases, explicit full precedence and actual native fixture reads. Local-gates configuration needs operator verification. In the meantime TL explicitly directed pair:1 to work #844 on its own branch from main; no stacking.

### 2026-10-25 — implemented (pair:6)
- 2026-10-25: closed — pre-close suite on pushed head a7acc677 exit 0 in 12m10s (recorded sdlc-local/suite); make test-affected-inventory clean; make test-affected-runner test-affected-config pass; new TestCertificatesAndCouchTestsNarrow, TestCertificateReads, test_record_keeps_declared_reads pass and fail under mutation (main affected.json; record dropping members) (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

TL approved the plan (delivery df08c2a2) and then option A for the pin (delivery 280602da).
- **TL's question:** main's suite is green today. Its checkout has no sibling ariadne, so `tests/affected-config-test.py`
  qualified against the `go.mod` pin ariadne@5a17f789619d, which predates #598. The suite logs for bf9f1501 and
  9ff56625 both print that source. It is red only where a #598 sibling exists (this slot: compile failure on the moved
  `jsonstrict` and `Plan`'s new arguments) or after a pin bump.
- **Pin bump spread** (`go get github.com/xianxu/ariadne@9678ee653`): only ariadne moved. No source fixes were needed:
  `go build ./...` passes, and so do editorial, repolint (golint catalog) and the couchcore/couchcmd layergraph and
  sidedoc tests. ariadne#622 had not landed, so its rule needs no exclusion.
- **Harness:** gated on `SidecarFile`, copies `cmd/internal/jsonstrict`, and compiles the real certificates once per test
  binary (`sync.OnceValues`; the coverage tests plan 788 inputs). It qualifies against the sibling and against the
  bumped module pin.
- **New cases** `TestCertificatesAndCouchTestsNarrow` and `TestCertificateReads` pass. With main's affected.json they
  fail: certificates full, `cmd/couch/main_test.go` full, and the sidecar under no convention.
  `test_record_keeps_declared_reads` fails when `--record` drops existing members.
- Follow-up filed unclaimed: pair#893.
- **Local-gates:** `.sdlc/affected.json` changed, so the operator verifies (via the TL); I don't apply the label.

## Revisions

- 2026-10-24: planning found a missing provider capability. At ops:0 direction, block on ariadne#598 and defer the consumer design until its contract lands; #844 proceeds independently meanwhile. Original Done-when remains in force, plus the TL's cmd/couch test-file scope.
- 2026-10-25: TL chose option A. The `go.mod` ariadne pin bump to a #598 provider joins #853, so the suite's
  qualification runs instead of skipping. Stop rule: if the spread exceeded a few mechanical fixes, split it out.
  Measured spread: none.
