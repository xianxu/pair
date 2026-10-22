package couchmessage

import (
	"encoding/json"
	"reflect"
	"testing"
)

func reapSignalRequest(t *testing.T) Request {
	t.Helper()
	const wire = `{"Op":"reap-orphan-signal","Confirmed":true,"OrphanSignal":"TERM","OrphanTarget":{"key":{"PID":42,"Identity":"child-start"},"root":{"PID":40,"Identity":"old-start"},"slot":"my-repo:0","thread":{"repo_scope":"0123456789abcdef","tag":"1-my-repo-0"},"workspace":{"schema_version":2,"repo":"my-repo","repo_identity":"/fleet/my-repo/.git","primary_root":"/fleet/my-repo","fleet_root":"/fleet","environment_root":"/fleet","worktree_root":"/fleet/my-repo","kind":"primary","address":"my-repo:0","slot":0,"branch":"main","head":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","resting_branch":"main"}}}`
	var r Request
	if err := json.Unmarshal([]byte(wire), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestProvenOrphanRequestClosedShapes(t *testing.T) {
	for _, op := range []string{"reap-orphans", "reap-orphan-signal"} {
		t.Run(op, func(t *testing.T) {
			base := Request{Op: op, Confirmed: true}
			if op == "reap-orphan-signal" {
				base = reapSignalRequest(t)
			}
			for _, bound := range []bool{false, true} {
				r := base
				if bound {
					r.Scope, r.Tag, r.Session, r.Nonce = "scope", "tag", "session", "nonce"
				}
				if err := ValidateRequest(r); err != nil {
					t.Fatalf("bound=%v valid: %v", bound, err)
				}
			}
			for mask := 1; mask < 15; mask++ {
				r := base
				fields := []*string{&r.Scope, &r.Tag, &r.Session, &r.Nonce}
				for i, p := range fields {
					if mask&(1<<i) != 0 {
						*p = "credential"
					}
				}
				if ValidateRequest(r) == nil {
					t.Fatalf("partial credentials accepted: %04b", mask)
				}
			}
			missing := base
			missing.Confirmed = false
			if ValidateRequest(missing) == nil {
				t.Fatal("missing confirmation accepted")
			}
			typ := reflect.TypeOf(base)
			for i := 0; i < typ.NumField(); i++ {
				name := typ.Field(i).Name
				switch name {
				case "Op", "Confirmed", "Scope", "Tag", "Session", "Nonce":
					continue
				case "OrphanTarget", "OrphanSignal":
					if op == "reap-orphan-signal" {
						continue
					}
				}
				t.Run(name, func(t *testing.T) {
					r := base
					v := reflect.ValueOf(&r).Elem().Field(i)
					switch v.Kind() {
					case reflect.String:
						v.SetString("unexpected")
					case reflect.Bool:
						v.SetBool(true)
					case reflect.Int:
						v.SetInt(1)
					case reflect.Pointer:
						v.Set(reflect.New(v.Type().Elem()))
					default:
						t.Fatalf("unhandled field %s", name)
					}
					if ValidateRequest(r) == nil {
						t.Fatalf("accepted unexpected %s", name)
					}
				})
			}
		})
	}
}

func TestProvenOrphanSignalWitnessAndSignal(t *testing.T) {
	for _, signal := range []string{"TERM", "KILL"} {
		r := reapSignalRequest(t)
		r.OrphanSignal = signal
		if err := ValidateRequest(r); err != nil {
			t.Fatalf("%s: %v", signal, err)
		}
	}
	for _, signal := range []string{"", "INT", "STOP", "CONT", "15", "-9", "term"} {
		r := reapSignalRequest(t)
		r.OrphanSignal = signal
		if ValidateRequest(r) == nil {
			t.Fatalf("accepted signal %q", signal)
		}
	}
	for name, mutate := range map[string]func(*Request){
		"nil":       func(r *Request) { r.OrphanTarget = nil },
		"pid":       func(r *Request) { r.OrphanTarget.Key.PID = 0 },
		"identity":  func(r *Request) { r.OrphanTarget.Key.Identity = "" },
		"root":      func(r *Request) { r.OrphanTarget.Root.Identity = "" },
		"slot":      func(r *Request) { r.OrphanTarget.Slot = "other:0" },
		"thread":    func(r *Request) { r.OrphanTarget.Thread.RepoScope = "" },
		"workspace": func(r *Request) { r.OrphanTarget.Workspace.RepoIdentity = "relative" },
	} {
		t.Run(name, func(t *testing.T) {
			r := reapSignalRequest(t)
			mutate(&r)
			if ValidateRequest(r) == nil {
				t.Fatal("accepted malformed witness")
			}
		})
	}
	// New fields cannot leak into older operation shapes that use field-by-field
	// validation rather than whole-request comparison.
	for _, op := range []string{"resources", "refusals", "broadcast-status", "actors", "status", "send", "register", "operator-submit", "session-write", "submission-status", "tail", "notify", "move-slot", "resume", "interrupt"} {
		for _, field := range []string{"target", "signal"} {
			r := Request{Op: op, Scope: "scope", Tag: "tag", Session: "session", Nonce: "nonce"}
			if field == "target" {
				r.OrphanTarget = reapSignalRequest(t).OrphanTarget
			} else {
				r.OrphanSignal = "TERM"
			}
			if ValidateRequest(r) == nil {
				t.Fatalf("%s accepted orphan %s", op, field)
			}
		}
	}
}

func TestProvenOrphanSignalResultWire(t *testing.T) {
	target := *reapSignalRequest(t).OrphanTarget
	want := Response{OrphanSignal: &ProvenOrphanSignalResult{Target: target, Status: "sent", Detail: "TERM delivered"}}
	wire, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got Response
	if err = json.Unmarshal(wire, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip changed witness: got %+v want %+v", got, want)
	}
}
