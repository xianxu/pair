# Boundary Review — pair#490 (whole-issue close)

| field | value |
|-------|-------|
| issue | 490 — couch: say each verb's relative cost in --help and --skill (light, seconds, heavy) |
| repo | pair |
| issue file | workshop/issues/000490-couch-say-each-verb-s-relative-cost-in-help-and-skill-light-seconds-heavy.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9ca0ae3cfbdddf33b8a4818bb95dee6428fe15e2..ad8c8828b2fc4d6f97afa522633f86f1684bbf76 |
| command | sdlc close --issue 490 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T11:20:33-07:00 |
| verdict | SHIP |

## Review

Still inspecting: the couchcmd tests need generated runtime assets, so I'm finding the generator to run them.

```verdict
verdict: SHIP
confidence: high
```

This change does what pair#490 asks. One table, `verbCosts` in `cmd/internal/couchcmd/cost.go`, holds every verb's cost. `couch --help` prints each verb line through `verbLine`, which panics if a verb has no recorded cost. `TestHelpAndSkillCarryEveryVerbCost` checks both ways that the help text and the skill's "What each verb costs" bullets match the table. I checked that this test catches real drift: removing `--reap` from the skill bullets made it fail (`couch --skill: --reap is "", want "seconds"`), and it passed again after the revert. All couchcmd and artifactpath tests pass at head. They first need the generated runtime assets, which come from `go run ./cmd/internal/runtimebundle/generatecmd`, per `Makefile.local:248`. I compared every `"--…"` literal in couchcmd with the table. The only flags not tagged are option flags and the hidden `--internal`, so no public verb is missing. Nothing blocks shipping. What's left is three small points: where the code departs from the Spec's wording, which tags were reasoned rather than measured, and one guidance sentence in the README that no test checks.

1. **Strengths**
   - `cost.go:61-67`: the panic means a new verb can't reach `--help` without a cost. The test calls `usage()`, so the panic shows up in CI and never only at runtime.
   - `cost_test.go:68-80`: `assertCosts` checks in both directions. A verb in the table but missing from a surface fails, and a tag on a surface with no table entry also fails.
   - The flag constants (`query.Flag`, `replayFlag`, `fidelityFlag`, `launchVerb`) serve as table keys, so a renamed flag can't leave a stale entry behind.
   - The Log records honestly which verbs were timed and which were tagged by reasoning, with the load at the time of each run.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **Spec departure (ARCH-PURPOSE, Minor):** the Spec put `--prepare` and `--reload-context` in the "seconds" tier and defined "light" as "milliseconds, no process starts". The code tags those two verbs heavy and defines light as "under a second". The Log gives reasons: both share the stop-and-boot path, and `--peek` and `--archived` take up to 0.7 s. But the Spec text was never revised to match.
   - **Done-when "backed by measured timings" is only partly met:** several tags were inferred rather than timed: `--reconcile`, `--reap`, `--reload-context`, `--prepare`, `--reboot`, `--recover`, `--send-to` and the bare `couch` launch. The inferences are reasonable and written down, but the Done-when overstates what was measured.
   - **README guidance sentence is not checked (ARCH-DRY, Minor):** `README.md:447-451` rewords `costGuidance` by hand, and no test ties it to the source. Only the help text and the skill are checked. It's prose only, so it's a low risk.

5. **Test coverage**
   - The help/skill sync test catches the kind of bug this change could ship: a missing tag, a wrong tier, a duplicate in the skill, or a missing guidance sentence.
   - `TestVerbLineRefusesAnUntaggedVerb` pins the panic.
   - Nothing checks that every verb the program dispatches appears in `--help`. That was outside this issue's scope.

6. **Architecture**
   - ARCH-DRY: pass. One table drives two checked surfaces. The README restatement is the Minor finding above.
   - ARCH-PURE: pass. `verbLine` and the table are pure, and the tests write to a buffer.
   - ARCH-PURPOSE: pass. Both consumers named in the Done-when (help and skill) are derived from or checked against the table.
   - ARCH-MOCK: does not apply; no external dependency was added.
   - ARCH-CONSTRAINTS: pass. The change exists to expose operating costs, and those costs were measured with the load noted.
   - ARCH-SECURE: does not apply. The code reads only static strings and the embedded skill, and touches no secrets.
   - ARCH-ORDER: does not apply. Nothing keeps state between events; it's a fixed table rendered once.
   - ARCH-FUNERAL: does not apply. Nothing durable is created; the change only adds help text.

7. **Plan revisions**
   - Add a `## Revisions` entry: "Spec tiers revised from measurement: light = under 1 s; `--prepare` and `--reload-context` moved from seconds to heavy (shared stop-and-boot path; not timed because the slot was parked for #489)."

```findings
findings:
  - id: new
    severity: Minor
    family: spec-not-revised-after-design-change
    title: |
      Spec still puts --prepare and --reload-context in "seconds" and defines light as milliseconds; the code says heavy and under a second
    detail: |
      The Log explains the change, but the Spec text was never updated. Add a Revisions entry so the Spec matches cost.go.
  - id: new
    severity: Minor
    family: done-when-overclaims-evidence
    title: |
      Done-when says the tags are backed by measured timings, but eight verbs were tagged by reasoning, not timed
    detail: |
      --reconcile, --reap, --reload-context, --prepare, --reboot, --recover, --send-to and the bare launch were not timed. The Log says so; the Done-when wording should too.
  - id: new
    severity: Minor
    family: unenforced-restatement-of-source
    title: |
      README rewords costGuidance by hand and no test checks it (ARCH-DRY)
    detail: |
      README.md:447-451 restates the tier meanings and the run-heavy-verbs-one-at-a-time advice. Only --help and the skill are checked against cost.go.
```

---

## Re-review — 2026-10-12T11:22:22-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 490 — couch: say each verb's relative cost in --help and --skill (light, seconds, heavy) |
| repo | pair |
| issue file | workshop/issues/000490-couch-say-each-verb-s-relative-cost-in-help-and-skill-light-seconds-heavy.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4d522ebb94c71b68a35812613ce299a53ce90fac..42a0a322c32fc8685371680be048dfe165051f4e |
| command | sdlc close --issue 490 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T11:22:22-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three prior Minor findings are fixed, and the fixes add no new defect. The `## Revisions` entry only records corrections and does not change scope. The README now carries `costGuidance` word for word, and `TestHelpAndSkillCarryEveryVerbCost` enforces it.

I could not run the Go test in this checkout: package setup fails because the generated `assets/runtime/files` don't exist here. So I checked the regression by hand, flattening whitespace the same way the test's `flat` helper does. The guidance sentence matches once in the new README and zero times in the base README. Without the fix, the new assertion would fail.

1. **Strengths**
   - `cmd/internal/couchcmd/cost_test.go:100-106`: the README now derives from the same `costGuidance` constant that `--help` and the skill are checked against. That closes the class the finding named (cost wording stated outside `cost.go`), not just the one site.
   - The `## Revisions` entry lists every tier that moved from the Spec's guesses, plus the untimed verbs, and names `cost.go` as the source.
   - `workshop/lessons.md` turns BR-1 and BR-2 into a general rule: when a measurement disagrees with the Spec, revise the Spec at once, and say in Done-when which values were reasoned rather than measured.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** none new. The test reads `../../../README.md` by relative path. That is fine for `go test` run from the package directory, but the test is tied to the repo layout.

5. **Test coverage:** The README check sits inside the existing sync test, so all three places that state the guidance (help, skill, README) are now checked. The README tier explanation that used to be hand-written was replaced by the constant's text, so nothing in the README restates the source unchecked.

6. **Architecture (interdiff only)**
   - **ARCH-DRY:** pass. The hand-written restatement is gone and the README is checked against `costGuidance`.
   - **ARCH-PURE:** pass. The change is test-only.
   - **ARCH-PURPOSE:** pass. All three consumers of the guidance now derive from the source.

7. **Plan revisions:** none. The Revisions entry now matches `cost.go` (I checked the tiers in `verbCosts`).

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Revisions entry records light as "under a second", and --prepare/--reload-context as heavy; matches cost.go verbCosts.
  - id: BR-2
    disposition: addressed
    note: |
      Revisions entry amends Done-when, naming the eight untimed verbs as tagged from a shared timed path.
  - id: BR-3
    disposition: addressed
    note: |
      README now carries costGuidance verbatim and cost_test.go:100-106 asserts it. Base README fails the flattened containment (checked by hand; go test cannot build here without generated assets).
```
