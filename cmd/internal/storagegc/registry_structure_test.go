package storagegc

import (
	"path/filepath"
	"testing"
)

func TestRegistryStructureValidatesCapturedSnapshotWithoutProbingPaths(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-mounted")
	good := StoreRegistry{Version: 1, Stores: []string{missing}}
	if err := good.ValidateStructure(); err != nil {
		t.Fatalf("structural validation probed missing directory: %v", err)
	}
	for _, bad := range []StoreRegistry{{Version: 2, Stores: []string{}}, {Version: 1}, {Version: 1, Stores: []string{"relative"}}, {Version: 1, Stores: []string{missing, missing}}} {
		if err := bad.ValidateStructure(); err == nil {
			t.Fatalf("invalid captured registry accepted: %+v", bad)
		}
	}
}
