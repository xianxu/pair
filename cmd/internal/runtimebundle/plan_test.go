package runtimebundle

import (
	"path/filepath"
	"testing"
)

func TestPlanExtractionWritesMissingAssets(t *testing.T) {
	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-wrap", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}

	plan, err := PlanExtraction(ExtractionInput{
		StoreRoot:   "/data/pair/runtime",
		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
		Manifest:    manifest,
		Existing:    map[string]ExistingAsset{},
	})
	if err != nil {
		t.Fatalf("PlanExtraction error = %v", err)
	}
	if len(plan.Writes) != 1 || plan.Writes[0].Path != "bin/pair-wrap" {
		t.Fatalf("Writes = %#v, want bin/pair-wrap", plan.Writes)
	}
	if len(plan.Skips) != 0 {
		t.Fatalf("Skips = %#v, want empty", plan.Skips)
	}
}

func TestPlanExtractionSkipsMatchingAssets(t *testing.T) {
	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-wrap", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}

	plan, err := PlanExtraction(ExtractionInput{
		StoreRoot:   "/data/pair/runtime",
		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
		Manifest:    manifest,
		Existing: map[string]ExistingAsset{
			"bin/pair-wrap": {Mode: 0o755, Size: 10, Digest: "sha256:a"},
		},
	})
	if err != nil {
		t.Fatalf("PlanExtraction error = %v", err)
	}
	if len(plan.Writes) != 0 {
		t.Fatalf("Writes = %#v, want empty", plan.Writes)
	}
	if len(plan.Skips) != 1 || plan.Skips[0] != "bin/pair-wrap" {
		t.Fatalf("Skips = %#v, want bin/pair-wrap", plan.Skips)
	}
}

func TestPlanExtractionRefreshesMismatchedDigest(t *testing.T) {
	manifest := RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-wrap", Mode: 0o755, Size: 10, Digest: "sha256:a"}}}

	plan, err := PlanExtraction(ExtractionInput{
		StoreRoot:   "/data/pair/runtime",
		RuntimeRoot: "/data/pair/runtime/abc/pair-home",
		Manifest:    manifest,
		Existing: map[string]ExistingAsset{
			"bin/pair-wrap": {Mode: 0o755, Size: 10, Digest: "sha256:old"},
		},
	})
	if err != nil {
		t.Fatalf("PlanExtraction error = %v", err)
	}
	if len(plan.Writes) != 1 || plan.Writes[0].Path != "bin/pair-wrap" {
		t.Fatalf("Writes = %#v, want refresh", plan.Writes)
	}
}

func TestPlanExtractionRejectsRuntimeRootOutsideStore(t *testing.T) {
	_, err := PlanExtraction(ExtractionInput{
		StoreRoot:   "/data/pair/runtime",
		RuntimeRoot: filepath.Clean("/data/pair/not-runtime/abc/pair-home"),
		Manifest:    RuntimeManifest{Assets: []RuntimeAsset{{Path: "bin/pair-wrap", Mode: 0o755, Digest: "sha256:a"}}},
	})
	if err == nil {
		t.Fatal("PlanExtraction error = nil, want root containment error")
	}
}
