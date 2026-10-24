package repolint

import (
	"strings"
	"testing"

	"github.com/xianxu/ariadne/pkg/golint"
)

func TestEmptyFIFOHandoffs(t *testing.T) {
	const imports = `package example
import (
 "os"
 "golang.org/x/sys/unix"
)
`
	for _, tc := range []struct {
		name, source string
		want         int
	}{
		{"legacy changelog handoff", imports + `func TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse(t *testing.T) {
 if err := unix.Mkfifo(log.Cleaned, 0600); err != nil { t.Fatal(err) }
 go func() { done <- RunWithEnv(args, env, &stderr) }()
 f, err := os.OpenFile(log.Cleaned, os.O_WRONLY, 0600)
 if err != nil { t.Fatal(err) }
 _ = f.Close()
 if code := <-done; code != 0 { t.Fatal(stderr.String()) }
}`, 1},
		{"scrollback payload", imports + `func helper() { unix.Mkfifo(capture.Raw, 0600); f, err := os.OpenFile(capture.Raw, os.O_WRONLY, 0600); _, _ = f.WriteString("captured\n"); _ = f.Close() }`, 0},
		{"aliased syscall and os", `package example
import (fs "os"; native "syscall")
func helper() { native.Mkfifo(path, 0600); f, err := fs.OpenFile(path, fs.O_WRONLY|fs.O_NONBLOCK, 0600); f.Close() }`, 1},
		{"aliased unix", `package example
import (fs "os"; native "golang.org/x/sys/unix")
func helper() { native.Mkfifo(path, 0600); f, err := fs.OpenFile(path, fs.O_WRONLY, 0600); f.Close() }`, 1},
		{"empty string", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.WriteString(""); f.Close() }`, 1},
		{"empty byte conversion", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Write([]byte("")); f.Close() }`, 1},
		{"empty byte literal", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Write([]byte{}); f.Close() }`, 1},
		{"nil bytes", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Write(nil); f.Close() }`, 1},
		{"nonempty byte literal", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Write([]byte{'\n'}); f.Close() }`, 0},
		{"newline bytes", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Write([]byte("\n")); f.Close() }`, 0},
		{"wrong handle write", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); other.WriteString("\n"); f.Close() }`, 1},
		{"wrong handle close", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); other.Close() }`, 0},
		{"write after close", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Close(); f.WriteString("\n") }`, 1},
		{"conditional payload is no proof", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); if condition { f.WriteString("\n") }; f.Close() }`, 1},
		{"async payload is no proof", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); go f.WriteString("\n"); f.Close() }`, 1},
		{"deferred payload is no proof", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); defer f.WriteString("\n"); f.Close() }`, 1},
		{"shadowed writer", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); { f := other; f.WriteString("\n") }; f.Close() }`, 1},
		{"reassigned writer limitation", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f = other; f.Close() }`, 0},
		{"read write", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_RDWR, 0600); f.Close() }`, 0},
		{"reader", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_RDONLY, 0600); f.Close() }`, 0},
		{"regular file", imports + `func helper() { f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Close() }`, 0},
		{"different path", imports + `func helper() { unix.Mkfifo(fifo, 0600); f, _ := os.OpenFile(regular, os.O_WRONLY, 0600); f.Close() }`, 0},
		{"different function", imports + `func makeFIFO() { unix.Mkfifo(path, 0600) }; func helper() { f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Close() }`, 0},
		{"unrelated import", `package example
import ("os"; unix "example.com/unix")
func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Close() }`, 0},
		{"dynamic payload limitation", imports + `func helper() { unix.Mkfifo(path, 0600); f, _ := os.OpenFile(path, os.O_WRONLY, 0600); f.Write(payload); f.Close() }`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings, err := golint.Check("fixture_test.go", []byte(tc.source), []golint.Rule{{Name: "test-fifo-empty-handoff", Incident: "pair#679", Message: "write a nonempty FIFO payload before closing the writer", Match: emptyFIFOHandoffs}})
			if err != nil || len(findings) != tc.want {
				t.Fatalf("findings=%+v err=%v want=%d", findings, err, tc.want)
			}
			for _, f := range findings {
				if f.Rule != "test-fifo-empty-handoff" || !strings.Contains(f.Message, "pair#679") {
					t.Fatalf("unexpected finding: %+v", f)
				}
			}
		})
	}
}

func TestEmptyFIFOHandoffCatalogScope(t *testing.T) {
	const source = `package example
import ("os"; "golang.org/x/sys/unix")
func helper() { unix.Mkfifo(path, 0600); f, err := os.OpenFile(path, os.O_WRONLY, 0600); f.Close() }`
	for _, tc := range []struct {
		path string
		want int
	}{{"cmd/internal/example/helper_test.go", 1}, {"cmd/internal/example/helper.go", 0}} {
		t.Run(tc.path, func(t *testing.T) {
			findings, err := golint.Check(tc.path, []byte(source), pairCatalogFor(tc.path))
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, f := range findings {
				if f.Rule == "test-fifo-empty-handoff" {
					count++
				}
			}
			if count != tc.want {
				t.Fatalf("catalog findings=%+v; FIFO count=%d want=%d", findings, count, tc.want)
			}
		})
	}
}
