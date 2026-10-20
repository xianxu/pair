package threadrecord

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func nativeRecoveryWire(t *testing.T) []byte {
	t.Helper()
	c := validRecoveryContext()
	raw, _ := json.Marshal(c)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	delete(fields, "capture")
	delete(fields, "generation")
	fields["version"] = json.RawMessage(`2`)
	fields["source"] = json.RawMessage(`{"kind":"native-only","cutoff":"1970-01-01T00:01:40Z","revision":7,"future_source":{"x":1}}`)
	fields["expires_at"], _ = json.Marshal(c.CreatedAt.Add(5 * time.Minute))
	fields["future_context"] = json.RawMessage(`{"x":2}`)
	var candidates []map[string]json.RawMessage
	_ = json.Unmarshal(fields["candidates"], &candidates)
	for _, candidate := range candidates {
		delete(candidate, "score")
		delete(candidate, "matched")
		delete(candidate, "total")
		candidate["cwd"] = json.RawMessage(`"/workspace/pair"`)
		candidate["future_candidate"] = json.RawMessage(`{"x":3}`)
	}
	fields["candidates"], _ = json.Marshal(candidates)
	raw, _ = json.Marshal(fields)
	return raw
}

func TestNativeRecoverySourcePreservesNestedEvidence(t *testing.T) {
	raw := nativeRecoveryWire(t)
	var c RecoveryContext
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if err := ValidateRecoveryContext(c, c.Address); err != nil {
		t.Fatal(err)
	}
	r := validRecord()
	r.RecoveryContext = &c
	r.RecoveryIntent = &RecoveryIntent{Context: c, SelectedKey: c.SelectedKey, NativeID: c.SelectedNativeID}
	encoded, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePersisted(encoded, r.Address, testValidators)
	if err != nil {
		t.Fatal(err)
	}
	clone := CloneRecoveryIntent(got.RecoveryIntent)
	rewritten, err := json.Marshal(clone)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"future_context", "future_source", "future_candidate"} {
		if !strings.Contains(string(rewritten), field) {
			t.Fatalf("lost %s: %s", field, rewritten)
		}
	}
	if strings.Contains(string(rewritten), `"capture"`) || strings.Contains(string(rewritten), `"score"`) {
		t.Fatalf("native evidence manufactured capture scoring: %s", rewritten)
	}
}

func TestNativeRecoverySourceRejectsMixedWire(t *testing.T) {
	raw := nativeRecoveryWire(t)
	for _, addition := range []string{`"capture":{},`, `"generation":"",`, `"source":null,`, `"version":2,`} {
		var c RecoveryContext
		bad := strings.Replace(string(raw), "{", "{"+addition, 1)
		if err := json.Unmarshal([]byte(bad), &c); err == nil && ValidateRecoveryContext(c, c.Address) == nil {
			t.Fatalf("accepted %s", addition)
		}
	}
	for _, addition := range []string{`"score":0,`, `"matched":0,`, `"total":0,`, `"cwd":null,`} {
		var c RecoveryContext
		bad := strings.Replace(string(raw), `"candidates":[{`, `"candidates":[{`+addition, 1)
		if err := json.Unmarshal([]byte(bad), &c); err == nil && ValidateRecoveryContext(c, c.Address) == nil {
			t.Fatalf("accepted candidate %s", addition)
		}
	}
}

func TestNativeRecoveryStructuralLifetimeAndClone(t *testing.T) {
	var c RecoveryContext
	if err := json.Unmarshal(nativeRecoveryWire(t), &c); err != nil {
		t.Fatal(err)
	}
	if err := ValidateNativeRecoverySourceAt(c.Source, c.Source.Cutoff, c.Source.Revision); err != nil {
		t.Fatal(err)
	}
	for _, now := range []time.Time{c.Source.Cutoff.Add(-time.Nanosecond), c.ExpiresAt, c.ExpiresAt.Add(time.Hour)} {
		if ValidateNativeRecoverySourceAt(c.Source, now, c.Source.Revision) == nil {
			t.Fatal("accepted outside fixed lifetime")
		}
	}
	if ValidateNativeRecoverySourceAt(c.Source, c.Source.Cutoff, c.Source.Revision+1) == nil {
		t.Fatal("accepted revised source")
	}
	c.SelectedKey = ""
	c.SelectedNativeID = ""
	if err := ValidateRecoveryContext(c, c.Address); err != nil {
		t.Fatal(err)
	}
	if ValidateRecoveryIntent(RecoveryIntent{Context: c, SelectedKey: "fresh"}, c.Address) == nil {
		t.Fatal("native source authorized fresh fallback")
	}
	clone := CloneRecoveryContext(&c)
	clone.Unknown["future_context"][0] = '0'
	clone.Source.Unknown["future_source"][0] = '0'
	clone.Candidates[0].Unknown["future_candidate"][0] = '0'
	for _, value := range []json.RawMessage{c.Unknown["future_context"], c.Source.Unknown["future_source"], c.Candidates[0].Unknown["future_candidate"]} {
		if !json.Valid(value) {
			t.Fatal("clone mutated retained evidence")
		}
	}
	for name, change := range map[string]func(*RecoveryContext){
		"revision": func(c *RecoveryContext) { c.Source.Revision = 0 },
		"cutoff":   func(c *RecoveryContext) { c.Source.Cutoff = time.Time{} },
		"created":  func(c *RecoveryContext) { c.CreatedAt = c.CreatedAt.Add(time.Nanosecond) },
		"expiry":   func(c *RecoveryContext) { c.ExpiresAt = c.ExpiresAt.Add(time.Nanosecond) },
		"empty":    func(c *RecoveryContext) { c.Candidates = nil },
		"cwd":      func(c *RecoveryContext) { c.Candidates[0].CWD = "relative" },
		"tail":     func(c *RecoveryContext) { c.Candidates[0].Total = 1 },
		"capture":  func(c *RecoveryContext) { c.Capture = validRecoveryContext().Capture },
	} {
		t.Run(name, func(t *testing.T) {
			copy := CloneRecoveryContext(&c)
			change(copy)
			if ValidateRecoveryContext(*copy, c.Address) == nil {
				t.Fatal("accepted invalid evidence")
			}
		})
	}
}

func TestNativeRecoveryRejectsNullAndMalformedKnownMembers(t *testing.T) {
	raw := string(nativeRecoveryWire(t))
	for _, mutation := range []struct{ old, new string }{
		{`"future_source":{"x":1}`, `"launch_hint":null`},
		{`"future_candidate":{"x":3}`, `"excerpt":null`},
		{`"future_context":{"x":2}`, `"branch":null`},
		{`"revision":7`, `"revision":"7"`},
		{`"cutoff":"1970-01-01T00:01:40Z"`, `"cutoff":null`},
	} {
		var c RecoveryContext
		invalid := strings.Replace(raw, mutation.old, mutation.new, 1)
		if invalid == raw {
			t.Fatalf("bad fixture mutation %s", mutation.old)
		}
		if err := json.Unmarshal([]byte(invalid), &c); err == nil && ValidateRecoveryContext(c, c.Address) == nil {
			t.Fatalf("accepted %s", mutation.new)
		}
	}
}

func TestNativeRecoverySourceRoundTripRetainsExactEmptyUnknown(t *testing.T) {
	var source NativeRecoverySource
	raw := []byte(`{"kind":"native-only","cutoff":"2026-10-19T12:00:00Z","revision":7}`)
	if err := json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	if source.Unknown != nil {
		t.Fatal("decoder manufactured unknown members")
	}
}

func TestNativeRecoveryRootWorkingDirectoryAndTimeBounds(t *testing.T) {
	var c RecoveryContext
	if err := json.Unmarshal(nativeRecoveryWire(t), &c); err != nil {
		t.Fatal(err)
	}
	c.Candidates[0].CWD = "/"
	if err := ValidateRecoveryContext(c, c.Address); err != nil {
		t.Fatalf("physical root cwd rejected: %v", err)
	}
	c.Source.Cutoff = time.Date(9999, 12, 31, 23, 59, 0, 0, time.UTC)
	c.CreatedAt = c.Source.Cutoff
	c.ExpiresAt = c.Source.Cutoff.Add(5 * time.Minute)
	if ValidateRecoveryContext(c, c.Address) == nil {
		t.Fatal("accepted expiry outside encodable time range")
	}
}

func TestSchemaThreeRejectsNativeRecoverySemanticsByPresence(t *testing.T) {
	var c RecoveryContext
	if err := json.Unmarshal(nativeRecoveryWire(t), &c); err != nil {
		t.Fatal(err)
	}
	r := validRecord()
	r.SchemaVersion = 3
	r.RecoveryContext = &c
	raw, _ := json.Marshal(r)
	if _, err := DecodePersisted(raw, r.Address, testValidators); err == nil {
		t.Fatal("schema3 accepted native v2 context")
	}
	c = validRecoveryContext()
	r.RecoveryContext = &c
	raw, _ = json.Marshal(r)
	for _, addition := range []string{`"cwd":null,`, `"cwd":"/workspace/pair",`} {
		wire := strings.Replace(string(raw), `"candidates":[{`, `"candidates":[{`+addition, 1)
		if _, err := DecodePersisted([]byte(wire), r.Address, testValidators); err == nil {
			t.Fatalf("schema3 accepted %s", addition)
		}
	}
	r.RecoveryContext = nil
	r.Incarnations[0].Start.RecoveryIntent = &RecoveryIntent{Context: c, SelectedKey: c.SelectedKey, NativeID: c.SelectedNativeID}
	raw, _ = json.Marshal(r)
	wire := strings.Replace(string(raw), `"candidates":[{`, `"candidates":[{"cwd":null,`, 1)
	if _, err := DecodePersisted([]byte(wire), r.Address, testValidators); err == nil {
		t.Fatal("schema3 nested intent accepted native cwd")
	}
}

func TestRecoveryContextV1KnownBytesStayFrozen(t *testing.T) {
	const want = `{"version":1,"address":{"repo_scope":"816fc349d3faebf8","tag":"couch-0102030405060708"},"capture":{"last_activity":"0001-01-01T00:00:00Z","raw":"/capture/raw","events":"/capture/events","raw_identity":{"size":10,"modified_at":"1970-01-01T00:01:40Z","device":1,"inode":2},"events_identity":{"size":10,"modified_at":"1970-01-01T00:01:40Z","device":1,"inode":2},"captured_at":"1970-01-01T00:01:40Z","launch_ordinal":1,"nonce":"nonce"},"generation":"generation","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","created_at":"1970-01-01T00:01:40Z","expires_at":"1970-01-01T01:01:40Z","branch":"branch","issue":"pair#528","candidates":[{"key":"opaque","agent":"codex","native_id":"native","artifact_root":"codex-sessions","artifact_relative_path":"rollout.jsonl","stable_file_id":"file","generation_token":"birth","mutation_token":"mtime","size":10,"mod_time":"1970-01-01T00:01:40Z","score":80,"matched":8,"total":10,"last_activity":"1970-01-01T00:01:40Z","excerpt":"previous work","eligible":true}],"selected_key":"opaque","selected_native_id":"native"}`
	raw, err := json.Marshal(validRecoveryContext())
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != want {
		t.Fatalf("v1 known bytes changed:\n%s", raw)
	}
}

func TestNativeRecoveryActivityCannotExceedSourceCutoff(t *testing.T) {
	var c RecoveryContext
	if err := json.Unmarshal(nativeRecoveryWire(t), &c); err != nil {
		t.Fatal(err)
	}
	c.Candidates[0].ModTime = c.Source.Cutoff.Add(time.Second)
	if err := ValidateRecoveryContext(c, c.Address); err != nil {
		t.Fatalf("observed metadata after cutoff should remain valid: %v", err)
	}
	c.Candidates[0].LastActivity = c.Source.Cutoff.Add(time.Nanosecond)
	if ValidateRecoveryContext(c, c.Address) == nil {
		t.Fatal("accepted native conversation activity after source cutoff")
	}
}

func TestNativeRecoveryIntentRequiresExplicitNativeMembership(t *testing.T) {
	var c RecoveryContext
	if err := json.Unmarshal(nativeRecoveryWire(t), &c); err != nil {
		t.Fatal(err)
	}
	c.SelectedKey = ""
	c.SelectedNativeID = ""
	if err := ValidateRecoveryContext(c, c.Address); err != nil {
		t.Fatalf("unselected preview should remain readable: %v", err)
	}
	for _, selection := range []struct{ key, native string }{
		{"", ""}, {"fresh", ""}, {c.Candidates[0].Key, ""}, {"missing", c.Candidates[0].NativeID}, {c.Candidates[0].Key, "other"},
	} {
		intent := RecoveryIntent{Context: c, SelectedKey: selection.key, NativeID: selection.native}
		if ValidateRecoveryIntent(intent, c.Address) == nil {
			t.Fatalf("accepted native intent without exact selected membership: %+v", selection)
		}
	}
	intent := RecoveryIntent{Context: c, SelectedKey: c.Candidates[0].Key, NativeID: c.Candidates[0].NativeID}
	if err := ValidateRecoveryIntent(intent, c.Address); err != nil {
		t.Fatalf("exact selected native membership rejected: %v", err)
	}
}
