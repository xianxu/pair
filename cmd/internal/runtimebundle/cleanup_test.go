package runtimebundle

import "testing"

func TestPlanCleanupKeepsSelectedRuntime(t *testing.T) {
	plan, err := PlanCleanup(CleanupInput{
		SelectedDigest: "bbbb",
		Keep:           1,
		Generations: []RuntimeGeneration{
			{Digest: "aaaa", HasMarker: true, ModUnix: 10},
			{Digest: "bbbb", HasMarker: true, ModUnix: 1},
			{Digest: "cccc", HasMarker: true, ModUnix: 20},
		},
	})
	if err != nil {
		t.Fatalf("PlanCleanup error = %v", err)
	}
	if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != "aaaa" {
		t.Fatalf("DeleteDigests = %#v, want only aaaa", plan.DeleteDigests)
	}
}

func TestPlanCleanupIgnoresNonRuntimeDirectories(t *testing.T) {
	plan, err := PlanCleanup(CleanupInput{
		SelectedDigest: "bbbb",
		Keep:           0,
		Generations: []RuntimeGeneration{
			{Digest: "not-a-digest", HasMarker: true, ModUnix: 10},
			{Digest: "aaaa", HasMarker: false, ModUnix: 20},
			{Digest: "bbbb", HasMarker: true, ModUnix: 1},
			{Digest: "cccc", HasMarker: true, ModUnix: 30},
		},
	})
	if err != nil {
		t.Fatalf("PlanCleanup error = %v", err)
	}
	if len(plan.DeleteDigests) != 1 || plan.DeleteDigests[0] != "cccc" {
		t.Fatalf("DeleteDigests = %#v, want only cccc", plan.DeleteDigests)
	}
}
