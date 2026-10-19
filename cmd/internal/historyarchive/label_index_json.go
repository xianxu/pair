package historyarchive

import (
	"encoding/json"
	"errors"
)

var errIndexBound = errors.New("label index exceeds 64 MiB")

// DecodeLabelIndex admits a complete bounded snapshot; it grants discovery only.
func DecodeLabelIndex(raw []byte) (LabelIndex, error) {
	var i LabelIndex
	if len(raw) > MaxIndexBytes {
		return i, errIndexBound
	}
	if err := json.Unmarshal(raw, &i); err != nil {
		return i, err
	}
	return i, ValidateLabelIndex(i)
}

func (v LabelIndex) MarshalJSON() ([]byte, error) {
	type wire LabelIndex
	raw, err := json.Marshal(wire(v))
	if err != nil {
		return nil, err
	}
	return marshalExtensibleObjectLimit(raw, v.Extensions, []string{"version", "records", "complete_destinations"}, MaxIndexBytes)
}
func (v *LabelIndex) UnmarshalJSON(raw []byte) error {
	fields, ext, err := decodeExtensibleObjectSchema(raw, []string{"version", "records"}, []string{"complete_destinations"}, MaxIndexBytes)
	if err != nil {
		return err
	}
	clean, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type wire LabelIndex
	var w wire
	if err := json.Unmarshal(clean, &w); err != nil {
		return err
	}
	*v = LabelIndex(w)
	v.Extensions = ext
	return nil
}

func (v DestinationObservation) MarshalJSON() ([]byte, error) {
	type wire DestinationObservation
	raw, err := json.Marshal(wire(v))
	if err != nil {
		return nil, err
	}
	return marshalExtensibleObject(raw, v.Extensions, []string{"backend", "locator", "folder_id", "observed_at"})
}

func (v *DestinationObservation) UnmarshalJSON(raw []byte) error {
	fields, ext, err := decodeExtensibleObject(raw, []string{"backend", "locator", "folder_id", "observed_at"})
	if err != nil {
		return err
	}
	clean, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type wire DestinationObservation
	var w wire
	if err := json.Unmarshal(clean, &w); err != nil {
		return err
	}
	*v = DestinationObservation(w)
	v.Extensions = ext
	return nil
}

func (v LabelIndexRecord) MarshalJSON() ([]byte, error) {
	type wire LabelIndexRecord
	raw, err := json.Marshal(wire(v))
	if err != nil {
		return nil, err
	}
	return marshalExtensibleObjectLimit(raw, v.Extensions, []string{"label", "locations"}, MaxMetadataBytes)
}
func (v *LabelIndexRecord) UnmarshalJSON(raw []byte) error {
	fields, ext, err := decodeExtensibleObjectLimit(raw, []string{"label", "locations"}, MaxMetadataBytes)
	if err != nil {
		return err
	}
	clean, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type wire LabelIndexRecord
	var w wire
	if err := json.Unmarshal(clean, &w); err != nil {
		return err
	}
	*v = LabelIndexRecord(w)
	v.Extensions = ext
	return nil
}

func (v BundleLocation) MarshalJSON() ([]byte, error) {
	type wire BundleLocation
	raw, err := json.Marshal(wire(v))
	if err != nil {
		return nil, err
	}
	return marshalExtensibleObjectLimit(raw, v.Extensions, []string{"backend", "locator", "folder_id", "bundle_key", "label_key", "available", "evidence", "first_observed", "last_observed", "proof"}, MaxMetadataBytes)
}
func (v *BundleLocation) UnmarshalJSON(raw []byte) error {
	fields, ext, err := decodeExtensibleObjectLimit(raw, []string{"backend", "locator", "folder_id", "bundle_key", "label_key", "available", "evidence", "first_observed", "last_observed", "proof"}, MaxMetadataBytes)
	if err != nil {
		return err
	}
	clean, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type wire BundleLocation
	var w wire
	if err := json.Unmarshal(clean, &w); err != nil {
		return err
	}
	*v = BundleLocation(w)
	v.Extensions = ext
	return nil
}

func (v LocationProof) MarshalJSON() ([]byte, error) {
	type wire LocationProof
	raw, err := json.Marshal(wire(v))
	if err != nil {
		return nil, err
	}
	return marshalExtensibleObjectLimit(raw, v.Extensions, []string{"method", "sha256", "verified_at"}, MaxMetadataBytes)
}
func (v *LocationProof) UnmarshalJSON(raw []byte) error {
	fields, ext, err := decodeExtensibleObjectLimit(raw, []string{"method", "sha256", "verified_at"}, MaxMetadataBytes)
	if err != nil {
		return err
	}
	clean, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type wire LocationProof
	var w wire
	if err := json.Unmarshal(clean, &w); err != nil {
		return err
	}
	*v = LocationProof(w)
	v.Extensions = ext
	return nil
}
