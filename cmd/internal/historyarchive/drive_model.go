package historyarchive

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
)

const MaxUploadReceiptBytes = 8 << 10

// DriveDestination names an operator-configured rclone remote and relative folder.
// pair:m5-concept pure
type DriveDestination struct{ Remote, Folder string }

func ParseDriveDestination(remote, folder string) (DriveDestination, error) {
	if remote == "" {
		remote = "pair-backup"
	}
	if folder == "" {
		folder = "backups/pair-history"
	}
	if len(remote) > 128 || remote[0] == '-' {
		return DriveDestination{}, errors.New("invalid Drive remote")
	}
	for _, r := range remote {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return DriveDestination{}, errors.New("invalid Drive remote")
		}
	}
	if len(folder) > 1024 || !validPath(folder) || strings.ContainsRune(folder, ':') {
		return DriveDestination{}, errors.New("invalid Drive folder")
	}
	for _, r := range folder {
		if unicode.IsControl(r) {
			return DriveDestination{}, errors.New("invalid Drive folder")
		}
	}
	return DriveDestination{remote, folder}, nil
}
func (d DriveDestination) String() string { return d.Remote + ":" + d.Folder }
func (d DriveDestination) Namespace() string {
	sum := sha256.Sum256([]byte(d.String()))
	return hex.EncodeToString(sum[:])
}
func (d DriveDestination) RemoteObjectNames(id string) (string, string, error) {
	if !validHex(id, 64) || strings.ToLower(id) != id {
		return "", "", errors.New("invalid bundle ID")
	}
	return id + ".tar.zst.gpg", id + ".json", nil
}

// UploadObject records exact bytes and the remote observation that verified them.
// pair:m5-concept pure
type UploadObject struct {
	Name         string                     `json:"name"`
	Size         int64                      `json:"size"`
	SHA256       string                     `json:"sha256"`
	MD5          string                     `json:"md5"`
	RemoteID     string                     `json:"remote_id"`
	Verification string                     `json:"verification"`
	Extensions   map[string]json.RawMessage `json:"-"`
}

// UploadReceipt attests a pair at one destination, separately from local authority.
// pair:m5-concept pure
type UploadReceipt struct {
	Version     int                        `json:"version"`
	Destination string                     `json:"destination"`
	FolderID    string                     `json:"folder_id"`
	BundleID    string                     `json:"bundle_id"`
	Bundle      UploadObject               `json:"bundle"`
	Label       UploadObject               `json:"label"`
	VerifiedAt  time.Time                  `json:"verified_at"`
	Extensions  map[string]json.RawMessage `json:"-"`
}

func validRemoteIdentity(s string) bool {
	if s == "" || len(s) > 1024 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func ValidateUploadReceipt(r UploadReceipt) error {
	remote, folder, ok := strings.Cut(r.Destination, ":")
	if !ok || remote == "" || folder == "" {
		return errors.New("invalid receipt destination")
	}
	d, err := ParseDriveDestination(remote, folder)
	if err != nil || d.String() != r.Destination {
		return errors.New("invalid receipt destination")
	}
	b, l, err := d.RemoteObjectNames(r.BundleID)
	if err != nil {
		return err
	}
	if r.Version != Version || !validTime(r.VerifiedAt) || !validRemoteIdentity(r.FolderID) {
		return errors.New("invalid receipt version, folder or verification time")
	}
	for _, v := range []struct {
		o    UploadObject
		name string
		max  int64
	}{{r.Bundle, b, MaxCiphertextBytes}, {r.Label, l, MaxMetadataBytes}} {
		o := v.o
		if o.Name != v.name || o.Size <= 0 || o.Size > v.max || !validHex(o.SHA256, 64) || !validHex(o.MD5, 32) || !validRemoteIdentity(o.RemoteID) {
			return errors.New("invalid receipt object identity")
		}
		switch o.Verification {
		case "backend-sha256", "backend-md5", "downloaded-sha256":
		default:
			return errors.New("invalid verification path")
		}
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(raw) > MaxUploadReceiptBytes {
		return errors.New("upload receipt exceeds 8 KiB")
	}
	return nil
}
func DecodeUploadReceipt(raw []byte) (UploadReceipt, error) {
	var r UploadReceipt
	if len(raw) > MaxUploadReceiptBytes {
		return r, errors.New("upload receipt exceeds 8 KiB")
	}
	if err := decodeJSONNoDuplicateKeys(raw, &r, true); err != nil {
		return r, err
	}
	return r, ValidateUploadReceipt(r)
}

// MatchReceipt compares all known observations, ignoring verification time and
// opaque extensions, whose semantics this version cannot interpret.
func MatchReceipt(got, want UploadReceipt) bool {
	if ValidateUploadReceipt(got) != nil || ValidateUploadReceipt(want) != nil {
		return false
	}
	return got.Version == want.Version && got.Destination == want.Destination &&
		got.FolderID == want.FolderID && got.BundleID == want.BundleID &&
		matchUploadObject(got.Bundle, want.Bundle) && matchUploadObject(got.Label, want.Label)
}

func matchUploadObject(got, want UploadObject) bool {
	return got.Name == want.Name && got.Size == want.Size && got.SHA256 == want.SHA256 &&
		got.MD5 == want.MD5 && got.RemoteID == want.RemoteID && got.Verification == want.Verification
}

// decodeJSONNoDuplicateKeys rejects case-fold-equivalent keys in every object, including
// nested arrays. Callers own byte bounds; nesting is bounded here.
func decodeJSONNoDuplicateKeys(raw []byte, v any, strict bool) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int) error
	walk = func(depth int) error {
		if depth > 64 {
			return errors.New("JSON nesting exceeds limit")
		}
		tok, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				s, ok := key.(string)
				if !ok {
					return errors.New("invalid JSON key")
				}
				folded := strings.Map(func(r rune) rune {
					min := r
					for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
						if next < min {
							min = next
						}
					}
					return min
				}, s)
				if seen[folded] {
					return errors.New("duplicate or invalid JSON key")
				}
				seen[folded] = true
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := walk(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := walk(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if strict {
		dec.DisallowUnknownFields()
	}
	return dec.Decode(v)
}

// UploadPhase names the states of one pair within a pass-wide transfer batch.
type UploadPhase uint8

const (
	UploadInventoryBefore UploadPhase = iota
	UploadPreparedForBatch
	UploadInventoryAfter
	UploadReadyToRecord
	UploadVerified
	UploadPending
	UploadRefused
)

// pair:m5-concept pure
type UploadState struct {
	Destination, BundleID string
	Phase                 UploadPhase
}
type UploadEventKind uint8

const (
	UploadPrepared UploadEventKind = iota
	UploadBatchObserved
	UploadFinalMatch
	UploadRecorded
	UploadConflict
	UploadUnknown
	UploadCancel
)

// pair:m5-concept pure
type UploadEvent struct {
	Destination, BundleID       string
	Kind                        UploadEventKind
	BundleMatched, LabelMatched bool
}
type UploadEffect uint8

const (
	UploadNone UploadEffect = iota
	UploadFinalInventory
	UploadRecordReceipt
	UploadReport
)

// NextUpload admits receipts from final pair checks, never from batch exit status.
// UploadBatchObserved includes failed or uncertain transfer requests: the final
// inventory determines which individual pairs can safely be adopted.
func NextUpload(s UploadState, e UploadEvent) (UploadState, UploadEffect, error) {
	bad := func() (UploadState, UploadEffect, error) {
		return s, UploadNone, fmt.Errorf("invalid upload event %d in phase %d", e.Kind, s.Phase)
	}
	if s.Destination == "" || s.BundleID == "" || e.Destination != s.Destination || e.BundleID != s.BundleID || s.Phase >= UploadVerified {
		return bad()
	}
	switch e.Kind {
	case UploadUnknown, UploadCancel:
		s.Phase = UploadPending
		return s, UploadReport, nil
	case UploadConflict:
		s.Phase = UploadRefused
		return s, UploadReport, nil
	}
	switch s.Phase {
	case UploadInventoryBefore:
		if e.Kind == UploadPrepared {
			s.Phase = UploadPreparedForBatch
			return s, UploadNone, nil
		}
	case UploadPreparedForBatch:
		if e.Kind == UploadBatchObserved {
			s.Phase = UploadInventoryAfter
			return s, UploadFinalInventory, nil
		}
	case UploadInventoryAfter:
		if e.Kind == UploadFinalMatch && e.BundleMatched && e.LabelMatched {
			s.Phase = UploadReadyToRecord
			return s, UploadRecordReceipt, nil
		}
	case UploadReadyToRecord:
		if e.Kind == UploadRecorded {
			s.Phase = UploadVerified
			return s, UploadNone, nil
		}
	}
	return bad()
}

// AdmitBatchPair consumes capacity for both objects atomically, or leaves it
// unchanged. Subtraction keeps even caller-supplied near-MaxInt64 limits safe.
func AdmitBatchPair(used, bundleBytes, labelBytes, limit int64) (next int64, admit bool, err error) {
	if limit < 0 || used < 0 || used > limit || bundleBytes <= 0 || bundleBytes > MaxCiphertextBytes || labelBytes <= 0 || labelBytes > MaxMetadataBytes {
		return used, false, errors.New("invalid batch pair bounds")
	}
	remaining := limit - used
	if bundleBytes > remaining || labelBytes > remaining-bundleBytes {
		return used, false, nil
	}
	return used + bundleBytes + labelBytes, true, nil
}
