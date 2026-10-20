# Boundary Review — pair#699 (whole-issue close)

| field | value |
|-------|-------|
| issue | 699 — CI: build weave from a pinned ariadne commit |
| repo | pair |
| issue file | workshop/issues/000699-ci-weave-pin.md |
| boundary | whole-issue close |
| milestone | — |
| window | ca645a018ccffcd03abc003ef8a07ccd7bf01b9e..ec35b5d820a6cb463bea6c1690fef44af5ee81ed |
| command | sdlc close --issue 699 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T22:11:00-07:00 |
| verdict | SHIP |

## Review

Now compare against ariadne's template.

```verdict
verdict: SHIP
confidence: high
```

The change set is small and does what the Spec asks. Both `.github/workflows/merge-check.yml` and `.github/workflows/local-gates-guard.yml` are byte-identical to ariadne's seeded templates at `0956e5e1`. I checked this with `diff` against `git show 0956e5e1:<path>`. That ariadne commit contains the approved compiler `63d6bfd8`. The authored `.sdlc/weave-compiler-ref` is 41 bytes: a lowercase full SHA plus a newline. It passes the workflow's own `[0-9a-f]{40}\n` validator, and it matches ariadne's sole supported ref in `.sdlc/weave-compiler-refs`. The pin does not appear in `construct/base.manifest`, and `.gitignore` does not ignore it. Nothing blocks SHIP. Two Done-when items are acceptance steps that come after close: the real CI run on a cold checkout, and reporting the result to the hub. The plan states this openly, and the close cannot evidence them.

1. **Strengths**
   - The workflow is not hand-written. It comes from `weave refresh --substrate` and matches the upstream template exactly, so ARCH-DRY holds and there is one source of truth.
   - When a consumer runs it (`source_ci=false`), there is no way back to an older weave. The pin is validated strictly. The fetched HEAD identity is checked, and so are `vcs.revision` and `vcs.modified=false` in the binary. A failure anywhere aborts under `set -euo pipefail`, and `bin-pin1` is prepended to PATH only after a successful build (`merge-check.yml:98-104`).
   - `bootstrap.sh` is unchanged and identical to upstream. The pinned `63d6bfd8` provides `weave dependencies` (`cmd/weave/dependencies.go`), so the probe on bootstrap's line 6 picks the pinned binary. It cannot fall through to `brew install`.
   - Hardening came in with the template: `permissions: contents: read`, `timeout-minutes: 45`, `GOTOOLCHAIN=local`, `-trimpath`, and Go installed from the pinned source's own `go.mod`.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - Real CI acceptance, the cold-checkout regeneration, and the hub report are all deferred past close. The plan acknowledges this. Make sure the operator records the compiler-identity line from the build step.
   - The "pin survives compile" claim rests only on the Log entry. `ownership.json` is generated and not committed, so this review could only check the manifest side.

5. **Test coverage:** Tests for malformed pins, fetch or build failure, and a stale weave earlier on PATH live upstream (`scripts/test/portable-ci.test.sh`, `weave-compatibility.test.sh`). That is appropriate because the workflow bytes are the upstream template. In this repo, the acceptance evidence is the PR's own merge-check run.

6. **Architecture**
   - **ARCH-DRY:** passes. The files are the template, not copies of it.
   - **ARCH-PURE:** N/A. The change is CI config only.
   - **ARCH-PURPOSE:** passes. The pin is authored and enforced, and the old unpinned formula path is gone.
   - **ARCH-MOCK:** N/A here. Upstream owns the tests that run against fakes.
   - **ARCH-CONSTRAINTS:** passes. There is a 45-minute timeout and at most two compiler slots.
   - **ARCH-SECURE:** passes. The pin is parsed into a strict shape before use, the fetched identity is verified, and permissions are read-only.
   - **ARCH-ORDER:** N/A. The workflow holds no state between events; each run starts from a fresh runner.
   - **ARCH-FUNERAL:** passes. The only artifacts go under `RUNNER_TEMP` and disappear with the runner. The pin file is replaced whenever it is bumped.
   - **ARCH-EVOLVE:** passes. It adds one new authored file and changes no existing format.
   - For future pin bumps: each one needs the same `local-gates-verified` operator check, and the guard now covers `scripts/ci/*`.

7. **Plan revisions:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: post-close-acceptance-evidence
    title: |
      Real CI acceptance and hub report are Done-when items deferred past close
    detail: |
      Cold-checkout regeneration and the compiler identity line are evidenced only by the PR merge-check run after close; ensure the operator records them on the hub.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"61190c98-9514-4183-bc9a-00e3cae05756","repository":"github.com/xianxu/pair","issue_id":"000699","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/92c5ed42e268e53a","reviewed_head":"ec35b5d820a6cb463bea6c1690fef44af5ee81ed","slot":"pair:1","requested_at":"2026-10-20T05:09:50.238624Z"},"phase":"finished","pid":45060,"started_at":"2026-10-20T05:09:50.358389Z","finished_at":"2026-10-20T05:10:59.958507Z","session_id":"2ba51ff1-4692-455e-a66b-567238c2342d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.79245075s
head: "ec35b5d820a6cb463bea6c1690fef44af5ee81ed"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
