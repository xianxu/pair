package vmcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"

	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type provisionLock struct {
	Base  string
	Tools map[string]ToolVersion
}

// Lock readers share the qualified manifest's strict types and lossless provenance.
func decodeProvisionLock(raw []byte) (provisionLock, error) {
	var lock provisionLock
	_, fields, err := strictjson.SplitObject(raw, nil, nil)
	if err != nil {
		return lock, err
	}
	fields["image"] = json.RawMessage(`"pair-conformance-provision"`)
	fields["created_at"] = json.RawMessage(`"2026-10-20T00:00:00Z"`)
	fields["qualified"] = json.RawMessage(`true`)
	encoded, err := json.Marshal(fields)
	if err != nil {
		return lock, err
	}
	m, err := DecodeImageManifest(encoded)
	if err != nil {
		return lock, err
	}
	return provisionLock{m.Base, m.Tools}, nil
}

type imageSaver interface {
	SaveImage(context.Context, string, string) error
}

// Provisioning uses Run's existing lifetime; sealing never runs with a live or
// uncertain VM. A failed operation leaves the active manifest untouched.
type provisionRuntime struct {
	*hostRuntime
	candidate  string
	collected  bool
	cleanGuest bool
	activate   func(context.Context) error
}

func (p *provisionRuntime) Collect(ctx context.Context) error {
	if err := p.hostRuntime.Collect(ctx); err != nil {
		return err
	}
	if p.executeExit != 0 {
		return nil
	}
	if p.cleanGuest {
		work := "/private/tmp/pair-vm-" + p.claim.Nonce
		err := p.secondary(ctx, func() error {
			code, err := p.driver.Execute(ctx, p.claim.VM, []string{"/bin/rm", "-rf", "--", work, p.guestResults}, io.Discard, p.stderrLog)
			if err != nil {
				return err
			}
			if code != 0 {
				return fmt.Errorf("guest provisioning residue removal exit %d", code)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	p.collected = true
	return nil
}
func (p *provisionRuntime) Release(ctx context.Context) error {
	var sealErr error
	if p.collected && p.executed && p.executeExit == 0 && p.stopConfirmed && !p.inputsUnconfirmed {
		saver, ok := p.driver.(imageSaver)
		if !ok {
			sealErr = fmt.Errorf("VM driver cannot seal a provisioned image")
		} else {
			sealErr = p.secondary(ctx, func() error { return saver.SaveImage(ctx, p.claim.VM, p.candidate) })
		}
	}
	if sealErr == nil && p.collected && p.executed && p.executeExit == 0 && p.stopConfirmed && !p.inputsUnconfirmed {
		p.hostRuntime.beforeUnlock = p.activate
	}
	return errors.Join(sealErr, p.hostRuntime.Release(ctx))
}

func readMachineImage(root string) ([]byte, error) {
	dir, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	f, err := dir.Open("image.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return nil, err
	}
	if len(raw) > 65536 {
		return nil, fmt.Errorf("image manifest exceeds 64 KiB")
	}
	return raw, nil
}

func readProvisionFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return nil, err
	}
	if len(raw) > 65536 {
		return nil, fmt.Errorf("provision metadata exceeds 64 KiB")
	}
	return raw, nil
}

// publishProvision requires the caller to hold the machine lane during activation. The
// immutable image has no guest writers and every ordinary run pins its manifest.
func publishProvision(ctx context.Context, root string, m ImageManifest, reported []byte) error {
	expected, err := EncodeImageManifest(m)
	if err != nil {
		return err
	}
	_, fields, err := strictjson.SplitObject(expected, nil, nil)
	if err != nil {
		return err
	}
	fields["tools"] = json.RawMessage(reported)
	reportRaw, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	observed, err := DecodeImageManifest(reportRaw)
	if err != nil {
		return err
	}
	if len(m.Tools) != len(observed.Tools) {
		return fmt.Errorf("provisioned tool population differs from lock")
	}
	for name, pin := range m.Tools {
		got, ok := observed.Tools[name]
		if !ok || got.Version != pin.Version {
			return fmt.Errorf("provisioned %s version differs from lock", name)
		}
		if got.SHA256 != pin.SHA256 {
			var download string
			if json.Unmarshal(got.Unknown["download_sha256"], &download) != nil || download != pin.SHA256 {
				return fmt.Errorf("provisioned %s download checksum differs from lock", name)
			}
		}
		for key, want := range pin.Unknown {
			var x, y any
			if json.Unmarshal(want, &x) != nil || json.Unmarshal(got.Unknown[key], &y) != nil {
				return fmt.Errorf("provisioned %s provenance missing: %s", name, key)
			}
			xb, _ := json.Marshal(x)
			yb, _ := json.Marshal(y)
			if !bytes.Equal(xb, yb) {
				return fmt.Errorf("provisioned %s provenance differs: %s", name, key)
			}
		}
		if digest, ok := pin.Unknown["binary_sha256"]; ok {
			var want string
			if json.Unmarshal(digest, &want) != nil || want != got.SHA256 {
				return fmt.Errorf("provisioned %s binary checksum differs", name)
			}
		}
	}
	m.Tools = observed.Tools
	if err := ctx.Err(); err != nil {
		return err
	}
	// Preserve future top-level provenance from the existing active manifest.
	active, err := readMachineImage(root)
	if err == nil {
		old, err := DecodeImageManifest(active)
		if err != nil {
			return err
		}
		m.Unknown = old.Unknown
		previous, _ := json.Marshal(old.Image)
		if m.Unknown == nil {
			m.Unknown = map[string]json.RawMessage{}
		}
		m.Unknown["previous_image"] = previous
	} else if !os.IsNotExist(err) {
		return err
	}
	raw, err := EncodeImageManifest(m)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(root, ".image-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(raw); err == nil {
		err = file.Sync()
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		return err
	}
	if err := os.Rename(name, filepath.Join(root, "image.json")); err != nil {
		return err
	}
	dir, err := os.Open(root)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

const provisionUsage = `Usage: pair vm provision --checkout PATH

Provision and qualify a fresh pinned machine image in the adhoc lane.
Preserves pair-test and existing cached images. Timeout 60m + cleanup up to 10s.
Requires scripts/vm/provision.sh and image.lock.json in the checkout.
`

func ProvisionCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprint(stdout, provisionUsage)
		return 0
	}
	if len(args) != 2 || args[0] != "--checkout" {
		fmt.Fprint(stderr, provisionUsage)
		return 2
	}
	checkout, err := filepath.Abs(args[1])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	raw, err := readProvisionFile(filepath.Join(checkout, "scripts", "vm", "image.lock.json"))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 125
	}
	lock, err := decodeProvisionLock(raw)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 125
	}
	driver, err := NewNativeDriver()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 125
	}
	account, err := user.Current()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 125
	}
	root := filepath.Join(account.HomeDir, ".tart", "pair-runner")
	// No automatic image pruning: retain the active and previous image, refusing
	// further refreshes until their exact managed inventory is reviewed.
	if prior, e := readMachineImage(root); e == nil {
		old, e := DecodeImageManifest(prior)
		if e != nil {
			fmt.Fprintln(stderr, e)
			return 125
		}
		if len(old.Unknown["previous_image"]) != 0 {
			fmt.Fprintln(stderr, "two qualified images retained; inspect before refreshing again")
			return 125
		}
	} else if !os.IsNotExist(e) {
		fmt.Fprintln(stderr, e)
		return 125
	}
	candidate := "pair-conformance-tahoe-" + time.Now().UTC().Format("20060102T150405.000000000")
	h := newHostRuntime(root, driver, stdout, stderr)
	h.provisionSource = lock.Base
	p := &provisionRuntime{hostRuntime: h, candidate: candidate, cleanGuest: true}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	request := RunRequest{Checkout: checkout, Lane: AdhocLane, Timeout: time.Hour, Args: []string{"/bin/bash", "-c", `set -eu
/bin/bash scripts/vm/provision.sh scripts/vm/image.lock.json > "$TMPDIR/tools.json"
/bin/bash scripts/vm/qualify-image.sh >&2
/bin/cat "$TMPDIR/tools.json"`}}
	p.activate = func(ctx context.Context) error {
		tools, err := readProvisionFile(filepath.Join(h.artifacts, "stdout.log"))
		if err != nil {
			return err
		}
		m := ImageManifest{Version: 1, Image: candidate, Base: lock.Base, CreatedAt: time.Now().UTC(), Tools: lock.Tools, Qualified: true}
		return publishProvision(ctx, root, m, bytes.TrimSpace(tools))
	}
	state := Run(ctx, request, p)
	code := state.ExitCode()
	if err := h.WriteResult(state); err != nil {
		fmt.Fprintln(stderr, err)
		if code == 0 {
			code = 125
		}
	}
	if code != 0 {
		fmt.Fprintln(stderr, state.Detail())
		return code
	}
	fmt.Fprintln(stdout, "Qualified image:", candidate)
	return 0
}
