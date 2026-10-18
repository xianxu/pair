package testoracle

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Run observes one owned Node oracle invocation. Evidence is kept when either
// the process or a later assertion fails; successful calls leave no artifacts.
func Run(t testing.TB, cmd *exec.Cmd, request []byte) []byte {
	t.Helper()
	dir, err := os.MkdirTemp(t.ArtifactDir(), "node-oracle-")
	if err != nil {
		t.Fatal(err)
	}
	input := bytes.Clone(request)
	var stdout, stderr bytes.Buffer
	var runErr error
	t.Cleanup(func() {
		if !t.Failed() {
			if err := os.RemoveAll(dir); err != nil {
				t.Errorf("remove oracle evidence: %v", err)
			}
			return
		}
		for name, data := range map[string][]byte{
			"request.json": input, "stdout.raw": stdout.Bytes(), "stderr.txt": stderr.Bytes(),
		} {
			if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
				t.Errorf("save oracle %s: %v", name, err)
			}
		}
		t.Logf("oracle executable=%q process-error=%v evidence=%q\nstderr phases:\n%s\ncomplete request (%d bytes):\n%s", cmd.Path, runErr, dir, stderr.Bytes(), len(input), input)
	})
	flags := []string{"--report-on-fatalerror", "--report-on-signal", "--report-directory=" + dir,
		"--report-filename=node-report.json", "--report-exclude-env", "--report-exclude-network"}
	cmd.Args = append(append([]string{cmd.Args[0]}, flags...), cmd.Args[1:]...)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr = cmd.Run()
	if runErr != nil {
		t.Fatalf("oracle executable=%q: %v\nstdout:\n%s\nstderr:\n%s", cmd.Path, runErr, stdout.Bytes(), stderr.Bytes())
	}
	return stdout.Bytes()
}
