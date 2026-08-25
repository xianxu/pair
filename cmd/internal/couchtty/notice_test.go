package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestExitNoticeIsPerActorControlAndNamesCause(t *testing.T) {
	n := ExitNotice(couchcore.ActorID("couch-b1"), "brain", 17)
	m := n.Message()
	if m.Kind != "exit:couch-b1" {
		t.Fatalf("kind = %q, want exit:couch-b1", m.Kind)
	}
	if !m.Control {
		t.Fatal("exit notice is not control priority")
	}
	for _, want := range []string{"brain", "couch-b1", "17"} {
		if !strings.Contains(m.Body, want) {
			t.Fatalf("body %q does not name %q", m.Body, want)
		}
	}
}
