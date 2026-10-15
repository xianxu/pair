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
// when it changes. Validated activity timestamps likewise reuse only unchanged
// ledger/catalog/native dependency observations; changed inputs still validate.
type OSRuntime struct {
	Home       string
	launches   *sessioninventory.OwnerLaunchCache
	activities *sessioninventory.ActivityCache
	observer   sessioninventory.QueryWorkObserver
}

func NewOSRuntime(home string) OSRuntime {
	return OSRuntime{Home: home, launches: sessioninventory.NewOwnerLaunchCache(), activities: sessioninventory.NewActivityCache()}
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
	activity, ok, err := r.activities.Query(ctx, runtime, t.Scope, t.Tag, sessioninventory.Agent(t.Agent))
	if err != nil || !ok || ctx.Err() != nil {
		return time.Time{}, false
	}
	return activity.LastActivityAt, true
}

// inventory is the session inventory over one thread's scope, sharing this
// runtime's ledger cache.
func (r OSRuntime) inventory(scopeDir string) sessioninventory.OSRuntime {
	return sessioninventory.NewOSRuntime(r.Home, scopeDir).WithOwnerLaunchCache(r.launches).WithWorkObserver(r.observer)
}

// WithWorkObserver exposes the same counted query seam to diagnostics and
// invariant tests. It changes no authority or cache policy.
func (r OSRuntime) WithWorkObserver(observer sessioninventory.QueryWorkObserver) OSRuntime {
	r.observer = observer
	return r
}

var _ Runtime = OSRuntime{}
