package historyarchive

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// StagingOwner is the durable authority for recovering an interrupted stage.
// Unknown fields survive decoding and encoding; incompatible cleanup semantics
// require a new Version, which this implementation refuses.
type StagingOwner struct {
	Version      int                        `json:"version"`
	Namespace    string                     `json:"namespace"`
	Nonce        string                     `json:"nonce"`
	ParentDevice uint64                     `json:"parent_device"`
	ParentInode  uint64                     `json:"parent_inode"`
	Extensions   map[string]json.RawMessage `json:"-"`
}

func (o *StagingOwner) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	token, err := dec.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return errors.New("staging owner must be an object")
	}
	fields := make(map[string]json.RawMessage)
	for dec.More() {
		token, err = dec.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return errors.New("invalid staging owner key")
		}
		if _, ok = fields[key]; ok {
			return errors.New("duplicate staging owner field")
		}
		var raw json.RawMessage
		if err = dec.Decode(&raw); err != nil {
			return err
		}
		fields[key] = raw
	}
	if _, err = dec.Token(); err != nil {
		return err
	}
	if err = dec.Decode(new(any)); err != io.EOF {
		return errors.New("trailing staging owner data")
	}
	var out StagingOwner
	for key, target := range map[string]any{"version": &out.Version, "namespace": &out.Namespace, "nonce": &out.Nonce, "parent_device": &out.ParentDevice, "parent_inode": &out.ParentInode} {
		raw, ok := fields[key]
		if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return errors.New("missing staging owner field")
		}
		if err = json.Unmarshal(raw, target); err != nil {
			return err
		}
		delete(fields, key)
	}
	out.Extensions = fields
	*o = out
	return nil
}

func (o StagingOwner) MarshalJSON() ([]byte, error) {
	fields := make(map[string]json.RawMessage, len(o.Extensions)+5)
	for key, value := range o.Extensions {
		fields[key] = value
	}
	for key, value := range map[string]any{"version": o.Version, "namespace": o.Namespace, "nonce": o.Nonce, "parent_device": o.ParentDevice, "parent_inode": o.ParentInode} {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		fields[key] = raw
	}
	return json.Marshal(fields)
}
