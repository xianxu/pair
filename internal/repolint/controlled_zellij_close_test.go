package repolint

import (
	"strings"
	"testing"
)

func TestControlledZellijCloseRule(t *testing.T) {
	const fixtureImport = `import fixture "github.com/xianxu/pair/cmd/internal/pairlifecycletest"`
	for _, tc := range []struct {
		name, imports, declarations, body string
		want                              int
	}{
		{"constructor blank", fixtureImport, "", `z, err := fixture.StartControlledZellij(ctx, "x"); _ = err; _ = z.Close()`, 1},
		{"options bare", fixtureImport, "", `z, _ := fixture.StartControlledZellijWithOptions(ctx, "x", opts); z.Close()`, 1},
		{"defer parentheses", fixtureImport, "", `z, _ := (fixture.StartControlledZellij)(ctx, "x"); defer ((z).Close)()`, 1},
		{"go", fixtureImport, "", `z, _ := fixture.StartControlledZellij(ctx, "x"); go z.Close()`, 1},
		{"typed parameter log", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); err != nil { t.Logf("cleanup: %v", err) } }`, "", 1},
		{"typed field range", fixtureImport, `type pane struct { fixture *fixture.ControlledZellij }`, `var panes []pane; for _, pane := range panes { _ = pane.fixture.Close() }`, 1},
		{"typed field receiver", fixtureImport, `type pane struct { fixture *fixture.ControlledZellij }; func (p *pane) stop() { defer p.fixture.Close() }`, "", 1},
		{"composite", fixtureImport, "", `_ = (&fixture.ControlledZellij{}).Close()`, 1},
		{"local alias", fixtureImport, "", `z, _ := fixture.StartControlledZellij(ctx, "x"); other := z; _ = other.Close()`, 1},
		{"typed local", fixtureImport, "", `var z *fixture.ControlledZellij; _ = z.Close()`, 1},
		{"propagate", fixtureImport, `func cleanup(z *fixture.ControlledZellij) error { return z.Close() }`, "", 0},
		{"wrap propagation", fixtureImport, `func cleanup(z *fixture.ControlledZellij) error { if err := z.Close(); err != nil { return fmt.Errorf("cleanup: %w", err) }; return nil }`, "", 0},
		{"join", fixtureImport, `func cleanup(z *fixture.ControlledZellij) error { if closeErr := z.Close(); closeErr != nil { err = errors.Join(err, closeErr) }; return err }`, "", 0},
		{"test error", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); err != nil { t.Errorf("cleanup: %v", err) } }`, "", 0},
		{"test fatal", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); err != nil { t.Fatal(err) } }`, "", 0},
		{"test fail", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); err != nil { t.Log(err); t.Fail() } }`, "", 0},
		{"success only", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); err == nil { t.Log("closed") } }`, "", 1},
		{"success else failure", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); err == nil { t.Log("closed") } else { t.Error(err) } }`, "", 0},
		{"success hides error", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); nil == err { t.Log("closed") } else { t.Log(err) } }`, "", 1},
		{"nil return hides error", fixtureImport, `func cleanup(z *fixture.ControlledZellij) error { if err := z.Close(); err != nil { t.Log(err); return nil }; return nil }`, "", 1},
		{"discarded join", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); err != nil { _ = errors.Join(other, err) } }`, "", 1},
		{"diagnostic errorf", fixtureImport, `func cleanup(z *fixture.ControlledZellij) { if err := z.Close(); err != nil { logger.Errorf("cleanup: %v", err) } }`, "", 1},
		{"unrelated closer", fixtureImport, `func cleanup(z io.Closer) { _ = z.Close(); defer z.Close(); go z.Close() }`, "", 0},
		{"other field", fixtureImport, `type pane struct { fixture io.Closer }; func (p *pane) stop() { _ = p.fixture.Close() }`, "", 0},
		{"shadowed local", fixtureImport, "", `z, _ := fixture.StartControlledZellij(ctx, "x"); { z := pty; _ = z.Close() }; _ = z`, 0},
		{"impostor import", `import fixture "example.com/pairlifecycletest"`, "", `z, _ := fixture.StartControlledZellij(ctx, "x"); _ = z.Close()`, 0},
		{"default import", `import "github.com/xianxu/pair/cmd/internal/pairlifecycletest"`, "", `z, _ := pairlifecycletest.StartControlledZellij(ctx, "x"); _ = z.Close()`, 1},
		{"text only", fixtureImport, "", "// _ = z.Close()\n_ = `fixture.StartControlledZellij(ctx).Close()`", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package example\n" + tc.imports + "\n" + tc.declarations + "\nfunc example(){\n" + tc.body + "\n}")
			repository := newFakeRepository()
			repository.index["cmd/internal/example/cleanup_test.go"] = source
			findings, _, count, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
			if err != nil || count != 1 || len(findings) != tc.want {
				t.Fatalf("findings=%v count=%d err=%v want=%d", findings, count, err, tc.want)
			}
			for _, f := range findings {
				if f.Finding.Rule != "controlled-zellij-close" || !strings.Contains(f.Finding.Message, "pair#764") {
					t.Fatalf("unexpected finding %+v", f)
				}
			}
		})
	}
}
