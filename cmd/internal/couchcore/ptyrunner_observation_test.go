package couchcore

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestPtyRunnerObservesFirstBytesAndGeometryBeforeDelivery(t *testing.T) {
	var mu sync.Mutex
	var observations []terminal.Observation
	r := &PtyRunner{
		Size: func() ptychild.Size { return ptychild.Size{Rows: 7, Cols: 19} },
		Observer: func(o terminal.Observation) {
			mu.Lock()
			defer mu.Unlock()
			o.Data = append([]byte(nil), o.Data...)
			observations = append(observations, o)
		},
	}
	h, err := r.Start(t.TempDir(), []string{"sh", "-c", "printf FIRST"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := h.(TerminalHandle).Terminal()
	defer c.Close()
	select {
	case <-c.Exited():
	case <-time.After(5 * time.Second):
		t.Fatal("child never exited")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(observations) < 3 || observations[0].Kind != "endpoint-open" || observations[len(observations)-1].Kind != "endpoint-end" {
		t.Fatalf("missing initial/final observations: %+v", observations)
	}
	var raw []byte
	for _, o := range observations {
		if o.EndpointID != h.ID() || o.Geometry != (terminal.Geometry{Cols: 19, Rows: 7}) {
			t.Fatalf("wrong endpoint attribution: %+v", o)
		}
		if o.Kind == "endpoint-feed" {
			raw = append(raw, o.Data...)
		}
	}
	if !bytes.Equal(raw, []byte("FIRST")) {
		t.Fatalf("early bytes lost or changed: %q", raw)
	}
}
