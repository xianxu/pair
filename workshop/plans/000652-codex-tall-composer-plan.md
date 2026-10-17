---
collaboration-mode: discussion
---

# Codex tall composer implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. This is one atomic review boundary; no milestone tags.

**Goal:** Confirm genuinely submitted tall Codex input during an existing turn, while keeping Return remapping conservative on historical text, menus and incomplete screens.

**Architecture:** Keep the existing short-composer evidence contract. Extend recognition beyond its 20-row local domain only when a complete visible viewport supplies both a qualified prompt/continuation block and the native trailing model footer below the cursor. Share the resulting geometry between Return remapping and peer text reading; preserve the existing delivery state machine and confirmation sequence checks.

**Tech stack:** Go, existing ultraviolet terminal snapshots, stateful harness fake, owned Codex CLI PTY.

## Scope and alternatives

Recommended: a geometry-qualified tall extension to the shared recognizer. A bare viewport-wide scan would recognize more distant historical prompts, and an arbitrary larger constant merely moves the failure threshold. A peer-only recognition bypass would leave plain Return shipping tall human drafts and duplicate authority policy. No Unknown-to-Occupied conversion, submission-by-write inference, retry or timeout change.

Codex batch registration/preparation remains disabled by #648. This issue qualifies its shared input-box evidence; enabling Codex hubs is a later rollout decision. A batch-shaped text fixture exercises the same tall geometry through allowed singleton delivery, without bypassing that policy.

The durable initial reproduction intentionally has no model footer. Preserve it as diagnosis of the original cutoff, but do not treat missing footer evidence as authority for a newly admitted tall surface. The accepted regression adds the complete native footer: it still fails on current main because its prompt is more than 20 rows above the cursor, and its post-submit clear is confirmed after the fix. Missing-footer tall frames remain Unknown and ordinary Return retains its existing unknown behavior.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Codex composer geometry (`codexComposerBounds`, name provisional) | `cmd/internal/wrapcmd/composer_recognizers.go` | new |
| Codex active-composer projection | `cmd/internal/wrapcmd/composer_recognizers.go` | modified |
| Codex footer evidence, shared from `peerCodexFooter` | `cmd/internal/wrapcmd/composer_recognizers.go` / `peer_composer.go` | modified |
| Qualified composer text | `cmd/internal/wrapcmd/peer_composer.go` | modified |

One snapshot has at most one current cursor-associated composer. Its nearest painted left edge must be a bold U+203A in column zero; intervening rows keep columns zero/one blank. Existing valid coordinates, visible cursor, text-column and trailing-status exclusion stay intact. For a prompt outside the original short domain, require Complete, a positive native footer below the cursor separated by a blank row, no intervening left-edge surface, and no painted row below that footer. Missing/clipped prompt, footer or partial frame declines. Reuse the existing model-footer signature rather than adding model/version allowlists. Return remapping and text extraction derive from one geometry helper (ARCH-DRY).

For short inputs, preserve existing recognized/ambiguous cases and extraction boundaries; no global tightening of legacy short screens. Extracting the common bounds must not broaden acceptance of menu rows or slash/shell drafts.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Return policy | `cmd/internal/wrapcmd/harness_tty.go`, `codex_return_test.go` | reused | operator key bytes and terminal snapshot |
| Peer input/confirmation | `cmd/internal/wrapcmd/peer_delivery.go`, new tall tests | reused | input writer, output sequence, turn state and clock |
| Native conformance | `cmd/internal/wrapcmd/peer_live_test.go`, Codex testdata | modified | new isolated native Codex PTY |

Existing stateful harness fake owns the terminal, repaint sequence, writer and clock. No new production state, resource owner, queue or external service. Never set private batch ownership to bypass #648's rollout refusal.

## Operating envelope and architecture

- **ARCH-PURE:** geometry consumes one immutable terminal snapshot and returns bounds/recognition; IO stays in existing input owner.
- **ARCH-PURPOSE:** exercise both peer confirmation and Return remapping, tall singleton and batch-shaped text, already-active and idle controls. Codex rollout remains separate.
- **ARCH-MOCK:** reuse terminal/harness fakes and actual writer path; qualify native rendering in a fresh owned PTY, never an existing live slot or clipboard.
- **ARCH-CONSTRAINTS:** keystroke/UI path; scan only the already validated viewport dimensions/cell storage. At most linear viewport work, no repeated suffix scans, scrollback reads, subprocesses or per-key IO. Test multiple heights around 19/20/21 and 26/32/60 rows, narrow/wide terminals, invalid coordinates and bounded large snapshots. Existing paste/submission deadlines remain unchanged.
- **ARCH-SECURE:** styled native output is untrusted evidence. A glyph or distant text alone grants no new authority. Require the complete tall geometry; preserve overlays, input ownership and command/shell draft guards. No credentials created. Opt-in native fixture copies only existing login into its private disposable home via the existing harness and removes it at cleanup.
- **ARCH-ORDER:** parser is stateless. Existing delivery ordering remains reserve -> full paste -> fresh qualified occupied -> submit -> fresh complete empty (or new turn if initially idle). Unknown/stale/partial observations do not advance confirmation. Operator input cancels owned input; incomplete writes and identity loss retain their current refusal/uncertainty paths. Use explicit repaint/clock events, not scheduling sleeps.
- **ARCH-FUNERAL:** no new runtime artifacts or growing state. Native PTY and disposable auth/home are owned by the existing capture fixture and collected on all exits; committed raw fixtures are bounded representative captures, not full logs.

## Tasks

### 1. Pin the recognition and Return contract

Files: `composer_recognizers_test.go`, `codex_return_test.go`, optionally a focused `codex_tall_composer_test.go`.

- [ ] Add a footer-complete tall native-shaped snapshot that is Unknown today, plus tall ordinary Return that wrongly emits CR today. Keep short, blank-continuation and existing mid-frame/status ambiguity controls.
- [ ] Add adversarial tall cases: historical prompt with indented output and no footer, nearer unrelated painted left edge, unbold menu marker, footer/status cursor, stale prompt below a replacement composer, hidden/out-of-range cursor, clipped prompt, missing or changed/unrecognized footer wording, unfinished synchronized frame, truncated cell storage, operator overlay. Independent expectation: none gains new newline/peer permission.
- [ ] Generate prompt/cursor distances and viewport sizes, with positive geometry and a single broken guard per case; add bounds-work protection against quadratic footer rescanning.
- [ ] Run the focused Codex/composer/Return tests and retain the expected new reds before implementation.

### 2. Share qualified geometry and prove confirmation

Files: `composer_recognizers.go`, `peer_composer.go`, `peer_delivery_test.go` or focused tall delivery tests; reuse `peer_integration_test.go` controls.

- [ ] Implement the smallest shared pure bounds helper and delegate active recognition/text extraction to it. Preserve the local short contract; require qualified complete footer evidence for the tall extension. Move/reuse the current footer signature, without a second copy.
- [ ] Stateful regression through actual `dispatchPeer`: 26-line singleton and five-report batch-shaped text, already-active turn, exactly one paste and one submit, then a fresh complete empty composer => Submitted. Short and initially-idle controls remain Submitted.
- [ ] Negative sequences: no occupied repaint, stale empty, open frame, partial paste/submit writer, intervening operator input, picker and binding loss; no false Submitted and no replay. Extend/reuse existing stateful tests rather than asserting callback counts.
- [ ] Mutate away the extension to show tall positives fail, and remove a tall safety guard to show its adversarial case fails; restore before green. Run focused and race checks, then all `wrapcmd` tests once.

### 3. Native qualification, docs and normal landing

Files: existing native peer capture seam/testdata, `atlas/couch.md` and index if needed, README peer-message evidence notes, issue Log, lessons for review findings.

- [ ] Add an opt-in tall singleton/body scenario using a new owned Codex PTY (100x60 representative viewport), positive footer, at least 26 visible composer rows, and batch-shaped Unicode reports. Record version, capture hash and exact evidence; do not infer byte retention from a collapsed marker. Never enable the Codex batch protocol as a test shortcut.
- [ ] Validate normal Return adds a newline and explicit submit remains distinct in the owned PTY. Any actual native submission uses a harmless explicitly approved prompt, never tools or an existing conversation.
- [x] Hub approved this item's live arrangement: owned-PTY Return/harmless submit now, operator Return/submit smoke after its next restart, no :0 staging. Ask the hub for the shared native lane when the capture is ready; pair:1 and pair:5 also use it. Never steal focus.
- [ ] Document safe tall evidence and remaining clipped/unknown behavior; retain #648's Claude-only rollout. Push, record the required full suite, close through the mandatory fresh-context review, then enter `sdlc merge --yes`; report verified landing to hub.

## Verification commands

- Focused: `go test ./cmd/internal/wrapcmd -run 'Test.*(Codex|Composer|Peer.*Tall|PeerIntegration)' -count=1`.
- Race: same selected tests with `-race`.
- Affected package: `go test ./cmd/internal/wrapcmd -count=1`.
- Native: existing `TestPeerLiveConformance` opt-in flags/environment, new tall scenario; record the exact command after the harness addition is settled.
- Recorded suite: `sdlc local-check run --name suite` on the pushed code head, then normal close/merge gates. Follow the issue-backed flake rule, retain every failed attempt; no blind retry or main chasing after a pass.

## Revisions

- 2026-10-17 15:38 PDT — Hub approval `8acc22f6-c522-4333-bd5e-250e5ddb66c4` adds an explicit changed/unrecognized footer-wording negative. Both design and live arrangement approved; native work must first obtain the shared lane, with no focus stealing. Real operator Return/submit smoke is deferred to its next restart. No production code changed.
