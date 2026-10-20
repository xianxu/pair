package couchcore

import (
	"encoding/json"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestRecoveryChoiceJSONNativeOnlyOmitsScreenScores(t *testing.T) {
	for _, source := range []sessioninventory.RecoverySourceKind{"", sessioninventory.RecoverySourceCapture, sessioninventory.RecoverySourceNative} {
		t.Run(string(source), func(t *testing.T) {
			choice := RecoveryChoice{SourceKind: source, Key: "exact-key", Label: "unconfirmed", Agent: "codex", NativeID: "native-id", CWD: "/physical/repo"}
			for _, nonzero := range []bool{false, true} {
				if nonzero {
					choice.TailMatch, choice.Matched, choice.Total = 100, 8, 8
				}
				b, err := json.Marshal(choice)
				if err != nil {
					t.Fatal(err)
				}
				var got map[string]json.RawMessage
				if err = json.Unmarshal(b, &got); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"tail_match", "matched", "total"} {
					_, present := got[name]
					if present != (source != sessioninventory.RecoverySourceNative) {
						t.Fatalf("source %q score field %q present=%v: %s", source, name, present, b)
					}
				}
				for _, name := range []string{"key", "native_id", "cwd", "last_activity"} {
					if _, ok := got[name]; !ok {
						t.Fatalf("missing evidence %s: %s", name, b)
					}
				}
				var decoded RecoveryChoice
				if err = json.Unmarshal(b, &decoded); err != nil {
					t.Fatal(err)
				}
				if decoded.SourceKind != source || decoded.CWD != choice.CWD || decoded.Key != choice.Key {
					t.Fatalf("lost source/identity: %+v", decoded)
				}
			}
		})
	}
}
