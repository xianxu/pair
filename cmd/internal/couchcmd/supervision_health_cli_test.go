package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func TestSupervisionHealthResourcesPublished(t *testing.T) {
	log := testFactLog(t, &factSink{})
	service := &messageService{}
	service.facts.Store(log)
	sampler := &resourceSampler{}
	sampler.cached.Store(&couchcore.ResourceView{})
	service.resources.Store(sampler)
	raw, err := json.Marshal(service.handle(context.Background(), couchmessage.Request{Op: "resources"}).Resources)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["supervision_log"]; !ok {
		t.Fatalf("resources omitted live supervision logger health: %s", raw)
	}
}

func supervisionResourceCLI(t *testing.T, service *messageService) func(bool) []byte {
	t.Helper()
	dir, err := os.MkdirTemp("", "s736-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket, err := couchmessage.SocketPath(dir, "broker")
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var req couchmessage.Request
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, err
		}
		return json.Marshal(service.handle(ctx, req))
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return func(asJSON bool) []byte {
		t.Helper()
		var out, stderr bytes.Buffer
		call := func(ctx context.Context, path string, req any, response any) error {
			return couchmessage.Call(ctx, path, req, response)
		}
		if code := runResourcesCLI(cliInvocation{jsonOutput: asJSON}, testRT{dir: dir}, &out, &stderr, call); code != 0 {
			t.Fatalf("resources CLI exit %d: %s", code, &stderr)
		}
		return out.Bytes()
	}
}

func TestSupervisionHealthFreshThroughCLIWithBlockedSink(t *testing.T) {
	sink := &factSink{block: make(chan struct{}), entered: make(chan struct{})}
	log := testFactLog(t, sink)
	var release sync.Once
	unblock := func() { release.Do(func() { close(sink.block) }) }
	defer unblock()
	service := &messageService{}
	service.facts.Store(log)
	sampler := &resourceSampler{}
	cached := couchcore.ResourceView{SampledAt: time.Unix(100, 0)}
	sampler.cached.Store(&cached)
	service.resources.Store(sampler)
	query := supervisionResourceCLI(t, service)
	log.offer(supervisionFact{Event: "snapshot"})
	select {
	case <-sink.entered:
	case <-time.After(time.Second):
		t.Fatal("sink did not enter held write")
	}
	// Both real socket requests complete while the sink is still held. Their
	// request deadline makes a blocked health read fail without releasing it.
	for _, wantOffered := range []uint64{1, 2} {
		if wantOffered == 2 {
			log.offer(supervisionFact{Event: "snapshot"})
		}
		var got couchcore.ResourceView
		if err := json.Unmarshal(query(true), &got); err != nil {
			t.Fatal(err)
		}
		h := got.SupervisionLog
		if h == nil || h.State != "working" || h.Offered != wantOffered || h.Written != 0 || !h.LastWriteAt.IsZero() {
			t.Fatalf("held writer health: %+v", h)
		}
		if !got.SampledAt.Equal(cached.SampledAt) || cached.SupervisionLog != nil || cached.HostedIngest != nil {
			t.Fatal("request health changed the cached sample")
		}
		text := string(query(false))
		if !strings.Contains(text, "Supervision log:") || !strings.Contains(text, "working") || !strings.Contains(text, fmt.Sprint(wantOffered)) {
			t.Fatalf("text omitted fresh logger health: %s", text)
		}
	}
	unblock()
	supervisionEventually(t, "both accepted facts not published", func() bool { return log.health().Written == 2 })
	var got couchcore.ResourceView
	if err := json.Unmarshal(query(true), &got); err != nil {
		t.Fatal(err)
	}
	if h := got.SupervisionLog; h == nil || h.Written != 2 || h.LastWriteAt.IsZero() || h.Dropped != 0 {
		t.Fatalf("completed writer health: %+v", h)
	}
}

func TestSupervisionHealthAbsentAndStrictLegacyResourceDecoder(t *testing.T) {
	// Derive the exact preceding ResourceView shape by removing only the
	// additive field. This still uses a real typed strict JSON decoder.
	current := reflect.TypeOf(couchcore.ResourceView{})
	var fields []reflect.StructField
	for i := 0; i < current.NumField(); i++ {
		if field := current.Field(i); field.Name != "SupervisionLog" {
			fields = append(fields, field)
		}
	}
	legacy := reflect.StructOf(fields)
	decodeLegacy := func(raw []byte) error {
		return strictjson.Decode(raw, reflect.New(legacy).Interface())
	}
	oldWire, err := json.Marshal(reflect.New(legacy).Interface())
	if err != nil {
		t.Fatal(err)
	}
	var newReader couchcore.ResourceView
	if err := strictjson.Decode(oldWire, &newReader); err != nil || newReader.SupervisionLog != nil {
		t.Fatalf("new reader of old server: %+v %v", newReader, err)
	}
	if err := decodeLegacy(oldWire); err != nil {
		t.Fatalf("legacy decoder rejected its own shape: %v", err)
	}
	newWire, err := json.Marshal(couchcore.ResourceView{SupervisionLog: &couchcore.SupervisionLogHealth{State: "working"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := decodeLegacy(newWire); err == nil || !strings.Contains(err.Error(), `unknown field "supervision_log"`) {
		t.Fatalf("strict legacy decoder did not expose additive-field refusal: %v", err)
	}
	var out bytes.Buffer
	renderResources(&out, newReader, time.Now())
	if !strings.Contains(out.String(), "Supervision log: health unavailable") || !strings.Contains(out.String(), "use the rebuilt CLI") || !strings.Contains(out.String(), "restart Couch") {
		t.Fatalf("missing owner/server was rendered as healthy: %s", &out)
	}
}

func TestSupervisionHealthProjectionAndCloneRetainBoundedHealth(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	health := couchcore.SupervisionLogHealth{State: "failed-write", Offered: 10, Written: 7, Dropped: 3, LastWriteAt: at, ErrorOperation: "write", LastError: "diagnostic log busy", LastErrorAt: at}
	v := couchcore.ResourceView{SupervisionLog: &health, Errors: []string{strings.Repeat("<", couchmessage.MaxFrameBytes)}}
	for _, got := range []couchcore.ResourceView{projectResourceView(v), cloneResourceView(v)} {
		if got.SupervisionLog == nil || *got.SupervisionLog != health {
			t.Fatalf("health lost during projection/clone: %+v", got.SupervisionLog)
		}
		got.SupervisionLog.State = "changed"
		if v.SupervisionLog.State != "failed-write" {
			t.Fatal("health pointer aliases source")
		}
	}
	projected := projectResourceView(v)
	raw, err := json.Marshal(couchmessage.Response{Code: "ok", Resources: &projected})
	if err != nil || len(raw) > couchmessage.MaxFrameBytes {
		t.Fatalf("health projection exceeded frame: %d %v", len(raw), err)
	}
	for _, key := range []string{"supervision_log", "last_write_at", "error_operation", "last_error", "last_error_at"} {
		if !bytes.Contains(raw, []byte(`"`+key+`"`)) {
			t.Fatalf("missing wire field %q: %s", key, raw)
		}
	}
}
