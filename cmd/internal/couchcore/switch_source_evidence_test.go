package couchcore

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSwitchSourceEvidenceRequiresCoreClassificationAndExactRecord(t *testing.T) {
	env := newTestEnv(t, "/repo")
	source := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
	argv := []string{}
	prepared, err := env.Couch.PrepareAgentSwitch(context.Background(), source.Address, "codex", &argv)
	if err != nil {
		t.Fatal(err)
	}
	evidence := prepared.SourceEvidence()
	if !evidence.Vacant() || !evidence.MatchesRecord(source) || evidence.WorkingPath() != source.WorkingPath {
		t.Fatal("core vacancy proof lost its source binding")
	}
	for _, mutate := range []func(*ThreadRecord){func(r *ThreadRecord) { r.Revision++ }, func(r *ThreadRecord) { r.WorkingPath += "/other" }, func(r *ThreadRecord) { r.Address.Tag += "-other" }} {
		changed := source
		mutate(&changed)
		if evidence.MatchesRecord(changed) {
			t.Fatal("vacancy proof accepted another source")
		}
	}
	relabelled := prepared
	relabelled.Address.Tag += "-other"
	relabelled.SourceRevision++
	relabelled.WorkingPath += "/other"
	if !relabelled.SourceEvidence().MatchesRecord(source) {
		t.Fatal("public preview edits changed core-owned source authority")
	}
	raw, err := json.Marshal(prepared)
	if err != nil {
		t.Fatal(err)
	}
	var decoded PreparedAgentSwitch
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SourceEvidence().Vacant() {
		t.Fatal("serialized preview minted vacancy authority")
	}
}
