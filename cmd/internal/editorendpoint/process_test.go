package editorendpoint

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

var _ EndpointProcessSource = procutil.Source{}

func TestClassifyEndpointProcess(t *testing.T) {
	for _, tt := range []struct {
		name, expected, observed string
		err                      error
		want                     EndpointProcessState
	}{
		{name: "same incarnation", expected: "birth", observed: "birth", want: EndpointProcessAlive},
		{name: "PID reused", expected: "birth", observed: "new birth", want: EndpointProcessReplaced},
		{name: "empty observation", expected: "birth", want: EndpointProcessUnknown},
		{name: "initial identity needs establishment", observed: "birth", want: EndpointProcessUnknown},
		{name: "both empty", want: EndpointProcessUnknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyEndpointProcess(tt.expected, tt.observed, tt.err); got != tt.want {
				t.Fatalf("classification = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestClassifyEndpointProcessErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want EndpointProcessState
	}{
		{"ESRCH", syscall.ESRCH, EndpointProcessDead},
		{"not exist", os.ErrNotExist, EndpointProcessDead},
		{"ENOENT alias", syscall.ENOENT, EndpointProcessDead},
		{"EPERM", syscall.EPERM, EndpointProcessUnknown},
		{"EACCES", syscall.EACCES, EndpointProcessUnknown},
		{"EIO", syscall.EIO, EndpointProcessUnknown},
		{"EINVAL", syscall.EINVAL, EndpointProcessUnknown},
		{"unsupported", procutil.ErrUnsupported, EndpointProcessUnknown},
		{"malformed", errors.New("malformed birth stamp"), EndpointProcessUnknown},
		{"generic", errors.New("unknown failure"), EndpointProcessUnknown},
	} {
		for _, observed := range []string{"", "birth", "replacement"} {
			for _, wrapped := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/birth=%q/wrapped=%v", tt.name, observed, wrapped), func(t *testing.T) {
					err := tt.err
					if wrapped {
						err = fmt.Errorf("native identity: %w", err)
					}
					if got := ClassifyEndpointProcess("birth", observed, err); got != tt.want {
						t.Fatalf("classification = %v; want %v", got, tt.want)
					}
				})
			}
		}
	}
}
