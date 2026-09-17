package terminal

import (
	"errors"
	"strings"
	"testing"
)

func TestNoDestinationIsClassifiableAndNamesTheView(t *testing.T) {
	err := noDestination("no admitted endpoint", View{State: Presenting, Selected: "couch-pty-3"})
	if !errors.Is(err, ErrNoDestination) {
		t.Fatalf("errors.Is(%v, ErrNoDestination) = false", err)
	}
	for _, want := range []string{"no admitted endpoint", "couch-pty-3", "state=1"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name %q", err, want)
		}
	}
}
