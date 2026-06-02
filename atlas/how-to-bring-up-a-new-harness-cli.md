# How to Bring Up a New Harness CLI in pair

`pair` is an agent-agnostic, Neovim-backed launcher environment. While the horizontal two-pane design is generic, delivering a premium, seamless pair-programming experience requires integrating the agent across seven critical integration surfaces.

This guide outlines how to bring up a new agent harness CLI (e.g., `agy`) and achieve parity with existing agents (`claude`, `codex`).

---

## 1. Key Integration Aspects

### Aspect 1: Return Key Remapping
By default, the bottom Neovim draft pane maps **Enter** to insert a newline, and **Alt+Enter** to send the buffer. To provide visual and interactive consistency, the top agent pane (which runs inside the transparent PTY proxy `pair-wrap`) should map keys similarly:
- **Plain Enter** inside textareas/prompts should insert a newline (preventing accidental premature sends).
- **Alt+Enter** should submit the input.

**Implementation:**
- **File:** [cmd/pair-wrap/main.go](file:///Users/xianxu/workspace/pair/cmd/pair-wrap/main.go)
- Add the agent to `sendKeymapByAgent` defining `plainCR` and `altCR`:
  ```go
  var sendKeymapByAgent = map[string]sendKeymap{
      "agy": {
          plainCR: []byte{'\n'}, // plain Enter inserts newline
          altCR:   []byte{'\r'}, // Alt+Enter sends query
      },
  }
  ```
- **Note:** Claude uses `\<Enter>` (`[]byte{'\\', '\r'}`) as a newline, while Codex and Antigravity (`agy`) use LF (`\n`) for newline and CR (`\r`) for send.

---

### Aspect 2: Overlay-Aware Return Suspension
If the agent presents blocking overlays, pickers (like file autocompletes), or yes/no confirmation modals, text-area Enter remapping will break the interaction. Inside an overlay, a plain **Enter** must send a bare carriage return (`\r`) to select/confirm.

`pair-wrap` suspends remapping by registering an overlay detector function which arms a temporary `pickerActive` flag. The next plain Enter is bypass-translated to a bare `\r`, and the flag is immediately cleared.

**Implementation:**
- **File:** [cmd/pair-wrap/main.go](file:///Users/xianxu/workspace/pair/cmd/pair-wrap/main.go)
- Register the detector in `overlayDetectorByAgent`:
  ```go
  var overlayDetectorByAgent = map[string]overlayDetector{
      "claude": detectClaudeOverlayOpen,
      "codex":  detectCodexOverlayOpen,
  }
  ```
- Implement the detector. Detectors can scan the rolling output stream for custom OSC escape sequences (e.g. Claude's permission OSC `OSC 777;notify;...`, or Codex's `OSC 9;Plan mode prompt:...`) or fallback to visible text substring matches (e.g., watching for `"Press enter to confirm"`).
- **For `agy`:** Since Antigravity communicates permission and option prompts via the IDE/launcher's tool-call overlay UI (`ask_permission` / `ask_question`) rather than terminal-based character overlays in the PTY, a custom terminal overlay detector is not required.

---

### Aspect 3: Session ID Watcher & Recovery
`pair` features a robust restart-in-place (`Alt+n`) and session reattach (`pair resume <tag>`) mechanism. To make this work, the launcher needs to discover the agent's unique conversation/session ID as soon as it is spawned.

**Discovery & Watcher:**
- **File:** [bin/pair-session-watch.sh](file:///Users/xianxu/workspace/pair/bin/pair-session-watch.sh)
- Since TUI agents do not always expose session IDs on stdout, `pair-session-watch.sh` runs in the background. It finds the agent process PID from `$PAIR_DATA_DIR/agent-pid-<tag>` (written by `pair-wrap`), walks its descendants, and inspects files held open by the processes via `lsof -p <pid>`.
- Configure the agent's session file criteria:
  ```bash
  agy)
      watch_dir="$HOME/.gemini/antigravity-cli/conversations"
      find_args=(-type f -name '*.db')
      ;;
  ```
- Extract the ID from the file path or contents in `extract_id()`:
  ```bash
  agy)
      # Extract UUID from SQLite DB name: ~/.gemini/antigravity-cli/conversations/<uuid>.db
      local fn
      fn=$(basename "$1" .db)
      if [[ "$fn" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
          echo "$fn"
      fi
      ;;
  ```
- When captured, the watcher writes `{ "agent": "<agent>", "args": [...], "session_id": "<uuid>" }` into `config-<tag>-<agent>.json`.

**Recovery Flags:**
- **File:** [bin/pair](file:///Users/xianxu/workspace/pair/bin/pair)
- Integrate the agent-specific resume argument in `bin/pair`:
  ```bash
  case "$r_agent" in
      claude)        resume_extra="--resume $r_sid" ;;
      codex)         resume_extra="resume $r_sid" ;;
      agy)           resume_extra="--conversation $r_sid" ;;
  esac
  ```
- Support checking for active/resumable session files under `has_resumable`:
  ```bash
  agy)
      [ -f "$HOME/.gemini/antigravity-cli/brain/$saved_session/.system_generated/logs/transcript.jsonl" ] && has_resumable=true
      ;;
  ```

---

### Aspect 4: pair-slug Generation
The `pair-slug` script summarizes what the current agent session is about to display in the Zellij list.

**Implementation:**
- **Transcript Parsing:** Register a parser in [cmd/pair-slug/slug.go](file:///Users/xianxu/workspace/pair/cmd/pair-slug/slug.go) under `parseTranscript()`. For JSONL transcripts like `agy`, extract the `content` where `type: "USER_INPUT"`.
- **Model Sandbox Execution:** Ensure that invoking the agent in summarize mode (`agy -p "<prompt>"`) runs inside a clean sandbox (e.g. setting `cmd.Dir = os.TempDir()` in [cmd/pair-slug/main.go](file:///Users/xianxu/workspace/pair/cmd/pair-slug/main.go)). This prevents the agent from triggering expensive workspace exploration tools, speeding up slug generation from 20s to 1s.

---

### Aspect 5: Mouse Scroll & PTY Output Filtering
Some agents emit DEC synchronized-output markers or other terminal control characters that interfere with Zellij's mouse scrollback.
- **PTY Filter:** If an agent behaves poorly with mouse scrolling, `pair-wrap` can intercept and strip specific sequences (e.g., Codex's `ESC[?2026h` synchronized-output toggles) in `updateAgentOutput()` before forwarding the stream to Zellij.

---

### Aspect 6: Agent Settings Configuration
To minimize confirmation prompt fatigue and allow the agent to run commands, create/modify the agent's permission profiles (e.g., `.claude/settings.json` or `.antigravitycli/settings.json`) to white-list common utility commands (like `git`, `make`, `sdlc`, `lsof`, `zellij`) and mount trusted directories. 

Align local settings in workspace directories with parent configurations (e.g. `../ariadne/`) to support continuous testing and seamless automation.

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
  }
  ```

---

## 2. Checklist for Bringing Up a New Agent

When introducing a new agent `<name>`, ensure you complete each item:

1. [ ] **Verify Return Key remapping** in `sendKeymapByAgent` (Enter = newline, Alt+Enter = send).
2. [ ] **Check for blocking TUI overlays** and implement a PTY overlay detector in `overlayDetectorByAgent` if needed.
3. [ ] **Implement Session Watching** in `bin/pair-session-watch.sh` (using `lsof` and target file patterns).
4. [ ] **Configure Launcher Recovery** in `bin/pair` (mapping `--conversation` or `--resume` flags).
5. [ ] **Add slug generation support** in `pair-slug` (transcript parsing + sandboxed print execution).
6. [ ] **Confirm mouse scroll and scrollback render** work smoothly without drawing glitch issues.
7. [ ] **White-list permissions** in the agent's global or workspace settings directory.
8. [ ] **Register the user-prompt glyph** in `nvim/scrollback.lua` for `Alt+b` jumping.
