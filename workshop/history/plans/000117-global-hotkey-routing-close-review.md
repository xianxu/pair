# Boundary Review — pair#117

- Boundary: whole-issue close
- Window: `0c6f0df..c251cfb`
- Reviewer: Codex
- Verdict: `FIX-THEN-SHIP`

## Finding

Important: draft Neovim emits the cached process ID as a JSON number, while
the Go `CachedPaneRecord` accepted only a string. Decoding therefore rejected
the production record and global routes fell back to synchronous pane
inventory, defeating the latency fix.

## Resolution

Added a `ProcessID` wire type that accepts numeric producer records and legacy
string records. Added a literal numeric-JSON regression matching Neovim’s
output. Focused Go routing suites and the real-overlay fake-Zellij process
suite pass after remediation.

## Verification

- `go test ./... -count=1`
- `make test-lua`
- `bash tests/term-pane-shortcuts-test.sh`
- `bash tests/review-toggle-test.sh`
- `bash tests/workbench-route-nvim-test.sh`
- Zellij configuration and layout parsing
- `git diff --check`
- Operator live right-terminal Alt+n latency/focus smoke
