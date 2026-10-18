package sessionsync

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

// FileApplyRequest is an apply with no Couch around it (pair#661 M2): the
// review workbench's `pair review apply`. It has no owner, owed events or
// registry; FlockPath and RoundPath live in pair's data dir, never beside the
// document, so a review leaves nothing in the repository.
type FileApplyRequest struct {
	File, IfHash         string
	FlockPath, RoundPath string
	Edits                []EditRecord
	Now                  time.Time
}

// FileApplyResult is what landed: the document's new hash, its round, and
// the agent's commit body (the round as review records).
type FileApplyResult struct {
	Hash  string
	Round *Round
	Body  string
}

// RoundArtifact is the review pane's round file: which document, the round,
// and the commit body.
type RoundArtifact struct {
	File  string `json:"file"`
	Round Round  `json:"round"`
	Body  string `json:"body"`
}

// ApplyFile writes one edit through the shared engine, in the apply order:
// flock, the operator lock (exit 3), the whole-file hash (exit 6), BuildEdit,
// then the previous round artifact removed before the document moves and the
// new one written after it, so a failure leaves no round, never a stale one.
func ApplyFile(req FileApplyRequest) (FileApplyResult, error) {
	if req.IfHash == "" {
		return FileApplyResult{}, refuse("apply needs --if-hash: pass the file's whole-file hash")
	}
	fl, err := os.OpenFile(req.FlockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return FileApplyResult{}, err
	}
	defer fl.Close()
	if err := syscall.Flock(int(fl.Fd()), syscall.LOCK_EX); err != nil {
		return FileApplyResult{}, err
	}
	raw, err := os.ReadFile(req.File)
	if err != nil {
		return FileApplyResult{}, refuse("reading %s: %v", req.File, err)
	}
	lockText, lockExists, err := readOptional(req.File + ".lock")
	if err != nil {
		return FileApplyResult{}, err
	}
	f := &file{holder: ParseHolder(lockText, lockExists), lockText: lockText}
	if err := f.guard(req.Now); err != nil {
		return FileApplyResult{}, err
	}
	base := string(raw)
	if h := Hash(base); h != req.IfHash {
		return FileApplyResult{}, &Refusal{Exit: ExitStale, Code: "stale", Hash: h, Reason: fmt.Sprintf(
			"it changed since you read it (hash now %s, you passed %s); read it again, rebuild the records, and apply again", h, req.IfHash)}
	}
	edit, err := BuildEdit(base, req.Edits, req.Now, false)
	if err != nil {
		return FileApplyResult{}, err
	}
	res := FileApplyResult{Hash: Hash(edit.Next)}
	if !edit.Changed {
		return res, nil
	}
	body, err := RoundBody(edit.Round, edit.Next)
	if err != nil {
		return FileApplyResult{}, err
	}
	artifact, err := json.Marshal(RoundArtifact{File: req.File, Round: edit.Round, Body: body})
	if err != nil {
		return FileApplyResult{}, err
	}
	if err := os.Remove(req.RoundPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return FileApplyResult{}, refuse("the previous round could not be cleared: %v", err)
	}
	if err := replaceShared(req.File, []byte(edit.Next)); err != nil {
		return FileApplyResult{}, err
	}
	if err := writeAtomic(req.RoundPath, artifact); err != nil {
		return FileApplyResult{}, err
	}
	res.Round, res.Body = &edit.Round, body
	return res, nil
}

// roundRecord is one span as a review record, the format the pane embeds in
// commit bodies and reconstructs from on resume (nvim/review/record.lua).
type roundRecord struct {
	Old           string `json:"old"`
	New           string `json:"new"`
	Explain       string `json:"explain,omitempty"`
	NewOccurrence int    `json:"new_occurrence,omitempty"`
}

// RoundBody is the agent's commit body for a round: a summary line, then a
// fenced review-records block (single-line JSON) of the spans as records,
// each anchored by the Nth plain match of its new text in the result.
func RoundBody(r Round, result string) (string, error) {
	starts := lineStarts(result)
	records := make([]roundRecord, len(r.Spans))
	for i, s := range r.Spans {
		records[i] = roundRecord{Old: s.Old, New: s.New, Explain: s.Explain}
		if s.New != "" {
			records[i].NewOccurrence = occurrenceAt(result, s.New, starts[s.Start[0]]+s.Start[1])
		}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(records); err != nil {
		return "", err
	}
	// The fence is nvim/review/record.lua's OPEN; its extract reads it back.
	var body strings.Builder
	fmt.Fprintf(&body, "agent round: %d change(s)\n\n", len(records))
	fmt.Fprintf(&body, "```review-records\n%s\n```", strings.TrimSuffix(buf.String(), "\n"))
	return body.String(), nil
}

// occurrenceAt is the new_occurrence of needle at offset at, as the pane's
// reconstruct.occurrence_at counts it: plain non-overlapping matches that
// start before at, plus one.
func occurrenceAt(hay, needle string, at int) int {
	n, from := 0, 0
	for needle != "" {
		i := strings.Index(hay[from:], needle)
		if i < 0 || from+i >= at {
			break
		}
		n++
		from += i + len(needle)
	}
	return n + 1
}

// ReadRoundArtifact reads the review pane's round file through the sidecar
// cap: nil, nil when there is none; round-invalid when it is unreadable.
func ReadRoundArtifact(path string) (*RoundArtifact, error) {
	raw, err := readRegular(path, roundMaxBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, roundInvalid(err.Error())
	}
	var a struct {
		File  string          `json:"file"`
		Round json.RawMessage `json:"round"`
		Body  string          `json:"body"`
	}
	if err := json.Unmarshal(raw, &a); err != nil || a.File == "" {
		return nil, roundInvalid("not a round artifact")
	}
	r, err := decodeRound(a.Round)
	if err != nil {
		return nil, err
	}
	return &RoundArtifact{File: a.File, Round: *r, Body: a.Body}, nil
}
