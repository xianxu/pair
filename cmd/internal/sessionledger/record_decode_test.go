package sessionledger

import (
	"fmt"
	"strings"
	"testing"
)

func TestRecordFieldDecodeRetainsWholeRowStrictness(t *testing.T) {
	const boundary = `{"storage_root":"native","relative_path":"rollout.jsonl","stable_file_id":"inode","mutation_token":"mtime","raw_size":7}`
	const prefix = `{"v":3,"kind":"launch","scope_key":"scope","tag":"tag","agent":"codex","pair_log_offset":0,"baseline_complete":true,"artifact_boundaries":[`
	valid := prefix + boundary + `]}`
	for _, test := range []struct {
		name, raw string
		valid     bool
	}{
		{"valid", valid, true},
		{"nested duplicate", strings.Replace(valid, `"raw_size":7`, `"raw_size":7,"raw_size":8`, 1), false},
		{"escaped nested duplicate", strings.Replace(valid, `"raw_size":7`, `"raw_size":7,"raw_\u0073ize":8`, 1), false},
		{"nested unknown", strings.Replace(valid, `"raw_size":7`, `"raw_size":7,"future":8`, 1), false},
		{"nested null", strings.Replace(valid, `"raw_size":7`, `"raw_size":null`, 1), false},
		{"array null", strings.Replace(valid, `[`+boundary+`]`, `null`, 1), false},
		{"scalar null", strings.Replace(valid, `"agent":"codex"`, `"agent":null`, 1), false},
		{"scalar wrong type", strings.Replace(valid, `"agent":"codex"`, `"agent":[]`, 1), false},
		{"root escaped duplicate", strings.Replace(valid, `"agent":"codex"`, `"agent":"codex","ag\u0065nt":"claude"`, 1), false},
		{"trailing object", valid + ` {}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeRecord([]byte(test.raw))
			if (err == nil) != test.valid {
				t.Fatalf("decodeRecord error=%v; want valid=%v", err, test.valid)
			}
		})
	}
}

func BenchmarkParseLedgerArtifactBoundaries(b *testing.B) {
	var raw strings.Builder
	raw.WriteString(`{"v":3,"kind":"launch","scope_key":"scope","tag":"tag","agent":"codex","pair_log_offset":0,"baseline_complete":true,"artifact_boundaries":[`)
	for i := 0; i < 4096; i++ {
		if i > 0 {
			raw.WriteByte(',')
		}
		fmt.Fprintf(&raw, `{"storage_root":"native","relative_path":"session-%06d.jsonl","stable_file_id":"inode-%06d","mutation_token":"mtime-%06d","raw_size":4096}`, i, i, i)
	}
	raw.WriteString("]}\n")
	data := []byte(raw.String())
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parsed := ParseLedger(data)
		if len(parsed.Records) != 1 || len(parsed.MalformedOrdinals) != 0 {
			b.Fatalf("parse failed: %#v", parsed.MalformedOrdinals)
		}
	}
}
