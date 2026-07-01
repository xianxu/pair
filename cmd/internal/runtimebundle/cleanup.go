package runtimebundle

import (
	"fmt"
	"sort"
)

type RuntimeGeneration struct {
	Digest    string
	HasMarker bool
	ModUnix   int64
}

type CleanupInput struct {
	SelectedDigest string
	Keep           int
	Generations    []RuntimeGeneration
}

type CleanupPlan struct {
	DeleteDigests []string
}

func PlanCleanup(input CleanupInput) (CleanupPlan, error) {
	if input.SelectedDigest == "" {
		return CleanupPlan{}, fmt.Errorf("selected digest is required")
	}
	if input.Keep < 0 {
		return CleanupPlan{}, fmt.Errorf("keep must be non-negative")
	}
	candidates := make([]RuntimeGeneration, 0, len(input.Generations))
	for _, gen := range input.Generations {
		if gen.Digest == input.SelectedDigest || !gen.HasMarker || !isDigestName(gen.Digest) {
			continue
		}
		candidates = append(candidates, gen)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ModUnix == candidates[j].ModUnix {
			return candidates[i].Digest > candidates[j].Digest
		}
		return candidates[i].ModUnix > candidates[j].ModUnix
	})
	if input.Keep >= len(candidates) {
		return CleanupPlan{}, nil
	}
	deleteCandidates := candidates[input.Keep:]
	sort.Slice(deleteCandidates, func(i, j int) bool {
		return deleteCandidates[i].Digest < deleteCandidates[j].Digest
	})
	plan := CleanupPlan{DeleteDigests: make([]string, 0, len(deleteCandidates))}
	for _, gen := range deleteCandidates {
		plan.DeleteDigests = append(plan.DeleteDigests, gen.Digest)
	}
	return plan, nil
}

func isDigestName(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
