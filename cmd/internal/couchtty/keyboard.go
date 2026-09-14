package couchtty

import (
	"io"

	"github.com/xianxu/pair/cmd/internal/hostty"
)

// keyboardDisambiguated maintains Couch's own input requirement at a complete
// output boundary. It leaves other flags and the terminal stack intact, and
// never inserts a control into a child's unfinished escape sequence.
func keyboardDisambiguated(p []byte, complete bool) []byte {
	if !complete {
		return p
	}
	out := make([]byte, len(p)+len(hostty.EnableKeyboardDisambiguation))
	copy(out, p)
	copy(out[len(p):], hostty.EnableKeyboardDisambiguation)
	return out
}

// lockTerminal grants one scanner/write transaction, or refuses after release.
// Callers must not hold c.mu, and must unlock before calling a child.
func (c *Console) lockTerminal() bool {
	c.terminalMu.Lock()
	if c.terminalReleased {
		c.terminalMu.Unlock()
		return false
	}
	return true
}

// writeHostControl serializes complete console cursor/startup controls. Child
// output and interleaved paints use their respective framing-aware paths.
func (c *Console) writeHostControl(p string) {
	if !c.lockTerminal() {
		return
	}
	defer c.terminalMu.Unlock()
	_, _ = io.WriteString(c.host, p)
}
