package submission

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func pageFixture() Submission {
	s := fixture()
	s.Version = 2
	s.Source.Page = &PageIdentity{URL: "http://localhost:4321/blog", Title: "Blog", Provenance: "cdp"}
	s.Notes[0].SourceDocument = s.Source.Document
	s.Notes[0].Crop = Crop{File: "crop-0.png", Box: [4]float64{0, 0, 2, 2}, CapturedAt: time.Now().UTC(), Document: s.Source.Document}
	s.PrivacyDecision = (OriginPolicy{}).Decide(s.Source.Page.URL)
	return s
}

func TestPageIdentityAndSourceBinding(t *testing.T) {
	s := pageFixture()
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Submission){
		func(s *Submission) { s.Source.Page = nil },
		func(s *Submission) { s.Source.Page.URL = "" },
		func(s *Submission) { s.Source.Page.URL = "relative/path" },
		func(s *Submission) { s.Source.Page.URL = "https://elsewhere.test" },
		func(s *Submission) { s.Source.Page.Title = strings.Repeat("x", MaxPageTitleBytes+1) },
		func(s *Submission) {
			s.Source.Page.URL = "https://example.test/" + strings.Repeat("x", MaxPageURLBytes)
		},
		func(s *Submission) { s.Source.Page.Provenance = "recording-head" },
		func(s *Submission) { s.Notes[0].SourceDocument++ },
		func(s *Submission) { s.Notes[0].Crop.Document++ },
		func(s *Submission) { s.Notes[0].At.Sel = "button:nth-child(1)" },
		func(s *Submission) {
			s.Notes[0].BoxText = BoxText{Region: [4]float64{0, 0, 2, 2}, VisibleText: []BoxTextEntry{{Selector: "#x", Text: "x"}}}
		},
	} {
		s := pageFixture()
		mutate(&s)
		if Validate(s) == nil {
			t.Fatalf("accepted inconsistent page: %+v", s)
		}
	}
	s.Source.Page.Title = ""
	if err := Validate(s); err != nil {
		t.Fatal("empty title rejected", err)
	}
	s = fixture()
	s.Version = 2
	s.Privacy = Private
	s.Notes[0].SourceDocument = s.Source.Document
	if err := Validate(s); err != nil {
		t.Fatal("unavailable batch rejected", err)
	}
	s.PrivacyOrigin = "https://invented.test"
	if Validate(s) == nil {
		t.Fatal("unavailable capture claimed origin")
	}
}

func TestPageManifestRejectsAliases(t *testing.T) {
	b, _ := json.Marshal(pageFixture())
	for _, pair := range [][2]string{{`"page":`, `"Page":`}, {`"url":`, `"URL":`}, {`"source_document":`, `"Source_document":`}, {`"source":`, `"Source":`}, {`"document":`, `"Document":`}, {`"version":`, `"Version":`}} {
		var s Submission
		if err := json.Unmarshal([]byte(strings.Replace(string(b), pair[0], pair[1], 1)), &s); err == nil {
			t.Fatalf("accepted alias %s", pair[1])
		}
	}
}

func TestPageStoreRoundTripAndLegacySelectors(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := pageFixture()
	if _, err = store.Prepare(s, map[string][]byte{"crop-0.png": pngBytes(t)}); err != nil {
		t.Fatal(err)
	}
	store, err = NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Transition(s.ID, s.Target, Begin)
	if err != nil {
		t.Fatal(err)
	}
	if *got.Source.Page != *s.Source.Page || got.Notes[0].SourceDocument != s.Source.Document {
		t.Fatal("lost page binding", got)
	}
	changed := s
	page := *s.Source.Page
	page.Title = "changed"
	changed.Source.Page = &page
	if _, err = store.Prepare(changed, map[string][]byte{"crop-0.png": pngBytes(t)}); err == nil {
		t.Fatal("mutable retry accepted")
	}
	for _, event := range []Event{Ack, Complete} {
		if _, err = store.Transition(s.ID, s.Target, event); err != nil {
			t.Fatal(err)
		}
	}
	legacy := fixture()
	legacy.ID = "legacy"
	legacy.Notes[0].At.Sel = "#old"
	legacy.Notes[0].BoxText = BoxText{Region: [4]float64{0, 0, 2, 2}, VisibleText: []BoxTextEntry{{Selector: "#old", Text: "old"}}}
	if _, err = store.Prepare(legacy, nil); err != nil {
		t.Fatal("legacy refused", err)
	}
	if _, err = store.Transition(legacy.ID, legacy.Target, Begin); err != nil {
		t.Fatal("legacy reload refused", err)
	}
}
