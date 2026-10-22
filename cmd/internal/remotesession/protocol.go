package remotesession

import (
	"bytes"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type RemoteAction string

const (
	ActionRead     RemoteAction = "read"
	ActionEdit     RemoteAction = "edit"
	ActionSubmit   RemoteAction = "submit"
	ActionNavigate RemoteAction = "navigate"
)

// DocumentRef is an opaque registration plus exact editor generation. It is
// never a filesystem path, socket, nvim buffer number or arbitrary RPC target.
type DocumentRef struct {
	ID         string `json:"id"`
	Generation string `json:"generation"`
}
type DocumentEdit struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
}
type NavigationTarget struct {
	Kind string `json:"kind"`
	Key  string `json:"key,omitempty"`
}
type DocumentRequest struct {
	Sequence     uint64            `json:"sequence"`
	Action       RemoteAction      `json:"action"`
	Document     DocumentRef       `json:"document"`
	BaseRevision string            `json:"base_revision,omitempty"`
	Edit         *DocumentEdit     `json:"edit,omitempty"`
	Navigation   *NavigationTarget `json:"navigation,omitempty"`
	destination  DocumentRef
}

// EffectDocument returns no target for unresolved navigation. Only the
// supervisor can install a destination after resolving and authorizing it.
func (r DocumentRequest) EffectDocument() DocumentRef {
	if r.Action == ActionNavigate {
		return r.destination
	}
	return r.Document
}
func requestObject(raw []byte, known, required []string) (map[string]json.RawMessage, error) {
	fields, err := remoteObject(raw, known...)
	if err != nil {
		return nil, err
	}
	for _, key := range known {
		if value, present := fields[key]; present && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, errors.New("null remote request field")
		}
	}
	for _, key := range required {
		if _, present := fields[key]; !present {
			return nil, errors.New("missing remote request field")
		}
	}
	return fields, nil
}
func (r DocumentRef) valid() bool {
	return validDocumentID(r.ID) && r.Generation != "" && auditOpaque(r.Generation, 128)
}
func decodeDocumentRequest(raw []byte) (DocumentRequest, error) {
	var r DocumentRequest
	if len(raw) > 64*1024 || !utf8.Valid(raw) {
		return r, errors.New("remote request exceeds limit")
	}
	if err := strictjson.RejectDuplicateKeys(raw); err != nil {
		return r, err
	}
	fields, err := requestObject(raw, []string{"sequence", "action", "document", "base_revision", "edit", "navigation"}, []string{"sequence", "action", "document"})
	if err != nil {
		return r, err
	}
	if _, err := requestObject(fields["document"], []string{"id", "generation"}, []string{"id", "generation"}); err != nil {
		return r, err
	}
	if edit, present := fields["edit"]; present {
		if _, err := requestObject(edit, []string{"start", "end", "text"}, []string{"start", "end", "text"}); err != nil {
			return r, err
		}
	}
	if nav, present := fields["navigation"]; present {
		if _, err := requestObject(nav, []string{"kind", "key"}, []string{"kind"}); err != nil {
			return r, err
		}
	}
	if err := strictjson.Decode(raw, &r); err != nil {
		return r, err
	}
	if r.Sequence == 0 || !r.Document.valid() {
		return r, errors.New("invalid remote request identity")
	}
	switch r.Action {
	case ActionRead:
		if r.Edit != nil || r.Navigation != nil || r.BaseRevision != "" {
			return r, errors.New("invalid remote read")
		}
	case ActionEdit:
		if r.Edit == nil || r.Navigation != nil || r.Edit.Start < 0 || r.Edit.End < r.Edit.Start || r.Edit.End > 4<<20 || !utf8.ValidString(r.Edit.Text) {
			return r, errors.New("invalid remote edit")
		}
		for _, c := range r.Edit.Text {
			if c == 0 {
				return r, errors.New("remote edit contains NUL")
			}
		}
	case ActionSubmit:
		if r.Edit != nil || r.Navigation != nil {
			return r, errors.New("invalid remote submit")
		}
	case ActionNavigate:
		if r.Edit != nil || r.Navigation == nil {
			return r, errors.New("invalid remote navigation")
		}
		switch r.Navigation.Kind {
		case "link":
			if r.Navigation.Key == "" || !auditOpaque(r.Navigation.Key, 128) {
				return r, errors.New("invalid remote navigation key")
			}
		case "parent", "back", "forward":
			if r.Navigation.Key != "" {
				return r, errors.New("unexpected remote navigation key")
			}
		default:
			return r, errors.New("remote navigation kind refused")
		}
	default:
		return r, errors.New("remote action refused")
	}
	if r.Action != ActionRead && (r.BaseRevision == "" || !auditOpaque(r.BaseRevision, 128)) {
		return r, errors.New("invalid remote base revision")
	}
	return r, nil
}
