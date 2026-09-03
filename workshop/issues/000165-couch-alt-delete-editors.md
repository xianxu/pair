---
id: 000165
status: open
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-02
estimate_hours:
---

# Clear a text input with cmd+delete

> Filename slug predates the chord change (`#178` merged in on 2026-09-02);
> the branch name will read `alt-delete`. Not renamed, because the id is the
> ref that matters and a rename breaks nothing usefully.

## Problem

Emptying a text input means holding backspace. Every text-entry surface in the
workbench should clear on one gesture.

**cmd+delete already reaches pair — pair discards it.** The operator's Ghostty
config maps `super+backspace` to `text:\x15`, so the terminal sends `0x15`
(`^U`). `couchtty/panelkeys.go` decodes `\r\n`→Enter, `\t`→Tab,
`0x7f`/`0x08`→Backspace, and `0x20..0x7e`→Rune; `0x15` falls through `default:`
and is dropped. The key arrives and is thrown away.

**The pair-side contract is therefore `^U`, not "cmd".** Nothing below the
terminal knows about a Command key — the Ghostty binding is the operator's, and
a different terminal or config sends something else. Implement `0x15` and
cmd+delete works as a consequence; implement "cmd+delete" and there is nothing
to implement.

**The panel has four separate text handlers.** `KeyRune`/`KeyBackspace` are
handled at `couchtty/menu.go:364`, `:494`, `:549`, and `:596` (the last inside
`reduceTextKey`). Adding a clear is four edits, and three of them are the ones
that get forgotten — a value cleared in the name field but not the describe
field is worse than the feature's absence, because it teaches a gesture that
sometimes silently does nothing.

## Spec

**`^U` (`0x15`) clears the whole contents of the focused text input.** One
behavior, single-line and multiline alike.

1. **Consolidate first, then add.** Fold the four rune/backspace handlers into
   one text-editing reducer and give the new key one home (`ARCH-DRY`). Adding
   a fifth case in four places is how the next input key gets forgotten too.
   The reducer is pure, so this is testable without a terminal (`ARCH-PURE`).
2. **Decode `0x15` in `panelkeys.go`** as a `KeyClear` kind rather than
   special-casing a byte inside the reducer — the decoder is where every other
   control byte is named.
3. **Enumerate the surfaces; do not fix only the one that prompted this.** At
   minimum: couch's panel edit boxes (name, describe, path/start-args, and the
   switcher typeahead), and the nvim draft pane. `## Done when` must name every
   surface found, and a surface deliberately excluded must say why.
4. **Cursor lands at the start of the now-empty input**, in every surface.

**Draft pane, two specifics:**

- `^U` in nvim insert mode is already bound: it deletes text entered in the
  current insert session (bounded by `'backspace'`). Making it clear the whole
  draft **overrides a standard vim binding** — a deliberate choice, not an
  omission to fix silently.
- Clearing a multi-line composed prompt is the most destructive instance of the
  gesture. It must be a single `u` undo away, asserted.

## Done when

- `^U` clears the focused input in every surface named in the enumeration, each
  asserted, with the cursor at the start.
- cmd+delete clears the input in Ghostty under the operator's binding — an
  end-to-end check that the byte survives couch's interceptor and reaches the
  focused editor.
- The four `menu.go` text handlers are one; a shadow sweep confirms no second
  rune/backspace path survives.
- `0x15` is named in the decoder, not matched raw inside a reducer.
- In the draft, `^U` clears and a single `u` restores the full prior text.

## Plan

- [ ] Enumerate every text-entry surface; record the list in `## Log`.
- [ ] Consolidate `menu.go`'s four rune/backspace handlers into one reducer.
- [ ] `KeyClear` in `panelkeys.go` for `0x15`; clear-all in the reducer.
- [ ] Draft pane binding + undo assertion.

## Log

### 2026-09-01

Opened as "Support Alt Delete in Couch editors": `Alt+Delete`, clearing all in
a single-line box and only the cursor's line in a multiline box.

### 2026-09-02

Merged `#178` in and superseded the original chord and semantics on the
operator's call: the gesture is **cmd+delete**, and it clears **all** text in
both single-line and multiline inputs. `#178` is closed against this issue.

**Dropped in the merge, recorded so it can be recovered rather than
rediscovered:** the original multiline behavior — `Alt+Delete` clearing only
the cursor's line — is a genuinely different and useful operation, and it no
longer exists anywhere in this spec. If clearing one line of a multiline field
turns out to be wanted, it is a new issue with its own chord, not a
reinterpretation of this one.

The Ghostty binding (`keybind = super+backspace=text:\x15`) is the operator's
own configuration, not something pair ships. On another machine cmd+delete does
nothing until the same binding exists, while `^U` typed directly works
everywhere. Whether pair should ship that binding is a separate question about
pair owning terminal config.
