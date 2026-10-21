---
collaboration-mode: discussion
---

# Two-line Couch tab bar implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Use bounded independent tasks where the context is capturable; preserve one owner for Console publication. The SDLC close owns the single fresh-context boundary review.

**Goal:** Show overflowing Couch tabs on two lines, keep the active tab reachable, and report any remaining hidden attention without corrupting pane, broadcast or input geometry.

**Architecture:** A pure bounded planner produces one immutable status-bar layout. The compositor derives a trusted chrome height from the rectangular bottom surface; Console publishes that surface, child geometry and hit extents under its existing publication lock. Broadcast visibility, privacy and annotation guards consume the composed frame's geometry rather than guessing the last row.

**Tech stack:** Existing Go terminal compositor, Couch fake-host/native PTY fixtures, broadcast server and vanilla JavaScript viewer. No dependency, timer, polling loop or background worker added.

## Authority and scope

Operator Spec is in pair#779. TL c5c04f63 approves full flow and these choices: attention badge stays right of line1; transient notices occupy free space on the final line; active remains visible as a full-label chip alone if needed, clipped but clickable at absurd widths. Attention chips take priority over ordinary chips; hidden attention is explicit in the overflow marker. This plan still needs TL approval before change-code.

Includes the manual #757 archive repair, ariadne#532 class: one separately named commit, two100%-identical renames into history beside its archived parent plan. TL c1d5a9d6 approves it riding with #779. Name it explicitly in the PR description. Do not reopen/re-close #757 or retry its old archive proof.

## Alternatives

1. **Shared layout plus composed-frame geometry (chosen).** All consumers derive the same reservation, including broadcasts and local clicks. More than a one-file render edit, but it preserves the existing ownership boundary.
2. Wrap an already-styled string. It loses group/active decisions and cannot produce reliable row-aware hit spans; it also leaves the child the wrong size.
3. Search the last two rows for red LIVE text or keep a separate broadcast offset. Child content can imitate that text, and independent offsets can disagree with accepted publication. Rejected: use compositor-owned metadata.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| statusPlan / planStatusRows | cmd/internal/couchtty/status_layout.go | new |
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
| Presenter Select/ResizeLayout/UpdateChrome | cmd/internal/terminal/presenter.go | modified | one serialized physical parent writer and child resize callback |
| Broadcast admission and live frame delivery | cmd/internal/broadcast/hub.go, session.go, stream.go, server.go | modified | trusted current publication and SSE/browser frame delivery |
| Viewer annotation geometry | cmd/internal/broadcast/web/notes.js and frame consumer | modified | server-owned frame geometry, never client authority |

Use existing stateful fake host, child endpoint and actual loopback broadcast server seams. Native conformance uses owned PTY children and joins every process; operator live look uses the approved staging procedure. No function-call-only replacement for terminal publication or broadcast interaction (ARCH-MOCK).

## Packing policy

1. Compute sanitized labels, full/short chip widths, fixed group boundaries and status prefix budget once. Existing group identity (including numbered hubs) remains authoritative. Preserve existing exact bytes when all actor chips fit on one line; a long transient notice alone never creates line2.
2. Pack complete groups greedily. Move a group intact to line2 when it fits a full line but not the first remainder. Split a group into chip units only if that original group exceeds a full line. A continuation begins with the repository's full label; later members may use :N. No separator starts a row.
3. If the unfolded groups fit in two lines, render all of them with no +N. Keep cast/Chrome/capture controls left of line1 and attention at its right edge. Notices use only free space on the final line.
4. Beyond two-line capacity, select active first, then attention (hub, blocked, bell), then ordinary actors. Among retained actors preserve original display order. No ordinary actor survives while an eligible attention actor is hidden. Overflow may show a full-label active chip separately when needed, as approved.
5. Draw one whole trailing +N on line2, counting every omitted actor, including placeholders. Draw +N ! if any omitted actor needs attention. Do not claim a count for merely shortened labels. If the terminal is too narrow for both active and the whole marker, active wins and the marker is omitted; never draw a bare +. At such widths visibility means a clipped positive-width clickable active span, not impossible complete text.
6. When terminal height permits only one chrome row, use the same bounded priority fallback on that row. With only one total host row, preserve today's no-reservation fallback so the child remains at least one row. At normal heights the bar is strictly one or two rows.

Overflow selection uses measured full-label chip widths so adding retained actors cannot reduce a candidate's cost through changed abbreviations. Group oversized classification is fixed from the original group, not recomputed per candidate. Precompute each actor's priority rank; search the largest retained priority prefix with allocation-free O(A) candidate packing, reserving a bounded worst-case whole overflow suffix. Render retained actors in source order once. Prefer a small conservative unused suffix over an unbounded search for a maximal packing; normal one/two-line cases use exact widths and no conservative reservation. This gives O(B + A log A), not repeated layoutChips/keptEnd prefix scans. The overflow fallback never repeats the old quadratic fitChips strategy.

## Shared geometry and publication

- Terminal composition accepts a rectangular bottom surface of zero or more complete rows, strictly fewer than host rows. A shared pure helper owns the arithmetic/validation used by Compose, Presenter selection, resize and input child bounds; no hardcoded subtract-one copy.
- Terminal Frame validation and history equality/reset comparisons include ChromeRows. A metadata-only change still reaches taps/broadcast even when the physical renderer produces no byte diff.
- Console builds a proposed immutable layout from one status-model snapshot under publishMu. Unchanged height uses UpdateChrome; a height transition uses existing ResizeLayout with the selected child's ResizePTY callback. Commit hit extents/reservation only after the publication succeeds. Resize remaining children to that accepted size; InputEnded children retain existing skip behavior.
- Startup/installation, actor selection, actor/status changes, menu presentation and host resize all use this same layout owner. Selection first resizes the candidate child when its geometry differs; it cannot Select a stale-size frame. Startup prediction is reconciled before first presentation. Background children resize through the existing owner; a future selection revalidates size.
- Switcher height, menu extents and footer use hostRows minus accepted chrome height. Resize rewraps and paints through one synchronized-output frame, never a separate erase/new-row paint. Existing routing-answer handling and publication fencing remain intact.
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
| Actor attention changes wrapping | one accepted new layout and hit map | click/focus/resize | child's resize/reflow only |
| Physical/child resize fails | existing terminal failure path; no accepted mismatched hit map | existing restart/recovery | existing in-flight terminal work limits |
| Private switcher broadcast | placeholder body, both public chrome rows retained | close switcher | private menu content is withheld |
| Frame metadata changes without ANSI delta | browser receives current guard geometry | retry navigation/place note | stale annotation attempt may be refused |

Covers one/two/tiny-height and active/attention overflow, both input paths, privacy and resize. Does not add scrolling tabs, a third row, persistence, a new service, changed agent lifetime, or guaranteed visibility of every attention chip when they physically exceed capacity; hidden attention remains explicit.

## Bounds, lifecycle and architecture

Growth drivers are actor count A, sanitized label/glyph bytes B and host cells C×R. Measure/sanitize once O(B); packing O(A log A), materialization O(A), at most2×C chrome cells. Existing compositor remains O(C×R) worst-case and owns its bounded frame queue. Count planner candidate/actor visits at N=64/128 and assert B + constant*A*(bits.Len(A)+1), with all-kept, many-group and oversized-group inputs; also assert bytes scanned do not multiply by search passes. No timing-only oracle (ARCH-CONSTRAINTS).

Creates no durable runtime artifact, cache, row history or background worker. Layouts/metadata die with the existing frame/Console lifetime. The only durable change outside normal code/process docs is the explicitly approved #757 evidence rename; its existing history lifetime is unchanged (ARCH-FUNERAL).

ARCH-DRY/PURE: one planner and one trusted geometry owner; existing styling and compositor reused. ARCH-PURPOSE: panes, input, broadcasts and notes consume the wrapped geometry. ARCH-MOCK: stateful host/endpoint/server tests plus native conformance. ARCH-SECURE: no child-text-derived broadcast authority; private body stays private. ARCH-ORDER: publication lock plus Presenter FIFO joins geometry and drawing. ARCH-EVOLVE: additive live message only; no persisted writer introduced.

## Verification by function/risk

| Owner | Adversarial family | Mechanical pin |
|---|---|---|
| planStatusRows / row drawer | fits1, groups2, oversized group, >2 capacity, active at tail, many attention chips, Unicode/control bytes, widths0/1/2 and normal widths | exact rows/counts/extents, attention-before-ordinary invariant, each row <=width, input model unchanged; seed fuzzing |
| planner cost | 64/128 actors, all attention, many groups and long labels | counted visits/bytes and logarithmic bound, no repeated sanitation |
| Compose / child geometry | no/one/two chrome rows, partial rows, all-host chrome, already-composed child | pure Frame validation/composition assertions, no overlap, unchanged one-row output |
| Presenter ResizeLayout / input | one↔two while drag or child output is active | stateful endpoint/writer barriers, one coherent frame, release before changed input destination |
| Console publication | startup, attach, select, attention changes, open switcher and repeated narrow↔wide resize | existing fake host/real child fixture; PTY geometry, footer and committed hit spans agree, no intermediate empty frame |
| local/remote mouse and AnchorFor | identical columns in different rows, blank chrome, control clicks on row0 | exact intended actor/control and no child-delivered chrome click; anchors on both rows |
| Indicator/ViewerFrame/guards | child imitates LIVE, two-row private frame, marks/notes on either control or ordinary cells | reject spoofed authority; preserve both chrome rows; real controls unpainted and private body absent |
| Stream/Join/browser notes | metadata-only height change, late join, current layout after resize, stale client fields | actual server message and JS guard assertions, server refuses client geometry authority |
| native and live setup | narrow two-row terminal then wide one-row, real TUI scroll/cursor/input | bounded owned native PTY test; stage selected head/build and operator look before landing |

## Implementation tasks — one atomic close boundary

- [ ] Pin RED pure packing/composition/guard regressions and record observed failures before implementation.
- [ ] Implement pure measured planner and shared row rendering with counted complexity pins; run targeted planner/fuzz tests and commit.
- [ ] Generalize compositor geometry/metadata and Presenter resizing; test one-row compatibility and one↔two transitions, then commit.
- [ ] Propagate trusted chrome geometry through broadcast privacy, marks/notes and additive viewer frames; run Go/JS tests and actual-server conformance, then commit.
- [ ] Integrate Console publication, child/menu sizing, row-aware local/remote input and anchors; run fake-host and native lifecycle tests, then commit.
- [ ] Update atlas/couch.md, atlas/terminal.md/broadcast map as needed and index links; audit artifact/source-reader classifications for new files, run inventory and focused race/repolint checks.
- [ ] Stage the concrete built head for the operator via hub, record the live result; checkpoint/push and run recorded affected (or suite fallback), close once through SDLC, then merge through its retained runner and report landing.

Targeted commands: go test ./cmd/internal/couchtty -run 'Test.*Status|Test.*Wrap|Test.*Chrome|Test.*Anchor' -count=1; go test ./cmd/internal/terminal ./cmd/internal/broadcast -count=1; scoped race runs for changed Console/Presenter/guard cases; existing Node viewer tests and make test-affected-inventory. Native test names are added beside the exact scenario, not a second full-suite loop. All source-reading tests receive per-reader audit/certificates after git add. Final recorded checks and CI remain mandatory.
