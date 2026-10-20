package workbenchshortcut

// RightTerminalPaneShell is the shared layout, split and recovery command.
// The term process itself owns error-only native-shell recovery.
const RightTerminalPaneShell = `zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" terminal 2>/dev/null; exec pair term`
