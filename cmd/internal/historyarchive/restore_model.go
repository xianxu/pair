package historyarchive

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	MaxRestoreIDs               = 64
	MaxRestoreCipherBytes int64 = 4 << 30
	MaxRestoreRawBytes    int64 = 16 << 30
	MaxRestoreEntries           = 100000
	MaxRestoreFiles             = 100000
	MaxRestoreTarOverhead int64 = 32 << 20
)

// RestoreExtractionBudget tracks request-wide created entries and processed
// files. Created directories consume entry capacity as well as regular files.
// pair:m5-concept pure
type RestoreExtractionBudget struct{ Entries, Files int }

// RestoreRequest selects known bundle IDs and an explicit new output path.
// Paths are literal relative file names; an empty list selects every file.
// pair:m5-concept pure
type RestoreRequest struct {
	IDs   []string
	To    string
	From  string
	Paths []string
}

// RestoreBundlePlan carries discovery metadata, never authority to skip hashes.
// BundleKey and LabelKey are the expected local layout keys.
// pair:m5-concept pure
type RestoreBundlePlan struct {
	Label               BundleLabel
	BundleKey, LabelKey string
}

// pair:m5-concept pure
type RestorePlan struct {
	Request               RestoreRequest
	Bundles               []RestoreBundlePlan
	CipherBytes, RawBytes int64
}

// PlanRestore admits whole-bundle resource costs before acquisition. Availability
// and historic verification are discovery facts; acquisition must verify bytes.
func PlanRestore(index LabelIndex, req RestoreRequest) (RestorePlan, error) {
	if err := ValidateLabelIndex(index); err != nil {
		return RestorePlan{}, err
	}
	if !filepath.IsAbs(req.To) || req.To == string(filepath.Separator) || filepath.Clean(req.To) != req.To || strings.ContainsRune(req.To, 0) {
		return RestorePlan{}, errors.New("restore requires an explicit absolute clean non-root output path")
	}
	switch req.From {
	case "", "auto", "local", "drive":
	default:
		return RestorePlan{}, errors.New("restore source must be auto, local or drive")
	}
	plan := RestorePlan{Request: req}
	plan.Request.IDs, plan.Request.Paths = nil, nil
	paths := map[string]bool{}
	for _, p := range req.Paths {
		if !validPath(p) {
			return RestorePlan{}, fmt.Errorf("invalid literal restore path %q", p)
		}
		if !paths[p] {
			paths[p] = true
			plan.Request.Paths = append(plan.Request.Paths, p)
		}
	}
	ids := map[string]bool{}
	files := 0
	for _, id := range req.IDs {
		if !validHex(id, 64) || strings.ToLower(id) != id {
			return RestorePlan{}, errors.New("restore requires full lowercase 64-hex bundle IDs")
		}
		if ids[id] {
			continue
		}
		if len(ids) >= MaxRestoreIDs {
			return RestorePlan{}, errors.New("restore exceeds 64 unique bundle IDs; select fewer bundles")
		}
		record, known := index.Records[id]
		if !known {
			return RestorePlan{}, fmt.Errorf("unknown bundle ID %s; use find or sync-index first", id)
		}
		label := record.Label
		if label.Bytes > MaxRestoreCipherBytes-plan.CipherBytes || label.RawBytes > MaxRestoreRawBytes-plan.RawBytes || label.FileCount > MaxRestoreFiles-files {
			return RestorePlan{}, errors.New("restore exceeds aggregate ciphertext, raw byte or file limits; select fewer bundles")
		}
		b, l, err := BundleKeys(label.Kind, label.From, label.To, id)
		if err != nil {
			return RestorePlan{}, err
		}
		label.Recipients = append([]string(nil), label.Recipients...)
		label.Extensions, err = mergeExtensions(nil, label.Extensions)
		if err != nil {
			return RestorePlan{}, err
		}
		ids[id] = true
		files += label.FileCount
		plan.CipherBytes += label.Bytes
		plan.RawBytes += label.RawBytes
		plan.Request.IDs = append(plan.Request.IDs, id)
		plan.Bundles = append(plan.Bundles, RestoreBundlePlan{Label: label, BundleKey: b, LabelKey: l})
	}
	if len(plan.Bundles) == 0 {
		return RestorePlan{}, errors.New("restore requires at least one bundle ID; use find first")
	}
	return plan, nil
}

// RestorePhase records validated execution milestones, independent of index proof.
// pair:m5-concept pure
type RestorePhase uint8

const (
	RestorePlanned RestorePhase = iota
	RestoreInputsVerified
	RestoreExtracting
	RestoreValidated
	RestorePublished
	RestoreRefused
	RestoreCancelled
	RestorePublicationUncertain
)

// pair:m5-concept pure
type RestoreState struct{ Phase RestorePhase }

// RestoreEvent advances a validated execution or records a terminal failure.
// pair:m5-concept pure
type RestoreEvent uint8

const (
	RestoreVerifyInputs RestoreEvent = iota
	RestoreBeginExtraction
	// RestoreValidateOutput requires every member and decoder exit to succeed.
	RestoreValidateOutput
	RestorePublishOutput
	RestoreFail
	RestoreCancel
	RestorePublishUncertain
)

func NextRestore(state RestoreState, event RestoreEvent) (RestoreState, error) {
	if state.Phase <= RestoreValidated {
		switch event {
		case RestoreFail:
			return RestoreState{Phase: RestoreRefused}, nil
		case RestoreCancel:
			return RestoreState{Phase: RestoreCancelled}, nil
		}
		next := map[RestorePhase]struct {
			event RestoreEvent
			phase RestorePhase
		}{
			RestorePlanned:        {RestoreVerifyInputs, RestoreInputsVerified},
			RestoreInputsVerified: {RestoreBeginExtraction, RestoreExtracting},
			RestoreExtracting:     {RestoreValidateOutput, RestoreValidated},
			RestoreValidated:      {RestorePublishOutput, RestorePublished},
		}
		transition := next[state.Phase]
		if transition.event == event {
			return RestoreState{Phase: transition.phase}, nil
		}
		if state.Phase == RestoreValidated && event == RestorePublishUncertain {
			return RestoreState{Phase: RestorePublicationUncertain}, nil
		}
	}
	return state, fmt.Errorf("invalid restore event %d in phase %d", event, state.Phase)
}
