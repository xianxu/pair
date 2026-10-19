package diagnosticlog

import (
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"os"
	"path/filepath"
	"testing"
)

type sweepProcesses struct{ calls int }

func (p *sweepProcesses) Snapshot(context.Context) ([]procutil.Observation, error) {
	p.calls++
	return nil, nil
}
func (*sweepProcesses) Identity(string) string                     { return "identity" }
func (*sweepProcesses) StrictIdentity(string) string               { return "birth" }
func (*sweepProcesses) Environment(int) (map[string]string, error) { return nil, nil }
func TestSweepNativeInspectionCountIsConstant(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			paths := []string{}
			for i := range n {
				p := filepath.Join(t.TempDir(), fmt.Sprint(i))
				if err := os.WriteFile(p, nil, 0600); err != nil {
					t.Fatal(err)
				}
				paths = append(paths, p)
			}
			for _, broken := range []bool{false, true} {
				processes := &sweepProcesses{}
				commands := 0
				inspector := OSInspection{processes: processes, openFileCommand: func(ctx context.Context, args ...string) ([]byte, []byte, error) {
					commands++
					if broken {
						return nil, nil, errors.New("denied")
					}
					return nil, nil, nil
				}}
				proof := InspectionProof(inspector)
				ctx := SweepContext(context.Background(), paths)
				for _, p := range paths {
					for range 2 {
						err := proof(ctx, p, nil)
						if (err != nil) != broken {
							t.Fatalf("broken=%t proof=%v", broken, err)
						}
					}
				}
				if commands != 1 || processes.calls != 1 {
					t.Fatalf("N=%d native-files=%d native-processes=%d", n, commands, processes.calls)
				}
				// A new sweep cannot reuse an earlier observation or its cached failure.
				_ = proof(SweepContext(context.Background(), paths), paths[0], nil)
				if commands != 2 || processes.calls != 2 {
					t.Fatal("observation escaped sweep lifetime")
				}
			}
		})
	}
}

func TestSweepParserRejectsPartialAndUnexpectedOutput(t *testing.T) {
	wanted := map[string]bool{"/owned": true}
	for _, data := range []string{"p1\n", "p1\nf3\n", "n/owned\n", "pbad\nn/owned\n", "xunknown\n"} {
		if _, err := parseOpenFileSnapshot(context.Background(), []byte(data), wanted); err == nil {
			t.Fatalf("partial output accepted: %q", data)
		}
	}
	parsed, err := parseOpenFileSnapshot(context.Background(), nil, wanted)
	if err != nil {
		t.Fatal(err)
	}
	observed := &observedInspection{holders: parsed}
	if _, err := observed.OpenFiles(context.Background(), "/owned"); err != nil {
		t.Fatal(err)
	}
	if _, err := observed.OpenFiles(context.Background(), "/not-observed"); err == nil {
		t.Fatal("unobserved/new path treated as proof of no holders")
	}
	bounded := &snapshotBuffer{limit: 8}
	if n, err := bounded.Write([]byte("123456789")); n != 9 || err != nil || !bounded.overflow || bounded.Len() != 8 {
		t.Fatal("output bound disconnected")
	}
}

func TestSweepNativeOpenFileConformance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned.log")
	fd, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()
	canonicalPath, err := canonical(path)
	if err != nil {
		t.Fatal(err)
	}
	segment := segmentPath(canonicalPath, fmt.Sprintf("segment-%032x.log", 1))
	if err := os.MkdirAll(filepath.Dir(segment), 0700); err != nil {
		t.Fatal(err)
	}
	reader, err := os.Create(segment)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	snapshot, err := (OSInspection{processes: &sweepProcesses{}}).Snapshot(context.Background(), []string{path})
	if err != nil {
		requireLsofObservation(t, err)
	}
	holders, err := snapshot.OpenFiles(context.Background(), path)
	requireLsofObservation(t, err)
	assertLsofHolder(t, holders, os.Getpid(), procutil.StrictIdentity(fmt.Sprint(os.Getpid())))
	if len(holders) < 2 {
		t.Fatal("immutable segment reader missing", holders)
	}
	if err := fd.Close(); err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	snapshot, err = (OSInspection{processes: &sweepProcesses{}}).Snapshot(context.Background(), []string{path})
	if err != nil {
		requireLsofObservation(t, err)
	}
	holders, err = snapshot.OpenFiles(context.Background(), path)
	requireLsofObservation(t, err)
	if len(holders) != 0 {
		t.Fatal("closed file retained live holders", holders)
	}
}

func TestSweepGlobalSnapshotMatchesCurrentAndClosedReadersOnly(t *testing.T) {
	owner := "/owned/wrap-events-test.jsonl"
	segment := fmt.Sprintf("%s.pair-diagnostics/segments/0/0/0/0/segment-%032x.log", owner, 1)
	data := fmt.Sprintf("p999999999\nf1\nn/unrelated/file\np%d\nf3\nn%s\nf4\nn%s\n", os.Getpid(), owner, segment)
	holders, err := parseOpenFileSnapshot(context.Background(), []byte(data), map[string]bool{owner: true})
	if err != nil || len(holders[owner]) != 2 {
		t.Fatalf("current/closed holder inventory %+v %v", holders, err)
	}
}
