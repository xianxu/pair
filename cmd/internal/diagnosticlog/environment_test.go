package diagnosticlog

import (
	"encoding/binary"
	"os"
	"testing"
)

func TestDarwinEnvironmentSeparatesArgumentsFromValues(t *testing.T) {
	b := make([]byte, 4)
	binary.NativeEndian.PutUint32(b, 2)
	b = append(b, []byte("/bin/pair\x00\x00pair\x00PAIR_RETENTION_PROTOCOL=1\x00PAIR_DATA_DIR=/tmp/with spaces\x00PAIR_RETENTION_PROTOCOL=0\x00\x00")...)
	env, e := parseDarwinEnvironment(b)
	if e != nil {
		t.Fatal(e)
	}
	if env["PAIR_RETENTION_PROTOCOL"] != "0" || env["PAIR_DATA_DIR"] != "/tmp/with spaces" {
		t.Fatal(env)
	}
	for _, bad := range [][]byte{nil, {1, 0, 0, 0}, b[:10]} {
		if _, e = parseDarwinEnvironment(bad); e == nil {
			t.Fatal("truncated environment accepted")
		}
	}
}

func TestNativeEnvironmentConformance(t *testing.T) {
	env, e := processEnvironment(os.Getpid())
	if e != nil {
		t.Fatal(e)
	}
	if env["PATH"] == "" {
		t.Fatal("native environment omitted PATH")
	}
}

func FuzzDarwinEnvironmentParser(f *testing.F) {
	f.Add([]byte{1, 0, 0, 0})
	f.Add([]byte("bad"))
	f.Fuzz(func(t *testing.T, b []byte) {
		env, e := parseDarwinEnvironment(b)
		if e == nil {
			for key := range env {
				if key == "" {
					t.Fatal("empty environment key accepted")
				}
			}
		}
	})
}
