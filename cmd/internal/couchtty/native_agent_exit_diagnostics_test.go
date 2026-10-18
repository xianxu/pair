package couchtty

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Hooks only observe the fixture's termination. In particular SIGINT keeps
// Python's KeyboardInterrupt behavior, and default signals are re-raised.
// atexit does not imply exit status zero (SystemExit also runs this hook).
const nativeAgentExitDiagnostics = `
import atexit,signal
_native_exit_recorded=False
def _native_record_exit(reason):
 global _native_exit_recorded
 if _native_exit_recorded: return
 _native_exit_recorded=True
 try:
  reason=''.join(c if ' ' <= c <= '~' else '?' for c in reason)[:96]
  log.write('EXIT='+reason+'\n')
  log.flush()
 except (OSError,ValueError): pass
atexit.register(lambda: _native_record_exit('atexit'))
_native_previous_excepthook=sys.excepthook
def _native_excepthook(kind,value,traceback):
 _native_record_exit('exception:'+kind.__name__)
 _native_previous_excepthook(kind,value,traceback)
sys.excepthook=_native_excepthook
_native_previous_signals={}
def _native_exit_signal(number,frame):
 _native_record_exit('signal:'+signal.Signals(number).name)
 previous=_native_previous_signals[number]
 signal.signal(number,previous)
 if callable(previous): previous(number,frame)
 elif previous==signal.SIG_DFL: os.kill(os.getpid(),number)
for _native_signal in (signal.SIGHUP,signal.SIGTERM,signal.SIGINT):
 _native_previous_signals[_native_signal]=signal.getsignal(_native_signal)
 if _native_previous_signals[_native_signal]!=signal.SIG_IGN:
  signal.signal(_native_signal,_native_exit_signal)
`

func TestNativeAgentExitDiagnostics(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable")
	}
	for _, tc := range []struct {
		name, body, receipt string
		signal              syscall.Signal
	}{
		{name: "eof", body: "sys.stdin.read()\n", receipt: "EXIT=atexit\n"},
		{name: "exception", body: "raise ValueError('fixture failure')\n", receipt: "EXIT=exception:ValueError\n"},
		{name: "exception_bounded", body: "raise type('oops\\n'+'X'*200,(Exception,),{})()\n", receipt: "EXIT=exception:oops?" + strings.Repeat("X", 81) + "\n"},
		{name: "system_exit", body: "sys.exit(7)\n", receipt: "EXIT=atexit\n"},
		{name: "hup", body: "sys.stdin.read()\n", signal: syscall.SIGHUP, receipt: "EXIT=signal:SIGHUP\n"},
		{name: "term", body: "sys.stdin.read()\n", signal: syscall.SIGTERM, receipt: "EXIT=signal:SIGTERM\n"},
		{name: "int", body: "sys.stdin.read()\n", signal: syscall.SIGINT, receipt: "EXIT=signal:SIGINT\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := func(instrumentation string) (syscall.WaitStatus, string) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				t.Cleanup(cancel)
				path := filepath.Join(t.TempDir(), "actor.log")
				script := "import os,sys\nlog=open(sys.argv[1],'a',buffering=1)\n" + instrumentation + "\nprint('ready',flush=True)\n" + tc.body
				cmd := exec.CommandContext(ctx, python, "-c", script, path)
				stdin, err := cmd.StdinPipe()
				if err != nil {
					t.Fatal(err)
				}
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				reaped := false
				t.Cleanup(func() {
					stdin.Close()
					if !reaped {
						_ = cmd.Process.Kill()
						_ = cmd.Wait()
					}
				})
				if line, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || line != "ready\n" {
					t.Fatalf("actor readiness: %q, %v", line, err)
				}
				if tc.signal != 0 {
					if err := cmd.Process.Signal(tc.signal); err != nil {
						t.Fatal(err)
					}
				} else {
					stdin.Close()
				}
				_ = cmd.Wait()
				reaped = true
				if ctx.Err() != nil {
					t.Fatalf("actor exceeded cleanup bound: %v", ctx.Err())
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				return cmd.ProcessState.Sys().(syscall.WaitStatus), string(data)
			}
			baseline, _ := run("")
			status, receipt := run(nativeAgentExitDiagnostics)
			if status != baseline {
				t.Errorf("instrumentation changed termination: got %v, baseline %v", status, baseline)
			}
			if receipt != tc.receipt {
				t.Errorf("exit receipt = %q, want %q", receipt, tc.receipt)
			}
			if len(receipt) > 128 || strings.Count(receipt, "\n") != 1 {
				t.Errorf("exit receipt not bounded to one line: %q", receipt)
			}
		})
	}
}
