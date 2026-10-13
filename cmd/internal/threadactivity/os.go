package threadactivity

import (
	"context"
	"os"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

// OSRuntime reads real file mtimes and the native session inventory. The
// inventory is built per thread over its own scope directory, the way
// couchcore's switch context does, because bindings live under the scope.
// Its ledger cache spans every probe (#507): a ledger is decoded again only
// when it changes.
type OSRuntime struct {
	Home     string
	launches *sessioninventory.OwnerLaunchCache
}

func NewOSRuntime(home string) OSRuntime {
	return OSRuntime{Home: home, launches: sessioninventory.NewOwnerLaunchCache()}
}

func (OSRuntime) ModTime(path string) (time.Time, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

// SessionActivity is the established root transcript's last activity, or false
// when the thread has no exact established binding. ctx bounds the query.
func (r OSRuntime) SessionActivity(ctx context.Context, t Thread) (time.Time, bool) {
	runtime := r.inventory(t.ScopeDir)
	query, err := sessioninventory.QuerySessionContext(ctx, runtime, t.Scope, t.Tag, sessioninventory.Agent(t.Agent))
	if err != nil || ctx.Err() != nil {
		return time.Time{}, false
	}
	activity, ok, err := sessioninventory.ActivityForSession(runtime, query)
	if err != nil || !ok {
		return time.Time{}, false
	}
	return activity.LastActivityAt, true
}

// inventory is the session inventory over one thread's scope, sharing this
// runtime's ledger cache.
func (r OSRuntime) inventory(scopeDir string) sessioninventory.OSRuntime {
	return sessioninventory.NewOSRuntime(r.Home, scopeDir).WithOwnerLaunchCache(r.launches)
}

var _ Runtime = OSRuntime{}
