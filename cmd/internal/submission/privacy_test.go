package submission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOriginPrivacyExactMatch(t *testing.T) {
	p, err := ParseOriginPolicy([]byte(`{"origins":[{"origin":"HTTP://LOCALHOST:80/","privacy":"shareable"},{"origin":"https://private.test","privacy":"private"},{"origin":"http://[::1]:4321","privacy":"shareable"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ url, want string }{
		{"http://localhost/path?query=secret", "shareable"}, {"http://localhost:80/", "shareable"},
		{"https://localhost/", "private"}, {"http://localhost:4321/", "private"},
		{"http://localhost.attacker/", "private"}, {"https://private.test/", "private"},
		{"http://[0:0:0:0:0:0:0:1]:4321/", "shareable"}, {"data:text/html,secret", "private"},
		{"http://user:password@localhost/", "private"}, {"", "private"},
	} {
		t.Run(tc.url, func(t *testing.T) {
			d := p.Decide(tc.url)
			if string(d.Privacy) != tc.want {
				t.Fatalf("%+v", d)
			}
			if err := d.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
	for i := 0; i < 100; i++ {
		if p.Decide(fmt.Sprintf("http://localhost.evil%d/", i)).Privacy != Private {
			t.Fatal("suffix granted")
		}
	}
	d := p.Decide("https://private.test/")
	if d.PrivacyRule.Rule == nil {
		t.Fatal("explicit private match lost")
	}
}

func TestOriginPrivacyRejectsAmbiguousPolicy(t *testing.T) {
	for _, data := range []string{
		`null`, `{}`, `{"origins":null}`, `{"origins":[],"extra":true}`, `{"origins":[]} {}`,
		`{"origins":[{"origin":"https://example.test/path","privacy":"shareable"}]}`,
		`{"origins":[{"origin":"https://example.test?","privacy":"shareable"}]}`,
		`{"origins":[{"origin":"https://example.test#x","privacy":"shareable"}]}`,
		`{"origins":[{"origin":"https://*.example.test","privacy":"shareable"}]}`,
		`{"origins":[{"origin":"https://example.test:99999","privacy":"shareable"}]}`,
		`{"origins":[{"origin":"https://example.test:","privacy":"shareable"}]}`,
		`{"origins":[{"origin":"https://example.test","privacy":"public"}]}`,
		`{"origins":[{"origin":"https://example.test","privacy":"shareable","extra":1}]}`,
		`{"origins":[{"origin":"https://example.test","privacy":"shareable"},{"origin":"https://EXAMPLE.test:443","privacy":"private"}]}`,
		strings.Repeat(" ", MaxOriginPolicyBytes+1),
	} {
		if _, err := ParseOriginPolicy([]byte(data)); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	rules := make([]string, MaxOriginRules+1)
	for i := range rules {
		rules[i] = fmt.Sprintf(`{"origin":"https://host%d.test","privacy":"shareable"}`, i)
	}
	if _, err := ParseOriginPolicy([]byte(`{"origins":[` + strings.Join(rules, ",") + `]}`)); err == nil {
		t.Fatal("unbounded rules")
	}
}

func TestPrivacyManifestAndInvalidPolicy(t *testing.T) {
	p, _ := ParseOriginPolicy([]byte(`{"origins":[{"origin":"https://example.test","privacy":"shareable"}]}`))
	for _, d := range []PrivacyDecision{p.Decide("https://example.test/path"), (OriginPolicy{}).Decide(""), InvalidOriginPolicy("invalid JSON").Decide("https://example.test")} {
		sub := fixture()
		sub.PrivacyDecision = d
		b, err := json.Marshal(sub)
		if err != nil {
			t.Fatal(err)
		}
		var got Submission
		if err = json.Unmarshal(b, &got); err != nil {
			t.Fatal(err)
		}
		if err = Validate(got); err != nil {
			t.Fatal(err)
		}
		b2, _ := json.Marshal(got)
		if string(b) != string(b2) {
			t.Fatal("decision did not round trip")
		}
		var wire map[string]json.RawMessage
		json.Unmarshal(b, &wire)
		if _, ok := wire["privacy"]; !ok {
			t.Fatal("missing privacy")
		}
		if _, ok := wire["privacy_rule"]; !ok {
			t.Fatal("missing rule")
		}
	}
	invalid := InvalidOriginPolicy("invalid JSON").Decide("https://example.test")
	b, _ := json.Marshal(invalid)
	if !strings.Contains(string(b), `"privacy_rule":"invalid policy file: invalid JSON"`) {
		t.Fatal(string(b))
	}
	invalid.Privacy = Shareable
	if invalid.Validate() == nil {
		t.Fatal("invalid file permitted sharing")
	}
	if err := Validate(fixture()); err != nil {
		t.Fatal("legacy bundle refused", err)
	}
	for _, wire := range []string{`{"privacy":"shareable","privacy_rule":null}`, `{"privacy":"public"}`, `{"privacy":"private","privacy_rule":42}`, `{"privacy":"private","privacy_rule":"anything"}`, `{"privacy":"shareable","privacy_origin":"https://other.test","privacy_rule":{"origin":"https://example.test","privacy":"shareable"}}`} {
		var d PrivacyDecision
		err := json.Unmarshal([]byte(wire), &d)
		if err == nil && d.Validate() == nil {
			t.Fatalf("accepted %s", wire)
		}
	}
}

func FuzzOriginPrivacy(f *testing.F) {
	for _, s := range []string{`{"origins":[]}`, `{"origins":[{"origin":"http://localhost:4321","privacy":"shareable"}]}`, `null`, `{"origins":`} {
		f.Add(s, "https://unlisted.test/path")
	}
	f.Fuzz(func(t *testing.T, data, url string) {
		p, err := ParseOriginPolicy([]byte(data))
		if err != nil {
			return
		}
		d := p.Decide(url)
		if err := d.Validate(); err != nil {
			t.Fatal(err)
		}
		if d.Privacy == Shareable && (d.PrivacyRule.Rule == nil || d.PrivacyRule.Rule.Origin != d.PrivacyOrigin) {
			t.Fatal("unbacked grant")
		}
	})
}

func TestPrivacyStoreFreezesDecisionAcrossDelivery(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := ParseOriginPolicy([]byte(`{"origins":[{"origin":"https://example.test","privacy":"shareable"}]}`))
	sub := fixture()
	sub.PrivacyDecision = p.Decide("https://example.test")
	if _, err = store.Prepare(sub, nil); err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(sub.PrivacyDecision)
	for _, event := range []Event{Begin, Ack, Complete} {
		store, err = NewStore(root)
		if err != nil {
			t.Fatal(err)
		}
		got, err := store.Transition(sub.ID, sub.Target, event)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(got.PrivacyDecision)
		if string(want) != string(b) {
			t.Fatal("permission changed on reopen/delivery")
		}
	}
	sub.PrivacyDecision = (OriginPolicy{}).Decide("https://example.test")
	if _, err = store.Prepare(sub, nil); err == nil {
		t.Fatal("retry changed immutable privacy")
	}
}

func TestOriginPrivacyRejectsCaseAliases(t *testing.T) {
	for _, data := range []string{
		`{"origins":[],"Origins":[{"origin":"https://example.test","privacy":"shareable"}]}`,
		`{"Origins":[{"origin":"https://example.test","privacy":"shareable"}]}`,
		`{"origins":[{"origin":"https://example.test","privacy":"private","Privacy":"shareable"}]}`,
		`{"origins":[{"origin":"https://private.test","Origin":"https://example.test","privacy":"shareable"}]}`,
		`{"originſ":[{"origin":"https://example.test","privacy":"shareable"}]}`,
	} {
		if p, err := ParseOriginPolicy([]byte(data)); err == nil {
			t.Errorf("accepted ambiguous policy: %s => %+v", data, p.Decide("https://example.test"))
		}
	}
	for _, data := range []string{
		`{"origin":"https://example.test","privacy":"private","Privacy":"shareable"}`,
		`{"Origin":"https://example.test","privacy":"shareable"}`,
		`{"origin":"https://example.test","Privacy":"shareable"}`,
	} {
		var evidence PolicyEvidence
		if err := json.Unmarshal([]byte(data), &evidence); err == nil {
			t.Errorf("accepted ambiguous evidence: %s", data)
		}
	}
}

func TestPrivacyManifestRejectsAmbiguousKeysOnReload(t *testing.T) {
	for _, suffix := range []string{
		`,"Privacy":"shareable","Privacy_rule":{"origin":"https://example.test","privacy":"shareable"}`,
		`,"privacy":"shareable","privacy_rule":{"origin":"https://example.test","privacy":"shareable"}`,
		`,"Privacy_origin":"https://other.test"`,
	} {
		t.Run(suffix, func(t *testing.T) {
			dir := t.TempDir()
			store, err := NewStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			sub := fixture()
			sub.PrivacyDecision = (OriginPolicy{}).Decide("https://example.test")
			if _, err := store.Prepare(sub, nil); err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(sub)
			if err != nil {
				t.Fatal(err)
			}
			b = append(b[:len(b)-1], []byte(suffix+"}")...)
			var decoded Submission
			if err := json.Unmarshal(b, &decoded); err == nil {
				t.Error("direct decoder accepted ambiguous permission")
			}
			if err := os.WriteFile(filepath.Join(dir, sub.ID, "submission.json"), b, 0600); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if got, err := readMetadata(root, sub.ID, false); err == nil {
				t.Fatalf("reload accepted ambiguous permission: %+v", got.PrivacyDecision)
			}
		})
	}
}

func TestPrivacySharedScanVisitsOpaqueExtensions(t *testing.T) {
	for _, key := range []string{"Privacy", "privacy_rule", "URL"} {
		raw := []byte(`{"future":[{"` + key + `":1e9999}]}`)
		seen := false
		sentinel := fmt.Errorf("visitor authority")
		err := checkJSONKeys(raw, func(depth int, name string) error {
			if name == key {
				if depth != 2 {
					t.Fatalf("depth=%d", depth)
				}
				seen = true
				return sentinel
			}
			return nil
		})
		if err != sentinel || !seen {
			t.Fatalf("visitor lost opaque extension key %q: %v", key, err)
		}
	}
	if err := checkJSONKeys([]byte(`{"future":{"a":1e9999,"A":2}}`), func(int, string) error { return nil }); err != nil {
		t.Fatal("exact-key policy changed", err)
	}
	for _, raw := range []string{`{"future":[{"a":1,"\u0061":2}]}`, `{} {}`} {
		if err := checkJSONKeys([]byte(raw), func(int, string) error { return nil }); err == nil {
			t.Fatalf("accepted ambiguous JSON %s", raw)
		}
	}
	for _, key := range []string{"Privacy", "URL", "Source_document"} {
		raw := []byte(`{"future":[{"` + key + `":1e9999}]}`)
		_, err := DecodePreserving(raw)
		if err == nil {
			t.Fatalf("permission alias accepted in unknown extension: %s", raw)
		}
	}
}
