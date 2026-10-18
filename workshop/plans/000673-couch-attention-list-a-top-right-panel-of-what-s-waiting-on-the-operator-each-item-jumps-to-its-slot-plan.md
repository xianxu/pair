# Couch attention list Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Couch keeps a durable list of what is waiting on the operator: hub notifications, slots blocked on a prompt, and slots that asked a question. A top-right panel shows the list, each item jumps to its slot, and items survive a Couch restart.

**Architecture:**
- One pure reducer, `AttentionList`, owns the items and every transition: raise, condition clear, select, dismiss, and restart reconcile.
- Three existing signals feed it. None of them is re-derived:
  - `handleNotify` (the hub's `couch --notify`);
  - the blocked-set diff that `publishBlocked` already produces;
  - the `waitWatcher` episode, which gains an explicit end.
- The Console draws the panel as a top-right box on the operator-only layer, composed with the cast-codes layer. It jumps through the existing `switchTo` funnel.
- A small versioned store under the Couch namespace persists the list (`durablefile.WriteAtomicStaged`, the `resources_persist` pattern).

**Tech Stack:** Go (`couchcore`, `couchcmd`, `couchtty`, `couchkeys`).

---

## Findings that shape the design (read-only survey, 2026-10-18)

1. **The Alt+u finder is Parley's, not Couch's.** It lives in `parley.nvim/lua/parley/attention*.lua`; its key is now Alt+Space (pair#619). It is an nvim float picker that opens files, not slots. So "share its look and jump" can only mean:
   - **Look:** its row grammar, `state  marker  name  dir  HH:MM`, with a state colour.
   - **Jump:** Couch's own switch funnel (`switchTo` → `selectActor`), the one Ctrl+Return and switcher clicks already use.
   No code can be shared across the repos.
2. **pair#352 (open)** already specifies "stateful operator notifications" (pending until visited), and its ARCH-FUNERAL note says nothing durable should outlive Couch "unless the design says so". #673 is that design. Proposal: #673 supersedes #352, which closes as folded in.
3. **Attention is documented as ephemeral.** See `AttentionLedger` and `atlas/architecture.md:813`. This list is deliberately durable (an operator decision), so the atlas entry changes. The bell ledger (a hidden pane rang its bell) stays ephemeral and out of this list.
4. **Couch draws nothing at the top-right today.** Its only chrome is the bottom status row; the top-right is child (zellij) content. A panel or badge there covers a few cells of the child. The operator-only layer (`SetOperatorLayer`) has one slot, already taken by `codesLayer`, so the attention layer composes with it. It stays off the tapped overlay: the list is private and never broadcast.
5. **Prompt detection** (the red tab) arrives as a set via `publishBlocked` → `SetBlockedThreads`. A slot leaving the set is the only clear signal, so the reducer diffs sets. `couchmessage.ReadPicker` labels the kind: permission prompt, question or picker.
6. **Waiting on input** is `wait_notice.go`'s `classifyWait` (prompt / question / idle / picker), judged on a 15 s tick after `COUCH_WAIT_NOTICE_AFTER` (10 min). It has no end event: an episode ends implicitly when its key changes. This plan adds an explicit end.
7. **Slot identity:** `ThreadAddress` changes when a slot changes conversations (pair#607). Items are keyed by the slot reference (`Binding.Slot`, e.g. `pair:7`, as wait notices use) and resolved to the current thread at jump time.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `AttentionItem` | `cmd/internal/couchcore/attention_list.go` | new |
| `AttentionList` (reducer) | `cmd/internal/couchcore/attention_list.go` | new |
| `AttentionRows` (row model + format) | `cmd/internal/couchtty/attention_panel.go` | new |
| `topRightBox` anchor for `renderMenu` | `cmd/internal/couchtty/notes_menu.go` | modified |
| `ActionAttention` key | `cmd/internal/couchkeys/couchkeys.go` | modified |

- **AttentionItem** has the fields `{ID, Kind, Slot, Title, Body, Since, Visited}`.
  - **Kind** is one of three:
    - `notify`: a hub `couch --notify`; `Slot` is the hub's slot.
    - `prompt`: a slot blocked on a permission prompt or picker, from the red tab.
    - `question`: a slot that ended its turn with a question to the operator, from the `classifyWait` question class.
  - **Slot** is the `repo:N` reference.
  - **ID:** a `notify` item gets a fresh, sortable ID per notification; a condition item is keyed `kind:slot`, so there is at most one per slot and kind.
  - **Relationships:** an `AttentionList` holds N items; an item names one slot; a slot has zero or more items.
  - **Future extensions:** more kinds (an `exited` slot from #657, a REWORK verdict) are one more `Kind` plus a source hook.
- **AttentionList** is a pure value with transitions that return the new list plus effects. The state table below is its contract.
  - `Raise(item)` is idempotent for a condition item: a re-raise keeps the original `Since`.
  - `ConditionSet(kind, slots)` diffs the live set: new slots raise, missing slots clear.
  - `Select(id)` returns a `Jump{Slot}` effect and applies the kind's select rule.
  - `Dismiss(id)` removes the item.
  - `Reconcile(kind, slots)` runs at restart.
  - The list is capped at 50 items; the oldest `notify` item drops first, and a condition item never drops while its condition holds.
  - **DRY:** it is the single owner of attention state; the sources only report facts, as with `LifecycleNoticeSequence` (#574).
- **AttentionRows** formats `state  kind  slot  title  HH:MM`, newest first, in Parley's attention-finder grammar (finding 1). It also handles selection, scrolling and row geometry. It is pure and is tested on strings.
- **topRightBox:** `renderMenu`/`NotesMenu` take an anchor (bottom-right today, top-right added), so the attention panel reuses the box, `Contains`, `ItemAt`, `CloseAt` and `Draw`, not a second widget (ARCH-DRY).
- **ActionAttention:** one new chord that opens and closes the panel; the proposal is Ctrl+Alt+a, enhanced encoding only, as Ctrl+Alt+b does. The keymap collision tests already cover the table.

### State table (the `AttentionList` contract)

| Kind | Raised when | Select (Enter or click in the panel) | Cleared when | At restart |
|------|-------------|---------------------------------------|--------------|------------|
| `notify` | `handleNotify` accepts a hub notification (after auth, beside the OSC write) | Jump to the hub slot, then remove the item: selecting it *is* reading it | Selected or dismissed. Never by merely viewing the hub, which is nearly always on screen (the reason #673 exists) | Kept as is |
| `prompt` | The slot enters the blocked set (`publishBlocked` diff) | Jump to the slot; `Visited` is set and the item stays until the prompt is answered | The slot leaves the blocked set (the prompt was answered, by anyone), or dismissed | Kept, then reconciled against the first blocked set after startup; dropped if the slot is not blocked |
| `question` | `waitWatcher` judges the slot `waitQuestion` and starts an episode (one per episode, as today) | As `prompt` | The episode ends: the slot is busy again, its class changed, or its binding went away (new explicit end). Or dismissed | Kept, then reconciled on the first wait tick; dropped if the slot is not in a question episode |

Every row also has these columns:
- **Dismiss** (`d` or Delete in the panel) removes any item.
- **Badge:** the closed panel shows `● N` at the top-right whenever N > 0, in the bell (amber) style; the red style is used if any `prompt` item is open.
- **Order:** newest `Since` first.

The open-panel keys:
- arrows (and j/k) move;
- Enter selects;
- `d` dismisses;
- Esc or the chord closes;
- a click on a row selects, and a click outside closes.
The OSC desktop write stays as it is (the operator's "bonus").

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `attentionStore` | `cmd/internal/couchcore/attention_store.go` | new | `durablefile.WriteAtomicStaged`, namespace dir |
| notify source | `cmd/internal/couchcmd/notify_service.go` | modified | `handleNotify` |
| prompt source | `cmd/internal/couchcmd/message_service.go` (`publishBlocked`) | modified | registry liveness |
| question source and episode end | `cmd/internal/couchcmd/wait_notice.go` | modified | `waitWatcher.tick` |
| `Console.ObserveAttention` / panel | `cmd/internal/couchtty/console_attention.go` | new | operator layer, input routing, `switchTo` |

- **attentionStore** writes `NAMESPACE/attention/list.json`: versioned JSON, a size limit, `O_NOFOLLOW` on load, strict decode. It saves on every list change, and a failure is traced, never blocking (the `resources_persist` pattern).
  - **Injected into:** the Console's attention owner, as a seam with a fake for tests.
  - Load runs once at Console start; the loaded items show immediately as stale-until-reconciled.
- **Sources** call one Console entry, `ObserveAttention(fact)`, where a fact is `{raise | condition-set | episode-end}`. They pass facts only, never state (ARCH-PURE). The notify hook sits in `handleNotify`, not `NotifyDesktop`, so automatic-input-stall desktop notices don't enter the list (finding 2 of the survey).
- **Panel:** an attention layer composed with `codesLayer` into the single operator layer; routed first in `routeMouseEvent`, next to `routeNotesMenu`. A jump resolves the slot reference to the current thread through the switcher rows' `Target.Slot`, then calls `switchTo(target, false, arrivalNotification)`. That is the same funnel as Ctrl+Return.

**Test surface:**
- **Pure:**
  - the `AttentionList` table, one case per cell above, including: re-raise keeps `Since`; selecting a condition item does not remove it; selecting a notify item removes it; viewing the hub never clears a notify item; the cap drops the oldest notify item and never an active condition item; reconcile drops stale condition items and keeps notify items;
  - `AttentionRows` formatting and selection;
  - `topRightBox` geometry at small and odd sizes;
  - store round-trip, version and size refusal.
- **Console fakes** (the Done-when tests):
  - a hub notify appears in the panel at once, and Enter jumps to the hub slot (the `FakeHost` screen plus the tracker);
  - a blocked-set entry raises a `prompt` item, and dropping it from the set clears it;
  - restart: save, then a new Console loads the items, and the first blocked set reconciles them;
  - the badge renders at the top-right and the panel is absent from the broadcast tap;
  - a mouse click on a row jumps.
- **Wait watcher:** an episode end is emitted when the class changes, when the slot is busy again, and when the binding disappears.

## Chunk 1: single pass (one close)

- [ ] `AttentionItem` / `AttentionList` reducer with the full state-table test.
- [ ] `attentionStore` with round-trip and refusal tests.
- [ ] Sources: the notify hook, the blocked-set diff and the wait-episode start/end, with tests on the existing rigs.
- [ ] Console: `ObserveAttention`, the `topRightBox` anchor, the panel rows, keys and mouse, the badge and the `ActionAttention` chord, with `FakeHost` tests for the Done-when cases.
- [ ] Atlas (`atlas/couch.md` attention list; `atlas/architecture.md:813` attention durability), key help, the artifactpath inventory, guard tests, the recorded suite, `sdlc close`.
- [ ] Operator live check at the next restart (TL-held).

## Decisions for the TL

1. **Supersede #352** with #673 (close #352 as folded in).
2. **Condition items clear when the condition clears, whoever answered it.** A visit alone does not clear them, and neither does a jump without the answer. The operator's wording was "jumped to its slot and the condition cleared". A prompt answered elsewhere (by the hub, say) no longer waits on the operator, so keeping it until visited would be noise. Notify items clear on select or dismiss.
3. **"Waiting on input" means the `question` class only.** The `idle` and `prompt` (empty composer) wait classes are workers waiting on the hub, the TL's queue, not the operator's. Including them would bury real items.
4. **Key: Ctrl+Alt+a** opens the list; Ctrl+Return is unchanged.
5. **The top-right badge covers a few cells of child content** (zellij's tab line) while items are pending. The alternative is putting the badge at the right end of Couch's own bottom status row and keeping only the open panel at the top-right.

## Revisions

- 2026-10-18 (TL approval, ops:0): all five decisions settled.
  1. #673 supersedes #352.
  2. A condition item clears when its condition clears, whoever answered it.
  3. "Waiting on input" means the `question` class only.
  4. The panel opens with Ctrl+Alt+a, provided the key help and the keymap conflict check show it free; checked against Ctrl+Alt+b's neighbours before it is wired.
  5. **The badge moves to the right end of Couch's bottom status row,** so it never covers slot content; only the open panel draws at the top-right. This changes the `Badge` row above: the badge is a status-row span (clickable to open the panel, like the other status spans, `ColumnSpan.Contains`) rendered in `RenderStatusRow`, not a top-right overlay cell. The test for the badge moves to the status-row renderer.
