package editorendpoint

import (
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func req(inc, doc, attach, mode string) RegisterRequest {
	return RegisterRequest{Version: 1, Namespace: "/project", DocumentID: doc, EditorIncarnation: inc, AttachID: attach, PID: 42, Socket: "/socket", Buffer: 1, Mode: mode}
}

const docA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const incA = "11111111111111111111111111111111"
const incB = "22222222222222222222222222222222"

func reg(t *testing.T, s EndpointState, r RegisterRequest) (EndpointState, EndpointReply) {
	t.Helper()
	n, p, e := ApplyEndpointEvent(s, EndpointEvent{Register: &r, ProcessIdentity: "birth", BindingGeneration: fmt.Sprintf("%032x", s.Revision+1), File: "/file"})
	if e != nil {
		t.Fatal(e)
	}
	return n, p
}
func TestRegisterMonotoneAndNoResurrection(t *testing.T) {
	s, p := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	if !p.Changed {
		t.Fatal("initial registration unchanged")
	}
	original := p.EditorEndpoint.BindingGeneration
	s, p = reg(t, s, req(incA, docA, "1", "desktop"))
	if p.Changed || p.EditorEndpoint.BindingGeneration != original {
		t.Fatal("retry changed binding")
	}
	s, _ = reg(t, s, req(incB, docA, "1", "desktop"))
	_, p = reg(t, s, req(incA, docA, "1", "desktop"))
	if p.Code == "ok" {
		t.Fatal("resurrected displaced request")
	}
	s, p = reg(t, s, req(incA, docA, "2", "headless"))
	if p.Code == "ok" {
		t.Fatal("headless displaced desktop")
	}
	s, p = reg(t, s, req(incA, docA, "3", "desktop"))
	if p.Code != "ok" || !p.Changed {
		t.Fatalf("higher fresh registration: %+v", p)
	}
	if len(s.Records) != 2 {
		t.Fatalf("retained attempts rather than pairs: %d", len(s.Records))
	}
}
func TestUnregisterFencedAtomic(t *testing.T) {
	s, p := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	old := *p.EditorEndpoint
	s, p = reg(t, s, req(incA, docA, "2", "desktop"))
	u := UnregisterRequest{Version: 1, Namespace: "/project", EditorIncarnation: incA, PID: 42, Entries: []UnregisterEntry{{DocumentID: docA, AttachID: old.AttachID, BindingGeneration: old.BindingGeneration}}}
	n, reply, e := ApplyEndpointEvent(s, EndpointEvent{Unregister: &u})
	if e != nil || reply.Changed || n.Revision != s.Revision {
		t.Fatalf("stale unregister mutated: %+v %v", reply, e)
	}
	u.Entries = append(u.Entries, u.Entries[0])
	if _, _, e = ApplyEndpointEvent(s, EndpointEvent{Unregister: &u}); e == nil {
		t.Fatal("duplicate batch accepted")
	}
}

// This oracle tracks only attempt identities and the selected owner, without
// inspecting implementation statuses or deriving expectations from its records.
func TestSeededSequenceIndependentSelectionOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(758))
	s := EndpointState{}
	high := map[string]int{}
	selected := map[string]string{}
	terminal := map[string]bool{}
	for step := 0; step < 1200; step++ {
		inc := fmt.Sprintf("%032x", 1+rng.Intn(8))
		doc := fmt.Sprintf("%032x", 100+rng.Intn(16))
		key := inc + doc
		id := 1 + rng.Intn(24)
		mode := "desktop"
		if rng.Intn(3) == 0 {
			mode = "headless"
		}
		r := req(inc, doc, strconv.Itoa(id), mode)
		previous := selected[doc]
		expected := "stale"
		if id > high[key] {
			high[key] = id
			if previous != "" && strings.HasSuffix(previous, "desktop") && mode == "headless" {
				expected = "desktop-active"
				terminal[key] = true
			} else {
				expected = "ok"
				if previous != "" {
					terminal[strings.Split(previous, ":")[0]] = true
				}
				terminal[key] = false
				selected[doc] = key + ":" + strconv.Itoa(id) + ":" + mode
			}
		} else if id == high[key] && terminal[key] {
			expected = "superseded"
		} else if id == high[key] && !terminal[key] && previous == key+":"+strconv.Itoa(id)+":"+mode {
			expected = "ok"
		} else if id == high[key] && !terminal[key] && strings.HasPrefix(previous, key+":") {
			expected = "conflict"
		}
		var p EndpointReply
		s, p = reg(t, s, r)
		if p.Code != expected {
			t.Fatalf("step %d got %s want %s", step, p.Code, expected)
		}
		got := map[string]string{}
		for _, x := range s.Records {
			if cb := CurrentBinding(x); cb != nil {
				b := *cb
				if got[b.DocumentID] != "" {
					t.Fatal("multiple current endpoints")
				}
				got[b.DocumentID] = b.EditorIncarnation + b.DocumentID + ":" + b.AttachID + ":" + b.Mode
			}
		}
		if !reflect.DeepEqual(got, selected) {
			t.Fatalf("step %d selection differs", step)
		}
	}
}
func TestPairRetentionPlateauAndProcessBounds(t *testing.T) {
	for _, count := range []int{8, 16} {
		s := EndpointState{}
		for round := 1; round <= 32; round++ {
			for i := 0; i < count; i++ {
				s, _ = reg(t, s, req(incA, fmt.Sprintf("%032x", i+1), strconv.Itoa(round), "desktop"))
			}
			if len(s.Records) != count {
				t.Fatalf("retention grew with attempts: %d", len(s.Records))
			}
		}
	}
	s := EndpointState{}
	for i := 0; i < 256; i++ {
		s, _ = reg(t, s, req(incA, fmt.Sprintf("%032x", i+1), "1", "desktop"))
	}
	before := s.Revision
	n, p := reg(t, s, req(incA, fmt.Sprintf("%032x", 257), "1", "desktop"))
	if p.Code != "capacity" || n.Revision != before {
		t.Fatal("active bound failed")
	}
}
func TestRetiredRetryCollectionAndInputImmutability(t *testing.T) {
	s, p := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	b := *p.EditorEndpoint
	retired, p, e := ApplyEndpointEvent(s, EndpointEvent{Retire: &b})
	if e != nil || !p.Changed || s.Records[0].Status != "current" {
		t.Fatal("retire mutated input or failed")
	}
	_, p = reg(t, retired, req(incA, docA, "1", "desktop"))
	if p.Code != "superseded" {
		t.Fatal("retired attempt revived")
	}
	wrong := ProcessEvidence{EditorIncarnation: incA, PID: 42, ProcessIdentity: "different-birth"}
	n, p, e := ApplyEndpointEvent(retired, EndpointEvent{Collect: &wrong})
	if e != nil || p.Changed || len(n.Records) != 1 {
		t.Fatal("wrong identity collected")
	}
	wrong.ProcessIdentity = "birth"
	n, p, e = ApplyEndpointEvent(retired, EndpointEvent{Collect: &wrong})
	if e != nil || !p.Changed || len(n.Records) != 0 {
		t.Fatal("exact identity not collected")
	}
}

func TestSamePairHeadlessHighWaterKeepsDesktopAuthority(t *testing.T) {
	s, p := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	desktop := *p.EditorEndpoint
	s, p = reg(t, s, req(incA, docA, "2", "headless"))
	if p.Code != "desktop-active" || !p.Changed {
		t.Fatal("headless was not remembered as refused")
	}
	if current := CurrentBinding(s.Records[0]); current == nil || !exact(*current, desktop) {
		t.Fatal("refused attempt displaced admitted desktop")
	}
	raw, e := EncodeEndpointState(s)
	if e != nil {
		t.Fatal(e)
	}
	s, e = DecodeEndpointState(raw)
	if e != nil {
		t.Fatal(e)
	}
	_, p = reg(t, s, req(incA, docA, "1", "desktop"))
	if p.Code != "stale" || p.EditorEndpoint != nil {
		t.Fatal("late old retry bypassed terminal high-water")
	}
	_, p = reg(t, s, req(incA, docA, "2", "headless"))
	if p.Code != "superseded" || p.Changed {
		t.Fatal("terminal headless retry reacquired")
	}
	s, p, e = ApplyEndpointEvent(s, EndpointEvent{Retire: &desktop})
	if e != nil || !p.Changed || CurrentBinding(s.Records[0]) != nil {
		t.Fatal("selected desktop retirement failed")
	}
	_, p = reg(t, s, req(incA, docA, "2", "headless"))
	if p.Code != "superseded" {
		t.Fatal("refused headless resurrected after retirement")
	}
	s, p = reg(t, s, req(incA, docA, "3", "desktop"))
	if p.Code != "ok" || CurrentBinding(s.Records[0]) == nil || s.Records[0].Current != nil {
		t.Fatal("new desktop did not replace high-water")
	}
}
func TestRequestBoundsCanonicalAttachAndPaths(t *testing.T) {
	good := req(incA, docA, "1", "desktop")
	for _, badID := range []string{"0", "01", "+1", "-1", "18446744073709551616", "1.0", " 1"} {
		r := good
		r.AttachID = badID
		if ValidateEndpointRequest(r) == nil {
			t.Fatalf("accepted %q", badID)
		}
	}
	for _, path := range []string{"relative", "/project/../project", "/project/", "/project\x00x"} {
		r := good
		r.Namespace = path
		if ValidateEndpointRequest(r) == nil {
			t.Fatalf("accepted path %q", path)
		}
	}
	good.AttachID = "18446744073709551615"
	if e := ValidateEndpointRequest(good); e != nil {
		t.Fatal(e)
	}
}
func TestRetainedPairBound(t *testing.T) {
	s := EndpointState{Version: 1, Namespace: "/project", Revision: 1, Records: make([]EndpointRecord, 0, MaxRecords)}
	for i := 0; i < MaxRecords; i++ {
		r := req(fmt.Sprintf("%032x", i/256+1), fmt.Sprintf("%032x", i+1), "1", "desktop")
		s.Records = append(s.Records, EndpointRecord{Status: "retired", Binding: EndpointBinding{Namespace: r.Namespace, DocumentID: r.DocumentID, EditorIncarnation: r.EditorIncarnation, AttachID: r.AttachID, PID: r.PID, ProcessIdentity: "birth", Socket: r.Socket, Buffer: r.Buffer, Mode: r.Mode, File: "/file", BindingGeneration: docA}})
	}
	n, p := reg(t, s, req(incB, fmt.Sprintf("%032x", MaxRecords+1), "1", "desktop"))
	if p.Code != "capacity" || len(n.Records) != MaxRecords || p.Changed {
		t.Fatal("retained process/document bound failed")
	}
}
func TestRevisionNeverWraps(t *testing.T) {
	s, _ := reg(t, EndpointState{}, req(incA, docA, "1", "desktop"))
	s.Revision = ^uint64(0)
	r := req(incA, docA, "2", "desktop")
	n, _, e := ApplyEndpointEvent(s, EndpointEvent{Register: &r, ProcessIdentity: "birth", BindingGeneration: docA, File: "/file"})
	if e == nil || n.Revision != s.Revision {
		t.Fatal("revision wrapped")
	}
}
func TestNoopPreservesEncodableEmptyState(t *testing.T) {
	s := EndpointState{Version: 1, Namespace: "/project", Records: []EndpointRecord{}}
	c := ProcessEvidence{EditorIncarnation: incA, PID: 42, ProcessIdentity: "birth"}
	n, p, e := ApplyEndpointEvent(s, EndpointEvent{Collect: &c})
	if e != nil || p.Changed {
		t.Fatal(e)
	}
	if _, e := EncodeEndpointState(n); e != nil {
		t.Fatal(e)
	}
}
