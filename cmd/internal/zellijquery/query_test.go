package zellijquery

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/zellijexec"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func queryFixture(t *testing.T) (*zellijtest.Fixture, Source) {
	t.Helper()
	f := zellijtest.New(t)
	f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"test": {
		Panes: []zellijtest.Pane{zellijtest.DefaultPane(1, "sh")},
		Tab:   zellijtest.DefaultTab(), Clients: []zellijtest.Client{},
	}}})
	return f, Source{Path: f.Path(), Env: f.Environment()}
}

func TestReadContextRecoversProductionReadShapes(t *testing.T) {
	for _, args := range [][]string{
		{"list-sessions", "--no-formatting"},
		{"--session", "test", "action", "list-clients"},
		{"action", "list-clients"},
		{"action", "list-panes", "--json"},
		{"action", "list-panes", "--json", "--command", "--state", "--geometry"},
		{"--session", "test", "action", "list-panes", "--json", "--command", "--state", "--tab", "--all"},
		{"action", "current-tab-info", "--json"},
	} {
		t.Run(args[len(args)-1], func(t *testing.T) {
			f, source := queryFixture(t)
			empty := zellijtest.Case(t, "launcher-frames-whitespace").Stdout
			crossed := zellijtest.Reply(t, "tab", zellijtest.DefaultTab()).Stdout
			if ReadKind(args) == TabReply {
				crossed = zellijtest.Reply(t, "clients", []zellijtest.Client{}).Stdout
			}
			f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &empty}, zellijtest.Fault{Ordinal: 2, Reply: &crossed})
			var observations []Observation
			ctx := WithObserver(t.Context(), func(o Observation) { observations = append(observations, o) })
			result, err := source.ReadContext(ctx, args...)
			if err != nil || len(result.Stdout) == 0 || len(f.Queries()) != 3 || len(f.Effects()) != 0 {
				t.Fatalf("result=%+v err=%v queries=%v effects=%v", result, err, f.Queries(), f.Effects())
			}
			if len(observations) != 3 || !errors.Is(observations[0].Err, ErrNoReply) || !errors.Is(observations[1].Err, ErrCrossedReply) || observations[2].Err != nil {
				t.Fatalf("attempts=%+v", observations)
			}
			for _, observation := range observations {
				if !observation.EffectiveDeadline.Equal(observations[0].EffectiveDeadline) {
					t.Fatal("attempt refreshed its deadline")
				}
			}
		})
	}
}

func TestReadContextCancellationJoinsBeforeNextAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	source := Source{Run: func(*exec.Cmd) error { calls++; cancel(); return nil }}
	_, err := source.ReadContext(ctx, "action", "list-clients")
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestReadContextObserverCancellationRetiresDiagnosticAuthority(t *testing.T) {
	f, source := queryFixture(t)
	f.SetFaults(zellijtest.Fault{Ordinal: 1, Error: "native refusal"})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx = WithObserver(ctx, func(Observation) { cancel() })
	result, err := source.ReadContext(ctx, "action", "list-clients")
	_, direct := result.ProcessErr.(*exec.ExitError)
	if !errors.Is(err, context.Canceled) || direct || len(f.Queries()) != 1 {
		t.Fatalf("err=%v raw=%v direct=%v calls=%d", err, result.ProcessErr, direct, len(f.Queries()))
	}
}

func TestReadContextWorkIsLinear(t *testing.T) {
	for _, n := range []int{8, 16} {
		for _, factor := range []int{1, 3} {
			f, source := queryFixture(t)
			empty := zellijtest.Case(t, "launcher-frames-whitespace").Stdout
			var faults []zellijtest.Fault
			for ordinal := 1; ordinal <= n*2*factor; ordinal++ {
				if ordinal%factor != 0 {
					faults = append(faults, zellijtest.Fault{Ordinal: ordinal, Reply: &empty})
				}
			}
			f.SetFaults(faults...)
			for range n {
				read := func(ctx context.Context) ([]byte, error) { return source.Output(ctx, "action", "list-clients") }
				if _, err := zellijreply.Confirm(t.Context(), read, func(raw []byte) (string, error) { return string(raw), nil }); err != nil {
					t.Fatal(err)
				}
			}
			if got := len(f.Queries()); got != 2*factor*n {
				t.Fatalf("%d reads made %d attempts", n, got)
			}
		}
	}
}

func TestReadContextRetainsProcessErrorProvenance(t *testing.T) {
	for _, collectorFails := range []bool{false, true} {
		f, source := queryFixture(t)
		f.SetFaults(zellijtest.Fault{Ordinal: 1, Error: "native refusal"})
		collection := errors.New("collection uncertain")
		if collectorFails {
			source.Run = func(cmd *exec.Cmd) error { return errors.Join(zellijexec.Run(cmd), collection) }
		}
		result, err := source.ReadContext(t.Context(), "action", "list-clients")
		if err == nil || len(f.Queries()) != 1 || string(result.Stderr) != "native refusal\n" {
			t.Fatalf("result=%+v err=%v queries=%v", result, err, f.Queries())
		}
		_, direct := result.ProcessErr.(*exec.ExitError)
		if direct == collectorFails || errors.Is(err, collection) != collectorFails {
			t.Fatalf("direct=%v collector=%v err=%v", direct, collectorFails, err)
		}
	}
}

func TestReadContextNeverReplaysMutationsOrMalformedReplies(t *testing.T) {
	for _, args := range [][]string{
		{"action", "close-pane", "--pane-id", "terminal_1"},
		{"action", "list-panes", "--json", "--future"},
		{"action", "list-panes", "--json"},
	} {
		f, source := queryFixture(t)
		if args[1] != "close-pane" {
			raw := "[{"
			f.SetFaults(zellijtest.Fault{Ordinal: 1, Reply: &raw})
		}
		_, _ = source.ReadContext(t.Context(), args...)
		if len(f.Queries()) != 1 {
			t.Fatalf("replayed %q: %v", args, f.Queries())
		}
		if args[1] == "close-pane" && len(f.Effects()) != 1 {
			t.Fatalf("mutation effects=%v", f.Effects())
		}
	}
}

func TestReadContextSocketProbeUsesTheChildEnvironment(t *testing.T) {
	for _, childPresent := range []bool{false, true} {
		f, source := queryFixture(t)
		f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{}})
		ambient, child := testpair.ShortRoot(t), testpair.ShortRoot(t)
		selected := ambient
		if childPresent {
			selected = child
		}
		socket := filepath.Join(selected, "contract_version_1", "pair-a")
		if err := os.MkdirAll(filepath.Dir(socket), 0700); err != nil {
			t.Fatal(err)
		}
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := listener.Close(); err != nil {
				t.Error(err)
			}
		})
		t.Setenv("ZELLIJ_SOCKET_DIR", ambient)
		source.Env = append(source.Env, "ZELLIJ_SOCKET_DIR="+child)
		_, err = source.ReadContext(t.Context(), "--session", "pair-a", "action", "list-clients")
		want := 1
		if childPresent {
			want = 3
		}
		if len(f.Queries()) != want || errors.Is(err, ErrSessionLookupMiss) != childPresent {
			t.Fatalf("child socket=%v queries=%d err=%v", childPresent, len(f.Queries()), err)
		}
	}
}
