package couchcmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Exercise the real service composition against an existing legacy list, not
// just a fresh Console without the production startup wiring.
func TestAttentionServiceIgnoresLegacyListAndDoesNotPersistEvents(t *testing.T) {
	rawRT, _, _ := scratchWiringRuntime(t)
	rt, lease, err := rawRT.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ns, err := rt.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	c, err := rt.NewCouchWith(couchcore.NewFakeRunner(), ns)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(c.Namespace.Dir(), "attention", "list.json")
	raw, err := json.Marshal(map[string]any{"version": 1, "items": []couchcore.AttentionItem{{ID: "legacy", Kind: couchcore.AttentionNotify, Label: "old hub", Body: "OLD-LEGACY-NOTICE", Since: time.Now()}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100})
	reader, input := io.Pipe()
	defer reader.Close()
	defer input.Close()
	console := couchtty.New(host, reader)
	child := ptychild.NewFakeChild(nil)
	console.Attach("fixture", "fixture", child)
	service, err := startMessageService(console, c)
	if err != nil {
		t.Fatal(err)
	}
	if service == nil {
		t.Fatal("real service not composed")
	}
	defer service.Close()
	done := make(chan struct{})
	go func() { console.Run(); close(done) }()
	defer func() {
		console.Stop()
		child.Exit(0)
		input.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("console failed to stop")
		}
	}()
	await := func(text string) {
		t.Helper()
		deadline := testwait.After(t)
		for {
			select {
			case <-host.Writes():
				if strings.Contains(host.Written(), text) {
					return
				}
			case <-deadline:
				t.Fatalf("missing %q in host output: %s", text, host.Written())
			}
		}
	}
	_, _ = input.Write([]byte("\x1b[97;7u"))
	// Empty-list key feedback is feed-only. The first new event's exact badge
	// count below proves that the legacy list was not restored, independently
	// of the removed general tab-bar notice.
	if strings.Contains(host.Written(), "OLD-LEGACY-NOTICE") {
		t.Fatal("legacy event restored")
	}
	console.ObserveAttention(couchcore.AttentionFact{Kind: couchcore.AttentionNotify, Notify: couchcore.AttentionItem{ID: "new", Label: "new hub", Body: "new nudge"}})
	await(couchtty.AttentionGlyph + "1")
	_, _ = input.Write([]byte("\x1b[97;7u"))
	await("Clear all (c)")
	_, _ = input.Write([]byte("c"))
	service.Close()
	console.Stop()
	child.Exit(0)
	select {
	case <-done:
	case <-testwait.After(t):
		t.Fatal("console failed to stop")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, raw) {
		t.Fatalf("ephemeral events rewrote legacy file: %v", err)
	}
}
