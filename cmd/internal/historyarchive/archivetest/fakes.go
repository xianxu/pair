// Package archivetest provides stateful, local test doubles. TestCodec does not
// compress or encrypt and must never be used to produce a real backup.
package archivetest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	archive "github.com/xianxu/pair/cmd/internal/historyarchive"
)

// MemorySink keeps published bytes after late failures so a fresh Archiver can
// observe incomplete publications. Configure errors/callbacks only between calls.
// Callbacks run outside the object lock and may call Corrupt.
// pair:m5-concept integration
type MemorySink struct {
	Objects                                    map[string][]byte
	BeforePutError, AfterPutError, VerifyError error
	Barrier                                    func(context.Context, string, string) error
	mu                                         sync.Mutex
}

var _ archive.Sink = (*MemorySink)(nil)

func keyValid(key string) bool {
	return key != "" && key != "." && key != ".." && !strings.HasPrefix(key, "../") && !strings.HasPrefix(key, "/") && !strings.ContainsAny(key, "\\\x00") && path.Clean(key) == key
}
func (s *MemorySink) barrier(ctx context.Context, stage, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Barrier != nil {
		if err := s.Barrier(ctx, stage, key); err != nil {
			return err
		}
	}
	return ctx.Err()
}
func (s *MemorySink) Put(ctx context.Context, key string, r io.Reader, max int64) error {
	if !keyValid(key) || max < 0 || max > archive.MaxCiphertextBytes {
		return errors.New("invalid sink key or bound")
	}
	if err := s.barrier(ctx, "before-put", key); err != nil {
		return err
	}
	if s.BeforePutError != nil {
		return s.BeforePutError
	}
	var data bytes.Buffer
	if err := copyBounded(ctx, &data, r, max); err != nil {
		return err
	}
	s.mu.Lock()
	if _, ok := s.Objects[key]; ok {
		s.mu.Unlock()
		return os.ErrExist
	}
	if s.Objects == nil {
		s.Objects = map[string][]byte{}
	}
	s.Objects[key] = append([]byte(nil), data.Bytes()...)
	s.mu.Unlock()
	if err := s.barrier(ctx, "after-put", key); err != nil {
		return err
	}
	return s.AfterPutError
}
func (s *MemorySink) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !keyValid(key) {
		return nil, errors.New("invalid sink key")
	}
	s.mu.Lock()
	b, ok := s.Objects[key]
	b = append([]byte(nil), b...)
	s.mu.Unlock()
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(&contextReader{ctx: ctx, r: bytes.NewReader(b)}), nil
}
func (s *MemorySink) List(ctx context.Context, prefix string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if prefix != "" && !keyValid(strings.TrimSuffix(prefix, "/")) {
		return nil, errors.New("invalid sink prefix")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var keys []string
	for k := range s.Objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys, nil
}
func (s *MemorySink) Verify(ctx context.Context, key, wantSHA string, wantBytes int64) error {
	if err := s.barrier(ctx, "before-verify", key); err != nil {
		return err
	}
	if s.VerifyError != nil {
		return s.VerifyError
	}
	r, err := s.Get(ctx, key)
	if err != nil {
		return err
	}
	defer r.Close()
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return err
	}
	if n != wantBytes || hex.EncodeToString(h.Sum(nil)) != wantSHA {
		return errors.New("stored object does not match size or checksum")
	}
	return s.barrier(ctx, "after-verify", key)
}
func (s *MemorySink) Corrupt(key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Objects[key]; !ok {
		return os.ErrNotExist
	}
	s.Objects[key] = append([]byte(nil), data...)
	return nil
}

// TestMarker visibly distinguishes test bytes from encrypted production data.
const TestMarker = "PAIR-ARCHIVE-TEST-ONLY-NOT-ENCRYPTED\n"

// TestCodec retains each attempted transform and each fully written output,
// including output whose after-encode callback subsequently reports failure.
// Configure fields between calls; use callbacks for deterministic barriers.
// pair:m5-concept integration
type TestCodec struct {
	KnownRecipients               map[string]bool
	Attempts                      int
	Outputs                       [][]byte
	EncodeError, AfterEncodeError error
	Barrier                       func(context.Context, string) error
}

var _ archive.Codec = (*TestCodec)(nil)

func (c *TestCodec) CheckRecipients(ctx context.Context, recipients []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(recipients) == 0 {
		return errors.New("recipients required")
	}
	for _, r := range recipients {
		if !c.KnownRecipients[strings.ToUpper(r)] {
			return fmt.Errorf("test codec unknown recipient %s", r)
		}
	}
	return nil
}
func (c *TestCodec) barrier(ctx context.Context, stage string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.Barrier != nil {
		if err := c.Barrier(ctx, stage); err != nil {
			return err
		}
	}
	return ctx.Err()
}
func (c *TestCodec) Encode(ctx context.Context, r io.Reader, w io.Writer, recipients []string, compression string) (err error) {
	c.Attempts++
	defer func() {
		if err != nil {
			if closer, ok := r.(io.Closer); ok {
				_ = closer.Close()
			}
		}
	}()
	if err = c.CheckRecipients(ctx, recipients); err != nil {
		return err
	}
	if err = c.barrier(ctx, "before-encode"); err != nil {
		return err
	}
	if c.EncodeError != nil {
		return c.EncodeError
	}
	var retained bytes.Buffer
	out := io.MultiWriter(w, &retained)
	if _, err = io.WriteString(out, TestMarker); err != nil {
		return err
	}
	if err = copyBounded(ctx, out, r, archive.MaxCiphertextBytes-int64(len(TestMarker))); err != nil {
		return err
	}
	c.Outputs = append(c.Outputs, append([]byte(nil), retained.Bytes()...))
	if err = c.barrier(ctx, "after-encode"); err != nil {
		return err
	}
	return c.AfterEncodeError
}
func DecodeTestOutput(b []byte) ([]byte, error) {
	if !bytes.HasPrefix(b, []byte(TestMarker)) {
		return nil, errors.New("not TestCodec output")
	}
	return append([]byte(nil), b[len(TestMarker):]...), nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.r.Read(p)
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		return n, ctxErr
	}
	return n, err
}

// Closing a blocking pipe on cancellation mirrors the real codec's ownership.
func copyBounded(ctx context.Context, w io.Writer, r io.Reader, max int64) error {
	if closer, ok := r.(io.Closer); ok {
		done := make(chan struct{})
		stop := context.AfterFunc(ctx, func() { defer close(done); _ = closer.Close() })
		defer func() {
			if !stop() {
				<-done
			}
		}()
	}
	n, err := io.Copy(w, io.LimitReader(&contextReader{ctx: ctx, r: r}, max+1))
	if err != nil {
		return err
	}
	if n > max {
		return errors.New("test stream exceeds byte limit")
	}
	return ctx.Err()
}
