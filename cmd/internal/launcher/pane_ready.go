package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/panebirth"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// PublishPaneReady is the first command of the actual initial agent pane. It
// performs no native query and acquires no namespace lock: the creator still
// owns that lock while waiting for this exact launch's sidecar (#866).
func PublishPaneReady(ctx context.Context, env []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	values := map[string]string{}
	for _, entry := range zellijquery.ChildEnvironment(env) {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	namespace, err := zellijquery.ReadinessNamespace(env)
	if err != nil {
		return err
	}
	identity := panebirth.LaunchIdentity{Namespace: namespace, Scope: values["PAIR_SCOPE_KEY"], Tag: values["PAIR_TAG"], Agent: values["PAIR_AGENT"], Session: values["ZELLIJ_SESSION_NAME"], Nonce: values["PAIR_LAUNCH_NONCE"]}
	if err := identity.Validate(); err != nil {
		return err
	}
	data := values["PAIR_DATA_DIR"]
	if filepath.Base(filepath.Dir(data)) == "repos" && filepath.Base(data) != identity.Scope {
		return errors.New("pane scope differs from selected data namespace")
	}
	paths, err := artifactpath.ResolveScoped(data, identity.Tag)
	if err != nil {
		return err
	}
	path, err := paths.PaneChecked(identity.Agent)
	if err != nil {
		return err
	}
	if bound := values["PAIR_AGENT_PANE_PATH"]; bound != "" && bound != path {
		return errors.New("pane sidecar binding differs from artifact owner")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	record := panebirth.Record{PaneID: values["ZELLIJ_PANE_ID"], Cwd: cwd, Identity: identity}
	if raw, err := osfs.ReadRegularMetadata(ctx, path, panebirth.MaxRecordBytes); err == nil {
		old, err := panebirth.DecodeRecord(raw)
		if err != nil {
			return err
		}
		if old.Identity.Equal(identity) {
			record.Unknown = old.Unknown
			record.IdentityUnknown = old.IdentityUnknown
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	raw, err := panebirth.EncodeRecord(record)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return (osfs.FS{}).WriteAtomic(path, string(raw))
}
