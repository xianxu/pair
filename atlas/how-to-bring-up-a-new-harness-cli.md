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
- **File:** [cmd/internal/wrapcmd/wrap.go](file:///Users/xianxu/workspace/pair/cmd/internal/wrapcmd/wrap.go)
- Add the agent to `sendKeymapByAgent` defining `plainCR` and `altCR`:
  ```go
  var sendKeymapByAgent = map[string]sendKeymap{
      "agy": {
          plainCR: []byte{'\n'}, // plain Enter inserts newline
          altCR:   []byte{'\r'}, // Alt+Enter sends query
      },
  }
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
- Register the detector in `overlayDetectorByAgent`:
  ```go
  var overlayDetectorByAgent = map[string]overlayDetector{
      "claude": detectClaudeOverlayOpen,
      "codex":  detectCodexOverlayOpen,
      "agy":    detectAgyOverlayOpen,
  }
  ```
- Implement the detector. Detectors can scan the rolling output stream for custom OSC escape sequences (e.g. Claude's permission OSC `OSC 777;notify;...`, or Codex's `OSC 9;Plan mode prompt:...`) or fallback to visible text substring matches (e.g., watching for `"Press enter to confirm"`).
- **For `codex`:** Codex plain Enter rewrites only when `codexComposerTracker` positively detects the active composer: visible cursor on or next to rows painted with the `48;2;57;57;57` background. The detector deliberately avoids Pair outer-screen bottom geometry because Codex can paint an editable composer on a smaller logical screen. Codex also uses OSC 9 plan/question bodies and visible-text picker footers; keep `codexPickerMarkers` current as override/fallback signals for menus, including variants like `"Press enter to confirm or esc to go back"` and `"Press enter to confirm or esc to cancel"`.
- **For `muse`:** Muse plain Enter rewrites only when `museComposerTracker` positively detects the active composer: visible cursor on or next to a row where the prompt glyph `›` (`e2 9f a9`, FG `38;2;90;160;255`) was painted (captured from `scrollback-fix-tty-muse.raw` at `30;1H` empty and `9;1H` filled). The logic is prompt-anchored (≥1 prompt row within `cursor±1` + visible cursor), not BG — do not reuse Codex's `48;2;57;57;57` background unless logs prove stability. Like Codex, the tracker falls back to bare CR when composer is hidden/unknown, so future Muse menus that lack a `musePickerMarkers` entry still select on plain Enter.
- **For `agy`:** Antigravity *does* render its permission picker in the PTY ("Do you want to proceed?", "Yes, and always allow", …), so `detectAgyOverlayOpen` matches those visible-text markers (no OSC) to arm `pickerActive` — without it, the remapped Enter can't confirm the picker and a stray newline leaks into the prompt (#000042).
- **For `muse`:** Muse renders both tool-permission pickers ("Permissions required", "Allow execution", …) **and** user selection menus (AskUserQuestion via `request_user_input` — "Select an option", "Use arrow keys", "Press Enter to select", …). Both families must be in `musePickerMarkers`; a missing selection marker reproduces as "Enter inserts newline, Alt+Enter required to select".

**Telemetry Signal** (aspect `2`, see §3): `overlay-detect` — `fired` when a registered marker arms `pickerActive` (the detail carries the matched marker); **`near-miss`** when the output looks like a confirm/permission prompt (`promptShape` heuristic in `checkOverlayOpen`) but *no* registered marker matched. A `near-miss` is the drift fingerprint: the harness renamed its picker wording, the detector went silent, and the next plain Enter will leak a newline (#000042). The `detail` field carries the unrecognized line verbatim — that's the new string to add to `codexPickerMarkers`/`agyPickerMarkers`/`musePickerMarkers` (or the OSC body for claude).

---

### Aspect 3: Session ID Watcher & Recovery
`pair` features a robust restart-in-place (`Alt+n`) and session reattach (`pair resume <tag>`) mechanism. To make this work, the launcher needs to discover the agent's unique conversation/session ID as soon as it is spawned.

**Discovery & Watcher:**
- **Files:** `cmd/pair-session-watch` and `cmd/internal/sessionwatch` (the launcher spawns the Go binary directly since #94 M2 — the `.sh` shim was retired).
- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
- Configure the agent's session file criteria in `cmd/internal/sessionwatch.SpecForAgent`, then teach `AgentSpec.Match` how to recognize that agent's file shape and return a `SessionID`.
- For example, agy watches `~/.gemini/antigravity-cli/conversations` and extracts the UUID from `<uuid>.db`; codex watches `~/.codex/sessions`, extracts a candidate UUID from `rollout-*.jsonl`, then authorizes it only when the first event is matching root `session_meta`; muse watches `~/.local/share/muse/sessions` and extracts the UUID from the parent dir of `session.jsonl` (`YYYY/MM/DD/<uuid>/session.jsonl`) — excluding `…/<uuid>/subagent/<sub-uuid>/session.jsonl` (only the root session is resumable via `muse resume <id>`).
- When captured, the watcher writes `{ "agent": "<agent>", "args": [...], "session_id": "<uuid>" }` into `config-<tag>-<agent>.json`.

**Recovery Flags:**
- **File:** [bin/pair-shell](../bin/pair-shell)
- Integrate the agent-specific resume argument in `bin/pair-shell`:
  ```bash
  case "$r_agent" in
      claude)        resume_extra="--resume $r_sid" ;;
      codex)         resume_extra="resume $r_sid" ;;
      agy)           resume_extra="--conversation $r_sid" ;;
  esac
  ```
- Support checking for active/resumable native session files in `agent_session_exists()`:
  ```bash
      agy)
          [ -f "$HOME/.gemini/antigravity-cli/conversations/$sid.db" ]
          ;;
  ```

**Telemetry Signal** (aspect `3`, see §3): `session-id` from `pair-session-watch` — `fired` when `AgentSpec.Match` resolves an id and the config is written, **`near-miss`** when a file matching the watch pattern is found but no id can be extracted (filename/format drift), `fail` when the 60-second fast-discovery window elapses without a fresh PID or an id. With a fresh PID, the watcher instead continues discovery every 60 seconds for the agent process lifetime, accommodating agents that create their transcript only after the first interaction. The resume mapping in `bin/pair-shell` is the *consumer* of this id; it's static config with no separate signal.

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

1. [ ] **Verify Return Key remapping** in `sendKeymapByAgent` (Enter = newline, Alt+Enter = send).
2. [ ] **Check for blocking TUI overlays** (permission pickers **and** user selection / AskUserQuestion menus) and implement a PTY overlay detector in `overlayDetectorByAgent` if needed — verify plain Enter confirms the picker and Alt+Enter is not required.
3. [ ] **Implement Session Watching** in `cmd/internal/sessionwatch` / `cmd/pair-session-watch` (using `lsof` and target file patterns).
4. [ ] **Configure Launcher Recovery** in `bin/pair-shell` (mapping `--conversation` or `--resume` flags).
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
one JSON line per trigger to `$PAIR_DATA_DIR/adapt-<tag>.jsonl` during normal use.
`bin/pair-shell` truncates the file once at session launch; all components then append
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
| 3 Session watch | `session-id` | pair-session-watch | fired, near-miss, fail | `fail` (timeout) / `near-miss` (file found, id unparsed) |
| 4 Slug gen | `slug-parse` | pair-slug | fired, near-miss, fail | `near-miss` (transcript parsed, 0 turns) / `fail` (resolved a transcript but couldn't read/parse it) |
| 5 PTY filter | `output-filter` | pair-wrap | fired | a `fired` line that *stops* appearing (its absence is the signal — the sequence was renamed) |
| 6 Settings | — | — | — | static config; no signal |
| 7 Prompt search | `prompt-search` | nvim/scrollback.lua | fired, near-miss | `near-miss` (0 matches in non-empty scrollback) |

> Status: all six runtime aspects emit today (#000045 M1: aspects 1 & 2; M2: aspects 3, 4, 5, 7) — verified for `claude`/`codex`/`agy`/`muse` via #000134.

**Privacy:** `detail` can carry a snippet of agent output (e.g. an unrecognized
prompt). It is capped at 200 bytes and the file stays local under `$PAIR_DATA_DIR`,
the same trust level as the existing scrollback logs. `bin/pair-shell` removes it on quit.
