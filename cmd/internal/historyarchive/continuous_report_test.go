package historyarchive

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func validCommandReport() ContinuousCommandReport {
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	return ContinuousCommandReport{Version: ContinuousReportVersion, Outcome: ContinuousComplete, MaxDatabaseAge: 48 * time.Hour, Report: ContinuousReport{Archived: map[string]int{"codex-database": 1}, Databases: []DatabaseBackupSummary{{Source: SourceIdentity{Host: "host", Root: "/native"}, Kind: "codex-database", Path: "state.sqlite", Status: DatabaseArchived, CheckedAt: now, LastVerified: now}}}}
}

func TestContinuousReportRoundTrip(t *testing.T) {
	for _, outcome := range []string{ContinuousComplete, ContinuousFailed, ContinuousNotConfigured, ContinuousAlreadyRunning} {
		t.Run(outcome, func(t *testing.T) {
			want := validCommandReport()
			want.Outcome = outcome
			if outcome == ContinuousNotConfigured || outcome == ContinuousAlreadyRunning {
				want.Report = ContinuousReport{}
			}
			raw, err := EncodeContinuousReport(want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeContinuousReport(raw)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
		})
	}
}

func TestContinuousReportInvalidValues(t *testing.T) {
	cases := map[string]func(*ContinuousCommandReport){
		"version":          func(r *ContinuousCommandReport) { r.Version++ },
		"outcome":          func(r *ContinuousCommandReport) { r.Outcome = "done" },
		"zero age":         func(r *ContinuousCommandReport) { r.MaxDatabaseAge = 0 },
		"negative age":     func(r *ContinuousCommandReport) { r.MaxDatabaseAge = -1 },
		"missing host":     func(r *ContinuousCommandReport) { r.Report.Databases[0].Source.Host = "" },
		"relative root":    func(r *ContinuousCommandReport) { r.Report.Databases[0].Source.Root = "native" },
		"kind":             func(r *ContinuousCommandReport) { r.Report.Databases[0].Kind = "unknown" },
		"path":             func(r *ContinuousCommandReport) { r.Report.Databases[0].Path = "../state.sqlite" },
		"status":           func(r *ContinuousCommandReport) { r.Report.Databases[0].Status = "done" },
		"missing checked":  func(r *ContinuousCommandReport) { r.Report.Databases[0].CheckedAt = time.Time{} },
		"missing verified": func(r *ContinuousCommandReport) { r.Report.Databases[0].LastVerified = time.Time{} },
		"non UTC": func(r *ContinuousCommandReport) {
			r.Report.Databases[0].CheckedAt = r.Report.Databases[0].CheckedAt.In(time.FixedZone("offset", 3600))
		},
		"duplicate row": func(r *ContinuousCommandReport) {
			r.Report.Databases = append(r.Report.Databases, r.Report.Databases[0])
		},
		"unconfigured rows": func(r *ContinuousCommandReport) { r.Outcome = ContinuousNotConfigured },
		"busy counts":       func(r *ContinuousCommandReport) { r.Outcome = ContinuousAlreadyRunning; r.Report.Databases = nil },
		"negative count":    func(r *ContinuousCommandReport) { r.Report.Archived["codex-database"] = -1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := validCommandReport()
			mutate(&r)
			if _, err := EncodeContinuousReport(r); err == nil {
				t.Error("encoder accepted invalid report")
			}
			raw, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeContinuousReport(raw); err == nil {
				t.Error("decoder accepted invalid report")
			}
		})
	}
}

func TestContinuousReportPartialAndDistinctSources(t *testing.T) {
	r := validCommandReport()
	r.Outcome = ContinuousFailed
	row := r.Report.Databases[0]
	row.Status = DatabaseFailed
	row.CheckedAt = time.Time{}
	row.LastVerified = time.Time{}
	row.Source.Root = "/other"
	r.Report.Databases = append(r.Report.Databases, row)
	row.Status = DatabaseDeferred
	row.Path = "another.sqlite"
	r.Report.Databases = append(r.Report.Databases, row)
	raw, err := EncodeContinuousReport(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeContinuousReport(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, got) {
		t.Fatal("partial rows lost")
	}
}

func TestContinuousReportJSONBoundary(t *testing.T) {
	raw, _ := json.Marshal(validCommandReport())
	cases := map[string]string{
		"empty": "", "truncated": string(raw[:len(raw)-1]), "trailing": string(raw) + "{}", "null": "null", "array": "[]",
		"duplicate": `{"version":1,"version":1}`, "nested duplicate": `{"extra":{"a":1,"a":2}}`,
		"missing report":     `{"version":1,"outcome":"complete","max_database_age_ns":1}`,
		"missing version":    `{"outcome":"complete","max_database_age_ns":1,"report":{}}`,
		"missing outcome":    `{"version":1,"max_database_age_ns":1,"report":{}}`,
		"missing max age":    `{"version":1,"outcome":"complete","report":{}}`,
		"version alias":      strings.Replace(string(raw), `"version":1`, `"version":2,"VERSION":1`, 1),
		"status alias":       strings.Replace(string(raw), `"Status":"archived"`, `"Status":"failed","status":"archived"`, 1),
		"wrong version type": strings.Replace(string(raw), `"version":1`, `"version":"1"`, 1),
		"null count":         strings.Replace(string(raw), `"codex-database":1`, `"codex-database":null`, 1),
		"null detail":        strings.Replace(string(raw), `"Detail":""`, `"Detail":null`, 1),
		"wrong rows type":    strings.Replace(string(raw), `"Databases":[`, `"Databases":{`, 1),
		"oversize":           strings.Repeat(" ", MaxContinuousReportBytes+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeContinuousReport([]byte(input)); err == nil {
				t.Fatal("accepted malformed report")
			}
		})
	}
	additive := strings.Replace(string(raw), `"report":{`, `"future":{"nested":[true]},"report":{"extra":{"ok":true},`, 1)
	if _, err := DecodeContinuousReport([]byte(additive)); err != nil {
		t.Fatalf("additive fields rejected: %v", err)
	}
	r := validCommandReport()
	r.Report.Skipped = []string{strings.Repeat("x", MaxContinuousReportBytes)}
	if raw, err := EncodeContinuousReport(r); err == nil || len(raw) != 0 {
		t.Fatal("oversize encoder must return no bytes and an error")
	}
}

func FuzzDecodeContinuousReport(f *testing.F) {
	raw, _ := json.Marshal(validCommandReport())
	f.Add(raw)
	for _, seed := range []string{"", "null", "[]", "{}{}", `{"version":1,"version":2}`, `{"report":{"extra":{"a":1,"a":2}}}`} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		r, err := DecodeContinuousReport(raw)
		if err != nil {
			return
		}
		encoded, err := EncodeContinuousReport(r)
		if err != nil {
			// An input within the limit can grow when absent fields are
			// materialized; the encoder is allowed to refuse that growth.
			return
		}
		if len(encoded) > MaxContinuousReportBytes {
			t.Fatal("encoded report exceeds limit")
		}
		if _, err := DecodeContinuousReport(encoded); err != nil {
			t.Fatal(err)
		}
	})
}
