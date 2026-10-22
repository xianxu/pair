package agentfixture

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestResourcesCommandIsReadOnlyClosed(t *testing.T) {
	if err := ValidateCommand(Command{ID: "resources", Kind: Resources}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []Command{{ID: "resources", Kind: Resources, Args: []string{"--confirm"}}, {ID: "resources", Kind: Resources, State: Idle}, {ID: "resources", Kind: Resources, Timeout: 1}} {
		if ValidateCommand(c) == nil {
			t.Fatalf("accepted %+v", c)
		}
	}
}
func TestResourceObservationRetainsExactFactsWithinExistingCap(t *testing.T) {
	row := couchcore.ResourceProcessView{ResourceProvenance: couchcore.ResourceProvenance{ResourceRoot: couchcore.ResourceRoot{Key: couchcore.ResourceProcessKey{PID: 42, Identity: "birth"}, Slot: "fixture:1"}, Orphan: true, OrphanKnown: true}, Name: "fixture"}
	result := resourceObservationResult("resources", couchcore.ResourceView{Processes: []couchcore.ResourceProcessView{row}, Orphans: []couchcore.ResourceProcessView{row}})
	var observed ResourceObservation
	if result.Error != "" || json.Unmarshal([]byte(result.Stdout), &observed) != nil || len(observed.Processes) != 1 || observed.Processes[0].Key != row.Key || !observed.Orphans[0].OrphanKnown {
		t.Fatalf("%+v %+v", result, observed)
	}
	row.Name = strings.Repeat("x", MaxOutputBytes+1)
	result = resourceObservationResult("oversize", couchcore.ResourceView{Processes: []couchcore.ResourceProcessView{row}})
	if result.Error == "" || result.Stdout != "" || result.Truncated {
		t.Fatalf("oversized readonly observation: %+v", result)
	}
}

func resourceObservationSocketRoot(t *testing.T) string {
	t.Helper()
	//lint:allow tmp-literal — private RPC socket fixtures need a short root even with a long inherited TMPDIR; prefer TestScratch when mounted
	parent := "/tmp"
	if info, err := os.Stat("/Volumes/TestScratch"); err == nil && info.IsDir() {
		parent = "/Volumes/TestScratch"
	}
	root, err := os.MkdirTemp(parent, "ro-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("remove private resource socket fixture: %v", err)
		}
	})
	physical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return physical
}

func TestResourceObservationUsesExistingPrivateRPCAndCallerCancellation(t *testing.T) {
	longTemp := filepath.Join(t.TempDir(), strings.Repeat("long", 30))
	if err := os.MkdirAll(longTemp, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", longTemp)
	root := resourceObservationSocketRoot(t)
	t.Setenv("COUCH_ISOLATED_ROOT", root)
	store := filepath.Join(root, "store")
	if err := os.Mkdir(store, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COUCH_STORE_DIR", store)
	socket, err := couchmessage.SocketPath(store, "broker")
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server, err := couchmessage.StartServer(t.Context(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if e := json.Unmarshal(raw, &request); e != nil {
			return nil, e
		}
		if request.Op != "resources" {
			return nil, fmt.Errorf("unexpected mutation %s", request.Op)
		}
		n := calls.Add(1)
		return json.Marshal(couchmessage.Response{Code: "ok", Resources: &couchcore.ResourceView{Processes: []couchcore.ResourceProcessView{{ResourceProvenance: couchcore.ResourceProvenance{ResourceRoot: couchcore.ResourceRoot{Key: couchcore.ResourceProcessKey{PID: int(n), Identity: "live"}}}}}}})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	for i := 1; i <= 2; i++ {
		result := readResourceObservation(t.Context(), fmt.Sprintf("read-%d", i))
		var observed ResourceObservation
		if result.Error != "" || json.Unmarshal([]byte(result.Stdout), &observed) != nil || observed.Processes[0].Key.PID != i {
			t.Fatalf("%+v %+v", result, observed)
		}
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if result := readResourceObservation(canceled, "canceled"); result.Error == "" || calls.Load() != 2 {
		t.Fatalf("canceled read: %+v calls%d", result, calls.Load())
	}
}
