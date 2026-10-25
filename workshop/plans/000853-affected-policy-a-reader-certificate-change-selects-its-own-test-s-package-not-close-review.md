# Boundary Review — pair#853 (whole-issue close)

| field | value |
|-------|-------|
| issue | 853 — Affected policy: a reader-certificate change selects its own test's package, not the full suite |
| repo | pair |
| issue file | workshop/issues/000853-affected-policy-a-reader-certificate-change-selects-its-own-test-s-package-not.md |
| boundary | whole-issue close |
| milestone | — |
| window | fd31b8116bba42b8405419a850e3cb19b2deff57..a7acc677ce2522130d9e7687f65a4bcad97ba265 |
| command | sdlc close --issue 853 |
| reviewer | claude (anthropic; author anthropic,openai) |
| isolation | fenced |
| timestamp | 2026-10-25T11:48:41-07:00 |
| verdict | SHIP |

## Review

Running the qualification tests against the pinned provider now.

```verdict
verdict: SHIP
confidence: high
```

This is ready to ship. The production change is two additive edits to `.sdlc/affected.json`: a `sidecars` declaration and two `full` entries that now carry `except` lists. Together they make a reader-certificate change select its test's package, and a `cmd/couch/*_test.go` edit select `cmd/couch` alone. Selection logic stays in the ariadne provider (ARCH-DRY), so the change matches the Spec.

I ran the qualification harness against the pinned provider `88846b44…` (head e19cdb92b, which contains 9678ee65384b). All tests pass, including the new `TestCertificatesAndCouchTestsNarrow` and `TestCertificateReads`. I also reverted only `.sdlc/affected.json` to the base version and re-ran: both new tests and `test_check_config` failed. So the tests catch a regression in the policy itself. `tests/affected-go-readers-test.py` (22 tests) and `tests/test-affected-test.py` (14 tests) also pass. I found two Minor issues and nothing blocking.

Done-when: current — both clauses are tested on the real planner. Clause 1 uses #825's real `request_lifetime_test.go.json` certificate (plus added, renamed and couch certificates), and the declared-reads part is covered by the fixture in `TestCertificateReads`. Clause 2 checks that `.sdlc/affected.json`, `scripts/test-affected.py`, all three selector/inventory tests, `tests/lib/**`, `cmd/couch/main.go` and a certificate changed together with the selector all stay full. The Revisions entry (the `go.mod` pin bump) only enables qualification and adds no behaviour that Done-when should cover.

1. **Strengths**
   - `.sdlc/affected.json:13-16,22,37`: the policy change is small and additive. `cmd/couch/testdata/**` stays full through `cmd/**/testdata/**`, and that is tested.
   - `tests/affected-config-test.py:124-158`: the harness compiles the **real** certificates through `affected.Compile`, the same way sdlc does, and caches the result with `sync.OnceValues`. The coverage tests (768 inputs) therefore exercise the real mapping.
   - `tests/affected-config-test.py:351-362`: the full-fallback edge cases are thorough. They cover a certificate with no package, non-`_test.go.json` names, a nested `cmd/couch/sub/nested_test.go` (`*` does not cross `/`), and a certificate changed together with the selector.
   - `tests/affected-go-readers-test.py:266-278`: there is a regression test that `--record` keeps `reads` while refreshing the hash (ARCH-EVOLVE pass).
   - The new provider gate (`SidecarFile` in `mapping.go`) skips older providers instead of failing on them. Its test fixture is updated to match.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - `tests/test-affected-test.py:29-35`: the new `full_matches` uses Python `fnmatchcase`, where `*` crosses `/`. The provider's glob does not, so the helper restates the provider's glob semantics with different rules. Today `except` is the only new instance, and a mismatch there makes the assertion stricter, so no false pass is possible yet. The base glob matching has the same mismatch and could falsely pass. This is the only instance in the window (ARCH-DRY).
   - `tests/affected-config-test.py:331-339`: `same()` only compares the certificate's selection with the test's selection, both computed by the same planner. The Plan names `…/testcouch/agentfixture` explicitly for #825, but the test never asserts that package appears. This is the only weak site; the couch cases use `exact` and do pin their package.

5. **Test coverage notes**
   - The revert check confirms the new Go tests fail without the policy change.
   - I did not run the module-pin path (no sibling, `GOPROXY=off`). Qualification was only checked through `ARIADNE_AFFECTED_SOURCE`.
   - None of the `cmd/couch` test files have build tags, and none of the groups' `go` lists include `cmd/couch`. So selecting the `cmd/couch` package covers every test in a couch-test-only edit.

6. **Architectural notes**
   - ARCH-DRY: **pass**. Selection stays in ariadne, and pair only declares. The one restatement is the Minor `full_matches` above.
   - ARCH-PURE: **pass**. The change is configuration and tests; filesystem reads are confined to the harness's `loadMapping`.
   - ARCH-PURPOSE: **pass**. Both consumers motivated by the issue (certificates, and couch test files per the TL's scope) now derive their selection from the declaration. pair#893 (filling in `reads` on the real certificates) can genuinely be done separately: `reads` cannot remove an input's own `full` rule, so populating them changes no selection today.
   - ARCH-EVOLVE: **pass**. The `affected.json` members are additive. A pre-#598 sdlc strictly refuses the object-form `full` entries instead of misreading them. The recorder keeps unknown members and `reads`, and both are tested.

7. **Plan revision recommendations:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: consumer-restates-provider-glob
    title: |
      full_matches in tests/test-affected-test.py restates provider glob semantics with fnmatch, where star crosses slash
    detail: |
      tests/test-affected-test.py:29-35 matches the glob and except entries with fnmatch.fnmatchcase, whose star crosses '/', but the provider's glob does not (the nested cmd/couch/sub/nested_test.go case relies on that). For except entries the mismatch only makes the assertion stricter; the base glob matching has the same mismatch and could falsely pass. This is the only instance in the window (ARCH-DRY). Either use a matcher where star stops at '/', or note that the helper is deliberately approximate.
  - id: new
    severity: Minor
    kind: defect
    family: relative-assertion-without-anchor
    title: |
      The certificate check for 825 compares two planner outputs but never asserts the agentfixture package is selected
    detail: |
      same() at tests/affected-config-test.py:331-339 checks certificate selection == test selection, non-full and non-empty, but never names the expected package. The Plan states the 825 certificate selects cmd/internal/testcouch/agentfixture and its dependents; adding a containment assertion would pin that. The couch cases already use exact(); this is the only weak site.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"591ccffe-2b47-4628-a88e-b03686be7513","repository":"github.com/xianxu/pair","issue_id":"000853","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/ffd8c5030b3024c0","reviewed_head":"a7acc677ce2522130d9e7687f65a4bcad97ba265","slot":"pair:6","requested_at":"2026-10-25T18:45:22.447805Z"},"phase":"finished","pid":91993,"started_at":"2026-10-25T18:45:22.54677Z","finished_at":"2026-10-25T18:48:38.184643Z","session_id":"8bf82acd-daab-4f2a-a930-3cef4a814637","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.376754167s
head: "a7acc677ce2522130d9e7687f65a4bcad97ba265"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
