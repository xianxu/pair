package vmcmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

type ToolVersion struct {
	Version string                     `json:"version"`
	SHA256  string                     `json:"sha256"`
	Unknown map[string]json.RawMessage `json:"-"`
}

type ImageManifest struct {
	Version   int                        `json:"version"`
	Image     string                     `json:"image"`
	Base      string                     `json:"base"`
	CreatedAt time.Time                  `json:"created_at"`
	Tools     map[string]ToolVersion     `json:"tools"`
	Qualified bool                       `json:"qualified"`
	Unknown   map[string]json.RawMessage `json:"-"`
}

var imageManifestFields = strictjson.KnownFields(reflect.TypeOf(ImageManifest{}))
var toolVersionFields = strictjson.KnownFields(reflect.TypeOf(ToolVersion{}))
var imageNamePattern = regexp.MustCompile(`^pair-conformance-[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

const imageRepositoryComponent = `[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*`
const imageRegistry = `[a-z0-9]+(?:[.-][a-z0-9]+)*(?::[0-9]+)?`

var baseReferencePattern = regexp.MustCompile(`^(?:` + imageRegistry + `/)?` + imageRepositoryComponent + `(?:/` + imageRepositoryComponent + `)*(?::[A-Za-z0-9_][A-Za-z0-9_.-]*)?@sha256:[0-9a-f]{64}$`)

func DecodeImageManifest(raw []byte) (ImageManifest, error) {
	var m ImageManifest
	owned, unknown, err := requiredObject(raw, imageManifestFields)
	if err != nil {
		return m, err
	}
	toolsRaw := owned["tools"]
	delete(owned, "tools")
	knownRaw, err := strictjson.EncodeObject(owned, nil, imageManifestFields)
	if err != nil {
		return m, err
	}
	if err := strictjson.Decode(knownRaw, &m); err != nil {
		return ImageManifest{}, err
	}
	_, tools, err := strictjson.SplitObject(toolsRaw, nil, nil)
	if err != nil {
		return ImageManifest{}, fmt.Errorf("tools: %w", err)
	}
	m.Tools = make(map[string]ToolVersion, len(tools))
	for name, rawTool := range tools {
		fields, extra, err := requiredObject(rawTool, toolVersionFields)
		if err != nil {
			return ImageManifest{}, fmt.Errorf("tool %q: %w", name, err)
		}
		encoded, err := strictjson.EncodeObject(fields, nil, toolVersionFields)
		if err != nil {
			return ImageManifest{}, err
		}
		var tool ToolVersion
		if err := strictjson.Decode(encoded, &tool); err != nil {
			return ImageManifest{}, fmt.Errorf("tool %q: %w", name, err)
		}
		tool.Unknown = extra
		m.Tools[name] = tool
	}
	m.Unknown = unknown
	if err := validateImageManifest(m); err != nil {
		return ImageManifest{}, err
	}
	return m, nil
}

// EncodeImageManifest retains additive members verbatim, including within tools.
func EncodeImageManifest(m ImageManifest) ([]byte, error) {
	if err := validateImageManifest(m); err != nil {
		return nil, err
	}
	tools := make(map[string]json.RawMessage, len(m.Tools))
	for name, tool := range m.Tools {
		fields, err := ownedValues(tool)
		if err != nil {
			return nil, err
		}
		encoded, err := strictjson.EncodeObject(fields, tool.Unknown, toolVersionFields)
		if err != nil {
			return nil, err
		}
		tools[name] = encoded
	}
	toolsRaw, err := strictjson.EncodeObject(tools, nil, nil)
	if err != nil {
		return nil, err
	}
	// Marshal only scalar fields: re-marshaling raw tools would compact unknown values.
	scalar := m
	scalar.Tools = nil
	owned, err := ownedValues(scalar)
	if err != nil {
		return nil, err
	}
	owned["tools"] = toolsRaw
	return strictjson.EncodeObject(owned, m.Unknown, imageManifestFields)
}

func ownedValues(value any) (map[string]json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	err = json.Unmarshal(raw, &fields)
	return fields, err
}

func requiredObject(raw []byte, known map[string]bool) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	owned, unknown, err := strictjson.SplitObject(raw, known, nil)
	if err != nil {
		return nil, nil, err
	}
	for name := range known {
		if _, ok := owned[name]; !ok {
			return nil, nil, fmt.Errorf("required member %q missing", name)
		}
	}
	return owned, unknown, nil
}

func validateImageManifest(m ImageManifest) error {
	if m.Version != 1 {
		return fmt.Errorf("unsupported image manifest version %d", m.Version)
	}
	if !imageNamePattern.MatchString(m.Image) {
		return fmt.Errorf("image must be a managed pair-conformance- local name")
	}
	if !baseReferencePattern.MatchString(m.Base) {
		return fmt.Errorf("base must be an OCI reference pinned by sha256 digest")
	}
	_, offset := m.CreatedAt.Zone()
	if m.CreatedAt.IsZero() || offset != 0 {
		return fmt.Errorf("created_at must be a nonzero UTC timestamp")
	}
	if !m.Qualified {
		return fmt.Errorf("image must be qualified")
	}
	if len(m.Tools) == 0 {
		return fmt.Errorf("image tools are required")
	}
	for name, tool := range m.Tools {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(tool.Version) == "" || !sha256Pattern.MatchString(tool.SHA256) {
			return fmt.Errorf("tool %q requires a version and lowercase sha256 checksum", name)
		}
	}
	return nil
}
