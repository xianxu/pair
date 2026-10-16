package wrapcmd

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

// Install before starting either pump. As in production, the transport owns
// descriptor flags and must restore them before the fixture closes its file.
func prepareMasterPumpTransport(t *testing.T, p *proxy, done <-chan struct{}) {
	t.Helper()
	transport, err := ttyio.NewFile(p.ptmx, p.ptmx, false)
	if err != nil {
		t.Fatal(err)
	}
	p.childIO = transport
	t.Cleanup(func() {
		if err := transport.Close(); err != nil {
			t.Errorf("close child terminal transport: %v", err)
		}
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("master pump failed to join after closing child transport")
		}
	})
}
