package webcast

import (
	"bufio"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// TestTranslateSpikeCapture replays events captured from Chrome 154 in the
// #454 spike (ids shortened) and pins what each one means.
func TestTranslateSpikeCapture(t *testing.T) {
	f, err := os.Open("testdata/spike_events.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	targetOf := func(s string) (string, bool) {
		if s == "5AF3" {
			return "0F72", true
		}
		return "", false
	}
	want := [][]any{
		{TargetCreated{ID: "0F72", URL: "about:blank"}},
		nil, // browser_ui is not a tab
		nil, // nor is a service worker
		{pageAttached{TargetID: "0F72", SessionID: "5AF3"}},
		nil, // the page's own world
		{contextCreated{SessionID: "5AF3", ContextID: 2}},
		nil, // our world in a subframe
		{PageReport{ID: "0F72", Visible: true}},
		{toggleRequested{SessionID: "5AF3"}},
		{helloRequested{SessionID: "5AF3", ContextID: 4}},
		{PageTitle{ID: "0F72", Title: "hi"}},
		nil, // malformed payload
		nil, // someone else's binding
		{TargetChanged{ID: "0F72", URL: "data:text/html,<title>hi</title><p>hello", Title: "hi"}},
		{contextDestroyed{SessionID: "5AF3", ContextID: 2}},
		{contextsCleared{SessionID: "5AF3"}},
		{pageDetached{SessionID: "5AF3"}},
		{TargetDestroyed{ID: "0F72"}},
		{DocumentChanged{ID: "0F72"}},
		nil, // a subframe's navigation
		nil, // unhandled method
	}
	sc := bufio.NewScanner(f)
	i := 0
	for ; sc.Scan(); i++ {
		var ev struct {
			Method    string          `json:"method"`
			Params    json.RawMessage `json:"params"`
			SessionID string          `json:"sessionId"`
		}
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			t.Fatalf("line %d: %v", i+1, err)
		}
		got := translate(ev.SessionID, ev.Method, ev.Params, targetOf)
		if i >= len(want) {
			t.Fatalf("line %d: no expectation", i+1)
		}
		if !reflect.DeepEqual(got, want[i]) {
			t.Errorf("line %d %s: got %#v, want %#v", i+1, ev.Method, got, want[i])
		}
	}
	if i != len(want) {
		t.Fatalf("%d lines, %d expectations", i, len(want))
	}
}

func TestTranslateBindingFromUnknownSession(t *testing.T) {
	none := func(string) (string, bool) { return "", false }
	params := json.RawMessage(`{"name":"__couchcast","payload":"{\"t\":\"vis\",\"visible\":true}"}`)
	if got := translate("nobody", "Runtime.bindingCalled", params, none); got != nil {
		t.Fatalf("got %#v", got)
	}
}

func FuzzTranslate(f *testing.F) {
	f.Add("s", "Runtime.bindingCalled", []byte(`{"name":"__couchcast","payload":"{\"t\":\"vis\"}"}`))
	f.Add("s", "Target.targetCreated", []byte(`{"targetInfo":{"targetId":"x","type":"page"}}`))
	f.Add("s", "Runtime.executionContextCreated", []byte(`{"context":{"name":"couchcast","auxData":{"frameId":"t"}}}`))
	f.Fuzz(func(t *testing.T, session, method string, params []byte) {
		translate(session, method, params, func(string) (string, bool) { return "t", true })
	})
}
