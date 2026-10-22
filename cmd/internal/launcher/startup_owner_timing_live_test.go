package launcher

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

type startupOwnerIOPhase struct {
	name    string
	elapsed time.Duration
	count   int
}
type startupOwnerIO struct {
	osSessionOwnerIO
	phases []startupOwnerIOPhase
}

func (r *startupOwnerIO) SessionServers(ctx context.Context, name string) ([]SessionServerIdentity, error) {
	start := time.Now()
	value, err := r.osSessionOwnerIO.SessionServers(ctx, name)
	r.phases = append(r.phases, startupOwnerIOPhase{"SessionServers", time.Since(start), len(value)})
	return value, err
}
func (r *startupOwnerIO) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	start := time.Now()
	value, err := r.osSessionOwnerIO.SessionPanes(ctx, name)
	r.phases = append(r.phases, startupOwnerIOPhase{"SessionPanes", time.Since(start), len(value)})
	return value, err
}
func (r *startupOwnerIO) SessionPresent(ctx context.Context, name string) (bool, error) {
	start := time.Now()
	value, err := r.osSessionOwnerIO.SessionPresent(ctx, name)
	r.phases = append(r.phases, startupOwnerIOPhase{"SessionPresent", time.Since(start), 1})
	return value, err
}
func (r *startupOwnerIO) Socket(path string) SocketState {
	start := time.Now()
	value := r.osSessionOwnerIO.Socket(path)
	r.phases = append(r.phases, startupOwnerIOPhase{"Socket", time.Since(start), 1})
	return value
}

// Component measurement: the exact real Probe + Revalidate exercised by a
// managed attach. The end-to-end parent/child measurement is in couchcore.
func TestStartupOwnerTimingLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" || os.Getenv("PAIR_LIVE_STARTUP_PERF") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 PAIR_LIVE_STARTUP_PERF=1")
	}
	testpair.ApplyEnvironment(t, t.TempDir())
	t.Setenv("TMPDIR", "/tmp")
	t.Setenv("SHELL", "/bin/sh")
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		t.Setenv(key, t.TempDir())
	}
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME"} {
		t.Setenv(key, "")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
	defer cancel()
	var seed [8]byte
	if _, err := rand.Read(seed[:]); err != nil {
		t.Fatal(err)
	}
	session := "pair-owner-perf-" + hex.EncodeToString(seed[:])
	fixture, err := pairlifecycletest.StartControlledZellij(ctx, session)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Errorf("controlled zellij cleanup: %v", err)
		}
	})
	repo := filepath.Join(t.TempDir(), "repo")
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	scope, err := ResolveRepoScope(repo)
	if err != nil {
		t.Fatal(err)
	}
	data := t.TempDir()
	tag := "couch-0000000000000001"
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: scope.Key, Tag: tag})
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.AddOwnerWrapperPane(ctx, paths.ScrollbackRaw("claude")); err != nil {
		t.Fatal(err)
	}
	r := &startupOwnerIO{}
	probe := SessionOwnerProbe{IO: r}
	start := time.Now()
	observed, err := probe.Probe(ctx, session, data, scope.Key, tag)
	probeWall := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if observed.State != SessionOwnerOwned {
		t.Fatalf("owner: %+v", observed)
	}
	start = time.Now()
	err = probe.Revalidate(ctx, observed)
	revalidateWall := time.Since(start)
	var measured time.Duration
	for i, row := range r.phases {
		measured += row.elapsed
		t.Logf("owner phase[%d] %s=%s count=%d", i+1, row.name, row.elapsed, row.count)
	}
	t.Logf("owner Probe=%s Revalidate=%s total=%s measured=%s residual=%s", probeWall, revalidateWall, probeWall+revalidateWall, measured, probeWall+revalidateWall-measured)
	if err != nil {
		t.Fatal(err)
	}
}
