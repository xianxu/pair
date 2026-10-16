package couchcmd

import (
	"bytes"
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
	v := couchcore.ResourceView{Slots: []couchcore.ResourceSlotView{{Slot: bad, Reason: bad}}, Orphans: []couchcore.ResourceProcessView{{Name: bad, Reason: bad}}, Unattributed: []couchcore.ResourceProcessView{{Name: bad}}, Errors: []string{bad}}
	var b bytes.Buffer
	renderResources(&b, v, time.Now())
	out := b.String()
	if strings.ContainsAny(out, "\x1b\r") || strings.Contains(out, "\nforged") {
		t.Fatalf("terminal injection: %q", out)
	}
	for _, s := range []string{"unavailable", "Proven orphans", "Unattributed (unproven; report only)", "Coverage:"} {
		if !strings.Contains(out, s) {
			t.Fatalf("missing %q: %s", s, out)
		}
	}
}
