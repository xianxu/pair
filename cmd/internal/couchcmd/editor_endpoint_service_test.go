package couchcmd

import (
	"context"
	"encoding/json"
	"os"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"testing"
)

func TestEditorEndpointServiceRefusesNamespaceAndMalformedBeforeAuthority(t *testing.T) {
	s := &editorEndpointService{namespace: "/physical/store"}
	for _, raw := range []string{
		`{"Operation":"lookup","Namespace":"/other","DocumentID":"0123456789abcdef0123456789abcdef"}`,
		`{"Operation":"lookup","Operation":"register"}`,
		`{"Operation":"execute","Namespace":"/physical/store"}`,
		`{"operation":"lookup","Namespace":"/physical/store","DocumentID":"0123456789abcdef0123456789abcdef"}`,
	} {
		out, err := s.handle(context.Background(), []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		var reply struct {
			Code           string
			Version        int
			EditorEndpoint any
		}
		if err = json.Unmarshal(out, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Code != "invalid" || reply.Version != 1 || reply.EditorEndpoint != nil {
			t.Fatalf("accepted malformed/foreign request: %s", out)
		}
	}
}

// Use the real producer, its private Unix transport and the CLI reader together:
// a server-only outcome code must never turn confirmed absence into uncertainty.
func TestEditorEndpointServiceEmptyLookupThroughCLI(t *testing.T) {
	rt, request := endpointTransportSetup(t)
	if err := os.Chmod(request.Namespace, 0700); err != nil {
		t.Fatal(err)
	}
	service, err := startEditorEndpointService(context.Background(), request.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)
	code, reply, stderr := endpointTransportRun(t, cliInvocation{args: []string{"lookup", request.Namespace, request.DocumentID}}, rt, "", couchmessage.Call)
	if code != 0 || reply.Code != "ok" || reply.EditorEndpoint != nil || reply.Error != "" || stderr != "" {
		t.Fatalf("confirmed empty lookup: exit=%d reply=%+v stderr=%q", code, reply, stderr)
	}
}

func TestEditorEndpointServiceDamagedStateRefusesThroughCLI(t *testing.T) {
	rt, request := endpointTransportSetup(t)
	if err := os.Chmod(request.Namespace, 0700); err != nil {
		t.Fatal(err)
	}
	paths, err := artifactpath.EditorEndpoints(request.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(paths.State(), []byte("damaged authority"), 0600); err != nil {
		t.Fatal(err)
	}
	service, err := startEditorEndpointService(context.Background(), request.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)
	code, reply, _ := endpointTransportRun(t, cliInvocation{args: []string{"lookup", request.Namespace, request.DocumentID}}, rt, "", couchmessage.Call)
	if code != 3 || reply.Code != "unavailable" || reply.EditorEndpoint != nil {
		t.Fatalf("damaged authority: exit=%d reply=%+v", code, reply)
	}
	raw, err := os.ReadFile(paths.State())
	if err != nil || string(raw) != "damaged authority" {
		t.Fatalf("authority changed: %q %v", raw, err)
	}
}
