package couchsingleton

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSingletonProcessHelper(t *testing.T) {
	mode := os.Getenv("PAIR_SINGLETON_TEST_HELPER")
	if mode == "" {
		return
	}
	root := os.Getenv("PAIR_SINGLETON_TEST_ROOT")
	m := Manager{AuthorityDir: filepath.Join(root, "singleton"), Defaults: Roots{filepath.Join(root, "pair", "couch"), filepath.Join(root, "pair"), filepath.Join(root, "identity")}}
	_, l, e := m.Acquire(Request{})
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	defer l.Close()
	fmt.Println("READY")
	if mode == "exec" {
		if e = syscall.Exec("/bin/sleep", []string{"sleep", "10"}, os.Environ()); e != nil {
			os.Exit(3)
		}
	}
	for {
		time.Sleep(time.Hour)
	}
}
func TestSingletonCrashAndExecReleaseBothLeases(t *testing.T) {
	for _, mode := range []string{"kill", "exec"} {
		t.Run(mode, func(t *testing.T) {
			m := fixture(t)
			cmd := exec.Command(os.Args[0], "-test.run=^TestSingletonProcessHelper$")
			cmd.Env = append(os.Environ(), "PAIR_SINGLETON_TEST_HELPER="+mode, "PAIR_SINGLETON_TEST_ROOT="+filepath.Dir(m.AuthorityDir))
			out, e := cmd.StdoutPipe()
			if e != nil {
				t.Fatal(e)
			}
			cmd.Stderr = os.Stderr
			if e = cmd.Start(); e != nil {
				t.Fatal(e)
			}
			defer func() { cmd.Process.Kill(); cmd.Wait() }()
			line, e := bufio.NewReader(out).ReadString('\n')
			if e != nil || line != "READY\n" {
				t.Fatal(line, e)
			}
			if mode == "kill" {
				if _, l, e := m.Acquire(Request{}); e == nil {
					l.Close()
					t.Fatal("contender admitted")
				}
				cmd.Process.Kill()
				cmd.Wait()
			}
			deadline := time.Now().Add(2 * time.Second)
			for {
				_, l, e := m.Acquire(Request{})
				if e == nil {
					l.Close()
					break
				}
				if time.Now().After(deadline) {
					t.Fatal(e)
				}
				time.Sleep(10 * time.Millisecond)
			}
			if mode == "exec" {
				if e = cmd.Process.Signal(syscall.Signal(0)); e != nil {
					t.Fatal("exec process already gone", e)
				}
			}
		})
	}
}
