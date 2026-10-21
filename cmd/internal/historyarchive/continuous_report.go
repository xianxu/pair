package historyarchive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const ContinuousReportVersion = 1
const MaxContinuousReportBytes = MaxMetadataBytes
const (
	ContinuousComplete       = "complete"
	ContinuousFailed         = "failed"
	ContinuousNotConfigured  = "not-configured"
	ContinuousAlreadyRunning = "already-running"
)

// ContinuousCommandReport is the versioned stdout result of a continuous command.
// pair:m5-concept pure
type ContinuousCommandReport struct {
	Version        int              `json:"version"`
	Outcome        string           `json:"outcome"`
	MaxDatabaseAge time.Duration    `json:"max_database_age_ns"`
	MaxDriveFiles  int              `json:"max_drive_files,omitempty"`
	Report         ContinuousReport `json:"report"`
}

// EncodeContinuousReport returns one bounded JSON document followed by a newline.
// It validates before returning any bytes so callers cannot emit a partial report.
func EncodeContinuousReport(r ContinuousCommandReport) ([]byte, error) {
	if err := validateContinuousCommandReport(r); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	if len(raw) >= MaxContinuousReportBytes {
		return nil, fmt.Errorf("continuous report exceeds %d bytes", MaxContinuousReportBytes)
	}
	return append(raw, '\n'), nil
}

// DecodeContinuousReport accepts additive fields but rejects ambiguous or invalid
// known data. The returned projection does not preserve unknown fields.
func DecodeContinuousReport(raw []byte) (ContinuousCommandReport, error) {
	var r ContinuousCommandReport
	if len(raw) > MaxContinuousReportBytes {
		return r, fmt.Errorf("continuous report exceeds %d bytes", MaxContinuousReportBytes)
	}
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return r, err
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return ContinuousCommandReport{}, err
	}
	if err := rejectContinuousScalarNulls(raw, reflect.TypeOf(r)); err != nil {
		return ContinuousCommandReport{}, err
	}
	if err := validateContinuousCommandReport(r); err != nil {
		return ContinuousCommandReport{}, err
	}
	return r, nil
}

func validateContinuousCommandReport(r ContinuousCommandReport) error {
	if r.Version != ContinuousReportVersion {
		return fmt.Errorf("unsupported continuous report version %d", r.Version)
	}
	switch r.Outcome {
	case ContinuousComplete, ContinuousFailed, ContinuousNotConfigured, ContinuousAlreadyRunning:
	default:
		return fmt.Errorf("invalid continuous outcome %q", r.Outcome)
	}
	if r.MaxDatabaseAge <= 0 {
		return fmt.Errorf("max database age must be positive")
	}
	noPass := r.Outcome == ContinuousNotConfigured || r.Outcome == ContinuousAlreadyRunning
	if r.MaxDriveFiles < 0 {
		return fmt.Errorf("max Drive files cannot be negative")
	}
	if r.Report.Drive != nil && (r.Report.Drive.Inventory != nil || len(r.Report.Drive.Inventories) != 0) {
		if noPass {
			return fmt.Errorf("%s cannot carry Drive inventory", r.Outcome)
		}
		if r.MaxDriveFiles == 0 {
			return fmt.Errorf("Drive inventory requires a positive file threshold")
		}
		if err := validateDriveInventories(r.Report.Drive); err != nil {
			return err
		}
	}
	if noPass && len(r.Report.Databases) != 0 {
		return fmt.Errorf("%s cannot carry database results", r.Outcome)
	}
	for kind, count := range r.Report.Archived {
		if err := validateKind(kind); err != nil {
			return err
		}
		if count < 0 || (noPass && count != 0) {
			return fmt.Errorf("invalid archive count for %s", r.Outcome)
		}
	}
	type identity struct {
		Source     SourceIdentity
		Kind, Path string
	}
	seen := map[identity]bool{}
	for i, row := range r.Report.Databases {
		if err := validateSource(row.Source); err != nil {
			return fmt.Errorf("database %d: %w", i, err)
		}
		if err := validateKind(row.Kind); err != nil {
			return fmt.Errorf("database %d: %w", i, err)
		}
		if !validPath(row.Path) {
			return fmt.Errorf("database %d: invalid relative path", i)
		}
		requireTimes := false
		switch row.Status {
		case DatabaseArchived, DatabaseUnchanged:
			requireTimes = true
		case DatabaseFailed, DatabaseDeferred:
		default:
			return fmt.Errorf("database %d: invalid status %q", i, row.Status)
		}
		for _, stamp := range []time.Time{row.CheckedAt, row.LastVerified} {
			if (requireTimes || !stamp.IsZero()) && !validTime(stamp) {
				return fmt.Errorf("database %d: invalid UTC evidence time", i)
			}
		}
		key := identity{row.Source, row.Kind, row.Path}
		if seen[key] {
			return fmt.Errorf("database %d: duplicate source, kind and path", i)
		}
		seen[key] = true
	}
	return nil
}

// encoding/json accepts null for scalars as an unchanged zero value. Reject it
// for known fields while allowing nil maps, slices and optional pointers as the
// encoder emits them. Unknown fields remain outside this protocol's authority.
func rejectContinuousScalarNulls(raw []byte, typ reflect.Type) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		switch typ.Kind() {
		case reflect.Map, reflect.Slice, reflect.Pointer:
			return nil
		default:
			return fmt.Errorf("null is invalid for %s", typ)
		}
	}
	switch typ.Kind() {
	case reflect.Pointer:
		return rejectContinuousScalarNulls(raw, typ.Elem())
	case reflect.Struct:
		if typ == reflect.TypeOf(time.Time{}) {
			return nil
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return err
		}
		if typ == reflect.TypeOf(ContinuousCommandReport{}) {
			for _, key := range []string{"version", "outcome", "max_database_age_ns", "report"} {
				if _, ok := fields[key]; !ok {
					return fmt.Errorf("missing required continuous report field %q", key)
				}
			}
		}
		if typ == reflect.TypeOf(DriveInventoryObservation{}) {
			for _, key := range []string{"destination", "folder_id", "phase", "files"} {
				if _, ok := fields[key]; !ok {
					return fmt.Errorf("missing required Drive inventory field %q", key)
				}
			}
		}
		seen := map[int]bool{}
		for key, value := range fields {
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				name := strings.Split(field.Tag.Get("json"), ",")[0]
				if name == "" {
					name = field.Name
				}
				if name != "-" && strings.EqualFold(key, name) {
					if seen[i] {
						return fmt.Errorf("duplicate aliases for continuous report field %q", name)
					}
					seen[i] = true
					if err := rejectContinuousScalarNulls(value, field.Type); err != nil {
						return fmt.Errorf("%s: %w", key, err)
					}
					break
				}
			}
		}
	case reflect.Map:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return err
		}
		for _, value := range fields {
			if err := rejectContinuousScalarNulls(value, typ.Elem()); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
		for _, value := range items {
			if err := rejectContinuousScalarNulls(value, typ.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}
