package sessioninventory_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

// readJSONLArtifact and visitJSONLinesAt are two loops over the same chunked
// read with the same per-record bound; they differ only in what they hand
// back (the raw body vs. framed lines). This pins the parts that must agree:
// the bound trips for exactly the same inputs, a partial tail comes back
// rather than erroring, and the body is the input byte for byte (#237 PQ-3).
func FuzzReadJSONLArtifactAgreesWithTheLineFramer(f *testing.F) {
	f.Add([]byte(`{"a":1}`+"\n"+`{"b":2}`+"\n"), int64(16))
	f.Add([]byte(`{"a":1}`+"\n"+`{"b":2}`), int64(16))
	f.Add([]byte(`{"long":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`+"\n"), int64(16))
	f.Add([]byte("\r\n\r\n"), int64(1))
	f.Add([]byte(""), int64(0))
	f.Fuzz(func(t *testing.T, content []byte, recordLimit int64) {
		if recordLimit < 0 || recordLimit > 1<<16 || len(content) > 1<<16 {
			t.Skip()
		}
		runtime := sessioninventorytest.NewFakeRuntime()
		root := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"}
		runtime.SetPairDataRoot(root)
		artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "ledger-fuzz.jsonl"}
		runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, content)

		body, readErr := sessioninventory.ReadJSONLArtifact(runtime, artifact, recordLimit)
		var framed [][]byte
		visitErr := sessioninventory.VisitJSONLinesAt(runtime, artifact, recordLimit, func(line []byte, _ uint64) bool {
			framed = append(framed, append([]byte(nil), line...))
			return false
		})

		if errors.Is(readErr, sessioninventory.ErrReadLimit) != errors.Is(visitErr, sessioninventory.ErrReadLimit) {
			t.Fatalf("read err=%v, framer err=%v: the per-record bound must trip for the same inputs", readErr, visitErr)
		}
		if readErr != nil {
			return
		}
		if !bytes.Equal(body, content) {
			t.Fatalf("body=%q, want the input byte for byte %q", body, content)
		}
		// A partial tail is the framer's only other failure, and the reader
		// hands it back instead: ParseLedger owns that rule.
		if visitErr != nil && !errors.Is(visitErr, sessioninventory.ErrTruncatedRecord) {
			t.Fatalf("framer err=%v on input the reader accepted", visitErr)
		}
		wantLines := bytes.Split(content, []byte{'\n'})
		if len(wantLines) > 0 {
			wantLines = wantLines[:len(wantLines)-1] // the piece after the last newline is the tail, not a line
		}
		if len(framed) != len(wantLines) {
			t.Fatalf("framer produced %d lines, want %d", len(framed), len(wantLines))
		}
		for i := range wantLines {
			want := bytes.TrimSuffix(wantLines[i], []byte{'\r'})
			if !bytes.Equal(framed[i], want) {
				t.Fatalf("line %d: framer=%q, want %q", i, framed[i], want)
			}
		}
	})
}
