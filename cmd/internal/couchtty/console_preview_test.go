package couchtty

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func previewFormState(t *testing.T, submit bool) (MenuState, []MenuEffect) {
	t.Helper()
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	state.Agents = []string{"claude", "codex"}
	state.RootAgent = "claude"
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyCtrlSpace}})
	for _, r := range "/repo" {
		state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: KeyRune, Rune: r}})
	}
	key := KeyTab
	if submit {
		key = KeyEnter
	}
	state, effects := ReduceMenu(state, MenuEvent{Kind: MenuEventKey, Key: PanelKey{Kind: key}})
	if len(effects) != 1 || effects[0].Preview == nil {
		t.Fatalf("preview form = state %+v effects %+v", state, effects)
	}
	return state, effects
}

func TestConsoleStartPreviewFeedsPreparedResultToReducer(t *testing.T) {
	f := newFixture(t, 24, 80)
	state, effects := previewFormState(t, false)
	want := couchcore.PreparedStart{Token: "accepted", Resolution: couchcore.StartResolution{
		CanonicalPath: "/repo", Profile: couchcore.LaunchProfile{Agent: "claude"},
	}}
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name != "prepare-start" || call.Context == nil || call.Args["path"] != "/repo" {
			return nil, errors.New("malformed preview call")
		}
		return want, nil
	})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady = state, true
	f.con.mu.Unlock()
	f.con.dispatchMenuEffects(effects)

	waitUpTo(t, 250*time.Millisecond, "prepared preview to enter reducer", func() bool {
		got := f.con.menuSnapshot().CurrentFrame()
		return got.PreviewAccepted == got.Generation && got.PreviewToken == want.Token
	})
}

func TestConsolePreviewCancellationRunsLatestPendingAndJoins(t *testing.T) {
	f := newFixture(t, 24, 80)
	firstStarted := make(chan struct{})
	firstCanceled := make(chan struct{})
	secondStarted := make(chan struct{})
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		switch call.Args["path"] {
		case "/one":
			close(firstStarted)
			<-call.Context.Done()
			close(firstCanceled)
			return nil, call.Context.Err()
		case "/two":
			close(secondStarted)
			return couchcore.PreparedStart{Token: "two", Resolution: couchcore.StartResolution{CanonicalPath: "/two"}}, nil
		default:
			return nil, errors.New("unexpected preview")
		}
	})
	f.con.dispatchMenuEffects([]MenuEffect{{Preview: &PreviewRequest{Generation: 1, Path: "/one", Agent: "claude"}}})
	select {
	case <-firstStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("first preview did not start")
	}
	f.con.dispatchMenuEffects([]MenuEffect{{Preview: &PreviewRequest{Generation: 2, Path: "/two", Agent: "codex"}}})
	select {
	case <-firstCanceled:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("replacement preview did not cancel running generation")
	}
	select {
	case <-secondStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("latest pending preview did not start after terminal cancellation")
	}

	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Console.Run did not join preview workers")
	}
}

func TestConsolePendingSubmitDispatchesAcceptedTokenOnce(t *testing.T) {
	f := newFixture(t, 24, 80)
	state, effects := previewFormState(t, true)
	calls := make(chan couchcore.OperationCall, 2)
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		calls <- call
		if call.Name == "prepare-start" {
			return couchcore.PreparedStart{Token: "accepted", Resolution: couchcore.StartResolution{
				CanonicalPath: "/repo", Profile: couchcore.LaunchProfile{Agent: "claude"},
			}}, nil
		}
		return nil, nil
	})
	f.con.mu.Lock()
	f.con.menu, f.con.menuReady = state, true
	f.con.mu.Unlock()
	f.con.dispatchMenuEffects(effects)

	got := make([]string, 0, 2)
	for len(got) < 2 {
		select {
		case call := <-calls:
			if call.Context == nil {
				t.Fatal("menu effect dispatched without Console context")
			}
			got = append(got, call.Name+":"+call.Args["token"])
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("operation sequence = %v, want prepare-start then start", got)
		}
	}
	if want := []string{"prepare-start:", "start:accepted"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("operation sequence = %v, want %v", got, want)
	}
	select {
	case extra := <-calls:
		t.Fatalf("pending submit redispatched %+v", extra)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestConsolePreviewCancellationOnStop(t *testing.T) {
	f := newFixture(t, 24, 80)
	started := make(chan struct{})
	finished := make(chan error, 1)
	f.con.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		close(started)
		<-call.Context.Done()
		finished <- call.Context.Err()
		return nil, call.Context.Err()
	})
	f.con.dispatchMenuEffects([]MenuEffect{{Preview: &PreviewRequest{Generation: 1, Path: "/repo"}}})
	<-started
	f.con.Stop()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("preview stopped with %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Stop did not cancel preview")
	}
	select {
	case <-f.done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Stop did not join preview worker")
	}
}
