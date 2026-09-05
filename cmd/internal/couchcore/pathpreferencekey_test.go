package couchcore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The operator's per-path agent+argv memory lives in files named by
// sha256(repoIdentity \0 physicalPath). Nothing reads those files by any other
// route, so a change to either input silently orphans every preference: couch
// keeps working, and simply forgets which agent you used in which tree.
//
// pair#170 M4 deletes the fleet-policy provider that supplied repoIdentity and
// derives it locally instead. This is the characterization test that makes that
// substitution provable rather than hopeful -- it must pass BEFORE the change
// and after.
//
// The expected digest is not invented for the test: it is the name of a real
// file in the operator's live store
// (~/.local/share/pair/couch/threadstore/path-preferences/), whose contents
// record repo_identity=/Users/xianxu/workspace/tools/.git and
// physical_path=/Users/xianxu/workspace/tools. Pinning a value the running
// system actually produced is what makes this a regression guard rather than a
// restatement of the implementation.
func TestPathLaunchPreferenceKeyIsStable(t *testing.T) {
	const (
		repoIdentity = "/Users/xianxu/workspace/tools/.git"
		physicalPath = "/Users/xianxu/workspace/tools"
		wantFile     = "26f59d14188f9bdb3ea1c088997dc236573a46f88bab8c535b0a4c614e086943.json"
	)
	store, _ := newTestThreadStore(t)
	got := store.pathLaunchPreferencePath(repoIdentity, physicalPath)
	if base := got[len(got)-len(wantFile):]; base != wantFile {
		t.Fatalf("preference key = %q, want %q -- re-keying orphans every saved agent preference", base, wantFile)
	}
}

// And the round trip, so the guard covers the read path too rather than only
// the name: a preference written under a repository identity must come back
// under the identical one.
func TestPathLaunchPreferenceRoundTripsUnderItsIdentity(t *testing.T) {
	const (
		repoIdentity = "/Users/xianxu/workspace/tools/.git"
		physicalPath = "/Users/xianxu/workspace/tools"
	)
	store, _ := newTestThreadStore(t)
	profile := LaunchProfile{Agent: "claude", Argv: []string{"--dangerously-skip-permissions"}}

	written, err := RecordSuccessfulLaunch(nil, repoIdentity, physicalPath, profile)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(written, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(store.pathLaunchPreferencePath(repoIdentity, physicalPath)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomicBytes(store.pathLaunchPreferencePath(repoIdentity, physicalPath), append(raw, '\n')); err != nil {
		t.Fatal(err)
	}

	got, found, err := store.GetPathLaunchPreference(repoIdentity, physicalPath)
	if err != nil || !found {
		t.Fatalf("GetPathLaunchPreference() = (_, %v, %v), want a stored preference", found, err)
	}
	if got.LastAgent != "claude" {
		t.Fatalf("last agent = %q, want claude", got.LastAgent)
	}
	if _, found, _ := store.GetPathLaunchPreference("/some/other/repo/.git", physicalPath); found {
		t.Fatal("a different repository identity read the same preference -- the identity is not part of the key")
	}
}
