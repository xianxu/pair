package launcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

const QuitIntentVersion = 1

type QuitIntentKind string

const (
	QuitIntentDirect QuitIntentKind = "direct"
	QuitIntentCouch  QuitIntentKind = "couch"
)

type QuitRequestReference struct {
	DataDir   string `json:"data_dir"`
	RepoScope string `json:"repo_scope"`
	Tag       string `json:"tag"`
	Nonce     string `json:"nonce"`
	Attempt   uint64 `json:"attempt"`
}

type QuitIntent struct {
	Version int                   `json:"version"`
	Kind    QuitIntentKind        `json:"kind"`
	Request *QuitRequestReference `json:"request,omitempty"`
}

// ReadQuitIntent accepts the legacy empty touch marker as direct Alt+x and
// strictly decodes versioned direct/Couch markers.
func ReadQuitIntent(raw []byte) (QuitIntent, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return QuitIntent{Version: QuitIntentVersion, Kind: QuitIntentDirect}, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var intent QuitIntent
	if err := decoder.Decode(&intent); err != nil {
		return QuitIntent{}, fmt.Errorf("decode quit intent: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return QuitIntent{}, errors.New("quit intent has trailing data")
	}
	if err := validateQuitIntent(intent); err != nil {
		return QuitIntent{}, err
	}
	return intent, nil
}

func WriteQuitIntent(intent QuitIntent) ([]byte, error) {
	if err := validateQuitIntent(intent); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func validateQuitIntent(intent QuitIntent) error {
	if intent.Version != QuitIntentVersion {
		return fmt.Errorf("unsupported quit intent version %d", intent.Version)
	}
	switch intent.Kind {
	case QuitIntentDirect:
		if intent.Request != nil {
			return errors.New("direct quit intent carries Couch request")
		}
	case QuitIntentCouch:
		if intent.Request == nil {
			return errors.New("Couch quit intent requires request reference")
		}
		paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: intent.Request.DataDir, RepoScope: intent.Request.RepoScope, Tag: intent.Request.Tag})
		if err != nil {
			return fmt.Errorf("invalid Couch request address: %w", err)
		}
		lifecycle, err := paths.Lifecycle(intent.Request.Nonce)
		if err != nil {
			return fmt.Errorf("invalid Couch request nonce: %w", err)
		}
		if _, err := lifecycle.Request(intent.Request.Attempt); err != nil {
			return fmt.Errorf("invalid Couch request attempt: %w", err)
		}
	default:
		return fmt.Errorf("unsupported quit intent kind %q", intent.Kind)
	}
	return nil
}

// Restart/quit marker logic (#99 M3, ported from bin/pair-shell's
// handle_restart_marker + pair-restart.sh handshake). The markers live under
// ~/.cache/pair/{quit,restart}-<session>; parsing + the re-launch decision are
// pure here, the read/clear IO sits on the Runtime seam.

// RestartMarker is the parsed ~/.cache/pair/restart-<session> handshake dropped
// by `pair restart` (Alt+n / Shift+Alt+N, #94 M1) or the #55 compaction branch.
type RestartMarker struct {
	Tag        string
	Agent      string
	SessionID  string // plain restart: live native session id captured before kill
	NewSession bool   // Shift+Alt+N / compaction: fresh agent conversation
	RenameTo   string // #22 inside-flow tag rename (native re-entry as of M5b)
	Continue   string // #55 compaction slug (native continue re-entry as of M5b)
}

// parseRestartMarker reads the `key=value` lines `pair restart` writes. Unknown
// keys are ignored; a missing marker is the caller's concern (empty content →
// zero value).
func parseRestartMarker(content string) RestartMarker {
	var m RestartMarker
	for _, line := range strings.Split(content, "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch strings.TrimSpace(key) {
		case "tag":
			m.Tag = val
		case "agent":
			m.Agent = val
		case "session_id":
			m.SessionID = val
		case "new_session":
			m.NewSession = val == "1"
		case "rename_to":
			m.RenameTo = val
		case "continue":
			m.Continue = val
		}
	}
	return m
}

// restartPlan is the decision the in-process restart loop acts on: the next
// launch (Args), whether to drop the saved config first, and — for a #55
// compaction re-entry — the continuation slug to re-seed the draft from.
type restartPlan struct {
	Args         LaunchArgs
	DropConfig   bool   // Shift+Alt+N / compaction: drop the saved config first
	ContinueSlug string // #55 compaction re-entry: re-seed the draft from this slug
}

// decideAutomaticResumeConfig rejects only persisted Codex bindings that no
// longer identify a verified root rollout. Keep the saved launch parameters so
// callers can still offer a fresh launch with the user's prior flags.
func decideAutomaticResumeConfig(agent string, saved savedConfig, sessionValid bool) (savedConfig, bool) {
	if agent != "codex" || saved.SessionID == "" || sessionValid {
		return saved, false
	}
	saved.SessionID = ""
	return saved, true
}

// planRestart maps a restart marker + the RESOLVED (tag, agent) + saved config
// into the next launch (#99 M5b makes rename/continue native). The caller has
// already applied the marker's tag/agent preference AND any rename_to move before
// calling this, so tag/agent here are final. Mirrors handle_restart_marker (shell
// 762-810): Shift+Alt+N / compaction drop the config and re-launch fresh; a plain
// Alt+n resumes only the established binding carried in the marker. Saved
// config contributes non-resume args but is never native-session authority.
func planRestart(m RestartMarker, tag, agent string, saved savedConfig) restartPlan {
	base := LaunchArgs{Agent: agent, ForcedTag: tag}
	if m.NewSession {
		// Fresh conversation: keep the saved args, drop the config so the create
		// path mints a new session id rather than resuming the prior one. A
		// Continue slug only ever rides new_session (shell 1055-1056), so re-seed
		// here (the loop resolves the slug → draft).
		base.AgentArgs = append([]string(nil), saved.Args...)
		return restartPlan{Args: base, DropConfig: true, ContinueSlug: m.Continue}
	}
	// Default Alt+n: an empty marker ID means the current typed generation is
	// still provisional. Drop stale config and relaunch fresh with saved flags.
	if m.SessionID == "" {
		base.AgentArgs = append([]string(nil), saved.Args...)
		return restartPlan{Args: base, DropConfig: true}
	}
	base.AgentArgs = composeResumeArgs(agent, saved.Args, m.SessionID)
	return restartPlan{Args: base}
}
