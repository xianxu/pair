package historyarchive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"golang.org/x/sys/unix"
)

// Restorer publishes only a fully verified, decoded and validated fresh tree.
// pair:m5-concept integration
type Restorer struct {
	Config  ContinuousConfig
	Decoder Decoder
	Rclone  Rclone
	Now     func() time.Time
	Fault   func(string) error
}

// RestoreReport describes the actual publication state, including uncertain commits.
type RestoreReport struct {
	Bundles, Files        int
	Target                string
	CipherBytes, RawBytes int64
	State                 RestoreState
}

func (r Restorer) fault(phase string) error {
	if r.Fault != nil {
		return r.Fault(phase)
	}
	return nil
}

func (r Restorer) Run(ctx context.Context, req RestoreRequest) (report RestoreReport, err error) {
	ctx, cancel := context.WithTimeout(ctx, drivePhaseBudget)
	defer cancel()
	report.State = RestoreState{Phase: RestorePlanned}
	outcome := commitoutcome.NotAuthoritative
	defer func() {
		if err != nil && report.State.Phase != RestorePublished && report.State.Phase != RestorePublicationUncertain {
			event := RestoreFail
			if ctx.Err() != nil {
				event = RestoreCancel
			}
			if state, e := NextRestore(report.State, event); e == nil {
				report.State = state
			}
		}
	}()
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	view, err := LoadLabelView(ctx, r.Config, now)
	if err != nil {
		return report, err
	}
	plan, err := PlanRestore(view, req)
	if err != nil {
		return report, err
	}
	if r.Decoder == nil {
		return report, errors.New("restore decoder unavailable")
	}
	report.Target = plan.Request.To
	parentPath := filepath.Dir(plan.Request.To)
	if overlaps(parentPath, r.Config.Sink) || overlaps(parentPath, filepath.Dir(r.Config.Catalog)) {
		return report, errors.New("restore scratch parent overlaps archive storage")
	}
	parent, err := openSafeDir(parentPath, false)
	if err != nil {
		return report, err
	}
	defer parent.Close()
	info, err := parent.Stat()
	if err != nil {
		return report, err
	}
	var stat unix.Stat_t
	if err := unix.Fstat(int(parent.Fd()), &stat); err != nil {
		return report, err
	}
	if info.Mode().Perm()&0077 != 0 || stat.Uid != uint32(os.Geteuid()) {
		return report, errors.New("restore requires a private scratch parent owned by the current user")
	}
	lock, err := openRegularAt(parent, ".pair-restore.lock", unix.O_RDWR|unix.O_CREAT)
	if err != nil {
		return report, err
	}
	defer lock.Close()
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return report, fmt.Errorf("restore parent busy; retry: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return report, err
	}
	target := filepath.Base(plan.Request.To)
	var targetStat unix.Stat_t
	if e := unix.Fstatat(int(parent.Fd()), target, &targetStat, unix.AT_SYMLINK_NOFOLLOW); e == nil {
		return report, errors.New("restore target already exists; choose a fresh directory")
	} else if !errors.Is(e, unix.ENOENT) {
		return report, e
	}
	if err = checkRestoreSourceRoots(ctx, r.Config, plan, parentPath); err != nil {
		return report, err
	}
	stage, err := OpenOwnedStaging(parent, ".pair-restore-staging", "history-restore", MaxRestoreEntries)
	if err != nil {
		return report, err
	}
	defer func() {
		if closeErr := stage.Close(); closeErr != nil {
			err = commitoutcome.Join(outcome, err, closeErr)
		}
	}()
	budget := RestoreExtractionBudget{Entries: 3} // stable lock, staging directory and ownership stamp
	cipherDir, err := CreateRestoreDir(stage.Dir, "ciphertext", &budget)
	if err != nil {
		return report, err
	}
	defer cipherDir.Close()
	inputs, err := r.acquireRestoreInputs(ctx, plan, stage, cipherDir, &budget)
	if err != nil {
		return report, err
	}
	defer func() {
		for _, in := range inputs {
			in.File.Close()
		}
	}()
	report.State, err = NextRestore(report.State, RestoreVerifyInputs)
	if err != nil {
		return report, err
	}
	output, err := CreateRestoreDir(stage.Dir, "output", &budget)
	if err != nil {
		return report, err
	}
	defer output.Close()
	report.State, err = NextRestore(report.State, RestoreBeginExtraction)
	if err != nil {
		return report, err
	}
	for _, in := range inputs {
		if err = ctx.Err(); err != nil {
			return report, err
		}
		dir, e := CreateRestoreDir(output, in.Plan.Label.ID, &budget)
		if e != nil {
			return report, e
		}
		childCtx, childCancel := context.WithCancel(ctx)
		plain, writer := io.Pipe()
		done := make(chan error, 1)
		go func(file *os.File) {
			decodeErr := r.Decoder.Decode(childCtx, file, writer)
			writer.CloseWithError(decodeErr)
			done <- decodeErr
		}(in.File)
		_, files, extractErr := ExtractBundle(childCtx, plain, in.Plan.Label, dir, parentPath, plan.Request.Paths, &budget)
		plain.Close()
		if extractErr != nil {
			childCancel()
		}
		decodeErr := <-done
		childCancel()
		closeErr := dir.Close()
		if err = errors.Join(extractErr, decodeErr, closeErr); err != nil {
			return report, fmt.Errorf("bundle %s: %w", in.Plan.Label.ID, err)
		}
		report.Files += files
	}
	if len(plan.Request.Paths) > 0 && report.Files == 0 {
		return report, errors.New("literal restore paths matched no files; choose paths from the archived manifest")
	}
	if err = syncRestoreTree(ctx, output, MaxRestoreEntries); err != nil {
		return report, err
	}
	if err = stage.Dir.Sync(); err != nil {
		return report, err
	}
	if err = checkRestoreParent(parentPath, parent); err != nil {
		return report, err
	}
	report.State, err = NextRestore(report.State, RestoreValidateOutput)
	if err != nil {
		return report, err
	}
	if err = r.fault("before-publish"); err != nil {
		return report, err
	}
	if err = ctx.Err(); err != nil {
		return report, err
	}
	if err = publishRestoreExclusive(stage.Dir, "output", parent, target); err != nil {
		return report, fmt.Errorf("exclusive restore publication refused; target may already exist: %w", err)
	}
	outcome = commitoutcome.Indeterminate
	if err = r.fault("sync-parent"); err == nil {
		err = parent.Sync()
	}
	if err != nil {
		report.State, _ = NextRestore(report.State, RestorePublishUncertain)
		return report, commitoutcome.Wrap(commitoutcome.Indeterminate, fmt.Errorf("complete restore output may exist at %q; inspect before retry: %w", plan.Request.To, err))
	}
	outcome = commitoutcome.Committed
	report.State, err = NextRestore(report.State, RestorePublishOutput)
	if err != nil {
		return report, commitoutcome.Wrap(outcome, err)
	}
	report.Bundles = len(plan.Bundles)
	report.CipherBytes = plan.CipherBytes
	report.RawBytes = plan.RawBytes
	return report, nil
}

func checkRestoreParent(path string, opened *os.File) error {
	current, err := openSafeDir(path, false)
	if err != nil {
		return err
	}
	defer current.Close()
	a, err := opened.Stat()
	if err != nil {
		return err
	}
	b, err := current.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(a, b) {
		return errors.New("restore scratch parent moved or changed")
	}
	return nil
}

func checkRestoreSourceRoots(ctx context.Context, cfg ContinuousConfig, plan RestorePlan, parent string) error {
	c, err := OpenExistingCatalog(ctx, cfg.Catalog)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer c.Close()
	selected := map[string]bool{}
	for _, b := range plan.Bundles {
		selected[b.Label.ID] = true
	}
	return c.Each(ctx, func(record CatalogRecord) error {
		if selected[record.Label.ID] && overlaps(parent, record.Manifest.Source.Root) {
			return errors.New("restore scratch parent overlaps an archived source root")
		}
		return nil
	})
}

func syncRestoreTree(ctx context.Context, root *os.File, remaining int) error {
	return syncRestoreTreeRemaining(ctx, root, &remaining)
}

func syncRestoreTreeRemaining(ctx context.Context, root *os.File, remaining *int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entries, err := root.Readdirnames(*remaining + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) > *remaining {
		return errors.New("restore tree exceeds entry bound")
	}
	*remaining -= len(entries)
	for _, name := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		var st unix.Stat_t
		if err := unix.Fstatat(int(root.Fd()), name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return err
		}
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFDIR:
			child, err := childDir(root, name, false)
			if err != nil {
				return err
			}
			err = syncRestoreTreeRemaining(ctx, child, remaining)
			err = errors.Join(err, child.Close())
			if err != nil {
				return err
			}
		case unix.S_IFREG:
			f, err := openRegularAt(root, name, unix.O_RDONLY)
			if err != nil {
				return err
			}
			err = errors.Join(f.Sync(), f.Close())
			if err != nil {
				return err
			}
		default:
			return errors.New("unsafe restore output entry")
		}
	}
	return root.Sync()
}
