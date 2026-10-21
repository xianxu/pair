package historyarchive

import "encoding/json"

var uploadReceiptJSONFields = []string{"version", "destination", "folder_id", "bundle_id", "bundle", "label", "verified_at"}
var uploadObjectJSONFields = []string{"name", "size", "sha256", "md5", "remote_id", "verification"}

func (v UploadReceipt) MarshalJSON() ([]byte, error) {
	type wire UploadReceipt
	raw, err := json.Marshal(wire(v))
	if err != nil {
		return nil, err
	}
	return marshalExtensibleObjectLimit(raw, v.Extensions, uploadReceiptJSONFields, MaxUploadReceiptBytes)
}

func (v *UploadReceipt) UnmarshalJSON(raw []byte) error {
	fields, extensions, err := decodeExtensibleObjectSchema(raw, uploadReceiptJSONFields, nil, MaxUploadReceiptBytes)
	if err != nil {
		return err
	}
	clean, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type wire UploadReceipt
	var decoded wire
	if err := json.Unmarshal(clean, &decoded); err != nil {
		return err
	}
	decoded.Extensions = extensions
	*v = UploadReceipt(decoded)
	return nil
}

func (v UploadObject) MarshalJSON() ([]byte, error) {
	type wire UploadObject
	raw, err := json.Marshal(wire(v))
	if err != nil {
		return nil, err
	}
	return marshalExtensibleObjectLimit(raw, v.Extensions, uploadObjectJSONFields, MaxUploadReceiptBytes)
}

func (v *UploadObject) UnmarshalJSON(raw []byte) error {
	fields, extensions, err := decodeExtensibleObjectSchema(raw, uploadObjectJSONFields, nil, MaxUploadReceiptBytes)
	if err != nil {
		return err
	}
	clean, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type wire UploadObject
	var decoded wire
	if err := json.Unmarshal(clean, &decoded); err != nil {
		return err
	}
	decoded.Extensions = extensions
	*v = UploadObject(decoded)
	return nil
}
