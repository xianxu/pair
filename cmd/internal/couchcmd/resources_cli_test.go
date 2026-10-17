package couchcmd

import (
	"bytes"
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestResourceQueryShape(t *testing.T) {
	for _, args := range [][]string{{"--resources"}, {"--resources", "--json"}} {
		inv, err := ParseCLI(args, nil)
		if err != nil || inv.messageOp != "resources" || inv.jsonOutput != (len(args) == 2) {
			t.Fatalf("%v: %+v %v", args, inv, err)
		}
	}
	for _, args := range [][]string{{"--resources", "pair:1"}, {"--resources", "--json", "--json"}, {"--resources", "--send-to", "hub"}} {
		if _, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if err := couchmessage.ValidateRequest(couchmessage.Request{Op: "resources"}); err != nil {
		t.Fatal(err)
	}
	if err := couchmessage.ValidateRequest(couchmessage.Request{Op: "resources", Body: "inject"}); err == nil {
		t.Fatal("accepted extra field")
	}
}

func TestResourceQueryRejectsEveryNonOperationField(t *testing.T) {
	typ := reflect.TypeOf(couchmessage.Request{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "Op" {
			continue
		}
		t.Run(field.Name, func(t *testing.T) {
			r := couchmessage.Request{Op: "resources"}
			v := reflect.ValueOf(&r).Elem().Field(i)
			switch v.Kind() {
			case reflect.String:
				v.SetString("extra")
			case reflect.Bool:
				v.SetBool(true)
			case reflect.Int:
				v.SetInt(1)
			case reflect.Pointer:
				v.Set(reflect.New(v.Type().Elem()))
			default:
				t.Fatalf("unhandled request field %s", field.Name)
			}
			if err := couchmessage.ValidateRequest(r); err == nil {
				t.Fatalf("accepted %s", field.Name)
			}
		})
	}
}
func TestResourceRenderSanitizesLabelsAndUnknown(t *testing.T) {
	bad := "unsafe\x1b[2J\r\nforged"
	v := couchcore.ResourceView{Slots: []couchcore.ResourceSlotView{{Slot: bad, Reason: bad}}, Orphans: []couchcore.ResourceProcessView{{Name: bad, Reason: bad}}, Unattributed: []couchcore.ResourceProcessView{{Name: bad}}, OrphanNvim: []couchcore.OrphanNvim{{Cwd: bad, Root: bad, Reason: bad}}, Errors: []string{bad}}
	var b bytes.Buffer
	renderResources(&b, v, time.Now())
	out := b.String()
	if strings.ContainsAny(out, "\x1b\r") || strings.Contains(out, "\nforged") {
		t.Fatalf("terminal injection: %q", out)
	}
	for _, s := range []string{"unavailable", "Proven orphans", "Unattributed (unproven; report only)", orphanNvimSection, "Coverage:"} {
		if !strings.Contains(out, s) {
			t.Fatalf("missing %q: %s", s, out)
		}
	}
}

// An absent section reads as "not checked", so an empty sweep still says it
// looked, and a row carries everything an operator needs to decide (#583).
func TestResourceRenderReportsTheOrphanNvimSweep(t *testing.T) {
	at := time.Unix(1760100000, 0)
	var b bytes.Buffer
	renderResources(&b, couchcore.ResourceView{SampledAt: at}, at)
	if !strings.Contains(b.String(), orphanNvimSection+":\n  none observed") {
		t.Fatalf("an empty sweep did not say it looked: %s", b.String())
	}
	rss := uint64(22 << 30)
	v := couchcore.ResourceView{SampledAt: at, OrphanNvim: []couchcore.OrphanNvim{{
		Key:      couchcore.ResourceProcessKey{PID: 4242, Identity: "1760000000.5"},
		Age:      13*24*time.Hour + 5*time.Minute + 20*time.Second,
		Cwd:      "/fleet/worktree/pair-slot1/pair/cmd",
		Root:     "/fleet/worktree/pair-slot1/pair",
		RSSBytes: &rss,
		Reason:   "unattributed embedded nvim",
	}}, Cost: couchcore.ResourceCost{NativeCalls: 3, CwdCalls: 3}}
	b.Reset()
	renderResources(&b, v, at)
	out := b.String()
	for _, want := range []string{
		orphanNvimSection + ":\n  pid=4242 age=312h5m0s RSS=23622320128 cwd=/fleet/worktree/pair-slot1/pair/cmd root=/fleet/worktree/pair-slot1/pair unattributed embedded nvim\n",
		// The health line names every native call kind, so the sweep's new probe
		// is attributable from the report alone.
		"(table 0, usage 0, command 0, cwd 3)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	encoded, err := json.Marshal(v)
	if err != nil || !strings.Contains(string(encoded), `"orphan_nvim":[{"key":{"pid":4242`) {
		t.Fatalf("--json must carry the same rows: %s %v", encoded, err)
	}
}
