# Boundary Review — pair#458 (milestone M1)

| field | value |
|-------|-------|
| issue | 458 — broadcast-web: recorder — CDP DOM stream per frame plus isolated-world observer |
| repo | pair |
| issue file | workshop/issues/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 26292514a113e10cb14354487c2ceb03fe688649..a005266576968cee74282ce83f5d768f166bd78d |
| command | sdlc milestone-close --issue 458 --milestone M1 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T01:05:32-07:00 |
| verdict | SHIP |

## Review

The tests pass. I'm checking how the new kinds reach downstream consumers (`query/surface.go`), then I'll write up the verdict.

```verdict
verdict: SHIP
confidence: high
```

M1 delivers everything the plan promised for this milestone, and nothing beyond it:
- **`hover` and `ptr` record kinds:** new types in `record.go` with `check()`, both written to events.jsonl.
- **Reducer tracking:** `State.Hover` and `State.Ptr` are tracked, reset by `seg`, and `Hover` is cleared in `forget` when its node is removed. Neither appears in `Canonical`.
- **`input.secret` and the password rules:** enforced in three places — `attr`, `input`, and `checkNew`, which covers `ins`, `doc` and `kids`. `secret` is folded into Canonical's `$input` but omitted when unset, so no fixture pin moves.
- **`ValidateDir` stray-file check:** it now reports anything in `frames/` that isn't the index or a frame's screenshot.
- **Docs:** the atlas has the new rows, the password rules and a Revisions entry.

`go test ./cmd/internal/webcast/...` passes, and every existing fixture pin is unchanged. Nothing blocks shipping; the three findings below are all Minor.

**1. Strengths**
- **Password rule checks the record's result, not just the record.** `reduce.go:306` builds the attributes the element would have after an `attr` (`withAttr`), so both routes are caught: setting `value` on a password input, and flipping `type` to password on an input that already has a `value`. The test at `format_v1_additions_test.go:83` pins the second route.
- **Password check covers every way a node arrives.** `isPassword` uses `EqualFold` and trims spaces, and the test uses `type=Password` to prove it. The `checkNew` hook means `ins`, `kids` and a whole `doc` snapshot are all refused, and the tests cover all three.
- **Canonical invariance is tested directly.** `TestHoverAndPtr` compares Canonical byte-for-byte before and after adding hover and ptr records, rather than trusting the claim.
- **Old tests repointed correctly.** Tests that used `hover` as their "unknown kind" example now use `wiggle`, so they still test unknown kinds now that `hover` is a real one.
- **Stray-file test covers both cases.** `TestValidateDirReportsStrayFrameFiles` checks a `.tmp-*` left by a crash and an orphan `7.png` that no frame record names.

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- **Extra allocation on every `attr` record** (`reduce.go:306`, ARCH-CONSTRAINTS). `withAttr` copies the element's whole attribute map on every `attr` record, for any element, even though the rule only applies to `input`. `attr` is the most frequent mutation on the stress page. Fix: guard with `n.N == "input" &&` before building the copy.
- **`query.UserAction` doesn't list `Hover`** (`query/surface.go:45`). Leaving `Ptr` out is right, since it is sampled noise. Hover may be intended as context for notes, but nothing in the plan says either way. Decide and record it when #461 consumes it.
- **Test ignores write errors** (`format_v1_additions_test.go:107-113`). `w.Put`, `w.PutFrame`, `w.Close` and `os.WriteFile` errors are discarded, so a failed write would surface as a confusing assertion failure.

**5. Test coverage notes**
- The reducer is the validator, so every refusal rule is exercised through the validator path by `mustFail`.
- Not covered:
  - a hovered node inside a removed subtree, or inside an inline frame `doc` replaced in place (`forget` recurses, so it should work);
  - `secret` on a non-password element, which is accepted. This is harmless, but the format page says "a password control".

**6. Architectural notes for upcoming work**
- In M2, the mirror's "apply to own State before emitting" step is what turns these reducer rules into a recorder guarantee. Keep the plan's property test that a planted password never appears in the output.
- When `by` / `recording.Source` lands from #466, `Hover` and `Ptr` gain it by embedding, with no change to their checks.

ARCH markers:
- **ARCH-DRY: pass.** The three password checks share `isPassword`.
- **ARCH-PURE: pass.** All the logic is in the pure reducer, and the only IO is `ValidateDir`'s `ReadDir`.
- **ARCH-PURPOSE: pass.** Every M1 item is delivered.
- **ARCH-MOCK: N/A.** This milestone adds no external dependency.
- **ARCH-CONSTRAINTS: pass,** apart from the Minor allocation above.
- **ARCH-SECURE: pass.** The validator refuses password values whichever record carries them. Values typed while the control was text are recorded, as the plan accepts (PQ-7).
- **ARCH-ORDER: pass.** Hover and ptr state changes only go through `applyInSegment`, `resetDOM` and `forget`.
- **ARCH-FUNERAL: pass.** Nothing durable is added. `ValidateDir` now reports stray frame files, but it does not remove them.

**7. Plan revision recommendations:** none. The code matches the plan's M1 section. Once the gate closes, tick the M1 checkbox (plan line 162).

```findings
findings:
  - id: new
    severity: Minor
    family: hot-path-allocation
    title: |
      withAttr copies the attribute map on every attr record, even for elements that aren't inputs (reduce.go:306)
    detail: |
      Put an n.N == "input" guard before building the copy; attr is the most frequent mutation kind when replaying.
  - id: new
    severity: Minor
    family: consumer-kind-classification
    title: |
      query.UserAction leaves out the new Hover kind with no recorded decision
    detail: |
      Leaving Ptr out is correct. Decide whether hover belongs in a note's action context, and write the decision down when #461 consumes it.
  - id: new
    severity: Minor
    family: test-ignores-setup-errors
    title: |
      TestValidateDirReportsStrayFrameFiles discards the errors from Put, PutFrame, Close and WriteFile
```
