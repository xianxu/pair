package threadrecord

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// liveRecordV2 is the shape a record written before pair#170 M4 actually has on
// disk, copied from the operator's store (paths anonymised, values otherwise
// verbatim). Every key here appears in real data.
//
// It matters because these structs ARE the on-disk schema and strictjson sets
// DisallowUnknownFields: a field deleted from the Go type makes every record
// still carrying it undecodable, and a record that will not decode is a thread
// that vanishes from every view at once. Measured against the live store,
// `claim_generation` was in 17/17 records and `policy` in 5/5 incarnations, so
// deleting them outright would have made the whole store unreadable.
//
// `legacy_actor_id` is here for a different reason: the live store has none
// left (the registry-cutover records it named were parked, which clears
// incarnations), but a store where one is still live would be bricked by its
// removal, and "absent from the one store I measured" is not "absent". The
// tombstone costs a line; being wrong costs the store.
//
// They are therefore TOMBSTONES: decoded, ignored, never written back, so
// records shed them on their next write without any migration pass.
const liveRecordV2 = `{
  "schema_version": 2,
  "address": {"repo_scope": "53998a195009a6ec", "tag": "couch-1539ce935d4238b7"},
  "starting_path": "/repo/example",
  "working_path": "/repo/example",
  "created_at": "2026-09-02T14:25:51.487991-07:00",
  "revision": 5,
  "claim_generation": 55,
  "incarnations": [
    {
      "legacy_actor_id": "couch-d80d318b",
      "pid": 22870,
      "identity": "1788384351.597473",
      "state": "live",
      "started_at": "2026-09-02T14:25:51.487991-07:00",
      "policy": {
        "policy_version": 1,
        "policy_digest": "b446286124e39db3c6117fe7b3f29aa4c43d266ff96eccfa475a84f1f8ba7844",
        "repo_identity": "/repo/example/.git",
        "admission_key": "/repo/example/.git",
        "capacity": {"kind": "bounded", "limit": 1},
        "on_capacity": "reject"
      },
      "launch_profile": {"agent": "claude", "argv": ["--dangerously-skip-permissions"]}
    }
  ],
  "latest_launch_profile": {"agent": "claude", "argv": ["--dangerously-skip-permissions"]},
  "last_active_at": "0001-01-01T00:00:00Z"
}`

func TestPreM4RecordsStillDecode(t *testing.T) {
	var record Record
	if err := strictjson.Decode([]byte(liveRecordV2), &record); err != nil {
		t.Fatalf("a record written before M4 no longer decodes: %v\n"+
			"every thread carrying this field would vanish from every view at once", err)
	}
	validators := Validators{
		RepoScope: func(string) error { return nil },
		Tag:       func(string) error { return nil },
	}
	if err := Validate(record, validators); err != nil {
		t.Fatalf("a record written before M4 no longer validates: %v", err)
	}
	if len(record.Incarnations) != 1 {
		t.Fatalf("incarnations = %+v, want the one in the fixture", record.Incarnations)
	}
	if record.Incarnations[0].LaunchProfile == nil || record.Incarnations[0].LaunchProfile.Agent != "claude" {
		t.Fatalf("launch profile lost: %+v", record.Incarnations[0].LaunchProfile)
	}
}
