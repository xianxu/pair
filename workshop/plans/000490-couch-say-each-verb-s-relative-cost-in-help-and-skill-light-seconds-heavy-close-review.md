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
