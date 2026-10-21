package remotesession

import (
	"encoding/json"
	"errors"
	"time"
)

const maxAuditRecordBytes = 2048

// AuditRecord accepts metadata only. It is never an authorization source.
// The writer assigns the intent/result kind; callers cannot select it.
type AuditRecord struct {
	At         time.Time
	Action     string
	Principal  string
	Session    string
	Epoch      uint64
	DocumentID string
	Sequence   uint64
	Before     string
	After      string
	Outcome    string
	ReceiptID  string
	kind       string
	raw        map[string]json.RawMessage
	metadata   map[string]json.RawMessage
}

func auditOpaque(s string, max int) bool {
	if len(s) > max {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == ':') {
			return false
		}
	}
	return true
}
func auditHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
func auditOwned(r *AuditRecord) []struct {
	key    string
	target any
} {
	return []struct {
		key    string
		target any
	}{{"at", &r.At}, {"action", &r.Action}, {"principal", &r.Principal}, {"session", &r.Session}, {"epoch", &r.Epoch}, {"document_id", &r.DocumentID}, {"sequence", &r.Sequence}, {"before", &r.Before}, {"after", &r.After}, {"outcome", &r.Outcome}, {"receipt_id", &r.ReceiptID}}
}
func decodeAuditRecord(raw []byte) (AuditRecord, error) {
	var r AuditRecord
	if err := validateRemoteJSON(raw, maxAuditRecordBytes); err != nil {
		return r, err
	}
	root, err := remoteObject(raw, "version", "kind", "metadata")
	if err != nil {
		return r, err
	}
	var version int
	if err = requiredRemoteField(root, "version", &version); err != nil {
		return r, err
	}
	if version != 1 {
		return r, errors.New("unsupported audit version")
	}
	if err = requiredRemoteField(root, "kind", &r.kind); err != nil {
		return r, err
	}
	metadata, err := remoteObject(root["metadata"], "at", "action", "principal", "session", "epoch", "document_id", "sequence", "before", "after", "outcome", "receipt_id")
	if err != nil {
		return r, err
	}
	for _, f := range auditOwned(&r) {
		if err := requiredRemoteField(metadata, f.key, f.target); err != nil {
			return r, err
		}
	}
	if r.Epoch == 0 || r.Sequence == 0 || r.At.IsZero() || r.At.Location() != time.UTC || r.At.After(time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)) || !auditHash(r.Principal) || !auditHash(r.Session) || !validDocumentID(r.DocumentID) || !auditOpaque(r.Before, 128) || !auditOpaque(r.After, 128) || !auditOpaque(r.ReceiptID, 128) {
		return r, errors.New("invalid audit metadata")
	}
	switch r.Action {
	case "read", "edit", "submit", "navigate":
	default:
		return r, errors.New("invalid audit action")
	}
	if r.kind != "intent" && r.kind != "result" {
		return r, errors.New("invalid audit kind")
	}
	switch r.Outcome {
	case "pending", "applied", "refused", "uncertain":
	default:
		return r, errors.New("invalid audit outcome")
	}
	if r.kind == "intent" && r.Outcome != "pending" || r.kind == "result" && r.Outcome == "pending" {
		return r, errors.New("audit kind/outcome mismatch")
	}
	r.raw, r.metadata = root, metadata
	return r, nil
}
func encodeAuditRecord(r AuditRecord) ([]byte, error) {
	root := make(map[string]json.RawMessage, len(r.raw)+3)
	for k, v := range r.raw {
		root[k] = v
	}
	meta := make(map[string]json.RawMessage, len(r.metadata)+11)
	for k, v := range r.metadata {
		meta[k] = v
	}
	for _, f := range auditOwned(&r) {
		raw, err := json.Marshal(f.target)
		if err != nil {
			return nil, err
		}
		meta[f.key] = raw
	}
	root["version"] = json.RawMessage("1")
	root["kind"], _ = json.Marshal(r.kind)
	root["metadata"] = marshalRemoteFields(meta)
	raw := marshalRemoteFields(root)
	if _, err := decodeAuditRecord(raw); err != nil {
		return nil, err
	}
	return raw, nil
}
