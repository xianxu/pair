package launcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const CouchSessionFlag = "--couch-session-v1"
const CouchSessionIntentEnv = "PAIR_COUCH_SESSION_INTENT"

// CouchSessionIntent is terminal authority, independent of the native agent
// profile. A warm attach carries this intent without a cold launch profile.
type CouchSessionIntent struct {
	Scope       string `json:"scope"`
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	Nonce       string `json:"nonce"`
	Disposition string `json:"disposition"`
}

func (intent CouchSessionIntent) Validate() error {
	if err := ValidateRepoScopeKey(intent.Scope); err != nil {
		return fmt.Errorf("couch session scope: %w", err)
	}
	if err := ValidatePairTag(intent.Tag); err != nil {
		return fmt.Errorf("couch session tag: %w", err)
	}
	if err := couchidentity.ValidateSessionName(intent.Name); err != nil {
		return fmt.Errorf("couch session: %w", err)
	}
	if err := couchidentity.ValidateStartNonce(intent.Nonce); err != nil {
		return fmt.Errorf("couch session: %w", err)
	}
	if intent.Disposition != "create" && intent.Disposition != "attach" {
		return errors.New("invalid couch session disposition")
	}
	return nil
}

func applyCouchSessionIntent(args LaunchArgs, raw string) (LaunchArgs, error) {
	if !args.CouchSessionV1 && raw == "" {
		return args, nil
	}
	if !args.CouchSessionV1 || raw == "" {
		return LaunchArgs{}, errors.New("couch session requires both protocol flag and intent")
	}
	var intent CouchSessionIntent
	if len(raw) > 4096 {
		return LaunchArgs{}, errors.New("couch session intent exceeds size limit")
	}
	if err := strictjson.Decode([]byte(raw), &intent); err != nil {
		return LaunchArgs{}, fmt.Errorf("couch session intent: %w", err)
	}
	if err := intent.Validate(); err != nil {
		return LaunchArgs{}, err
	}
	if args.ForcedTag != intent.Tag || args.Command != "" {
		return LaunchArgs{}, errors.New("couch session intent does not match selected tag")
	}
	if intent.Disposition == "attach" && args.Layout.Explicit {
		return LaunchArgs{}, errors.New("couch session attach cannot request a layout change")
	}
	args.CouchSession = &intent
	return args, nil
}

func validateCouchSessionLaunch(args LaunchArgs, env Env) error {
	if !args.CouchSessionV1 && args.CouchSession == nil {
		return nil
	}
	if !args.CouchSessionV1 || args.CouchSession == nil {
		return errors.New("couch session requires both protocol flag and intent")
	}
	intent := args.CouchSession
	if err := intent.Validate(); err != nil {
		return err
	}
	if args.ForcedTag != intent.Tag || args.Command != "" || (args.SelectedTag != "" && args.SelectedTag != intent.Tag) {
		return errors.New("couch session selected address changed")
	}
	scope, err := ResolveRepoScope(envScopeRoot(env))
	if err != nil || scope.Key != intent.Scope {
		return errors.New("couch session repository scope changed")
	}
	if env.CouchThreadScope != intent.Scope || env.CouchThreadTag != intent.Tag {
		return errors.New("couch session does not match hosted thread address")
	}
	if intent.Disposition == "attach" && (args.Layout.Explicit || args.ResumeRequired || args.FreshRequired || args.AgentArgsFromCouch) {
		return errors.New("couch session attach cannot carry a cold launch profile or layout")
	}
	return nil
}

func sessionNamePresent(live []Session, name string) bool {
	for _, session := range live {
		if session.Name == name {
			return true
		}
	}
	return false
}

func assignCouchSessionName(rt Runtime, live []Session, root, tag string, intent *CouchSessionIntent) (string, SessionNameEntry, error) {
	if err := intent.Validate(); err != nil {
		return "", SessionNameEntry{}, err
	}
	scope, err := ResolveRepoScope(root)
	if err != nil || scope.Key != intent.Scope || tag != intent.Tag {
		return "", SessionNameEntry{}, errors.New("couch session selected address changed")
	}
	occupied := sessionNamePresent(live, intent.Name)
	if intent.Disposition == "create" {
		if occupied {
			return "", SessionNameEntry{}, fmt.Errorf("couch session name %q is occupied", intent.Name)
		}
		if err := rt.ProbeSessionName(intent.Name); err != nil {
			return "", SessionNameEntry{}, fmt.Errorf("couch session exact name %q is not accepted: %w", intent.Name, err)
		}
	} else {
		found := false
		for _, session := range live {
			if session.Name == intent.Name && session.State != SessionExited {
				found = true
			}
		}
		if !found {
			return "", SessionNameEntry{}, errors.New("couch session disappeared; refusing attach-to-create fallback")
		}
	}
	return intent.Name, SessionNameEntry{SessionName: intent.Name, ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: tag}, nil
}

type couchSessionOwnerRuntime interface {
	ObserveSessionOwner(context.Context, string, string, string, string) (SessionOwnerObservation, error)
	RevalidateSessionOwner(context.Context, SessionOwnerObservation) error
}

func verifyCouchSessionAttach(rt Runtime, globalDataDir string, intent CouchSessionIntent) (SessionOwnerObservation, error) {
	observer, ok := rt.(couchSessionOwnerRuntime)
	if !ok {
		return SessionOwnerObservation{}, errors.New("couch session owner observer unavailable")
	}
	observation, err := observer.ObserveSessionOwner(context.Background(), intent.Name, globalDataDir, intent.Scope, intent.Tag)
	if err != nil {
		return observation, err
	}
	if observation.State != SessionOwnerOwned {
		return observation, fmt.Errorf("couch session ownership is not proven: %s", observation.Diagnostic)
	}
	return observation, observer.RevalidateSessionOwner(context.Background(), observation)
}

type sessionNameReplacer interface{ ReplaceSessionNameIndex(SessionNameEntry) error }

func publishLaunchSessionName(rt Runtime, entry SessionNameEntry, managed bool) error {
	if !managed {
		return rt.AppendSessionNameIndex(entry)
	}
	publisher, ok := rt.(sessionNameReplacer)
	if !ok {
		return errors.New("couch session binding replacement unavailable")
	}
	return publisher.ReplaceSessionNameIndex(entry)
}

// verifyCouchSessionCreate checks the conversation's current association before
// replacing it. The index identifies candidates; live pane ownership decides
// whether a candidate still belongs to this conversation.
func verifyCouchSessionCreate(rt Runtime, globalDataDir string, intent CouchSessionIntent) error {
	if intent.Disposition != "create" {
		return nil
	}
	index, err := rt.ReadSessionNameIndex()
	if err != nil {
		return fmt.Errorf("couch session current association: %w", err)
	}
	live, err := rt.SessionLiveness()
	if err != nil {
		return fmt.Errorf("couch session current liveness: %w", err)
	}
	candidates := map[string]bool{legacySessionPrefix + intent.Tag: true}
	for _, entry := range index.Entries {
		if entry.ScopeKey == intent.Scope && entry.Tag == intent.Tag {
			candidates[entry.SessionName] = true
		}
	}
	for _, session := range live {
		if session.State == SessionExited || !candidates[session.Name] {
			continue
		}
		observer, ok := rt.(couchSessionOwnerRuntime)
		if !ok {
			return errors.New("couch session owner observer unavailable")
		}
		observed, err := observer.ObserveSessionOwner(context.Background(), session.Name, globalDataDir, intent.Scope, intent.Tag)
		if err != nil {
			return err
		}
		switch observed.State {
		case SessionOwnerForeign, SessionOwnerAbsent:
			continue
		default:
			return fmt.Errorf("couch conversation terminal %q is live or unresolved; refusing another terminal", session.Name)
		}
	}
	return nil
}
