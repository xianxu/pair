package operationalrun

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/durablefile"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type fileIdentity struct {
	Unknown  map[string]json.RawMessage `json:"-"`
	Device   uint64                     `json:"device"`
	Inode    uint64                     `json:"inode"`
	Mode     uint32                     `json:"mode"`
	Size     int64                      `json:"size"`
	Modified int64                      `json:"modified"`
}
type bundle struct {
	Unknown   map[string]json.RawMessage `json:"-"`
	Directory fileIdentity               `json:"directory"`
	Record    fileIdentity               `json:"record"`
	Output    fileIdentity               `json:"output"`
}

func identity(path string) (fileIdentity, error) {
	st, e := os.Lstat(path)
	if e != nil {
		return fileIdentity{}, e
	}
	return identityInfo(st)
}
func identityInfo(st os.FileInfo) (fileIdentity, error) {
	if !st.IsDir() && !st.Mode().IsRegular() {
		return fileIdentity{}, errors.New("unsafe operational bundle member")
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return fileIdentity{}, errors.New("operational identity unavailable")
	}
	return fileIdentity{Device: uint64(sys.Dev), Inode: uint64(sys.Ino), Mode: uint32(st.Mode()), Size: st.Size(), Modified: st.ModTime().UnixNano()}, nil
}
func (a fileIdentity) matches(b fileIdentity) bool {
	return a.Device == b.Device && a.Inode == b.Inode && a.Mode == b.Mode && (os.FileMode(a.Mode).IsDir() || a.Size == b.Size && a.Modified == b.Modified)
}
func bundleIdentity(dir string) (bundle, error) {
	var b bundle
	if e := safeDir(dir, false); e != nil {
		return b, e
	}
	ns, e := names(dir)
	if e != nil {
		return b, e
	}
	if len(ns) != 2 || ns[0] != "output" || ns[1] != "record.json" {
		return b, errors.New("operational bundle contains missing or unknown members")
	}
	b.Directory, e = identity(dir)
	if e != nil {
		return b, e
	}
	b.Record, e = identity(filepath.Join(dir, "record.json"))
	if e != nil {
		return b, e
	}
	b.Output, e = identity(filepath.Join(dir, "output"))
	if e != nil {
		return b, e
	}
	if !os.FileMode(b.Record.Mode).IsRegular() || !os.FileMode(b.Output.Mode).IsRegular() {
		return b, errors.New("operational payload not regular")
	}
	return b, nil
}

type collectionJournal struct {
	Unknown  map[string]json.RawMessage `json:"-"`
	Version  int                        `json:"version"`
	Key      RunKey                     `json:"key"`
	Target   string                     `json:"target"`
	Phase    string                     `json:"phase"`
	Identity bundle                     `json:"identity"`
}

func saveJournal(task string, j collectionJournal) error {
	raw, e := encodeJournal(j)
	if e != nil {
		return e
	}
	return durablefile.WriteAtomicStaged(filepath.Join(task, ".collection.json"), raw, filepath.Join(task, ".collection-stage"))
}
func verifyBundle(dir string, want bundle, allowMissing bool) error {
	got, e := identity(dir)
	if e != nil {
		return e
	}
	if !want.Directory.matches(got) {
		return errors.New("operational directory identity changed")
	}
	ns, e := names(dir)
	if e != nil {
		return e
	}
	for _, n := range ns {
		var expected fileIdentity
		switch n {
		case "record.json":
			expected = want.Record
		case "output":
			expected = want.Output
		default:
			return errors.New("unknown detached operational member")
		}
		got, e := identity(filepath.Join(dir, n))
		if e != nil {
			return e
		}
		if !expected.matches(got) {
			return errors.New("operational payload identity changed")
		}
	}
	if !allowMissing && len(ns) != 2 {
		return errors.New("operational payload disappeared")
	}
	return nil
}
func (s Store) recoverCollection(ctx context.Context, task string) error {
	raw, e := readFile(filepath.Join(task, ".collection.json"), RecordLimit)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	var j collectionJournal
	if e = json.Unmarshal(raw, &j); e != nil {
		return e
	}
	expected, e := s.task(j.Key)
	if e != nil || expected != task || j.Version != 1 || len(j.Target) != 32 || !component.MatchString(j.Target) {
		return errors.New("invalid operational collection journal")
	}
	if _, e = hex.DecodeString(j.Target); e != nil {
		return e
	}
	if j.Phase != "prepared" && j.Phase != "detached" && j.Phase != "finalized" {
		return errors.New("invalid operational collection phase")
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	dest := filepath.Join(task, ".quarantine-"+j.Target)
	source := filepath.Join(task, "runs", j.Key.ID)
	if j.Phase == "prepared" {
		if _, e = os.Lstat(dest); errors.Is(e, os.ErrNotExist) {
			if e = verifyBundle(source, j.Identity, false); e != nil {
				return e
			}
			if e = os.Rename(source, dest); e != nil {
				return e
			}
			if e = s.fail("renamed"); e != nil {
				return e
			}
		} else if e != nil {
			return e
		}
		if e = verifyBundle(dest, j.Identity, false); e != nil {
			return e
		}
		if e = errors.Join(s.syncDirectory(filepath.Dir(source)), s.syncDirectory(task)); e != nil {
			return e
		}
		j.Phase = "detached"
		if e = saveJournal(task, j); e != nil {
			return e
		}
		if e = s.fail("detached"); e != nil {
			return e
		}
	}
	// Detached recovery never looks at source, even if its name is reused.
	if j.Phase == "detached" {
		if e = verifyBundle(dest, j.Identity, true); e != nil {
			return e
		}
		for _, name := range []string{"output", "record.json"} {
			if e = ctx.Err(); e != nil {
				return e
			}
			path := filepath.Join(dest, name)
			if e = os.Remove(path); e != nil && !errors.Is(e, os.ErrNotExist) {
				return e
			}
			if e = s.fail("member-" + map[string]string{"output": "output", "record.json": "record"}[name]); e != nil {
				return e
			}
		}
		if e = s.syncDirectory(dest); e != nil {
			return e
		}
		j.Phase = "finalized"
		if e = saveJournal(task, j); e != nil {
			return e
		}
		if e = s.fail("finalized"); e != nil {
			return e
		}
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if _, e = os.Lstat(dest); e == nil {
		if e = verifyBundle(dest, j.Identity, true); e != nil {
			return e
		}
		ns, e := names(dest)
		if e != nil {
			return e
		}
		if len(ns) != 0 {
			return errors.New("finalized operational bundle not empty")
		}
		if e = os.Remove(dest); e != nil {
			return e
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if e = s.syncDirectory(task); e != nil {
		return e
	}
	if e = os.Remove(filepath.Join(task, ".collection.json")); e != nil {
		return e
	}
	return s.syncDirectory(task)
}
func (s Store) collect(ctx context.Context, key RunKey, task string) error {
	if e := s.recoverCollection(ctx, task); e != nil {
		return e
	}
	source := filepath.Join(task, "runs", key.ID)
	b, e := bundleIdentity(source)
	if e != nil {
		return e
	}
	var token [16]byte
	if _, e = rand.Read(token[:]); e != nil {
		return e
	}
	j := collectionJournal{Version: 1, Key: key, Target: hex.EncodeToString(token[:]), Phase: "prepared", Identity: b}
	if e = saveJournal(task, j); e != nil {
		return e
	}
	if e = s.fail("journal"); e != nil {
		return e
	}
	if e = s.recoverCollection(ctx, task); e != nil {
		return fmt.Errorf("operational collection: %w", e)
	}
	return nil
}

type identityWire fileIdentity
type bundleWire bundle
type journalWire collectionJournal

var identityKnown = strictjson.KnownFields(reflect.TypeFor[identityWire]())
var bundleKnown = strictjson.KnownFields(reflect.TypeFor[bundleWire]())
var journalKnown = strictjson.KnownFields(reflect.TypeFor[journalWire]())

func (i *fileIdentity) UnmarshalJSON(raw []byte) error {
	var w identityWire
	u, e := decodeOwned(raw, identityKnown, nil, &w)
	if e != nil {
		return e
	}
	*i = fileIdentity(w)
	i.Unknown = u
	return nil
}
func (b *bundle) UnmarshalJSON(raw []byte) error {
	var w bundleWire
	u, e := decodeOwned(raw, bundleKnown, nil, &w)
	if e != nil {
		return e
	}
	*b = bundle(w)
	b.Unknown = u
	return nil
}
func (j *collectionJournal) UnmarshalJSON(raw []byte) error {
	var w journalWire
	u, e := decodeOwned(raw, journalKnown, nil, &w)
	if e != nil {
		return e
	}
	*j = collectionJournal(w)
	j.Unknown = u
	return nil
}
func ownedFields(v any) (map[string]json.RawMessage, error) {
	raw, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	var fields map[string]json.RawMessage
	e = json.Unmarshal(raw, &fields)
	return fields, e
}
func encodeIdentity(i fileIdentity) ([]byte, error) {
	fields, e := ownedFields(identityWire(i))
	if e != nil {
		return nil, e
	}
	return strictjson.EncodeObject(fields, i.Unknown, identityKnown)
}
func encodeBundle(b bundle) ([]byte, error) {
	fields, e := ownedFields(bundleWire(b))
	if e != nil {
		return nil, e
	}
	for k, i := range map[string]fileIdentity{"directory": b.Directory, "record": b.Record, "output": b.Output} {
		fields[k], e = encodeIdentity(i)
		if e != nil {
			return nil, e
		}
	}
	return strictjson.EncodeObject(fields, b.Unknown, bundleKnown)
}
func encodeJournal(j collectionJournal) ([]byte, error) {
	fields, e := ownedFields(journalWire(j))
	if e != nil {
		return nil, e
	}
	fields["identity"], e = encodeBundle(j.Identity)
	if e != nil {
		return nil, e
	}
	fields["key"], e = j.Key.MarshalJSON()
	if e != nil {
		return nil, e
	}
	raw, e := strictjson.EncodeObject(fields, j.Unknown, journalKnown)
	if len(raw) > RecordLimit {
		return nil, errors.New("operational journal exceeds byte cap")
	}
	return raw, e
}
