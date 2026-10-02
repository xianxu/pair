package couchcmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestAdoptionCommandSurvivingIncarnationCannotBeExcluded(t *testing.T) {
	for _, selected := range []bool{false, true} {
		name := "other-store-refused"
		if selected {
			name = "selected-store-reconnect-allowed"
		}
		t.Run(name, func(t *testing.T) {
			f := newAdoptionFixture(t)
			store := f.store
			args := f.args()
			if !selected {
				store = filepath.Join(f.sources, "other-store")
				args = f.args("--exclude-store", store)
			}
			ns, err := couchcore.ResolveCouchNamespace(store, "")
			if err != nil {
				t.Fatal(err)
			}
			proc := couchcore.OSProcOps{}
			identity, err := proc.Identity(os.Getpid())
			if err != nil {
				t.Fatal(err)
			}
			record := couchcore.ThreadRecord{
				SchemaVersion: couchcore.ThreadSchemaVersion,
				Address:       couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "couch-0123456789abcdef"},
				StartingPath:  store, WorkingPath: store, CreatedAt: time.Now().UTC(), Revision: 1,
				Incarnations: []couchcore.ThreadIncarnation{{PID: os.Getpid(), Identity: identity, State: couchcore.IncarnationLive}},
			}
			if _, err := couchcore.NewThreadStore(ns).CreateThread(record); err != nil {
				t.Fatal(err)
			}
			// The wrapper/process survived its supervisor. Holding a supervisor
			// lease here would exercise a different, already-covered refusal.
			observation, err := couchcore.ObserveSupervisor(ns, proc, nil)
			if err != nil || observation.Held {
				t.Fatalf("fixture unexpectedly supervised: %+v %v", observation, err)
			}
			before := adoptionSnapshot(t, f.root)
			code, output, stderr := f.run(args...)
			report := decodeAdoptionReport(t, output)
			if after := adoptionSnapshot(t, f.root); !reflect.DeepEqual(before, after) {
				t.Fatal("liveness preview changed source or authority bytes")
			}
			if !selected {
				if code == 0 || report.Status != "UNMIGRATED" || len(report.Blockers) == 0 || !strings.Contains(stderr, "UNMIGRATED") {
					t.Fatalf("excluded a store with a surviving incarnation: code=%d report=%+v stderr=%s", code, report, stderr)
				}
				return
			}
			if code != 0 || report.Status != "READY" || len(report.Blockers) != 0 {
				t.Fatalf("selected-store survivor prevented reconnect adoption: code=%d report=%+v stderr=%s", code, report, stderr)
			}
			if code, _, stderr := f.run(append(args, "--apply", report.Digest)...); code != 0 {
				t.Fatalf("selected-store reconnect adoption: %d %s", code, stderr)
			}
			code, output, stderr = f.run(args...)
			if code != 0 || decodeAdoptionReport(t, output).Status != "SELECTED" {
				t.Fatalf("selected-store reconnect preview: %d %s %s", code, output, stderr)
			}
		})
	}
}
