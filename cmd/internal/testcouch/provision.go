package testcouch

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchsingleton"
	"github.com/xianxu/pair/cmd/internal/pathscope"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// Workspaces holds only workspaces qualified by real provisioning and observed
// Git/SDLC/setup evidence. Partial successes remain available after a failure.
type Workspaces struct {
	Primary  string
	Workers  map[int]string
	Evidence map[string]string
}

type provisionRunner func(context.Context, string, []string, string, ...string) (string, error)

// PrepareWorkspaces provisions the owned primary and numbered environments.
// The caller owns the roots and must retain them until all fleet children end.
func PrepareWorkspaces(ctx context.Context, b Binaries, p RootsPlan, env []string, workers int) (Workspaces, error) {
	return prepareWorkspaces(ctx, b, p, env, workers, runProvisionCommand)
}

func prepareWorkspaces(ctx context.Context, b Binaries, p RootsPlan, env []string, workers int, run provisionRunner) (Workspaces, error) {
	w := Workspaces{Workers: map[int]string{}, Evidence: map[string]string{}}
	if workers < 1 || workers > 8 {
		return w, fmt.Errorf("native Couch requires 1–8 workers")
	}
	if err := ctx.Err(); err != nil {
		return w, err
	}
	if env == nil {
		return w, fmt.Errorf("explicit provisioning environment required")
	}
	for _, tool := range []string{b.Git, b.Couch, b.SDLC} {
		if !filepath.IsAbs(tool) {
			return w, fmt.Errorf("absolute provisioning tool required: %q", tool)
		}
	}
	physical, err := physicalRoots(p)
	if err != nil {
		return w, err
	}
	fleet := physical.Mutable["fleet"]
	if fleet == "" {
		return w, fmt.Errorf("owned fleet root required")
	}
	for _, key := range []string{"store", "data", "identity"} {
		if physical.Mutable[key] == "" {
			return w, fmt.Errorf("owned %s root required for public store adoption", key)
		}
	}
	adoptRoots := couchsingleton.Roots{StoreDir: physical.Mutable["store"], PairDataDir: filepath.Join(physical.Mutable["data"], "pair"), IdentityDir: physical.Mutable["identity"]}
	primary := filepath.Join(fleet, "fixture")
	remote := filepath.Join(fleet, "upstream.git")
	for _, path := range []string{primary, remote} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return w, fmt.Errorf("fresh provisioning destination required: %s (%v)", path, err)
		}
	}
	// Each command checks cancellation before invoking the adapter. Once a
	// producer closes, no subsequent setup, Git, or observation command can start.
	command := func(dir, tool string, args ...string) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return run(ctx, dir, append([]string{}, env...), tool, args...)
	}
	git := func(dir string, args ...string) (string, error) {
		out, err := command(dir, b.Git, args...)
		return strings.TrimSpace(out), err
	}
	if err := ctx.Err(); err != nil {
		return w, err
	}
	if err := os.MkdirAll(filepath.Join(primary, "construct"), 0700); err != nil {
		return w, err
	}
	w.Primary = primary
	manifest, err := os.OpenFile(filepath.Join(primary, "construct", "base.manifest"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return w, err
	}
	_, writeErr := manifest.WriteString("# fixture\n")
	closeErr := manifest.Close()
	if writeErr != nil {
		return w, writeErr
	}
	if closeErr != nil {
		return w, closeErr
	}
	for _, step := range []struct {
		dir  string
		args []string
	}{
		{primary, []string{"init", "-b", "main"}},
		{primary, []string{"add", "construct/base.manifest"}},
		{primary, []string{"-c", "user.name=Couch Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "fixture baseline"}},
		{fleet, []string{"clone", "--bare", primary, remote}},
		{primary, []string{"remote", "add", "upstream", remote}},
		{primary, []string{"push", "-u", "upstream", "main"}},
	} {
		if _, err := git(step.dir, step.args...); err != nil {
			return w, err
		}
	}
	baseline, err := git(primary, "rev-parse", "HEAD")
	if err != nil {
		return w, err
	}
	common, err := git(primary, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return w, err
	}
	common, err = pathscope.CanonicalFuture(common)
	if err != nil || !pathscope.Within(physical.Root, common) {
		return w, fmt.Errorf("primary Git admin escapes owned root: %q (%v)", common, err)
	}
	w.Evidence["baseline"] = baseline
	w.Evidence["primary-admin"] = common
	// A fresh public provision invocation reads singleton selection; initialize
	// it through the same preview/digest/apply contract available to an operator.
	// Adoption already emits JSON and deliberately accepts no --json option.
	adoptArgs := []string{"--adopt-store", adoptRoots.StoreDir, "--pair-data", adoptRoots.PairDataDir, "--identity-dir", adoptRoots.IdentityDir}
	preview, previewErr := command(primary, b.Couch, adoptArgs...)
	var report couchsingleton.Report
	if err := strictjson.Decode([]byte(preview), &report); err != nil {
		return w, fmt.Errorf("public adoption preview: %v; decode: %w", previewErr, err)
	}
	if len(report.Blockers) > 0 {
		return w, fmt.Errorf("public adoption blocked: %s", strings.Join(report.Blockers, "; "))
	}
	if previewErr != nil {
		return w, fmt.Errorf("public adoption preview: %w", previewErr)
	}
	_, digestErr := hex.DecodeString(report.Digest)
	if digestErr != nil || len(report.Digest) != 64 || strings.ToLower(report.Digest) != report.Digest || report.Selection.Version != 1 || report.Selection.Roots != adoptRoots || len(report.Selection.Excluded) != 0 {
		return w, fmt.Errorf("public adoption preview does not select exact owned roots with a valid digest")
	}
	selectedRaw, err := command(primary, b.Couch, append(append([]string{}, adoptArgs...), "--apply", report.Digest)...)
	if err != nil {
		return w, fmt.Errorf("public adoption apply: %w", err)
	}
	var selected struct {
		Status    string                   `json:"status"`
		Selection couchsingleton.Selection `json:"selection"`
	}
	if err := strictjson.Decode([]byte(selectedRaw), &selected); err != nil {
		return w, fmt.Errorf("public adoption apply output: %w", err)
	}
	if selected.Status != "SELECTED" || selected.Selection.Version != 1 || selected.Selection.Roots != adoptRoots || len(selected.Selection.Excluded) != 0 {
		return w, fmt.Errorf("public adoption did not confirm exact owned selection")
	}
	w.Evidence["adoption-digest"] = report.Digest
	w.Evidence["adoption-preview"] = preview
	w.Evidence["adoption-selection"] = selectedRaw
	for n := 1; n <= workers; n++ {
		raw, err := command(fleet, b.Couch, "--internal", "provision-workspace", primary, fmt.Sprintf("--slot=%d", n), "--remote=upstream")
		if err != nil {
			return w, fmt.Errorf("provision slot %d: %w", n, err)
		}
		var result couchcore.ProvisionResult
		if err := strictjson.Decode([]byte(raw), &result); err != nil {
			return w, fmt.Errorf("slot %d provisioning result: %w", n, err)
		}
		layout := couchcore.NewSlotLayout(primary, common, n)
		host := layout.Host()
		address, rest := fmt.Sprintf("fixture:%d", n), layout.RestingBranch()
		if result.SchemaVersion != 1 || result.Path != host || result.Address != address || result.RestingBranch != rest || result.BaselineSHA != baseline || result.Disposition != "created" || result.Warning != "" {
			return w, fmt.Errorf("slot %d unexpected provisioning result: %+v", n, result)
		}
		raw, err = command(host, b.SDLC, "workspace", "--json")
		if err != nil {
			return w, err
		}
		id, err := couchcore.ParseWorkspaceIdentity([]byte(raw))
		if err != nil {
			return w, err
		}
		if id.Kind != "slot" || id.Slot == nil || *id.Slot != n || id.Address == nil || *id.Address != address || id.PrimaryRoot != primary || id.WorktreeRoot != host || id.FleetRoot != fleet || id.RepoIdentity != common || id.Head == nil || *id.Head != baseline || id.Branch == nil || *id.Branch != rest {
			return w, fmt.Errorf("slot %d identity does not match owned provisioned workspace", n)
		}
		head, err := git(host, "rev-parse", "HEAD")
		if err != nil {
			return w, err
		}
		upstream, err := git(host, "rev-parse", "--abbrev-ref", "@{upstream}")
		if err != nil {
			return w, err
		}
		if head != baseline || upstream != "upstream/main" {
			return w, fmt.Errorf("slot %d unexpected HEAD/upstream: %s %s", n, head, upstream)
		}
		admin, err := git(host, "rev-parse", "--absolute-git-dir")
		if err != nil {
			return w, err
		}
		admin, err = pathscope.CanonicalFuture(admin)
		if err != nil || !pathscope.Within(common, admin) {
			return w, fmt.Errorf("slot %d Git admin escapes primary: %q (%v)", n, admin, err)
		}
		markerPath := couchcore.SetupMarkerPath(admin)
		if err := ctx.Err(); err != nil {
			return w, err
		}
		var marker couchcore.SetupSuccess
		exists, err := (couchcore.ProvisionStore{}).Read(markerPath, &marker)
		if err != nil {
			return w, err
		}
		if !exists || marker.SchemaVersion != 1 || marker.Host != host || marker.Common != common || marker.Admin != admin || marker.Slot != n || marker.BaselineSHA != baseline {
			return w, fmt.Errorf("slot %d lacks matching actual setup success: %s", n, markerPath)
		}
		w.Workers[n] = host
		w.Evidence[fmt.Sprintf("slot-%d-identity", n)] = raw
		w.Evidence[fmt.Sprintf("slot-%d-setup", n)] = markerPath
		w.Evidence[fmt.Sprintf("slot-%d-upstream", n)] = upstream
	}
	return w, nil
}

// runProvisionCommand never inherits the process environment. Diagnostics and
// machine-readable output are separately bounded so stderr cannot corrupt JSON.
func runProvisionCommand(ctx context.Context, dir string, env []string, tool string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// Even nil means an explicitly empty environment for provisioning.
	runner := couchcore.OSProvisionIO{Env: append([]string{}, env...)}
	raw, err := runner.Run(ctx, couchcore.ProvisionCommand{Dir: dir, Program: tool, Args: append([]string(nil), args...)})
	var output buildOutput
	_, _ = output.Write(raw)
	if err != nil {
		if len(err.Error()) >= buildOutputLimit {
			err = fmt.Errorf("%w\n[diagnostics truncated]", err)
		}
		return output.String(), fmt.Errorf("%s %v: %w", filepath.Base(tool), args, err)
	}
	return output.String(), nil
}
