package termcmd

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifyosc"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestShellClipboardPolicyThroughOutput(t *testing.T) {
	m, parent := presentationFixture(t)
	hidden := addPresentationTab(t, m, 1, "hidden")
	selected := addPresentationTab(t, m, 2, "selected")
	visibleWire := []byte("\x1b]52;c;c2VsZWN0ZWQ=\x1b\\")
	hiddenWire := []byte("\x1b]52;c;aGlkZGVu\x1b\\")
	selected.Feed(visibleWire)
	flushPresentation(t, m, selected)
	hidden.Feed(hiddenWire)
	flushPresentation(t, m, hidden)
	if n := bytes.Count(parent.Bytes(), visibleWire); n != 1 {
		t.Errorf("selected clipboard count=%d", n)
	}
	if n := bytes.Count(parent.Bytes(), hiddenWire); n != 0 {
		t.Errorf("hidden clipboard count=%d", n)
	}
	m.previousTab()
	flushPresentation(t, m, hidden)
	if n := bytes.Count(parent.Bytes(), hiddenWire); n != 0 {
		t.Errorf("clipboard replayed on selection=%d", n)
	}
	if n := bytes.Count(parent.Bytes(), visibleWire); n != 1 {
		t.Errorf("selected clipboard replayed on selection=%d", n)
	}
	for _, child := range []*ptychild.Child{hidden, selected} {
		child.Feed([]byte("\x1b]52;c;?\a"))
		flushPresentation(t, m, child)
		if got := string(bytes.Join(child.Writes(), nil)); got != "\x1b]52;c;\x1b\\" {
			t.Errorf("clipboard local reply=%q", got)
		}
	}
	if bytes.Contains(parent.Bytes(), []byte("\x1b]52;c;?")) {
		t.Fatal("clipboard read reached physical terminal")
	}
}
func TestShellNotificationsHiddenSelectedAndRedeliveryOnce(t *testing.T) {
	m, parent := presentationFixture(t)
	hidden := addPresentationTab(t, m, 1, "hidden")
	selected := addPresentationTab(t, m, 2, "selected")
	for i, child := range []*ptychild.Child{hidden, selected} {
		wire := notifyosc.Encode("policy-notification")
		split := len(wire) / 2
		child.Feed(wire[:split])
		flushPresentation(t, m, child)
		envelope := []byte("\x1b]777;notify;pair;policy-notification\x1b\\")
		if n := bytes.Count(parent.Bytes(), envelope); n != i {
			t.Fatalf("partial notification escaped count=%d want%d", n, i)
		}
		child.Feed(wire[split:])
		flushPresentation(t, m, child)
		if n := bytes.Count(parent.Bytes(), envelope); n != i+1 {
			t.Fatalf("completed notification count=%d want%d", n, i+1)
		}
	}
	m.previousTab()
	flushPresentation(t, m, hidden)
	if n := bytes.Count(parent.Bytes(), []byte("\x1b]777;notify;pair;policy-notification\x1b\\")); n != 2 {
		t.Fatal("selection replayed notifications")
	}
	out, err := hidden.Endpoint().Feed(notifyosc.Encode("dedup-policy"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	batch := ptychild.OutputBatch{Terminal: out}
	for range 2 {
		if err := m.handleOutput(context.Background(), 1, batch); err != nil {
			t.Fatal(err)
		}
	}
	if n := bytes.Count(parent.Bytes(), []byte("\x1b]777;notify;pair;dedup-policy\x1b\\")); n != 1 {
		t.Fatalf("redelivery repeated notification %d", n)
	}
}

func TestShellMappedZellijNotificationsPreserveBothOriginsOnce(t *testing.T) {
	m, parent := presentationFixture(t)
	hidden := addPresentationTab(t, m, 1, "hidden")
	selected := addPresentationTab(t, m, 2, "selected")
	for i, child := range []*ptychild.Child{hidden, selected} {
		message := []string{"hidden-mapped", "selected-mapped"}[i]
		out, err := child.Endpoint().Feed([]byte("\x1b]9;pair: "+message+"\x1b\\"), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		batch := ptychild.OutputBatch{Terminal: out}
		for range 2 {
			if err := m.handleOutput(context.Background(), i+1, batch); err != nil {
				t.Fatal(err)
			}
		}
		if n := bytes.Count(parent.Bytes(), []byte("\x1b]777;notify;pair;"+message+"\x1b\\")); n != 1 {
			t.Fatalf("origin%d delivery count=%d", i, n)
		}
	}
	m.previousTab()
	flushPresentation(t, m, hidden)
	if n := bytes.Count(parent.Bytes(), []byte("\x1b]777;notify;pair;")); n != 2 {
		t.Fatalf("switch replayed mapped notifications:%d", n)
	}
}
