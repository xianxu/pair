package threadrecord

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func validRecoveryContext() RecoveryContext {
	at := time.Unix(100, 0).UTC()
	identity := artifactpath.CaptureFileIdentity{Size: 10, ModifiedAt: at, Device: 1, Inode: 2}
	return RecoveryContext{Version: 1, Address: validRecord().Address, Capture: RecoveryCapture{Raw: "/capture/raw", Events: "/capture/events", RawIdentity: identity, EventsIdentity: identity, CapturedAt: at, LaunchOrdinal: 1, Nonce: "nonce"}, Generation: "generation", Digest: strings.Repeat("a", 64), CreatedAt: at, ExpiresAt: at.Add(time.Hour), Branch: "branch", Issue: "pair#528", Candidates: []RecoveryCandidate{{Key: "opaque", Agent: "codex", NativeID: "native", ArtifactRoot: "codex-sessions", ArtifactRelativePath: "rollout.jsonl", StableFileID: "file", GenerationToken: "birth", MutationToken: "mtime", Size: 10, ModTime: at, Score: 80, Matched: 8, Total: 10, LastActivity: at, Excerpt: "previous work", Eligible: true}}, SelectedKey: "opaque", SelectedNativeID: "native"}
}
func TestRecoveryContextAndIntentRoundTrip(t *testing.T) {
	r := validRecord()
	c := validRecoveryContext()
	r.RecoveryContext = &c
	r.Incarnations[0].Start.RecoveryIntent = &RecoveryIntent{Context: c, SelectedKey: "opaque", NativeID: "native"}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePersisted(raw, r.Address, testValidators)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.RecoveryContext, r.RecoveryContext) || !reflect.DeepEqual(got.Incarnations[0].Start.RecoveryIntent, r.Incarnations[0].Start.RecoveryIntent) {
		t.Fatal("recovery fields lost")
	}
	clone := CloneRecoveryIntent(r.Incarnations[0].Start.RecoveryIntent)
	clone.Context.Candidates[0].Excerpt = "changed"
	if c.Candidates[0].Excerpt == "changed" {
		t.Fatal("candidate slice aliased")
	}
}
func TestRecoveryContextRejectsInvalidEvidence(t *testing.T) {
	for name, mutate := range map[string]func(*RecoveryContext){
		"version":  func(c *RecoveryContext) { c.Version = 2 },
		"address":  func(c *RecoveryContext) { c.Address.Tag = "foreign" },
		"path":     func(c *RecoveryContext) { c.Capture.Raw = "relative" },
		"identity": func(c *RecoveryContext) { c.Capture.EventsIdentity.Inode = 0 },
		"expiry":   func(c *RecoveryContext) { c.ExpiresAt = c.Capture.CapturedAt },
		"producer": func(c *RecoveryContext) { c.Capture.Nonce = "" },
		"digest":   func(c *RecoveryContext) { c.Digest = "short" },
		"too many": func(c *RecoveryContext) {
			c.Candidates = append(c.Candidates, c.Candidates[0], c.Candidates[0], c.Candidates[0], c.Candidates[0], c.Candidates[0])
		},
		"duplicate":         func(c *RecoveryContext) { c.Candidates = append(c.Candidates, c.Candidates[0]) },
		"score":             func(c *RecoveryContext) { c.Candidates[0].Score = 101 },
		"counts":            func(c *RecoveryContext) { c.Candidates[0].Matched = 11 },
		"excerpt":           func(c *RecoveryContext) { c.Candidates[0].Excerpt = strings.Repeat("x", 2049) },
		"control":           func(c *RecoveryContext) { c.Generation = "bad\x00" },
		"selected key":      func(c *RecoveryContext) { c.SelectedKey = "missing" },
		"selected identity": func(c *RecoveryContext) { c.SelectedNativeID = "other" },
		"ineligible":        func(c *RecoveryContext) { c.Candidates[0].Eligible = false },
	} {
		t.Run(name, func(t *testing.T) {
			c := validRecoveryContext()
			mutate(&c)
			if ValidateRecoveryContext(c, validRecord().Address) == nil {
				t.Fatal("accepted invalid context")
			}
		})
	}
}
func TestRecoveryIntentFreshSelectionAndUnknownSource(t *testing.T) {
	c := validRecoveryContext()
	c.Capture.LaunchOrdinal = 0
	c.Capture.Nonce = ""
	intent := RecoveryIntent{Context: c, SelectedKey: "fresh-key"}
	if err := ValidateRecoveryIntent(intent, c.Address); err != nil {
		t.Fatal(err)
	}
	intent.SelectedKey = ""
	intent.NativeID = "native"
	if ValidateRecoveryIntent(intent, c.Address) == nil {
		t.Fatal("accepted native without key")
	}
}

func TestRecoverySchemaV2MigrationAndStrictBoundaries(t *testing.T) {
	record := validRecord()
	record.SchemaVersion = 2
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	migrated, err := DecodePersisted(raw, record.Address, testValidators)
	if err != nil || migrated.SchemaVersion != SchemaVersion || migrated.RecoveryContext != nil || migrated.Incarnations[0].Start.RecoveryIntent != nil {
		t.Fatalf("migration: %+v %v", migrated, err)
	}
	for _, version := range []string{"1", "2", "3"} {
		wire := strings.Replace(string(raw), `"schema_version":2`, `"schema_version":`+version, 1)
		for _, addition := range []string{`"unknown":null,`, `"recovery_context":null,`} {
			if version == "3" && strings.Contains(addition, "recovery_context") {
				continue
			}
			invalid := strings.Replace(wire, "{", "{"+addition, 1)
			if _, err := DecodePersisted([]byte(invalid), record.Address, testValidators); err == nil {
				t.Fatalf("schema %s accepted %s", version, addition)
			}
		}
		if version != "3" {
			invalid := strings.Replace(wire, `"start":{`, `"start":{"recovery_intent":null,`, 1)
			if _, err := DecodePersisted([]byte(invalid), record.Address, testValidators); err == nil {
				t.Fatalf("schema %s accepted nested new semantics", version)
			}
		}
	}
}

func TestRecoverySnapshotMayBeUnselectedAndExpired(t *testing.T) {
	c := validRecoveryContext()
	c.SelectedKey = ""
	c.SelectedNativeID = ""
	c.Candidates[0].GenerationToken = ""
	if err := ValidateRecoveryContext(c, c.Address); err != nil {
		t.Fatal(err)
	}
	if !c.ExpiresAt.Before(time.Now()) {
		t.Fatal("fixture must represent expired retained evidence")
	}
	if err := ValidateRecoveryIntent(RecoveryIntent{Context: c}, c.Address); err == nil {
		t.Fatal("intent accepted no explicit key")
	}
	if err := ValidateRecoveryIntent(RecoveryIntent{Context: c, SelectedKey: "fresh-key"}, c.Address); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverySchemaRejectsUnknownNestedFields(t *testing.T) {
	r := validRecord()
	c := validRecoveryContext()
	r.RecoveryContext = &c
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, old := range []string{`"recovery_context":{`, `"capture":{`, `"candidates":[{`} {
		invalid := strings.Replace(string(raw), old, old+`"unknown":true,`, 1)
		if _, err := DecodePersisted([]byte(invalid), r.Address, testValidators); err == nil {
			t.Fatalf("accepted unknown fields in %s", old)
		}
	}
}

func TestRecoveryCaptureActivityAndTokenBounds(t *testing.T) {
	for _, change := range []func(*RecoveryContext){
		func(c *RecoveryContext) { c.Capture.LastActivity = c.Capture.CapturedAt.Add(time.Second) },
		func(c *RecoveryContext) { c.Capture.Token = "../other" },
		func(c *RecoveryContext) { c.Capture.Token = strings.Repeat("x", 129) },
	} {
		c := validRecoveryContext()
		change(&c)
		if ValidateRecoveryContext(c, c.Address) == nil {
			t.Fatal("accepted invalid original screen coordinates")
		}
	}
	c := validRecoveryContext()
	c.Capture.Token = "20261013T120000"
	c.Capture.LastActivity = c.Capture.CapturedAt.Add(-time.Second)
	if err := ValidateRecoveryContext(c, c.Address); err != nil {
		t.Fatal(err)
	}
}

func TestTaggedRecoveryIntentWithoutCapture(t *testing.T) {
	for _, mode := range []string{"proven", "fresh"} {
		t.Run(mode, func(t *testing.T) {
			r := validRecord()
			wire := map[string]any{"mode": mode, "address": r.Address, "source_revision": r.Revision, "created_at": r.CreatedAt, "agent": "muse", "selected_key": "explicit"}
			if mode == "proven" {
				wire["native_id"] = "native"
			}
			raw, _ := json.Marshal(wire)
			var intent RecoveryIntent
			if err := json.Unmarshal(raw, &intent); err != nil {
				t.Fatal(err)
			}
			if err := ValidateRecoveryIntent(intent, r.Address); err != nil {
				t.Fatalf("explicit %s requires inference capture: %v", mode, err)
			}
		})
	}
}

func TestSchemaThreeMigratesRecoveryReadOnly(t *testing.T) {
	r := validRecord()
	r.SchemaVersion = 3
	c := validRecoveryContext()
	r.RecoveryContext = &c
	r.Incarnations[0].Start.RecoveryIntent = &RecoveryIntent{Context: c, SelectedKey: "opaque", NativeID: "native"}
	raw, _ := json.Marshal(r)
	before := append([]byte(nil), raw...)
	got, err := DecodePersisted(raw, r.Address, testValidators)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != 4 {
		t.Fatalf("schema=%d, want 4", got.SchemaVersion)
	}
	if !reflect.DeepEqual(got.RecoveryContext, r.RecoveryContext) || !reflect.DeepEqual(got.Incarnations[0].Start.RecoveryIntent, r.Incarnations[0].Start.RecoveryIntent) {
		t.Fatal("legacy inference lost")
	}
	if string(raw) != string(before) {
		t.Fatal("decode mutated input")
	}
}

func TestTaggedRecoveryValidationAndCloning(t *testing.T) {
	base := RecoveryIntent{Mode: RecoveryProven, Address: validRecord().Address, SourceRevision: 1, CreatedAt: time.Unix(100, 0).UTC(), Agent: "muse", SelectedKey: "key", NativeID: "native"}
	c := validRecoveryContext().Capture
	base.Orientation = &RecoveryOrientation{Branch: "branch", Issue: "pair#556", Capture: &c}
	for name, mutate := range map[string]func(*RecoveryIntent){
		"fresh relaunch":  func(i *RecoveryIntent) { i.Relaunch = true; i.Mode = RecoveryFresh; i.NativeID = "" },
		"address":         func(i *RecoveryIntent) { i.Address.Tag = "other" },
		"revision":        func(i *RecoveryIntent) { i.SourceRevision = 0 },
		"time":            func(i *RecoveryIntent) { i.CreatedAt = time.Time{} },
		"agent":           func(i *RecoveryIntent) { i.Agent = "bad/agent" },
		"selection":       func(i *RecoveryIntent) { i.SelectedKey = "" },
		"proven identity": func(i *RecoveryIntent) { i.NativeID = "" },
		"fresh identity":  func(i *RecoveryIntent) { i.Mode = RecoveryFresh },
		"mode":            func(i *RecoveryIntent) { i.Mode = "other" },
		"capture":         func(i *RecoveryIntent) { i.Orientation.Capture.Raw = "relative" },
		"orientation":     func(i *RecoveryIntent) { i.Orientation.Unavailable = "bad\x00" },
	} {
		t.Run(name, func(t *testing.T) {
			i := CloneRecoveryIntent(&base)
			mutate(i)
			if ValidateRecoveryIntent(*i, base.Address) == nil {
				t.Fatal("accepted invalid explicit intent")
			}
		})
	}
	cloned := CloneRecoveryIntent(&base)
	cloned.Orientation.Capture.Raw = "changed"
	cloned.Orientation.Branch = "changed"
	if base.Orientation.Capture.Raw == "changed" || base.Orientation.Branch == "changed" {
		t.Fatal("clone aliases orientation")
	}
	legacy := RecoveryIntent{SelectedKey: "key", NativeID: "native"}
	if legacy.Kind() != RecoveryInferred || ValidateRecoveryIntent(legacy, base.Address) == nil {
		t.Fatal("legacy inference accepted missing capture")
	}
}

func TestSchemaThreeRejectsTaggedFieldsByPresence(t *testing.T) {
	r := validRecord()
	r.SchemaVersion = 3
	c := validRecoveryContext()
	r.Incarnations[0].Start.RecoveryIntent = &RecoveryIntent{Context: c, SelectedKey: "opaque", NativeID: "native"}
	raw, _ := json.Marshal(r)
	for _, field := range []string{"mode", "address", "source_revision", "created_at", "agent", "orientation"} {
		wire := strings.Replace(string(raw), `"recovery_intent":{`, `"recovery_intent":{"`+field+`":null,`, 1)
		if _, err := DecodePersisted([]byte(wire), r.Address, testValidators); err == nil {
			t.Fatalf("schema3 accepted %s", field)
		}
	}
	wire := strings.Replace(string(raw), "{", `{"recovery_intent":null,`, 1)
	if _, err := DecodePersisted([]byte(wire), r.Address, testValidators); err == nil {
		t.Fatal("schema3 accepted retained intent")
	}
}

func TestRetainedTaggedIntentRoundTripAndValidation(t *testing.T) {
	r := validRecord()
	r.RecoveryIntent = &RecoveryIntent{Mode: RecoveryFresh, Address: r.Address, SourceRevision: r.Revision, CreatedAt: r.CreatedAt, Agent: "agy", SelectedKey: "explicit"}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePersisted(raw, r.Address, testValidators)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.RecoveryIntent, r.RecoveryIntent) {
		t.Fatal("retained selection lost")
	}
	r.RecoveryIntent.Address.Tag = "foreign"
	if Validate(r, testValidators) == nil {
		t.Fatal("retained intent escaped validation")
	}
}
