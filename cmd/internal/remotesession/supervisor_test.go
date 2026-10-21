package remotesession

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
)

func supervisorFixture(t *testing.T, tunnel *broadcast.FakeTunnel) (*Supervisor, func() error) {
	t.Helper()
	host, err := broadcast.NewProtectedHost(&broadcast.FakeProtectedTunnel{FakeTunnel: tunnel})
	if err != nil {
		t.Fatal(err)
	}
	s, reset, err := NewSupervisor(SupervisorConfig{Namespace: t.TempDir(), Origin: "https://remote.example.com", Host: host})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.Off()
		if err := s.Join(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return s, reset
}
func awaitRemote(t *testing.T, s *Supervisor, status string) {
	t.Helper()
	ctx := t.Context()
	for {
		if s.Snapshot().Status == status {
			return
		}
		select {
		case <-s.Changes():
		case <-ctx.Done():
			t.Fatalf("wanted %s, got %+v", status, s.Snapshot())
		}
	}
}
func TestSupervisorDefaultOffAndRotate(t *testing.T) {
	f := &broadcast.FakeTunnel{}
	s, _ := supervisorFixture(t, f)
	if s.Snapshot().Status != "off" || s.ConsoleLink() != "" || f.Stats().Opens != 0 {
		t.Fatal("default off exposed transport")
	}
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	first := s.ConsoleLink()
	if !strings.HasPrefix(first, "https://remote.example.com/remote/") {
		t.Fatal("missing console capability")
	}
	s.Off()
	awaitRemote(t, s, "off")
	if s.ConsoleLink() != "" {
		t.Fatal("off retained link")
	}
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	if s.ConsoleLink() == first {
		t.Fatal("re-enable retained old capability")
	}
}
func TestSupervisorKillFencesLateOpenAndResetWaits(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	f := &broadcast.FakeTunnel{OpenStarted: started, OpenRelease: release, IgnoreCancel: true}
	s, reset := supervisorFixture(t, f)
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	<-started
	s.Kill()
	if s.ConsoleLink() != "" || s.Snapshot().Status != "disabled" {
		t.Fatal("kill did not fence synchronously")
	}
	if err := reset(); err == nil {
		t.Fatal("reset before late handle joined")
	}
	if err := s.On(); err == nil {
		t.Fatal("kill reset through on")
	}
	close(release)
	if err := s.Join(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !s.Snapshot().CanReset {
		t.Fatalf("confirmed teardown cannot reset: %+v", s.Snapshot())
	}
	if f.Stats().Closes != 1 {
		t.Fatalf("late handle not disposed: %+v", f.Stats())
	}
	if err := reset(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "off")
	if f.Stats().Opens != 1 {
		t.Fatal("reset automatically reopened")
	}
}
func TestSupervisorCaptureRefusal(t *testing.T) {
	s, _ := supervisorFixture(t, &broadcast.FakeTunnel{})
	s.captureConfigured = true
	err := s.On()
	if err == nil || !strings.Contains(err.Error(), "unset COUCH_CAPTURE_DIR and restart Couch, or keep capture and skip remote") {
		t.Fatalf("wrong capture recovery: %v", err)
	}
	if s.Snapshot().Status != "off" {
		t.Fatal("capture refusal armed state")
	}
}
func TestSupervisorLatchFailureNeverOpens(t *testing.T) {
	f := &broadcast.FakeTunnel{}
	s, reset := supervisorFixture(t, f)
	s.store.publish = func(string, []byte) error { return errors.New("disk full") }
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "blocked")
	if err := s.Join(t.Context()); err != nil {
		t.Fatal(err)
	}
	if f.Stats().Opens != 0 {
		t.Fatal("opened without synced arm")
	}
	if reset() == nil {
		t.Fatal("uncertain publication reset")
	}
}

func TestSupervisorResetThenKillWritesInAcceptedOrder(t *testing.T) {
	s, reset := supervisorFixture(t, &broadcast.FakeTunnel{})
	s.Kill()
	if err := s.Join(t.Context()); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	original := s.store.publish
	s.store.publish = func(path string, raw []byte) error {
		entered <- struct{}{}
		<-release
		return original(path, raw)
	}
	if err := reset(); err != nil {
		t.Fatal(err)
	}
	<-entered
	s.Kill()
	if s.Snapshot().CanReset {
		t.Fatal("reset ahead of queued kill")
	}
	close(release)
	// Drain publication notifications without depending on scheduling.
	<-entered
	if err := s.Join(t.Context()); err != nil {
		t.Fatal(err)
	}
	phase, _ := s.store.Snapshot()
	if phase != latchKilled || s.Snapshot().Status != "disabled" {
		t.Fatal("late reset overrode kill")
	}
}

func TestSupervisorOneTransportPerEpochAtNAnd2N(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			tunnel := &broadcast.FakeTunnel{}
			s, _ := supervisorFixture(t, tunnel)
			for range n {
				if err := s.On(); err != nil {
					t.Fatal(err)
				}
				awaitRemote(t, s, "on")
				s.Off()
				awaitRemote(t, s, "off")
			}
			stats := tunnel.Stats()
			if stats.Opens != n || stats.Closes != n || stats.DoubleCloses != 0 || stats.MaxOpen != 1 {
				t.Fatalf("N=%d stats=%+v", n, stats)
			}
		})
	}
}
func TestSupervisorCleanupErrorNeverResets(t *testing.T) {
	closeStarted := make(chan struct{}, 1)
	tunnel := &broadcast.FakeTunnel{CloseError: errors.New("cleanup uncertain"), CloseStarted: closeStarted}
	host, err := broadcast.NewProtectedHost(&broadcast.FakeProtectedTunnel{FakeTunnel: tunnel})
	if err != nil {
		t.Fatal(err)
	}
	s, reset, err := NewSupervisor(SupervisorConfig{Namespace: t.TempDir(), Origin: "https://remote.example.com", Host: host})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.On(); err != nil {
		t.Fatal(err)
	}
	awaitRemote(t, s, "on")
	s.Kill()
	if err := s.Join(t.Context()); err == nil {
		t.Fatal("uncertain cleanup claimed confirmed")
	}
	if s.Snapshot().Status != "blocked" || s.Snapshot().CanReset || reset() == nil || s.On() == nil {
		t.Fatal("uncertain cleanup regained authority")
	}
	select {
	case <-closeStarted:
	default:
		t.Fatal("cleanup did not attempt handle close")
	}
}
