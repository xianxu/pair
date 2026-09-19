# Boundary Review — pair#289 (whole-issue close)

| field | value |
|-------|-------|
| issue | 289 — Retire hostty's test-only control surface |
| repo | pair |
| issue file | workshop/issues/000289-retire-hostty-s-test-only-control-surface.md |
| boundary | whole-issue close |
| milestone | — |
| window | 69c120bae7b656cb7bae5825da41241ed07746c0..07355017baf511bb5b093f464ae4aa291bf4dc7f |
| command | sdlc close --issue 289 |
| reviewer | claude |
| timestamp | 2026-09-18T18:16:14-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

pair#289 does what its Spec and every `## Done when` clause ask, and I checked each clause myself rather than trusting the Log. I ran the new guard against the base tree in a scratch copy. It failed on exactly 9 symbols: the 8 dead `control.go` symbols plus `Reservation.Paint`, with `EdgeTop` allowlisted. At head it passes. I also mutated `console.go` in a scratch copy so it re-selects the current actor in both newest-page branches. Both tests then failed at the new selection check ("selection 2 -> 3"), and `landingOf` stayed silent. At base, the old `HomeAndClear` oracle could never fail: it had no production writer. The #152 contract, the core-concepts row, the guard, `go vet` and the pure hostty tests all pass. The only hostty failures are the two OS-pty tests, which the sandbox blocks. The newest-page tests passed 30 runs in a row under `-race`. Nothing blocks shipping; the four findings below are all Minor.

**1. Strengths**
- **Wire bytes unchanged.** `cmd/internal/hostty/reserve.go:149` composes the same sequence the old `ReserveAndPaint` did (save, region, move, reset, clear, text, reset, restore), so the probe's output doesn't change.
- **Guard extension is well designed.** Each scope gets its own allowlist (`deadsymbols_test.go:37`), and the iota-zero rule is narrow. With that rule disabled, exactly the five `ops.go` `*Unknown` values show up, as the comment says.
- **Better test oracle.** `selectionOf` (`console_newest_page_test.go:239`) reads presenter state, not bytes. `Presenter.View()` is locked by a mutex. `Token` only increases on Select, Panel and Resize, not on `UpdateChrome`, so a notice repaint can't make the test flaky.
- **Path-shaped references swept.** `NonArtifactSources` and the #152 plan contract both named the deleted file. The `issue151M3GoSources` entry that still names `control.go` is correctly left alone, because it reads a pinned historical commit.
- **Package doc is accurate.** In `host.go:1-14`, "spells no escape sequence production writes" holds: the only `\x1b` literals left in non-test hostty files are in `reserve.go`.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- **Count and date claims that don't match the tree:**
  - `deadsymbols_test.go:23` and `workshop/lessons.md:10` say "for a week". #255 M3 (`f32bb4cf`) landed 2026-09-15, which is three days, and the issue's own Problem section says "three days".
  - The #289 Plan row at line 86 says "ten hostty orphans". The measured count is nine (8 + `Paint`), which the Log gets right.
  - #281's table (line 22) says "`setRegion` and five more". `reserve.go` holds seven sequences, so that should be "six more".
- **Open issues still describe deleted symbols as live:**
  - `workshop/issues/000217-*.md:36` says "`hostty.ResetInteractiveModes` already includes `1004` in teardown". That was false after #255 M3, and the symbol is now gone.
  - `000241-*.md:42` suggests generalising `PrivateModes` inside `hostty`, which contradicts hostty's new package doc.
  - `000207`'s mentions are measured history and are fine.
  - This is the same kind of misleading prose #289 is about (ARCH-PURPOSE shadow-sweep).
- **Guard blind spot for fake files:** fake files are skipped when collecting declarations but still counted as references. A production symbol used only by fakes therefore reads as live. Excluding fakes from the reference count as well turns up couchcore `strings.go:joinArgs`, which only `git_fake.go` and `runner_fake.go` use. Hostty has no such case today. The gap predates this change, but this change widened the fake-skip rule.
- **The guard's new rules have no test of their own:** nothing pins `isIotaZero` or the const/var collection. A regression there would silently hide orphans. The red-before-fix run was manual; I replayed it.

**5. Test coverage notes**
- The rows=0/1 degenerate `ReserveAndPaint` case is still covered by `TestDegenerateHeightsNeverProduceAZeroRowChild`, so dropping the two `Paint` tests loses nothing.
- The bracket check moved into `TestSaveComesBeforeTheRegionChangeThatHomesTheCursor` (region < text < restore).
- If #281 deletes the painters but leaves their sequences, the hostty guard fails. I traced this: the seven sequences' only production references are the painters' bodies.

**6. Architecture**
- **ARCH-DRY: pass.** The presenter and `reserve.go` spell some of the same sequences separately. The Spec decided this on purpose: the presenter's bytes differ, and reserve.go is probe-only code pending #281. Test oracles spelling literals independently is right for an oracle. The #152 retired-concept map is small and follows the per-issue contract style; there's no shared helper it should have used.
- **ARCH-PURE: pass.** `reserve.go` is still pure string building. The guard is a filesystem meta-test by nature. The new oracle reads state rather than intercepting IO.
- **ARCH-PURPOSE: pass, with the Minor above.** Every code consumer was swept: contract rows, `NonArtifactSources`, the #152 plan contract, the atlas and #281's table. What remains is the prose in open issues 217 and 241.

**7. Plan revision recommendations**
- #289: add a `## Revisions` line correcting "ten hostty orphans" to nine (8 `control.go` symbols + `Reservation.Paint`).

```findings
findings:
  - id: new
    severity: Minor
    family: prose-count-claim-unmeasured
    title: |
      Count/date claims in new prose disagree with the tree (a week vs three days; ten vs nine; five more vs six)
    detail: |
      All instances in window: deadsymbols_test.go:23 and workshop/lessons.md:10 say "for a week", but #255 M3 (f32bb4cf) is 2026-09-15 and the issue Problem itself says three days. Issue 000289 Plan line 86 says "ten hostty orphans"; replaying the guard on base flags nine. Issue 000281 line 22 says "setRegion and five more in reserve.go"; reserve.go holds seven sequences, so six more.
  - id: new
    severity: Minor
    family: stale-prose-names-retired-surface
    title: |
      Open issues 217 and 241 still cite deleted hostty symbols as live mechanism
    detail: |
      000217:36 claims hostty.ResetInteractiveModes includes 1004 in teardown, which was false after 255 M3 and the symbol is now deleted. 000241:42 directs generalising PrivateModes in hostty, which contradicts the new package doc. 000207 mentions are measured history and fine. A one-line note per open issue closes the class.
  - id: new
    severity: Minor
    family: dead-symbol-guard-fake-asymmetry
    title: |
      Dead-symbol guard skips fake files as declarations but counts them as production references
    detail: |
      productionIdentifierCounts still counts fake.go and *_fake.go, so a production symbol reached only by fakes reads as live. Excluding fakes on the reference side surfaces couchcore strings.go joinArgs, used only by git_fake.go and runner_fake.go. Pre-existing, but this window widened the fake-skip rule; hostty has no hidden orphan today.
  - id: new
    severity: Minor
    family: guard-rules-unpinned
    title: |
      The guard's new const/var, fake.go and iota-zero rules have no fixture test of their own
    detail: |
      isIotaZero is a pure function and productionDeclarations could run over a testdata dir. Without that, a regression that widens the iota or fake skip silently hides orphans. Red-before-fix was shown manually only.
```
