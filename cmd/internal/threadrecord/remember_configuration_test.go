package threadrecord

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestStartClaimRememberConfigurationCodec(t *testing.T) {
	record := validRecord()
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte(`"nonce":"nonce"`), []byte(`"remember_configuration":true,"nonce":"nonce"`), 1)
	decoded, err := DecodePersisted(raw, record.Address, testValidators)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodePersisted(decoded)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Incarnations []struct {
			Start map[string]json.RawMessage `json:"start"`
		} `json:"incarnations"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatal(err)
	}
	if string(envelope.Incarnations[0].Start["remember_configuration"]) != "true" {
		t.Fatalf("explicit acceptance lost: %s", encoded)
	}
	for _, invalid := range []string{`"true"`, `1`, `{}`} {
		bad := bytes.Replace(raw, []byte(`"remember_configuration":true`), []byte(`"remember_configuration":`+invalid), 1)
		if _, err := DecodePersisted(bad, record.Address, testValidators); err == nil {
			t.Fatalf("accepted untyped intent %s", invalid)
		}
	}
}
