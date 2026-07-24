package agentcmd

import (
	"bytes"
	"errors"
	"syscall"
	"testing"
)

type fakeRuntime struct {
	data      []byte
	readPath  string
	signalPID int
	signal    syscall.Signal
}

func (f *fakeRuntime) ReadFile(path string) ([]byte, error) {
	f.readPath = path
	if f.data == nil {
		return nil, errors.New("missing")
	}
	return f.data, nil
}
func (f *fakeRuntime) Signal(pid int, sig syscall.Signal) error {
	f.signalPID, f.signal = pid, sig
	return nil
}

func TestRunRestartSignalsStableWrapper(t *testing.T) {
	rt := &fakeRuntime{data: []byte("42\n")}
	var stderr bytes.Buffer
	env := func(key string) string {
		switch key {
		case "PAIR_TAG":
			return "work"
		case "PAIR_DATA_DIR":
			return "/data"
		default:
			return ""
		}
	}
	if code := RunRestart(nil, env, rt, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if rt.readPath != "/data/pair-wrap-pid-work" || rt.signalPID != 42 || rt.signal != syscall.SIGUSR2 {
		t.Fatalf("restart target = (%q,%d,%v)", rt.readPath, rt.signalPID, rt.signal)
	}
}
