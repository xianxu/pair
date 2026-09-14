package launcher

import (
	"errors"
	"fmt"
)

// prepareContinuationRetry selects one exact retained standalone intent. Session
// absence is observed again by the create path; an existing session is never attached.
func prepareContinuationRetry(opts *LaunchOptions, rt Runtime, tag string) error {
	if err := ValidatePairTag(tag); err != nil {
		return err
	}
	index, err := rt.ReadSessionNameIndex()
	if err != nil {
		return err
	}
	scope := scopeKeyFromDataDir(opts.GlobalDataDir, opts.Env.DataDir)
	session := ""
	for _, entry := range index.Entries {
		if entry.Tag == tag && (scope == "" || entry.ScopeKey == scope) {
			if session != "" && session != entry.SessionName {
				return errors.New("continuation retry has ambiguous session identity")
			}
			session = entry.SessionName
		}
	}
	if session == "" {
		return errors.New("continuation retry has no exact Pair session binding")
	}
	sessions, err := rt.SessionLiveness()
	if err != nil {
		return fmt.Errorf("cannot prove continuation source absent: %w", err)
	}
	for _, s := range sessions {
		if s.Name == session && s.State != SessionExited {
			return errors.New("continuation source session is still present; inspect it before retrying")
		}
	}
	marker, present, err := rt.ReadRestartMarker(session)
	if err != nil {
		return err
	}
	if !present {
		return errors.New("no retained continuation restart intent")
	}
	if marker.Version != 1 || marker.Tag != tag || marker.Checkpoint.Validate() != nil {
		return errors.New("retained marker is not a valid exact continuation")
	}
	saved := readSavedConfig(rt, resolveConfigPath(rt, opts.Env.DataDir, tag, marker.Agent))
	saved.SessionID = ""
	plan := planRestart(marker, tag, marker.Agent, saved)
	if err := ValidateFreshAgentArgs(marker.Agent, plan.Args.AgentArgs); err != nil {
		return err
	}
	opts.Args = plan.Args
	opts.ContinueCheckpoint = marker.Checkpoint
	opts.ContinueDoc = marker.Checkpoint.SourcePath
	opts.RestartSession = session
	opts.RestartAttempt = marker
	opts.SkipConfigPicker = true
	return nil
}
