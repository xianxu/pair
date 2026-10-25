package couchcmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

func TestHubConflictNoticesRetainWinningOwner(t *testing.T) {
	_, defs, _ := moveFixture(t)
	defs[0].Claims, _ = couchcore.ParseHubClaims([]byte("pair\npair:7\n"))
	defs[1].Claims, _ = couchcore.ParseHubClaims([]byte("pair\npair:8\n"))
	state, err := couchcore.NewHubOwnership(defs)
	if err != nil {
		t.Fatal(err)
	}
	notices := hubConflictNotices(state)
	if len(notices) != 1 {
		t.Fatalf("notices: %+v", notices)
	}
	n := notices[0]
	if n.Class != couchcore.NoticeHubClaimConflict || n.OwnerRoot != defs[0].Root || n.Thread != (couchcore.ThreadAddress{}) || !strings.Contains(n.Body, defs[1].Root) {
		t.Fatalf("conflict lost exact winning owner: %+v", n)
	}
}

func TestLifecycleRemoteDiagnosticRetainsOriginalSource(t *testing.T) {
	q, store, jobs := newLifecycleQueueFixture()
	store.SaveError = errors.New("diagnostic disk unavailable")
	var notices []couchtty.Notice
	q.notify = func(n couchtty.Notice) { notices = append(notices, n) }
	if err := q.Queue("remote\x00/project:1", "relaunch", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "relaunch", Context: ctx}, errors.New("launch failed")
	}, func() {}, func(any, error) {}); err != nil {
		t.Fatal(err)
	}
	q.rows = func() []couchcore.ActionableThreadSummary { return nil }
	_, err := (*jobs)[0].prepare(t.Context())
	(*jobs)[0].finish(nil, err)
	if len(notices) != 1 {
		t.Fatalf("diagnostic notices: %+v", notices)
	}
	n := notices[0]
	if n.Class != couchcore.NoticeRemoteLifecycleDiagnostic || n.Thread != (couchcore.ThreadAddress{RepoScope: "scope", Tag: "original"}) || n.SourceRoot != "/slots/project" || n.OwnerRoot != "" || n.Label != "project:1" {
		t.Fatalf("completion lost original source: %+v", n)
	}
}

func TestRenicerPublishesTypedOriginalBinding(t *testing.T) {
	console, _, _ := warningConsole(t)
	notices := make(chan couchtty.Notice, 4)
	console.SetNoticeRouter(func(n couchtty.Notice) { notices <- n })
	binding := slot3
	binding.Scope = "original-scope"
	machine := slotMachine()
	machine.set(pr(300, 200, 10, "agent@1"))
	r := newConsoleRenicer(console, machine, func() []couchmessage.Binding { return []couchmessage.Binding{binding} }, func(couchmessage.Binding) (int, error) { return 300, nil }, 10, 100)
	r.pass()
	r.pass()
	select {
	case n := <-notices:
		if n.Class != couchcore.NoticeAgentPriorityWarning || n.Thread != bindingThread(binding) || n.Label != binding.Slot || n.OwnerRoot != "" {
			t.Fatalf("priority warning source changed: %+v", n)
		}
	default:
		t.Fatal("missing typed warning")
	}
	select {
	case n := <-notices:
		t.Fatalf("duplicate priority warning: %+v", n)
	default:
	}
}

func TestScratchMetadataWarningPublishesTypedGlobalSource(t *testing.T) {
	console, _, _ := warningConsole(t)
	notices := make(chan couchtty.Notice, 1)
	console.SetNoticeRouter(func(n couchtty.Notice) { notices <- n })
	ctx := withScratchWarnings(t.Context(), console)
	warn := ctx.Value(scratchWarningContextKey{}).(func(string))
	warn("Zellij metadata unchanged: native metadata authority unavailable")
	select {
	case n := <-notices:
		if n.Class != couchcore.NoticeScratchMetadataWarning || n.Thread != (couchcore.ThreadAddress{}) || n.OwnerRoot != "" || n.SourceRoot != "" {
			t.Fatalf("metadata warning gained guessed ownership: %+v", n)
		}
	default:
		t.Fatal("missing typed metadata warning")
	}
}

func TestLifecycleRemoteDiagnosticUsesPreparedSourceWhenUncached(t *testing.T) {
	q, store, jobs := newLifecycleQueueFixture()
	q.rows = func() []couchcore.ActionableThreadSummary { return nil }
	store.SaveError = errors.New("diagnostic disk unavailable")
	var got couchtty.Notice
	q.notify = func(n couchtty.Notice) { got = n }
	source := couchcore.LifecycleFailureTarget{Address: couchcore.ThreadAddress{RepoScope: "scope", Tag: "prepared"}, StartingPath: "/prepared/root", Slot: "prepared:2"}
	if err := q.Queue("remote\x00/project:2", "relaunch", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "relaunch", Context: ctx, FailureTarget: &source}, errors.New("launch failed")
	}, func() {}, func(any, error) {}); err != nil {
		t.Fatal(err)
	}
	_, err := (*jobs)[0].prepare(t.Context())
	(*jobs)[0].finish(nil, err)
	if got.Class != couchcore.NoticeRemoteLifecycleDiagnostic || got.Thread != source.Address || got.SourceRoot != source.StartingPath || got.Label != source.Slot || got.OwnerRoot != "" {
		t.Fatalf("prepared source lost: %+v", got)
	}
}

func TestLifecycleRemoteDiagnosticDoesNotChooseAmbiguousSource(t *testing.T) {
	q, store, jobs := newLifecycleQueueFixture()
	rows := q.rows()
	other := rows[0]
	other.Address.Tag = "replacement"
	other.Target.Slot.WorktreeRoot = "/replacement"
	q.rows = func() []couchcore.ActionableThreadSummary { return append(rows, other) }
	store.SaveError = errors.New("diagnostic disk unavailable")
	var got couchtty.Notice
	q.notify = func(n couchtty.Notice) { got = n }
	if err := q.Queue("remote\x00/project:1", "relaunch", func(context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{}, errors.New("launch failed")
	}, func() {}, func(any, error) {}); err != nil {
		t.Fatal(err)
	}
	_, err := (*jobs)[0].prepare(t.Context())
	(*jobs)[0].finish(nil, err)
	if got.Thread != (couchcore.ThreadAddress{}) || got.SourceRoot != "" {
		t.Fatalf("ambiguous cached rows selected a source: %+v", got)
	}
}
