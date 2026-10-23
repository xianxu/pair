package testcouchbridge

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func goodReadiness() Readiness {
	return Readiness{Version: 1, Namespace: "/private/bridge/store", Socket: "/private/bridge/service.sock", Environment: map[string]string{"COUCH_ISOLATED_ROOT": "/private/bridge", "COUCH_STORE_DIR": "/private/bridge/store", "PATH": "/private/bridge/bin", "PAIR_HOME": "/source/pair"}, Documents: []Document{{ID: strings.Repeat("a", 32), File: "/private/bridge/artifacts/first.md"}, {ID: strings.Repeat("b", 32), File: "/private/bridge/artifacts/first.side/second.md"}}}
}

func TestReadinessHasOneBoundedJSONLine(t *testing.T) {
	r := goodReadiness()
	line, err := EncodeReadiness(r)
	if err != nil {
		t.Fatal(err)
	}
	if len(line) > ReadinessLimit || bytes.Count(line, []byte{'\n'}) != 1 || line[len(line)-1] != '\n' {
		t.Fatal("not one bounded line")
	}
	var got Readiness
	if err = json.Unmarshal(line, &got); err != nil {
		t.Fatal(err)
	}
	if got.Documents[1].File != r.Documents[1].File || got.Environment["PAIR_HOME"] != "/source/pair" {
		t.Fatal("lost complete environment or side document")
	}
}

func TestReadinessRejectsMalformedAuthorityAndLayout(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*Readiness)
	}{
		{"version", func(r *Readiness) { r.Version = 2 }},
		{"namespace", func(r *Readiness) { r.Namespace = "/elsewhere" }},
		{"socket length", func(r *Readiness) { r.Socket = "/" + strings.Repeat("x", 103) }},
		{"path", func(r *Readiness) { r.Environment["PATH"] = "/usr/bin" }},
		{"ambient path suffix", func(r *Readiness) { r.Environment["PATH"] += ":/usr/bin" }},
		{"private path sibling", func(r *Readiness) { r.Environment["PATH"] += ":/private/bridge-other/bin" }},
		{"empty path component", func(r *Readiness) { r.Environment["PATH"] += ":" }},
		{"environment name", func(r *Readiness) { r.Environment["BAD=NAME"] = "value" }},
		{"environment nul", func(r *Readiness) { r.Environment["OTHER"] = "a\x00b" }},
		{"utf8", func(r *Readiness) { r.Environment["OTHER"] = string([]byte{255}) }},
		{"duplicate ID", func(r *Readiness) { r.Documents[1].ID = r.Documents[0].ID }},
		{"uppercase ID", func(r *Readiness) { r.Documents[0].ID = strings.Repeat("A", 32) }},
		{"document order", func(r *Readiness) { r.Documents[0], r.Documents[1] = r.Documents[1], r.Documents[0] }},
		{"document count", func(r *Readiness) { r.Documents = r.Documents[:1] }},
		{"oversized", func(r *Readiness) { r.Environment["OTHER"] = strings.Repeat("x", ReadinessLimit) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := goodReadiness()
			tt.mutate(&r)
			if _, err := EncodeReadiness(r); err == nil {
				t.Fatal("accepted invalid readiness")
			}
		})
	}
}

func TestReadinessBoundIncludesEscapingAndNewline(t *testing.T) {
	r := goodReadiness()
	// Raw input fits, but JSON expands each '<' to a six-byte escape.
	r.Environment["PAYLOAD"] = strings.Repeat("<", ReadinessLimit/6)
	if err := ValidateReadiness(r); err != nil {
		t.Fatalf("raw preflight unexpectedly rejected bounded input: %v", err)
	}
	if _, err := EncodeReadiness(r); err == nil {
		t.Fatal("accepted oversized escaped JSON")
	}
	r.Environment["PAYLOAD"] = ""
	base, err := EncodeReadiness(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Environment["PAYLOAD"] = strings.Repeat("x", ReadinessLimit-len(base))
	line, err := EncodeReadiness(r)
	if err != nil || len(line) != ReadinessLimit {
		t.Fatalf("exact limit: bytes=%d error=%v", len(line), err)
	}
	r.Environment["PAYLOAD"] += "x"
	if _, err := EncodeReadiness(r); err == nil {
		t.Fatal("newline not included in output limit")
	}
}

func TestReadinessCountedWorkIsLinear(t *testing.T) {
	var counts []int
	for _, n := range []int{4096, 8192} {
		r := goodReadiness()
		r.Environment["PAYLOAD"] = strings.Repeat("x", n)
		_, work, err := validateReadiness(r)
		if err != nil {
			t.Fatal(err)
		}
		counts = append(counts, work)
	}
	if counts[1]-counts[0] != 4096 {
		t.Fatalf("work must visit payload bytes once: %v", counts)
	}
}

func FuzzReadiness(f *testing.F) {
	f.Add("/private/bridge/service.sock", "VALUE", "hello")
	f.Fuzz(func(t *testing.T, socket, key, value string) {
		if len(socket)+len(key)+len(value) > ReadinessLimit {
			t.Skip()
		}
		r := goodReadiness()
		r.Socket = socket
		r.Environment[key] = value
		line, err := EncodeReadiness(r)
		if err == nil && (len(line) > ReadinessLimit || !json.Valid(line)) {
			t.Fatal("invalid encoded result")
		}
	})
}
