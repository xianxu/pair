package agentfixture

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// ResourceObservation contains only native-fixture readiness facts. Authority
// and effect acceptance still traverse the real public CLI independently.
type ResourceObservation struct {
	Processes []ResourceProcessObservation
	Orphans   []ResourceProcessObservation
}
type ResourceProcessObservation struct {
	Key                 couchcore.ResourceProcessKey
	Slot, Name          string
	Orphan, OrphanKnown bool
}

func resourceObservationResult(id string, view couchcore.ResourceView) Result {
	project := func(rows []couchcore.ResourceProcessView) []ResourceProcessObservation {
		out := make([]ResourceProcessObservation, len(rows))
		for i, r := range rows {
			out[i] = ResourceProcessObservation{Key: r.Key, Slot: r.Slot, Name: r.Name, Orphan: r.Orphan, OrphanKnown: r.OrphanKnown}
		}
		return out
	}
	data, err := json.Marshal(ResourceObservation{Processes: project(view.Processes), Orphans: project(view.Orphans)})
	if err != nil {
		return Result{ID: id, ExitCode: -1, Error: err.Error()}
	}
	if len(data) > MaxOutputBytes {
		return Result{ID: id, ExitCode: -1, Error: "readonly resource observation exceeds existing fixture output limit"}
	}
	return Result{ID: id, Stdout: string(data)}
}

func readResourceObservation(parent context.Context, id string) Result {
	fail := func(err error) Result { return Result{ID: id, ExitCode: -1, Error: err.Error()} }
	ctx, cancel := context.WithTimeout(parent, couchmessage.TransportTimeout)
	defer cancel()
	namespace, err := couchcore.ExistingCouchNamespace(os.Getenv("COUCH_STORE_DIR"))
	if err != nil {
		return fail(err)
	}
	socket, err := couchmessage.SocketPath(namespace.Dir(), "broker")
	if err != nil {
		return fail(err)
	}
	var reply couchmessage.Response
	if err = couchmessage.Call(ctx, socket, couchmessage.Request{Op: "resources"}, &reply); err != nil {
		return fail(err)
	}
	if reply.Code != "ok" || reply.Resources == nil {
		return fail(fmt.Errorf("readonly resources: %s %s", reply.Code, reply.Error))
	}
	return resourceObservationResult(id, *reply.Resources)
}
