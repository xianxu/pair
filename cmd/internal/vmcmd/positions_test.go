//go:build darwin || linux

package vmcmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

func positionFixture(t *testing.T, lane Lane) (*Position, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "positions")
	p, err := AcquirePosition(context.Background(), root, lane, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	return p, root
}
func positionClaim(lane Lane) LaunchClaim {
	return LaunchClaim{Version: 1, Lane: lane, VM: ManagedVMName(lane), Nonce: strings.Repeat("a", 32), Stage: ClaimLaunching}
}
func TestPositionClaimCAS(t *testing.T) {
	p, _ := positionFixture(t, TrainLane)
	if _, ok, err := p.ReadClaim(); err != nil || ok {
		t.Fatal(ok, err)
	}
	c := positionClaim(TrainLane)
	c.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`{"n":1}`)}
	if err := p.WriteClaim(c); err != nil {
		t.Fatal(err)
	}
	c.Unknown = nil
	c.Stage = ClaimLaunched
	c.PID = 123
	if err := p.WriteClaim(c); err != nil {
		t.Fatal(err)
	}
	got, ok, err := p.ReadClaim()
	if err != nil || !ok || string(got.Unknown["future"]) != `{"n":1}` {
		t.Fatalf("%+v %v %v", got, ok, err)
	}
	before, err := os.ReadFile(filepath.Join(p.Dir(), "claim.json"))
	if err != nil {
		t.Fatal(err)
	}
	c.Nonce = strings.Repeat("b", 32)
	if err := p.WriteClaim(c); err == nil {
		t.Fatal("overwrote other generation")
	}
	if err := p.ClearClaim(c.Nonce); err == nil {
		t.Fatal("cleared other generation")
	}
	after, _ := os.ReadFile(filepath.Join(p.Dir(), "claim.json"))
	if string(after) != string(before) {
		t.Fatal("failed CAS changed claim")
	}
	if err := p.ClearClaim(got.Nonce); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := p.ReadClaim(); ok || err != nil {
		t.Fatal(ok, err)
	}
}
func TestPositionPreservesInvalidClaim(t *testing.T) {
	for _, raw := range []string{`{"version":1,"version":1}`, strings.Repeat(" ", 65537), `null`} {
		t.Run(raw[:min(len(raw), 20)], func(t *testing.T) {
			p, _ := positionFixture(t, AdhocLane)
			path := filepath.Join(p.Dir(), "claim.json")
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := p.ReadClaim(); err == nil {
				t.Fatal("read malformed claim")
			}
			if err := p.WriteClaim(positionClaim(AdhocLane)); err == nil {
				t.Fatal("overwrote malformed claim")
			}
			if err := p.ClearClaim(strings.Repeat("a", 32)); err == nil {
				t.Fatal("cleared malformed claim")
			}
			got, _ := os.ReadFile(path)
			if string(got) != raw {
				t.Fatal("changed malformed claim")
			}
		})
	}
}

type positionProgress struct {
	once    sync.Once
	waiting chan struct{}
}

func (w *positionProgress) Write(b []byte) (int, error) {
	w.once.Do(func() { close(w.waiting) })
	return len(b), nil
}
func TestPositionExclusionCancellationAndStableLock(t *testing.T) {
	p, root := positionFixture(t, TrainLane)
	lock := filepath.Join(p.Dir(), "position.lock")
	before, err := os.Stat(lock)
	if err != nil {
		t.Fatal(err)
	}
	other, err := AcquirePosition(context.Background(), root, AdhocLane, nil)
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	progress := &positionProgress{waiting: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		q, e := AcquirePosition(ctx, root, TrainLane, progress)
		if q != nil {
			q.Close()
		}
		done <- e
	}()
	select {
	case <-progress.waiting:
	case <-testwait.After(t):
		t.Fatal("no contention observation")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-testwait.After(t):
		t.Fatal("cancel stuck")
	}
	p.Close()
	p.Close()
	q, err := AcquirePosition(context.Background(), root, TrainLane, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	after, err := os.Stat(lock)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("lock inode changed", err)
	}
	if err := p.WriteClaim(positionClaim(TrainLane)); err == nil {
		t.Fatal("closed position accepted write")
	}
}
func TestPositionRejectsAliasesAndInvalidInputs(t *testing.T) {
	for _, kind := range []string{"root", "lane", "lock", "lock-symlink", "claim-symlink", "claim-hardlink"} {
		t.Run(kind, func(t *testing.T) {
			base := t.TempDir()
			root := filepath.Join(base, "positions")
			target := filepath.Join(base, "target")
			if err := os.Mkdir(target, 0700); err != nil {
				t.Fatal(err)
			}
			if kind == "root" {
				if err := os.Symlink(target, root); err != nil {
					t.Fatal(err)
				}
			} else {
				os.Mkdir(root, 0700)
				lane := filepath.Join(root, "train")
				if kind == "lane" {
					os.Symlink(target, lane)
				} else {
					os.Mkdir(lane, 0700)
					file := filepath.Join(target, "file")
					os.WriteFile(file, []byte("{}"), 0600)
					switch kind {
					case "lock-symlink":
						os.Symlink(file, filepath.Join(lane, "position.lock"))
					case "lock":
						os.Link(file, filepath.Join(lane, "position.lock"))
					case "claim-symlink":
						os.Symlink(file, filepath.Join(lane, "claim.json"))
					case "claim-hardlink":
						os.Link(file, filepath.Join(lane, "claim.json"))
					}
				}
			}
			p, err := AcquirePosition(context.Background(), root, TrainLane, nil)
			if strings.HasPrefix(kind, "claim-") {
				if err != nil {
					t.Fatal(err)
				}
				defer p.Close()
				if _, _, err = p.ReadClaim(); err == nil {
					t.Fatal("claim alias accepted")
				}
			} else if err == nil {
				p.Close()
				t.Fatal("alias accepted")
			}
		})
	}
	root := filepath.Join(t.TempDir(), "absent")
	if p, err := AcquirePosition(context.Background(), root, Lane("../bad"), nil); err == nil {
		p.Close()
		t.Fatal("invalid lane accepted")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("invalid lane had effects", err)
	}
	for _, bad := range []string{"", "relative", "/", "/a/../b"} {
		if p, err := AcquirePosition(context.Background(), bad, TrainLane, nil); err == nil {
			p.Close()
			t.Fatal("invalid root", bad)
		}
	}
}

func TestPositionPrivateDirectoriesAndUnrelatedFiles(t *testing.T) {
	base := t.TempDir()
	if err := os.Chmod(base, 0755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "positions")
	p, err := AcquirePosition(context.Background(), root, TrainLane, nil)
	if err != nil {
		t.Fatal(err)
	}
	p.Close()
	for _, path := range []string{root, filepath.Join(root, "train")} {
		st, err := os.Stat(path)
		if err != nil || st.Mode().Perm() != 0700 {
			t.Fatal(path, st, err)
		}
	}
	marker := filepath.Join(root, "unrelated")
	if err := os.WriteFile(marker, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	p, err = AcquirePosition(context.Background(), root, AdhocLane, nil)
	if err != nil {
		t.Fatal(err)
	}
	p.Close()
	raw, err := os.ReadFile(marker)
	if err != nil || string(raw) != "keep" {
		t.Fatal("unrelated file changed", err)
	}
	st, err := os.Stat(base)
	if err != nil || st.Mode().Perm() != 0755 {
		t.Fatal("parent mode changed", err)
	}
	for _, path := range []string{root, filepath.Join(root, "train"), filepath.Join(root, "train", "position.lock")} {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		mode := st.Mode().Perm()
		if err := os.Chmod(path, 0755); err != nil {
			t.Fatal(err)
		}
		q, err := AcquirePosition(context.Background(), root, TrainLane, nil)
		if err == nil {
			q.Close()
			t.Fatal("nonprivate path accepted", path)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPositionRejectsSpecialDirectoryMode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "positions")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0700|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	p, err := AcquirePosition(context.Background(), root, TrainLane, nil)
	if err == nil {
		p.Close()
		t.Fatal("non-0700 directory mode accepted")
	}
}
