package remotesession

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xianxu/pair/cmd/internal/durablefile"
)

const maxAuditSegmentBytes = 16 << 20
const auditRetention = 30 * 24 * time.Hour

// Audit has one owner and one serialized writer. The supervisor must never hold
// its admission lock while waiting here. A write error permanently poisons this
// instance; an absent result remains uncertain, never evidence of no effect.
type Audit struct {
	mu                       sync.Mutex
	activePath, previousPath string
	now                      func() time.Time
	segmentLimit             int
	size                     int
	nextExpiry               time.Time
	blocked                  bool
	publish                  func(string, []byte) error
	appendLine               func(string, []byte) error
}

func OpenAudit(activePath, previousPath string, now func() time.Time) (*Audit, error) {
	if now == nil || filepath.Clean(activePath) == filepath.Clean(previousPath) {
		return nil, errors.New("invalid audit configuration")
	}
	a := &Audit{activePath: activePath, previousPath: previousPath, now: now, segmentLimit: maxAuditSegmentBytes, publish: func(p string, b []byte) error { return durablefile.WriteAtomic(p, b, ".remote-audit-*") }, appendLine: appendAuditLine}
	// Validate both segments completely before publishing a retention sweep.
	if err := a.sweep(); err != nil {
		return nil, err
	}
	return a, nil
}
func readAuditSegment(path string) ([]byte, []AuditRecord, error) {
	b, err := readRemotePrivate(path, maxAuditSegmentBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if len(b) == 0 {
		return b, nil, nil
	}
	if b[len(b)-1] != '\n' {
		return nil, nil, errors.New("truncated audit segment")
	}
	lines := bytes.Split(b[:len(b)-1], []byte("\n"))
	records := make([]AuditRecord, 0, len(lines))
	for _, line := range lines {
		r, err := decodeAuditRecord(line)
		if err != nil {
			return nil, nil, err
		}
		records = append(records, r)
	}
	return b, records, nil
}
func (a *Audit) sweep() error {
	var original [2][]byte
	var kept [2][]byte
	paths := [2]string{a.activePath, a.previousPath}
	next := time.Time{}
	now := a.now()
	for i, p := range paths {
		b, records, err := readAuditSegment(p)
		if err != nil {
			return err
		}
		original[i] = b
		for _, r := range records {
			expiry := r.At.Add(auditRetention)
			if !now.Before(expiry) {
				continue
			}
			if next.IsZero() || expiry.Before(next) {
				next = expiry
			}
			encoded, err := encodeAuditRecord(r)
			if err != nil {
				return err
			}
			kept[i] = append(kept[i], encoded...)
			kept[i] = append(kept[i], '\n')
		}
	}
	for i, p := range paths {
		if !bytes.Equal(original[i], kept[i]) {
			if err := a.publish(p, kept[i]); err != nil {
				return err
			}
		}
	}
	a.size = len(kept[0])
	a.nextExpiry = next
	return nil
}
func (a *Audit) AppendIntent(r AuditRecord) error {
	r.kind = "intent"
	r.Outcome = "pending"
	return a.append(r)
}
func (a *Audit) AppendResult(r AuditRecord) error { r.kind = "result"; return a.append(r) }
func (a *Audit) append(r AuditRecord) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.blocked {
		return errors.New("audit writer blocked")
	}
	raw, err := encodeAuditRecord(r)
	if err != nil {
		return err
	}
	now := a.now()
	if r.At.After(now) || !now.Before(r.At.Add(auditRetention)) {
		return errors.New("audit timestamp outside retention")
	}
	fail := func(err error) error { a.blocked = true; return err }
	if !a.nextExpiry.IsZero() && !now.Before(a.nextExpiry) {
		if err := a.sweep(); err != nil {
			return fail(err)
		}
	}
	line := append(raw, '\n')
	if len(line) > a.segmentLimit {
		return errors.New("audit record exceeds segment capacity")
	}
	if a.size+len(line) > a.segmentLimit {
		b, _, err := readAuditSegment(a.activePath)
		if err != nil {
			return fail(err)
		}
		// Publish the replacement previous first. Any later failure leaves evidence
		// in at least one segment and blocks the writer until an explicit reopen.
		if err := a.publish(a.previousPath, b); err != nil {
			return fail(err)
		}
		if err := a.publish(a.activePath, nil); err != nil {
			return fail(err)
		}
		a.size = 0
	}
	if err := a.appendLine(a.activePath, line); err != nil {
		return fail(err)
	}
	a.size += len(line)
	expiry := r.At.Add(auditRetention)
	if a.nextExpiry.IsZero() || expiry.Before(a.nextExpiry) {
		a.nextExpiry = expiry
	}
	return nil
}
func appendAuditLine(path string, line []byte) error {
	info, err := os.Lstat(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if exists && !info.Mode().IsRegular() {
		return errors.New("unsafe audit path")
	}
	flags := os.O_WRONLY | os.O_APPEND
	if !exists {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return err
	}
	if !opened.Mode().IsRegular() || exists && !os.SameFile(info, opened) {
		return errors.New("audit file changed during open")
	}
	n, err := f.Write(line)
	if err != nil {
		return err
	}
	if n != len(line) {
		return io.ErrShortWrite
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if !exists {
		d, err := os.Open(filepath.Dir(path))
		if err != nil {
			return err
		}
		defer d.Close()
		return d.Sync()
	}
	return nil
}
