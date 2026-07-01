package runtimebundle

import (
	"strings"
	"testing"
)

func TestPlanCleanupKeepsSelectedRuntime(t *testing.T) {
	a := strings.Repeat("a", 64)
	b := strings.Repeat("b", 64)
	c := strings.Repeat("c", 64)
	plan, err := PlanCleanup(CleanupInput{
		SelectedDigest: b,
		Keep:           1,
		Generations: []RuntimeGeneration{
			{Digest: a, HasMarker: true, ModUnix: 10},
			{Digest: b, HasMarker: true, ModUnix: 1},
			{Digest: c, HasMarker: true, ModUnix: 20},
		},
	})
	if err != nil {
		t.Fatalf("PlanCleanup error = %v", err)
	}
	if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != a {
		t.Fatalf("DeleteDigests = %#v, want only %s", plan.DeleteDigests, a)
	}
}

func TestPlanCleanupIgnoresNonRuntimeDirectories(t *testing.T) {
	b := strings.Repeat("b", 64)
	c := strings.Repeat("c", 64)
	plan, err := PlanCleanup(CleanupInput{
		SelectedDigest: b,
		Keep:           0,
		Generations: []RuntimeGeneration{
			{Digest: "not-a-digest", HasMarker: true, ModUnix: 10},
			{Digest: strings.Repeat("a", 64), HasMarker: false, ModUnix: 20},
			{Digest: "abcd", HasMarker: true, ModUnix: 25},
			{Digest: b, HasMarker: true, ModUnix: 1},
			{Digest: c, HasMarker: true, ModUnix: 30},
		},
	})
	if err != nil {
		t.Fatalf("PlanCleanup error = %v", err)
	}
	if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != c {
		t.Fatalf("DeleteDigests = %#v, want only %s", plan.DeleteDigests, c)
	}
}
