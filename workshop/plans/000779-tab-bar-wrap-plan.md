---
collaboration-mode: discussion
---

# Two-line Couch tab bar implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Use bounded independent tasks where the context is capturable; preserve one owner for Console publication. The SDLC close owns the single fresh-context boundary review.

**Goal:** Show overflowing Couch tabs on two lines, keep the active tab reachable, and report any remaining hidden attention without corrupting pane, broadcast or input geometry.

**Architecture:** A pure bounded planner produces one immutable status-bar layout. The compositor derives a trusted chrome height from the rectangular bottom surface; Console publishes that surface, child geometry and hit extents under its existing publication lock. Broadcast visibility, privacy and annotation guards consume the composed frame's geometry rather than guessing the last row.

**Tech stack:** Existing Go terminal compositor, Couch fake-host/native PTY fixtures, broadcast server and vanilla JavaScript viewer. No dependency, timer, polling loop or background worker added.

## Authority and scope

Operator Spec is in pair#779. TL c5c04f63 approves full flow and these choices: attention badge stays right of line1; transient notices occupy free space on the final line; active remains visible as a full-label chip alone if needed, clipped but clickable at absurd widths. Attention chips take priority over ordinary chips; hidden attention is explicit in the overflow marker. TL plan approval and subsequent refinement approval are recorded in Revisions.

Includes the manual #757 archive repair, ariadne#532 class: one separately named commit, two100%-identical renames into history beside its archived parent plan. TL c1d5a9d6 approves it riding with #779. Name it explicitly in the PR description. Do not reopen/re-close #757 or retry its old archive proof.

## Alternatives

1. **Shared layout plus composed-frame geometry (chosen).** All consumers derive the same reservation, including broadcasts and local clicks. More than a one-file render edit, but it preserves the existing ownership boundary.
2. Wrap an already-styled string. It loses group/active decisions and cannot produce reliable row-aware hit spans; it also leaves the child the wrong size.
3. Search the last two rows for red LIVE text or keep a separate broadcast offset. Child content can imitate that text, and independent offsets can disagree with accepted publication. Rejected: use compositor-owned metadata.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| statusPacking / renderStatusBar / packStatusChips | cmd/internal/couchtty/status_layout.go | new |
| RenderedStatusBar and row-aware chip spans | cmd/internal/couchtty/status_layout.go, reserve.go | new/modified |
| RenderStatusRow and shared row drawing | cmd/internal/couchtty/reserve.go | modified |
| Frame.ChromeRows and shared child-geometry validation | cmd/internal/terminal/frame.go, render.go, presenter.go | modified |
| trusted control-row projection; ViewerFrame; annotation geometry | cmd/internal/broadcast/indicator.go, privacy.go, marks.go, notes.go, notehalo.go | modified |
| ScreenLayout / AnchorFor | cmd/internal/couchtty/noteanchor.go | modified |

RenderedStatusBar owns at most two RenderedStatusRow values and their row-relative extents. Existing styling, controls, sanitation and width clipping remain in one row drawer; the planner supplies actor cells rather than reimplementing those rules. No mutable layout authority leaks to consumers (ARCH-DRY, ARCH-PURE).

Frame.ChromeRows is a scalar on an owned immutable publication. Endpoint snapshots have zero; Compose derives it from bottom cells, validates whole rows and leaves at least one child row. Already-composed child frames are refused. Console private panels explicitly declare their bottom surface. Clone/retained frames preserve it. It is not inferred from colors/text. Existing zero/one-row terminal consumers retain their geometry and output.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Console chrome publication and resizing | cmd/internal/couchtty/terminal.go, console.go, console_menu.go | modified | Presenter FIFO and owned child PTYs |
| Console mouse/codes/attention/note routing | terminal_input.go, console_castcodes.go, console_attention.go, console_noteanchor.go | modified | committed layout and local/remote input |
| Presenter Select/ResizeLayout/UpdateChrome/ChromeState | cmd/internal/terminal/presenter.go | modified | one serialized physical parent writer and child resize callback |
| Broadcast admission and live frame delivery | cmd/internal/broadcast/hub.go, session.go, stream.go, server.go | modified | trusted current publication and SSE/browser frame delivery |
| Viewer annotation geometry | cmd/internal/broadcast/web/notes.js and frame consumer | modified | server-owned frame geometry, never client authority |

Use existing stateful fake host, child endpoint and actual loopback broadcast server seams. Native conformance uses owned PTY children and joins every process; operator live look uses the approved staging procedure. No function-call-only replacement for terminal publication or broadcast interaction (ARCH-MOCK).

## Packing policy

1. Compute sanitized labels, full/short chip widths, fixed group boundaries and status prefix budget once. Existing group identity (including numbered hubs) remains authoritative. Preserve existing exact bytes when all actor chips fit on one line; a long transient notice alone never creates line2.
2. Pack complete groups greedily. Move a group intact to line2 when it fits a full line but not the first remainder. Split a group into chip units only if that original group exceeds a full line. A continuation begins with the repository's full label; later members may use :N. No separator starts a row.
3. If the unfolded groups fit in two lines, render all of them with no +N. Keep cast/Chrome/capture controls left of line1 and attention at its right edge. Notices use only free space on the final line.
4. Beyond two-line capacity, select active first, then attention (hub, blocked, bell), then ordinary actors. Among retained actors preserve original display order. No ordinary actor survives while an eligible attention actor is hidden. Overflow may show a full-label active chip separately when needed, as approved.
5. Draw one whole trailing +N on line2, counting every omitted actor, including placeholders. Draw +N ! if any omitted actor needs attention. Do not claim a count for merely shortened labels. If the terminal is too narrow for both active and the whole marker, active wins and the marker is omitted; never draw a bare +. At such widths visibility means a clipped positive-width clickable active span, not impossible complete text.
6. When terminal height permits only one chrome row, use the same bounded priority fallback on that row. Reserve a positive-width active chip before fixed controls, attention badge or overflow; draw only controls that fit the remaining prefix. Omitted/clipped LIVE fails the broadcast indicator gate, never authorizes an unmarked stream. With only one total host row, preserve today's no-reservation fallback so the child remains at least one row. At normal heights the bar is strictly one or two rows.

Overflow selection uses measured full-label chip widths so adding retained actors cannot reduce a candidate's cost through changed abbreviations. Group oversized classification is fixed from the original group, not recomputed per candidate. Precompute each actor's priority rank; search the largest retained priority prefix with allocation-free O(A) candidate packing, reserving a bounded worst-case whole overflow suffix. Render retained actors in source order once. Prefer a small conservative unused suffix over an unbounded search for a maximal packing; normal one/two-line cases use exact widths and no conservative reservation. This gives O(B + A log A), not repeated layoutChips/keptEnd prefix scans. The overflow fallback never repeats the old quadratic fitChips strategy.

## Shared geometry and publication

- Terminal composition accepts a rectangular bottom surface of zero or more complete rows, strictly fewer than host rows. A shared pure helper owns the arithmetic/validation used by Compose, Presenter selection, resize and input child bounds; no hardcoded subtract-one copy.
- Terminal Frame validation and history equality/reset comparisons include ChromeRows. A metadata-only change still reaches taps/broadcast even when the physical renderer produces no byte diff.
- Console builds a proposed immutable layout from one status-model snapshot under publishMu. Unchanged height uses UpdateChrome; a height transition uses existing ResizeLayout with the selected child's ResizePTY callback. A nil publication result means admitted, not necessarily physically painted. Presenter.ChromeState returns a serialized read-only snapshot of the admitted chrome revision, host/row geometry and confirmation. Each Select, Panel, ResizeLayout or UpdateChrome admits a revision; successful paint confirms that revision, and an output stall revokes confirmation even if the method returned nil. Automatic retry paints the admitted layout under the same revision; no new timer or Console poll is added. Console stores pending hit extents with the admitted revision after the call and snapshots confirmation before geometry-dependent actions. Resize remaining children to admitted geometry; InputEnded children retain existing skip behavior.
- Startup/installation, actor selection, actor/status changes, menu presentation and host resize all use this same layout owner. Selection first resizes the candidate child when its geometry differs; it cannot Select a stale-size frame. Startup prediction is reconciled before first presentation. Background children resize through the existing owner; a future selection revalidates size.
- Switcher height, menu extents and footer use hostRows minus accepted chrome height. Resize rewraps and paints through one synchronized-output frame, never a separate erase/new-row paint. Existing routing-answer handling and publication fencing remain intact.
- While painting is unconfirmed, or Console pending revision differs from the confirmed Presenter revision, local/remote geometry-dependent mouse presses and chrome actions are dropped; no stale layout is used. Presenter independently refuses child mouse admission under unconfirmed geometry. Motion/releases still traverse existing gesture ownership so cancellation/release joins correctly. Keyboard and paste keep existing logical-admission behavior. After a successful automatic repaint, the same pending extents become usable through the matching confirmation snapshot; no callback needs to re-enter Console from Presenter.
- Chip spans include relative row. Controls retain column spans on relative row0; convert SGR/remote coordinates once against the committed top chrome row. Blank second-row cells are chrome and never go to the child. Pointer motion/release keep existing gesture ownership/cancellation.
- ScreenLayout carries per-row chip extents and chrome top/height. Note anchors identify either chrome row correctly, with actual text from the captured frame.

## Broadcast contract

Compose/Console are the trusted producers of ChromeRows. IndicatorShown and PointerShown read the first declared chrome row, and fail closed without declared valid chrome. A child's red LIVE-looking text in either last row is never authority. ViewerFrame hides private menu contents while retaining all declared chrome rows and their metadata.

Propagate the trusted current control row through Hub.Current, Session point/note admission, mark overlays, pin placement, outlines and halos. Structural note parsing validates shape/limits; current-frame admission validates available pins and protected controls. Existing notes retain their stored grid; current layout determines guard visibility after resize. Guard the actual LIVE/pointer cells on row0 of chrome, including wide-cell overlaps; do not reserve the same columns of line2 unnecessarily.

Add the bounded integer chrome_rows field to the live server-owned frame message, Stream Next and Join. Send geometry changes even when ANSI bytes are empty. Browser note guards consume that field; absent field retains the legacy one-row server convention for old messages, but clients never submit authoritative control geometry. Server-side admission/overlay always enforces the current trusted frame even if an old browser ignores the new additive field. Viewer labels/counts and transport privacy do not change.

No persisted terminal.Frame encoder/digest exists: diagnostic capture stores byte observations, not typed frames. Thus there is no capture-schema migration. The live wire field is additive and readers continue tolerating unknown fields; known field bounds remain strict (ARCH-EVOLVE).

## State design

Actions: resize host, add/remove/focus a tab, change attention, open/close switcher, click either row, place a broadcast mark/note.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| All actors fit first row | existing one-row bar, full child height | resize/add tabs | none |
| Groups require second row | two rows, child/menu one row shorter | resize/remove tabs | child's own resize/reflow only |
| Two rows still insufficient | active and attention prioritized; +N or +N ! | focus a tab/open switcher/resize | hidden labels only, no tab/state deletion |
| Physically tiny host | clipped active, whole marker if possible; bounded height | enlarge terminal | unreadable labels while tiny |
| Actor attention changes wrapping | one admitted layout plus matching revisioned hit map; usable after paint confirmation | click/focus/resize | child's resize/reflow only |
| Resize admitted, writer stalls or accepts only a prefix | pending layout, geometry actions suppressed; keyboard/paste unchanged | existing automatic repaint after output recovers | geometry clicks during uncertainty are dropped, no queued click replay |
| Automatic repaint succeeds on pending revision | hit map becomes usable without another layout write | click/focus | none |
| Physical/child resize fails | existing terminal failure path; no accepted mismatched hit map | existing restart/recovery | existing in-flight terminal work limits |
| Private switcher broadcast | placeholder body, both public chrome rows retained | close switcher | private menu content is withheld |
| Frame metadata changes without ANSI delta | browser receives current guard geometry | retry navigation/place note | stale annotation attempt may be refused |

Covers one/two/tiny-height and active/attention overflow, both input paths, privacy and resize. Does not add scrolling tabs, a third row, persistence, a new service, changed agent lifetime, or guaranteed visibility of every attention chip when they physically exceed capacity; hidden attention remains explicit.

## Bounds, lifecycle and architecture

Growth drivers are actor count A, sanitized label/glyph bytes B and host cells C×R. Measure/sanitize once O(B); packing O(A log A), materialization O(A), at most2×C chrome cells. Existing compositor remains O(C×R) worst-case and owns its bounded frame queue. Count planner candidate/actor visits at N=64/128 and assert B + constant*A*(bits.Len(A)+1), with all-kept, many-group and oversized-group inputs; also assert bytes scanned do not multiply by search passes. No timing-only oracle (ARCH-CONSTRAINTS).

Creates no durable runtime artifact, cache, row history or background worker. Layouts/metadata die with the existing frame/Console lifetime. The only durable change outside normal code/process docs is the explicitly approved #757 evidence rename; its existing history lifetime is unchanged (ARCH-FUNERAL).

ARCH-DRY/PURE: one planner and one trusted geometry owner; existing styling and compositor reused. ARCH-PURPOSE: panes, input, broadcasts and notes consume the wrapped geometry. ARCH-MOCK: stateful host/endpoint/server tests plus native conformance. ARCH-SECURE: no child-text-derived broadcast authority; private body stays private. ARCH-ORDER: publication lock plus Presenter FIFO joins geometry and drawing. ARCH-EVOLVE: additive live message only; no persisted writer introduced.

## Verification strategy by function

| Risky function | Adversarial family | Mechanical guard |
|---|---|---|
| renderStatusBar / packStatusChips / RenderStatusBar / shared row drawer | constrained grouped labels and conflicting active/control/attention priorities | pure exact layout/span invariants plus seeded fuzzing; per-row width bounds and no ordinary survivor while attention is hidden |
| packStatusChips measured candidate packing | hostile actor/byte growth and kept-heavy inputs | N/2N operation and sanitation counters reject worse than declared logarithmic visits and linear bytes |
| Compose / ChildGeometry / Frame.Validate | malformed or incompatible rectangular chrome | pure validation/composition tests preserve child minimum and one-row compatibility |
| Presenter.ResizeLayout / ChromeState / paintPublication / mouseInput | admitted but unpainted layouts, partial writes and geometry-changing drags | controllable writer and endpoint barriers prove pending revision blocks mouse until successful retry; keyboard/paste and drag cleanup unchanged |
| Console.paintNow / publishSelectionPublishing / showMenuPublishing / onResize / installChild | status and resize transitions across publication boundaries | existing fake host/PTY tests assert admitted child/menu geometry, pending extents and confirmation agree; native resize/scroll conformance |
| routeMouseEvent / routeCastCodes / routeAttentionPanel / AnchorFor | stale or ambiguous row/column coordinates | both local and remote routes use confirmed revision; positive actor/control targeting and zero child-delivered chrome actions |
| IndicatorShown / PointerShown / ViewerFrame | spoofed child authority and private two-row publications | explicit trusted-metadata negatives; exact public-chrome preservation and private-body absence |
| Marks.Add / Marks.Overlay / NoteBoard.Add / DrawNotes / Session.acceptNote | guard movement and stale or hostile submitted geometry | actual current-frame admission and rendered-cell assertions protect real controls, including wide-cell overlaps |
| Stream.Next / Stream.Join / server frame delivery / viewer note guards | metadata-only transitions and late/stale clients | stateful server/browser contract tests assert current additive geometry, no client-supplied authority and legacy missing-field behavior |

## Implementation tasks — one atomic close boundary

- [x] Pin RED pure packing/composition/guard regressions and record observed failures before implementation.
- [x] Implement pure measured planner and shared row rendering with counted complexity pins; run targeted planner/fuzz tests and commit.
- [x] Generalize compositor geometry/metadata and Presenter resizing; test one-row compatibility and one↔two transitions, then commit.
- [x] Propagate trusted chrome geometry through broadcast privacy, marks/notes and additive viewer frames; run Go/JS tests and actual-server conformance, then commit.
- [x] Integrate Console publication, child/menu sizing, row-aware local/remote input and anchors; run fake-host and native lifecycle tests, then commit.
- [x] Update atlas/couch.md, atlas/terminal.md/broadcast map as needed and index links; audit artifact/source-reader classifications for new files, run inventory and focused race/repolint checks.
- [ ] Stage the concrete built head for the operator via hub, record the live result; checkpoint/push and run recorded affected (or suite fallback), close once through SDLC, then merge through its retained runner and report landing.

Targeted commands: go test ./cmd/internal/couchtty -run 'Test.*Status|Test.*Wrap|Test.*Chrome|Test.*Anchor' -count=1; go test ./cmd/internal/terminal ./cmd/internal/broadcast -count=1; scoped race runs for changed Console/Presenter/guard cases; existing Node viewer tests and make test-affected-inventory. Native test names are added beside the exact scenario, not a second full-suite loop. All source-reading tests receive per-reader audit/certificates after git add. Final recorded checks and CI remain mandatory.

## Revisions

- 2026-10-21: TL a01bc13f-f20c-4084-86d5-2f1e85796a01 approves this full design and test set; proceed to change-code. Operator stage must wait until #752 lands because it currently holds pair:0; TL sequences the look with #737's live day. No staging before hub clears that dependency.

- 2026-10-21: Plan-quality round1 PQ-1/PQ-2 and minor PQ-3: clarified admitted versus confirmed geometry through Presenter.ChromeState, pending Console revisions and dropped geometry-dependent actions during uncertain paint; preserved keyboard/paste and existing automatic retry. Tiny one-row active wins over controls as well as count; omitted/clipped LIVE fails closed. Compressed the verification contract into named risky functions and strategies. TL ba876ba6-14f8-4bdc-80a4-6ede8d57ed6a approves this refinement. No implementation yet.

- 2026-10-21 implementation clarification (ARCH-ORDER): confirmation snapshots also hold the existing publication lock through Presenter FIFO read and Console revision comparison; anchors already owning it use the no-lock Publishing helper. This realizes the approved atomic pending-map contract and fixes a measured close-click race without new state or timing policy. TL b07bf583 explicitly orders recorded suite and close while operator staging waits for pair:0; live acceptance still precedes landing.

- 2026-10-21 BR-1 (core-concepts-code-traceability): corrected Core concepts and verification table names to delivered pure planner entities statusPacking, renderStatusBar and packStatusChips. These replace the provisional statusPlan/planStatusRows names; behavior and approved design are unchanged. Sweep found no other unqualified provisional planner entity references.

- 2026-10-21: Made the approved operator qualification concrete below while the recorded suite waits. These steps exercise existing approved behavior; no implementation or release-order change. TL still owns the stage slot and acceptance.

- 2026-10-21 TL22868012: integrate current main before operator staging so the live supervisor retains accepted #752/#757/#776 features. Targeted Console checks precede replacement stage; the landing merge runs the authoritative integrated full suite. The initial0b0fa2db stage is superseded, with no operator acceptance claimed.

## Operator look after stage clearance

Use a fresh Couch process built from the staged revision; the running supervisor keeps its old binary until restarted. The hub sequences this with #752 and #737.

1. At a wide width, confirm the familiar single status row and full pane height. Narrow until groups occupy two rows; confirm the pane gives up exactly one additional row and the active chip remains visible.
2. Click a slot on each row, then open the switcher and resize in both directions. Check the selected child and menu end above the actual chrome, with no stale row or flicker. Repeat with Claude and Codex slots.
3. Narrow further until overflow appears. Active and attention chips take precedence over ordinary chips; any hidden attention is explicit in a whole +N ! marker. At an absurd width, the active fragment is still clickable and a marker is omitted if it cannot fit whole.
4. Check that cast/Chrome controls and the attention badge stay on the first row, while a transient notice uses free space on the final row. Open an existing codes, notes or attention popup and resize; its box and clicks follow the composed geometry.
5. If checking a disposable broadcast, confirm LIVE remains visible and private switcher contents stay hidden. Shrinking until LIVE cannot fit must stop frame admission through the existing visibility gate. Annotation controls are protected on the first chrome row, without unnecessarily blocking the second row.

Record the operator's result in the issue Log before landing. A finding becomes a recorded fixture/regression and is fixed on this branch before restaging.
