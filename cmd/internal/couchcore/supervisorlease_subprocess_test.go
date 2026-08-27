package couchcore

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

const supervisorLeaseHelperEnv = "PAIR_TEST_SUPERVISOR_LEASE_HELPER"

func TestSupervisorLeaseSubprocessHelper(t *testing.T) {
	mode := os.Getenv(supervisorLeaseHelperEnv)
	if mode == "" {
		return
	}
	ns, err := ResolveCouchNamespace(os.Getenv("COUCH_STORE_DIR"), "/unused")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	lease, err := AcquireSupervisorLease(ns, OSProcOps{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	defer lease.Close()
	fmt.Println("READY")
	if mode == "exec" {
		if err := syscall.Exec("/bin/sleep", []string{"sleep", "3"}, os.Environ()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(4)
		}
	}
	select {}
}

func startSupervisorLeaseHelper(t *testing.T, ns CouchNamespace, mode string) (*exec.Cmd, *bufio.Reader) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSupervisorLeaseSubprocessHelper$")
	cmd.Env = append(os.Environ(), supervisorLeaseHelperEnv+"="+mode, "COUCH_STORE_DIR="+ns.Dir())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})
	reader := bufio.NewReader(stdout)
	line, err := reader.ReadString('\n')
	if err != nil || line != "READY\n" {
		t.Fatalf("helper readiness = %q, %v", line, err)
	}
	return cmd, reader
}

func TestSupervisorLeaseOwnerCrashReleasesLock(t *testing.T) {
	ns := testCouchNamespace(t)
	cmd, _ := startSupervisorLeaseHelper(t, ns, "hold")
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill helper: %v", err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("killed helper unexpectedly exited cleanly")
	}
	cmd.Process = nil

	lease, err := AcquireSupervisorLease(ns, OSProcOps{})
	if err != nil {
		t.Fatalf("acquire after owner crash: %v", err)
	}
	defer lease.Close()
}

func TestSupervisorLeaseDescriptorDoesNotSurviveExec(t *testing.T) {
	ns := testCouchNamespace(t)
	cmd, _ := startSupervisorLeaseHelper(t, ns, "exec")

	deadline := time.Now().Add(750 * time.Millisecond)
	for {
		lease, err := AcquireSupervisorLease(ns, OSProcOps{})
		if err == nil {
			defer lease.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("lease remained held after helper exec: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("execed child was not alive when lease became available: %v", err)
	}
}
