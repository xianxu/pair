package couchcore

import (
	"testing"
	"time"
)

func TestResourcePolicyStrict(t *testing.T) {
	p, e := ParseResourcePolicy(nil)
	if e != nil || p.Cadence != time.Minute || p.ProcessRSSBytes != 8<<30 {
		t.Fatal(p, e)
	}
	for _, m := range []map[string]string{{"COUCH_RESOURCE_CADENCE": "0s"}, {"COUCH_RESOURCE_PROCESS_RSS_BYTES": "-1"}, {"COUCH_RESOURCE_IDLE_CPU_PERCENT": "NaN"}, {"COUCH_RESOURCE_WRONG": "1"}} {
		if _, e := ParseResourcePolicy(m); e == nil {
			t.Fatal(m)
		}
	}
}
