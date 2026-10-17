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
	if p.OrphanNvimAge != 6*time.Hour {
		t.Fatalf("the orphan age defaulted to %v; the sweep would name a freshly launched nvim", p.OrphanNvimAge)
	}
	if p, e = ParseResourcePolicy(map[string]string{"COUCH_RESOURCE_ORPHAN_NVIM_AGE": "90m"}); e != nil || p.OrphanNvimAge != 90*time.Minute {
		t.Fatal(p, e)
	}
	// A malformed knob must refuse rather than silently widen the sweep.
	for _, v := range []string{"", "13 days", "0s", "-6h", "25h"} {
		if _, e := ParseResourcePolicy(map[string]string{"COUCH_RESOURCE_ORPHAN_NVIM_AGE": v}); e == nil {
			t.Fatalf("COUCH_RESOURCE_ORPHAN_NVIM_AGE=%q was accepted", v)
		}
	}
	for _, m := range []map[string]string{{"COUCH_RESOURCE_CADENCE": "0s"}, {"COUCH_RESOURCE_PROCESS_RSS_BYTES": "-1"}, {"COUCH_RESOURCE_IDLE_CPU_PERCENT": "NaN"}, {"COUCH_RESOURCE_WRONG": "1"}} {
		if _, e := ParseResourcePolicy(m); e == nil {
			t.Fatal(m)
		}
	}
}
