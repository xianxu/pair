package operationalrun

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRecordManualIntentDigestCompatibility(t *testing.T) {
	old := sample("manual-id", time.Now().UTC())
	old.Source = "manual"
	legacy, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(legacy, []byte("manual_intent_digest")) {
		t.Fatal("legacy empty digest should be omitted")
	}
	var decoded Record
	if err = json.Unmarshal(legacy, &decoded); err != nil {
		t.Fatal(err)
	}
	old.ManualIntentDigest = strings.Repeat("a", 64)
	old.Unknown = map[string]json.RawMessage{"summary_extra": json.RawMessage(`{ "x": 2 }`)}
	raw, err := encodeRecord(old)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ManualIntentDigest != old.ManualIntentDigest || string(decoded.Unknown["summary_extra"]) != `{ "x": 2 }` {
		t.Fatalf("lost immutable intent/additive facts: %+v", decoded)
	}
	for _, field := range []string{`"manual_intent_digest":12,`, `"manual_intent_digest":null,`, `"manual_intent_digest":"a","manual_intent_digest":"b",`} {
		changed := append([]byte("{"+field), legacy[1:]...)
		if err = json.Unmarshal(changed, &decoded); err == nil {
			t.Fatalf("accepted malformed digest: %s", field)
		}
	}
}
