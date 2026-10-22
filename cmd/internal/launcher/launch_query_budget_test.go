package launcher

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

// Keep the real launcher orchestration; only the query/interactive OS boundary
// is substituted. Legacy methods intentionally remain available to catch an
// accidental bypass of the context-aware boundary.
type launchBudgetRuntime struct {
	*couchSessionRuntime
	queries   []context.Context
	lifetime  context.Context
	startup   context.Context
	admission context.Context
}

func (r *launchBudgetRuntime) SessionLivenessContext(ctx context.Context) ([]Session, error) {
	r.queries = append(r.queries, ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return r.fakeRuntime.SessionLiveness()
}
func (r *launchBudgetRuntime) AttachSessionContext(lifetime, startup, admission context.Context, session, config string) (int, error) {
	r.lifetime, r.startup, r.admission = lifetime, startup, admission
	if err := admission.Err(); err != nil {
		return 1, err
	}
	return r.fakeRuntime.AttachSession(session, config)
}

func TestCouchAttachSharesQueryBudgetThroughHandoff(t *testing.T) {
	opts := managedSessionOptions("attach")
	r := &launchBudgetRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}}
	r.sessions = []Session{{Name: opts.Args.CouchSession.Name, State: SessionDetached}}
	before := time.Now()
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, r, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, &stderr)
	}
	if len(r.queries) != 3 || r.admission == nil || r.lifetime == nil {
		t.Fatalf("context boundary bypassed: queries=%d admission=%v lifetime=%v", len(r.queries), r.admission, r.lifetime)
	}
	deadline, ok := r.admission.Deadline()
	if !ok || deadline.Before(before) || deadline.After(time.Now().Add(5*time.Second)) {
		t.Fatalf("unbounded/renewed admission: %v %v", deadline, ok)
	}
	for i, ctx := range r.queries {
		if ctx != r.admission {
			t.Fatalf("fence %d got a fresh context", i)
		}
	}
	if r.lifetime.Err() != nil {
		t.Fatalf("admission cancellation ended interactive lifetime: %v", r.lifetime.Err())
	}
	if _, ok := r.lifetime.Deadline(); ok {
		t.Fatal("interactive lifetime inherited query deadline")
	}
	if !errors.Is(r.admission.Err(), context.Canceled) {
		t.Fatalf("query owner did not close admission: %v", r.admission.Err())
	}
}

type cancelOwnerBudgetRuntime struct {
	*couchSessionRuntime
	cancel context.CancelFunc
}

func (r *cancelOwnerBudgetRuntime) ObserveSessionOwner(ctx context.Context, name, root, scope, tag string) (SessionOwnerObservation, error) {
	r.cancel()
	return SessionOwnerObservation{State: SessionOwnerForeign}, nil
}
func TestCouchCreateCancellationDuringOwnerObservationRefuses(t *testing.T) {
	opts := managedSessionOptions("create")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &cancelOwnerBudgetRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}, cancel: cancel}
	r.sessions = []Session{{Name: legacySessionPrefix + opts.Args.ForcedTag, State: SessionDetached}}
	err := verifyCouchSessionCreate(ctx, r, opts.GlobalDataDir, *opts.Args.CouchSession)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled owner evidence admitted: %v", err)
	}
}

func TestCouchAttachAdmissionPreservesEarlierCallerDeadline(t *testing.T) {
	opts := managedSessionOptions("attach")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	opts.Context = ctx
	r := &launchBudgetRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}}
	r.sessions = []Session{{Name: opts.Args.CouchSession.Name, State: SessionDetached}}
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, r, &stderr)
	if code != 0 || err != nil {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, &stderr)
	}
	want, _ := ctx.Deadline()
	for _, q := range r.queries {
		got, ok := q.Deadline()
		if !ok || !got.Equal(want) {
			t.Fatalf("lost earlier deadline: %v != %v", got, want)
		}
	}
	if r.lifetime != ctx {
		t.Fatal("interactive lifetime replaced caller context")
	}
	if ctx.Err() != nil {
		t.Fatalf("query completion canceled caller: %v", ctx.Err())
	}
}

func TestCouchAttachCanceledCallerRefusesBeforeEffects(t *testing.T) {
	for _, exported := range []bool{false, true} {
		t.Run(map[bool]string{false: "launch", true: "exported"}[exported], func(t *testing.T) {
			opts := managedSessionOptions("attach")
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			opts.Context = ctx
			r := &launchBudgetRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}}
			r.sessions = []Session{{Name: opts.Args.CouchSession.Name, State: SessionDetached}}
			var stderr bytes.Buffer
			var code int
			var err error
			if exported {
				code, err = AttachExistingSession(opts, opts.Env, r, opts.Args.CouchSession.Tag, opts.Args.CouchSession.Name, "")
			} else {
				code, err = RunLaunch(opts, r, &stderr)
			}
			if code == 0 || len(r.attached) != 0 || r.launchCount != 0 || len(r.files) != 0 || len(r.queries) != 0 {
				t.Fatalf("canceled launch made effects: code=%d err=%v queries=%d files=%v attached=%v", code, err, len(r.queries), r.files, r.attached)
			}
		})
	}
}

type createBudgetRuntime struct {
	*launchBudgetRuntime
	probes, owners []context.Context
}

func (r *createBudgetRuntime) ProbeSessionNameContext(ctx context.Context, name string) error {
	r.probes = append(r.probes, ctx)
	return r.fakeRuntime.ProbeSessionNameContext(ctx, name)
}
func (r *createBudgetRuntime) ObserveSessionOwner(ctx context.Context, name, root, scope, tag string) (SessionOwnerObservation, error) {
	r.owners = append(r.owners, ctx)
	return r.couchSessionRuntime.ObserveSessionOwner(ctx, name, root, scope, tag)
}
func (r *createBudgetRuntime) LaunchSessionContext(lifetime, startup, admission context.Context, session, config, layout string) (int, error) {
	r.lifetime, r.startup, r.admission = lifetime, startup, admission
	if err := admission.Err(); err != nil {
		return 1, err
	}
	return r.fakeRuntime.LaunchSession(lifetime, session, config, layout)
}
func TestCouchCreateSharesPrecreateQueryBudget(t *testing.T) {
	opts := managedSessionOptions("create")
	r := &createBudgetRuntime{launchBudgetRuntime: &launchBudgetRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime(), ownerState: SessionOwnerForeign}}}
	r.sessions = []Session{{Name: legacySessionPrefix + opts.Args.ForcedTag, State: SessionDetached}}
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, r, &stderr)
	if code != 0 || err != nil {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, &stderr)
	}
	if len(r.queries) != 4 || len(r.probes) != 2 || len(r.owners) != 2 || r.admission == nil {
		t.Fatalf("queries=%d probes=%d owners=%d admission=%v", len(r.queries), len(r.probes), len(r.owners), r.admission)
	}
	for _, set := range [][]context.Context{r.queries, r.probes, r.owners} {
		for _, ctx := range set {
			if ctx != r.admission {
				t.Fatal("precreate admission renewed query budget")
			}
		}
	}
	if _, ok := r.admission.Deadline(); !ok {
		t.Fatal("precreate admission has no deadline")
	}
}

func TestRunLaunchCarriesStartupDeadlineSeparateFromInteractiveLifetime(t *testing.T) {
	for _, disposition := range []string{"create", "attach"} {
		t.Run(disposition, func(t *testing.T) {
			opts := managedSessionOptions(disposition)
			life, stop := context.WithCancel(context.Background())
			defer stop()
			startup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			opts.Context, opts.StartupContext = life, startup
			base := &launchBudgetRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime(), ownerState: SessionOwnerForeign}}
			rt := &createBudgetRuntime{launchBudgetRuntime: base}
			if disposition == "attach" {
				base.sessions = []Session{{Name: opts.Args.CouchSession.Name, State: SessionDetached}}
			}
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if code != 0 || err != nil {
				t.Fatalf("code=%d error=%v stderr=%s", code, err, &stderr)
			}
			if base.lifetime == nil || life.Err() != nil {
				t.Fatalf("interactive lifetime missing or parent ended: %v", base.lifetime)
			}
			if _, bounded := base.lifetime.Deadline(); bounded {
				t.Fatal("interactive client inherited startup deadline")
			}
			if base.startup == nil {
				t.Fatal("startup context lost at Runtime boundary")
			}
			want, _ := startup.Deadline()
			got, ok := base.startup.Deadline()
			if !ok || !got.Equal(want) {
				t.Fatalf("startup=%v want=%v", got, want)
			}
			query, _ := base.admission.Deadline()
			if !query.Before(want) {
				t.Fatal("short native-query budget lost")
			}
		})
	}
}
