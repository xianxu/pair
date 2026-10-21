package sessionsync

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

func operatorRig(t *testing.T) (*rig, OperatorLockRequest) {
	t.Helper()
	r := newRig(t)
	registration, err := r.sync.Register(r.p.File, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	return r, OperatorLockRequest{File: r.p.File, ID: registration.Registration.ID, Turn: modelTurn.Token}
}

func TestOperatorLockLifecycle(t *testing.T) {
	r, req := operatorRig(t)
	before := r.read(r.p.File)
	first, err := r.sync.AcquireOperator(req)
	if err != nil || first.State != OperatorAcquired || first.Holder != HolderOperator || first.ID != req.ID || first.Turn != req.Turn {
		t.Fatalf("acquire %+v %v", first, err)
	}
	if first.LockText != r.read(first.LockPath) {
		t.Fatal("ack did not witness published bytes")
	}
	proof, ok := parseOperatorProof(first.LockText)
	if !ok || proof != (operatorTurn{ID: req.ID, Token: req.Turn}) {
		t.Fatalf("unrecoverable proof: %q", first.LockText)
	}
	r.now = r.now.Add(time.Hour)
	retry, err := r.sync.AcquireOperator(req)
	if err != nil || retry.State != OperatorAlreadyHeld || retry.LockText != first.LockText || retry.LockPath != first.LockPath {
		t.Fatalf("retry changed ownership %+v %v", retry, err)
	}
	foreign := req
	foreign.Turn = strings.Repeat("a", 32)
	_, err = r.sync.ReleaseOperator(foreign)
	wantRefusal(t, err, 3, "operator-lock")
	if r.read(first.LockPath) != first.LockText || r.read(r.p.File) != before {
		t.Fatal("foreign release changed protected files")
	}
	released, err := r.sync.ReleaseOperator(req)
	if err != nil || released.State != OperatorReleased || released.Holder != HolderNone || released.LockText != "" || released.ID != req.ID || released.Turn != req.Turn {
		t.Fatalf("release %+v %v", released, err)
	}
	released, err = r.sync.ReleaseOperator(req)
	if err != nil || released.State != OperatorAlreadyReleased {
		t.Fatalf("release retry %+v %v", released, err)
	}
	if _, err := r.sync.AcquireOperator(foreign); err != nil {
		t.Fatal(err)
	}
	_, err = r.sync.ReleaseOperator(req)
	wantRefusal(t, err, 3, "operator-lock")
}

func TestOperatorLockAliasRecoveryAcknowledgesCanonicalWitness(t *testing.T) {
	r, req := operatorRig(t)
	alias := filepath.Join(filepath.Dir(req.File), "operator-alias.md")
	if err := os.Symlink(req.File, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Register(alias, "ops:0"); err != nil {
		t.Fatal(err)
	}
	text := operatorLockText(operatorTurn{ID: req.ID, Token: req.Turn}, r.now) + "future: retained alias proof\n"
	if err := os.WriteFile(alias+".lock", []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := r.sync.AcquireOperator(req)
	if err != nil || got.State != OperatorAlreadyHeld || got.LockPath != got.File+".lock" || got.LockText != text || r.read(got.LockPath) != text {
		t.Fatalf("noncanonical or changed witness: %+v %v", got, err)
	}
	if r.read(alias+".lock") != text {
		t.Fatal("canonicalization stole alias lifecycle cleanup")
	}
	canonicalText := text + "canonical: preserve this newer unknown line\n"
	if err := os.WriteFile(got.LockPath, []byte(canonicalText), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = r.sync.AcquireOperator(req)
	if err != nil || got.LockText != canonicalText || got.LockPath != got.File+".lock" || r.read(got.LockPath) != canonicalText {
		t.Fatalf("canonical proof overwritten by alias: %+v %v", got, err)
	}
}

func TestOperatorLockWaitsForEntireWriterTransaction(t *testing.T) {
	for _, apply := range []bool{false, true} {
		t.Run(map[bool]string{false: "write", true: "apply"}[apply], func(t *testing.T) {
			r, req := operatorRig(t)
			hash := r.hash("")
			paused, resume, contention := make(chan struct{}), make(chan struct{}), make(chan struct{})
			roundPaused, roundResume, roundContention := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var releaseOnce, contentionOnce, roundReleaseOnce, roundContentionOnce sync.Once
			var workers sync.WaitGroup
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(resume) })
				roundReleaseOnce.Do(func() { close(roundResume) })
				workers.Wait()
			})
			await := func(ch <-chan struct{}) {
				t.Helper()
				select {
				case <-ch:
				case <-testwait.After(t):
					t.Fatal("ownership ordering barrier not reached")
				}
			}
			writer := *r.sync
			clockCalls := 0
			writer.Now = func() time.Time {
				clockCalls++
				if clockCalls == 2 {
					close(paused)
					<-resume
				}
				return r.now
			}
			writer.RoundWriter = func(path string, raw []byte) error {
				if err := writeAtomic(path, raw); err != nil {
					return err
				}
				close(roundPaused)
				<-roundResume
				return nil
			}
			acquirer := *r.sync
			acquirer.Registry = &registryStore{Dir: r.sync.registryStore().Dir, TryLock: func(fd int) error {
				err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
				if errors.Is(err, syscall.EWOULDBLOCK) {
					contentionOnce.Do(func() { close(contention) })
					select {
					case <-roundPaused:
						roundContentionOnce.Do(func() { close(roundContention) })
					default:
					}
				}
				return err
			}}
			wrote := make(chan error, 1)
			workers.Add(1)
			go func() {
				defer workers.Done()
				var err error
				if apply {
					_, err = writer.Apply(ApplyRequest{File: req.File, IfHash: hash, Edits: []EditRecord{{Old: "trains", New: "committed", Occurrence: 1}}})
				} else {
					_, err = writer.Write(WriteRequest{File: req.File, IfHash: hash, Body: strings.Replace(statusDoc, "trains", "committed", 1)})
				}
				wrote <- err
			}()
			await(paused)
			ack := make(chan error, 1)
			workers.Add(1)
			go func() { defer workers.Done(); _, err := acquirer.AcquireOperator(req); ack <- err }()
			await(contention)
			select {
			case err := <-ack:
				t.Fatalf("acquire crossed paused writer: %v", err)
			default:
			}
			releaseOnce.Do(func() { close(resume) })
			await(roundPaused)
			select {
			case <-roundContention:
			case err := <-ack:
				t.Fatalf("acquire crossed writer's final round effect: %v", err)
			case <-testwait.After(t):
				t.Fatal("final writer effect did not retain registry exclusion")
			}
			roundReleaseOnce.Do(func() { close(roundResume) })
			if err := <-ack; err != nil {
				t.Fatal(err)
			}
			if err := <-wrote; err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(r.read(req.File), "committed") {
				t.Fatal("earlier writer's completed bytes missing at ack")
			}
			central := filepath.Join(r.sync.registryStore().Dir, "state", req.ID)
			paths := []string{req.File, filepath.Join(central, "shadow"), filepath.Join(central, "state.json"), filepath.Join(central, "round.json")}
			snapshot := func() []string {
				out := make([]string, len(paths))
				for i, p := range paths {
					raw, err := os.ReadFile(p)
					if err != nil && !errors.Is(err, os.ErrNotExist) {
						t.Fatal(err)
					}
					out[i] = string(raw)
				}
				return out
			}
			before := snapshot()
			_, err := r.sync.Write(WriteRequest{File: req.File, IfHash: Hash(before[0]), Body: "must not commit"})
			wantRefusal(t, err, 3, "operator-lock")
			_, err = r.sync.Apply(ApplyRequest{File: req.File, IfHash: Hash(before[0]), Edits: []EditRecord{{Old: "committed", New: "must not commit", Occurrence: 1}}})
			wantRefusal(t, err, 3, "operator-lock")
			if !reflect.DeepEqual(before, snapshot()) {
				t.Fatal("future writer changed document/metadata after ack")
			}
		})
	}
}

func TestOperatorLockMovesWithJournalBytes(t *testing.T) {
	r, req := operatorRig(t)
	owned, err := r.sync.AcquireOperator(req)
	if err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(filepath.Dir(req.File), "moved.md")
	if err := os.WriteFile(newPath+".lock", []byte(owned.LockText), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(req.File, newPath); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Rename(req.File, newPath); err != nil {
		t.Fatal(err)
	}
	proof, ok := parseOperatorProof(owned.LockText)
	if !ok {
		t.Fatal("move journal cannot recover turn")
	}
	newReq := OperatorLockRequest{File: newPath, ID: req.ID, Turn: proof.Token}
	retry, err := r.sync.AcquireOperator(newReq)
	if err != nil || retry.State != OperatorAlreadyHeld || retry.LockText != owned.LockText {
		t.Fatalf("moved ownership %+v %v", retry, err)
	}
	if _, err := r.sync.ReleaseOperator(newReq); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(newPath + ".lock"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new lock remained: %v", err)
	}
	// OLD belongs to the existing move journal, never a stale current-path release.
	if r.read(req.File+".lock") != owned.LockText {
		t.Fatal("producer destroyed journal-owned old protection")
	}
	if _, err := r.sync.Forget(newPath); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Register(newPath, "ops:0"); err != nil {
		t.Fatal(err)
	}
	_, err = r.sync.AcquireOperator(newReq)
	wantRefusal(t, err, 1, "operator-identity")
}
