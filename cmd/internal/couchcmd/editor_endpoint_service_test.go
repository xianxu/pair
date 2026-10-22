package couchcmd

import (
	"context"
	"encoding/json"
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
