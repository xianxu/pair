# Boundary Review — pair#283 (whole-issue close)

| field | value |
|-------|-------|
| issue | 283 — A child's default cursor style overrides the terminal's configured cursor |
| repo | pair |
| issue file | workshop/issues/000283-default-cursor-style-overrides-terminal.md |
| boundary | whole-issue close |
| milestone | — |
| window | b4ef9bccfeccb2e6830bbc6b65ccf035436092c8..1949fdfd5c2fd75c2a764fa5b2360e6a5a4c3dbb |
| command | sdlc close --issue 283 |
| reviewer | claude |
| timestamp | 2026-09-18T09:20:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The diff does what the issue asks for, and I only found Minor issues. "Terminal default" is now the zero value at every layer. `CursorDefault` is 0 in vt. The endpoint publishes vt's style number directly (`endpoint.go:229`), so the `+1` that lost the default is gone. `cursorEpilogue` sends `ESC[0 q` (`render.go:25-33`), and `Frame.Validate` accepts only one form of the default (`frame.go:93`). Because the default is the zero value, RIS, a fresh screen and a fresh alternate screen start at the default without extra code. I ran vt, `cmd/internal/terminal` and `cmd/internal/terminalqualify` at `1949fdfd` and all passed; gofmt and vet are clean. I also reverted three parts of the fix in a scratch copy, one at a time:
- **Endpoint blink guard removed:** the endpoint cursor table and `TestChildCursorStyleReachesParentVerbatim` both fail.
- **Old epilogue restored (0 sent as a block):** `TestCursorEpilogueCarriesDECSCUSR` and the default cases of the acceptance test fail.
- **Handler gives the default a steady cursor:** the vt test fails. The endpoint hides this one, because it forces Blink off for the default anyway.

`couchtty` failed in this environment: `mkdir /tmp/...` and the pty-child start both returned "operation not permitted". That is the known sandbox limit on pty-child tests, not this diff. Nothing blocks SHIP. The only traceability gap is that the operator-smoke checkbox claims more than the Log reports.

1. **Strengths**
   - **Zero value as the default.** RIS and fresh screens reset through `Cursor{}` (`screen.go:41-42`), so they reach the default for free. The endpoint mapping is now the identity, so the site that lost the default no longer exists (ARCH-ORDER: 8 representable states for 7 legal ones instead of 12).
   - **One representation of the default.** Refusing a blinking default in `Frame.Validate` keeps the `prev.Cursor != next.Cursor` dirtiness checks exact in `render.go:48` and `history_render.go:108`.
   - **Real acceptance seam.** `presenter_test.go:1027-1046` runs the production endpoint and `Presenter.Select` into a `ttyio.Fake` parent, over codes 0–6, an absent parameter and RIS. It uses `presenterFixture`, whose `t.Cleanup` releases the presenter, which settles plan-gate PQ-2.
   - **Stronger vt test.** `TestPairCursorStyleRejectsInvalid` (`pair_edges_test.go:261`) now checks against an explicit underline, because a fresh value would equal the default.
   - **Honest Log.** It names the console-menu caret change (`console_menu.go:216` goes from `ESC[2 q` to `ESC[0 q`) and the true scope of the operator smoke.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - **Operator-smoke checkbox overclaims.** The issue ticks "plain shell under couch shows a bar", but the Log says the smoke covered couch's switcher caret, not a plain-shell pane. The pane path is covered by `TestChildCursorStyleReachesParentVerbatim`. Either run the short plain-shell check, or reword the checkbox and Done-when and add a `## Revisions` entry.
   - **Blink differs between vt and the frame.** vt reports the default with Steady=false, so the callback and the qualifier's raw observation (`"0,true"`) show blink=true. The frame forces Blink=false for the default. No pair code registers the callback, and both layers are documented (`screen_cases.go:16-17`, `frame.go:15-18`). A future callback consumer should apply the endpoint's rule, not trust `blink`.

5. **Test coverage notes**
   - Every seam has a table test: vt handler, endpoint (every byte split), `Validate`, epilogue, and endpoint → parent.
   - The case most likely to be mishandled, `ESC[0 q` over an explicit blinking block, is tested at both vt and endpoint layers. So is a callback firing on block → default.
   - The xterm oracle maps 0 to 1 and can't observe any of this. Ghostty's one-shot smoke is the only live conformance check.

6. **Architecture (ARCH-\*)**
   - **ARCH-DRY: pass.** DECSCUSR decoding (`handlers.go:853`) and encoding (`render.go:25-33`) are inverses in different layers, and the acceptance test pins the round trip.
   - **ARCH-PURE: pass.** The handler, the capture mapping and the epilogue are pure and tested without IO.
   - **ARCH-PURPOSE: pass.** I checked every consumer and producer independently:
     - Consumers of `Frame.Cursor`: `cursorEpilogue`, used by both `Render` and `HistoryRender.Emit`, and `Validate`.
     - Producers of `Frame.Cursor`: `Endpoint.capture` and `console_menu`'s `PanelFrame`.
     - vt cursor style: read by the endpoint and the qualifier candidate. No pair caller registers the callback.
     - Other vt importers (`scrollbackcmd`, `wrapcmd`, `couchnestedrows`) never read cursor style.
     - Frames are not serialized across processes, so there is no version-skew path.
   - **ARCH-MOCK: pass, with a note.** The parent is a stateful `ttyio.Fake`. Live conformance is the one-shot Ghostty smoke, with no scheduled check.
   - **ARCH-CONSTRAINTS: pass.** `ESC[0 q` is the same length as `ESC[n q`, and there is no new per-frame work.
   - **ARCH-SECURE: pass.** The removed `param > n` guard did not protect against negative values that can occur. ansi's `Param` masks every value to [0, 2³¹−1] (`parser_decode.go:504-509`), n>6 is still rejected, and `Validate` is a backstop.
   - **ARCH-ORDER: pass.** State is a tagged enum. The one state vt can represent but should never hold, `(CursorDefault, Steady=true)`, is never produced and is normalized at the endpoint anyway.
   - **ARCH-FUNERAL: pass.** Nothing durable is created, only an existing in-memory field changes value.
   - **For upcoming work:** vt keeps one cursor style per screen, while xterm keeps one for the whole terminal. Entering 47/1047 now shows the host default instead of a blinking block. The plan already marks this as out of scope; fixing it would move the field, not change its values.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry saying the operator smoke checked the switcher caret instead of a plain-shell pane, and that the pane path is covered by `TestChildCursorStyleReachesParentVerbatim`.
   - Plan Task 4 Step 1 predicted "68 executable cases", but the Log measured 84 passing and 6 not-covered. The Revisions note can record the measured count.

```findings
findings:
  - id: new
    severity: Minor
    family: checkbox-claims-exceed-evidence
    title: |
      Operator-smoke checkbox claims a plain shell under couch was checked, but the Log says only the switcher caret was
    detail: |
      The issue Plan ticks the plain-shell-shows-a-bar row and Done-when names the same check. The Log says the smoke covered the couch switcher panel caret, not a plain-shell pane. Either run the plain-shell check, or reword the checkbox and Done-when and add a Revisions entry.
  - id: new
    severity: Minor
    family: default-blink-representation-per-layer
    title: |
      vt reports the default cursor with blink=true, while the frame requires Blink=false for it
    detail: |
      The vt handler stores the default as Steady=false, so the CursorStyle callback and the qualifier raw observation report blink=true, while Frame.Validate refuses a blinking default. No pair code registers the callback today; a future consumer must apply the endpoint rule instead of trusting blink.
```
