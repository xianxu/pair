package testoracle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func oracleRequest(n int) []byte {
	b, _ := json.Marshal(map[string]any{"Cols": 80, "Rows": 24, "Chunks": []string{strings.Repeat("界\x1b[31m\n", n)}})
	return b
}

// This process is the controlled oracle, reached through a shim that accepts
// Node's flags. It never invokes Node or any native terminal application.
func TestOracleChild(t *testing.T) {
	if os.Getenv("ORACLE_CHILD") != "1" {
		t.Skip("subprocess only")
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(9)
	}
	cwd, _ := os.Getwd()
	record := struct {
		Input    []byte
		Dir, Env string
	}{input, cwd, os.Getenv("ORACLE_VALUE")}
	b, _ := json.Marshal(record)
	if os.WriteFile(os.Getenv("ORACLE_RECORD"), b, 0600) != nil {
		os.Exit(9)
	}
	if os.Getenv("ORACLE_MODE") == "report" {
		if os.WriteFile(filepath.Join(os.Getenv("ORACLE_REPORT_DIR"), "node-report.json"), []byte("owned-report"), 0600) != nil {
			os.Exit(9)
		}
	}
	fmt.Fprint(os.Stdout, `[{"Lines":["ok"]}]`)
	fmt.Fprint(os.Stderr, "oracle phase=complete\n")
	switch os.Getenv("ORACLE_MODE") {
	case "exit":
		os.Exit(7)
	case "signal":
		_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
		os.Exit(9)
	}
	os.Exit(0)
}

func TestRunFixture(t *testing.T) {
	mode := os.Getenv("ORACLE_FIXTURE")
	if mode == "" {
		t.Skip("subprocess only")
	}
	dir := os.Getenv("ORACLE_WORK")
	shim := filepath.Join(dir, "node-shim")
	// Save all arguments before stripping precisely the six Node options.
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$ORACLE_ARGS\"\nexport ORACLE_REPORT_DIR=\"${3#--report-directory=}\"\nshift 6\nexec \"$@\"\n"
	if err := os.WriteFile(shim, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(shim, os.Args[0], "-test.run=^TestOracleChild$")
	cmd.Dir = dir
	cmd.Env = []string{"ORACLE_CHILD=1", "ORACLE_MODE=" + mode, "ORACLE_VALUE=private", "ORACLE_RECORD=" + filepath.Join(dir, "record.json"), "ORACLE_ARGS=" + filepath.Join(dir, "args")}
	if mode == "canceled" {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		canceled := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		canceled.Dir, canceled.Env = cmd.Dir, cmd.Env
		Run(t, canceled, oracleRequest(1024))
		t.Fatal("canceled command returned successfully")
	}
	input := oracleRequest(1024)
	if os.Getenv("ORACLE_DOUBLE") == "1" {
		input = oracleRequest(2048)
	}
	wantPath, wantDir, wantEnv := cmd.Path, cmd.Dir, append([]string(nil), cmd.Env...)
	out := Run(t, cmd, input)
	if cmd.Path != wantPath || cmd.Dir != wantDir || !reflect.DeepEqual(cmd.Env, wantEnv) {
		t.Fatal("caller command configuration changed")
	}
	if string(out) != `[{"Lines":["ok"]}]` {
		t.Fatalf("stdout polluted: %q", out)
	}
	if mode == "success" {
		first, err := os.ReadFile(filepath.Join(dir, "args"))
		if err != nil {
			t.Fatal(err)
		}
		second := exec.Command(shim, os.Args[0], "-test.run=^TestOracleChild$")
		second.Dir, second.Env = cmd.Dir, cmd.Env
		Run(t, second, input)
		last, err := os.ReadFile(filepath.Join(dir, "args"))
		if err != nil || bytes.Equal(first, last) {
			t.Fatalf("invocations shared report directory: %v", err)
		}
		firstDir := strings.TrimPrefix(strings.Split(string(first), "\n")[2], "--report-directory=")
		if err := os.WriteFile(filepath.Join(dir, "first-directory"), []byte(firstDir), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Mutating the caller's buffer cannot corrupt deferred evidence.
	for i := range input {
		input[i] = 'x'
	}
	if mode == "later" || mode == "report" {
		t.Error("intentional caller frame assertion failure")
	}
}

func TestRunEvidence(t *testing.T) {
	for _, mode := range []string{"success", "exit", "signal", "later", "report"} {
		t.Run(mode, func(t *testing.T) {
			for _, n := range []int{1024, 2048} {
				t.Run(fmt.Sprint(n), func(t *testing.T) {
					work, artifacts := t.TempDir(), t.TempDir()
					cmd := exec.Command(os.Args[0], "-test.run=^TestRunFixture$", "-test.v", "-test.artifacts", "-test.outputdir="+artifacts)
					cmd.Env = append(os.Environ(), "ORACLE_FIXTURE="+mode, "ORACLE_WORK="+work, fmt.Sprintf("ORACLE_DOUBLE=%d", n/1024-1))
					log, err := cmd.CombinedOutput()
					if (err != nil) != (mode != "success") {
						t.Fatalf("fixture exit=%v output=%s", err, log)
					}
					if mode == "exit" && !bytes.Contains(log, []byte("exit status 7")) {
						t.Fatalf("ordinary exit status lost: %s", log)
					}
					if mode == "signal" && !bytes.Contains(log, []byte("signal: killed")) {
						t.Fatalf("abnormal termination evidence lost: %s", log)
					}
					var record struct {
						Input    []byte
						Dir, Env string
					}
					b, err := os.ReadFile(filepath.Join(work, "record.json"))
					if err != nil {
						t.Fatalf("record: %v; output=%s", err, log)
					}
					if err := json.Unmarshal(b, &record); err != nil {
						t.Fatal(err)
					}
					want := oracleRequest(n)
					physicalWork, err := filepath.EvalSymlinks(work)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(record.Input, want) || record.Dir != physicalWork || record.Env != "private" {
						t.Fatalf("child request/configuration mismatch: bytes=%d want=%d dir=%s env=%s", len(record.Input), len(want), record.Dir, record.Env)
					}
					args, err := os.ReadFile(filepath.Join(work, "args"))
					if err != nil {
						t.Fatal(err)
					}
					lines := strings.Split(strings.TrimSpace(string(args)), "\n")
					if len(lines) != 8 || lines[0] != "--report-on-fatalerror" || lines[1] != "--report-on-signal" || lines[3] != "--report-filename=node-report.json" || lines[4] != "--report-exclude-env" || lines[5] != "--report-exclude-network" || lines[6] != os.Args[0] {
						t.Fatalf("Node flags/script arguments: %q", lines)
					}
					dir := strings.TrimPrefix(lines[2], "--report-directory=")
					rel, err := filepath.Rel(artifacts, dir)
					if err != nil || strings.HasPrefix(rel, "..") || !strings.HasPrefix(filepath.Base(dir), "node-oracle-") {
						t.Fatalf("report outside owned artifact directory: %s", dir)
					}
					if mode == "success" {
						if _, err := os.Stat(dir); !os.IsNotExist(err) {
							t.Fatalf("successful evidence survived: %v", err)
						}
						first, err := os.ReadFile(filepath.Join(work, "first-directory"))
						if err != nil {
							t.Fatal(err)
						}
						if _, err := os.Stat(string(first)); !os.IsNotExist(err) {
							t.Fatalf("first successful invocation survived: %v", err)
						}
						return
					}
					for name, want := range map[string][]byte{"request.json": want, "stdout.raw": []byte(`[{"Lines":["ok"]}]`), "stderr.txt": []byte("oracle phase=complete\n")} {
						got, err := os.ReadFile(filepath.Join(dir, name))
						if err != nil || !bytes.Equal(got, want) {
							t.Errorf("%s retained bytes=%d want=%d err=%v", name, len(got), len(want), err)
						}
					}
					if !bytes.Contains(log, want) || !bytes.Contains(log, []byte(filepath.Join(work, "node-shim"))) || !bytes.Contains(log, []byte("oracle phase=complete")) {
						t.Errorf("failure log lost full request/path/phases: %s", log)
					}
					report, err := os.ReadFile(filepath.Join(dir, "node-report.json"))
					if mode == "report" {
						if err != nil || string(report) != "owned-report" {
							t.Fatalf("report lost: %q %v", report, err)
						}
					} else if !os.IsNotExist(err) {
						t.Fatalf("missing report invented: %q %v", report, err)
					}
					t.Logf("mode=%s complete request bytes=%d stderr bytes=%d", mode, len(want), len("oracle phase=complete\n"))
				})
			}
		})
	}
}

func TestRunCanceledContext(t *testing.T) {
	work, artifacts := t.TempDir(), t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunFixture$", "-test.v", "-test.artifacts", "-test.outputdir="+artifacts)
	cmd.Env = append(os.Environ(), "ORACLE_FIXTURE=canceled", "ORACLE_WORK="+work)
	out, err := cmd.CombinedOutput()
	if err == nil || !bytes.Contains(out, []byte("context canceled")) {
		t.Fatalf("original context lost: %v %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(work, "record.json")); !os.IsNotExist(err) {
		t.Fatalf("canceled child was started: %v", err)
	}
	var found int
	if err := filepath.WalkDir(artifacts, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == "request.json" {
			found++
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, oracleRequest(1024)) {
				t.Errorf("cancellation request lost: %v", err)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if found != 1 {
		t.Fatalf("cancellation evidence directories=%d", found)
	}
}

func TestRunTemporaryArtifactLifetime(t *testing.T) {
	work := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunFixture$", "-test.v")
	cmd.Env = append(os.Environ(), "ORACLE_FIXTURE=later", "ORACLE_WORK="+work)
	out, err := cmd.CombinedOutput()
	if err == nil || !bytes.Contains(out, oracleRequest(1024)) {
		t.Fatalf("failed invocation omitted diagnostic: %v %s", err, out)
	}
	args, err := os.ReadFile(filepath.Join(work, "args"))
	if err != nil {
		t.Fatal(err)
	}
	dir := strings.TrimPrefix(strings.Split(string(args), "\n")[2], "--report-directory=")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("unpreserved artifact survived Go invocation: %s %v", dir, err)
	}
}
