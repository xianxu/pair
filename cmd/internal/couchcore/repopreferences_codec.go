package couchcore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func repoJSONFields(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, name := range names {
		m[name] = true
	}
	return m
}

var (
	repoDocumentFields = repoJSONFields("schema_version", "repositories")
	repoRecordFields   = repoJSONFields("repo_identity", "revision", "agents", "legacy_sources")
	repoAgentFields    = repoJSONFields("argv", "confirmed", "conflict")
	repoLegacyFields   = repoJSONFields("source_id", "digest", "raw")
	repoPathFields     = repoJSONFields("schema_version", "repo_identity", "physical_path", "last_agent", "argv_by_agent", "revision")
)

func splitRepoJSON(raw []byte, known map[string]bool) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	owned, unknown, err := strictjson.SplitObject(raw, known, nil)
	if err != nil {
		return nil, nil, err
	}
	for key := range known {
		if _, ok := owned[key]; !ok {
			return nil, nil, fmt.Errorf("repository preferences: missing %s", key)
		}
	}
	return owned, unknown, nil
}

func repoJSONValue(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }

func decodeLegacyPath(raw []byte) (PathLaunchPreference, error) {
	var p PathLaunchPreference
	owned, _, err := splitRepoJSON(raw, repoPathFields)
	if err != nil {
		return p, err
	}
	knownRaw, err := strictjson.EncodeObject(owned, nil, repoPathFields)
	if err != nil {
		return p, err
	}
	if err := strictjson.Decode(knownRaw, &p); err != nil {
		return p, err
	}
	_, agentValues, err := strictjson.SplitObject(owned["argv_by_agent"], nil, nil)
	if err != nil {
		return p, err
	}
	for agent, value := range agentValues {
		argv, err := DecodeArgumentArray(value)
		if err != nil {
			return p, err
		}
		p.ArgvByAgent[agent] = argv
	}
	if err := validatePathLaunchPreference(p); err != nil {
		return p, err
	}
	if !validRepoIdentity(p.RepoIdentity) {
		return p, errors.New("repository preferences: invalid legacy repository identity")
	}
	if _, ok := p.ArgvByAgent[p.LastAgent]; !ok {
		return p, errors.New("repository preferences: missing last agent arguments")
	}
	for agent, argv := range p.ArgvByAgent {
		if err := validateRepoArgv(agent, argv); err != nil {
			return p, err
		}
	}
	return p, nil
}

// DecodeLegacyRepoPreference retains the original image, including future
// members, rather than re-encoding the projection this version understands.
func DecodeLegacyRepoPreference(sourceID string, raw []byte) (LegacyRepoPreference, error) {
	var s LegacyRepoPreference
	if !validRepoIdentity(sourceID) || !utf8.Valid(raw) || len(raw) > MaxRepoPreferenceBytes {
		return s, errors.New("repository preferences: invalid or oversized legacy source")
	}
	p, err := decodeLegacyPath(raw)
	if err != nil {
		return s, err
	}
	digest := sha256.Sum256(raw)
	return LegacyRepoPreference{SourceID: sourceID, Digest: hex.EncodeToString(digest[:]), Preference: p, Raw: append(json.RawMessage(nil), raw...)}, nil
}

func validateLegacyRepoPreference(s LegacyRepoPreference, repoIdentity string) error {
	decoded, err := DecodeLegacyRepoPreference(s.SourceID, s.Raw)
	if err != nil {
		return err
	}
	if decoded.Digest != s.Digest || decoded.Preference.RepoIdentity != repoIdentity || !reflect.DeepEqual(decoded.Preference, s.Preference) {
		return errors.New("repository preferences: legacy source identity, digest or projection mismatch")
	}
	return nil
}

func decodeRepoAgent(raw []byte) (RepoAgentArguments, error) {
	var a RepoAgentArguments
	owned, unknown, err := splitRepoJSON(raw, repoAgentFields)
	if err != nil {
		return a, err
	}
	for key, target := range map[string]any{"confirmed": &a.Confirmed, "conflict": &a.Conflict} {
		if err := strictjson.Decode(owned[key], target); err != nil {
			return a, err
		}
	}
	a.Argv, err = DecodeArgumentArray(owned["argv"])
	if err != nil {
		return a, err
	}
	a.Unknown = unknown
	return a, nil
}

func decodeRepoLegacy(raw []byte) (LegacyRepoPreference, error) {
	var s LegacyRepoPreference
	owned, unknown, err := splitRepoJSON(raw, repoLegacyFields)
	if err != nil {
		return s, err
	}
	if err := strictjson.Decode(owned["source_id"], &s.SourceID); err != nil {
		return s, err
	}
	if err := strictjson.Decode(owned["digest"], &s.Digest); err != nil {
		return s, err
	}
	// A string retains the entire source image, including whitespace around the
	// original JSON value. An embedded object would lose that whitespace when
	// decoded and invalidate the source's exact-byte digest.
	var original string
	if err := strictjson.Decode(owned["raw"], &original); err != nil {
		return s, err
	}
	s.Raw = json.RawMessage(original)
	s.Preference, err = decodeLegacyPath(s.Raw)
	if err != nil {
		return s, err
	}
	s.Unknown = unknown
	return s, nil
}

func decodeRepoRecord(raw []byte) (RepoLaunchPreference, error) {
	var p RepoLaunchPreference
	owned, unknown, err := splitRepoJSON(raw, repoRecordFields)
	if err != nil {
		return p, err
	}
	if err := strictjson.Decode(owned["repo_identity"], &p.RepoIdentity); err != nil {
		return p, err
	}
	if err := strictjson.Decode(owned["revision"], &p.Revision); err != nil {
		return p, err
	}
	_, agents, err := strictjson.SplitObject(owned["agents"], nil, nil)
	if err != nil {
		return p, err
	}
	_, sources, err := strictjson.SplitObject(owned["legacy_sources"], nil, nil)
	if err != nil {
		return p, err
	}
	if len(agents) > len(launcher.AgentInventory()) || len(sources) > MaxRepoPreferenceSources {
		return p, errors.New("repository preferences: oversized record inventory")
	}
	p.Agents = make(map[string]RepoAgentArguments, len(agents))
	p.LegacySources = make(map[string]LegacyRepoPreference, len(sources))
	p.Unknown = unknown
	for key, value := range agents {
		a, err := decodeRepoAgent(value)
		if err != nil {
			return p, err
		}
		p.Agents[key] = a
	}
	for key, value := range sources {
		s, err := decodeRepoLegacy(value)
		if err != nil {
			return p, err
		}
		p.LegacySources[key] = s
	}
	if err := validateRepoPreference(p); err != nil {
		return p, err
	}
	return p, nil
}

func DecodeRepoPreferences(raw []byte) (RepoPreferences, error) {
	var doc RepoPreferences
	if !utf8.Valid(raw) {
		return doc, errors.New("repository preferences: document must be UTF-8")
	}
	if len(raw) > MaxRepoPreferenceBytes {
		return doc, errors.New("repository preferences: document exceeds 4 MiB; inspect/archive settings")
	}
	owned, unknown, err := splitRepoJSON(raw, repoDocumentFields)
	if err != nil {
		return doc, err
	}
	if err := strictjson.Decode(owned["schema_version"], &doc.SchemaVersion); err != nil {
		return doc, err
	}
	if doc.SchemaVersion != RepoPreferencesSchemaVersion {
		return doc, errors.New("repository preferences: unsupported schema version")
	}
	_, repos, err := strictjson.SplitObject(owned["repositories"], nil, nil)
	if err != nil {
		return doc, err
	}
	if len(repos) > MaxRepoPreferenceRepos {
		return doc, errors.New("repository preferences: more than 256 repositories; inspect/archive settings")
	}
	doc.Repositories = make(map[string]RepoLaunchPreference, len(repos))
	doc.Unknown = unknown
	for key, value := range repos {
		p, err := decodeRepoRecord(value)
		if err != nil {
			return doc, err
		}
		if key != p.RepoIdentity {
			return doc, errors.New("repository preferences: repository key mismatch")
		}
		doc.Repositories[key] = p
	}
	return doc, nil
}

func encodeRepoAgent(a RepoAgentArguments) ([]byte, error) {
	return strictjson.EncodeObject(map[string]json.RawMessage{"argv": repoJSONValue(a.Argv), "confirmed": repoJSONValue(a.Confirmed), "conflict": repoJSONValue(a.Conflict)}, a.Unknown, repoAgentFields)
}

func encodeRepoLegacy(s LegacyRepoPreference) ([]byte, error) {
	return strictjson.EncodeObject(map[string]json.RawMessage{"source_id": repoJSONValue(s.SourceID), "digest": repoJSONValue(s.Digest), "raw": repoJSONValue(string(s.Raw))}, s.Unknown, repoLegacyFields)
}

func encodeRepoRecord(p RepoLaunchPreference) ([]byte, error) {
	if err := validateRepoPreference(p); err != nil {
		return nil, err
	}
	agents := make(map[string]json.RawMessage, len(p.Agents))
	sources := make(map[string]json.RawMessage, len(p.LegacySources))
	for key, a := range p.Agents {
		raw, err := encodeRepoAgent(a)
		if err != nil {
			return nil, err
		}
		agents[key] = raw
	}
	for key, s := range p.LegacySources {
		raw, err := encodeRepoLegacy(s)
		if err != nil {
			return nil, err
		}
		sources[key] = raw
	}
	agentsRaw, err := strictjson.EncodeObject(agents, nil, nil)
	if err != nil {
		return nil, err
	}
	sourcesRaw, err := strictjson.EncodeObject(sources, nil, nil)
	if err != nil {
		return nil, err
	}
	return strictjson.EncodeObject(map[string]json.RawMessage{"repo_identity": repoJSONValue(p.RepoIdentity), "revision": repoJSONValue(p.Revision), "agents": agentsRaw, "legacy_sources": sourcesRaw}, p.Unknown, repoRecordFields)
}

func EncodeRepoPreferences(doc RepoPreferences) ([]byte, error) {
	if doc.SchemaVersion != RepoPreferencesSchemaVersion || doc.Repositories == nil {
		return nil, errors.New("repository preferences: invalid document")
	}
	if len(doc.Repositories) > MaxRepoPreferenceRepos {
		return nil, errors.New("repository preferences: more than 256 repositories; inspect/archive settings")
	}
	repos := make(map[string]json.RawMessage, len(doc.Repositories))
	for key, p := range doc.Repositories {
		if key != p.RepoIdentity {
			return nil, errors.New("repository preferences: repository key mismatch")
		}
		raw, err := encodeRepoRecord(p)
		if err != nil {
			return nil, err
		}
		repos[key] = raw
	}
	reposRaw, err := strictjson.EncodeObject(repos, nil, nil)
	if err != nil {
		return nil, err
	}
	raw, err := strictjson.EncodeObject(map[string]json.RawMessage{"schema_version": repoJSONValue(doc.SchemaVersion), "repositories": reposRaw}, doc.Unknown, repoDocumentFields)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(raw) {
		return nil, errors.New("repository preferences: document must be UTF-8")
	}
	if len(raw) > MaxRepoPreferenceBytes {
		return nil, errors.New("repository preferences: document exceeds 4 MiB; inspect/archive settings")
	}
	return raw, nil
}
