package couchtty

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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

// nativeSignalSource gives each SIGINT case one owner of signal delivery.
type nativeSignalSource uint8

const (
	nativeSignalFromParent nativeSignalSource = iota
	nativeSignalFromCallback
	nativeSignalOmitted
)

type nativeExitCase struct {
	name, body, receipt string
	signal              syscall.Signal
	source              nativeSignalSource
	wantExit            int
}

type nativeExitOutcome struct {
	status  syscall.WaitStatus
	waitErr error
	stderr  string
	receipt string
}

func (out nativeExitOutcome) String() string {
	return fmt.Sprintf("status=%d wait=%v stderr=%q receipt=%q", out.status, out.waitErr, out.stderr, out.receipt)
}

// nativeExitMatches is independent of the baseline/instrumented comparison.
func nativeExitMatches(tc nativeExitCase, status syscall.WaitStatus) bool {
	if tc.signal != 0 {
		return status.Signaled() && status.Signal() == tc.signal ||
			tc.signal == syscall.SIGINT && status.Exited() && status.ExitStatus() == 130
	}
	return status.Exited() && status.ExitStatus() == tc.wantExit
}

const nativeExitCallback = `
import signal,weakref
class Victim: pass
def deliver(_):
 signal.raise_signal(signal.SIGINT)
victim=Victim()
reference=weakref.ref(victim,deliver)
del victim
`

func nativeExitScript(tc nativeExitCase, instrumentation string) string {
	script := "import os,sys,signal\nlog=open(sys.argv[1],'a',buffering=1)\n"
	if tc.signal == syscall.SIGINT {
		// #703: readiness alone does not constrain where KeyboardInterrupt
		// lands. A weakref callback can swallow it even after ready was read.
		script += "assert signal.getsignal(signal.SIGINT) == signal.default_int_handler, 'SIGINT default handler required'\nsignal.pthread_sigmask(signal.SIG_BLOCK,{signal.SIGINT})\nprint('SIGINT default handler, blocked until EOF release',file=sys.stderr,flush=True)\n"
	}
	script += instrumentation + "\nprint('ready',flush=True)\n"
	if tc.signal == syscall.SIGINT {
		script += "sys.stdin.read()\n"
	}
	script += tc.body
	if tc.signal == syscall.SIGINT {
		script += "assert signal.SIGINT in signal.sigpending(), 'SIGINT missing after EOF release'\nsignal.pthread_sigmask(signal.SIG_UNBLOCK,{signal.SIGINT})\n"
	}
	return script
}

func runNativeExitSpecimen(t *testing.T, tc nativeExitCase, instrumentation string) nativeExitOutcome {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	path := filepath.Join(t.TempDir(), "actor.log")
	cmd := exec.CommandContext(ctx, python, "-c", nativeExitScript(tc, instrumentation), path)
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
	var waitErr error
	cleanup := func() {
		_ = stdin.Close()
		if !reaped {
			_ = cmd.Process.Kill()
			waitErr = cmd.Wait()
			reaped = true
		}
	}
	t.Cleanup(cleanup)
	if line, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || line != "ready\n" {
		cleanup()
		data, readErr := os.ReadFile(path)
		t.Fatalf("actor readiness: %q, %v; wait=%v status=%v stderr=%q receipt=%q read=%v", line, err, waitErr, cmd.ProcessState, stderr.String(), data, readErr)
	}
	if tc.signal != 0 && tc.source == nativeSignalFromParent {
		if err := cmd.Process.Signal(tc.signal); err != nil {
			cleanup()
			data, readErr := os.ReadFile(path)
			t.Fatalf("actor signal: %v; wait=%v status=%v stderr=%q receipt=%q read=%v", err, waitErr, cmd.ProcessState, stderr.String(), data, readErr)
		}
	}
	if tc.signal == 0 || tc.signal == syscall.SIGINT {
		_ = stdin.Close()
	}
	waitErr = cmd.Wait()
	reaped = true
	data, readErr := os.ReadFile(path)
	if ctx.Err() != nil || readErr != nil || cmd.ProcessState == nil {
		t.Fatalf("actor completion: context=%v read=%v wait=%v status=%v stderr=%q receipt=%q", ctx.Err(), readErr, waitErr, cmd.ProcessState, stderr.String(), data)
	}
	return nativeExitOutcome{status: cmd.ProcessState.Sys().(syscall.WaitStatus), waitErr: waitErr, stderr: stderr.String(), receipt: string(data)}
}

func TestNativeAgentExitDiagnostics(t *testing.T) {
	for _, tc := range []nativeExitCase{
		{name: "eof", body: "sys.stdin.read()\n", receipt: "EXIT=atexit\n"},
		{name: "exception", body: "raise ValueError('fixture failure')\n", wantExit: 1, receipt: "EXIT=exception:ValueError\n"},
		{name: "exception_bounded", body: "raise type('oops\\n'+'X'*200,(Exception,),{})()\n", wantExit: 1, receipt: "EXIT=exception:oops?" + strings.Repeat("X", 81) + "\n"},
		{name: "system_exit", body: "sys.exit(7)\n", wantExit: 7, receipt: "EXIT=atexit\n"},
		{name: "hup", body: "sys.stdin.read()\n", signal: syscall.SIGHUP, receipt: "EXIT=signal:SIGHUP\n"},
		{name: "term", body: "sys.stdin.read()\n", signal: syscall.SIGTERM, receipt: "EXIT=signal:SIGTERM\n"},
		{name: "int", signal: syscall.SIGINT, receipt: "EXIT=signal:SIGINT\n"},
		{name: "int_callback", body: nativeExitCallback, signal: syscall.SIGINT, source: nativeSignalFromCallback, receipt: "EXIT=signal:SIGINT\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseline := runNativeExitSpecimen(t, tc, "")
			observed := runNativeExitSpecimen(t, tc, nativeAgentExitDiagnostics)
			if !nativeExitMatches(tc, baseline.status) {
				t.Errorf("baseline did not terminate as requested: %+v; instrumented=%+v", baseline, observed)
			}
			if observed.status != baseline.status {
				t.Errorf("instrumentation changed termination: baseline=%+v instrumented=%+v", baseline, observed)
			}
			if observed.receipt != tc.receipt {
				t.Errorf("exit receipt = %q, want %q; baseline=%+v instrumented=%+v", observed.receipt, tc.receipt, baseline, observed)
			}
			if len(observed.receipt) > 128 || strings.Count(observed.receipt, "\n") != 1 {
				t.Errorf("exit receipt not bounded to one line: %+v", observed)
			}
		})
	}
}

func TestNativeExitMatchesRejectsSharedWrongOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		want   nativeExitCase
		status syscall.WaitStatus
		match  bool
	}{
		{"eof", nativeExitCase{}, 0, true},
		{"exception", nativeExitCase{wantExit: 1}, 1 << 8, true},
		{"wrong exception", nativeExitCase{wantExit: 1}, 0, false},
		{"system exit", nativeExitCase{wantExit: 7}, 7 << 8, true},
		{"hup", nativeExitCase{signal: syscall.SIGHUP}, syscall.WaitStatus(syscall.SIGHUP), true},
		{"term", nativeExitCase{signal: syscall.SIGTERM}, syscall.WaitStatus(syscall.SIGTERM), true},
		{"int", nativeExitCase{signal: syscall.SIGINT}, syscall.WaitStatus(syscall.SIGINT), true},
		{"python int status", nativeExitCase{signal: syscall.SIGINT}, 130 << 8, true},
		{"swallowed int", nativeExitCase{signal: syscall.SIGINT}, 0, false},
		{"missing int", nativeExitCase{signal: syscall.SIGINT}, 1 << 8, false},
		{"wrong signal", nativeExitCase{signal: syscall.SIGINT}, syscall.WaitStatus(syscall.SIGTERM), false},
		{"wrong numeric signal status", nativeExitCase{signal: syscall.SIGHUP}, 130 << 8, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nativeExitMatches(tc.want, tc.status); got != tc.match {
				t.Fatalf("termination match=%v, want %v", got, tc.match)
			}
		})
	}
}

func TestNativeAgentExitDiagnosticsRejectsMissingSIGINT(t *testing.T) {
	for _, tc := range []struct {
		name, instrumentation, receipt string
	}{
		{name: "baseline"},
		{name: "instrumented", instrumentation: nativeAgentExitDiagnostics, receipt: "EXIT=exception:AssertionError\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := runNativeExitSpecimen(t, nativeExitCase{signal: syscall.SIGINT, source: nativeSignalOmitted}, tc.instrumentation)
			if !out.status.Exited() || out.status.ExitStatus() != 1 || out.waitErr == nil || !strings.Contains(out.stderr, "SIGINT missing after EOF release") || out.receipt != tc.receipt {
				t.Fatalf("missing signal did not fail at the pending assertion: %+v", out)
			}
		})
	}
}

func TestNativeExitOutcomeKeepsFailureEvidence(t *testing.T) {
	out := nativeExitOutcome{status: 1 << 8, waitErr: fmt.Errorf("sentinel wait failure"), stderr: "sentinel stderr", receipt: "sentinel receipt"}
	rendered := fmt.Sprintf("%+v", out)
	for _, evidence := range []string{"sentinel wait failure", "sentinel stderr", "sentinel receipt", "256"} {
		if !strings.Contains(rendered, evidence) {
			t.Errorf("missing %q in failure diagnostics: %s", evidence, rendered)
		}
	}
}
