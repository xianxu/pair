package runtimebundle

import (
	"fmt"
	"path/filepath"
)

type ExistingAsset struct {
	Mode   uint32
	Size   int64
	Digest string
}

type ExtractionInput struct {
	StoreRoot   string
	RuntimeRoot string
	Manifest    RuntimeManifest
	Existing    map[string]ExistingAsset
}

type ExtractionPlan struct {
	Writes []RuntimeAsset
	Skips  []string
}

func PlanExtraction(input ExtractionInput) (ExtractionPlan, error) {
	if err := input.Manifest.Validate(); err != nil {
		return ExtractionPlan{}, err
	}
	if err := validateRuntimeRoot(input.StoreRoot, input.RuntimeRoot); err != nil {
		return ExtractionPlan{}, err
	}
	existing := input.Existing
	if existing == nil {
		existing = map[string]ExistingAsset{}
	}
	plan := ExtractionPlan{}
	for _, asset := range input.Manifest.Assets {
		got, ok := existing[asset.Path]
		if ok && got.Mode == asset.Mode && got.Size == asset.Size && got.Digest == asset.Digest {
			plan.Skips = append(plan.Skips, asset.Path)
			continue
		}
		plan.Writes = append(plan.Writes, asset)
	}
	return plan, nil
}

func validateRuntimeRoot(storeRoot, runtimeRoot string) error {
	if storeRoot == "" || runtimeRoot == "" {
		return fmt.Errorf("store root and runtime root are required")
	}
	store := filepath.Clean(storeRoot)
	root := filepath.Clean(runtimeRoot)
	rel, err := filepath.Rel(store, root)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".." || rel == "" || (len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)) {
		return fmt.Errorf("runtime root %q is outside store root %q", runtimeRoot, storeRoot)
	}
	return nil
}
