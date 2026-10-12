# Boundary Review — pair#453 (whole-issue close)

| field | value |
|-------|-------|
| issue | 453 — broadcast-web: decide the recorder base by prototype (rrweb vs our own on CDP) |
| repo | pair |
| issue file | workshop/issues/000453-broadcast-web-decide-the-recorder-base-by-prototype-rrweb-vs-our-own-on-cdp.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e941c926f0c345c15fdfbf4b28963967a77385b..5ff1c0aacba39918db41514ac4c804307d081530 |
| command | sdlc close --issue 453 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T20:41:38-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Only the issue file changed. Read it.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The only file this window changes is the issue file itself (`git diff --stat` and `--name-status` with the issue and history paths excluded both come back empty). So the review checks the write-up against its own Spec and Done-when, and checks its numbers against each other. The write-up is strong. Both prototypes were run in the same session against the same screenshots. All five comparison points have numbers. The recommendation (B, own-on-CDP, plus an isolated-world observer) gives its reasons and leaves the decision to the operator. No prototype code was merged. Most of the arithmetic checks out: overhead +0.51s/33.6s ≈ 1.5% and +0.34s ≈ 1.0%; A's build estimate of 1.5+1+2–3+1.5 = 6–7d.

What keeps this from a plain SHIP is one Done-when clause: the comparison must also be "summarized in the project's Log". That project file lives in the ops repo, and nothing in this window shows it was written. A few summary claims are also stronger than the tables behind them.

**1. Strengths**
- The method is fair. Both recorders were scored in one session against the same screenshots, with a ±window for paint lag. Overhead was measured in separate runs, two each.
- The fidelity list goes item by item and records the mechanism for each (pure CDP, isolated world, or hook), as the Spec asked: "record which items needed it".
- It reports what went against B, not only what favours it: 10k-character text truncation, user-agent shadow trees, larger raw volume, the replayer edge cases rrweb already solves.
- Volume is compared after gzip as well as raw, with the reason B's raw stream is inflated (53% of bytes are `setChildNodes`).
- Product scope was kept: the window contains no code.

**2. Critical findings**
None.

**3. Important findings**
- **Project-Log summary not shown in this window.** Done-when requires the comparison to be "summarized in the project's Log" (ops `teams/devinfra/projects/couch-broadcast-web/project.md`). This window only touches pair. The fix: write the summary in the ops project Log, or confirm it exists, and record a pointer to it in this issue's Log before closing.

**4. Minor findings**
- **Some summary claims are stronger than the data.** Every case in this window:
  - "~1.5× the gzipped bytes" (Volume bullet and Recommendation). The per-page ratios are 1.48× (stress), 1.15× (own app), 2.05× (Wikipedia) and 1.63× (GitHub). Say "1.1–2×, ~1.5× on the busy page".
  - "At least as faithful on every page". On Hacker News, B lagged on three moments against A's one. B's max was lower (8.80 vs 9.88) and both medians were 0, so this needs a qualifier.
  - "Lowest page overhead" was measured only on the stress page. Say so.
  - B's build estimate of "~7–8 days" sums to 7–7.5d from its own parts (3 + 2.5–3 + 1.5).
- **B's gap list is missing a known limit.** The fidelity table says B's form-value observer does not work inside closed shadow roots, but section 3 doesn't list it.
- **Prototype code can't be recovered.** The Spec allowed either a `prototypes/` directory or a scratch branch. The work lived in a "session scratchpad", which is neither, so the numbers can't be re-run. Pushing an unmerged scratch branch and naming it in the Log would fix this cheaply.
- **Volume table drops Hacker News.** The fidelity table has five pages and the volume table has four. That is acceptable, since the Spec only asks for volume on the busy page.

**5. Test coverage notes**
There is no executable change, so there are no tests to judge. The evidence is the measurements in the Log, and they can't be re-run (see the scratchpad finding above).

**6. Architectural notes for upcoming work**
- **ARCH-DRY: pass.** There is no code, so no duplication.
- **ARCH-PURE: pass.** There is no code. For the MVP: the planned "compact operations (insert/remove/attr/text/value/scroll/sheet)" applied by both the viewer's replayer and the agent CLI is a good fit for a pure core. Keep the normalizer and the applier pure, with the CDP client as a thin injected shell.
- **ARCH-PURPOSE: pass.** The purpose was a decision made from evidence. All five comparison points are measured, apart from the project-Log clause flagged above.

**7. Plan revision recommendations**
None needed. The plan checkboxes match what was delivered. If the scratch branch is pushed, add a one-line Revisions or Log note naming it.

```findings
findings:
  - id: new
    severity: Important
    family: done-when-clause-unevidenced
    title: |
      Done-when requires a summary in the ops project Log; nothing in the window shows it
    detail: |
      Done-when says the comparison is summarized in teams/devinfra/projects/couch-broadcast-web/project.md (ops repo). The window only touches the pair issue file. Write or confirm the project-Log summary and put a pointer to it in this issue's Log before closing.
  - id: new
    severity: Minor
    family: summary-claim-exceeds-data
    title: |
      Summary claims are stronger than the tables behind them
    detail: |
      All instances: "~1.5x gz bytes" (the ratios are 1.15x/1.48x/1.63x/2.05x, and Wikipedia is 2x); "at least as faithful on every page" (on HN, B lagged on 3 moments vs A's 1); "lowest page overhead" (measured only on the stress page); B's estimate "7-8d" while its parts sum to 7-7.5d.
  - id: new
    severity: Minor
    family: gap-list-incomplete
    title: |
      B's coverage-gap list leaves out form values inside closed shadow roots
    detail: |
      The fidelity table says B's observer covers form values "not inside closed roots", but section 3's B gaps do not list it.
  - id: new
    severity: Minor
    family: evidence-not-reproducible
    title: |
      The prototypes lived in a session scratchpad, not the prototypes/ dir or scratch branch the Spec allowed
    detail: |
      The measurements can't be re-run. Push an unmerged scratch branch and name it in the Log.
```
