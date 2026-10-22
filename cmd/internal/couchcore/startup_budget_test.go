package couchcore

import (
	"context"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/startupbudget"
)

func TestTrackedStartupSharesDeadlineBeforeHelperAckAndRegistration(t *testing.T) {
	env := newTestEnv(t, "/repo")
	parked := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	address := parked.Address
	env.Artifacts.SetNativeBinding(address, "claude", sessioninventory.BindingEstablished, "native-root-1")
	synctest.Test(t, func(t *testing.T) {
		defer func() {
			for _, id := range env.Runner.order {
				if c := env.Runner.Child(id).terminal; c != nil {
					_ = c.Close()
				}
			}
		}()
		parent, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		want, _ := parent.Deadline()
		var transported string
		var registration context.Context
		env.Runner.BeforeAcknowledge = func(id string) error {
			for _, v := range env.Runner.Child(id).Env {
				if strings.HasPrefix(v, startupbudget.DeadlineEnv+"=") {
					transported = strings.TrimPrefix(v, startupbudget.DeadlineEnv+"=")
				}
			}
			// Observe this timer in testing/synctest virtual time; no wall-clock guess.
			<-time.After(12 * time.Second)
			return nil
		}
		env.Runner.AfterAcknowledge = func(id string) error {
			env.Artifacts.SetPaneSidecar(address, "claude")
			env.Artifacts.SetPairSession(address, continuationChildSession(t, env.Runner, id), true)
			return nil
		}
		env.Artifacts.PaneSidecarsHook = func(ThreadAddress) error { return nil }
		env.Couch.registrationDeadline = func(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
			registration = ctx
			return context.WithTimeout(ctx, d)
		}
		_, _, err := env.Couch.ResumeContextWith(parent, address, ResumeOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if transported != startupbudget.Encode(want) {
			t.Errorf("helper deadline=%q want=%q", transported, startupbudget.Encode(want))
		}
		if registration == nil {
			t.Fatal("missing startup budget owner")
		}
		got, ok := registration.Deadline()
		if !ok || !got.Equal(want) {
			t.Errorf("registration deadline=%v want=%v", got, want)
		}
	})
}

// These convenience launch APIs have no caller deadline. Require their one
// transported deadline to be the shared default selected during this call;
// the interval brackets selection, rather than imposing a wall-clock budget.
func startupDeadlineEnvForTest(t *testing.T, env []string, began, returned time.Time) string {
	t.Helper()
	var found string
	for _, entry := range env {
		raw, ok := strings.CutPrefix(entry, startupbudget.DeadlineEnv+"=")
		if !ok {
			continue
		}
		if found != "" {
			t.Fatal("duplicate startup deadline")
		}
		deadline, err := startupbudget.Decode(raw)
		if err != nil {
			t.Fatalf("startup deadline: %v", err)
		}
		if deadline.Before(began.Add(startupbudget.Default)) || deadline.After(returned.Add(startupbudget.Default)) {
			t.Fatalf("startup deadline %v not selected from shared fallback during call [%v, %v]", deadline, began, returned)
		}
		found = entry
	}
	if found == "" {
		t.Fatal("startup deadline missing")
	}
	return found
}
