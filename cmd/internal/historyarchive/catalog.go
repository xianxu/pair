package historyarchive

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"golang.org/x/sys/unix"
)

const catalogShardBytes int64 = 64 << 20

// Catalog holds the one cross-process lock for all numbered shards. Records
// are streamed, not retained as a growing in-memory index.
type Catalog struct {
	dir    *os.File
	lock   *os.File
	base   string
	limit  int64
	shards []string
	fault  func(string) error
}

func OpenCatalog(ctx context.Context, path string) (*Catalog, error) {
	return openCatalog(ctx, path, catalogShardBytes)
}
func openCatalog(ctx context.Context, path string, limit int64) (c *Catalog, err error) {
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(path) || filepath.Base(path) == "." || limit <= 0 || limit > catalogShardBytes {
		return nil, errors.New("invalid catalog path or shard limit")
	}
	dir, err := openSafeDir(filepath.Dir(path), true)
	if err != nil {
		return nil, err
	}
	c = &Catalog{dir: dir, base: filepath.Base(path), limit: limit}
	defer func(owned *Catalog) {
		if err != nil {
			owned.Close()
		}
	}(c)
	c.lock, err = openRegularAt(dir, c.base+".lock", unix.O_RDWR|unix.O_CREAT)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(c.lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, fmt.Errorf("catalog busy; retry: %w", err)
	}
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	numbers := map[int]string{}
	for _, name := range names {
		if name == c.base {
			numbers[0] = name
			continue
		}
		if !strings.HasPrefix(name, c.base+".") {
			continue
		}
		suffix := strings.TrimPrefix(name, c.base+".")
		if len(suffix) != 6 {
			continue
		}
		n, e := strconv.Atoi(suffix)
		if e != nil || n < 1 {
			continue
		}
		numbers[n] = name
	}
	if len(numbers) == 0 {
		f, e := openRegularAt(dir, c.base, unix.O_RDWR|unix.O_CREAT|unix.O_EXCL)
		if e != nil {
			return nil, e
		}
		e = f.Sync()
		e = errors.Join(e, f.Close(), dir.Sync())
		if e != nil {
			return nil, commitoutcome.Wrap(commitoutcome.Indeterminate, e)
		}
		numbers[0] = c.base
	}
	for i := 0; i < len(numbers); i++ {
		name, ok := numbers[i]
		if !ok {
			return nil, errors.New("catalog shard gap; inspect catalog")
		}
		c.shards = append(c.shards, name)
	}
	for _, name := range c.shards {
		if err = c.scan(ctx, name, true, nil); err != nil {
			return nil, err
		}
	}
	return c, nil
}
func (c *Catalog) Close() error {
	if c == nil {
		return nil
	}
	var err error
	if c.lock != nil {
		err = errors.Join(err, unix.Flock(int(c.lock.Fd()), unix.LOCK_UN), c.lock.Close())
		c.lock = nil
	}
	if c.dir != nil {
		err = errors.Join(err, c.dir.Close())
		c.dir = nil
	}
	return err
}
func (c *Catalog) sync(f *os.File) error {
	if c.fault != nil {
		if err := c.fault("catalog-sync"); err != nil {
			return commitoutcome.Wrap(commitoutcome.Indeterminate, err)
		}
	}
	if err := f.Sync(); err != nil {
		return commitoutcome.Wrap(commitoutcome.Indeterminate, err)
	}
	if err := c.dir.Sync(); err != nil {
		return commitoutcome.Wrap(commitoutcome.Indeterminate, err)
	}
	return nil
}

// scan repairs only a missing newline tail, never a complete malformed record.
func (c *Catalog) scan(ctx context.Context, name string, repair bool, visit func(CatalogRecord) error) error {
	f, err := openRegularAt(c.dir, name, unix.O_RDWR)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if st.Size() > c.limit {
		return errors.New("catalog shard exceeds writer bound; inspect catalog")
	}
	reader := bufio.NewReader(f)
	var complete int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, e := readCatalogLine(reader)
		if e == io.EOF {
			if len(line) > 0 {
				if !repair {
					return errors.New("catalog changed while locked")
				}
				if err = f.Truncate(complete); err != nil {
					return err
				}
			}
			break
		}
		if e != nil {
			return e
		}
		var record CatalogRecord
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.DisallowUnknownFields()
		if err = dec.Decode(&record); err != nil {
			return fmt.Errorf("complete catalog record invalid; inspect catalog: %w", err)
		}
		var extra any
		if err = dec.Decode(&extra); err != io.EOF {
			return errors.New("trailing complete catalog record data; inspect catalog")
		}
		if err = ValidateRecord(record); err != nil {
			return fmt.Errorf("complete catalog record invalid; inspect catalog: %w", err)
		}
		if visit != nil {
			if err = visit(record); err != nil {
				return err
			}
		}
		complete += int64(len(line))
	}
	return c.sync(f)
}
func readCatalogLine(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		part, err := reader.ReadSlice('\n')
		if len(line)+len(part) > MaxMetadataBytes+1 {
			return nil, errors.New("catalog record exceeds metadata bound")
		}
		line = append(line, part...)
		if err == bufio.ErrBufferFull {
			continue
		}
		return line, err
	}
}
func (c *Catalog) Find(ctx context.Context, id string) (CatalogRecord, bool, error) {
	var found CatalogRecord
	ok := false
	for _, name := range c.shards {
		err := c.scan(ctx, name, false, func(r CatalogRecord) error {
			if r.Label.ID != id {
				return nil
			}
			if ok && !sameCatalogRecord(found, r) {
				return errors.New("conflicting complete catalog records; inspect catalog")
			}
			found = r
			ok = true
			return nil
		})
		if err != nil {
			return CatalogRecord{}, false, err
		}
	}
	return found, ok, nil
}
func sameCatalogRecord(a, b CatalogRecord) bool {
	// Re-verification time is observation metadata, not object identity.
	a.VerifiedAt = b.VerifiedAt
	ar, _ := json.Marshal(a)
	br, _ := json.Marshal(b)
	return bytes.Equal(ar, br)
}
func (c *Catalog) Append(ctx context.Context, record CatalogRecord) error {
	if err := ValidateRecord(record); err != nil {
		return err
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if len(raw) > MaxMetadataBytes+1 || int64(len(raw)) > c.limit {
		return errors.New("catalog record exceeds writer bound")
	}
	old, found, err := c.Find(ctx, record.Label.ID)
	if err != nil {
		return err
	}
	if found {
		if !sameCatalogRecord(old, record) {
			return errors.New("catalog record conflicts with verified object; inspect catalog")
		}
		return nil
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	name := c.shards[len(c.shards)-1]
	f, err := openRegularAt(c.dir, name, unix.O_RDWR|unix.O_APPEND)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	if st.Size() > c.limit-int64(len(raw)) {
		if err = f.Close(); err != nil {
			return err
		}
		if len(c.shards) > 999999 {
			return errors.New("catalog shard namespace exhausted")
		}
		name = fmt.Sprintf("%s.%06d", c.base, len(c.shards))
		f, err = openRegularAt(c.dir, name, unix.O_RDWR|unix.O_APPEND|unix.O_CREAT|unix.O_EXCL)
		if err != nil {
			return err
		}
		c.shards = append(c.shards, name)
		if err = c.sync(f); err != nil {
			f.Close()
			return err
		}
	}
	defer f.Close()
	n, err := f.Write(raw)
	if err != nil {
		return commitoutcome.Wrap(commitoutcome.Indeterminate, err)
	}
	if n != len(raw) {
		return commitoutcome.Wrap(commitoutcome.Indeterminate, io.ErrShortWrite)
	}
	return c.sync(f)
}
