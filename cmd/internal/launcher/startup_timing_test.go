package launcher

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestStartupTimingManagedAttachIsOptInAndStopsBeforeBlocking(t *testing.T) {
	for _, enabled := range []string{"", "true", "1"} {
		t.Run(enabled, func(t *testing.T) {
			t.Setenv("PAIR_STARTUP_TIMING", enabled)
			opts := managedSessionOptions("attach")
			var out bytes.Buffer
			rt := &startupTimingAttachRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime(), ownerState: SessionOwnerOwned}}
			rt.sessions = []Session{{Name: opts.Args.CouchSession.Name, State: SessionDetached}}
			rt.beforeAttach = func() {
				if enabled == "1" && !strings.Contains(out.String(), "phase=attach-handoff ") {
					t.Fatal("blocking attach began before final startup mark")
				}
			}
			code, err := RunLaunch(opts, rt, &out)
			if err != nil || code != 0 || len(rt.attached) != 1 {
				t.Fatalf("attach = %d, %v: %s", code, err, &out)
			}
			if enabled != "1" {
				if out.Len() != 0 {
					t.Fatalf("default timing output: %s", &out)
				}
				return
			}
			phases := []string{"launch-entry", "migration", "ancestry", "prelude", "discovery", "decision", "attach-verification", "attach-layout", "attach-retention", "attach-setup", "attach-revalidation", "attach-handoff"}
			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			if len(lines) != len(phases) {
				t.Fatalf("timing phases = %q", lines)
			}
			for i, phase := range phases {
				if !strings.HasPrefix(lines[i], "pair startup: phase="+phase+" delta_ms=") || !strings.Contains(lines[i], " elapsed_ms=") {
					t.Fatalf("phase %d: %s", i, lines[i])
				}
			}
			for _, private := range []string{opts.Args.ForcedTag, opts.Args.CouchSession.Name, opts.Env.DataDir, opts.PairHome} {
				if private != "" && strings.Contains(out.String(), private) {
					t.Fatalf("private value in timing: %q", private)
				}
			}
		})
	}
}

type startupTimingAttachRuntime struct {
	*couchSessionRuntime
	beforeAttach func()
}

func (r *startupTimingAttachRuntime) AttachSession(session, configDir string) (int, error) {
	r.beforeAttach()
	return r.fakeRuntime.AttachSession(session, configDir)
}

func TestStartupTimingDeltasPartitionElapsedAndExcludeClientLifetime(t *testing.T) {
	start := time.Unix(0, 0)
	now := start
	var out bytes.Buffer
	timing := newStartupTiming("1", &out, func() time.Time { return now }, start)
	now = start.Add(2 * time.Millisecond)
	timing.mark("entry")
	now = start.Add(7 * time.Millisecond)
	timing.handoff("attach-handoff")
	before := out.String()
	now = start.Add(time.Hour)
	timing.mark("after-client-exit")
	timing.handoff("restart")
	want := "pair startup: phase=entry delta_ms=2.000 elapsed_ms=2.000\n" +
		"pair startup: phase=attach-handoff delta_ms=5.000 elapsed_ms=7.000\n"
	if out.String() != want || before != want {
		t.Fatalf("timing output = %q, want %q", out.String(), want)
	}
	for _, enabled := range []string{"", "true", "0"} {
		if timing := newStartupTiming(enabled, &out, func() time.Time { t.Fatal("disabled clock read"); return now }, start); timing != nil {
			t.Fatal("disabled timing constructed")
		}
	}
}

func (r *startupTimingAttachRuntime) AttachSessionContext(lifetime, startup, admission context.Context, session, config string) (int, error) {
	if err := admission.Err(); err != nil {
		return 1, err
	}
	return r.AttachSession(session, config)
}
