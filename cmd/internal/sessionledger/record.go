package sessionledger

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type RecordKind string

const (
	RecordLaunch  RecordKind = "launch"
	RecordBinding RecordKind = "binding"
)

// NativeWatermark is the last accepted event position for one baseline root.
type NativeWatermark struct {
	RootNativeID  string `json:"root_native_id"`
	EventPosition uint64 `json:"event_position"`
}

// LaunchBaseline is content-free durable delimiting state captured pre-input.
// pair:155-concept pure new M2
type LaunchBaseline struct {
	PairLogOffset    uint64
	NativeWatermarks []NativeWatermark
}

// LaunchArtifactBoundary is a content-free exclusion watermark captured before
// launch input can create a native causal record.
// pair:156-concept pure new final
type LaunchArtifactBoundary struct {
	StorageRoot     string `json:"storage_root"`
	RelativePath    string `json:"relative_path"`
	StableFileID    string `json:"stable_file_id"`
	GenerationToken string `json:"generation_token,omitempty"`
	MutationToken   string `json:"mutation_token"`
	RawSize         int64  `json:"raw_size"`
}

// ArtifactProof records the exact native artifact generation and parser offset
// authorized by a binding.
// pair:156-concept pure new final
type ArtifactProof struct {
	StorageRoot          string `json:"storage_root"`
	RelativePath         string `json:"relative_path"`
	StableFileID         string `json:"stable_file_id"`
	GenerationToken      string `json:"generation_token,omitempty"`
	MutationToken        string `json:"mutation_token"`
	Size                 int64  `json:"size"`
	ParserCompleteOffset int64  `json:"parser_complete_offset"`
}

// AuthorizationProof is the scanner-owned authority that lets a binding
// survive loss of the derived catalog without replaying its transcript body.
// pair:156-concept pure new final AuthorizationProof / ArtifactProof
type AuthorizationProof struct {
	Version       int             `json:"version"`
	RootNativeID  string          `json:"root_native_id"`
	ScannerSchema string          `json:"scanner_schema"`
	ScannerState  json.RawMessage `json:"scanner_state"`
	Artifacts     []ArtifactProof `json:"artifacts"`
}

// LedgerRecord is the versioned launch/binding union. Ordinal is physical
// source position and is deliberately not encoded in the JSON row.
// pair:155-concept pure new M2
type LedgerRecord struct {
	Ordinal                  uint64
	Version                  int
	Kind                     RecordKind
	ScopeKey                 string
	Tag                      string
	Agent                    string
	PairLogOffset            uint64
	NativeWatermarks         []NativeWatermark
	LaunchArtifactBoundaries []LaunchArtifactBoundary
	LaunchOrdinal            uint64
	RootNativeID             string
	AuthorizationProof       *AuthorizationProof
}

type Record = LedgerRecord

type Owner struct {
	ScopeKey string
	Tag      string
	Agent    string
}

type ParseResult struct {
	Records               []Record
	CompatibilityOrdinals []uint64
	MalformedOrdinals     []uint64
}

type Current struct {
	Launch   Record
	Binding  *Record
	Bindings []Record
	Conflict bool
}

type wireRecord struct {
	Version            int                       `json:"v"`
	Kind               RecordKind                `json:"kind"`
	ScopeKey           string                    `json:"scope_key"`
	Tag                string                    `json:"tag"`
	Agent              string                    `json:"agent"`
	PairLogOffset      *uint64                   `json:"pair_log_offset,omitempty"`
	NativeWatermarks   *[]NativeWatermark        `json:"native_watermarks,omitempty"`
	ArtifactBoundaries *[]LaunchArtifactBoundary `json:"artifact_boundaries,omitempty"`
	LaunchOrdinal      *uint64                   `json:"launch_ordinal,omitempty"`
	RootNativeID       string                    `json:"root_native_id,omitempty"`
	AuthorizationProof *AuthorizationProof       `json:"authorization_proof,omitempty"`
}

type strictField[T any] struct {
	Value   T
	Present bool
}

func (field *strictField[T]) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("field must not be null")
	}
	if err := strictjson.Decode(raw, &field.Value); err != nil {
		return err
	}
	field.Present = true
	return nil
}

type nullableArgsField struct {
	Value   []string
	Present bool
}

func (field *nullableArgsField) UnmarshalJSON(raw []byte) error {
	if err := json.Unmarshal(raw, &field.Value); err != nil {
		return err
	}
	field.Present = true
	return nil
}

type compatibilityWireRecord struct {
	Agent        strictField[string] `json:"agent"`
	Args         nullableArgsField   `json:"args"`
	SessionID    strictField[string] `json:"session_id"`
	Started      strictField[string] `json:"started"`
	LastActive   strictField[string] `json:"last_active"`
	RepoRoot     strictField[string] `json:"repo_root"`
	RepoName     strictField[string] `json:"repo_name"`
	LegacyImport strictField[bool]   `json:"legacy_import"`
}

type decodeNativeWatermark struct {
	RootNativeID  strictField[string] `json:"root_native_id"`
	EventPosition strictField[uint64] `json:"event_position"`
}

type decodeLaunchArtifactBoundary struct {
	StorageRoot     strictField[string] `json:"storage_root"`
	RelativePath    strictField[string] `json:"relative_path"`
	StableFileID    strictField[string] `json:"stable_file_id"`
	GenerationToken strictField[string] `json:"generation_token"`
	MutationToken   strictField[string] `json:"mutation_token"`
	RawSize         strictField[int64]  `json:"raw_size"`
}

type decodeArtifactProof struct {
	StorageRoot          strictField[string] `json:"storage_root"`
	RelativePath         strictField[string] `json:"relative_path"`
	StableFileID         strictField[string] `json:"stable_file_id"`
	GenerationToken      strictField[string] `json:"generation_token"`
	MutationToken        strictField[string] `json:"mutation_token"`
	Size                 strictField[int64]  `json:"size"`
	ParserCompleteOffset strictField[int64]  `json:"parser_complete_offset"`
}

type decodeAuthorizationProof struct {
	Version       strictField[int]                   `json:"version"`
	RootNativeID  strictField[string]                `json:"root_native_id"`
	ScannerSchema strictField[string]                `json:"scanner_schema"`
	ScannerState  strictField[json.RawMessage]       `json:"scanner_state"`
	Artifacts     strictField[[]decodeArtifactProof] `json:"artifacts"`
}

type decodeWireRecord struct {
	Version            strictField[int]                            `json:"v"`
	Kind               strictField[RecordKind]                     `json:"kind"`
	ScopeKey           strictField[string]                         `json:"scope_key"`
	Tag                strictField[string]                         `json:"tag"`
	Agent              strictField[string]                         `json:"agent"`
	PairLogOffset      strictField[uint64]                         `json:"pair_log_offset"`
	NativeWatermarks   strictField[[]decodeNativeWatermark]        `json:"native_watermarks"`
	ArtifactBoundaries strictField[[]decodeLaunchArtifactBoundary] `json:"artifact_boundaries"`
	LaunchOrdinal      strictField[uint64]                         `json:"launch_ordinal"`
	RootNativeID       strictField[string]                         `json:"root_native_id"`
	AuthorizationProof strictField[decodeAuthorizationProof]       `json:"authorization_proof"`
}

func EncodeRecord(record Record) ([]byte, error) {
	record.NativeWatermarks = slices.Clone(record.NativeWatermarks)
	sortWatermarks(record.NativeWatermarks)
	record.LaunchArtifactBoundaries = slices.Clone(record.LaunchArtifactBoundaries)
	sortLaunchArtifactBoundaries(record.LaunchArtifactBoundaries)
	if record.AuthorizationProof != nil {
		proof := cloneAuthorizationProof(*record.AuthorizationProof)
		sortArtifactProofs(proof.Artifacts)
		record.AuthorizationProof = &proof
	}
	if err := validateRecord(record); err != nil {
		return nil, err
	}
	wire := wireRecord{Version: record.Version, Kind: record.Kind, ScopeKey: record.ScopeKey, Tag: record.Tag, Agent: record.Agent}
	switch record.Kind {
	case RecordLaunch:
		wire.PairLogOffset = &record.PairLogOffset
		if record.Version == 1 {
			watermarks := record.NativeWatermarks
			if watermarks == nil {
				watermarks = []NativeWatermark{}
			}
			wire.NativeWatermarks = &watermarks
		} else {
			boundaries := record.LaunchArtifactBoundaries
			if boundaries == nil {
				boundaries = []LaunchArtifactBoundary{}
			}
			wire.ArtifactBoundaries = &boundaries
		}
	case RecordBinding:
		wire.LaunchOrdinal = &record.LaunchOrdinal
		wire.RootNativeID = record.RootNativeID
		wire.AuthorizationProof = record.AuthorizationProof
	}
	return json.Marshal(wire)
}

func ParseLedger(raw []byte) ParseResult {
	lines := bytes.Split(raw, []byte{'\n'})
	terminated := len(raw) == 0 || raw[len(raw)-1] == '\n'
	if terminated && len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	result := ParseResult{}
	for i, line := range lines {
		ordinal := uint64(i + 1)
		if !terminated && i == len(lines)-1 {
			result.MalformedOrdinals = append(result.MalformedOrdinals, ordinal)
			continue
		}
		record, err := decodeRecord(line)
		if err == nil {
			record.Ordinal = ordinal
			result.Records = append(result.Records, record)
			continue
		}
		if isCompatibilityRecord(line) {
			result.CompatibilityOrdinals = append(result.CompatibilityOrdinals, ordinal)
			continue
		}
		result.MalformedOrdinals = append(result.MalformedOrdinals, ordinal)
	}
	return result
}

func isCompatibilityRecord(raw []byte) bool {
	var wire compatibilityWireRecord
	if err := strictjson.Decode(raw, &wire); err != nil {
		return false
	}
	if !wire.Agent.Present || !wire.Args.Present || !wire.SessionID.Present || !wire.Started.Present || !wire.LastActive.Present || !wire.RepoRoot.Present || !wire.RepoName.Present {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, wire.Started.Value); err != nil {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, wire.LastActive.Value); err != nil {
		return false
	}
	return isSupportedAgent(wire.Agent.Value)
}

func decodeRecord(raw []byte) (Record, error) {
	var wire decodeWireRecord
	if err := strictjson.Decode(raw, &wire); err != nil {
		return Record{}, err
	}
	if !wire.Version.Present || !wire.Kind.Present || !wire.ScopeKey.Present || !wire.Tag.Present || !wire.Agent.Present {
		return Record{}, errors.New("missing common ledger field")
	}
	record := Record{Version: wire.Version.Value, Kind: wire.Kind.Value, ScopeKey: wire.ScopeKey.Value, Tag: wire.Tag.Value, Agent: wire.Agent.Value}
	if wire.NativeWatermarks.Present {
		record.NativeWatermarks = make([]NativeWatermark, 0, len(wire.NativeWatermarks.Value))
		for _, watermark := range wire.NativeWatermarks.Value {
			if !watermark.RootNativeID.Present || !watermark.EventPosition.Present {
				return Record{}, errors.New("missing native watermark field")
			}
			record.NativeWatermarks = append(record.NativeWatermarks, NativeWatermark{RootNativeID: watermark.RootNativeID.Value, EventPosition: watermark.EventPosition.Value})
		}
	}
	if wire.PairLogOffset.Present {
		record.PairLogOffset = wire.PairLogOffset.Value
	}
	if wire.ArtifactBoundaries.Present {
		record.LaunchArtifactBoundaries = make([]LaunchArtifactBoundary, 0, len(wire.ArtifactBoundaries.Value))
		for _, boundary := range wire.ArtifactBoundaries.Value {
			if !boundary.StorageRoot.Present || !boundary.RelativePath.Present || !boundary.StableFileID.Present || !boundary.MutationToken.Present || !boundary.RawSize.Present {
				return Record{}, errors.New("missing launch artifact boundary field")
			}
			record.LaunchArtifactBoundaries = append(record.LaunchArtifactBoundaries, LaunchArtifactBoundary{
				StorageRoot: boundary.StorageRoot.Value, RelativePath: boundary.RelativePath.Value, StableFileID: boundary.StableFileID.Value,
				GenerationToken: boundary.GenerationToken.Value, MutationToken: boundary.MutationToken.Value, RawSize: boundary.RawSize.Value,
			})
		}
	}
	if wire.LaunchOrdinal.Present {
		record.LaunchOrdinal = wire.LaunchOrdinal.Value
	}
	if wire.RootNativeID.Present {
		record.RootNativeID = wire.RootNativeID.Value
	}
	if wire.AuthorizationProof.Present {
		proof, err := authorizationProofFromWire(wire.AuthorizationProof.Value)
		if err != nil {
			return Record{}, err
		}
		record.AuthorizationProof = &proof
	}
	sortWatermarks(record.NativeWatermarks)
	sortLaunchArtifactBoundaries(record.LaunchArtifactBoundaries)
	if (record.Kind == RecordLaunch) != wire.PairLogOffset.Present || (record.Kind == RecordBinding) != wire.LaunchOrdinal.Present || (record.Kind == RecordBinding) != wire.RootNativeID.Present {
		return Record{}, errors.New("missing or extraneous kind-specific field")
	}
	if record.Kind == RecordLaunch && ((record.Version == 1) != wire.NativeWatermarks.Present || (record.Version == 2) != wire.ArtifactBoundaries.Present || wire.AuthorizationProof.Present) {
		return Record{}, errors.New("invalid versioned launch fields")
	}
	if record.Kind == RecordBinding && (wire.NativeWatermarks.Present || wire.ArtifactBoundaries.Present || (record.Version == 2) != wire.AuthorizationProof.Present) {
		return Record{}, errors.New("invalid versioned binding fields")
	}
	if err := validateRecord(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func validateRecord(record Record) error {
	if (record.Version != 1 && record.Version != 2) || record.ScopeKey == "" || record.Tag == "" || !isSupportedAgent(record.Agent) {
		return errors.New("invalid common ledger fields")
	}
	switch record.Kind {
	case RecordLaunch:
		if record.Version == 1 {
			for i, watermark := range record.NativeWatermarks {
				if watermark.RootNativeID == "" || (i > 0 && record.NativeWatermarks[i-1].RootNativeID == watermark.RootNativeID) {
					return errors.New("invalid native watermark")
				}
			}
			if len(record.LaunchArtifactBoundaries) != 0 {
				return errors.New("v1 launch carries artifact boundaries")
			}
		} else if len(record.NativeWatermarks) != 0 {
			return errors.New("v2 launch carries native watermarks")
		} else if err := validateLaunchArtifactBoundaries(record.LaunchArtifactBoundaries); err != nil {
			return err
		}
		if record.LaunchOrdinal != 0 || record.RootNativeID != "" || record.AuthorizationProof != nil {
			return errors.New("launch carries binding fields")
		}
	case RecordBinding:
		if record.LaunchOrdinal == 0 || record.RootNativeID == "" || record.PairLogOffset != 0 || len(record.NativeWatermarks) != 0 || len(record.LaunchArtifactBoundaries) != 0 {
			return errors.New("invalid binding fields")
		}
		if record.Version == 1 && record.AuthorizationProof != nil {
			return errors.New("v1 binding carries authorization proof")
		}
		if record.Version == 2 {
			if record.AuthorizationProof == nil {
				return errors.New("v2 binding is missing authorization proof")
			}
			if err := ValidateAuthorizationProof(*record.AuthorizationProof, record.RootNativeID); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported ledger kind %q", record.Kind)
	}
	return nil
}

func ValidateAuthorizationProof(proof AuthorizationProof, rootNativeID string) error {
	if proof.Version != 1 || rootNativeID == "" || proof.RootNativeID != rootNativeID || proof.ScannerSchema == "" {
		return errors.New("invalid authorization proof identity")
	}
	trimmedState := bytes.TrimSpace(proof.ScannerState)
	if len(trimmedState) == 0 || bytes.Equal(trimmedState, []byte("null")) || !json.Valid(trimmedState) {
		return errors.New("invalid authorization proof scanner state")
	}
	if len(proof.Artifacts) == 0 {
		return errors.New("authorization proof has no artifacts")
	}
	artifacts := slices.Clone(proof.Artifacts)
	sortArtifactProofs(artifacts)
	for i, artifact := range artifacts {
		if artifact.StorageRoot == "" || artifact.RelativePath == "" || artifact.StableFileID == "" || artifact.MutationToken == "" || artifact.Size < 0 || artifact.ParserCompleteOffset != artifact.Size {
			return errors.New("invalid authorization artifact proof")
		}
		if i > 0 && artifact.StorageRoot == artifacts[i-1].StorageRoot && artifact.RelativePath == artifacts[i-1].RelativePath {
			return errors.New("duplicate authorization artifact proof")
		}
	}
	return nil
}

func validateLaunchArtifactBoundaries(boundaries []LaunchArtifactBoundary) error {
	for i, boundary := range boundaries {
		if boundary.StorageRoot == "" || boundary.RelativePath == "" || boundary.StableFileID == "" || boundary.MutationToken == "" || boundary.RawSize < 0 {
			return errors.New("invalid launch artifact boundary")
		}
		if i > 0 && boundary.StorageRoot == boundaries[i-1].StorageRoot && boundary.RelativePath == boundaries[i-1].RelativePath {
			return errors.New("duplicate launch artifact boundary")
		}
	}
	return nil
}

func authorizationProofFromWire(wire decodeAuthorizationProof) (AuthorizationProof, error) {
	if !wire.Version.Present || !wire.RootNativeID.Present || !wire.ScannerSchema.Present || !wire.ScannerState.Present || !wire.Artifacts.Present {
		return AuthorizationProof{}, errors.New("missing authorization proof field")
	}
	proof := AuthorizationProof{Version: wire.Version.Value, RootNativeID: wire.RootNativeID.Value, ScannerSchema: wire.ScannerSchema.Value, ScannerState: append(json.RawMessage(nil), wire.ScannerState.Value...)}
	for _, artifact := range wire.Artifacts.Value {
		if !artifact.StorageRoot.Present || !artifact.RelativePath.Present || !artifact.StableFileID.Present || !artifact.MutationToken.Present || !artifact.Size.Present || !artifact.ParserCompleteOffset.Present {
			return AuthorizationProof{}, errors.New("missing authorization artifact proof field")
		}
		proof.Artifacts = append(proof.Artifacts, ArtifactProof{
			StorageRoot: artifact.StorageRoot.Value, RelativePath: artifact.RelativePath.Value, StableFileID: artifact.StableFileID.Value,
			GenerationToken: artifact.GenerationToken.Value, MutationToken: artifact.MutationToken.Value, Size: artifact.Size.Value, ParserCompleteOffset: artifact.ParserCompleteOffset.Value,
		})
	}
	sortArtifactProofs(proof.Artifacts)
	return proof, nil
}

func cloneAuthorizationProof(proof AuthorizationProof) AuthorizationProof {
	proof.ScannerState = append(json.RawMessage(nil), proof.ScannerState...)
	proof.Artifacts = slices.Clone(proof.Artifacts)
	return proof
}

func sortLaunchArtifactBoundaries(boundaries []LaunchArtifactBoundary) {
	slices.SortFunc(boundaries, func(a, b LaunchArtifactBoundary) int {
		if result := cmp.Compare(a.StorageRoot, b.StorageRoot); result != 0 {
			return result
		}
		return cmp.Compare(a.RelativePath, b.RelativePath)
	})
}

func sortArtifactProofs(artifacts []ArtifactProof) {
	slices.SortFunc(artifacts, func(a, b ArtifactProof) int {
		if result := cmp.Compare(a.StorageRoot, b.StorageRoot); result != 0 {
			return result
		}
		return cmp.Compare(a.RelativePath, b.RelativePath)
	})
}

func isSupportedAgent(agent string) bool {
	switch agent {
	case "claude", "codex", "agy", "muse":
		return true
	default:
		return false
	}
}

func sortWatermarks(watermarks []NativeWatermark) {
	slices.SortFunc(watermarks, func(a, b NativeWatermark) int {
		if result := cmp.Compare(a.RootNativeID, b.RootNativeID); result != 0 {
			return result
		}
		return cmp.Compare(a.EventPosition, b.EventPosition)
	})
}

// CurrentLaunch returns only a binding joined to the latest physical launch
// generation for the exact owner. Differing bindings for one generation fail
// closed as a conflict.
func CurrentLaunch(records []Record, owner Owner) (Current, bool) {
	var current Current
	found := false
	for _, record := range records {
		if record.Kind != RecordLaunch || !recordOwner(record, owner) || record.Ordinal == 0 {
			continue
		}
		if !found || record.Ordinal > current.Launch.Ordinal {
			current = Current{Launch: record}
			found = true
		}
	}
	if !found {
		return Current{}, false
	}
	byRoot := map[string]Record{}
	for _, record := range records {
		if record.Kind != RecordBinding || !recordOwner(record, owner) || record.LaunchOrdinal != current.Launch.Ordinal || record.Ordinal <= current.Launch.Ordinal {
			continue
		}
		previous, exists := byRoot[record.RootNativeID]
		if !exists || record.Ordinal < previous.Ordinal {
			byRoot[record.RootNativeID] = record
		}
	}
	for _, record := range byRoot {
		current.Bindings = append(current.Bindings, record)
	}
	slices.SortFunc(current.Bindings, func(a, b Record) int {
		if result := cmp.Compare(a.RootNativeID, b.RootNativeID); result != 0 {
			return result
		}
		return cmp.Compare(a.Ordinal, b.Ordinal)
	})
	if len(current.Bindings) == 1 {
		copy := current.Bindings[0]
		current.Binding = &copy
	} else if len(current.Bindings) > 1 {
		current.Conflict = true
	}
	return current, true
}

func recordOwner(record Record, owner Owner) bool {
	return record.ScopeKey == owner.ScopeKey && record.Tag == owner.Tag && record.Agent == owner.Agent
}
