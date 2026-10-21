package operationalrun

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestFinalizeNoticeMonotonicAndPreserving(t *testing.T) {
	for _, final := range []string{"submitted", "expired", "dropped: hub offline", "uncertain"} {
		t.Run(final, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			r := sample("one", time.Now().UTC())
			r.Notice = "pending"
			r.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{ "keep": true }`)}
			r.Key.Unknown = map[string]json.RawMessage{"future_key": json.RawMessage(`42`)}
			if err := s.Publish(t.Context(), r.Key, r, []byte("immutable output")); err != nil {
				t.Fatal(err)
			}
			task, _ := s.task(r.Key)
			out := filepath.Join(task, "runs", r.Key.ID, "output")
			before, err := os.Stat(out)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if err := s.FinalizeNotice(t.Context(), r.Key, "rev", final); err != nil {
					t.Fatal(err)
				}
			}
			got, output, err := s.Read(t.Context(), r.Key)
			if err != nil {
				t.Fatal(err)
			}
			r.Notice = final
			if !reflect.DeepEqual(got, r) || string(output) != "immutable output" {
				t.Fatalf("changed execution/unknown/output: %#v %q", got, output)
			}
			after, _ := os.Stat(out)
			if !os.SameFile(before, after) || before.ModTime() != after.ModTime() {
				t.Fatal("output rewritten")
			}
			other := "submitted"
			if final == other {
				other = "uncertain"
			}
			if err := s.FinalizeNotice(t.Context(), r.Key, "rev", other); err == nil {
				t.Fatal("changed terminal notice")
			}
		})
	}
}

func TestFinalizeNoticeRefusesInvalidTransitions(t *testing.T) {
	for _, kind := range []string{"revision", "running", "uncertain execution", "not pending", "bad terminal", "empty drop"} {
		t.Run(kind, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			r := sample("one", time.Now().UTC())
			r.Notice = "pending"
			rev, notice := "rev", "submitted"
			switch kind {
			case "revision":
				rev = "stale"
			case "running":
				r.EndedAt = time.Time{}
				r.Outcome = "running"
			case "uncertain execution":
				r.Outcome = "uncertain"
			case "not pending":
				r.Notice = "manual"
			case "bad terminal":
				notice = "pending"
			case "empty drop":
				notice = "dropped: "
			}
			if err := s.Publish(t.Context(), r.Key, r, []byte("keep")); err != nil {
				t.Fatal(err)
			}
			before, err := s.readMetadata(r.Key)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.FinalizeNotice(t.Context(), r.Key, rev, notice); err == nil {
				t.Fatal("accepted invalid transition")
			}
			got, _, err := s.Read(t.Context(), r.Key)
			if err != nil || !reflect.DeepEqual(got, before) {
				t.Fatal("changed refused record", got, err)
			}
		})
	}
}

func TestPendingNoticeProtectedUntilTerminal(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	now := time.Now().UTC()
	r := sample("one", now.Add(-2*RetentionPeriod))
	r.Notice = "pending"
	if err := s.Publish(t.Context(), r.Key, r, nil); err != nil {
		t.Fatal(err)
	}
	rows, _, _, err := s.SweepPage(t.Context(), now, Cursor{}, true, 100)
	if err != nil || len(rows) != 1 || rows[0].Eligible {
		t.Fatal(rows, err)
	}
	if _, _, err := s.Read(t.Context(), r.Key); err != nil {
		t.Fatal("pending collected", err)
	}
	if err := s.FinalizeNotice(t.Context(), r.Key, "rev", "uncertain"); err != nil {
		t.Fatal(err)
	}
	rows, _, _, err = s.SweepPage(t.Context(), now, Cursor{}, true, 100)
	if err != nil || len(rows) != 1 || !rows[0].Eligible {
		t.Fatal(rows, err)
	}
}

func TestFinalizeNoticeCrashHelper(t *testing.T) {
	step := os.Getenv("NOTICE_CRASH_STEP")
	if step == "" {
		return
	}
	s := Store{Namespace: os.Getenv("NOTICE_NAMESPACE"), fault: func(at string) error {
		if at == step {
			os.Exit(73)
		}
		return nil
	}}
	_ = s.FinalizeNotice(t.Context(), RunKey{Hub: "hub", Task: "task", ID: "one"}, "rev", "submitted")
	os.Exit(74)
}
func TestFinalizeNoticeAbruptRecovery(t *testing.T) {
	for _, step := range []string{"notice-ready", "notice-renamed"} {
		t.Run(step, func(t *testing.T) {
			s := Store{Namespace: t.TempDir()}
			r := sample("one", time.Now().UTC())
			r.Notice = "pending"
			if err := s.Publish(t.Context(), r.Key, r, []byte("keep")); err != nil {
				t.Fatal(err)
			}
			program, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(t.Context(), program, "-test.run=^TestFinalizeNoticeCrashHelper$")
			cmd.Env = append(os.Environ(), "NOTICE_CRASH_STEP="+step, "NOTICE_NAMESPACE="+s.Namespace)
			err = cmd.Run()
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 73 {
				t.Fatal("helper did not crash at boundary", err)
			}
			got, output, err := s.Read(t.Context(), r.Key)
			if err != nil || string(output) != "keep" {
				t.Fatal(got, string(output), err)
			}
			want := "pending"
			if step == "notice-renamed" {
				want = "submitted"
			}
			if got.Notice != want {
				t.Fatal(got.Notice, want)
			}
			if err := s.FinalizeNotice(t.Context(), r.Key, "rev", "submitted"); err != nil {
				t.Fatal(err)
			}
			task, _ := s.task(r.Key)
			entries, err := os.ReadDir(task)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if stageName.MatchString(entry.Name()) {
					t.Fatal("stage retained", entry.Name())
				}
			}
		})
	}
}

func TestFinalizeNoticeHoldsTaskLock(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	r := sample("one", time.Now().UTC())
	r.Notice = "pending"
	if err := s.Publish(t.Context(), r.Key, r, nil); err != nil {
		t.Fatal(err)
	}
	ready, release := make(chan struct{}), make(chan struct{})
	s.fault = func(step string) error {
		if step == "notice-ready" {
			close(ready)
			<-release
		}
		return nil
	}
	done := make(chan error, 1)
	go func() { done <- s.FinalizeNotice(t.Context(), r.Key, "rev", "submitted") }()
	<-ready
	other := Store{Namespace: s.Namespace}
	err := other.FinalizeNotice(t.Context(), r.Key, "rev", "uncertain")
	close(release)
	if !errors.Is(err, ErrBusy) {
		t.Fatal("concurrent finalizer acquired task", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := other.FinalizeNotice(t.Context(), r.Key, "rev", "uncertain"); err == nil {
		t.Fatal("changed committed terminal")
	}
}

func TestPendingNoticeDoesNotConsumeRetentionCount(t *testing.T) {
	s := Store{Namespace: t.TempDir()}
	now := time.Now().UTC()
	for i := 0; i < MaxRunsPerTask; i++ {
		r := sample(fmt.Sprintf("known-%03d", i), now.Add(-time.Hour))
		if err := s.Publish(t.Context(), r.Key, r, nil); err != nil {
			t.Fatal(err)
		}
	}
	r := sample("pending", now)
	r.Notice = "pending"
	if err := s.Publish(t.Context(), r.Key, r, nil); err != nil {
		t.Fatal(err)
	}
	cursor := Cursor{}
	for {
		rows, next, done, err := s.SweepPage(t.Context(), now, cursor, false, 100)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if row.Eligible {
				t.Fatal("pending delivery consumed completed-run allowance", row)
			}
		}
		if done {
			break
		}
		cursor = next
	}
}
