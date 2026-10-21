package vmcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const maxRunArtifacts int64 = 256 << 20
const maxLaneArtifacts int64 = 4 << 30

// hostRuntime owns one lane's host files and durable launch uncertainty. The
// injected driver owns native child handles; Run owns the ordering of effects.
type hostRuntime struct {
	beforeUnlock    func(context.Context) error
	provisionSource string // private pinned-base input, used only by explicit provisioning

	guestResults                                     string
	stdoutLog, stderrLog                             *runLog
	inputsUnconfirmed                                bool
	artifactsOwned                                   bool
	root                                             string
	driver                                           VMDriver
	stdout, stderr                                   io.Writer
	request                                          RunRequest
	position                                         *Position
	image                                            ImageManifest
	claim                                            LaunchClaim
	claimed, cloneAttempted, stopConfirmed, executed bool
	scratch, input, artifacts, results               string
	snapshot                                         SnapshotReport
	child                                            VMChild
	logs                                             *runLog
	executeExit                                      int
}

func newHostRuntime(root string, driver VMDriver, stdout, stderr io.Writer) *hostRuntime {
	return &hostRuntime{root: root, driver: driver, stdout: stdout, stderr: stderr}
}
func (h *hostRuntime) Acquire(ctx context.Context, r RunRequest) (err error) {
	source, err := filepath.EvalSymlinks(r.Checkout)
	if err != nil {
		return err
	}
	root, err := canonicalFuturePath(h.root)
	if err != nil {
		return err
	}
	if _, err := PlanSnapshot(source, root, nil); err != nil {
		return err
	}
	if r.Artifacts != "" {
		artifacts, err := canonicalFuturePath(r.Artifacts)
		if err != nil {
			return err
		}
		if _, err := PlanSnapshot(source, artifacts, nil); err != nil {
			return err
		}
		if pathOverlap(root, artifacts) {
			return fmt.Errorf("explicit artifacts overlap managed VM storage")
		}
		if _, err := os.Lstat(artifacts); !os.IsNotExist(err) {
			return fmt.Errorf("artifacts destination must be fresh")
		}
		r.Artifacts = artifacts
	}
	h.root = root
	r.Checkout = source
	h.request = r
	if err := os.Mkdir(root, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if !rootInfo.IsDir() || rootInfo.Mode().Perm() != 0700 {
		return fmt.Errorf("VM machine root must be private 0700")
	}
	h.position, err = AcquirePosition(ctx, filepath.Join(root, "positions"), r.Lane, h.stderr)
	if err != nil {
		return err
	}
	acquired := false
	defer func() {
		if !acquired {
			err = errors.Join(err, h.position.Close())
			h.position = nil
		}
	}()
	h.scratch = filepath.Join(h.position.Dir(), "scratch")
	old, exists, err := h.position.ReadClaim()
	if err != nil {
		return err
	}
	if exists {
		if old.Stage != ClaimLaunched {
			return fmt.Errorf("VM lane has unresolved startup/cleanup; inspect %s", old.VM)
		}
		h.claim = old
		h.claimed = true
		err := h.secondary(ctx, func() error {
			proof, err := h.driver.Recover(ctx, old)
			if err != nil {
				return err
			}
			if err := RecoverLaunchClaim(old, proof); err != nil {
				return err
			}
			status, err := h.driver.Inspect(ctx, old.VM)
			if err != nil {
				return err
			}
			if status.Exists {
				return h.driver.Delete(ctx, old.VM)
			}
			return nil
		})
		if err != nil {
			return err
		}
		if err := os.RemoveAll(h.scratch); err != nil {
			return err
		}
		if err := h.position.ClearClaim(old.Nonce); err != nil {
			return err
		}
		h.claimed = false
	} else {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return err
		}
		h.claim = LaunchClaim{Version: 1, Lane: r.Lane, VM: ManagedVMName(r.Lane), Nonce: hex.EncodeToString(nonce[:]), Stage: ClaimLaunching}
		h.claimed = true
		err := h.secondary(ctx, func() error {
			status, err := h.driver.Inspect(ctx, h.claim.VM)
			if err != nil {
				return err
			}
			if status.Exists {
				return fmt.Errorf("managed VM %s exists without ownership; inspect before reuse", h.claim.VM)
			}
			return nil
		})
		if !h.inputsUnconfirmed {
			clearErr := h.position.ClearClaim(h.claim.Nonce)
			h.claimed = clearErr != nil
			err = errors.Join(err, clearErr)
		}
		if err != nil {
			return err
		}
		if _, err := os.Lstat(h.scratch); !os.IsNotExist(err) {
			return fmt.Errorf("unclaimed VM scratch exists; inspect before reuse")
		}
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	h.claim = LaunchClaim{Version: 1, Lane: r.Lane, VM: ManagedVMName(r.Lane), Nonce: hex.EncodeToString(nonce[:]), Stage: ClaimLaunching}
	acquired = true
	return nil
}
func (h *hostRuntime) Prepare(ctx context.Context) error {
	var err error
	if h.provisionSource != "" {
		if !baseReferencePattern.MatchString(h.provisionSource) {
			return fmt.Errorf("provision base must be digest-pinned")
		}
		h.image = ImageManifest{Image: h.provisionSource, Base: h.provisionSource}
	} else {
		raw, readErr := readMachineImage(h.root)
		if readErr != nil {
			return fmt.Errorf("qualified VM image unavailable: %w", readErr)
		}
		h.image, err = DecodeImageManifest(raw)
		if err != nil {
			return err
		}
	}
	if h.request.Artifacts != "" {
		h.artifacts = h.request.Artifacts
	} else {
		parent := filepath.Join(h.position.Dir(), "artifacts")
		if err := reserveOrdinaryArtifacts(ctx, parent); err != nil {
			return err
		}
		h.artifacts = filepath.Join(parent, h.claim.Nonce)
	}
	if err := os.Mkdir(h.artifacts, 0700); err != nil {
		return err
	}
	h.artifactsOwned = true
	h.results = filepath.Join(h.artifacts, "guest")
	if err := os.Mkdir(h.results, 0700); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(h.artifacts, "run.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	budget := &artifactBudget{remaining: maxRunArtifacts - maxGuestExport - maxResultRecord}
	h.logs = &runLog{file: log, budget: budget}
	for _, item := range []struct {
		name   string
		target **runLog
	}{{"stdout.log", &h.stdoutLog}, {"stderr.log", &h.stderrLog}} {
		file, err := os.OpenFile(filepath.Join(h.artifacts, item.name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		*item.target = &runLog{file: file, budget: budget}
	}
	fmt.Fprintf(h.stderr, "VM artifacts: %s\n", h.artifacts)
	if err := os.Mkdir(h.scratch, 0700); err != nil {
		return err
	}
	h.input = filepath.Join(h.scratch, "input")
	if err := os.Mkdir(h.input, 0700); err != nil {
		return err
	}
	h.claimed = true
	if err := h.position.WriteClaim(h.claim); err != nil {
		return err
	}
	h.snapshot, err = SnapshotCheckout(ctx, h.request.Checkout, filepath.Join(h.input, "checkout"))
	if err != nil {
		h.inputsUnconfirmed = errors.Is(err, ErrUnconfirmedProcessCleanup)
		return err
	}
	if err := os.WriteFile(filepath.Join(h.input, "guest-run.sh"), GuestScript, 0600); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	h.cloneAttempted = true
	err = h.driver.Clone(ctx, h.image.Image, h.claim.VM)
	h.inputsUnconfirmed = errors.Is(err, ErrUnconfirmedProcessCleanup)
	return err
}
func (h *hostRuntime) Start(ctx context.Context) error {
	child, launchErr := h.driver.Launch(ctx, h.claim.VM, h.input, h.logs)
	h.child = child
	if child != nil {
		h.claim.Stage = ClaimLaunched
		h.claim.PID = child.PID()
		h.claim.StartIdentity = child.Identity()
		launchErr = errors.Join(launchErr, h.position.WriteClaim(h.claim))
	}
	return launchErr
}
func (h *hostRuntime) AwaitGuest(ctx context.Context) error {
	return h.secondary(ctx, func() error { return h.driver.AwaitGuest(ctx, h.claim.VM) })
}
func (h *hostRuntime) Execute(ctx context.Context) (int, error) {
	h.guestResults = "/private/tmp/pair-vm-results-" + h.claim.Nonce
	invocation, err := BuildGuestInvocation(h.request, GuestPaths{Input: "/Volumes/My Shared Files/input", Results: h.guestResults, WorkRoot: "/private/tmp/pair-vm-" + h.claim.Nonce})
	if err != nil {
		return -1, err
	}
	h.executed = true
	err = h.secondary(ctx, func() error {
		var operationErr error
		h.executeExit, operationErr = h.driver.Execute(ctx, h.claim.VM, invocation.Args, h.stdoutLog, h.stderrLog)
		return operationErr
	})
	return h.executeExit, err
}
func (h *hostRuntime) Collect(ctx context.Context) error {
	for _, log := range []*runLog{h.logs, h.stdoutLog, h.stderrLog} {
		if log != nil {
			if err := log.Err(); err != nil {
				return err
			}
		}
	}
	if !h.executed {
		return nil
	}
	if err := h.secondary(ctx, func() error {
		return h.driver.ExportResults(ctx, h.claim.VM, h.guestResults, h.results, maxGuestExport)
	}); err != nil {
		return err
	}
	return h.validateGuestStatus()
}
func (h *hostRuntime) validateGuestStatus() error {
	root, err := os.OpenRoot(h.results)
	if err != nil {
		return err
	}
	defer root.Close()
	status, err := root.Open("status")
	if err != nil {
		return fmt.Errorf("guest result unconfirmed: %w", err)
	}
	raw, readErr := io.ReadAll(io.LimitReader(status, 129))
	closeErr := status.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}
	expected := fmt.Sprintf("exit_code=%d\ncollection_error=0\n", h.executeExit)
	if len(raw) > 128 || string(raw) != expected {
		return fmt.Errorf("guest status missing, inconsistent or collection failed")
	}
	return nil
}
func (h *hostRuntime) Stop(ctx context.Context) error {
	if !h.cloneAttempted {
		h.stopConfirmed = true
		return nil
	}
	err := h.secondary(ctx, func() error { return h.driver.Stop(ctx, h.claim.VM, h.child) })
	h.stopConfirmed = err == nil
	return err
}
func (h *hostRuntime) Release(ctx context.Context) (err error) {
	if h.position == nil {
		return nil
	}
	defer func() { err = errors.Join(err, h.position.Close()); h.position = nil }()
	defer func() {
		for _, log := range []*runLog{h.logs, h.stdoutLog, h.stderrLog} {
			if log != nil {
				err = errors.Join(err, log.Close())
			}
		}
		if err == nil && h.beforeUnlock != nil {
			err = h.beforeUnlock(ctx)
		}
	}()
	if h.inputsUnconfirmed || (h.claimed && !h.stopConfirmed) {
		return fmt.Errorf("VM cleanup unconfirmed; lane remains claimed: inspect %s (inputs %s)", h.claim.VM, h.scratch)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("cleanup expired; lane remains claimed: inspect %s: %w", h.claim.VM, err)
	}
	if h.cloneAttempted {
		if err := h.secondary(ctx, func() error { return h.driver.Delete(ctx, h.claim.VM) }); err != nil {
			return fmt.Errorf("VM removal failed; lane remains claimed: %w", err)
		}
	}
	var validationErr error
	for _, log := range []*runLog{h.logs, h.stdoutLog, h.stderrLog} {
		if log != nil {
			validationErr = errors.Join(validationErr, log.Err())
		}
	}
	if h.artifactsOwned {
		_, err := boundedArtifactSize(ctx, h.artifacts, maxRunArtifacts-maxResultRecord)
		validationErr = errors.Join(validationErr, err)
	}
	if h.executed {
		validationErr = errors.Join(validationErr, h.validateGuestStatus())
	}
	if h.scratch != "" {
		if err := os.RemoveAll(h.scratch); err != nil {
			return err
		}
	}
	if h.claimed {
		if err := h.position.ClearClaim(h.claim.Nonce); err != nil {
			return err
		}
		h.claimed = false
	}
	return validationErr
}

// WriteResult runs after the reducer's terminal state, when the native lifetime
// is either confirmed stopped or explicitly retained. Guest shares cannot write
// the parent bundle, so the final host result is never guest-supplied authority.
func (h *hostRuntime) WriteResult(state RunState) error {
	if !h.artifactsOwned {
		return nil
	}
	result := struct {
		Version  int            `json:"version"`
		Exit     int            `json:"exit_code"`
		Detail   string         `json:"detail"`
		Envelope string         `json:"timeout_envelope"`
		Image    ImageManifest  `json:"image"`
		Snapshot SnapshotReport `json:"snapshot"`
		Occupied bool           `json:"lane_claim_retained"`
	}{1, state.ExitCode(), state.Detail(), fmt.Sprintf("timeout %s + cleanup up to 10 s", h.request.Timeout), h.image, h.snapshot, h.claimed}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if len(raw) > 65536 {
		return fmt.Errorf("VM result exceeds 64 KiB")
	}
	file, err := os.OpenFile(filepath.Join(h.artifacts, "result.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(raw)
	return errors.Join(writeErr, file.Close())
}

type artifactBudget struct {
	mu        sync.Mutex
	remaining int64
}
type runLog struct {
	mu     sync.Mutex
	file   *os.File
	budget *artifactBudget
	err    error
}

func (w *runLog) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return 0, w.err
	}
	w.budget.mu.Lock()
	defer w.budget.mu.Unlock()
	if int64(len(p)) > w.budget.remaining {
		w.err = fmt.Errorf("VM artifact byte budget exceeded")
		return 0, w.err
	}
	n, err := w.file.Write(p)
	w.budget.remaining -= int64(n)
	w.err = err
	return n, err
}
func (w *runLog) Err() error   { w.mu.Lock(); defer w.mu.Unlock(); return w.err }
func (w *runLog) Close() error { w.mu.Lock(); defer w.mu.Unlock(); return w.file.Close() }

func boundedArtifactSize(ctx context.Context, path string, limit int64) (int64, error) {
	var size int64
	files := 0
	err := filepath.WalkDir(path, func(_ string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		files++
		if files > 100000 {
			return fmt.Errorf("VM artifact entry budget exceeded")
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("VM artifacts contain symlink")
		}
		if !e.IsDir() {
			info, err := e.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("VM artifacts contain non-file")
			}
			if info.Size() > limit-size {
				return fmt.Errorf("VM artifacts exceed byte budget")
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}
func reserveOrdinaryArtifacts(ctx context.Context, parent string) error {
	if err := os.Mkdir(parent, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	type bundle struct {
		path     string
		bytes    int64
		at       time.Time
		complete bool
	}
	bundles := make([]bundle, 0, len(entries))
	var total int64
	for _, entry := range entries {
		if !entry.IsDir() || !claimNonce.MatchString(entry.Name()) {
			return fmt.Errorf("unknown VM artifact residue; inspect %s", parent)
		}
		path := filepath.Join(parent, entry.Name())
		size, err := boundedArtifactSize(ctx, path, maxLaneArtifacts-total)
		if err != nil {
			return err
		}
		total += size
		info, err := entry.Info()
		if err != nil {
			return err
		}
		_, err = os.Lstat(filepath.Join(path, "result.json"))
		complete := err == nil
		bundles = append(bundles, bundle{path, size, info.ModTime(), complete})
	}
	sort.Slice(bundles, func(i, j int) bool { return bundles[i].at.Before(bundles[j].at) })
	remaining := len(bundles)
	for _, bundle := range bundles {
		if remaining < 8 && total <= maxLaneArtifacts-maxRunArtifacts {
			return nil
		}
		if !bundle.complete {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.RemoveAll(bundle.path); err != nil {
			return err
		}
		remaining--
		total -= bundle.bytes
	}
	if remaining >= 8 || total > maxLaneArtifacts-maxRunArtifacts {
		return fmt.Errorf("VM artifact retention saturated; inspect/archive %s", parent)
	}
	return nil
}

// Persist uncertainty before every secondary process, including the crash gap.
// Only confirmed completion restores the prior recovery authority.
func (h *hostRuntime) secondary(ctx context.Context, operation func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	previous := h.claim.Stage
	h.claim.Stage = ClaimCleanupUnknown
	if err := h.position.WriteClaim(h.claim); err != nil {
		h.inputsUnconfirmed = true
		return err
	}
	err := operation()
	if errors.Is(err, ErrUnconfirmedProcessCleanup) {
		h.inputsUnconfirmed = true
	}
	if h.inputsUnconfirmed {
		return err
	}
	h.claim.Stage = previous
	if writeErr := h.position.WriteClaim(h.claim); writeErr != nil {
		h.inputsUnconfirmed = true
		h.claim.Stage = ClaimCleanupUnknown
		return errors.Join(err, writeErr)
	}
	return err
}
