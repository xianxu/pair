package testcouchbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/internal/reporoot"
)

type nativeFacts struct {
	root, namespace, dir, couch string
	env                         []string
}
type nativeFleet interface {
	Prepare(context.Context) error
	Start(context.Context) error
	Done() <-chan struct{}
	Close(context.Context) error
	AssertCleaned() error
	Facts() nativeFacts
}
type ownedNativeFleet struct{ *testcouch.Fleet }

func (f ownedNativeFleet) Facts() nativeFacts {
	return nativeFacts{root: f.Roots.Root, namespace: f.Roots.Mutable["store"], dir: f.Workspaces.Primary, couch: filepath.Join(f.Roots.Mutable["bin"], "couch"), env: append([]string(nil), f.Environment...)}
}

type nativeDependencies struct {
	agentKind string
	source    func() (string, error)
	build     func(context.Context, testcouch.BuildConfig) (*testcouch.CandidateOwner, error)
	newFleet  func(context.Context, testcouch.Binaries, testcouch.Options, testcouch.OwnerConfig) (nativeFleet, error)
	run       func(context.Context, string, []string, string, []string) ([]byte, error)
}

type nativeReporter struct {
	diagnostics io.Writer
	failed      atomic.Bool
}

func (r *nativeReporter) Logf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.diagnostics, format+"\n", args...)
}
func (r *nativeReporter) Failed() bool { return r.failed.Load() }

// NativeOwner adapts the shared native-fleet owner without giving the consumer
// authority over descendant discovery or cleanup. Calls are serialized by RunBridge.
type NativeOwner struct {
	options                            Options
	reporter                           *nativeReporter
	dependencies                       nativeDependencies
	candidate                          *testcouch.CandidateOwner
	fleet                              nativeFleet
	cancel                             context.CancelFunc
	prepared, started, closed, cleaned bool
	closeErr                           error
}

// NewNativeOwner performs no filesystem or process effects. Prepare selects the
// checkout from the physical bridge executable, never the caller's working dir.
func NewNativeOwner(options Options, diagnostics io.Writer) *NativeOwner {
	return newNativeOwner(options, diagnostics, nativeDependencies{source: nativeSource, build: testcouch.BuildOwned, newFleet: func(ctx context.Context, b testcouch.Binaries, o testcouch.Options, c testcouch.OwnerConfig) (nativeFleet, error) {
		f, err := testcouch.NewOwned(ctx, b, o, c)
		if f == nil {
			return nil, err
		}
		return ownedNativeFleet{f}, err
	}, run: nativeCommand})
}
func newNativeOwner(options Options, diagnostics io.Writer, deps nativeDependencies) *NativeOwner {
	if diagnostics == nil {
		diagnostics = io.Discard
	}
	return &NativeOwner{options: options, reporter: &nativeReporter{diagnostics: diagnostics}, dependencies: deps}
}
func nativeSource() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	return reporoot.Find(filepath.Dir(executable))
}
func nativeCommand(ctx context.Context, program string, args []string, dir string, env []string) ([]byte, error) {
	return (couchcore.OSProvisionIO{Env: env}).Run(ctx, couchcore.ProvisionCommand{Program: program, Args: args, Dir: dir})
}
func (o *NativeOwner) failed(err error) error {
	if err != nil {
		o.reporter.failed.Store(true)
	}
	return err
}

func (o *NativeOwner) Prepare(ctx context.Context) (err error) {
	defer func() { o.failed(err) }()
	if o.closed || o.prepared || o.candidate != nil || o.fleet != nil {
		return errors.New("native owner preparation already attempted")
	}
	if err = ValidateBridgeOptions(o.options); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	source, err := o.dependencies.source()
	if err != nil {
		return err
	}
	o.candidate, err = o.dependencies.build(ctx, testcouch.BuildConfig{Source: source, ScratchParent: o.options.ScratchParent, Couch: o.options.Couch, Log: o.reporter.diagnostics})
	if err != nil {
		return err
	}
	if o.candidate == nil {
		return errors.New("candidate build returned no owner")
	}
	o.reporter.Logf("native candidate owner: source=%s root=%s", o.candidate.Binaries.Source, o.candidate.Binaries.Dir)
	// Admission consumes ctx, while the admitted supervisor belongs to this
	// adapter until Close; startup cancellation cannot kill a ready fleet.
	lifetime, cancel := context.WithCancel(context.WithoutCancel(ctx))
	o.cancel = cancel
	kind := o.dependencies.agentKind
	if kind == "" {
		kind = "claude"
	}
	o.fleet, err = o.dependencies.newFleet(lifetime, o.candidate.Binaries, testcouch.Options{Workers: 1, AgentKind: kind, RuntimeBudget: LifetimeBudget}, testcouch.OwnerConfig{ScratchParent: o.options.ScratchParent, Reporter: o.reporter})
	if err != nil {
		return err
	}
	if o.fleet == nil {
		return errors.New("native fleet returned no owner")
	}
	if err = o.fleet.Prepare(ctx); err != nil {
		return err
	}
	o.prepared = true
	return nil
}
func (o *NativeOwner) Start(ctx context.Context) (err error) {
	defer func() { o.failed(err) }()
	if o.closed || !o.prepared || o.started {
		return errors.New("native owner is not prepared for start")
	}
	if err = o.fleet.Start(ctx); err != nil {
		return err
	}
	o.started = true
	return nil
}
func (o *NativeOwner) Done() <-chan struct{} {
	if o.fleet == nil {
		return nil
	}
	return o.fleet.Done()
}

func (o *NativeOwner) command(ctx context.Context, program string, args []string, dir string, env []string) ([]byte, error) {
	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return o.dependencies.run(bounded, program, args, dir, env)
}

func (o *NativeOwner) Readiness(ctx context.Context) (r Readiness, err error) {
	defer func() { o.failed(err) }()
	if o.closed || !o.started {
		return r, errors.New("native owner is not running")
	}
	facts := o.fleet.Facts()
	r = Readiness{Version: 1, Namespace: facts.namespace, Environment: map[string]string{}}
	for _, entry := range facts.env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			return r, errors.New("invalid native environment entry")
		}
		if _, exists := r.Environment[key]; exists {
			return r, fmt.Errorf("duplicate native environment key %s", key)
		}
		r.Environment[key] = value
	}
	r.Socket, err = couchmessage.SocketPathForIsolation(facts.namespace, "editor-endpoint", facts.root)
	if err != nil {
		return r, err
	}
	paths := []string{filepath.Join(facts.root, "artifacts", "first.md"), filepath.Join(facts.root, "artifacts", "first.side", "second.md")}
	contents := []string{"---\ntype: prose\n---\n\n# Bridge parent\n", "---\ntype: side-doc\nparent: ../first.md\n---\n\n# Bridge side document\n"}
	for i, path := range paths {
		if err = ctx.Err(); err != nil {
			return r, err
		}
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return r, err
		}
		var file *os.File
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return r, err
		}
		_, writeErr := io.WriteString(file, contents[i])
		err = errors.Join(writeErr, file.Close())
		if err != nil {
			return r, err
		}
		var raw []byte
		raw, err = o.command(ctx, facts.couch, []string{"--session-sync", "register", path, "--owner", "fixture:0", "--json"}, facts.dir, facts.env)
		if err != nil {
			return r, fmt.Errorf("register bridge document: %w", err)
		}
		if len(raw) > ReadinessLimit {
			return r, errors.New("document registration exceeds readiness limit")
		}
		var registration struct {
			Code, Error  string
			Registration *struct{ ID, Namespace, File, Owner string }
		}
		if err = strictjson.RejectDuplicateKeysPreserveNumbers(raw); err != nil {
			return r, err
		}
		if err = json.Unmarshal(raw, &registration); err != nil {
			return r, err
		}
		if registration.Code != "ok" || registration.Error != "" || registration.Registration == nil {
			return r, errors.New("bridge document registration not confirmed")
		}
		doc := registration.Registration
		if !documentID(doc.ID) || doc.Namespace != facts.namespace || doc.File != path || doc.Owner != "fixture:0" {
			return r, errors.New("bridge registration identity mismatch")
		}
		r.Documents = append(r.Documents, Document{ID: doc.ID, File: path})
	}
	for _, doc := range r.Documents {
		raw, e := o.command(ctx, facts.couch, []string{"--editor-endpoint", "lookup", "--namespace", facts.namespace, "--document-id", doc.ID, "--json"}, facts.dir, facts.env)
		if e != nil {
			return r, fmt.Errorf("editor endpoint readiness roundtrip: %w", e)
		}
		if len(raw) > ReadinessLimit {
			return r, errors.New("editor endpoint response exceeds readiness limit")
		}
		var reply struct {
			Version        int
			Code, Error    string
			EditorEndpoint json.RawMessage
		}
		if e = strictjson.RejectDuplicateKeysPreserveNumbers(raw); e != nil {
			return r, e
		}
		if e = json.Unmarshal(raw, &reply); e != nil {
			return r, e
		}
		if reply.Version != 1 || reply.Code != "ok" || reply.Error != "" || string(reply.EditorEndpoint) != "null" {
			return r, errors.New("editor endpoint did not confirm empty readiness")
		}
	}
	return r, ValidateReadiness(r)
}

func (o *NativeOwner) Close(ctx context.Context) error {
	if o.closed {
		return o.closeErr
	}
	o.closed = true
	if o.fleet != nil {
		if err := o.fleet.Close(ctx); err != nil {
			o.closeErr = err
			return err
		}
		if err := o.fleet.AssertCleaned(); err != nil {
			o.closeErr = err
			return err
		}
	}
	if o.cancel != nil {
		o.cancel()
	}
	if o.candidate != nil {
		if err := o.candidate.Close(ctx); err != nil {
			o.closeErr = err
			return err
		}
	}
	o.cleaned = true
	return nil
}
func (o *NativeOwner) AssertCleaned() error {
	if !o.cleaned {
		return errors.Join(errors.New("native owner cleanup unconfirmed"), o.closeErr)
	}
	if o.fleet != nil {
		return o.fleet.AssertCleaned()
	}
	return nil
}
func (o *NativeOwner) RetainedRoot() string {
	if o.cleaned {
		return ""
	}
	if o.fleet != nil {
		return o.fleet.Facts().root
	}
	if o.candidate != nil {
		return o.candidate.Binaries.Dir
	}
	return ""
}
