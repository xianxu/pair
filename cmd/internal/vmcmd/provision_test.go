package vmcmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProvisionLockPinsBaseAndOfficialTools(t *testing.T) {
	raw := `{"version":1,"base":"ghcr.io/example/base@sha256:` + strings.Repeat("a", 64) + `","tools":{"zellij":{"version":"0.45.1","sha256":"` + strings.Repeat("b", 64) + `","url":"https://example.com/asset","future":{"x":1}}}}`
	lock, err := decodeProvisionLock([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if string(lock.Tools["zellij"].Unknown["future"]) != `{"x":1}` {
		t.Fatal("lost provenance")
	}
	for _, bad := range []string{strings.Replace(raw, "@sha256:", ":latest/", 1), strings.Replace(raw, `"version":1`, `"version":1,"version":1`, 1), strings.Replace(raw, `"tools":{`, `"tools":{},"other":{`, 1)} {
		if _, err := decodeProvisionLock([]byte(bad)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

type provisionFake struct {
	*hostFakeDriver
	saved    bool
	saveErr  error
	saveHook func()
}

func (f *provisionFake) SaveImage(_ context.Context, source, image string) error {
	if !f.exists || f.running || !managedVM(source) || !imageNamePattern.MatchString(image) {
		return errors.New("unsafe save")
	}
	f.saved = true
	if f.saveHook != nil {
		f.saveHook()
	}
	return f.saveErr
}
func TestProvisionRuntimeSealsOnlyQualifiedStoppedVM(t *testing.T) {
	for _, mode := range []string{"success", "command-failure", "collection-failure", "unknown-stop", "save-failure"} {
		t.Run(mode, func(t *testing.T) {
			h, f, r := hostFixture(t)
			p := &provisionFake{hostFakeDriver: f}
			h.driver = p
			if mode == "command-failure" {
				f.commandExit = 7
			}
			if mode == "unknown-stop" {
				f.uncertainStop = true
			}
			if mode == "save-failure" {
				p.saveErr = errors.New("save failed")
			}
			if mode == "collection-failure" {
				f.exportErr = errors.New("export failed")
			}
			rt := &provisionRuntime{hostRuntime: h, candidate: "pair-conformance-test"}
			state := Run(context.Background(), r, rt)
			if (state.ExitCode() == 0) != (mode == "success") {
				t.Fatalf("exit %d: %s", state.ExitCode(), state.Detail())
			}
			if p.saved != (mode == "success" || mode == "save-failure") {
				t.Fatalf("saved=%v", p.saved)
			}
			if mode == "unknown-stop" {
				if _, err := os.Stat(filepath.Join(h.root, "positions", "adhoc", "claim.json")); err != nil {
					t.Fatal("lost claim", err)
				}
			}
		})
	}
}
func TestPublishProvisionRequiresExactToolsAndAtomicManifest(t *testing.T) {
	h, _, _ := hostFixture(t)
	old, err := os.ReadFile(filepath.Join(h.root, "image.json"))
	if err != nil {
		t.Fatal(err)
	}
	m, _ := DecodeImageManifest(old)
	m.Image = "pair-conformance-new"
	m.CreatedAt = time.Now().UTC()
	tools, _ := json.Marshal(m.Tools)
	if err := publishProvision(context.Background(), h.root, m, tools); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(h.root, "image.json"))
	got, err := DecodeImageManifest(raw)
	if err != nil || got.Image != m.Image {
		t.Fatal(string(raw), err)
	}
	before := string(raw)
	if err := publishProvision(context.Background(), h.root, m, []byte(`{}`)); err == nil {
		t.Fatal("published mismatched tools")
	}
	raw, _ = os.ReadFile(filepath.Join(h.root, "image.json"))
	if string(raw) != before {
		t.Fatal("failed refresh changed active image")
	}
}

func TestProvisionCLIHelpAndParseHaveNoNativeEffects(t *testing.T) {
	var out strings.Builder
	if code := ProvisionCLI([]string{"--help"}, &out, &out); code != 0 || !strings.Contains(out.String(), "adhoc") {
		t.Fatal(code, out.String())
	}
	if code := ProvisionCLI([]string{}, &out, &out); code != 2 {
		t.Fatal(code)
	}
}

func TestProvisionActivationKeepsExistingPositionAndCleanupDeadline(t *testing.T) {
	h, f, r := hostFixture(t)
	p := &provisionRuntime{hostRuntime: h, candidate: "pair-conformance-test"}
	h.driver = &provisionFake{hostFakeDriver: f}
	calls := 0
	p.activate = func(ctx context.Context) error {
		calls++
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > cleanupTimeout {
			t.Fatal("activation outside cleanup deadline")
		}
		wait, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		if got, err := AcquirePosition(wait, filepath.Join(h.root, "positions"), AdhocLane, nil); err == nil {
			got.Close()
			t.Fatal("activation released position")
		}
		if _, err := h.stdoutLog.Write([]byte("late")); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("log not closed before activation: %v", err)
		}
		return errors.New("activation failed")
	}
	state := Run(context.Background(), r, p)
	if calls != 1 || state.ExitCode() == 0 {
		t.Fatalf("calls %d exit %d: %s", calls, state.ExitCode(), state.Detail())
	}
}

func TestProvisionActivationSuppressedByFinalValidationAndCloseFailure(t *testing.T) {
	for _, mode := range []string{"validation", "close"} {
		t.Run(mode, func(t *testing.T) {
			h, f, r := hostFixture(t)
			p := &provisionRuntime{hostRuntime: h, candidate: "pair-conformance-test"}
			h.driver = &provisionFake{hostFakeDriver: f}
			f.stopHook = func() {
				if mode == "close" {
					_ = h.stdoutLog.Close()
				} else {
					_ = os.WriteFile(filepath.Join(h.results, "status"), []byte("tampered"), 0600)
				}
			}
			calls := 0
			p.activate = func(context.Context) error { calls++; return nil }
			state := Run(context.Background(), r, p)
			if state.ExitCode() == 0 || calls != 0 {
				t.Fatalf("exit %d activation %d: %s", state.ExitCode(), calls, state.Detail())
			}
		})
	}
}

func TestProvisionCancellationNeverActivates(t *testing.T) {
	for _, stage := range []string{"execute", "export", "stop", "save", "delete", "publish"} {
		t.Run(stage, func(t *testing.T) {
			h, f, r := hostFixture(t)
			old, err := os.ReadFile(filepath.Join(h.root, "image.json"))
			if err != nil {
				t.Fatal(err)
			}
			m, err := DecodeImageManifest(old)
			if err != nil {
				t.Fatal(err)
			}
			m.Image = "pair-conformance-cancelled"
			tools, _ := json.Marshal(m.Tools)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.observe = func(event string) {
				if event == stage {
					cancel()
				}
			}
			driver := &provisionFake{hostFakeDriver: f}
			if stage == "save" {
				driver.saveHook = cancel
			}
			h.driver = driver
			activated := false
			p := &provisionRuntime{hostRuntime: h, candidate: m.Image}
			p.activate = func(cleanup context.Context) error {
				if stage == "publish" {
					cancel()
				}
				err := publishProvision(cleanup, h.root, m, tools)
				activated = err == nil
				return err
			}
			state := Run(ctx, r, p)
			got, err := os.ReadFile(filepath.Join(h.root, "image.json"))
			if err != nil {
				t.Fatal(err)
			}
			if state.ExitCode() == 0 || activated || string(got) != string(old) {
				t.Fatalf("cancel at %s: exit=%d activated=%v manifestChanged=%v: %s", stage, state.ExitCode(), activated, string(got) != string(old), state.Detail())
			}
			if (stage == "execute" || stage == "export" || stage == "stop") && driver.saved {
				t.Fatal("interrupted run sealed image")
			}
		})
	}
}
