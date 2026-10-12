package perfbudget

import (
	"testing"
	"time"
)

type recorder struct {
	testing.TB
	failed bool
}

func (r *recorder) Helper()               {}
func (r *recorder) Logf(string, ...any)   {}
func (r *recorder) Fatalf(string, ...any) { r.failed = true }

func TestCheckFailsOnlyWhenEnforced(t *testing.T) {
	for _, c := range []struct {
		env     string
		elapsed time.Duration
		fail    bool
	}{
		{"", 2 * time.Second, false},
		{"1", 2 * time.Second, true},
		{"1", time.Millisecond, false},
		{"yes", 2 * time.Second, false},
	} {
		t.Setenv(Env, c.env)
		r := &recorder{TB: t}
		Check(r, "x", c.elapsed, time.Second)
		if r.failed != c.fail {
			t.Errorf("%s=%q elapsed %v: failed %v, want %v", Env, c.env, c.elapsed, r.failed, c.fail)
		}
	}
}
