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

type strictField[T any] struct {
	Value   T
	Present bool
}

func (field *strictField[T]) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("field must not be null")
	}
	if err := json.Unmarshal(raw, &field.Value); err != nil {
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

type decodeWireRecord struct {
	Version          strictField[int]                     `json:"v"`
	Kind             strictField[RecordKind]              `json:"kind"`
	ScopeKey         strictField[string]                  `json:"scope_key"`
	Tag              strictField[string]                  `json:"tag"`
	Agent            strictField[string]                  `json:"agent"`
	PairLogOffset    strictField[uint64]                  `json:"pair_log_offset"`
	NativeWatermarks strictField[[]decodeNativeWatermark] `json:"native_watermarks"`
	LaunchOrdinal    strictField[uint64]                  `json:"launch_ordinal"`
	RootNativeID     strictField[string]                  `json:"root_native_id"`
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
	if wire.LaunchOrdinal.Present {
		record.LaunchOrdinal = wire.LaunchOrdinal.Value
	}
	if wire.RootNativeID.Present {
		record.RootNativeID = wire.RootNativeID.Value
	}
	sortWatermarks(record.NativeWatermarks)
	if (record.Kind == RecordLaunch) != wire.PairLogOffset.Present || (record.Kind == RecordLaunch) != wire.NativeWatermarks.Present || (record.Kind == RecordBinding) != wire.LaunchOrdinal.Present || (record.Kind == RecordBinding) != wire.RootNativeID.Present {
		return Record{}, errors.New("missing or extraneous kind-specific field")
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
