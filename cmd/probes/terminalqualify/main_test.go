package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	q "github.com/xianxu/pair/cmd/internal/terminalqualify"
	"io"
	"testing"
)

func TestRunExitStatusMatchesQualification(t *testing.T) {
	for _, status := range []q.Status{q.Pass, q.Fail, q.NotCovered} {
		var out, diagnostic bytes.Buffer
		code := run(context.Background(), &out, &diagnostic, func(context.Context) (q.Report, error) {
			return q.Report{Required: []string{"a"}, Results: []q.Result{{ID: "a", Status: status}}}, nil
		})
		want := 1
		if status == q.Pass {
			want = 0
		}
		if code != want {
			t.Fatalf("%s: exit=%d diagnostic=%s", status, code, &diagnostic)
		}
		var doc struct {
			Qualified bool       `json:"qualified"`
			Results   []q.Result `json:"results"`
		}
		if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.Qualified != (want == 0) || len(doc.Results) != 1 {
			t.Fatalf("%s", out.Bytes())
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestRunInfrastructureRefusesQualification(t *testing.T) {
	runners := []func(context.Context) (q.Report, error){func(context.Context) (q.Report, error) { return q.Report{}, errors.New("backend unavailable") }, func(context.Context) (q.Report, error) { return q.Report{}, nil }}
	for _, runner := range runners {
		if got := run(context.Background(), io.Discard, io.Discard, runner); got != 2 {
			t.Fatalf("exit=%d", got)
		}
	}
	if got := run(context.Background(), brokenWriter{}, io.Discard, func(context.Context) (q.Report, error) {
		return q.Report{Required: []string{"a"}, Results: []q.Result{{ID: "a", Status: q.Pass}}}, nil
	}); got != 2 {
		t.Fatalf("write failure exit=%d", got)
	}
}

func TestRunJSONPreservesStructuredEvidence(t *testing.T) {
	var out bytes.Buffer
	result := q.Result{ID: "a", Status: q.Fail, Expected: q.Observation{"cursor": "1,0"}, Observed: q.Observation{"cursor": "2,0"}, ObservedTruncated: true, Comparison: "whole-split"}
	code := run(context.Background(), &out, io.Discard, func(context.Context) (q.Report, error) {
		return q.Report{Required: []string{"a"}, Results: []q.Result{result}}, nil
	})
	if code != 1 {
		t.Fatalf("exit=%d", code)
	}
	var doc struct {
		Results []struct {
			Expected, Observed map[string]string
			Comparison         string `json:"comparison"`
			ExpectedTruncated  bool   `json:"expected_truncated"`
			ObservedTruncated  bool   `json:"observed_truncated"`
		}
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Results) != 1 {
		t.Fatalf("%s", out.Bytes())
	}
	got := doc.Results[0]
	if got.Expected["cursor"] != "1,0" || got.Observed["cursor"] != "2,0" || got.ExpectedTruncated || !got.ObservedTruncated || got.Comparison != "whole-split" {
		t.Fatalf("%s", out.Bytes())
	}
}
