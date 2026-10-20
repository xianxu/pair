# Boundary Review — pair#716 (whole-issue close)

| field | value |
|-------|-------|
| issue | 716 — couch --keys help: list the steps and the panel round trip (open /usage, close with esc) where agents look |
| repo | pair |
| issue file | workshop/issues/000716-couch-keys-help-list-the-steps-and-the-panel-round-trip-open-usage-close-with-esc-where-agents-look.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4b3105cb2d8d4b4a99ec29e8e7d91a3974de820a..5dddd69efbe9733a203d68128f3d55f7c8401b51 |
| command | sdlc close --issue 716 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-19T20:09:41-07:00 |
| verdict | SHIP |

## Review

Sandbox git-init failures are environmental. Running the targeted tests:

```verdict
verdict: SHIP
confidence: high
```

The range does what pair#716 asks for. `agentdrive.Steps` (`cmd/internal/agentdrive/keys.go:44`) is now the one table of key steps: the parser accepts a named key only if it is a row there, and the `couch --keys --help` output, the `couch --help` line and the skill's step sentence all come from it or are tested against it. `couch --keys --help` and `-h` print the steps, the limits (taken from the constants), the admission rules and the full open/peek/close round trip for a panel. The advisory carried over from #691 is also done: one rule, `Admission.Refusal`, is used by the reducer, the wrapper and the service, so a refusal while a panel shows names the lone `esc`. That includes the service's early refusal for an agent that isn't settled. It gets the hint through one read-only `ObserveDrive` on a path that refuses anyway. The changed-package tests pass where the sandbox allows them: `agentdrive` and `wrapcmd` fully, `couchcmd` filtered to `Keys|Skill|CouchHelp|Drive`. The unfiltered `couchcmd` run failed only in tests that call `git init` outside the reviewer's allowed directories (sdlc #339), which is the sandbox, not this change. Nothing blocks SHIP; I found only Minor issues.

1. **Strengths**
   - The parser's accepted set comes from the table by construction: `Validate`'s `default:` branch calls `isStep` (`keys.go:136-142`). A test confirms that every row parses and that the internal-only keys (`KeyLeft`, `KeyRight`) are refused (`panel_test.go:166`).
   - `TestSkillKeyStepsMatchTheTable` (`keys_help_test.go:45`) pulls the skill's step sentence out with a regex and compares it in both directions, so the hand-written skill prose is held to the table.
   - The refusal wording lives in one pure function, `Admission.Refusal` (`recipe.go:95`), used at all three layers. The `wrapcmd` and `couchcmd` panel tests now check the exact `PanelRefusal` text. The `text`/`submit` cases in `TestDrivePanelAdmitsOnlyEsc` go through the new early observation in the not-settled branch.
   - The extra observation is strictly read-only and sits on a path that already refuses. If it fails, the service falls back to the old wording, so the refusal outcome can't change.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `recipe.go:95`: `Refusal` ignores which request was refused. A lone `esc` refused for another reason (an incomplete screen at the wrapper, or an empty `Instance` at `agent_drive_service.go:109`) is still told "only a lone esc is admitted … then retry". That is harmless but slightly misleading. One fix is to pass in whether the request is `DismissOnly` and, if so, keep the generic wording.
   - `keys.go:52`: the `wait:MS` meaning hardcodes "0–1000 ms … 5 seconds", and the skill prose hardcodes "64 steps, 8 KiB". The help's Limits line uses the constants, but these strings could drift from them (ARCH-DRY, small).
   - `cli.go:31`: `cliKeysHelp` is inserted in the middle of an `iota` enum, which shifts the values after it. Those values are only used in memory, so nothing breaks.

5. **Test coverage**
   - The table and the parser are checked against each other in both directions, and so are the table and the skill.
   - The help output is checked for every row, both limits and all three recipe lines. `couch --help` is checked for the inline `StepSyntax()` and the pointer to `couch --keys --help`.
   - Refusals are checked at the reducer, the wrapper and the service, including a negative case: with no panel, there is no hint.
   - I did not re-run the implementer's mutation runs (their Log entry lists them as red). From reading the assertions, each test would fail with its fix removed.

6. **Architecture**
   - **ARCH-DRY:** pass, apart from the hardcoded-limit Minor above.
   - **ARCH-PURE:** pass. `Steps`, `StepSyntax` and `Refusal` are pure, and the IO stays in the service shell.
   - **ARCH-PURPOSE:** pass. Every consumer either derives from the table (parser, help, `couch --help`) or is held to it by a test (skill). README and atlas describe the new surface without listing the steps again.
   - **ARCH-MOCK:** pass. The new observation goes through the existing `ObserveDrive` endpoint, which already has a fake.
   - **ARCH-CONSTRAINTS:** pass. It adds one bounded observation, only on a path that refuses.
   - **ARCH-SECURE:** pass. Decoded sequences with unknown kinds still fail `Validate`.
   - **ARCH-ORDER:** pass. There are no new states or transitions; only the refusal text changed.
   - **ARCH-FUNERAL:** pass. Nothing durable is created; the help is printed to stdout and the text is constant.
   - **ARCH-EVOLVE:** pass. No persisted format changes; `KeyStep` JSON is unchanged.

7. **Plan revisions:** none. The plan still matches the code.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: refusal-wording-ignores-request
    title: |
      Panel refusal hint tells a refused lone esc that only a lone esc is admitted
    detail: |
      Admission.Refusal (recipe.go:95) only looks at the admission. A DismissOnly request refused for an incomplete screen (wrapper) or an empty Instance (agent_drive_service.go:109) still gets "only a lone esc is admitted ... then retry". Consider keeping the generic wording when the request is already DismissOnly.
  - id: new
    severity: Minor
    kind: defect
    family: limit-prose-restates-constants
    title: |
      Step meaning and skill prose hardcode limits that the help renders from constants
    detail: |
      The wait:MS meaning in agentdrive.Steps says "0–1000 ms, at most 5 seconds" and SKILL.md says "64 steps, 8 KiB". If MaxKey* changes, neither is checked. Render them with fmt from the constants, or test them against the constants.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9e9f7ffa-eb36-47d8-9766-12df26aeeb92","repository":"github.com/xianxu/pair","issue_id":"000716","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/eccb048f8ef41b9d","reviewed_head":"5dddd69efbe9733a203d68128f3d55f7c8401b51","slot":"pair:2","requested_at":"2026-10-20T03:05:39.332059Z"},"phase":"finished","pid":42621,"started_at":"2026-10-20T03:05:39.714697Z","finished_at":"2026-10-20T03:09:40.780623Z","session_id":"15bf7af5-29c4-447c-b16d-cedb07d31717","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.657268583s
head: "5dddd69efbe9733a203d68128f3d55f7c8401b51"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
