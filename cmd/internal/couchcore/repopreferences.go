package couchcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

const (
	RepoPreferencesSchemaVersion = 1
	MaxRepoPreferenceBytes       = 4 << 20
	MaxRepoPreferenceRepos       = 256
	MaxRepoPreferenceSources     = 256
	MaxRepoArgumentBytes         = 4096
)

type RepoPreferences struct {
	SchemaVersion int
	Repositories  map[string]RepoLaunchPreference
	Unknown       map[string]json.RawMessage
}

type RepoLaunchPreference struct {
	RepoIdentity  string
	Revision      uint64
	Agents        map[string]RepoAgentArguments
	LegacySources map[string]LegacyRepoPreference
	Unknown       map[string]json.RawMessage
}

type RepoAgentArguments struct {
	Argv      []string
	Confirmed bool
	Conflict  bool
	Unknown   map[string]json.RawMessage
}

type LegacyRepoPreference struct {
	SourceID   string
	Digest     string
	Preference PathLaunchPreference
	Raw        json.RawMessage
	Unknown    map[string]json.RawMessage
}

type RepoArgumentResolution struct {
	Argv     []string
	Found    bool
	Conflict bool
	Revision uint64
}

type BorrowedArguments struct {
	RepoIdentity string
	Agent        string
	Argv         []string
	Revision     uint64
}

func validRepoIdentity(id string) bool {
	return id != "" && utf8.ValidString(id) && !strings.ContainsRune(id, 0)
}

func validateRepoArgv(agent string, argv []string) error {
	if !launcher.IsSupportedAgent(agent) {
		return fmt.Errorf("repository preferences: unsupported agent %q", agent)
	}
	if argv == nil {
		return errors.New("repository preferences: arguments must not be null")
	}
	for _, arg := range argv {
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return errors.New("repository preferences: invalid argument text")
		}
	}
	if len(launcher.FormatLaunchParameters(argv)) > MaxRepoArgumentBytes {
		return fmt.Errorf("repository preferences: arguments exceed %d bytes", MaxRepoArgumentBytes)
	}
	return nil
}

func validateRepoPreference(p RepoLaunchPreference) error {
	if !validRepoIdentity(p.RepoIdentity) || p.Revision == 0 || p.Agents == nil || p.LegacySources == nil {
		return errors.New("repository preferences: incomplete repository record")
	}
	if len(p.Agents) > len(launcher.AgentInventory()) || len(p.LegacySources) > MaxRepoPreferenceSources {
		return errors.New("repository preferences: record inventory exceeds bound")
	}
	for agent, a := range p.Agents {
		if err := validateRepoArgv(agent, a.Argv); err != nil {
			return err
		}
		if a.Confirmed && a.Conflict {
			return errors.New("repository preferences: confirmed arguments cannot conflict")
		}
	}
	for key, s := range p.LegacySources {
		if key != s.SourceID {
			return errors.New("repository preferences: legacy source key mismatch")
		}
		if err := validateLegacyRepoPreference(s, p.RepoIdentity); err != nil {
			return err
		}
	}
	return nil
}

func cloneLegacyRepoPreference(s LegacyRepoPreference) LegacyRepoPreference {
	s.Preference = clonePathLaunchPreference(s.Preference)
	s.Raw = append(json.RawMessage(nil), s.Raw...)
	s.Unknown = strictjson.CloneRaw(s.Unknown)
	return s
}

func cloneRepoPreference(p RepoLaunchPreference) RepoLaunchPreference {
	out := p
	out.Unknown = strictjson.CloneRaw(p.Unknown)
	out.Agents = make(map[string]RepoAgentArguments, len(p.Agents))
	for agent, a := range p.Agents {
		a.Argv = cloneArgv(a.Argv)
		a.Unknown = strictjson.CloneRaw(a.Unknown)
		out.Agents[agent] = a
	}
	out.LegacySources = make(map[string]LegacyRepoPreference, len(p.LegacySources))
	for key, s := range p.LegacySources {
		out.LegacySources[key] = cloneLegacyRepoPreference(s)
	}
	return out
}

func repoPreferenceForUpdate(current *RepoLaunchPreference, repoIdentity string) (RepoLaunchPreference, error) {
	if !validRepoIdentity(repoIdentity) {
		return RepoLaunchPreference{}, errors.New("repository preferences: missing repository identity")
	}
	if current == nil {
		return RepoLaunchPreference{RepoIdentity: repoIdentity, Revision: 1, Agents: map[string]RepoAgentArguments{}, LegacySources: map[string]LegacyRepoPreference{}}, nil
	}
	if err := validateRepoPreference(*current); err != nil {
		return RepoLaunchPreference{}, err
	}
	if current.RepoIdentity != repoIdentity {
		return RepoLaunchPreference{}, errors.New("repository preferences: repository identity mismatch")
	}
	return cloneRepoPreference(*current), nil
}

func advanceRepoPreferenceRevision(next *RepoLaunchPreference, current *RepoLaunchPreference) error {
	if current == nil || reflect.DeepEqual(*next, *current) {
		return nil
	}
	if current.Revision == math.MaxUint64 {
		return errors.New("repository preferences: revision exhausted")
	}
	next.Revision = current.Revision + 1
	return nil
}

// ResolveRepoArguments never accepts another repository's suggestion implicitly.
func ResolveRepoArguments(current *RepoLaunchPreference, agent string, explicit *[]string) (RepoArgumentResolution, error) {
	out := RepoArgumentResolution{}
	if !launcher.IsSupportedAgent(agent) {
		return out, fmt.Errorf("repository preferences: unsupported agent %q", agent)
	}
	if current != nil {
		if err := validateRepoPreference(*current); err != nil {
			return out, err
		}
		out.Revision = current.Revision
	}
	if explicit != nil {
		if err := validateRepoArgv(agent, *explicit); err != nil {
			return out, err
		}
		out.Argv = cloneArgv(*explicit)
		out.Found = true
		return out, nil
	}
	if current == nil {
		return out, nil
	}
	a, ok := current.Agents[agent]
	if !ok {
		return out, nil
	}
	out.Conflict = a.Conflict
	if !a.Conflict {
		out.Argv = cloneArgv(a.Argv)
		out.Found = true
	}
	return out, nil
}

func RememberRepoArguments(current *RepoLaunchPreference, repoIdentity string, profile LaunchProfile) (RepoLaunchPreference, error) {
	if err := validateRepoArgv(profile.Agent, profile.Argv); err != nil {
		return RepoLaunchPreference{}, err
	}
	next, err := repoPreferenceForUpdate(current, repoIdentity)
	if err != nil {
		return next, err
	}
	a := next.Agents[profile.Agent]
	a.Argv = cloneArgv(profile.Argv)
	a.Confirmed = true
	a.Conflict = false
	next.Agents[profile.Agent] = a
	if err := advanceRepoPreferenceRevision(&next, current); err != nil {
		return RepoLaunchPreference{}, err
	}
	return next, nil
}

// FoldPathPreferences replaces changed source snapshots, then derives provisional
// values from the complete retained set. Source revision and discovery order
// never resolve disagreements; only explicit acceptance confirms a value.
func FoldPathPreferences(current *RepoLaunchPreference, repoIdentity string, sources []LegacyRepoPreference) (RepoLaunchPreference, error) {
	if len(sources) > MaxRepoPreferenceSources {
		return RepoLaunchPreference{}, errors.New("repository preferences: too many legacy sources")
	}
	next, err := repoPreferenceForUpdate(current, repoIdentity)
	if err != nil {
		return next, err
	}
	seen := map[string]LegacyRepoPreference{}
	for _, s := range sources {
		if err := validateLegacyRepoPreference(s, repoIdentity); err != nil {
			return RepoLaunchPreference{}, err
		}
		if old, ok := seen[s.SourceID]; ok && (!reflect.DeepEqual(old.Raw, s.Raw) || old.Digest != s.Digest) {
			return RepoLaunchPreference{}, errors.New("repository preferences: ambiguous repeated legacy source")
		}
		seen[s.SourceID] = s
		if old, ok := next.LegacySources[s.SourceID]; ok {
			s.Unknown = strictjson.MergeRaw(old.Unknown, s.Unknown)
		}
		next.LegacySources[s.SourceID] = cloneLegacyRepoPreference(s)
	}
	if len(next.LegacySources) > MaxRepoPreferenceSources {
		return RepoLaunchPreference{}, errors.New("repository preferences: too many retained legacy sources")
	}
	provisional := map[string]RepoAgentArguments{}
	for _, s := range next.LegacySources {
		for agent, argv := range s.Preference.ArgvByAgent {
			if accepted, ok := next.Agents[agent]; ok && accepted.Confirmed {
				continue
			}
			v, ok := provisional[agent]
			if !ok {
				v = RepoAgentArguments{Argv: cloneArgv(argv), Unknown: strictjson.CloneRaw(next.Agents[agent].Unknown)}
			} else if !reflect.DeepEqual(v.Argv, argv) {
				v.Conflict = true
				v.Argv = []string{}
			}
			provisional[agent] = v
		}
	}
	for agent, a := range next.Agents {
		if !a.Confirmed {
			delete(next.Agents, agent)
		}
	}
	for agent, a := range provisional {
		next.Agents[agent] = a
	}
	if err := advanceRepoPreferenceRevision(&next, current); err != nil {
		return RepoLaunchPreference{}, err
	}
	return next, nil
}

// ChooseBorrowedArguments makes one bounded pass and returns a suggestion only.
func ChooseBorrowedArguments(records map[string]RepoLaunchPreference, target, agent string) (BorrowedArguments, bool, error) {
	return chooseBorrowedArguments(records, target, agent, nil)
}

// visit observes the one record pass for the counted growth contract.
func chooseBorrowedArguments(records map[string]RepoLaunchPreference, target, agent string, visit func(string)) (BorrowedArguments, bool, error) {
	var chosen BorrowedArguments
	if !validRepoIdentity(target) || !launcher.IsSupportedAgent(agent) || len(records) > MaxRepoPreferenceRepos {
		return chosen, false, errors.New("repository preferences: invalid borrowing inventory")
	}
	for key, p := range records {
		if visit != nil {
			visit(key)
		}
		if key != p.RepoIdentity {
			return BorrowedArguments{}, false, errors.New("repository preferences: repository key mismatch")
		}
		if err := validateRepoPreference(p); err != nil {
			return BorrowedArguments{}, false, err
		}
		a, ok := p.Agents[agent]
		if key == target || !ok || !a.Confirmed || a.Conflict {
			continue
		}
		if chosen.RepoIdentity == "" || key < chosen.RepoIdentity {
			chosen = BorrowedArguments{RepoIdentity: key, Agent: agent, Argv: cloneArgv(a.Argv), Revision: p.Revision}
		}
	}
	return chosen, chosen.RepoIdentity != "", nil
}
