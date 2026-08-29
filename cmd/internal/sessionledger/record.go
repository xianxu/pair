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

// LedgerRecord is the versioned launch/binding union. Ordinal is physical
// source position and is deliberately not encoded in the JSON row.
// pair:155-concept pure new M2
type LedgerRecord struct {
	Ordinal          uint64
	Version          int
	Kind             RecordKind
	ScopeKey         string
	Tag              string
	Agent            string
	PairLogOffset    uint64
	NativeWatermarks []NativeWatermark
	LaunchOrdinal    uint64
	RootNativeID     string
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
	Version          int                `json:"v"`
	Kind             RecordKind         `json:"kind"`
	ScopeKey         string             `json:"scope_key"`
	Tag              string             `json:"tag"`
	Agent            string             `json:"agent"`
	PairLogOffset    *uint64            `json:"pair_log_offset,omitempty"`
	NativeWatermarks *[]NativeWatermark `json:"native_watermarks,omitempty"`
	LaunchOrdinal    *uint64            `json:"launch_ordinal,omitempty"`
	RootNativeID     string             `json:"root_native_id,omitempty"`
}

type compatibilityWireRecord struct {
	Agent        *string         `json:"agent"`
	Args         json.RawMessage `json:"args"`
	SessionID    *string         `json:"session_id"`
	Started      *string         `json:"started"`
	LastActive   *string         `json:"last_active"`
	RepoRoot     *string         `json:"repo_root"`
	RepoName     *string         `json:"repo_name"`
	LegacyImport *bool           `json:"legacy_import,omitempty"`
}

func EncodeRecord(record Record) ([]byte, error) {
	record.NativeWatermarks = slices.Clone(record.NativeWatermarks)
	sortWatermarks(record.NativeWatermarks)
	if err := validateRecord(record); err != nil {
		return nil, err
	}
	wire := wireRecord{Version: record.Version, Kind: record.Kind, ScopeKey: record.ScopeKey, Tag: record.Tag, Agent: record.Agent}
	switch record.Kind {
	case RecordLaunch:
		wire.PairLogOffset = &record.PairLogOffset
		watermarks := record.NativeWatermarks
		if watermarks == nil {
			watermarks = []NativeWatermark{}
		}
		wire.NativeWatermarks = &watermarks
	case RecordBinding:
		wire.LaunchOrdinal = &record.LaunchOrdinal
		wire.RootNativeID = record.RootNativeID
	}
	return json.Marshal(wire)
}

func ParseLedger(raw []byte) ParseResult {
	lines := bytes.Split(raw, []byte{'\n'})
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	result := ParseResult{}
	for i, line := range lines {
		ordinal := uint64(i + 1)
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
	var args []string
	if wire.Agent == nil || len(wire.Args) == 0 || json.Unmarshal(wire.Args, &args) != nil || wire.SessionID == nil || wire.Started == nil || wire.LastActive == nil || wire.RepoRoot == nil || wire.RepoName == nil {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, *wire.Started); err != nil {
		return false
	}
	if _, err := time.Parse(time.RFC3339Nano, *wire.LastActive); err != nil {
		return false
	}
	return isSupportedAgent(*wire.Agent)
}

func decodeRecord(raw []byte) (Record, error) {
	var wire wireRecord
	if err := strictjson.Decode(raw, &wire); err != nil {
		return Record{}, err
	}
	record := Record{Version: wire.Version, Kind: wire.Kind, ScopeKey: wire.ScopeKey, Tag: wire.Tag, Agent: wire.Agent, RootNativeID: wire.RootNativeID}
	if wire.NativeWatermarks != nil {
		record.NativeWatermarks = slices.Clone(*wire.NativeWatermarks)
	}
	if wire.PairLogOffset != nil {
		record.PairLogOffset = *wire.PairLogOffset
	}
	if wire.LaunchOrdinal != nil {
		record.LaunchOrdinal = *wire.LaunchOrdinal
	}
	sortWatermarks(record.NativeWatermarks)
	if (record.Kind == RecordLaunch) != (wire.PairLogOffset != nil) || (record.Kind == RecordBinding) != (wire.LaunchOrdinal != nil) {
		return Record{}, errors.New("missing or extraneous kind-specific field")
	}
	if record.Kind == RecordLaunch && wire.RootNativeID != "" || record.Kind == RecordBinding && (wire.PairLogOffset != nil || wire.NativeWatermarks != nil) {
		return Record{}, errors.New("extraneous kind-specific field")
	}
	if err := validateRecord(record); err != nil {
		return Record{}, err
	}
	return record, nil
}

func validateRecord(record Record) error {
	if record.Version != 1 || record.ScopeKey == "" || record.Tag == "" || !isSupportedAgent(record.Agent) {
		return errors.New("invalid common ledger fields")
	}
	switch record.Kind {
	case RecordLaunch:
		for i, watermark := range record.NativeWatermarks {
			if watermark.RootNativeID == "" || (i > 0 && record.NativeWatermarks[i-1].RootNativeID == watermark.RootNativeID) {
				return errors.New("invalid native watermark")
			}
		}
		if record.LaunchOrdinal != 0 || record.RootNativeID != "" {
			return errors.New("launch carries binding fields")
		}
	case RecordBinding:
		if record.LaunchOrdinal == 0 || record.RootNativeID == "" || record.PairLogOffset != 0 || len(record.NativeWatermarks) != 0 {
			return errors.New("invalid binding fields")
		}
	default:
		return fmt.Errorf("unsupported ledger kind %q", record.Kind)
	}
	return nil
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
