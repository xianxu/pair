package couchcmd

import (
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"strings"
	"testing"
)

func TestOSRuntimeRefusesRelativeHostIdentityAuthority(t *testing.T) {
	t.Setenv("HOME", "")
	namespace, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/unused")
	if err != nil {
		t.Fatal(err)
	}
	_, err = (OSRuntime{}).NewCouchWith(nil, namespace)
	if err == nil || !strings.Contains(err.Error(), "absolute HOME") {
		t.Fatalf("err=%v", err)
	}
}
