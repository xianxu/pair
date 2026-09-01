package notifycmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type fakeRuntime struct {
	env      map[string]string
	files    map[string][]byte
	writes   map[string][][]byte
	writeErr error
}

func (f *fakeRuntime) Getenv(key string) string { return f.env[key] }
func (f *fakeRuntime) ReadFile(path string) ([]byte, error) {
	b, ok := f.files[path]
	if !ok {
		return nil, errors.New("missing")
	}
	return append([]byte(nil), b...), nil
}
func (f *fakeRuntime) WriteNonblocking(path string, p []byte) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.writes[path] = append(f.writes[path], append([]byte(nil), p...))
	return nil
}

func validRuntime() *fakeRuntime {
	return &fakeRuntime{
		env:    map[string]string{"PAIR_TAG": "pair", "PAIR_OUTER_TTY_PATH": "/state/outer"},
		files:  map[string][]byte{"/state/outer": []byte("/dev/tty42\n")},
		writes: map[string][][]byte{},
	}
}

func TestRunWritesCanonicalEnvelopeForLegacyOptions(t *testing.T) {
	for _, args := range [][]string{{"ready"}, {"--osc", "9", "ready"}, {"--osc=777", "ready"}} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			rt := validRuntime()
			var stderr bytes.Buffer
			if code := Run(args, rt, &stderr); code != 0 {
				t.Fatalf("Run() = %d, stderr=%q", code, stderr.String())
			}
			writes := rt.writes["/dev/tty42"]
			if len(writes) != 1 || string(writes[0]) != "\x1b]777;notify;pair;ready\x07" {
				t.Fatalf("writes = %q", writes)
			}
		})
	}
}

func TestRunRejectsUsageErrors(t *testing.T) {
	for _, args := range [][]string{nil, {"--osc", "8", "ready"}, {"--osc"}} {
		rt := validRuntime()
		var stderr bytes.Buffer
		if code := Run(args, rt, &stderr); code != 2 || stderr.Len() == 0 || len(rt.writes) != 0 {
			t.Fatalf("Run(%v) = %d, stderr=%q writes=%v", args, code, stderr.String(), rt.writes)
		}
	}
}

func TestRunToleratesUnavailableOuterTTY(t *testing.T) {
	cases := []func(*fakeRuntime){
		func(rt *fakeRuntime) { delete(rt.env, "PAIR_TAG") },
		func(rt *fakeRuntime) { delete(rt.env, "PAIR_OUTER_TTY_PATH") },
		func(rt *fakeRuntime) { delete(rt.files, "/state/outer") },
		func(rt *fakeRuntime) { rt.files["/state/outer"] = []byte("\n") },
		func(rt *fakeRuntime) { rt.writeErr = errors.New("stale tty") },
	}
	for i, mutate := range cases {
		rt := validRuntime()
		mutate(rt)
		var stderr bytes.Buffer
		if code := Run([]string{"ready"}, rt, &stderr); code != 0 || stderr.Len() == 0 {
			t.Fatalf("case %d: code=%d stderr=%q", i, code, stderr.String())
		}
	}
}
