package procutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSnapshotDiagnosticsPublicEntriesAreOptIn(t *testing.T) {
	for _, flag := range []string{"", "true", "1"} {
		t.Run(flag, func(t *testing.T) {
			t.Setenv("PAIR_STARTUP_TIMING", flag)
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			original := os.Stderr
			os.Stderr = writer
			defer func() { os.Stderr = original; writer.Close() }()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			for _, capture := range []func() ([]Observation, error){
				func() ([]Observation, error) { return Snapshot(ctx) },
				func() ([]Observation, error) { return SnapshotNamed(ctx, "PRIVATE-REQUESTED-NAME") },
			} {
				if got, err := capture(); got != nil || !errors.Is(err, context.Canceled) {
					t.Fatalf("capture = %v, %v", got, err)
				}
			}
			writer.Close()
			os.Stderr = original
			raw, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if flag != "1" {
				if len(raw) != 0 {
					t.Fatalf("default diagnostics: %s", raw)
				}
				return
			}
			lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
			if len(lines) != 2 || !strings.Contains(lines[0], "kind=full table_calls=0 command_calls=0 identity_calls=0 duration_ms=") || !strings.Contains(lines[1], "kind=named table_calls=0 command_calls=0 identity_calls=0 duration_ms=") {
				t.Fatalf("capture diagnostics = %q", raw)
			}
			if strings.Contains(string(raw), "PRIVATE") {
				t.Fatal("diagnostics leaked requested name")
			}
		})
	}
}

func TestSnapshotDiagnosticsCountActualReadsAndPreserveEvidence(t *testing.T) {
	world := func() *snapshotWorld {
		return &snapshotWorld{
			rows: []Process{
				{PID: 1, Name: "PRIVATE-NAME", Identity: "PRIVATE-ID"},
				{PID: 2, Name: "PRIVATE-NAME", Identity: "gone"},
				{PID: 3, Name: "PRIVATE-NAME"},
				{PID: 4, Name: "PRIVATE-NAME", Identity: "denied"},
				{PID: 5, Name: "PRIVATE-NAME", Identity: "old"},
				{PID: 6, Identity: "uncertain"},
				{PID: 7, Name: "other", Identity: "other"},
				{PID: 8, Name: "PRIVATE-NAME", Identity: "dead", Zombie: true},
				{PID: 0, Name: "PRIVATE-NAME", Identity: "invalid"},
			},
			commands: map[int]CommandInfo{
				1: {Executable: "/PRIVATE-PATH", Args: []string{"PRIVATE-ARGV"}},
				4: {}, 5: {}, 6: {}, 7: {},
			},
			commandErrors: map[int]error{4: errors.New("PRIVATE-ERROR")},
			identities:    map[int]string{1: "PRIVATE-ID", 4: "denied", 5: "replacement", 7: "other"},
		}
	}
	for _, named := range []bool{false, true} {
		var selectRow func(Process) bool
		kind, commands, identities := "full", 6, 5
		if named {
			kind, commands, identities = "named", 5, 4
			selectRow = func(p Process) bool { return processNameCandidate(p, "PRIVATE-NAME") }
		}
		want, wantErr := snapshotSelected(context.Background(), world(), selectRow)
		var out bytes.Buffer
		now, clockCalls := diagnosticClock()
		got, err := snapshotCapture(context.Background(), world(), selectRow, "1", &out, now)
		if err != wantErr || !reflect.DeepEqual(got, want) {
			t.Fatalf("diagnostics changed observation: got=%+v err=%v want=%+v err=%v", got, err, want, wantErr)
		}
		wantLine := fmt.Sprintf("pair startup snapshot: pid=%d kind=%s table_calls=1 command_calls=%d identity_calls=%d duration_ms=2.250\n", os.Getpid(), kind, commands, identities)
		if out.String() != wantLine || *clockCalls != 2 {
			t.Fatalf("diagnostics=%q clocks=%d, want %q", out.String(), *clockCalls, wantLine)
		}
	}
}

func diagnosticClock() (func() time.Time, *int) {
	count := new(int)
	return func() time.Time {
		at := time.Unix(0, 0).Add(time.Duration(*count) * 2250 * time.Microsecond)
		*count++
		return at
	}, count
}

func TestSnapshotDiagnosticsFailedCapturesKeepExactCountsAndErrors(t *testing.T) {
	for _, failure := range []string{"table", "unsupported", "pre-canceled", "during-command"} {
		t.Run(failure, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			w := &snapshotWorld{}
			wantErr, tables, commands := error(os.ErrPermission), 1, 0
			switch failure {
			case "table":
				w.err = wantErr
			case "unsupported":
				wantErr = ErrUnsupported
				w.err = wantErr
			case "pre-canceled":
				cancel()
				wantErr = context.Canceled
				tables = 0
			case "during-command":
				wantErr, commands = context.Canceled, 1
				w.rows = []Process{{PID: 1, Identity: "one"}, {PID: 2, Identity: "two"}}
				w.commands = map[int]CommandInfo{1: {}}
				w.after = func(int) { cancel() }
			}
			var out bytes.Buffer
			now, _ := diagnosticClock()
			got, err := snapshotCapture(ctx, w, nil, "1", &out, now)
			if got != nil || err != wantErr {
				t.Fatalf("capture=%+v, %v; want original %v", got, err, wantErr)
			}
			want := fmt.Sprintf("pair startup snapshot: pid=%d kind=full table_calls=%d command_calls=%d identity_calls=0 duration_ms=2.250\n", os.Getpid(), tables, commands)
			if out.String() != want {
				t.Fatalf("diagnostics=%q want=%q", out.String(), want)
			}
		})
	}
}

func TestSnapshotDiagnosticsDisabledDoesNotReadClockOrWrite(t *testing.T) {
	for _, flag := range []string{"", "0", "true"} {
		w := &snapshotWorld{rows: []Process{{PID: 1, Identity: "one"}}, commands: map[int]CommandInfo{1: {}}, identities: map[int]string{1: "one"}}
		var out bytes.Buffer
		got, err := snapshotCapture(context.Background(), w, nil, flag, &out, func() time.Time { t.Fatal("disabled clock read"); return time.Time{} })
		if err != nil || len(got) != 1 || got[0].Err != nil || out.Len() != 0 || w.tableCalls != 1 || !reflect.DeepEqual(w.calls, []int{1}) {
			t.Fatalf("disabled path changed: %+v %v %q", got, err, &out)
		}
	}
}

func BenchmarkSnapshotDiagnosticsOverhead(b *testing.B) {
	for _, flag := range []string{"", "1"} {
		b.Run("flag="+flag, func(b *testing.B) {
			w := &snapshotWorld{rows: []Process{{PID: 1, Identity: "one"}}, commands: map[int]CommandInfo{1: {}}, identities: map[int]string{1: "one"}}
			b.ReportAllocs()
			for b.Loop() {
				w.calls = w.calls[:0]
				if _, err := snapshotCapture(context.Background(), w, nil, flag, io.Discard, time.Now); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
