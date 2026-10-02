package couchcmd

import (
	"errors"
	"testing"
)

func TestIsolatedAuthorityDoesNotLookUpAccountHome(t *testing.T) {
	isolatedCouchTestEnvironment(t)
	called := false
	rt := OSRuntime{accountHome: func() (string, error) {
		called = true
		return "", errors.New("account-home lookup must not run in isolation")
	}}
	_, lease, err := rt.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if called {
		t.Fatal("isolated authority consulted the real account home")
	}
}
