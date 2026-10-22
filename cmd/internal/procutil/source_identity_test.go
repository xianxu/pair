package procutil

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"
)

type identityEvidenceReader struct {
	birth string
	err   error
	calls []int
}

func (*identityEvidenceReader) Table() ([]Process, error) {
	panic("identity must not enumerate the host")
}
func (*identityEvidenceReader) Command(int) (CommandInfo, error) {
	panic("identity must not read commands")
}
func (r *identityEvidenceReader) Identity(pid int) (string, error) {
	r.calls = append(r.calls, pid)
	return r.birth, r.err
}

func TestSourceReadIdentityPreservesEvidence(t *testing.T) {
	terminal := fmt.Errorf("native identity: %w", syscall.ESRCH)
	for _, tt := range []struct {
		name, birth string
		err         error
	}{
		{name: "identity", birth: "birth"},
		{name: "empty successful read"},
		{name: "absent", err: terminal},
		{name: "absent with identity", birth: "birth", err: terminal},
		{name: "missing", err: os.ErrNotExist},
		{name: "permission", err: syscall.EPERM},
		{name: "access", err: syscall.EACCES},
		{name: "io", err: syscall.EIO},
		{name: "invalid native data", err: syscall.EINVAL},
		{name: "unsupported", err: ErrUnsupported},
		{name: "malformed", err: errors.New("malformed identity")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &identityEvidenceReader{birth: tt.birth, err: tt.err}
			s := NewSource(r)
			birth, err := s.ReadIdentity(4242)
			if birth != tt.birth || err != tt.err {
				t.Fatalf("ReadIdentity = %q, %v; want exact evidence %q, %v", birth, err, tt.birth, tt.err)
			}
			if len(r.calls) != 1 || r.calls[0] != 4242 {
				t.Fatalf("identity calls = %v", r.calls)
			}
		})
	}
}

func TestSourceReadIdentityRejectsInvalidPIDBeforeReader(t *testing.T) {
	for _, pid := range []int{0, -1, -4242} {
		t.Run(fmt.Sprint(pid), func(t *testing.T) {
			r := &identityEvidenceReader{birth: "unexpected"}
			birth, err := NewSource(r).ReadIdentity(pid)
			if birth != "" || err == nil {
				t.Fatalf("ReadIdentity(%d) = %q, %v; want validation error", pid, birth, err)
			}
			if len(r.calls) != 0 {
				t.Fatalf("invalid PID reached reader: %v", r.calls)
			}
			birth, err = (Source{}).ReadIdentity(pid)
			if birth != "" || err == nil {
				t.Fatalf("native ReadIdentity(%d) = %q, %v; want validation error", pid, birth, err)
			}
		})
	}
}
