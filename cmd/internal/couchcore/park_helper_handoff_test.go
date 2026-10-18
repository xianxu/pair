package couchcore

import (
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// A completed attach is a handoff even when Zellij reports a nonzero exit.
// With no quit intent, reaching the consumer returns 4; refusing the attach
// returns 3. No terminal or subprocess is started by this regression.
func TestParkHelperConsumesIntentAfterNonzeroAttach(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		err  error
		want int
	}{
		{"success", 0, nil, 4},
		{"lost-server", 1, nil, 4},
		{"other-exit", 7, nil, 4},
		{"execution-error", 1, errors.New("attach could not start"), 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PAIR_TEST_PARK_ATTEMPT", "1")
			t.Setenv("PAIR_TEST_PARK_COMPLETED_AT", "100")
			t.Setenv("PAIR_TEST_PARK_GLOBAL_DATA_DIR", t.TempDir())
			t.Setenv("PAIR_TEST_PARK_SCOPE", "0123456789abcdef")
			t.Setenv("PAIR_TEST_PARK_TAG", "fixture")
			t.Setenv("PAIR_TEST_PARK_SESSION", "fixture-session")
			t.Setenv("PAIR_TEST_PARK_HOME", t.TempDir())
			got := runParkLifecycleHelperWithAttach(func(launcher.LaunchOptions, launcher.Env, launcher.Runtime, string, string, string) (int, error) {
				return tc.code, tc.err
			})
			if got != tc.want {
				t.Fatalf("helper exit=%d, want %d (attach code=%d error=%v)", got, tc.want, tc.code, tc.err)
			}
		})
	}
}
