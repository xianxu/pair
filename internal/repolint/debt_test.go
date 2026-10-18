package repolint

import (
	"encoding/json"
	"github.com/xianxu/ariadne/pkg/golint"
	"strings"
	"testing"
)

func TestSleepIdentityAndDebt(t *testing.T) {
	src := []byte("package p\nimport \"time\"\nfunc TestA(){time.Sleep(time.Second);time.Sleep(time.Second)}\n")
	sites, err := sleepSites("x_test.go", src)
	if err != nil || len(sites) != 2 {
		t.Fatalf("sites=%v err=%v", sites, err)
	}
	shifted, err := sleepSites("x_test.go", []byte(strings.ReplaceAll(string(src), "func TestA", "\n\nfunc TestA")))
	if err != nil || sites[0].Site != shifted[0].Site || sites[0].Site.Ordinal != 1 || sites[1].Site.Ordinal != 2 {
		t.Fatalf("unstable: %v %v", sites, shifted)
	}
	debt := Debt{Version: 1, Sites: []SleepSite{sites[0].Site, sites[1].Site}}
	data, _ := json.Marshal(debt)
	decoded, err := decodeDebt(data)
	if err != nil {
		t.Fatal(err)
	}
	findings := []locatedFinding{{Path: "x_test.go", Finding: golint.Finding{Line: 3, Rule: "test-fixed-sleep"}}}
	// The deliberately same-line pair must both be accounted for, never collapsed.
	if _, err := checkDebt(findings, sites, decoded); err == nil {
		t.Fatal("ambiguous line admitted")
	}
	if err := checkShrink(debt, Debt{Version: 1, Sites: debt.Sites[:1]}); err != nil {
		t.Fatal(err)
	}
	added := debt
	added.Sites = append([]SleepSite(nil), debt.Sites...)
	added.Sites[0].Function = "Other"
	if err := checkShrink(debt, added); err == nil {
		t.Fatal("same-count replacement admitted")
	}
}

func TestDecodeDebtRejectsMalformed(t *testing.T) {
	for _, data := range []string{`null`, `{}`, `{"version":2,"sites":[]}`, `{"version":1,"sites":[],"extra":true}`, `{"version":1,"sites":[]} {}`, `{"version":1,"sites":[{"path":"../x_test.go","function":"F","call":"time.Sleep(time.Second)","ordinal":1}]}`} {
		if _, err := decodeDebt([]byte(data)); err == nil {
			t.Errorf("accepted %s", data)
		}
	}
}

func FuzzDecodeDebt(f *testing.F) {
	f.Add([]byte(`{"version":1,"sites":[]}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := decodeDebt(data)
		if err != nil {
			return
		}
		encoded, e := json.Marshal(d)
		if e != nil {
			t.Fatal(e)
		}
		again, e := decodeDebt(encoded)
		if e != nil || len(again.Sites) != len(d.Sites) {
			t.Fatalf("round trip %v", e)
		}
	})
}

func TestSleepSiteIdentityMutations(t *testing.T) {
	source := "package p\nimport \"time\"\nfunc TestA(){\n time.Sleep(time.Second)\n}\n"
	original, err := sleepSites("x_test.go", []byte(source))
	if err != nil || len(original) != 1 {
		t.Fatalf("sites %v %v", original, err)
	}
	for _, spacing := range []string{"\n", "\n\n\n", " /* spacing */ "} {
		changed := strings.ReplaceAll(source, "time.Sleep(time.Second)", "time.Sleep( "+spacing+"time.Second )")
		sites, err := sleepSites("x_test.go", []byte(changed))
		if err != nil || len(sites) != 1 || sites[0].Site != original[0].Site {
			t.Fatalf("whitespace identity changed %v %v", sites, err)
		}
	}
	for _, mutation := range []struct{ path, source string }{{"y_test.go", source}, {"x_test.go", strings.ReplaceAll(source, "TestA", "TestB")}, {"x_test.go", strings.ReplaceAll(source, "time.Second", "2*time.Second")}} {
		sites, err := sleepSites(mutation.path, []byte(mutation.source))
		if err != nil || len(sites) != 1 || sites[0].Site == original[0].Site {
			t.Fatalf("identity mutation undetected %v %v", sites, err)
		}
	}
	if _, err := sleepSites("x_test.go", []byte("package ???")); err == nil {
		t.Fatal("parse error swallowed")
	}
}

func TestDecodeDebtRejectsDuplicateFieldsAndSites(t *testing.T) {
	site := SleepSite{Path: "x_test.go", Function: "TestA", Call: "time.Sleep(time.Second)", Ordinal: 1}
	data, _ := json.Marshal(Debt{Version: 1, Sites: []SleepSite{site, site}})
	if _, err := decodeDebt(data); err == nil {
		t.Fatal("duplicate site accepted")
	}
	for _, data := range []string{`{"version":0,"version":1,"sites":[]}`, `{"version":1,"sites":[{"path":"bad","path":"x_test.go","function":"F","call":"time.Sleep(time.Second)","ordinal":1}]}`} {
		if _, err := decodeDebt([]byte(data)); err == nil {
			t.Fatalf("duplicate field accepted: %s", data)
		}
	}
}

func TestDebtRejectsStaleAndNewSleep(t *testing.T) {
	site := sleepAt{SleepSite{"x_test.go", "TestA", "time.Sleep(time.Second)", 1}, 3}
	debt := Debt{Version: 1, Sites: []SleepSite{site.Site}}
	finding := locatedFinding{"x_test.go", golint.Finding{Line: 3, Rule: sleepRule}}
	remaining, err := checkDebt([]locatedFinding{finding}, []sleepAt{site}, debt)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("exact site refused %v %v", remaining, err)
	}
	if _, err := checkDebt(nil, []sleepAt{site}, debt); err == nil {
		t.Fatal("unused site accepted")
	}
	moved := site
	moved.Site.Function = "TestOther"
	remaining, err = checkDebt([]locatedFinding{finding}, []sleepAt{moved}, Debt{Version: 1, Sites: []SleepSite{}})
	if err != nil || len(remaining) != 1 {
		t.Fatalf("new site concealed %v %v", remaining, err)
	}
}

func TestDecodeDebtRejectsInvalidSiteValues(t *testing.T) {
	base := SleepSite{Path: "x_test.go", Function: "TestA", Call: "time.Sleep(time.Second)", Ordinal: 1}
	for _, mutate := range []func(*SleepSite){
		func(s *SleepSite) { s.Path = "/absolute_test.go" },
		func(s *SleepSite) { s.Path = "x/../x_test.go" },
		func(s *SleepSite) { s.Path = "x\\x_test.go" },
		func(s *SleepSite) { s.Path = "workshop/x_test.go" },
		func(s *SleepSite) { s.Path = "p/testdata/x_test.go" },
		func(s *SleepSite) { s.Path = "x.go" },
		func(s *SleepSite) { s.Function = "" },
		func(s *SleepSite) { s.Ordinal = 0 },
		func(s *SleepSite) { s.Call = "time.Sleep(5 * time.Millisecond)" },
		func(s *SleepSite) { s.Call = "time.Sleep(variable)" },
		func(s *SleepSite) { s.Call = "time.Other(time.Second)" },
		func(s *SleepSite) { s.Call = "time.Sleep( time.Second )" },
	} {
		site := base
		mutate(&site)
		data, _ := json.Marshal(Debt{Version: 1, Sites: []SleepSite{site}})
		if _, err := decodeDebt(data); err == nil {
			t.Errorf("invalid site admitted %+v", site)
		}
	}
}
