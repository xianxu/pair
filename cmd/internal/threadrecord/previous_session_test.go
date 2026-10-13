package threadrecord

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
)

func validPreviousSession() PreviousSession {
	address := Address{RepoScope: "repo", Tag: "2"}
	return PreviousSession{
		Source:  ParkIdentity{Nonce: "park-source", Address: address, PID: 42, ProcessIdentity: "source-helper"},
		Binding: &couchidentity.SessionBinding{Name: "pair-source", ScopeKey: address.RepoScope, Tag: address.Tag, StartNonce: "start-source", Legacy: true},
		Agent:   "claude", NativeSession: "native-source", Transcripts: []string{"/private/source/session.jsonl"},
		Capture:    &pairlifecycle.PreservedScrollback{Agent: "claude", Token: "capture-source", Events: true},
		CapturedAt: time.Unix(100, 0).UTC(), Unavailable: []string{"optional evidence unavailable"},
	}
}

func TestPreviousSessionValidate(t *testing.T) {
	t.Parallel()
	valid := validPreviousSession()
	if err := valid.Validate(valid.Source.Address); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*PreviousSession)
	}{
		{"wrong source address", func(p *PreviousSession) { p.Source.Address.Tag = "3" }},
		{"wrong binding address", func(p *PreviousSession) { p.Binding.ScopeKey = "other" }},
		{"malformed binding", func(p *PreviousSession) { p.Binding.StartNonce = "../escape" }},
		{"missing process", func(p *PreviousSession) { p.Source.PID = 0 }},
		{"missing process identity", func(p *PreviousSession) { p.Source.ProcessIdentity = "" }},
		{"traversal park nonce", func(p *PreviousSession) { p.Source.Nonce = "../escape" }},
		{"invalid agent", func(p *PreviousSession) { p.Agent = "../claude" }},
		{"long agent", func(p *PreviousSession) { p.Agent = strings.Repeat("x", 65) }},
		{"missing timestamp", func(p *PreviousSession) { p.CapturedAt = time.Time{} }},
		{"foreign capture agent", func(p *PreviousSession) { p.Capture.Agent = "codex" }},
		{"absolute capture token", func(p *PreviousSession) { p.Capture.Token = "/tmp/archive" }},
		{"traversal capture token", func(p *PreviousSession) { p.Capture.Token = "../archive" }},
		{"capture glob", func(p *PreviousSession) { p.Capture.Token = "capture-*" }},
		{"long capture token", func(p *PreviousSession) { p.Capture.Token = strings.Repeat("x", 257) }},
		{"long native session", func(p *PreviousSession) { p.NativeSession = strings.Repeat("x", 1025) }},
		{"terminal controls", func(p *PreviousSession) { p.NativeSession = "session\x1b[2J" }},
		{"invalid UTF8", func(p *PreviousSession) { p.NativeSession = string([]byte{0xff}) }},
		{"relative transcript", func(p *PreviousSession) { p.Transcripts[0] = "session.jsonl" }},
		{"unclean transcript", func(p *PreviousSession) { p.Transcripts[0] = "/private/../session.jsonl" }},
		{"transcript newline", func(p *PreviousSession) { p.Transcripts[0] = "/private/session\n.jsonl" }},
		{"long transcript", func(p *PreviousSession) { p.Transcripts[0] = "/" + strings.Repeat("x", 4096) }},
		{"many transcripts", func(p *PreviousSession) { p.Transcripts = make([]string, 9) }},
		{"many reasons", func(p *PreviousSession) { p.Unavailable = make([]string, 17) }},
		{"long reason", func(p *PreviousSession) { p.Unavailable[0] = strings.Repeat("x", 1025) }},
		{"reason control", func(p *PreviousSession) { p.Unavailable[0] = "bad\rreason" }},
		{"aggregate budget", func(p *PreviousSession) {
			p.Transcripts = make([]string, 8)
			for i := range p.Transcripts {
				p.Transcripts[i] = "/" + strings.Repeat("x", 4095)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ClonePreviousSession(&valid)
			tt.mutate(p)
			if err := p.Validate(valid.Source.Address); err == nil {
				t.Fatal("accepted invalid previous session")
			}
		})
	}
	// A known source with explicitly unavailable evidence remains representable.
	valid.Binding, valid.Capture, valid.Transcripts = nil, nil, nil
	valid.NativeSession = ""
	if err := valid.Validate(valid.Source.Address); err != nil {
		t.Fatal(err)
	}
}

func TestPreviousSessionCloneOwnsAllReferences(t *testing.T) {
	t.Parallel()
	original := validPreviousSession()
	before, _ := json.Marshal(original)
	clone := ClonePreviousSession(&original)
	if !reflect.DeepEqual(clone, &original) {
		t.Fatal("clone lost evidence")
	}
	clone.Source.Nonce = "other"
	clone.Binding.StartNonce = "other"
	clone.Capture.Token = "other"
	clone.Transcripts[0] = "/other"
	clone.Unavailable[0] = "other"
	after, _ := json.Marshal(original)
	if string(before) != string(after) {
		t.Fatal("clone aliases original evidence")
	}
	if ClonePreviousSession(nil) != nil {
		t.Fatal("nil clone should remain nil")
	}
}

func FuzzPreviousSessionValidate(f *testing.F) {
	seed := validPreviousSession()
	raw, _ := json.Marshal(seed)
	f.Add(raw)
	f.Add([]byte(`{"agent":"claude"}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var p PreviousSession
		if json.Unmarshal(raw, &p) != nil {
			return
		}
		if p.Validate(seed.Source.Address) != nil {
			return
		}
		normalized, err := json.Marshal(p)
		if err != nil || len(normalized) > 32*1024 {
			t.Fatal("accepted descriptor exceeds persistence budget")
		}
		cloned := ClonePreviousSession(&p)
		if err := cloned.Validate(seed.Source.Address); err != nil {
			t.Fatalf("clone invalidated accepted descriptor: %v", err)
		}
	})
}

func TestPreviousSessionStrictPersistedShapeAndLegacyAbsence(t *testing.T) {
	t.Parallel()
	record := validRecord()
	legacy, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodePersisted(legacy, record.Address, testValidators)
	if err != nil || decoded.PreviousSession != nil {
		t.Fatalf("old record without descriptor: %v", err)
	}
	previous := validPreviousSession()
	previous.Source.Address = record.Address
	previous.Binding.ScopeKey, previous.Binding.Tag = record.Address.RepoScope, record.Address.Tag
	record.PreviousSession = &previous
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err = DecodePersisted(raw, record.Address, testValidators)
	if err != nil || !reflect.DeepEqual(decoded.PreviousSession, &previous) {
		t.Fatalf("round trip: %+v, %v", decoded.PreviousSession, err)
	}
	for _, injected := range []string{
		`"previous_session":{"raw_path":"/tmp/untrusted",`,
		`"previous_session":{"agent":"codex",`,
	} {
		corrupt := strings.Replace(string(raw), `"previous_session":{`, injected, 1)
		if _, err := DecodePersisted([]byte(corrupt), record.Address, testValidators); err == nil {
			t.Fatal("accepted unknown or duplicate descriptor field")
		}
	}
}
