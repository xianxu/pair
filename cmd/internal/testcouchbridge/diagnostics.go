package testcouchbridge

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
)

// A physical retained path fits the usual 4096-byte path limit; quoting even
// all-control bytes remains bounded independently of the fixture output.
const diagnosticMarkerLimit = 32 << 10

// Diagnostics bounds all fixture output in one stream. It deliberately embeds
// no writer: io.Copy's optional method dispatch must not bypass Write's budget.
type Diagnostics struct {
	mu                  sync.Mutex
	output              io.Writer
	remaining           int
	truncated, finished bool
	err                 error
}

func NewDiagnostics(output io.Writer) *Diagnostics {
	if output == nil {
		output = io.Discard
	}
	return &Diagnostics{output: output, remaining: DiagnosticLimit}
}

func (d *Diagnostics) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.finished {
		return 0, errors.New("diagnostics already finished")
	}
	if d.err != nil {
		return 0, d.err
	}
	count := len(p)
	if count > d.remaining {
		count = d.remaining
		d.truncated = true
	}
	d.remaining -= count
	if count > 0 {
		n, err := d.output.Write(p[:count])
		if err == nil && n != count {
			err = io.ErrShortWrite
		}
		if err != nil {
			d.err = err
			return n, err
		}
	}
	// Dropped diagnostic bytes are consumed, so copying never retries them.
	return len(p), nil
}

// Finish emits at most one bounded marker after all writers have joined. A
// retained root is independent of the content budget, so a full fixture log
// cannot hide the evidence location. Repeated finalization performs no IO.
func (d *Diagnostics) Finish(root string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.finished {
		return d.err
	}
	d.finished = true
	if d.err != nil {
		return d.err
	}
	var parts []string
	if d.truncated {
		parts = append(parts, "diagnostics truncated")
	}
	if root != "" {
		suffix := ""
		if len(root) > 4096 {
			root = root[:4096]
			suffix = " [root truncated]"
		}
		parts = append(parts, "retained root "+strconv.Quote(root)+suffix)
	}
	if len(parts) == 0 {
		return nil
	}
	marker := "bridge: " + strings.Join(parts, "; ") + "\n"
	if len(marker) > diagnosticMarkerLimit {
		return errors.New("diagnostic marker exceeds bound")
	}
	n, err := io.WriteString(d.output, marker)
	if err == nil && n != len(marker) {
		err = io.ErrShortWrite
	}
	d.err = err
	return err
}
