package pairlifecycle

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestLifecycleLockReleasedOnHolderDeath(t *testing.T) {
	paths := osLifecyclePaths(t)
	cmd, stdin, output := startLifecycleHelper(t, "lock", paths.Dir(), paths.Lock())
	expectHelperLine(t, output, "locked")
	before, err := os.Stat(paths.Lock())
	if err != nil {
		t.Fatal(err)
	}
	killHelper(t, cmd, stdin)

	lock, err := (OSRuntime{}).Lock(paths.Lock())
	if err != nil {
		t.Fatalf("acquire after holder death: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(paths.Lock())
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("stable lifecycle lock inode was replaced")
	}
}

func TestLifecycleStoreReconcilesHolderDeathAfterRename(t *testing.T) {
	for _, kind := range []RecordKind{RecordRequest, RecordCompletion} {
		kind := kind
		t.Run(string(kind), func(t *testing.T) {
			paths := osLifecyclePaths(t)
			final := testRecordPath(t, paths, kind)
			cmd, stdin, output := startLifecycleHelper(t, "rename", paths.Dir(), paths.Lock(), final, string(kind))
			expectHelperLine(t, output, "renamed")
			lockBefore, err := os.Stat(paths.Lock())
			if err != nil {
				t.Fatal(err)
			}
			killHelper(t, cmd, stdin)

			store := Store{Runtime: OSRuntime{}}
			if err := store.Reconcile(paths, kind, 1); err != nil {
				t.Fatalf("reconcile prepared final: %v", err)
			}
			if err := store.Reconcile(paths, kind, 1); err != nil {
				t.Fatalf("repeated reconcile: %v", err)
			}
			lockAfter, err := os.Stat(paths.Lock())
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(lockBefore, lockAfter) {
				t.Fatal("reconciliation replaced stable lock inode")
			}
		})
	}
}

func TestLifecycleStoreSubprocessHelper(t *testing.T) {
	if os.Getenv("PAIR_LIFECYCLE_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	args := helperArgs(os.Args)
	if len(args) < 3 {
		t.Fatal("missing helper arguments")
	}
	dir, lockPath := args[1], args[2]
	runtime := OSRuntime{}
	if err := runtime.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	lock, err := runtime.Lock(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()

	switch args[0] {
	case "lock":
		fmt.Println("locked")
	case "rename":
		if len(args) != 5 {
			t.Fatal("missing rename helper arguments")
		}
		kind := RecordKind(args[4])
		var record any = validQuitRequest()
		if kind == RecordCompletion {
			record = validQuitCompletion()
		}
		raw, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, '\n')
		temp, err := runtime.CreateTemp(dir, ".helper-*")
		if err != nil {
			t.Fatal(err)
		}
		if err := writeStoreAll(temp, raw); err != nil {
			t.Fatal(err)
		}
		if err := temp.Sync(); err != nil {
			t.Fatal(err)
		}
		if err := temp.Close(); err != nil {
			t.Fatal(err)
		}
		if err := runtime.Rename(temp.Name(), args[3]); err != nil {
			t.Fatal(err)
		}
		fmt.Println("renamed")
	default:
		t.Fatalf("unknown helper mode %q", args[0])
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func osLifecyclePaths(t *testing.T) artifactpath.LifecyclePaths {
	t.Helper()
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: t.TempDir(), RepoScope: "scope", Tag: "work"})
	if err != nil {
		t.Fatal(err)
	}
	lifecycle, err := paths.Lifecycle("nonce-1")
	if err != nil {
		t.Fatal(err)
	}
	return lifecycle
}

func startLifecycleHelper(t *testing.T, args ...string) (*exec.Cmd, io.WriteCloser, *bufio.Reader) {
	t.Helper()
	cmdArgs := append([]string{"-test.run=^TestLifecycleStoreSubprocessHelper$", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "PAIR_LIFECYCLE_HELPER=1")
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
	return cmd, stdin, bufio.NewReader(stdout)
}

func expectHelperLine(t *testing.T, output *bufio.Reader, want string) {
	t.Helper()
	line, err := output.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if line != want+"\n" {
		t.Fatalf("helper output=%q, want %q", line, want+"\n")
	}
}

func killHelper(t *testing.T, cmd *exec.Cmd, stdin io.Closer) {
	t.Helper()
	defer stdin.Close()
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
}

func helperArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}
	return nil
}
