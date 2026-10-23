package couchcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestRepoPreferencesCodecPreservesRawAtEveryLevel(t *testing.T) {
	source := repoLegacy(t, "legacy", "/repo/a", "one")
	p, err := FoldPathPreferences(nil, "/repo/.git", []LegacyRepoPreference{source})
	if err != nil {
		t.Fatal(err)
	}
	p, err = RememberRepoArguments(&p, "/repo/.git", LaunchProfile{Agent: "codex", Argv: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	rawUnknown := json.RawMessage(`{ "future" : [ 1,  2 ] }`)
	p.Unknown = map[string]json.RawMessage{"repo_extra": rawUnknown}
	a := p.Agents["codex"]
	a.Unknown = map[string]json.RawMessage{"agent_extra": rawUnknown}
	p.Agents["codex"] = a
	s := p.LegacySources["legacy"]
	s.Unknown = map[string]json.RawMessage{"source_extra": rawUnknown}
	p.LegacySources["legacy"] = s
	doc := RepoPreferences{SchemaVersion: 1, Repositories: map[string]RepoLaunchPreference{p.RepoIdentity: p}, Unknown: map[string]json.RawMessage{"envelope_extra": rawUnknown}}
	raw, err := EncodeRepoPreferences(doc)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(raw, rawUnknown) != 4 {
		t.Fatalf("raw values compacted: %s", raw)
	}
	decoded, err := DecodeRepoPreferences(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.Repositories[p.RepoIdentity].LegacySources["legacy"].Raw, source.Raw) {
		t.Fatal("lost original legacy source image")
	}
	again, err := EncodeRepoPreferences(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, again) {
		t.Fatalf("unstable round trip\n%s\n%s", raw, again)
	}
}

func TestRepoPreferencesCodecRejectsAmbiguousKnownFields(t *testing.T) {
	for _, raw := range []string{
		`null`, `[]`, `{"schema_version":1,"schema_version":1,"repositories":{}}`,
		`{"schema_version":1,"repositories":null}`, `{"schema_version":"1","repositories":{}}`,
		`{"schema_version":2,"repositories":{}}`,
		`{"schema_version":1,"repositories":{"repo":{"repo_identity":"other","revision":1,"agents":{},"legacy_sources":{}}}}`,
		`{"schema_version":1,"repositories":{"repo":{"repo_identity":"repo","revision":1,"agents":{"codex":{"argv":null,"confirmed":true,"conflict":false}},"legacy_sources":{}}}}`,
		`{"schema_version":1,"repositories":{"repo":{"repo_identity":"repo","revision":1,"agents":{"codex":{"argv":[],"confirmed":null,"conflict":false}},"legacy_sources":{}}}}`,
		`{"schema_version":1,"repositories":{"repo":{"repo_identity":"repo","revision":1,"agents":{"codex":{"argv":[],"confirmed":true,"conflict":null}},"legacy_sources":{}}}}`,
		`{"schema_version":1,"repositories":{"repo":{"repo_identity":"repo","revision":1,"agents":{"codex":{"argv":[null],"confirmed":true,"conflict":false}},"legacy_sources":{}}}}`,
		`{"schema_version":1,"repositories":{"repo":{"repo_identity":"repo","revision":1,"agents":{"codex":{"argv":[],"confirmed":true,"conflict":true}},"legacy_sources":{}}}}`,
		`{"schema_version":1,"repositories":{},"future":{"x":1,"x":2}}`,
	} {
		if _, err := DecodeRepoPreferences([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestRepoPreferencesCodecEncodedBounds(t *testing.T) {
	doc := RepoPreferences{SchemaVersion: 1, Repositories: map[string]RepoLaunchPreference{}}
	for i := 0; i < MaxRepoPreferenceRepos; i++ {
		key := fmt.Sprint(i)
		p, err := RememberRepoArguments(nil, key, LaunchProfile{Agent: "codex", Argv: []string{}})
		if err != nil {
			t.Fatal(err)
		}
		doc.Repositories[key] = p
	}
	if _, err := EncodeRepoPreferences(doc); err != nil {
		t.Fatal(err)
	}
	doc.Repositories["extra"] = doc.Repositories["0"]
	if _, err := EncodeRepoPreferences(doc); err == nil {
		t.Fatal("accepted repository overflow")
	}
	doc.Repositories = map[string]RepoLaunchPreference{}
	doc.Unknown = map[string]json.RawMessage{"future": json.RawMessage(`""`)}
	base, err := EncodeRepoPreferences(doc)
	if err != nil {
		t.Fatal(err)
	}
	doc.Unknown["future"] = json.RawMessage(`"` + strings.Repeat("x", MaxRepoPreferenceBytes-len(base)) + `"`)
	raw, err := EncodeRepoPreferences(doc)
	if err != nil || len(raw) != MaxRepoPreferenceBytes {
		t.Fatalf("exact byte bound: %d %v", len(raw), err)
	}
	if _, err := DecodeRepoPreferences(raw); err != nil {
		t.Fatal(err)
	}
	doc.Unknown["future"] = append(doc.Unknown["future"][:len(doc.Unknown["future"])-1], 'x', '"')
	if _, err := EncodeRepoPreferences(doc); err == nil {
		t.Fatal("accepted encoded byte overflow")
	}
}

func TestRepoPreferencesLegacyUnknownAndDigest(t *testing.T) {
	raw := []byte(`{"schema_version":1,"repo_identity":"/repo/.git","physical_path":"/repo/sub","last_agent":"codex","argv_by_agent":{"codex":[]},"revision":1,"extra":{ "x" : 2 }}`)
	s, err := DecodeLegacyRepoPreference("source", raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Digest) != 64 || !bytes.Equal(s.Raw, raw) {
		t.Fatalf("lost original evidence: %+v", s)
	}
	if _, err := DecodeLegacyRepoPreference("source", []byte(strings.Replace(string(raw), `"revision":1`, `"revision":1,"revision":2`, 1))); err == nil {
		t.Fatal("accepted duplicate legacy key")
	}
}

func TestRepoPreferencesLegacyOriginalWhitespaceSurvives(t *testing.T) {
	s := repoLegacy(t, "legacy", "/repo/a", "one")
	raw := append([]byte(" \n"), s.Raw...)
	raw = append(raw, '\n', '\t')
	s, err := DecodeLegacyRepoPreference("legacy", raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := FoldPathPreferences(nil, "/repo/.git", []LegacyRepoPreference{s})
	if err != nil {
		t.Fatal(err)
	}
	doc := RepoPreferences{SchemaVersion: 1, Repositories: map[string]RepoLaunchPreference{p.RepoIdentity: p}}
	encoded, err := EncodeRepoPreferences(doc)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeRepoPreferences(encoded)
	if err != nil {
		t.Fatal(err)
	}
	got := decoded.Repositories[p.RepoIdentity].LegacySources["legacy"]
	if !bytes.Equal(got.Raw, raw) || got.Digest != s.Digest {
		t.Fatalf("source image changed: %q", got.Raw)
	}
}

func TestRepoPreferencesLegacyRejectsLossySourceText(t *testing.T) {
	s := repoLegacy(t, "legacy", "/repo/a", "one")
	raw := bytes.Replace(s.Raw, []byte(`"one"`), []byte{'"', 0xff, '"'}, 1)
	if _, err := DecodeLegacyRepoPreference("legacy", raw); err == nil {
		t.Fatal("accepted source text that cannot round-trip through JSON string transport")
	}
}

func TestRepoPreferencesLegacyRejectsNullArgumentElement(t *testing.T) {
	s := repoLegacy(t, "legacy", "/repo/a", "one")
	raw := bytes.Replace(s.Raw, []byte(`"one"`), []byte(`null`), 1)
	if _, err := DecodeLegacyRepoPreference("legacy", raw); err == nil {
		t.Fatal("accepted null legacy argument as empty string")
	}
}

func TestRepoPreferencesRejectInvalidUTF8Document(t *testing.T) {
	raw := []byte("{\"schema_version\":1,\"repositories\":{},\"future\":\"\xff\"}")
	if _, err := DecodeRepoPreferences(raw); err == nil {
		t.Fatal("accepted document whose raw text cannot survive UTF-8 round trip")
	}
}

func TestRepoPreferencesCannotWriteInvalidUTF8UnknownValue(t *testing.T) {
	doc := RepoPreferences{SchemaVersion: 1, Repositories: map[string]RepoLaunchPreference{}, Unknown: map[string]json.RawMessage{"future": json.RawMessage{'"', 0xff, '"'}}}
	if _, err := EncodeRepoPreferences(doc); err == nil {
		t.Fatal("wrote raw document the reader must reject")
	}
}
