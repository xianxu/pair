# How to Bring Up a New Harness CLI in pair

`pair` is an agent-agnostic, Neovim-backed launcher environment. While the horizontal two-pane design is generic, delivering a premium, seamless pair-programming experience requires integrating the agent across seven critical integration surfaces.

This guide outlines how to bring up a new agent harness CLI (e.g., `muse`) and achieve parity with existing agents (`claude`, `codex`, `agy`, `muse`).

---

## 1. Key Integration Aspects

### Aspect 1: Return Key Remapping
By default, the bottom Neovim draft pane maps **Enter** to insert a newline, and **Alt+Enter** to send the buffer. To provide visual and interactive consistency, the top agent pane (which runs inside the transparent PTY proxy `pair-wrap`) should map keys similarly:
- **Plain Enter** inside a positively detected composer/input box should insert a newline (preventing accidental premature sends).
- **Alt+Enter** should submit the input.

**Implementation:**
- **File:** [cmd/internal/wrapcmd/harness_tty.go](file:///Users/xianxu/workspace/pair/cmd/internal/wrapcmd/harness_tty.go)
- Add the agent's `keymap` to its `harnessTTYProfiles` entry, defining `plainCR` and `altCR`:
  ```go
  "agy": {
      keymap: sendKeymap{
          plainCR: []byte{'\n'}, // plain Enter inserts newline
          altCR:   []byte{'\r'}, // Alt+Enter sends query
          altBS:   []byte{0x15},  // Alt+Backspace kills to line start
      },
      // ... overlay, composerGate, recognize
  },
  ```
- **Note:** Claude uses `\<Enter>` (`[]byte{'\\', '\r'}`) as a newline, while Codex, Antigravity (`agy`), and Muse (`muse`) use LF (`\n`) for newline and CR (`\r`) for send.
- **Composer detection:** Do not rewrite plain Enter merely because no menu was detected. New integrations should positively detect the agent's composer/input box from stable raw terminal signals (cursor position/visibility, prompt/composer chrome, or agent OSC). Prefer the agent's native composer-availability signal; do not copy another agent's terminal heuristic unless captured logs prove the same signal is stable. If the composer is unknown or inactive, plain Enter should pass through as the agent's normal Enter key.

**Telemetry Signal** (aspect `1`, see §3): `return-remap` — `fired` each time a plain Enter is remapped to the agent's newline; `bypass` each time it passes through as a bare `\r` while an overlay is active or the composer is not positively detected. Emitted from `emitPlainCR`. The `fired:bypass` ratio is the health signal; an all-`bypass` or zero-`fired` session means the remap stopped engaging or composer detection drifted.

---

### Aspect 2: Overlay-Aware Return Suspension
If the agent presents blocking overlays, pickers (like file autocompletes), yes/no confirmation modals, **or user selection menus (AskUserQuestion / request_user_input pickers)** , text-area Enter remapping will break the interaction. Every option-selection UI needs the same bypass — plain Enter must confirm the highlighted option, not insert a newline (e.g. muse bug: selection menu required Alt+Enter to confirm because no marker matched). Inside an overlay, a plain **Enter** must send a bare carriage return (`\r`) to select/confirm.

`pair-wrap` should treat overlay detection as an override on top of positive composer detection. The normal rewrite path is "composer active → newline"; overlay detectors arm a temporary `pickerActive` flag so the next plain Enter is bypass-translated to a bare `\r`, and the flag is immediately cleared.

**Implementation:**
- **File:** [cmd/internal/wrapcmd/wrap.go](file:///Users/xianxu/workspace/pair/cmd/internal/wrapcmd/wrap.go)
- Register the harness in `harnessTTYProfiles` ([cmd/internal/wrapcmd/harness_tty.go](file:///Users/xianxu/workspace/pair/cmd/internal/wrapcmd/harness_tty.go)) — one registry owns the keymap, the overlay detector, the gate policy, and the composer recognizer:
  ```go
  "muse": {
      keymap:       sendKeymap{plainCR: []byte{'\n'}, altCR: []byte{'\r'}, altBS: []byte{0x15}},
      overlay:      detectMuseOverlayOpen,
      composerGate: composerGatePositive,
      recognize:    museComposerActive,
  },
  ```
  Leave `composerGate` unset and the harness fails closed: every plain Return passes through as bare CR. That is the correct starting point for a new harness — opt into `composerGatePositive` only once you have a recognizer backed by captured bytes.
- Implement the detector. Detectors can scan the rolling output stream for custom OSC escape sequences (e.g. Claude's permission OSC `OSC 777;notify;...`, or Codex's `OSC 9;Plan mode prompt:...`) or fallback to visible text substring matches (e.g., watching for `"Press enter to confirm"`).
- **Composer recognition is a pure function over a screen snapshot.** A shared x/vt terminal model publishes immutable `terminalSnapshot` values; recognizers live in `composer_recognizers.go` and never retain their own screen state. **For `codex`:** a **bold** U+203A at column 0, reachable from the cursor through painted continuation rows — Codex reuses the same glyph unemphasized as a menu selection marker and parks the cursor on its status line mid-paint, so neither qualifies. **For `claude`:** a non-rule glyph at column 0 between two `─` rules sharing one foreground. Deliberately glyph- and colour-agnostic: Claude repaints both per input mode (bash mode is `!` with pink rules where the default is `❯` with grey), and since Claude is Pair's default agent a false negative submits a half-written draft — so the recognizer is biased against declining. Claude and Muse share `ruledBoxComposerActive`; add a spec rather than a fourth near-copy if your harness paints the same shape. **For `muse`:** a non-faint `⟩` at column 0 as the first row inside a pair of faint `─` rules, with the cursor anywhere in the box — it anchors on the *enclosing* rules, so the composer may grow past one line. **For `agy`:** the cursor inside a coherent box — a `>` prompt column enclosed by two `─` border rows that both span the cursor column. Do not copy another harness's signal without capturing bytes that prove it: Codex's old `48;2;57;57;57` background gate silently stopped matching when Codex dropped that paint, and plain Enter could no longer insert a newline until live conformance caught it.
- **For `agy`:** Antigravity *does* render its permission picker in the PTY ("Do you want to proceed?", "Yes, and always allow", …), so `detectAgyOverlayOpen` matches those visible-text markers (no OSC) to arm `pickerActive` — without it, the remapped Enter can't confirm the picker and a stray newline leaks into the prompt (#000042).
- **For `muse`:** Muse renders both tool-permission pickers ("Permissions required", "Allow execution", …) **and** user selection menus (AskUserQuestion via `request_user_input` — "Select an option", "Use arrow keys", "Press Enter to select", …). Both families must be in `musePickerMarkers`; a missing selection marker reproduces as "Enter inserts newline, Alt+Enter required to select".

**Capture evidence before you define a recognizer.** Record literal bytes from the real CLI through the bounded PTY seam — never hand-author them — into `cmd/internal/wrapcmd/testdata/tty/<agent>/<version>/`, alongside a `metadata.json` carrying the exact `--version` string, argv, RFC3339 capture time, and a SHA-256 per raw file:

```bash
PAIR_LIVE_HARNESS=<agent> \
PAIR_LIVE_CAPTURE_OUT=cmd/internal/wrapcmd/testdata/tty/<agent>/<version>/composer.raw \
  go test ./cmd/internal/wrapcmd -run TestHarnessTTYLiveConformance -count=1 -v
```

`composer.raw` is required and must remap to LF; add an `overlay.raw` for any blocking dialog that must stay bare CR, captured by `TestHarnessTTYLiveDrivenConformance`, which drives the harness one keystroke past startup (`PAIR_LIVE_SCENARIO` picks a single scenario). A harness whose declining evidence does not separate a composer from a same-shaped picker must say so in `ttyFixtureDiscriminationGaps`. `TestHarnessTTYFixtureConformance` then replays every fixture through the production proxy at *every* split point and requires the Return decision to be identical at all of them, so PTY chunk boundaries can never change behavior. The live tests are opt-in and assert behavior rather than byte identity: real output embeds per-account and per-moment content, and harnesses self-update. When one drifts it prints the exact recapture destination.

**Telemetry Signal** (aspect `2`, see §3): `overlay-detect` — `fired` when a registered marker arms `pickerActive` (the detail carries the matched marker); **`near-miss`** when the output looks like a confirm/permission prompt (`promptShape` heuristic in `checkOverlayOpen`) but *no* registered marker matched. A `near-miss` is the drift fingerprint: the harness renamed its picker wording, the detector went silent, and the next plain Enter will leak a newline (#000042). The `detail` field carries the unrecognized line verbatim — that's the new string to add to `codexPickerMarkers`/`agyPickerMarkers`/`musePickerMarkers` (or the OSC body for claude).

---

### Aspect 3: Session ID Watcher & Recovery
`pair` features restart-in-place (`Alt+n`) and session reattach (`pair resume <tag>`). Recovery identity is established only after one completed native causal round.

**Discovery & Watcher:**
- **Files:** native shape/event parsing lives in `cmd/internal/sessioninventory`; generation monitoring and persistence live in `cmd/internal/sessionwatch`.
- Add one versioned facts-only scanner and sanitized fixtures. It must enumerate roots and descendants, validate native parent edges, and emit allowlisted operator/progress events. Unknown shapes become coded diagnostics.
- Whole-workbench launch and agent-only restart synchronously append a provisional launch baseline before input, then pass its physical ordinal to `pair session-watch`.
- The watcher uses the shared scanner and exact Pair-log matcher. Process/open-file snapshots corroborate a candidate but never select one; only a unique completed round persists a binding and refreshes config.
- Add the agent to `ScannerForAgent`, `SupportsAgent`, conformance, `QuerySession`, activity, and `pair session-inventory` tests. A pre-round quit must remain provisional; repeated rounds must remain ambiguous. Consumers must read only the established root projection, never reintroduce a native path formula.

**Recovery Flags:**
- **File:** `cmd/internal/launcher/agentargs.go`
- Add the agent-specific binding to `resumeToken`, place it through
  `composeResumeArgs`, and extend the table tests. Codex and Muse use a leading
  `resume <id>` subcommand; Claude uses `--resume <id>`, and Agy uses
  `--conversation <id>`.
- Extend `OSRuntime.AgentSessionExists` with the agent's native artifact and add
  a focused launcher test for both present and absent sessions.

**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair session-watch` — `fired` after a durable binding append, **`near-miss`** when PID identity changes or native records cannot form an allowlisted round, and `fail` when no completed round appears in the startup window. The ledger remains authoritative if config refresh fails.

---

### Aspect 4: pair-slug Generation
The `pair-slug` script summarizes what the current agent session is about to display in the Zellij list.

**Implementation:**
- **Transcript Parsing:** Register a parser in [cmd/internal/slugcmd/slug.go](file:///Users/xianxu/workspace/pair/cmd/internal/slugcmd/slug.go) under `parseTranscript()`. For JSONL transcripts like `agy`, extract the `content` where `type: "USER_INPUT"`; for `muse`, extract `payload.event.prompt` where `payload_type=="runtime.session"` and `payload.kind=="run"` / `event.kind=="started"`.
- **Model Sandbox Execution:** Ensure that invoking the agent in summarize mode (`agy -p "<prompt>"` / `muse exec "<prompt>"`) runs inside a clean sandbox (e.g. setting `cmd.Dir = os.TempDir()` in [cmd/internal/model/model.go](file:///Users/xianxu/workspace/pair/cmd/internal/model/model.go), the shared model runner). This prevents the agent from triggering expensive workspace exploration tools, speeding up slug generation from 20s to 1s.

**Telemetry Signal** (aspect `4`, see §3): `slug-parse` from `pair-slug` — `fired` when the transcript parses into ≥1 turn, **`near-miss`** when a transcript is read but yields 0 turns (the transcript schema changed and the parser no longer extracts anything), `fail` when no transcript resolves at all. A run of `near-miss` lines points straight at a `parseTranscript` parser that needs updating.

---

### Aspect 5: Mouse Scroll & PTY Output Filtering
Some agents emit DEC synchronized-output markers or other terminal control characters that interfere with Zellij's mouse scrollback.
- **PTY Filter:** If an agent behaves poorly with mouse scrolling, `pair-wrap` can intercept and strip specific sequences (e.g., Codex's `ESC[?2026h` synchronized-output toggles) in `stdoutChunk()` before queueing filtered visible stdout for batched delivery to Zellij. Raw scrollback capture remains immediate and unfiltered.

**Telemetry Signal** (aspect `5`, see §3): `output-filter` from `pair-wrap` (`stripCodexSyncOutput`) — `fired` once per distinct marker stripped per session (deduped; the markers repeat many times per render, so presence is the signal). If a codex update renames a sequence, its `fired` line stops appearing — an *absence* the operator reads against the expected marker set.

---

### Aspect 6: Agent Settings Configuration
To minimize confirmation prompt fatigue and allow the agent to run commands, create/modify the agent's permission profiles (e.g., `.claude/settings.json` or `.antigravitycli/settings.json`) to white-list common utility commands (like `git`, `make`, `sdlc`, `lsof`, `zellij`) and mount trusted directories. 

Align local settings in workspace directories with parent configurations (e.g. `../ariadne/`) to support continuous testing and seamless automation.

**Telemetry Signal:** none. This aspect is *static config*, not a runtime mechanism — there is no per-run trigger to emit, so it has no flight-recorder signal. Drift here surfaces as confirmation-prompt fatigue, not a missing signal.

---

### Aspect 7: Human Prompt Search (Alt+b)
The scrollback viewer (`Alt+/`) maps **Alt+b** (and **Alt+Shift+B**) to jump between user turns. To do this, Neovim needs to know what unique leading glyph or marker the agent uses to format the user's prompt input line in the console.

**Implementation:**
- **File:** [nvim/scrollback.lua](file:///Users/xianxu/workspace/pair/nvim/scrollback.lua)
- Register the prompt regex in `PROMPT_PATTERN_BY_AGENT`:
  ```lua
  local PROMPT_PATTERN_BY_AGENT = {
    claude = [[^❯]],
    codex  = [[^›]],
    agy    = [[\(──.*\n\)\zs>]],
    muse   = [[^>]],
  }
  ```

**Telemetry Signal** (aspect `7`, see §3): `prompt-search` from `nvim/scrollback.lua` (`jump_to_prompt`, via `nvim/adapt.lua`) — `fired` on a successful Alt+b jump; **`near-miss`** (deduped per viewer) when the pattern matches *nowhere* in a non-empty scrollback, which means the agent's prompt glyph changed and Alt+b can never land. (A miss in only one direction is ordinary end-of-scrollback and is *not* logged.)

---

## 2. Checklist for Bringing Up a New Agent

When introducing a new agent `<name>`, ensure you complete each item:

1. [ ] **Verify Return Key remapping** on the harness profile in `harnessTTYProfiles` (Enter = newline, Alt+Enter = send), and pin it with a captured fixture under `cmd/internal/wrapcmd/testdata/tty/`.
2. [ ] **Check for blocking TUI overlays** (permission pickers **and** user selection / AskUserQuestion menus) and implement a PTY overlay detector and register it on the harness profile in `harnessTTYProfiles` if needed — verify plain Enter confirms the picker and Alt+Enter is not required.
3. [ ] **Implement Session Inventory + Watching** with a versioned scanner/event adapter, conformance fixture, provisional launch baseline, and completed-round watcher; use open files only as corroboration.
4. [ ] **Configure Launcher Recovery** in `cmd/internal/launcher`: extend `resumeToken` and `composeResumeArgs`, then prove `OSRuntime.AgentSessionExists` and `EstablishedSessionID` consume scanner inventory rather than native paths or config identity.
5. [ ] **Add slug generation support** in `pair-slug` (transcript parsing + sandboxed print execution).
6. [ ] **Confirm mouse scroll and scrollback render** work smoothly without drawing glitch issues.
7. [ ] **White-list permissions** in the agent's global or workspace settings directory.
8. [ ] **Register the user-prompt glyph** in `nvim/scrollback.lua` for `Alt+b` jumping.

---

## 3. Drift Telemetry

Harnesses update constantly and break the adaptations above *silently* — a renamed
picker string or a changed transcript shape doesn't error, the adaptation just
stops firing. Unit tests can't catch this: they validate our matchers against
strings we froze, so they pass forever even after the live harness moves.

The **adaptation flight recorder** makes drift observable. Every adaptation appends
one JSON line per trigger to the launcher's exact `$PAIR_ADAPT_LOG_PATH` during normal use.
The native launcher's create flow truncates the file once at session launch; all components then append
(`O_APPEND`, atomic per-line across processes). A user runs `pair` normally; when
something feels off they run **`doctor/doctor.sh`** (see [`doctor/README.md`](file:///Users/xianxu/workspace/pair/doctor/README.md)),
which reads the trace and points at the broken aspect — no need to describe the
symptom. The same procedure is packaged as the `doctor/SKILL.md` skill, so an
agent can run and interpret it on demand.

**Line schema** (flat — `detail` is a single capped string so shell/Lua emitters
stay one-liners):

```json
{"ts":"...","comp":"pair-wrap","agent":"codex","aspect":2,"signal":"overlay-detect","outcome":"near-miss","detail":"unmatched prompt-shaped output: Do you want to apply this patch? (y/n)"}
```

`outcome` ∈ `fired` (matched + acted) · `bypass` (deliberately stepped aside) ·
`near-miss` (**the drift fingerprint** — the harness did something we half-recognized
but no matcher caught it) · `fail` (expected to work, couldn't).

**The key idea is logging near-misses, not just successes.** A success-only log
can't detect drift because breakage manifests as *absence* of a signal — invisible.
A near-miss records the unrecognized string verbatim in `detail`, which is exactly
what you paste into the relevant matcher to fix the drift.

**Signal registry** — when adding an aspect, add its row here and emit the signal
from the owning component (the Go binaries use `cmd/internal/adapt`; shell and Lua
write the same line shape directly):

| Aspect | Signal | Component | Outcomes | Drift looks like |
|---|---|---|---|---|
| 1 Return remap | `return-remap` | pair-wrap | fired, bypass | zero `fired` / all `bypass` |
| 2 Overlay suspend | `overlay-detect` | pair-wrap | fired, near-miss | any `near-miss` |
| 3 Session watch | `session-id` | pair session-watch | fired, near-miss, fail | `fail` (timeout) / `near-miss` (file found, id unparsed) |
| 4 Slug gen | `slug-parse` | pair-slug | fired, near-miss, fail | `near-miss` (transcript parsed, 0 turns) / `fail` (resolved a transcript but couldn't read/parse it) |
| 5 PTY filter | `output-filter` | pair-wrap | fired | a `fired` line that *stops* appearing (its absence is the signal — the sequence was renamed) |
| 6 Settings | — | — | — | static config; no signal |
| 7 Prompt search | `prompt-search` | nvim/scrollback.lua | fired, near-miss | `near-miss` (0 matches in non-empty scrollback) |

> Status: all six runtime aspects emit today (#000045 M1: aspects 1 & 2; M2: aspects 3, 4, 5, 7) — verified for `claude`/`codex`/`agy`/`muse` via #000134.

**Privacy:** `detail` can carry a snippet of agent output (e.g. an unrecognized
prompt). It is capped at 200 bytes and the file stays local under `$PAIR_DATA_DIR`,
the same trust level as the existing scrollback logs. Native launcher lifecycle cleanup removes it on quit.
