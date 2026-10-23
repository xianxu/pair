package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/rowtext"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

// ChromeStatusCells includes the ellipsis; source diagnostics remain unmodified.
const ChromeStatusCells = 40

// ChromeStatusText is the single-row display projection, never a storage limit.
func ChromeStatusText(text string) string { return chromeStatusText(text, nil) }

// work counts bytes admitted to the existing linear text passes, as the status
// packer's work ledger does for label sanitization; it changes no decisions.
func chromeStatusText(text string, work *int) string {
	charge := func() {
		if work != nil {
			*work += len(text)
		}
	}
	charge()
	text = rowtext.Sanitize(text)
	charge()
	if textwidth.Width(text) <= ChromeStatusCells {
		return text
	}
	charge()
	return rowtext.Fit(text, ChromeStatusCells-1) + "…"
}
