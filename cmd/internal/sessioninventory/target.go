package sessioninventory

import "sort"

type TargetMode string

const (
	TargetNewLaunch      TargetMode = "new-launch"
	TargetEstablished    TargetMode = "established"
	TargetExplicitResume TargetMode = "explicit-resume"
	TargetActivity       TargetMode = "activity"
	TargetDiagnostic     TargetMode = "diagnostic"
)

type TargetArtifactBoundary struct {
	StorageRoot  string
	RelativePath string
}

// TargetRequest describes the only artifact set a latency-sensitive consumer
// is authorized to inspect.
// pair:156-concept pure new final TargetRequest / TargetResult
type TargetRequest struct {
	Mode                TargetMode
	Agent               Agent
	NativeID            string
	Baseline            []TargetArtifactBoundary
	AuthorizedArtifacts []Artifact
}

type TargetResult struct {
	Eligible    []ArtifactObservation
	Unavailable bool
}

// SelectTargetWork is a pure fail-closed eligibility boundary. Metadata may
// identify candidates, but cannot broaden durable or explicit authority.
func SelectTargetWork(request TargetRequest, observations []ArtifactObservation) TargetResult {
	observed := sortedTargetObservations(observations, request.Agent)
	switch request.Mode {
	case TargetDiagnostic:
		return TargetResult{Eligible: observed}
	case TargetNewLaunch:
		return selectNewLaunchTargets(request, observed)
	case TargetEstablished, TargetActivity:
		if len(request.AuthorizedArtifacts) == 0 {
			return TargetResult{Unavailable: true}
		}
		return TargetResult{Eligible: selectAuthorizedArtifacts(observed, request.AuthorizedArtifacts)}
	case TargetExplicitResume:
		if len(request.AuthorizedArtifacts) != 0 {
			return TargetResult{Eligible: selectAuthorizedArtifacts(observed, request.AuthorizedArtifacts)}
		}
		eligible := selectNamedArtifacts(request.Agent, request.NativeID, observed)
		return TargetResult{Eligible: eligible, Unavailable: len(eligible) == 0}
	default:
		return TargetResult{Unavailable: true}
	}
}

func selectNewLaunchTargets(request TargetRequest, observed []ArtifactObservation) TargetResult {
	baseline := map[string]bool{}
	for _, boundary := range request.Baseline {
		baseline[boundary.StorageRoot+"\x00"+boundary.RelativePath] = true
	}
	if request.Agent != AgentAgy {
		var eligible []ArtifactObservation
		for _, observation := range observed {
			if !baseline[targetArtifactKey(observation.Entry.Artifact)] && observationNativeID(request.Agent, observation.Entry.Artifact) != "" {
				eligible = append(eligible, observation)
			}
		}
		return TargetResult{Eligible: eligible}
	}
	byID := map[string][]ArtifactObservation{}
	for _, observation := range observed {
		if baseline[targetArtifactKey(observation.Entry.Artifact)] {
			continue
		}
		if id := observationNativeID(AgentAgy, observation.Entry.Artifact); id != "" {
			byID[id] = append(byID[id], observation)
		}
	}
	var eligible []ArtifactObservation
	for _, joined := range byID {
		database, transcript := false, false
		for _, observation := range joined {
			database = database || observation.Entry.Artifact.StorageRoot == "agy-conversations"
			transcript = transcript || observation.Entry.Artifact.StorageRoot == "agy-brain"
		}
		if database && transcript {
			eligible = append(eligible, joined...)
		}
	}
	return TargetResult{Eligible: sortedTargetObservations(eligible, request.Agent)}
}

func selectAuthorizedArtifacts(observed []ArtifactObservation, authorized []Artifact) []ArtifactObservation {
	wanted := map[string]bool{}
	for _, artifact := range authorized {
		wanted[targetArtifactKey(artifact)] = true
	}
	var result []ArtifactObservation
	for _, observation := range observed {
		if wanted[targetArtifactKey(observation.Entry.Artifact)] {
			result = append(result, observation)
		}
	}
	return result
}

func selectNamedArtifacts(agent Agent, nativeID string, observed []ArtifactObservation) []ArtifactObservation {
	if nativeID == "" {
		return nil
	}
	var result []ArtifactObservation
	for _, observation := range observed {
		if observationNativeID(agent, observation.Entry.Artifact) == nativeID {
			result = append(result, observation)
		}
	}
	if agent == AgentAgy {
		database, transcript := false, false
		for _, observation := range result {
			database = database || observation.Entry.Artifact.StorageRoot == "agy-conversations"
			transcript = transcript || observation.Entry.Artifact.StorageRoot == "agy-brain"
		}
		if !database || !transcript {
			return nil
		}
	}
	return result
}

func observationNativeID(agent Agent, artifact Artifact) string {
	switch agent {
	case AgentClaude:
		id, _, role, ok := claudePathFact(artifact.RelativePath)
		if ok && role == RoleRoot && artifact.StorageRoot == "claude-projects" {
			return id
		}
	case AgentCodex:
		id, ok := codexPathID(artifact.RelativePath)
		if ok && artifact.StorageRoot == "codex-sessions" {
			return id
		}
	case AgentMuse:
		id, _, role, ok := musePathFact(artifact.RelativePath)
		if ok && role == RoleRoot && artifact.StorageRoot == "muse-sessions" {
			return id
		}
	case AgentAgy:
		if artifact.StorageRoot == "agy-conversations" {
			id, _ := agyDatabasePathID(artifact.RelativePath)
			return id
		}
		if artifact.StorageRoot == "agy-brain" {
			id, _ := agyTranscriptPathID(artifact.RelativePath)
			return id
		}
	}
	return ""
}

func sortedTargetObservations(observations []ArtifactObservation, agent Agent) []ArtifactObservation {
	var result []ArtifactObservation
	for _, observation := range observations {
		if observation.Agent == agent {
			result = append(result, cloneObservation(observation))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return targetArtifactKey(result[i].Entry.Artifact) < targetArtifactKey(result[j].Entry.Artifact)
	})
	return result
}

func targetArtifactKey(artifact Artifact) string {
	return artifact.StorageRoot + "\x00" + artifact.RelativePath
}
