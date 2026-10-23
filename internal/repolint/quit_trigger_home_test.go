package repolint

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestQuitTriggerHomeRule(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"unisolated bound", `func TestQuit(t *testing.T){checker.TriggerBoundQuit(ctx,binding,intent)}`, 1},
		{"unisolated named", `func TestQuit(t *testing.T){checker.TriggerQuit(name,intent)}`, 1},
		{"data root alone", `func TestQuit(t *testing.T){checker:=NewScopedThreadArtifactCollisionChecker(t.TempDir());checker.TriggerQuit(name,intent)}`, 1},
		{"direct private home", `func TestQuit(t *testing.T){t.Setenv("HOME",t.TempDir());checker.TriggerQuit(name,intent)}`, 0},
		{"private local", `func TestQuit(t *testing.T){home:=t.TempDir();t.Setenv("HOME",home);checker.TriggerQuit(name,intent)}`, 0},
		{"private local overwritten", `func TestQuit(t *testing.T){home:=t.TempDir();home=os.Getenv("HOME");t.Setenv("HOME",home);checker.TriggerQuit(name,intent)}`, 1},
		{"home reset", `func TestQuit(t *testing.T){t.Setenv("HOME",t.TempDir());t.Setenv("HOME",os.Getenv("OTHER_HOME"));checker.TriggerQuit(name,intent)}`, 1},
		{"ambient home", `func TestQuit(t *testing.T){t.Setenv("HOME",os.Getenv("HOME"));checker.TriggerQuit(name,intent)}`, 1},
		{"after trigger", `func TestQuit(t *testing.T){checker.TriggerQuit(name,intent);t.Setenv("HOME",t.TempDir())}`, 1},
		{"conditional setup", `func TestQuit(t *testing.T){if enabled {t.Setenv("HOME",t.TempDir())};checker.TriggerQuit(name,intent)}`, 1},
		{"subtest inherits home", `func TestQuit(t *testing.T){t.Setenv("HOME",t.TempDir());t.Run("x",func(t *testing.T){checker.TriggerQuit(name,intent)})}`, 0},
		{"subtest owns home", `func TestQuit(t *testing.T){t.Run("x",func(t *testing.T){t.Setenv("HOME",t.TempDir());checker.TriggerQuit(name,intent)})}`, 0},
		{"subtest cannot isolate parent", `func TestQuit(t *testing.T){t.Run("x",func(t *testing.T){t.Setenv("HOME",t.TempDir())});checker.TriggerQuit(name,intent)}`, 1},
		{"sandbox helper", `func TestQuit(t *testing.T){checker,_:=sandboxedChecker(t,root,sessions);checker.TriggerQuit(name,intent)}`, 0},
		{"native fixture helper inherits caller", `func(d *parkLifecycleDriver) TriggerQuit(){d.checker.TriggerBoundQuit(ctx,binding,intent)}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "fixture_test.go", "package couchcore;"+tc.body, 0)
			if err != nil {
				t.Fatal(err)
			}
			got := quitTriggersWithoutPrivateHome(file)
			if len(got) != tc.want {
				t.Fatalf("findings=%d want=%d", len(got), tc.want)
			}
		})
	}
}

func TestQuitTriggerHomeCatalog(t *testing.T) {
	source := []byte(`package couchcore; func TestQuit(t *testing.T){checker.TriggerBoundQuit(ctx,binding,intent)}`)
	repository := newFakeRepository()
	repository.index["cmd/internal/couchcore/fixture_test.go"] = source
	findings, _, _, err := scanTrackedWithRepository(repository, "repo", func(string) ([]byte, error) { return source, nil })
	if err != nil || len(findings) != 1 {
		t.Fatalf("findings=%v err=%v", findings, err)
	}
	if findings[0].Finding.Rule != "quit-trigger-private-home" || findings[0].Finding.Line <= 0 {
		t.Fatalf("unexpected finding: %+v", findings[0])
	}
}
