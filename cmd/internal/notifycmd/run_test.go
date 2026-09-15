package notifycmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifytransport"
)

// The fake resolves a mutable binding to live inboxes, so replacement and
// unavailable brokers exercise the same state transitions as real senders.
type fakeRuntime struct {
	env      map[string]string
	bindings map[string]string
	inboxes  map[string][]string
}

func (f *fakeRuntime) Getenv(key string) string { return f.env[key] }
func (f *fakeRuntime) SendNotification(binding, message string) error {
	id, ok := f.bindings[binding]
	if !ok {
		return errors.New("missing PID binding")
	}
	if _, ok = f.inboxes[id]; !ok {
		return errors.New("broker unavailable")
	}
	f.inboxes[id] = append(f.inboxes[id], message)
	return nil
}
func validRuntime() *fakeRuntime {
	return &fakeRuntime{env: map[string]string{"PAIR_TAG": "pair", "PAIR_PAIR_WRAP_PID_PATH": "/state/wrapper"}, bindings: map[string]string{"/state/wrapper": "first"}, inboxes: map[string][]string{"first": nil}}
}
func TestRunSendsMessagesForLegacyOptions(t *testing.T) {
	for _, args := range [][]string{{"ready"}, {"--osc", "9", "ready"}, {"--osc=777", "ready"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			rt := validRuntime()
			var stderr bytes.Buffer
			if code := Run(args, rt, &stderr); code != 0 || stderr.Len() != 0 {
				t.Fatalf("code=%d stderr=%q", code, stderr.String())
			}
			if got := rt.inboxes["first"]; len(got) != 1 || got[0] != "ready" {
				t.Fatalf("messages=%q", got)
			}
		})
	}
}
func TestRunFollowsCurrentBrokerBinding(t *testing.T) {
	rt := validRuntime()
	var stderr bytes.Buffer
	Run([]string{"before"}, rt, &stderr)
	rt.bindings["/state/wrapper"] = "second"
	rt.inboxes["second"] = nil
	Run([]string{"after"}, rt, &stderr)
	if strings.Join(rt.inboxes["first"], ",") != "before" || strings.Join(rt.inboxes["second"], ",") != "after" {
		t.Fatalf("inboxes=%v", rt.inboxes)
	}
}
func TestRunRejectsUsageErrors(t *testing.T) {
	for _, args := range [][]string{nil, {"--osc", "8", "ready"}, {"--osc"}} {
		rt := validRuntime()
		var stderr bytes.Buffer
		if code := Run(args, rt, &stderr); code != 2 || stderr.Len() == 0 || len(rt.inboxes["first"]) != 0 {
			t.Fatalf("args=%v code=%d stderr=%q", args, code, stderr.String())
		}
	}
}
func TestRunWarnsWithoutOuterTTYFallback(t *testing.T) {
	cases := []func(*fakeRuntime){func(rt *fakeRuntime) { delete(rt.env, "PAIR_TAG") }, func(rt *fakeRuntime) { delete(rt.env, "PAIR_PAIR_WRAP_PID_PATH") }, func(rt *fakeRuntime) { delete(rt.bindings, "/state/wrapper") }, func(rt *fakeRuntime) { delete(rt.inboxes, "first") }}
	for i, mutate := range cases {
		rt := validRuntime()
		rt.env["PAIR_OUTER_TTY_PATH"] = "/unrelated/tty"
		mutate(rt)
		var stderr bytes.Buffer
		if code := Run([]string{"ready"}, rt, &stderr); code != 0 || stderr.Len() == 0 {
			t.Fatalf("case%d code=%d stderr=%q", i, code, stderr.String())
		}
	}
}
func TestOSRuntimeSendsToRealBroker(t *testing.T) {
	binding := filepath.Join(t.TempDir(), "pid")
	b, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	t.Setenv("PAIR_TAG", "test")
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", binding)
	var stderr bytes.Buffer
	if code := Run([]string{"actual hook"}, OSRuntime{}, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	select {
	case msg := <-b.Messages():
		if msg != "actual hook" {
			t.Fatalf("msg=%q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("no hook message")
	}
}
