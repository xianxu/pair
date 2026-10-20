package repolint

import (
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/golint"
)

func TestDrainStateReducerRule(t *testing.T) {
	for _, tc := range []struct {
		name, function, body string
		want                 int
	}{
		{"direct phase", "func (c *drainCoordinator) reconcile()", `c.record.State.Phase = next`, 1},
		{"renamed receiver", "func (owner *drainCoordinator) persist()", `owner.record.State.Error = "failed"`, 1},
		{"new method is guarded", "func (c drainCoordinator) newlyAdded()", `c.record.State.Error = "failed"`, 1},
		{"nested closure", "func (c *drainCoordinator) reconcile()", `go func() { c.record.State.Error = "failed" }()`, 1},
		{"multiple fields", "func (c *drainCoordinator) reconcile()", `c.record.State.Phase, c.record.State.Error = next, "failed"`, 2},
		{"state copy", "func (c *drainCoordinator) reconcile()", `state := c.record.State; state.Error = "failed"; c.record.State = state`, 1},
		{"state pointer", "func (c *drainCoordinator) reconcile()", `state := &c.record.State; state.Phase = next`, 1},
		{"coordinator alias", "func (c *drainCoordinator) reconcile()", `owner := c; owner.record.State.Error = "failed"`, 1},
		{"record alias", "func (c *drainCoordinator) reconcile()", `record := &c.record; record.State.Error = "failed"`, 1},
		{"field pointer alias", "func (c *drainCoordinator) reconcile()", `failure := &c.record.State.Error; *failure = "failed"`, 1},
		{"alias chain", "func (c *drainCoordinator) reconcile()", `state := &c.record.State; failure := &(state.Error); another := failure; *another = "failed"`, 1},
		{"var aliases", "func (c *drainCoordinator) reconcile()", `var state = c.record.State; state.Phase = next`, 1},
		{"multiple aliases", "func (c *drainCoordinator) reconcile()", `state, record := c.record.State, &c.record; state.Phase, record.State.Error = next, "failed"`, 2},
		{"reducer output cannot be patched", "func (c *drainCoordinator) reconcile()", `state, err := couchcore.ReduceDrain(c.record.State, event); state.Error = err.Error(); c.record.State = state`, 1},
		{"typed local state", "func (c *drainCoordinator) reconcile()", `var state couchcore.DrainState; state.Error = "failed"`, 1},
		{"free helper typed owner", "func persist(owner *drainCoordinator)", `owner.record.State.Error = "failed"`, 1},
		{"parenthesized target", "func (owner *drainCoordinator) persist()", `(owner.record.State).Error = "failed"`, 1},
		{"new owner", "func makeOwner()", `owner := new(drainCoordinator); owner.record.State.Error = "failed"`, 1},
		{"typed owner", "func makeOwner()", `var owner *drainCoordinator; owner.record.State.Error = "failed"`, 1},
		{"declared alias assigned later", "func (c *drainCoordinator) reconcile()", `var state couchcore.DrainState; state = c.record.State; state.Phase = next`, 1},
		{"nested alias scope", "func (c *drainCoordinator) reconcile()", `state := c.record.State; func() { state.Error = "failed" }()`, 1},
		{"constructor local owner", "func newOwner()", `owner := &drainCoordinator{}; owner.record.State.Phase = next`, 1},
		{"typed nested parameter", "func (c *drainCoordinator) reconcile()", `func(owner *drainCoordinator) { owner.record.State.Error = "failed" }(c)`, 1},
		{"whole state initialization", "func (c *drainCoordinator) reconcile()", `c.record.State = couchcore.DrainState{Phase: initial, Error: "unknown"}`, 0},
		{"whole record initialization", "func (c *drainCoordinator) reconcile()", `c.record = drainServiceRecord{State: couchcore.DrainState{Phase: initial, Error: "unknown"}}`, 0},
		{"reducer result", "func (c *drainCoordinator) reconcile()", `c.record.State, _ = couchcore.ReduceDrain(c.record.State, event)`, 0},
		{"unrelated fields", "func (c *drainCoordinator) reconcile()", `view := other{}; view.Phase, view.Error = next, "failed"; c.opts.Error = "failed"`, 0},
		{"unrelated receiver", "func (c *other) reconcile()", `c.record.State.Phase = next`, 0},
		{"fixture state", "func makeFixture()", `record := drainServiceRecord{}; record.State.Phase = next; state := couchcore.DrainState{}; state.Error = "fixture"`, 0},
		{"shadowed receiver", "func (c *drainCoordinator) reconcile()", `{ c := other{}; c.record.State.Phase = next }`, 0},
		{"shadowed state alias", "func (c *drainCoordinator) reconcile()", `state := c.record.State; { state := other{}; state.Error = "other" }; _ = state`, 0},
		{"read only", "func (c *drainCoordinator) reconcile()", `if c.record.State.Error != "" { publish(c.record.State.Phase) }`, 0},
		{"string only", "func (c *drainCoordinator) reconcile()", `_ = "c.record.State.Error = failure"`, 0},
		{"reasoned suppression", "func (c *drainCoordinator) reconcile()", "//lint:allow drain-state-reducer -- controlled compatibility example\nc.record.State.Error = failure", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package couchcmd\n" + tc.function + " {\n" + tc.body + "\n}\n")
			findings, err := golint.Check("cmd/internal/couchcmd/drain_service.go", source, pairCatalog)
			if err != nil || len(findings) != tc.want {
				t.Fatalf("err=%v findings=%+v want=%d", err, findings, tc.want)
			}
			for _, f := range findings {
				if f.Rule != "drain-state-reducer" || !strings.Contains(f.Message, "pair#722") {
					t.Fatalf("unexpected finding: %+v", f)
				}
			}
		})
	}
	t.Run("other package", func(t *testing.T) {
		findings, err := golint.Check("other.go", []byte(`package other
func (c *drainCoordinator) reconcile() { c.record.State.Error = "failed" }`), pairCatalog)
		if err != nil || len(findings) != 0 {
			t.Fatalf("err=%v findings=%+v", err, findings)
		}
	})
}
