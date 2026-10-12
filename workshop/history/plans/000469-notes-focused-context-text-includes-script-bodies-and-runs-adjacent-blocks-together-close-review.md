# Boundary Review — pair#469 (whole-issue close)

| field | value |
|-------|-------|
| issue | 469 — notes: focused-context text includes script bodies and runs adjacent blocks together |
| repo | pair |
| issue file | workshop/issues/000469-notes-focused-context-text-includes-script-bodies-and-runs-adjacent-blocks-together.md |
| boundary | whole-issue close |
| milestone | — |
| window | e1328545db49f9dacf059e3ceb8d0d7847f18b62..915ba1031c61587463bbf321b2b78334399cf493 |
| command | sdlc close --issue 469 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T03:14:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I'm checking `query/render.go` for a second visible-text walker that might duplicate the new one (ARCH-DRY).

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix does what the issue asks. `DOMNode.RenderedText` (`cmd/internal/webcast/recording/dom.go:47`) skips the bodies of script, style, template and noscript, plus elements the recording marks hidden. It puts a space between blocks, table cells, list items, `br`, and flex/grid items, and collapses whitespace. Both models now build `Context.Text` from it, and the 500-rune window is unchanged. app-n3's context no longer contains the script, and the regenerated goldens read cleanly. I ran `go test ./cmd/internal/webcast/...` at head and all 7 packages pass. I also reverted `Structural.Context` to `TextContent` as a mutation check: both `TestStructuralMeetsTheModelContract` and `TestContextOfAppReportsIsRenderedText` went red, so the contract test really catches the bug. One thing should be fixed before shipping, and it's non-blocking. `query/render.go` still has its own walker, `spacedText`, that works out "what a reader sees" in a different way. So the claim of one walker in the plan's ARCH-DRY line covers the notes models but not the whole tree, and the label in the same golden line is now worked out differently from the `text:` underneath it.

1. **Strengths**
   - `RenderedText` sits beside `TextContent` in `recording`, so both models use one pure function (`structural.go:62`, `fake.go:55`). No IO.
   - The model contract (`contract.go:27,88`) puts a script inside the article and checks for both defects. The mutation check above confirms it fails without the fix.
   - `TestRenderedText` covers each rule in a table, including the edge cases: Tailwind `hidden` with `md:block` and with `hover:`, and an upper-case inline style.
   - `NoteAt.Text` stays on `TextContent`, which is the right call: changing it would change stored anchors. The Log says so explicitly.
   - The atlas entry is updated (`atlas/webcast-format.md:216`).

2. **Critical findings:** none.

3. **Important findings**
   - **ARCH-DRY / ARCH-PURPOSE:** `cmd/internal/webcast/query/render.go:277` `spacedText` is a second, older walker for visible text. It skips only script and style, not template, noscript or hidden elements. It also puts a space between every text node, so inline runs get split (`Ship<b>ping</b>` comes out as "Ship ping"). It feeds the `context: within @… "…"` label at `render.go:269`, so the label in the golden is worked out differently from the `text:` line under it. The fix is to replace `spacedText(n)` with `n.RenderedText()` and regenerate the goldens. The bug is older than this work, but it's the same rule the issue fixes, and it's cheap to fix.

4. **Minor findings**
   - `visibility:hidden` drops the whole subtree, but in a browser a descendant with `visibility:visible` still shows (`dom.go:97`).
   - Every `option` counts as text (the stress golden shows "alpha beta gamma"), but a closed `<select>` shows only the selected option.
   - `setOf` (`dom.go:124`) copies `notes.set` (`landmark.go:95`). They're in different packages, so this is acceptable.

5. **Test coverage notes:** the pure function has a direct table test, the contract is checked for both models, and the app-n3 fixture is pinned. Not covered: a focus element that is itself hidden or empty, so `window` falls back to start 0. That's harmless.

6. **Architectural notes**
   - ARCH-DRY: flagged; see item 3.
   - ARCH-PURE: pass.
   - ARCH-PURPOSE: passes for notes; flagged for the query sibling in item 3.
   - ARCH-MOCK: pass. There are no external dependencies.
   - ARCH-CONSTRAINTS: pass. Each `Context` call walks the subtree once.
   - ARCH-SECURE: pass. Missing attributes or a nil attribute map are read safely, and nothing can panic.
   - ARCH-ORDER: pass. It holds no state between events because it's a pure function over one DOM snapshot.
   - ARCH-FUNERAL: pass. It creates nothing durable because it only computes in memory.

7. **Plan revision recommendations:** if `spacedText` stays as it is, add a Revisions entry to the plan saying the "one walker" in its ARCH-DRY line covers the notes models only, and `query`'s `spacedText` is left for later.

```findings
findings:
  - id: new
    severity: Important
    family: single-rendered-text-walker
    title: |
      query/render.go spacedText is a second visible-text walker that disagrees with RenderedText
    detail: |
      render.go:277 skips only script/style (not template/noscript/hidden) and splits inline text nodes; it feeds the "context: within" label at render.go:269 beside the RenderedText-based text line. Replace with n.RenderedText() and regenerate the goldens.
  - id: new
    severity: Minor
    family: rendered-text-fidelity
    title: |
      visibility:hidden drops visible descendants; every select option counts as rendered text
    detail: |
      dom.go:97 treats visibility:hidden as hiding the whole subtree, unlike CSS; blockTags includes option, so a closed select contributes every option (see the stress golden's "alpha beta gamma").
```

---

## Re-review — 2026-10-12T03:17:18-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 469 — notes: focused-context text includes script bodies and runs adjacent blocks together |
| repo | pair |
| issue file | workshop/issues/000469-notes-focused-context-text-includes-script-bodies-and-runs-adjacent-blocks-together.md |
| boundary | whole-issue close |
| milestone | — |
| window | e1328545db49f9dacf059e3ceb8d0d7847f18b62..9a8b36a714da56bebd08ea85f4aa14920697b151 |
| command | sdlc close --issue 469 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T03:17:18-07:00 |
| verdict | SHIP |

## Review

Disposing the two prior findings now. I'm running a quick mutation check that the visibility-inheritance fix has a test that goes red without it.

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are fixed and backed by evidence, and nothing new needs to block. `RenderedText` (`recording/dom.go:48`) is now the only function that decides what visible text is. Both note models use it (`structural.go:62`, `fake.go:55`), and so does query `Describe` (`render.go:269`), so the context label and the context text can no longer disagree. All webcast packages pass `go test ./cmd/internal/webcast/...`. The Done-when items hold: app-n3 has no script text and its cells read apart (`structural_test.go:147`), and the contract and goldens pass.

1. **Strengths**
   - There is one source for visible text, and every consumer of it now uses it. The remaining `TextContent` calls are deliberate: `render.go:93` and `:202` count script/style characters and read `<title>`, and `structural.go:26/45` and `fake.go:26` keep `NoteAt.Text` unchanged. The issue Log explains why: changing it would change stored anchors and how they match.
   - The model contract (`contract.go:88`) puts a script that names the anchor's own text inside the article. Both models must exclude it, so either model falling back to `TextContent` fails.
   - `TestRenderedText` is a pure table with no IO. It covers blocks, inline runs, cells, `br`, flex items, script/style/noscript/template, the hidden variants, and the Tailwind display variants.
   - The select behaviour is a decision, not an oversight. A select's live choice isn't in the DOM, so listing every option avoids contradicting the `selected [2]` event printed beside it. This is documented on `RenderedText` and pinned by a test case.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** none new. One edge I noticed: `visibility()` checks with `strings.Contains` on the flattened inline style, so a style like `visibility:hidden;visibility:visible` resolves as hidden. It belongs to family `rendered-text-fidelity`, and inline styles are rarely written that way, so I'm not raising it.

5. **Test coverage**
   - I removed the inheritance logic in `dom.go` as a mutation; `TestRenderedText` failed (`"", want "shown"`) and I restored the file (status clean).
   - The BR-1 fix is pinned by the regenerated goldens (`blog.events.txt` `<head>` line) and the `TestSelect` row `"AB"`. Putting `spacedText` back turns both red.

6. **Architecture**
   - **ARCH-DRY: pass.** The second walker is gone, and `setOf` and `inlineStyle` are shared helpers.
   - **ARCH-PURE: pass.** `RenderedText` is a pure tree walk.
   - **ARCH-PURPOSE: pass.** Every consumer was checked. The one exception, `NoteAt.Text`, is out of scope and recorded as such in the Log.
   - **ARCH-MOCK: pass.** `FakeModel` and `Structural` share the same contract and the same text source.
   - **ARCH-CONSTRAINTS: pass.** A single O(n) walk per context, and the 500-character window is unchanged.
   - **ARCH-SECURE: pass.** Recorded DOM attributes are only string-matched; nothing is evaluated or trusted further.
   - **ARCH-ORDER: N/A.** It holds no state between events because `RenderedText` is a stateless function of one DOM snapshot.
   - **ARCH-FUNERAL: N/A.** It creates nothing durable because the output is computed on demand. The goldens only changed content.

   For upcoming work: if `NoteAt.Text` ever moves to rendered text, that needs a migration plan for stored anchors and for `Resolve` matching.

7. **Plan revisions:** none. The plan's checklist matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      spacedText removed; Describe uses n.RenderedText() (render.go:269); blog.events golden and TestSelect "AB" row pin it and go red if the old walker is restored.
  - id: BR-2
    disposition: addressed
    note: |
      visibility now inherits and a descendant can set it back (dom.go:62); the dom_test case fails under the old subtree-skip, confirmed by mutation. Listing every select option is a documented decision: the live choice is not in the DOM.
```
