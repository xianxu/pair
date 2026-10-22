package agentfixture

import "testing"

func TestRetainedCommandIsClosed(t *testing.T) {
	if err := ValidateCommand(Command{ID: "retain", Kind: Retain}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []Command{{ID: "retain", Kind: Retain, Args: []string{"arbitrary"}}, {ID: "retain", Kind: Retain, State: Idle}, {ID: "retain", Kind: Retain, Timeout: 1}} {
		if ValidateCommand(c) == nil {
			t.Fatalf("accepted %+v", c)
		}
	}
}
func TestRetainedEnvironmentPreservesOnlyRunOwnership(t *testing.T) {
	got := retainedEnvironment([]string{"PATH=/private/bin", "PAIR_TESTCOUCH_RUN_ID=mine", "COUCH_THREAD_SCOPE=old", "COUCH_THREAD_TAG=old", "COUCH_SESSION_ID=old", "COUCH_CONNECTION_NONCE=old"})
	if len(got) != 2 || got[0] != "PATH=/private/bin" || got[1] != "PAIR_TESTCOUCH_RUN_ID=mine" {
		t.Fatalf("%v", got)
	}
}

func TestRetainedHelperRejectsUnownedMode(t *testing.T) {
	t.Setenv("PAIR_TESTCOUCH_ROOT", "")
	handled, status := RetainedHelper([]string{retainedParentMode, t.TempDir()})
	if !handled || status != 2 {
		t.Fatalf("handled=%v status=%d", handled, status)
	}
	if handled, _ := RetainedHelper([]string{"ordinary-agent-arg"}); handled {
		t.Fatal("claimed normal agent mode")
	}
}
