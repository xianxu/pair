package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"os"
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

// A resources request reads hosted counters now, even when its process sample
// is old. Exercise the same Console, Child, socket and CLI paths as production.
func TestResourcesHostedFreshThroughCLI(t *testing.T) {
	console := couchtty.New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	defer console.Stop()
	child := ptychild.NewFakeChild([]byte("first"))
	defer child.Close()
	console.Attach("hosted", "pair", child)
	service := &messageService{}
	hosted := console.HostedIngestSnapshot
	service.hostedIngest.Store(&hosted)
	sampler := &resourceSampler{}
	cached := couchcore.ResourceView{SampledAt: time.Unix(100, 0)}
	sampler.cached.Store(&cached)
	service.resources.Store(sampler)
	dir, err := os.MkdirTemp("/tmp", "r622-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket, err := couchmessage.SocketPath(dir, "broker")
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, b []byte) ([]byte, error) {
		var req couchmessage.Request
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, err
		}
		return json.Marshal(service.handle(ctx, req))
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	call := func(ctx context.Context, path string, req any, response any) error {
		return couchmessage.Call(ctx, path, req, response)
	}
	var previous uint64
	for _, chunk := range []string{"", "second", "third"} {
		child.Feed([]byte(chunk))
		var out, stderr bytes.Buffer
		if code := runResourcesCLI(cliInvocation{jsonOutput: true}, testRT{dir: dir}, &out, &stderr, call); code != 0 {
			t.Fatalf("CLI %d: %s", code, &stderr)
		}
		var got couchcore.ResourceView
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		h := got.HostedIngest
		if h == nil || h.SampledAt.Before(time.Now().Add(-time.Minute)) || h.ServerPID != os.Getpid() || !h.ServerStartedAt.Equal(processStartedAt) || len(h.Clients) != 1 {
			t.Fatalf("hosted identity/freshness: %+v", h)
		}
		if !got.SampledAt.Equal(cached.SampledAt) || cached.HostedIngest != nil {
			t.Fatal("cached process sample changed")
		}
		want := previous + uint64(len(chunk))
		if previous == 0 {
			want += 5
		}
		row := h.Clients[0]
		if row.IngestBytes != want || row.EndpointID != child.Endpoint().ID() || row.Thread.Tag != "hosted" {
			t.Fatalf("counter: %+v want %d", row, want)
		}
		previous = want
		out.Reset()
		if code := runResourcesCLI(cliInvocation{}, testRT{dir: dir}, &out, &stderr, call); code != 0 {
			t.Fatalf("text: %s", &stderr)
		}
		if !strings.Contains(out.String(), "cumulative ingest bytes") || !strings.Contains(out.String(), fmt.Sprint(want)) {
			t.Fatalf("text: %s", &out)
		}
	}
}

func TestResourceHostedAbsentEmptyAndSanitized(t *testing.T) {
	var out bytes.Buffer
	renderResources(&out, couchcore.ResourceView{}, time.Now())
	if !strings.Contains(out.String(), "Hosted ingest: unavailable") {
		t.Fatal(out.String())
	}
	out.Reset()
	renderResources(&out, couchcore.ResourceView{HostedIngest: &couchcore.HostedIngestSnapshot{}}, time.Now())
	if !strings.Contains(out.String(), "0 clients") {
		t.Fatal(out.String())
	}
	out.Reset()
	renderResources(&out, couchcore.ResourceView{HostedIngest: &couchcore.HostedIngestSnapshot{Clients: []couchcore.HostedIngestClient{{EndpointID: "evil\x1b[2J\nforged", Thread: couchcore.ThreadAddress{RepoScope: "bad\r\nforged"}}}}}, time.Now())
	if strings.ContainsAny(out.String(), "\x1b\r") || strings.Contains(out.String(), "\nforged") {
		t.Fatal(out.String())
	}
}
