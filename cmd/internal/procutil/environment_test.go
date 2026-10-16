package procutil

import "testing"

func TestParseEnvironmentPicksOnlyNamedKeys(t *testing.T) {
	raw := []byte("PATH=/bin\x00COUCH_THREAD_SCOPE=abc\x00COUCH_THREAD_TAG=1-pair-8\x00SECRET=x\x00COUCH_THREAD_TAG=dup\x00junk\x00")
	got := ParseEnvironment(raw, "COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG")
	if len(got) != 2 || got["COUCH_THREAD_SCOPE"] != "abc" || got["COUCH_THREAD_TAG"] != "1-pair-8" {
		t.Fatalf("parsed %v", got)
	}
}
