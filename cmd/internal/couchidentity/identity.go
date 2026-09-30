// Package couchidentity owns Couch's independent store, conversation and terminal identities.
package couchidentity

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

type StoreRegistration struct {
	StorePath string `json:"store_path"`
	C         uint64 `json:"c"`
	LastN     uint64 `json:"last_n"`
	LastM     uint64 `json:"last_m"`
}
type AllocationState struct {
	SchemaVersion int    `json:"schema_version"`
	C             uint64 `json:"c"`
	StorePath     string `json:"store_path"`
	LastN         uint64 `json:"last_n"`
	LastM         uint64 `json:"last_m"`
}
type AllocationRequest struct {
	Conversation, Terminal bool
	RepositoryToken        string
}
type AllocationResult struct {
	C, N, M              uint64
	PairTag, SessionName string
}

func (s AllocationState) Validate() error {
	if s.SchemaVersion != 1 || s.C == 0 || !filepath.IsAbs(s.StorePath) || filepath.Clean(s.StorePath) != s.StorePath {
		return errors.New("invalid allocation snapshot")
	}
	return nil
}
func NormalizeRepositoryToken(raw string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(raw) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		} else {
			dash = true
		}
	}
	return b.String()
}
func FormatPairTag(c uint64, repository string, n uint64) (string, error) {
	token := NormalizeRepositoryToken(repository)
	if c == 0 || n == 0 || token == "" {
		return "", errors.New("pair tag requires positive C/N and a repository token")
	}
	return fmt.Sprintf("%d-%s-%d", c, token, n), nil
}
func FormatSessionName(c, m uint64) (string, error) {
	if c == 0 || m == 0 {
		return "", errors.New("session name requires positive C/M")
	}
	return fmt.Sprintf("📁%d-%d", c, m), nil
}
func AdvanceAllocation(s AllocationState, q AllocationRequest) (AllocationState, AllocationResult, error) {
	fail := func(err error) (AllocationState, AllocationResult, error) { return s, AllocationResult{}, err }
	if err := s.Validate(); err != nil {
		return fail(err)
	}
	if !q.Conversation && !q.Terminal {
		return fail(errors.New("allocation must request a conversation or terminal"))
	}
	if q.Conversation && s.LastN == math.MaxUint64 || q.Terminal && s.LastM == math.MaxUint64 {
		return fail(errors.New("identity counter exhausted"))
	}
	next := s
	r := AllocationResult{C: s.C}
	var err error
	if q.Conversation {
		next.LastN++
		r.N = next.LastN
		r.PairTag, err = FormatPairTag(s.C, q.RepositoryToken, r.N)
		if err != nil {
			return fail(err)
		}
	}
	if q.Terminal {
		next.LastM++
		r.M = next.LastM
		r.SessionName, _ = FormatSessionName(s.C, r.M)
	}
	return next, r, nil
}

func ParsePairTag(tag string) (c uint64, token string, n uint64, err error) {
	first, last := strings.IndexByte(tag, '-'), strings.LastIndexByte(tag, '-')
	if first < 1 || last <= first {
		return 0, "", 0, errors.New("invalid pair tag")
	}
	c, e1 := strconv.ParseUint(tag[:first], 10, 64)
	n, e2 := strconv.ParseUint(tag[last+1:], 10, 64)
	token = tag[first+1 : last]
	formatted, e3 := FormatPairTag(c, token, n)
	if e1 != nil || e2 != nil || e3 != nil || formatted != tag {
		return 0, "", 0, errors.New("invalid pair tag")
	}
	return c, token, n, nil
}
func ParseSessionName(name string) (c, m uint64, err error) {
	parts := strings.Split(strings.TrimPrefix(name, "📁"), "-")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid terminal session name")
	}
	c, e1 := strconv.ParseUint(parts[0], 10, 64)
	m, e2 := strconv.ParseUint(parts[1], 10, 64)
	formatted, e3 := FormatSessionName(c, m)
	if e1 != nil || e2 != nil || e3 != nil || formatted != name {
		return 0, 0, errors.New("invalid terminal session name")
	}
	return c, m, nil
}
