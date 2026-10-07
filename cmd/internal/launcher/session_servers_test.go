package launcher

import (
	"errors"
	"io/fs"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestParseServerProcessesKeepsTheSocketPath(t *testing.T) {
	raw := "  101 /opt/homebrew/bin/zellij --server /T/zellij-501/contract_version_1/📁1-37\n" +
		"  102 /usr/local/bin/pair wrap\n" +
		"  103 zellij --server /T/zellij-501/contract_version_1/other extra\n" +
		"  104 zellij --server /T/zellij-501/contract_version_1/📁1-38\n" +
		"  junk\n"
	got := ParseServerProcesses(raw)
	want := []SessionServerIdentity{
		{PID: 101, Session: "📁1-37", Socket: "/T/zellij-501/contract_version_1/📁1-37"},
		{PID: 104, Session: "📁1-38", Socket: "/T/zellij-501/contract_version_1/📁1-38"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

// Strategy: any ps text either parses into exact server commands or is
// skipped; it never panics, and what it returns round-trips the exact-argv rule
// the quiescence path signals on.
func FuzzParseServerProcesses(f *testing.F) {
	for _, seed := range []string{
		"  101 /opt/homebrew/bin/zellij --server /T/zellij-501/contract_version_1/📁1-37\n",
		"1 zellij --server /s/a\n2 zellij --server\n3 zellij x /s/b\n",
		"", "\n\n", "-5 zellij --server /s/c\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		for _, s := range ParseServerProcesses(raw) {
			if s.PID <= 0 || !isExactZellijServerCommand("zellij --server "+s.Socket, s.Session) {
				t.Fatalf("parsed a non-server row: %+v", s)
			}
		}
	})
}

func TestObserveSocketTrustsOnlyENOENTAsGone(t *testing.T) {
	socketInfo := fakeFileInfo{mode: fs.ModeSocket}
	cases := []struct {
		name string
		info fs.FileInfo
		err  error
		want SocketState
	}{
		{"socket", socketInfo, nil, SocketPresent},
		{"enoent", nil, &fs.PathError{Op: "lstat", Err: syscall.ENOENT}, SocketGone},
		{"permission", nil, &fs.PathError{Op: "lstat", Err: syscall.EACCES}, SocketUnknown},
		{"other error", nil, errors.New("io"), SocketUnknown},
		{"regular file", fakeFileInfo{}, nil, SocketUnknown},
	}
	for _, tc := range cases {
		if got := ObserveSocket(tc.info, tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

// Strategy: the table over {present, gone, unknown} x {one server, two servers
// for a name}. Only a lone server with a gone socket is an orphan.
func TestClassifyServers(t *testing.T) {
	servers := []SessionServerIdentity{
		{PID: 1, Session: "a", Socket: "/s/a"},
		{PID: 2, Session: "b", Socket: "/s/b"},
		{PID: 3, Session: "c", Socket: "/s/c"},
		{PID: 4, Session: "d", Socket: "/s/d"},
		{PID: 5, Session: "d", Socket: "/s/d"},
	}
	sockets := map[string]SocketState{"/s/a": SocketPresent, "/s/b": SocketGone, "/s/c": SocketUnknown, "/s/d": SocketGone}
	got := ClassifyServers(servers, sockets)
	if s := got["a"]; s.Orphaned || s.Unresolved {
		t.Fatalf("reachable server classified %+v", s)
	}
	if s := got["b"]; !s.Orphaned || s.Server.PID != 2 {
		t.Fatalf("orphan missed %+v", s)
	}
	if s := got["c"]; s.Orphaned || !s.Unresolved {
		t.Fatalf("unknown socket must be unresolved, never orphaned: %+v", s)
	}
	if s := got["d"]; s.Orphaned || !s.Unresolved || !s.Contested {
		t.Fatalf("two servers for one name must be unresolved and contested: %+v", s)
	}
}

type fakeFileInfo struct{ mode fs.FileMode }

func (f fakeFileInfo) Name() string       { return "x" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }
