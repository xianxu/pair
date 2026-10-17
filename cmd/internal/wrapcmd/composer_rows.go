package wrapcmd

import "strings"

// codexStartupStatus reads Codex's startup card: whether it is on screen, and
// whether its model or directory is still resolving. Codex paints its real
// composer while startup authority is still resolving, so peer delivery
// treats a resolving card as not yet a composer.
func codexStartupStatus(snapshot terminalSnapshot) (bool, bool) {
	card, model, directory := false, false, false
	for y := 0; y < snapshot.Cursor.Y; y++ {
		line := strings.TrimSpace(terminalRowText(snapshot, y))
		if strings.Contains(line, "OpenAI Codex (") {
			card = true
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "│"))
		for _, field := range []string{"model:", "directory:"} {
			if len(line) >= len(field) && strings.EqualFold(line[:len(field)], field) {
				value := strings.Fields(strings.TrimSpace(line[len(field):]))
				resolved := len(value) > 0 && value[0] != "loading" && value[0] != "│"
				if field == "model:" {
					model = resolved
				} else {
					directory = resolved
				}
			}
		}
	}
	return card, !model || !directory
}

// terminalRowText projects a screen row, preserving erased blank cells.
func terminalRowText(snapshot terminalSnapshot, y int) string {
	var b strings.Builder
	for x := 0; x < snapshot.Width; x++ {
		if c := snapshot.CellAt(x, y); c != nil {
			if c.Content == "" && c.Width > 0 {
				b.WriteByte(' ')
			} else {
				b.WriteString(c.Content)
			}
		}
	}
	return b.String()
}
