package sessionsync

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/xianxu/pair/cmd/internal/testwait"
)

type pausingOperatorFiles struct {
	osOperatorLockFiles
	phase string
}

func (f pausingOperatorFiles) Publish(path string, raw []byte) error {
	if f.phase == "before" {
		if err := os.WriteFile(operatorStagePath(path), []byte("partial private stage"), 0600); err != nil {
			return err
		}
	} else if err := f.osOperatorLockFiles.Publish(path, raw); err != nil {
		return err
	}
	// The parent kills this process only after this marker; no elapsed-time guess.
	fmt.Fprintln(os.Stdout, "operator-publication-paused")
	var byte [1]byte
	_, err := os.Stdin.Read(byte[:])
	return err
}

func TestOperatorLockCrashHelper(t *testing.T) {
	file := os.Getenv("PAIR_OPERATOR_CRASH_FILE")
	if file == "" {
		t.Skip("subprocess helper")
	}
	s := &Sync{Ledger: &Ledger{Dir: os.Getenv("PAIR_OPERATOR_CRASH_STORE")}}
	s.Registry = &registryStore{Dir: s.registryStore().Dir, OperatorLocks: pausingOperatorFiles{phase: os.Getenv("PAIR_OPERATOR_CRASH_PHASE")}}
	_, err := s.AcquireOperator(OperatorLockRequest{File: file, ID: os.Getenv("PAIR_OPERATOR_CRASH_ID"), Turn: modelTurn.Token})
	t.Fatalf("publication helper returned before parent termination: %v", err)
}

func TestOperatorLockProcessDeathReconciles(t *testing.T) {
	for _, phase := range []string{"before", "after"} {
		t.Run(phase, func(t *testing.T) {
			r, req := operatorRig(t)
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestOperatorLockCrashHelper$")
			cmd.Env = append(os.Environ(), "PAIR_OPERATOR_CRASH_FILE="+req.File, "PAIR_OPERATOR_CRASH_STORE="+r.sync.Ledger.Dir, "PAIR_OPERATOR_CRASH_ID="+req.ID, "PAIR_OPERATOR_CRASH_PHASE="+phase)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			joined := false
			t.Cleanup(func() {
				_ = stdin.Close()
				if !joined {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			})
			type output struct {
				line string
				err  error
			}
			observed := make(chan output, 1)
			go func() { line, err := bufio.NewReader(stdout).ReadString('\n'); observed <- output{line, err} }()
			select {
			case got := <-observed:
				if got.err != nil || got.line != "operator-publication-paused\n" {
					t.Fatalf("publication barrier %+v", got)
				}
			case <-testwait.After(t):
				t.Fatal("child publication barrier not reached")
			}
			prior, _ := os.ReadFile(req.File + ".lock")
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			waitErr := cmd.Wait()
			joined = true
			if waitErr == nil {
				t.Fatal("crash helper exited successfully")
			}
			got, err := r.sync.AcquireOperator(req)
			if err != nil {
				t.Fatal(err)
			}
			if phase == "before" {
				if got.State != OperatorAcquired {
					t.Fatalf("partial stage granted ownership: %+v", got)
				}
			} else if got.State != OperatorAlreadyHeld || got.LockText != string(prior) {
				t.Fatalf("lost ack changed exact protection: %+v", got)
			}
			if _, err := os.Stat(operatorStagePath(req.File + ".lock")); !os.IsNotExist(err) {
				t.Fatalf("stage not reclaimed: %v", err)
			}
			if _, err := r.sync.ReleaseOperator(req); err != nil {
				t.Fatal(err)
			}
			_, _ = io.Copy(io.Discard, stdout)
		})
	}
}
