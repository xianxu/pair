package couchtty

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func messageBinding() couchmessage.Binding {
	return couchmessage.Binding{Slot: "pair:1", Repository: "/repo/.git", Scope: "scope", Tag: "thread", Session: "session", Nonce: "nonce", Agent: "codex", Version: "test", PID: 23, Start: "wrapper-start"}
}
func TestMessageThreadsOnlyExposeLiveCommittedPanes(t *testing.T) {
	f := newFixture(t, 24, 80)
	b := messageBinding()
	f.con.mu.Lock()
	p := f.con.panes["c1"]
	p.thread = couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
	p.tree = "/repo"
	p.process = couchcore.ProcessIdentity{PID: 42, Identity: "client-start"}
	f.con.mu.Unlock()
	rows := f.con.MessageThreads()
	if len(rows) != 1 || rows[0].Scope != b.Scope || rows[0].Tree != "/repo" || rows[0].Process.PID != 42 {
		t.Fatalf("snapshot %+v", rows)
	}
	root, err := f.con.MessageBinding(context.Background(), b)
	if err != nil || root != "/repo" {
		t.Fatalf("binding %q %v", root, err)
	}
	b.Tag = "other"
	if _, err = f.con.MessageBinding(context.Background(), b); err == nil {
		t.Fatal("foreign thread accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = f.con.MessageBinding(ctx, messageBinding()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel %v", err)
	}
	f.child.Exit(0)
	waitFor(t, "dead pane absent from message registry", func() bool { return len(f.con.MessageThreads()) == 0 })
}
func TestMessageActivityCountsKeysAndPasteButNotFocus(t *testing.T) {
	base := time.Unix(1000, 0)
	var nanos atomic.Int64
	nanos.Store(base.UnixNano())
	broker := couchmessage.NewBroker(context.Background(), func() time.Time { return time.Unix(0, nanos.Load()) }, nil)
	defer broker.Close()
	binding := messageBinding()
	if err := broker.Register(binding, nil); err != nil {
		t.Fatal(err)
	}
	f := newFixture(t, 24, 80)
	f.con.mu.Lock()
	f.con.panes["c1"].thread = couchcore.ThreadAddress{RepoScope: binding.Scope, Tag: couchcore.ThreadTag(binding.Tag)}
	f.con.mu.Unlock()
	f.con.SetMessageBroker(broker)
	last := func() time.Time {
		rows, err := broker.Actors(context.Background(), binding)
		if err != nil || len(rows) != 1 {
			t.Fatalf("actors %v %v", rows, err)
		}
		return rows[0].LastActivity
	}
	nanos.Store(base.Add(time.Minute).UnixNano())
	f.con.deliverChildInput(uv.FocusEvent{})
	if !last().Equal(base) {
		t.Fatal("focus counted as operator typing")
	}
	f.con.deliverChildInput(uv.KeyPressEvent{Code: 'x', Text: "x"})
	if !last().Equal(base.Add(time.Minute)) {
		t.Fatal("keypress did not reset quiet")
	}
	nanos.Store(base.Add(2 * time.Minute).UnixNano())
	f.con.deliverChildInput(uv.PasteEvent{Content: "draft"})
	if !last().Equal(base.Add(2 * time.Minute)) {
		t.Fatal("paste did not reset quiet")
	}
}
