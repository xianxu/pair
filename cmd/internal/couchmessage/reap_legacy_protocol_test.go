package couchmessage

import (
	"encoding/json"
	"reflect"
	"testing"
)

func legacySignalRequest(t *testing.T) Request {
	t.Helper()
	var r Request
	if err := json.Unmarshal([]byte(`{"Op":"reap-legacy-signal","Confirmed":true,"OrphanSignal":"TERM","LegacyTarget":{"family":"left-running","root":{"PID":10,"Identity":"root"},"key":{"PID":11,"Identity":"child"}}}`), &r); err != nil {
		t.Fatal(err)
	}
	return r
}
func TestLegacyOrphanRequestClosedShape(t *testing.T) {
	base := legacySignalRequest(t)
	for _, bound := range []bool{false, true} {
		r := base
		if bound {
			r.Scope, r.Tag, r.Session, r.Nonce = "scope", "tag", "session", "nonce"
		}
		if err := ValidateRequest(r); err != nil {
			t.Fatalf("bound%v: %v", bound, err)
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
			t.Fatalf("partial credentials %04b", mask)
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
		case "Op", "Confirmed", "Scope", "Tag", "Session", "Nonce", "LegacyTarget", "OrphanSignal":
			continue
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
			case reflect.Struct:
				if name != "AgentArgs" {
					t.Fatalf("unhandled %s", name)
				}
				arguments, err := NewSwitchArguments(nil)
				if err != nil {
					t.Fatal(err)
				}
				v.Set(reflect.ValueOf(arguments))
			default:
				t.Fatalf("unhandled %s", name)
			}
			if ValidateRequest(r) == nil {
				t.Fatalf("accepted unexpected %s", name)
			}
		})
	}
}

func TestLegacyOrphanSignalWitnessAndSignal(t *testing.T) {
	for _, family := range []string{"orphan-nvim", "left-running"} {
		for _, signal := range []string{"TERM", "KILL"} {
			r := legacySignalRequest(t)
			r.LegacyTarget.Family = family
			r.OrphanSignal = signal
			if err := ValidateRequest(r); err != nil {
				t.Fatalf("%s/%s: %v", family, signal, err)
			}
		}
	}
	for _, signal := range []string{"", "INT", "STOP", "CONT", "15", "-9", "term"} {
		r := legacySignalRequest(t)
		r.OrphanSignal = signal
		if ValidateRequest(r) == nil {
			t.Fatalf("accepted %q", signal)
		}
	}
	for name, mutate := range map[string]func(*Request){
		"nil": func(r *Request) { r.LegacyTarget = nil }, "family": func(r *Request) { r.LegacyTarget.Family = "other" }, "root-pid": func(r *Request) { r.LegacyTarget.Root.PID = 0 }, "root-identity": func(r *Request) { r.LegacyTarget.Root.Identity = "" }, "target-pid": func(r *Request) { r.LegacyTarget.Key.PID = -1 }, "target-identity": func(r *Request) { r.LegacyTarget.Key.Identity = "" },
	} {
		t.Run(name, func(t *testing.T) {
			r := legacySignalRequest(t)
			mutate(&r)
			if ValidateRequest(r) == nil {
				t.Fatal("accepted malformed legacy witness")
			}
		})
	}
	for _, op := range []string{"resources", "reap-orphans", "reap-orphan-signal", "actors", "broadcast-status", "session-write", "notify", "send", "tail", "register", "move-slot"} {
		r := legacySignalRequest(t)
		r.Op = op
		if ValidateRequest(r) == nil {
			t.Fatalf("%s accepted legacy target", op)
		}
	}
}

func TestLegacyOrphanSignalResultWire(t *testing.T) {
	target := *legacySignalRequest(t).LegacyTarget
	want := Response{LegacySignal: &LegacyOrphanSignalResult{Target: target, Status: "sent", Detail: "TERM delivered"}}
	wire, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got Response
	if err = json.Unmarshal(wire, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roundtrip changed witness: %+v", got)
	}
}
