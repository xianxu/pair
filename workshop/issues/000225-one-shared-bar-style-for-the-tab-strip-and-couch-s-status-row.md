---
id: 000225
status: open
deps: []
github_issue:
created: 2026-09-10
updated: 2026-09-10
estimate_hours:
---

# one shared bar style for the tab strip and couch's status row

## Problem

Three bars sit at the bottom of the workbench and read as unrelated:

- **draft nvim's statusline** — the `Alt: ← history 115 < * [q=queue] > 0 queued →` bar;
- **`pair term`'s tab strip** in the layout3 right pane (`#199`) — renders `[terminal 1]`
  as plain text;
- **couch's status row** — `tools  parley.nvim  ariadne  [brain]  pair`.

The operator wants the two Go bars to adopt the draft bar's treatment, **with one
style shared between them** rather than two that drift.

### The operator's four rules

1. **Foreground: keep the current colour** (reads white on this machine).
2. **Inactive tabs subdued**, like the draft bar's `history 115`.
3. **Keep couch's notification colour** — it reads well.
4. **Background: keep the current colour**, slightly darker than the draft bar. Note
   the observed inconsistency when the right pane runs a program with its own
   colour scheme (nvim with lualine) — the strip's row does not match what surrounds
   it.

**And it must work on a light scheme.**

### What each bar does today — read from the code

| bar | foreground | inactive | notification | background |
|---|---|---|---|---|
| draft nvim | colorscheme `StatusLine` | *(no distinct style)* | — | colorscheme `StatusLine` |
| couch row (`couchtty/reserve.go:103-107`) | terminal default | brackets mark *active* only | `\x1b[38;5;220m` | terminal default |
| tab strip (`termcmd/strip.go`) | terminal default | *(no distinct style)* | — | terminal default |

One detail changes rule 2's meaning: in the draft bar, `history %d` is **not wrapped in
any highlight group** (`nvim/init.lua:2292` — only `Alt:`, `<-`, `->` use
`%#PairAltKey#`). So "the `history 115` colour" is the **colorscheme's `StatusLine`
foreground**. It lives inside nvim, and the Go bars cannot read it.

### Prior art — `#217` already made the hard decision

`#217` (*dim the right pane's tab strip when the pane loses focus*, open) styles the
**same strip**, and its commit `33b42320` settled the both-schemes question:

> Faint was rejected for the reason this repo keeps paying for: a terminal that
> ignores SGR 2 renders nothing different, silently. A colour either shows or is
> visibly wrong.
>
> …menu_render's 238/240/245/250 age ramp is fixed greys chosen against a dark
> background, and on a light scheme it inverts — 238 becomes more prominent than
> 250, so oldest would read loudest. ANSI 90 (bright black) is proposed instead,
> being the one grey terminals theme themselves, with an OSC 11 fallback only if
> measurement demands it.

**Build on that decision; do not re-derive it.** Both issues edit `strip.go`'s
rendering, so they must coordinate or they will collide.

## Spec

**One style definition, two renderers.** `#199` M1 already lifted the row *primitive*
into `hostty.Reservation` so both bars share it. The style belongs beside it — one
palette consumed by `RenderStrip` and `RenderStatusRow`, never two copies of the
escape codes (`ARCH-DRY`). The two bars keep their own *content* (tabs vs actors), as
the atlas's "shared structure, local policy" split already prescribes.

### The four rules, made theme-safe

**1. Foreground → terminal default (`SGR 39`), not literal white.** Read carefully: on
a light scheme, literally keeping white makes the active tab invisible. The operator
sees white because the terminal's default foreground *is* white on a dark scheme.
Default foreground preserves what they see today **and** is correct on light.

**2. Inactive → ANSI 90**, per `#217`. The terminal themes bright black for its own
scheme, so it is subdued on both. **Keep the brackets** on the active tab as a
non-colour cue: they survive a terminal with colour off, a scheme where 90 lands close
to the foreground, and colourblindness.

**3. Notification → decide, don't assume.** 256-colour 220 is a **fixed** amber — the
same class `#217` identified in the fixed-grey ramp. It reads well on dark; amber on a
light background is low-contrast. Two options:

- keep 220 and verify it on a light scheme;
- switch to ANSI 33 / 93 (yellow / bright yellow), which the terminal themes.

The operator likes 220, so the default is to keep it — but the light-scheme check is
not optional, because that is exactly the failure `#217` just caught.

**4. Background → terminal default (`SGR 49`).** "Keep current" and "works on light"
are *compatible* only if the bars use the default background rather than a hardcoded
one. Do not introduce a hardcoded dark bg to match the draft bar — that is the one
choice that breaks light schemes outright.

### The image-15 inconsistency — confirm, don't assume

With nvim + lualine running in the right pane, the strip's row does not match its
surroundings. Two candidate causes, and they need different responses:

- **The child's own painting.** nvim paints its background and lualine its own bar
  colours. Nothing the strip does can match an arbitrary child's scheme, and it should
  not try — a default background is the correct invariant.
- **Background-colour erase.** An erase (`\x1b[K`) fills with the **currently active**
  background; if a child's SGR state is live when the strip paints, the cleared row
  takes the child's colour. `#199` already addressed this ordering (`a58b99f3` *the
  reset belongs to the erase, not in front of it*; `ac09b4a6` *the cursor and the
  colour, both wrong at the shared primitive*), so a recurrence would be a residual,
  not a new mechanism.

Determine which it is on the real screen before changing anything; the second is a
defect, the first is expected behaviour.

### Matching the draft bar exactly is not achievable as stated

The draft bar's subdued text is a colorscheme colour inside nvim; the Go bars emit raw
SGR. They will be *close*, not identical. If an exact match matters, the lever is the
**other** direction: give the draft bar's `history`/queue text an explicit highlight
group mapped to the same bright-black, so nvim follows the shared palette instead of
the Go bars chasing a colorscheme they cannot see. Recorded as optional; the operator
may be satisfied with close.

### Interaction with `#217`

`#217` dims the **whole** strip when the pane loses focus; this issue subdues
**inactive tabs**. If both use ANSI 90, an inactive tab inside a dimmed strip is
indistinguishable from the active one. Decide what the active tab looks like in an
unfocused strip before either lands — probably the brackets alone carry it.

## Done when

- Both Go bars render from **one** style definition; `grep` finds the palette's escape
  codes in one place.
- Active tab / actor: default foreground, with brackets. Inactive: ANSI 90.
  Notification: the chosen colour. Background: terminal default.
- **Verified on a light scheme and a dark scheme**, recorded as screenshots or a
  captured render — not asserted from the code. Includes the notification colour.
- The image-15 cause is identified and either fixed (if BCE) or documented as expected
  (if the child's painting).
- `#217`'s dimmed-strip state and this issue's inactive state are distinguishable.
- `atlas/` records the shared palette and the both-schemes rule, so the next bar does
  not reach for a fixed 256-colour grey.

## Plan

- [ ] Coordinate with `#217` — sequence or merge; both touch `strip.go`.
- [ ] Decide the notification colour (keep 220 vs ANSI 33/93) after a light-scheme check.
- [ ] Extract the shared palette beside `hostty.Reservation`; point both renderers at it.
- [ ] Apply the four rules; keep brackets.
- [ ] Diagnose the image-15 inconsistency on the real screen.
- [ ] Verify on light and dark; record the renders.
- [ ] Optional: explicit draft-bar highlight group for an exact match.

## Log

### 2026-09-10

Operator request with two screenshots, covering the right-pane strip and couch's row
in one ask ("another task, or maybe in the same task"). Filed as **one** issue because
the requirement is a *shared* style — two issues would mean two palettes, which is the
drift the request is trying to remove.

Scoped after reading all three bars and finding `#217`'s commit, which had already
answered the hardest constraint — subdued on both schemes — with a measured argument.
The one place this issue departs from the operator's literal wording is rule 1:
"keep white" is read as "keep the default foreground", because literal white is
invisible on a light scheme and the operator explicitly asked for light-scheme
support.
