package diagnosticlog

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// One reserved staging name is owned by the log lock. Its inode is recorded in
// Creating before a no-replace hard link can expose it at the current path.
func creationStagePath(path string) string {
	return filepath.Join(directory(path), ".current-creation")
}

func removeCreationStage(path string, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	stage := creationStagePath(path)
	st, err := regular(stage)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if st.Size() != 0 {
		return errors.New("unexpected contents in diagnostic creation stage")
	}
	if err := o.checkContext(); err != nil {
		return err
	}
	if err = os.Remove(stage); err != nil && !os.IsNotExist(err) {
		return err
	}
	return syncDir(directory(path))
}

// recoverCreation never creates a replacement inode for a recorded intent.
// A crash can leave either both hard links or just the published current link.
func recoverCreation(path string, s *diskState, o Options) error {
	if err := o.checkContext(); err != nil {
		return err
	}
	if s.Creating == nil {
		return removeCreationStage(path, o)
	}
	intended := *s.Creating
	stage := creationStagePath(path)
	st, err := regular(path)
	if os.IsNotExist(err) {
		staged, e := regular(stage)
		if e != nil {
			return e
		}
		if e = matches(staged, intended); e != nil {
			return e
		}
		if err = o.checkContext(); err != nil {
			return err
		}
		// Link fails if any file already occupies the final path; never overwrite it.
		if err = os.Link(stage, path); err != nil {
			return err
		}
		if err = syncDir(filepath.Dir(path)); err != nil {
			return err
		}
		if err = fault(o, "creation-published"); err != nil {
			return err
		}
		st, err = regular(path)
	}
	if err != nil {
		return err
	}
	if err = matches(st, intended); err != nil {
		return err
	}
	if staged, e := regular(stage); e == nil {
		if e = matches(staged, intended); e != nil {
			return e
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	if err = o.checkContext(); err != nil {
		return err
	}
	s.Current = intended
	s.Creating = nil
	if err = save(path, *s, true, o); err != nil {
		return err
	}
	if err = fault(o, "creation-finalized"); err != nil {
		return err
	}
	return removeCreationStage(path, o)
}

// ensureCurrent runs under the log lock before append admission. An established
// generation must match exactly; only an uninitialized state may adopt a legacy
// preexisting file. New empty files publish through the recoverable intent.
func ensureCurrent(path string, s *diskState, o Options) error {
	if err := recoverCreation(path, s, o); err != nil {
		return err
	}
	st, err := regular(path)
	if err == nil {
		if s.Current.Identity != (identity{}) {
			return matches(st, s.Current)
		}
		now := o.Now().UTC()
		s.Current = generation{Identity: fileIdentity(st), Start: now, LastWrite: st.ModTime(), Size: st.Size(), ModTime: st.ModTime()}
		return save(path, *s, true, o)
	}
	if !os.IsNotExist(err) {
		return err
	}
	if s.Current.Identity != (identity{}) {
		return errors.New("recorded diagnostic current file is missing")
	}
	if err = o.checkContext(); err != nil {
		return err
	}
	f, err := openRegular(creationStagePath(path), syscall.O_CREAT|syscall.O_EXCL|syscall.O_RDWR)
	if err != nil {
		return err
	}
	st, err = f.Stat()
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = syncDir(directory(path)); err != nil {
		return err
	}
	if err = fault(o, "creation-staged"); err != nil {
		return err
	}
	now := o.Now().UTC()
	intended := generation{Identity: fileIdentity(st), Start: now, LastWrite: now, Size: 0, ModTime: st.ModTime()}
	s.Creating = &intended
	if err = save(path, *s, true, o); err != nil {
		return err
	}
	if err = fault(o, "creation-intent"); err != nil {
		return err
	}
	return recoverCreation(path, s, o)
}
