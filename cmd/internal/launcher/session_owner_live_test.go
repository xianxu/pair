package launcher

import (
	"context"
	"os"
	"testing"
)

// TestSessionOwnerLiveConformance only reads an explicitly selected session.
// It never launches, attaches, deletes, or writes Pair/Couch state.
func TestSessionOwnerLiveConformance(t *testing.T) {
	name := os.Getenv("PAIR_LIVE_OWNER_SESSION")
	if name == "" {
		t.Skip("set PAIR_LIVE_OWNER_SESSION/SCOPE/TAG/DATA_DIR for read-only owner conformance")
	}
	scope, tag, root := os.Getenv("PAIR_LIVE_OWNER_SCOPE"), os.Getenv("PAIR_LIVE_OWNER_TAG"), os.Getenv("PAIR_LIVE_OWNER_DATA_DIR")
	if scope == "" || tag == "" || root == "" {
		t.Fatal("live conformance requires exact expected scope, tag and data directory")
	}
	probe := SessionOwnerProbe{}
	observation, err := probe.Probe(context.Background(), name, root, scope, tag)
	if err != nil || observation.State != SessionOwnerOwned {
		t.Fatalf("live owner: %+v %v", observation, err)
	}
	if err := probe.Revalidate(context.Background(), observation); err != nil {
		t.Fatal(err)
	}
	t.Logf("verified %s server PID=%d generation=%s owner=%s/%s", name, observation.Server.PID, observation.Server.Identity, scope, tag)
}
