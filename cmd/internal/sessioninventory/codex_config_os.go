package sessioninventory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/xianxu/pair/cmd/internal/osfs"
)

const codexConfigFileLimit = 1 << 20
const codexConfigTotalLimit = 4 << 20
const codexConfigFileCount = 64

var errCodexConfigObservation = errors.New("Codex configuration observation incomplete")
var codexProfileName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type codexConfigOSReader struct {
	getenv     func(string) string
	read       func(context.Context, string) ([]byte, error)
	mdm        func() (bool, bool)
	systemRoot string
}

func newCodexConfigOSReader() codexConfigOSReader {
	return codexConfigOSReader{getenv: os.Getenv, read: func(ctx context.Context, path string) ([]byte, error) {
		return osfs.ReadRegularMetadata(ctx, path, codexConfigFileLimit)
	}, mdm: codexForcedPreferences, systemRoot: "/etc/codex"}
}

// CodexManagedPolicyStatus is a platform observation, not configuration authority.
type CodexManagedPolicyStatus string

const (
	CodexManagedPolicyAbsent  CodexManagedPolicyStatus = "absent"
	CodexManagedPolicyPresent CodexManagedPolicyStatus = "present"
	CodexManagedPolicyUnknown CodexManagedPolicyStatus = "unknown"
)

// CodexConfigReader's zero value reads the host system policy. Injection replaces
// only platform policy observation and system file location, never the projector.
type CodexConfigReader struct {
	SystemDir     string
	ManagedPolicy func() (CodexManagedPolicyStatus, error)
}

func (reader CodexConfigReader) Read(ctx context.Context, q CodexConfigInvocation, getenv ...func(string) string) (CodexConfigProjection, error) {
	r := newCodexConfigOSReader()
	if reader.SystemDir != "" {
		r.systemRoot = reader.SystemDir
	}
	if reader.ManagedPolicy != nil {
		r.mdm = func() (bool, bool) {
			s, err := reader.ManagedPolicy()
			if err != nil {
				return false, false
			}
			return s == CodexManagedPolicyPresent, s == CodexManagedPolicyPresent || s == CodexManagedPolicyAbsent
		}
	}
	if len(getenv) > 1 {
		return CodexConfigProjection{}, errors.New("one Codex environment reader required")
	}
	if len(getenv) == 1 {
		if getenv[0] == nil {
			return CodexConfigProjection{}, errors.New("nil Codex environment reader")
		}
		r.getenv = getenv[0]
	}
	return readCodexSelectionConfig(ctx, q, r)
}

// ReadCodexSelectionConfig observes only named config files and nonsecret scope
// environment keys. It never starts Codex or loads auth.
func ReadCodexSelectionConfig(ctx context.Context, q CodexConfigInvocation, getenv ...func(string) string) (CodexConfigProjection, error) {
	return (CodexConfigReader{}).Read(ctx, q, getenv...)
}
func readCodexSelectionConfig(ctx context.Context, q CodexConfigInvocation, r codexConfigOSReader) (CodexConfigProjection, error) {
	compat := func(why CodexConfigReason) (CodexConfigProjection, error) {
		return CodexConfigProjection{Status: CodexConfigCompatibility, Reason: why}, nil
	}
	if ctx == nil {
		return CodexConfigProjection{}, errors.New("Codex config requires context")
	}
	if err := ctx.Err(); err != nil {
		return CodexConfigProjection{}, err
	}
	if q.Unsupported || q.OSS || q.LocalProvider != "" || (q.Mode != CodexModeTUI && q.Mode != CodexModeExec) || !filepath.IsAbs(q.CWD) {
		return compat(CodexConfigScopeUnsupported)
	}
	if r.getenv("CODEX_EXEC_SERVER_URL") != "" {
		return compat(CodexConfigScopeUnsupported)
	}
	home := r.getenv("HOME")
	if !filepath.IsAbs(home) {
		return compat(CodexConfigObservationIncomplete)
	}
	startup := q.StartupCWD
	if startup == "" {
		startup = q.CWD
	}
	ch, err := ResolveCodexHome(startup, r.getenv)
	if err != nil {
		return compat(CodexConfigObservationIncomplete)
	}
	if len(strings.Split(filepath.Clean(q.CWD), string(filepath.Separator))) > 128 {
		return compat(CodexConfigObservationIncomplete)
	}
	if codexNestedSymlink(q.CWD) {
		return compat(CodexConfigScopeUnsupported)
	}
	cwd, err := filepath.EvalSymlinks(q.CWD)
	if err != nil {
		return compat(CodexConfigObservationIncomplete)
	}
	st, err := os.Stat(cwd)
	if err != nil || !st.IsDir() {
		return compat(CodexConfigObservationIncomplete)
	}
	present, known := r.mdm()
	if !known || present {
		return compat(CodexConfigManagedUnsupported)
	}
	bytesRead, files := 0, 0
	read := func(path string) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		files++
		if files > codexConfigFileCount {
			return nil, errCodexConfigObservation
		}
		b, err := r.read(ctx, path)
		bytesRead += len(b)
		if len(b) > codexConfigFileLimit || bytesRead > codexConfigTotalLimit {
			return nil, errCodexConfigObservation
		}
		return b, err
	}
	for _, name := range []string{"requirements.toml", "managed_config.toml"} {
		_, err := read(filepath.Join(r.systemRoot, name))
		if err == nil {
			return compat(CodexConfigManagedUnsupported)
		}
		if !os.IsNotExist(err) {
			if ctx.Err() != nil {
				return CodexConfigProjection{}, ctx.Err()
			}
			return compat(CodexConfigObservationIncomplete)
		}
	}
	// Merely observing configured environments is enough to retain native behavior;
	// no executor addresses or credentials need to be loaded.
	if _, err := read(filepath.Join(ch, "environments.toml")); err == nil {
		return compat(CodexConfigScopeUnsupported)
	} else if !os.IsNotExist(err) {
		return compat(CodexConfigObservationIncomplete)
	}
	load := func(path string) (codexConfigLayer, error) {
		b, err := read(path)
		if os.IsNotExist(err) {
			return codexConfigLayer{base: filepath.Dir(path), values: map[string]any{}}, nil
		}
		if err != nil {
			return codexConfigLayer{}, err
		}
		var v map[string]any
		if err = toml.Unmarshal(b, &v); err != nil {
			return codexConfigLayer{}, err
		}
		for _, key := range []string{"sqlite_home"} {
			if s, ok := v[key].(string); ok {
				v[key] = codexExpandHome(s, home)
			}
		}
		return codexConfigLayer{base: filepath.Dir(path), values: v}, nil
	}
	layers := []codexConfigLayer{}
	for _, path := range []string{filepath.Join(r.systemRoot, "config.toml"), filepath.Join(ch, "config.toml")} {
		l, err := load(path)
		if err != nil {
			return compat(CodexConfigInvalid)
		}
		layers = append(layers, l)
	}
	if q.Profile != "" {
		if !codexProfileName.MatchString(q.Profile) {
			return compat(CodexConfigScopeUnsupported)
		}
		base := layers[len(layers)-1].values
		if s, _ := base["profile"].(string); s == q.Profile {
			return compat(CodexConfigInvalid)
		}
		if m, ok := base["profiles"].(map[string]any); ok {
			if _, exists := m[q.Profile]; exists {
				return compat(CodexConfigInvalid)
			}
		}
		l, err := load(filepath.Join(ch, q.Profile+".config.toml"))
		if err != nil {
			return compat(CodexConfigInvalid)
		}
		layers = append(layers, l)
	}
	q.CWD = cwd
	cli, err := codexCLIConfigLayer(q, home)
	if err != nil {
		return compat(CodexConfigInvalid)
	}
	// Trust/root discovery sees session flags but never project self-assertions.
	discovery := append(append([]codexConfigLayer(nil), layers...), cli)
	project, err := codexProjectConfigLayers(cwd, ch, discovery, load)
	if err != nil {
		return compat(CodexConfigObservationIncomplete)
	}
	layers = append(layers, project...)
	layers = append(layers, cli)
	e := codexConfigEvidence{scope: CodexSelectionScope{CodexHome: ch, CWD: cwd, Mode: q.Mode}, layers: layers, envSQLite: codexExpandHome(strings.TrimSpace(r.getenv("CODEX_SQLITE_HOME")), home)}
	p := projectCodexConfig(e)
	if err := ctx.Err(); err != nil {
		return CodexConfigProjection{}, err
	}
	if p.Status != CodexConfigComplete {
		return p, nil
	}
	p.Scope.CWDs = []string{cwd}
	if q.Mode == CodexModeTUI && p.Scope.Worktrees {
		cwds, err := codexWorktreeCWDs(cwd)
		if err != nil {
			return compat(CodexConfigObservationIncomplete)
		}
		p.Scope.CWDs = cwds
	}
	return p, nil
}
func codexReadConfigFile(path string) ([]byte, error) {
	return codexReadBoundedRegular(path, codexConfigFileLimit)
}
func codexReadBoundedRegular(path string, limit int64) ([]byte, error) {
	return osfs.ReadRegularMetadata(context.Background(), path, limit)
}
func codexExpandHome(s, home string) string {
	if s == "~" {
		return home
	}
	if strings.HasPrefix(s, "~/") {
		return filepath.Join(home, strings.TrimLeft(s[2:], "/"))
	}
	return s
}
func codexCLIConfigLayer(q CodexConfigInvocation, home string) (codexConfigLayer, error) {
	values := map[string]any{}
	for _, raw := range q.Overrides {
		k, v, ok := strings.Cut(raw, "=")
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if !ok || k == "" {
			return codexConfigLayer{}, errCodexConfigObservation
		}
		parsed := map[string]any{}
		if err := toml.Unmarshal([]byte("_x_="+v), &parsed); err != nil {
			parsed["_x_"] = strings.Trim(v, "\"'")
		}
		value := parsed["_x_"]
		if k == "sqlite_home" {
			if s, ok := value.(string); ok {
				value = codexExpandHome(s, home)
			}
		}
		// Native apply_toml_override splits literal dots; it does not parse TOML
		// key syntax. Newlines, comments and quotes remain literal key characters.
		segments := strings.Split(k, ".")
		layer := map[string]any{}
		current := layer
		for i, segment := range segments {
			if i == len(segments)-1 {
				current[segment] = value
				break
			}
			next := map[string]any{}
			current[segment] = next
			current = next
		}
		codexMergeConfig(values, layer)
	}
	for _, a := range []struct {
		names   []string
		enabled bool
	}{{q.Enable, true}, {q.Disable, false}} {
		for _, name := range a.names {
			// Pair disables shell snapshots for Couch ownership. This native
			// feature does not change history selection or its namespace.
			if name != "worktrees" && name != "shell_snapshot" {
				return codexConfigLayer{}, errCodexConfigObservation
			}
			codexMergeConfig(values, map[string]any{"features": map[string]any{name: a.enabled}})
		}
	}
	return codexConfigLayer{base: q.CWD, values: values}, nil
}
func codexMergeConfig(dst, src map[string]any) {
	for k, v := range src {
		if m, ok := v.(map[string]any); ok {
			if d, ok := dst[k].(map[string]any); ok {
				codexMergeConfig(d, m)
				continue
			}
		}
		if m, ok := v.(map[string]any); ok {
			cloned := map[string]any{}
			codexMergeConfig(cloned, m)
			v = cloned
		}
		dst[k] = v
	}
}

// ResolveCodexHome is independent of latest-selection configuration: explicit
// UUIDs need only the native physical transcript namespace.
func ResolveCodexHome(cwd string, getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	home := getenv("HOME")
	if !filepath.IsAbs(home) || !filepath.IsAbs(cwd) {
		return "", errCodexConfigObservation
	}
	ch := getenv("CODEX_HOME")
	if ch == "" {
		return filepath.Join(home, ".codex"), nil
	}
	if !filepath.IsAbs(ch) {
		ch = filepath.Join(cwd, ch)
	}
	p, err := filepath.EvalSymlinks(ch)
	if err != nil {
		return "", errCodexConfigObservation
	}
	st, err := os.Stat(p)
	if err != nil || !st.IsDir() {
		return "", errCodexConfigObservation
	}
	return p, nil
}

// Codex preserves nested logical aliases, but canonicalizes top-level aliases.
// Until all logical project-path origins are represented, nested aliases stay
// native compatibility instead of broadening a query beyond its native filter.
func codexNestedSymlink(path string) bool {
	for d := filepath.Clean(path); filepath.Dir(d) != d; d = filepath.Dir(d) {
		st, err := os.Lstat(d)
		if err != nil {
			continue
		}
		parent := filepath.Dir(d)
		if st.Mode()&os.ModeSymlink != 0 && filepath.Dir(parent) != parent {
			return true
		}
	}
	return false
}
