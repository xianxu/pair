package couchcore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// terminalClass names how a run ended: converged, the set of resources it
// stopped at, or the resource whose step failed.
func terminalClass(r ReconcileResult, err error) string {
	var rerr *ReconcileError
	switch {
	case errors.As(err, &rerr):
		return "error@" + string(rerr.Failure.Step.Resource)
	case err != nil:
		return "error:" + err.Error()
	case r.Plan.Empty():
		return "converged"
	}
	var at []string
	for _, s := range r.Plan.Stops {
		at = append(at, string(s.Resource)+"/"+string(s.Class))
	}
	sort.Strings(at)
	return "stopped@" + strings.Join(at, ",")
}

// affected is every resource a perturbation set may legitimately stop or fail
// at: the perturbed resources, what depends on them, and the couplings disks
// have (a missing environment takes the host, a missing host strands the
// registration).
func affected(ps []perturbation) map[SlotResourceID]bool {
	out := map[SlotResourceID]bool{}
	add := func(id SlotResourceID) {
		out[id] = true
		for d := range SlotResourceDependents(templateOf(id)) {
			out[d] = true
		}
	}
	for _, p := range ps {
		add(p.id)
		switch templateOf(p.id) {
		case ResourceEnv, ResourceHost:
			add(ResourceHost)
			add(ResourceRegistration)
		case ResourceDep:
			add(ResourceSetup)
		}
	}
	return out
}

func runWorld(t *testing.T, label string, agent EvidenceAgent, ps ...perturbation) {
	t.Helper()
	w := newSlotWorld()
	w.Agent = agent
	for _, p := range ps {
		w.set(p)
	}
	first, err := reconcileLoop(context.Background(), w, nil)
	class := terminalClass(first, err)
	ok := affected(ps)
	switch {
	case class == "converged":
	case strings.HasPrefix(class, "error@"):
		if id := SlotResourceID(strings.TrimPrefix(class, "error@")); !ok[templateOf(id)] && !ok[id] {
			t.Errorf("%s: failed at unrelated %s", label, id)
		}
	case strings.HasPrefix(class, "stopped@"):
		for _, s := range first.Plan.Stops {
			if !ok[templateOf(s.Resource)] && !ok[s.Resource] {
				t.Errorf("%s: stopped at unrelated %s (%s)", label, s.Resource, s.Reason)
			}
		}
	default:
		t.Errorf("%s: ended %s", label, class)
	}
	running, known := AgentRunning(agent)
	for _, e := range w.Effects {
		if strings.HasPrefix(e, string(StepSetAside)) && (running || !known) {
			t.Errorf("%s: set aside under agent %s: %s", label, agent, e)
		}
	}
	effects := len(w.Effects)
	second, err := reconcileLoop(context.Background(), w, nil)
	if again := terminalClass(second, err); again != class && !(class != "converged" && strings.HasPrefix(class, "error@")) {
		t.Errorf("%s: second run ended %s, first %s", label, again, class)
	}
	if class == "converged" && len(w.Effects) != effects {
		t.Errorf("%s: second run of a converged slot applied %v", label, w.Effects[effects:])
	}
}

// TestReconcileConvergesFromEveryPerturbation is Done-when bullet 2 over the
// derived domain: every single and pair perturbation of the table, under every
// agent state, in SlotWorld (agreeing with real git on the single domain,
// TestReconcileAgreesWithRealGit).
func TestReconcileConvergesFromEveryPerturbation(t *testing.T) {
	domain := slotPerturbations()
	runs := 0
	for _, agent := range AllEvidenceAgents() {
		for i, a := range domain {
			runWorld(t, fmt.Sprintf("agent=%s %s", agent, a), agent, a)
			runs++
			for _, b := range domain[i+1:] {
				if b.id != a.id {
					runWorld(t, fmt.Sprintf("agent=%s %s %s", agent, a, b), agent, a, b)
					runs++
				}
			}
		}
	}
	if runs < 5000 {
		t.Fatalf("only %d runs; the domain is not read from the table", runs)
	}
}

// TestReconcileCrashAfterEveryStep: a run interrupted after any step, then run
// again, converges as an uninterrupted run does, and no checkout is set aside
// twice.
func TestReconcileCrashAfterEveryStep(t *testing.T) {
	scenarios := map[string][]perturbation{
		"deleted slot directory": {{ResourceEnv, StateAbsent, ""}, {ResourceRegistration, StateBroken, SubStale}},
		"broken dependency":      {{DepResource(fakeDep), StateBroken, SubUnreadable}},
		"unreadable host":        {{ResourceHost, StateBroken, SubUnreadable}},
		"interrupted setup":      {{DepResource(fakeDep), StateAbsent, ""}, {ResourceSetup, StateAbsent, ""}},
	}
	for name, ps := range scenarios {
		clean := newSlotWorld()
		clean.RepairFixes = false
		for _, p := range ps {
			clean.set(p)
		}
		want, err := reconcileLoop(context.Background(), clean, nil)
		wantClass := terminalClass(want, err)
		for k := 1; k <= len(clean.Effects); k++ {
			w := newSlotWorld()
			w.RepairFixes = false
			for _, p := range ps {
				w.set(p)
			}
			w.CrashAfter = k
			crashed := func() (died bool) {
				defer func() { died = recover() == errInjectedCrash }()
				reconcileLoop(context.Background(), w, nil)
				return false
			}()
			if !crashed && k < len(clean.Effects) {
				t.Errorf("%s crash@%d: the run did not reach the injected crash", name, k)
			}
			w.CrashAfter = 0
			got, err := reconcileLoop(context.Background(), w, nil)
			if class := terminalClass(got, err); class != wantClass {
				t.Errorf("%s crash@%d: rerun ended %s, an uninterrupted run %s", name, k, class, wantClass)
			}
			asides := map[string]int{}
			for _, e := range w.Effects {
				if strings.HasPrefix(e, string(StepSetAside)) {
					asides[e]++
				}
			}
			for e, n := range asides {
				if n > 1 {
					t.Errorf("%s crash@%d: %s ran %d times", name, k, e, n)
				}
			}
		}
	}
}

// lyingWorld reports success for worktree-add without doing it.
type lyingWorld struct{ *SlotWorld }

func (w lyingWorld) apply(ctx context.Context, s PlannedStep) error {
	if s.Step == StepWorktreeAdd {
		w.Effects = append(w.Effects, s.String())
		return nil
	}
	return w.SlotWorld.apply(ctx, s)
}

func TestReconcileNoProgressStops(t *testing.T) {
	w := newSlotWorld()
	w.set(perturbation{ResourceEnv, StateAbsent, ""})
	_, err := reconcileLoop(context.Background(), lyingWorld{w}, nil)
	var rerr *ReconcileError
	if !errors.As(err, &rerr) || rerr.Failure.Step.Step != StepWorktreeAdd || !errors.Is(err, errNoProgress) {
		t.Fatalf("err = %v, want no progress at worktree-add (never an empty plan read as converged)", err)
	}
}

func TestReconcileKeepOnFailureEndsWithAWarning(t *testing.T) {
	w := newSlotWorld()
	w.SourceKnown = false
	w.set(perturbation{DepResource(fakeDep), StateAbsent, ""})
	result, err := reconcileLoop(context.Background(), w, nil)
	if err != nil || len(result.Warnings) != 1 || result.Warnings[0].Step.Step != StepCompile {
		t.Fatalf("result %+v err %v, want one compile warning and a usable slot", result, err)
	}
	if n := len(w.Effects); n != 1 {
		t.Fatalf("effects %v: the failing compile must not be retried in the same run", w.Effects)
	}
}

// realGitScenarios are the perturbations the fixture can produce on real git,
// keyed like the derived domain. TestReconcileAgreesWithRealGit asserts that
// every perturbation of the domain is either here or named in
// notProducibleOnRealGit with its reason, so the list cannot silently shrink.
var realGitScenarios = map[string]func(t *testing.T, s *observedSlot){
	"env=absent/":             func(t *testing.T, s *observedSlot) { os.RemoveAll(s.layout.Env()) },
	"host=absent/":            func(t *testing.T, s *observedSlot) { os.RemoveAll(s.layout.Host()) },
	"setup=absent/":           func(t *testing.T, s *observedSlot) { os.Remove(SetupMarkerPath(s.admin(t))) },
	"setup=present/lock-held": func(t *testing.T, s *observedSlot) { holdLock(t, s.layout.SetupLock()) },
	"intent=present/":         func(t *testing.T, s *observedSlot) { write(t, s.layout.Intent(), "{}") },
	"deps=unknown/": func(t *testing.T, s *observedSlot) {
		write(t, filepath.Join(s.layout.Host(), "construct", "deps"), "substrate\n")
	},
	"dep:ariadne=absent/": func(t *testing.T, s *observedSlot) { os.RemoveAll(filepath.Join(s.layout.Env(), fakeDep)) },
	"dep:ariadne=broken/unreadable": func(t *testing.T, s *observedSlot) {
		os.Remove(filepath.Join(s.layout.Env(), fakeDep, ".git", "HEAD"))
	},
	"host=broken/unreadable": func(t *testing.T, s *observedSlot) {
		write(t, filepath.Join(s.layout.Host(), ".git"), "gitdir: /nonexistent/admin\n")
	},
	"setup=broken/": func(t *testing.T, s *observedSlot) {
		path := SetupMarkerPath(s.admin(t))
		raw, _ := os.ReadFile(path)
		os.WriteFile(path, []byte(strings.Replace(string(raw), `"slot":1`, `"slot":2`, 1)), 0o600)
	},
	"branch=broken/elsewhere": func(t *testing.T, s *observedSlot) {
		s.f.git(s.layout.Host(), "switch", "-q", "-c", "issue-work")
		s.f.git(s.f.Primary, "worktree", "add", "-q", filepath.Join(t.TempDir(), "elsewhere"), s.layout.RestingBranch())
	},
	"upstream=broken/conflict": func(t *testing.T, s *observedSlot) {
		s.f.git(s.f.Primary, "config", "branch."+s.layout.RestingBranch()+".merge", "refs/heads/other")
	},
}

// The SlotWorld counterparts of scenarios whose real-git break implies more
// than one perturbation.
var realGitScenarioExtras = map[string][]perturbation{
	"branch=broken/elsewhere": {{ResourceHost, StatePresent, ""}},
}

func TestReconcileAgreesWithRealGit(t *testing.T) {
	keys := make([]string, 0, len(realGitScenarios))
	for k := range realGitScenarios {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			s := newObservedSlot(t)
			os.MkdirAll(s.layout.Store(), 0o700)
			s.addDep(t, fakeDep)
			s.f.DepSources = map[string]bool{fakeDep: true}
			realGitScenarios[key](t, s)
			p := NewWorkspaceProvisioner(s.f)
			first, err := p.Reconcile(context.Background(), ReconcileRequest{Layout: s.layout, Agent: AgentNone})
			realClass := terminalClass(first, err)
			before := s.observe(t)
			second, err := p.Reconcile(context.Background(), ReconcileRequest{Layout: s.layout, Agent: AgentNone})
			if again := terminalClass(second, err); again != realClass {
				t.Errorf("real git: second run ended %s, first %s", again, realClass)
			}
			if realClass == "converged" && (len(second.Executed) != 0 || !s.observe(t).Equal(before)) {
				t.Errorf("real git: second run of a converged slot executed %v", second.Executed)
			}
			w := newSlotWorld()
			for _, p := range slotPerturbations() {
				if p.String() == key {
					w.set(p)
				}
			}
			for _, extra := range realGitScenarioExtras[key] {
				w.set(extra)
			}
			fake, err := reconcileLoop(context.Background(), w, nil)
			if fakeClass := terminalClass(fake, err); fakeClass != realClass {
				t.Errorf("SlotWorld ends %s, real git %s (executed %v)", fakeClass, realClass, first.Executed)
			}
		})
	}
}
