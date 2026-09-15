package ptychild

import (
	"context"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"sync/atomic"
	"time"
)

var endpointSequence atomic.Uint64

func (c *Child) initTerminal(id string) error {
	if id == "" {
		id = fmt.Sprintf("pty-%d", endpointSequence.Add(1))
	}
	var err error
	c.endpoint, err = terminal.NewEndpoint(id, terminal.Geometry{Cols: int(c.size.Cols), Rows: int(c.size.Rows)}, childInput{c})
	if err != nil {
		return err
	}
	c.publication = newPublication()
	return nil
}

type childInput struct{ child *Child }

func (w childInput) WriteContext(ctx context.Context, p []byte) (int, error) {
	c := w.child
	if c.fake != nil {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		c.fake.mu.Lock()
		defer c.fake.mu.Unlock()
		if c.fake.exited {
			return 0, terminal.ErrInputEnded
		}
		c.fake.writes = append(c.fake.writes, append([]byte(nil), p...))
		return len(p), nil
	}
	return c.transport.WriteContext(ctx, p)
}
func (c *Child) Endpoint() *terminal.Endpoint          { return c.endpoint }
func (c *Child) FlushOutput(ctx context.Context) error { return c.publication.flush(ctx) }
func (c *Child) ingest(p []byte) error {
	c.feedMu.Lock()
	defer c.feedMu.Unlock()
	if c.endpoint.InputEnded() {
		return terminal.ErrInputEnded
	}
	chunk := append([]byte(nil), p...)
	output, outputErr := c.endpoint.Feed(chunk, time.Now())
	c.mu.Lock()
	c.ring.Append(chunk)
	batch := OutputBatch{Raw: chunk, Terminal: output, Err: outputErr}
	sink := c.sink
	c.mu.Unlock()
	return c.publication.enqueue(batch, sink)
}

// finish publishes exit only after every final output callback acknowledged it.
// Close cancels publication, allowing teardown to join even if UI work stopped.
func (c *Child) finish(code int) {
	c.endpoint.EndInput()
	_ = c.publication.flush(context.Background())
	c.code = code
	close(c.done)
}
