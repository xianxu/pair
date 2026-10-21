package couchschedule

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"reflect"
	"sort"
)

const (
	maxBuildInputBytes   = 64 << 20
	maxBuildInputFiles   = 10000
	maxBuildCommandBytes = 8 << 20
)

type buildInput struct {
	Module, Path string
	Data         []byte
}

type BuildManifest struct {
	Version                      int
	Key, BinarySHA256, Toolchain string
	LastUsed                     int64                      `json:"last_used,omitempty"`
	Unknown                      map[string]json.RawMessage `json:"-"`
	Inputs                       []BuildInputDigest
}

type BuildInputDigest struct {
	Module, Path, SHA256 string
	Unknown              map[string]json.RawMessage `json:"-"`
}

func digestBuild(raw []byte) string {
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

func manifestFor(toolchain string, inputs []buildInput) BuildManifest {
	sort.Slice(inputs, func(i, j int) bool {
		if inputs[i].Module != inputs[j].Module {
			return inputs[i].Module < inputs[j].Module
		}
		return inputs[i].Path < inputs[j].Path
	})
	m := BuildManifest{Version: 1, Toolchain: toolchain}
	for _, in := range inputs {
		m.Inputs = append(m.Inputs, BuildInputDigest{Module: in.Module, Path: in.Path, SHA256: digestBuild(in.Data)})
	}
	m.Key = buildContentKey(m)
	return m
}

type manifestWire BuildManifest
type inputDigestWire BuildInputDigest

var manifestKnown = strictjson.KnownFields(reflect.TypeFor[manifestWire]())
var inputDigestKnown = strictjson.KnownFields(reflect.TypeFor[inputDigestWire]())

func decodeCacheObject(raw []byte, known map[string]bool, target any) (map[string]json.RawMessage, error) {
	owned, unknown, e := strictjson.SplitObject(raw, known, nil)
	if e != nil {
		return nil, e
	}
	for name := range known {
		if name == "last_used" {
			continue
		}
		if _, ok := owned[name]; !ok {
			return nil, fmt.Errorf("missing cache member %s", name)
		}
	}
	raw, e = strictjson.EncodeObject(owned, nil, known)
	if e != nil {
		return nil, e
	}
	return unknown, strictjson.Decode(raw, target)
}
func (i *BuildInputDigest) UnmarshalJSON(raw []byte) error {
	var w inputDigestWire
	u, e := decodeCacheObject(raw, inputDigestKnown, &w)
	if e != nil {
		return e
	}
	*i = BuildInputDigest(w)
	i.Unknown = u
	if i.Module == "" || i.Path == "" || !cacheKey(i.SHA256) {
		return errors.New("invalid cache input digest")
	}
	return nil
}
func (m *BuildManifest) UnmarshalJSON(raw []byte) error {
	var w manifestWire
	u, e := decodeCacheObject(raw, manifestKnown, &w)
	if e != nil {
		return e
	}
	*m = BuildManifest(w)
	m.Unknown = u
	if m.Version != 1 || m.Toolchain == "" || len(m.Inputs) == 0 || len(m.Inputs) > maxBuildInputFiles || !cacheKey(m.Key) || !cacheKey(m.BinarySHA256) || m.LastUsed < 0 {
		return errors.New("invalid build manifest")
	}
	return nil
}
func cacheFields(v any) (map[string]json.RawMessage, error) {
	raw, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	var fields map[string]json.RawMessage
	e = json.Unmarshal(raw, &fields)
	return fields, e
}
func encodeBuildInput(i BuildInputDigest) ([]byte, error) {
	fields, e := cacheFields(inputDigestWire(i))
	if e != nil {
		return nil, e
	}
	return strictjson.EncodeObject(fields, i.Unknown, inputDigestKnown)
}

// encodeBuildManifest composes nested raw values directly: encoding/json would
// compact a Marshaler's opaque members and erase their original representation.
func encodeBuildManifest(m BuildManifest) ([]byte, error) {
	fields, e := cacheFields(manifestWire(m))
	if e != nil {
		return nil, e
	}
	var inputs []byte
	inputs = append(inputs, '[')
	for n, i := range m.Inputs {
		if n > 0 {
			inputs = append(inputs, ',')
		}
		raw, e := encodeBuildInput(i)
		if e != nil {
			return nil, e
		}
		inputs = append(inputs, raw...)
	}
	inputs = append(inputs, ']')
	fields["Inputs"] = inputs
	raw, e := strictjson.EncodeObject(fields, m.Unknown, manifestKnown)
	if len(raw) > 2<<20 {
		return nil, errors.New("cache manifest exceeds byte cap")
	}
	return raw, e
}

// Future metadata never changes the meaning of the owned build recipe. Exclude
// it, the published binary digest and access time from content-key validation.
func buildContentKey(m BuildManifest) string {
	m.Key = ""
	m.BinarySHA256 = ""
	m.LastUsed = 0
	m.Unknown = nil
	m.Inputs = append([]BuildInputDigest(nil), m.Inputs...)
	for n := range m.Inputs {
		m.Inputs[n].Unknown = nil
	}
	raw, _ := json.Marshal(manifestWire(m))
	return digestBuild(raw)
}
func mergeBuildMetadata(current, old BuildManifest) BuildManifest {
	current.Unknown = strictjson.MergeRaw(old.Unknown, current.Unknown)
	previous := map[string]BuildInputDigest{}
	for _, prior := range old.Inputs {
		previous[prior.Module+"\x00"+prior.Path] = prior
	}
	for n := range current.Inputs {
		prior, ok := previous[current.Inputs[n].Module+"\x00"+current.Inputs[n].Path]
		if ok {
			current.Inputs[n].Unknown = strictjson.MergeRaw(prior.Unknown, current.Inputs[n].Unknown)
		}
	}
	return current
}
