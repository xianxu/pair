package couchcmd

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// PrepareForcedAgentSwitch pins the current source before the local form's
// confirmation. Only Git display errors and idle/cleanliness policy are waived.
func (g *switchServiceGuard) PrepareForcedAgentSwitch(ctx context.Context, address couchcore.ThreadAddress, evidence couchcore.SwitchSourceEvidence) (string, *int, error) {
	return g.inspectForcedAgentSwitch(ctx, address, "", evidence)
}

// ConfirmForcedAgentSwitch rechecks the source pinned by the consumed local
// consent. An empty token cannot turn a disconnected live source into vacancy.
func (g *switchServiceGuard) ConfirmForcedAgentSwitch(ctx context.Context, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence) error {
	if source == "" && !evidence.Vacant() {
		return errors.New("switch-agent: local force source token unavailable")
	}
	_, _, err := g.inspectForcedAgentSwitch(ctx, address, source, evidence)
	return err
}

func (g *switchServiceGuard) inspectForcedAgentSwitch(ctx context.Context, address couchcore.ThreadAddress, source string, evidence couchcore.SwitchSourceEvidence) (string, *int, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	s := g.service.Load()
	if s == nil {
		return "", nil, errors.New("switch-agent: supervisor unavailable")
	}
	var status couchcore.SlotGitStatus
	var err error
	if evidence.Vacant() {
		status, err = inspectVacantSwitchSource(ctx, s, address, source, evidence, true)
	} else {
		if s.authority.record == nil || s.authority.workspace == nil || s.authority.process == nil || s.authority.thread == nil || s.authority.wrapperPID == nil || s.authority.recorded == nil || s.authority.branch == nil {
			return "", nil, errors.New("switch-agent: local force source verification unavailable")
		}
		var live couchmessage.SlotLiveness
		live, status, err = inspectSwitchLiveSourceWithOptions(ctx, s, address, source, switchSourceOptions{allowUnclean: true, forceEvidence: &evidence})
		if err == nil {
			source = switchSourceToken(live)
		}
	}
	if err != nil {
		return "", nil, err
	}
	var count *int
	if status.ChangedFiles != nil {
		n := *status.ChangedFiles
		count = &n
	}
	return source, count, nil
}

func verifyForcedSwitchLiveSource(ctx context.Context, s *messageService, live couchmessage.SlotLiveness, address couchcore.ThreadAddress, evidence couchcore.SwitchSourceEvidence, identity couchcore.WorkspaceIdentity) error {
	if err := verifyForcedSwitchWorkspace(ctx, s, evidence, identity); err != nil {
		return err
	}
	if err := s.authority.process(live.Binding); err != nil {
		return err
	}
	if _, err := s.authority.current(ctx, live.Binding); err != nil {
		return err
	}
	current, err := s.connectedWorkspace(ctx, live.Binding)
	if err != nil {
		return err
	}
	if current.WorktreeRoot != identity.WorktreeRoot || current.RepoIdentity != identity.RepoIdentity {
		return errors.New("switch-agent: local force workspace changed; review it again")
	}
	record, err := s.authority.record(ctx, address)
	if err != nil {
		return err
	}
	if !evidence.MatchesRecord(record) {
		return errors.New("switch-agent: local force source generation changed; review it again")
	}
	latest, ok := s.LivenessForThread(address.RepoScope, string(address.Tag))
	if !ok || switchSourceToken(latest) != switchSourceToken(live) || !s.isConnected(latest.Binding) {
		return errors.New("switch-agent: local force source changed while checking; review it again")
	}
	return ctx.Err()
}

func verifyForcedSwitchWorkspace(ctx context.Context, s *messageService, evidence couchcore.SwitchSourceEvidence, identity couchcore.WorkspaceIdentity) error {
	current, err := s.authority.workspace(ctx, evidence.WorkingPath())
	if err != nil {
		return err
	}
	relative, relErr := filepath.Rel(current.WorktreeRoot, evidence.WorkingPath())
	if evidence.RepositoryIdentity() == "" || current.RepoIdentity != evidence.RepositoryIdentity() || current.WorktreeRoot != identity.WorktreeRoot || current.RepoIdentity != identity.RepoIdentity || !filepath.IsAbs(current.WorktreeRoot) || relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("switch-agent: local force workspace identity changed; review it again")
	}
	return ctx.Err()
}
