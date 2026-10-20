package historyarchive

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func stagingParent(t *testing.T) (*os.File, string) {
	t.Helper()
	p := t.TempDir()
	f, e := os.Open(p)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { f.Close() })
	return f, p
}
func TestOwnedStagingLifecycle(t *testing.T) {
	p, path := stagingParent(t)
	s, e := OpenOwnedStaging(p, "stage", "history-restore", 20)
	if e != nil {
		t.Fatal(e)
	}
	info, e := os.Stat(filepath.Join(path, "stage"))
	if e != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("mode: %v %v", info, e)
	}
	raw, e := os.ReadFile(filepath.Join(path, "stage", ".owner.json"))
	if e != nil {
		t.Fatal(e)
	}
	var owner StagingOwner
	if e = json.Unmarshal(raw, &owner); e != nil {
		t.Fatal(e)
	}
	if owner.Version != 1 || owner.Namespace != "history-restore" || len(owner.Nonce) != 32 || owner.ParentInode == 0 {
		t.Fatalf("bad owner: %+v", owner)
	}
	if e = os.Mkdir(filepath.Join(path, "stage", "output"), 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(path, "stage", "output", "result"), []byte("secret"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(path, "stage")); !os.IsNotExist(e) {
		t.Fatalf("stage remains: %v", e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
}
func TestOwnedStagingRefusesForeign(t *testing.T) {
	p, path := stagingParent(t)
	stage := filepath.Join(path, "stage")
	if e := os.Mkdir(stage, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(stage, "labels"), []byte("foreign"), 0600); e != nil {
		t.Fatal(e)
	}
	if s, e := OpenOwnedStaging(p, "stage", "history-label-index", 20); e == nil {
		s.Close()
		t.Fatal("foreign stage accepted")
	}
	b, e := os.ReadFile(filepath.Join(stage, "labels"))
	if e != nil || string(b) != "foreign" {
		t.Fatalf("foreign changed: %q %v", b, e)
	}
}
func TestOwnedStagingInterruptedRecovery(t *testing.T) {
	p, path := stagingParent(t)
	s, e := OpenOwnedStaging(p, "stage", "history-restore", 20)
	if e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(filepath.Join(path, "stage", ".owner.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(filepath.Join(path, "stage", "output"), 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(path, "stage", "output", "old"), []byte("old"), 0600); e != nil {
		t.Fatal(e)
	}
	// Simulate an interrupted owner without running cleanup.
	s.Dir.Close()
	s.parent.Close()
	next, e := OpenOwnedStaging(p, "stage", "history-restore", 20)
	if e != nil {
		t.Fatal(e)
	}
	after, e := os.ReadFile(filepath.Join(path, "stage", ".owner.json"))
	if e != nil {
		t.Fatal(e)
	}
	if string(before) == string(after) {
		t.Fatal("recovery reused stale owner nonce")
	}
	if _, e = os.Stat(filepath.Join(path, "stage", "output")); !os.IsNotExist(e) {
		t.Fatalf("old payload remains: %v", e)
	}
	if e = next.Close(); e != nil {
		t.Fatal(e)
	}
}
func TestOwnedStagingRefusalPreservesTree(t *testing.T) {
	for _, kind := range []string{"malformed", "version", "parent", "namespace", "nonce", "duplicate-field", "symlink", "unknown-child", "bound"} {
		t.Run(kind, func(t *testing.T) {
			p, path := stagingParent(t)
			s, e := OpenOwnedStaging(p, "stage", "history-restore", 20)
			if e != nil {
				t.Fatal(e)
			}
			stage := filepath.Join(path, "stage")
			stamp := filepath.Join(stage, ".owner.json")
			raw, e := os.ReadFile(stamp)
			if e != nil {
				t.Fatal(e)
			}
			var fields map[string]any
			if e = json.Unmarshal(raw, &fields); e != nil {
				t.Fatal(e)
			}
			limit := 20
			switch kind {
			case "malformed":
				raw = []byte("{")
			case "version":
				fields["version"] = 2
			case "parent":
				fields["parent_inode"] = 0
			case "namespace":
				fields["namespace"] = "history-label-index"
			case "nonce":
				fields["nonce"] = "short"
			case "duplicate-field":
				raw = []byte(`{"version":1,"version":2}`)
			case "symlink":
				if e = os.Symlink(t.TempDir(), filepath.Join(stage, "output")); e != nil {
					t.Fatal(e)
				}
			case "unknown-child":
				if e = os.WriteFile(filepath.Join(stage, "precious"), []byte("keep"), 0600); e != nil {
					t.Fatal(e)
				}
			case "bound":
				limit = 2
			}
			if kind != "malformed" && kind != "duplicate-field" {
				raw, e = json.Marshal(fields)
				if e != nil {
					t.Fatal(e)
				}
			}
			if e = os.WriteFile(stamp, raw, 0600); e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(filepath.Join(stage, "files-from"), []byte("keep"), 0600); e != nil {
				t.Fatal(e)
			}
			if kind == "bound" {
				if e = os.Mkdir(filepath.Join(stage, "output"), 0700); e != nil {
					t.Fatal(e)
				}
			}
			s.Dir.Close()
			s.parent.Close()
			if next, e := OpenOwnedStaging(p, "stage", "history-restore", limit); e == nil {
				next.Close()
				t.Fatal("unsafe recovery accepted")
			}
			got, e := os.ReadFile(filepath.Join(stage, "files-from"))
			if e != nil || string(got) != "keep" {
				t.Fatalf("preflight deleted data: %q %v", got, e)
			}
			got, e = os.ReadFile(stamp)
			if e != nil || string(got) != string(raw) {
				t.Fatalf("stamp changed: %q %v", got, e)
			}
		})
	}
}
func TestStagingOwnerPreservesUnknownMembers(t *testing.T) {
	raw := []byte(`{"version":1,"namespace":"history-restore","nonce":"0123456789abcdef0123456789abcdef","parent_device":1,"parent_inode":2,"future":{"guard":true}}`)
	var owner StagingOwner
	if e := json.Unmarshal(raw, &owner); e != nil {
		t.Fatal(e)
	}
	got, e := json.Marshal(owner)
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(got, &fields); e != nil {
		t.Fatal(e)
	}
	if string(fields["future"]) != `{"guard":true}` {
		t.Fatalf("unknown lost: %s", got)
	}
}

func TestOwnedStagingRecoveryAllowsUnknownOwnerMembers(t *testing.T) {
	p, path := stagingParent(t)
	s, err := OpenOwnedStaging(p, "stage", "history-label-index", 20)
	if err != nil {
		t.Fatal(err)
	}
	stamp := filepath.Join(path, "stage", ".owner.json")
	raw, err := os.ReadFile(stamp)
	if err != nil {
		t.Fatal(err)
	}
	var owner StagingOwner
	if err = json.Unmarshal(raw, &owner); err != nil {
		t.Fatal(err)
	}
	owner.Extensions = map[string]json.RawMessage{"future": json.RawMessage(`{"opaque":true}`)}
	raw, err = json.Marshal(owner)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(stamp, raw, 0600); err != nil {
		t.Fatal(err)
	}
	s.Dir.Close()
	s.parent.Close()
	next, err := OpenOwnedStaging(p, "stage", "history-label-index", 20)
	if err != nil {
		t.Fatal(err)
	}
	if err = next.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOwnedStagingClosePreservesReplacementDirectory(t *testing.T) {
	p, path := stagingParent(t)
	s, err := OpenOwnedStaging(p, "stage", "history-restore", 20)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(filepath.Join(path, "stage"), filepath.Join(path, "original")); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(path, "stage"), 0700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(path, "stage", "files-from")
	if err = os.WriteFile(foreign, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err == nil {
		t.Fatal("cleanup accepted replacement directory")
	}
	if b, err := os.ReadFile(foreign); err != nil || string(b) != "keep" {
		t.Fatalf("foreign changed: %q %v", b, err)
	}
	if _, err = os.Stat(filepath.Join(path, "original", ".owner.json")); err != nil {
		t.Fatalf("original changed before identity refusal: %v", err)
	}
}
func TestOwnedStagingCloseRefusesUnexpectedContents(t *testing.T) {
	p, path := stagingParent(t)
	s, e := OpenOwnedStaging(p, "stage", "history-restore", 20)
	if e != nil {
		t.Fatal(e)
	}
	foreign := filepath.Join(path, "stage", "foreign")
	if e = os.WriteFile(foreign, []byte("keep"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e == nil {
		t.Fatal("cleanup accepted unknown payload")
	}
	if _, e = os.Stat(foreign); e != nil {
		t.Fatal(e)
	}
}
