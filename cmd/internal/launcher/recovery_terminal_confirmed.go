package launcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// RecoveryProcess is a copied confirmation summary, never signal authority.
type RecoveryProcess struct {
	PID       int
	Identity  string
	Role      string
	ParentPID int
}

func PrepareConfirmedRecoveryTerminal(ctx context.Context, root, scope, tag, session string) (RecoveryTerminalPlan, error) {
	return (recoveryTerminalDriver{ops: newRecoveryTerminalOS(), timeout: zjTimeout, poll: 25 * time.Millisecond}).prepareConfirmed(ctx, root, scope, tag, session)
}

// RecheckRecoveryTerminal observes the exact prepared generation without signals
// or lease acquisition. Retirement acquires its lease and repeats the check.
func RecheckRecoveryTerminal(ctx context.Context, plan RecoveryTerminalPlan) error {
	return (recoveryTerminalDriver{ops: newRecoveryTerminalOS(), timeout: zjTimeout, poll: 25 * time.Millisecond}).recheck(ctx, plan)
}

func (p RecoveryTerminalPlan) Processes() []RecoveryProcess {
	var out []RecoveryProcess
	for _, s := range p.servers {
		out = append(out, RecoveryProcess{PID: s.Server.PID, Identity: s.Server.Identity, Role: "server", ParentPID: s.Witness.PPID})
		for _, v := range s.Descendants {
			out = append(out, RecoveryProcess{PID: v.PID, Identity: v.Identity, Role: v.Role, ParentPID: v.PPID})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out
}

func (p RecoveryTerminalPlan) Digest() string {
	if !p.valid {
		return ""
	}
	raw, _ := json.Marshal(struct {
		Owner     artifactpath.StorageOwner
		Session   string
		Confirmed bool
		Servers   []recoveryAttributedServer
		Ready     map[int]readiness.ReadyRecord
	}{p.owner, p.session, p.confirmed, p.servers, p.ready})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (d recoveryTerminalDriver) prepareConfirmed(ctx context.Context, root, scope, tag, session string) (RecoveryTerminalPlan, error) {
	return d.prepareMode(ctx, root, scope, tag, session, true)
}
func (d recoveryTerminalDriver) recheck(ctx context.Context, p RecoveryTerminalPlan) error {
	if ctx == nil || d.ops == nil || d.timeout <= 0 || !p.valid {
		return errors.New("recheck requires prepared exact terminal authorization")
	}
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	snapshot, err := d.ops.Observe(ctx, p.session)
	if err != nil {
		return err
	}
	if p.confirmed {
		_, err = d.checkConfirmed(ctx, p, snapshot, false)
		return err
	}
	current, err := attributeRecoveryServers(p.owner, p.session, snapshot)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, p.servers) {
		return errors.New("terminal ownership snapshot changed; refresh recovery preview")
	}
	return ctx.Err()
}

func (d recoveryTerminalDriver) readRecoveryReady(ctx context.Context, path string, limit int64) ([]byte, error) {
	if d.read != nil {
		return d.read(ctx, path, limit)
	}
	return osfs.ReadRegularMetadata(ctx, path, limit)
}

// Encode the observed argv for the existing pure ownership parser without
// erasing argument boundaries. No shell executes this representation.
func recoveryArgCommand(args []string) string {
	words := make([]string, len(args))
	for i, arg := range args {
		words[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	return strings.Join(words, " ")
}
func recoveryExecutable(args []string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	name := filepath.Base(args[0])
	return name, name == "pair-wrap" || name == "pair" && len(args) > 1 && args[1] == "wrap"
}
func recoveryRows(snapshot recoveryTerminalSnapshot) (map[int]procutil.Process, error) {
	rows := map[int]procutil.Process{}
	for _, p := range snapshot.Processes {
		if p.PID <= 0 {
			return nil, errors.New("invalid process identity")
		}
		if _, ok := rows[p.PID]; ok {
			return nil, errors.New("ambiguous process snapshot")
		}
		rows[p.PID] = p
	}
	return rows, nil
}

func (d recoveryTerminalDriver) attributeConfirmed(ctx context.Context, owner artifactpath.StorageOwner, session string, snapshot recoveryTerminalSnapshot) ([]recoveryAttributedServer, map[int]readiness.ReadyRecord, error) {
	if len(snapshot.Servers) > 32 {
		return nil, nil, errors.New("too many contested terminal generations")
	}
	rows, err := recoveryRows(snapshot)
	if err != nil {
		return nil, nil, err
	}
	var out []recoveryAttributedServer
	proofs := map[int]readiness.ReadyRecord{}
	seen := map[int]bool{}
	for _, server := range snapshot.Servers {
		row := rows[server.PID]
		if !validQuiescenceServer(server, session) || seen[server.PID] || row.Identity != server.Identity || len(snapshot.Args[server.PID]) == 0 {
			return nil, nil, errors.New("terminal server identity is incomplete or changed")
		}
		seen[server.PID] = true
		item := recoveryAttributedServer{Server: server, Witness: recoveryProcessWitness{PID: row.PID, PPID: row.PPID, Identity: row.Identity, Args: append([]string(nil), snapshot.Args[row.PID]...)}}
		anchors := map[int]string{}
		native := map[int]bool{}
		for _, p := range snapshot.Processes {
			if p.PID == server.PID || !recoveryDescendant(server.PID, p.PID, rows) {
				continue
			}
			args := snapshot.Args[p.PID]
			if p.Identity == "" || len(args) == 0 || !AgentBelongsToServer(server, p.PID, p.Identity, snapshot.Processes) {
				return nil, nil, errors.New("terminal descendant native argv or identity unavailable")
			}
			command := recoveryArgCommand(args)
			observed := ClassifySessionOwner(owner, []zellijpane.Pane{{PaneCommand: command}})
			name, wrapper := recoveryExecutable(args)
			if observed.State == SessionOwnerForeign {
				return nil, nil, errors.New("terminal descendant has a foreign Pair owner")
			}
			if wrapper {
				if observed.State != SessionOwnerOwned {
					return nil, nil, errors.New("live wrapper has no exact storage owner")
				}
				wrappers, err := recoveryNativeWrappers(owner.DataDir, recoveryTerminalSnapshot{Processes: []procutil.Process{p}, Commands: map[int]string{p.PID: command}})
				if err != nil || len(wrappers) != 1 || wrappers[0].Owner != owner {
					return nil, nil, errors.New("live wrapper attribution unavailable")
				}
				ready, err := recoveryWrapperReady(ctx, snapshot, wrappers[0], d.readRecoveryReady)
				if err != nil {
					return nil, nil, err
				}
				if ready.Session != session || ready.Nonce == "" {
					return nil, nil, errors.New("live wrapper readiness session or nonce mismatch")
				}
				if native[ready.PID] {
					return nil, nil, errors.New("ambiguous ready agent ancestry")
				}
				proofs[p.PID] = ready
				native[ready.PID] = true
				anchors[p.PID] = "wrapper"
			} else if name == "nvim" && observed.State == SessionOwnerOwned {
				anchors[p.PID] = "editor"
			}
			item.Descendants = append(item.Descendants, recoveryProcessWitness{PID: p.PID, PPID: p.PPID, Identity: p.Identity, Command: command, Args: append([]string(nil), args...)})
		}
		if len(anchors) == 0 {
			return nil, nil, fmt.Errorf("terminal server %d has no proven live Pair owner", server.PID)
		}
		for i := range item.Descendants {
			v := &item.Descendants[i]
			v.Role = anchors[v.PID]
			if native[v.PID] {
				v.Role = "native-agent"
			}
			if v.Role == "" {
				for anchor := range anchors {
					if recoveryDescendant(anchor, v.PID, rows) {
						v.Role = "descendant"
						break
					}
				}
			}
			if v.Role == "" {
				return nil, nil, errors.New("terminal has an unattributed descendant; inspect before retirement")
			}
		}
		sort.Slice(item.Descendants, func(i, j int) bool { return item.Descendants[i].PID < item.Descendants[j].PID })
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Server.PID < out[j].Server.PID })
	return out, proofs, ctx.Err()
}

// During retirement accepted members may disappear. Every surviving member must
// keep its generation, parent and argv, and every current descendant must have
// been in the accepted set. Re-attribution can never expand kill authority.
func (d recoveryTerminalDriver) checkConfirmed(ctx context.Context, plan RecoveryTerminalPlan, snapshot recoveryTerminalSnapshot, allowGone bool) ([]SessionServerIdentity, error) {
	rows, err := recoveryRows(snapshot)
	if err != nil {
		return nil, err
	}
	accepted := map[int]recoveryProcessWitness{}
	servers := map[int]SessionServerIdentity{}
	for _, s := range plan.servers {
		servers[s.Server.PID] = s.Server
		accepted[s.Server.PID] = s.Witness
		for _, v := range s.Descendants {
			accepted[v.PID] = v
		}
	}
	seen := map[int]bool{}
	for _, server := range snapshot.Servers {
		if prior, ok := servers[server.PID]; !ok || prior != server || seen[server.PID] {
			return nil, errors.New("terminal server set changed; refresh recovery preview")
		}
		seen[server.PID] = true
	}
	if !allowGone && len(seen) != len(servers) {
		return nil, errors.New("terminal server disappeared; refresh recovery preview")
	}
	for pid, witness := range accepted {
		row, ok := rows[pid]
		if !ok {
			if allowGone {
				continue
			}
			return nil, errors.New("accepted terminal process disappeared")
		}
		if row.Identity != witness.Identity || row.PPID != witness.PPID {
			return nil, errors.New("accepted terminal process replaced or reparented")
		}
		if !reflect.DeepEqual(snapshot.Args[pid], witness.Args) {
			return nil, errors.New("accepted terminal process argv changed")
		}
	}
	for _, p := range snapshot.Processes {
		for pid := range servers {
			if p.PID != pid && recoveryDescendant(pid, p.PID, rows) {
				if _, ok := accepted[p.PID]; !ok {
					return nil, errors.New("new terminal descendant; refusing further signals")
				}
			}
		}
	}
	for wrapperPID, prior := range plan.ready {
		wrapper, wrapperAlive := rows[wrapperPID]
		agent, agentAlive := rows[prior.PID]
		if allowGone && !wrapperAlive {
			continue
		}
		if allowGone && !agentAlive {
			raw, err := d.readRecoveryReady(ctx, AgentReadyPath(plan.owner.Directory(), plan.owner.Tag, prior.Agent), maxAgentReadyBytes)
			if err != nil {
				return nil, err
			}
			current, err := readiness.Decode(string(raw))
			if err != nil || !reflect.DeepEqual(current, prior) {
				return nil, errors.New("terminal readiness changed during retirement")
			}
			continue
		}
		if !wrapperAlive || !agentAlive || agent.Identity != prior.AgentIdentity {
			return nil, errors.New("ready agent generation changed")
		}
		current, err := recoveryWrapperReady(ctx, snapshot, recoveryNativeWrapper{Process: wrapper, Owner: plan.owner, Agent: prior.Agent}, d.readRecoveryReady)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(current, prior) {
			return nil, errors.New("terminal readiness changed; refresh recovery preview")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]SessionServerIdentity(nil), snapshot.Servers...), nil
}

// retireConfirmed never expands the accepted set. Children must be observed
// absent before their parents are signalled, avoiding induced reparenting of
// still-running native owners. A signal's success is not absence evidence.
func (d recoveryTerminalDriver) retireConfirmed(ctx context.Context, plan RecoveryTerminalPlan) error {
	signal, ok := d.ops.(interface {
		KillProcess(context.Context, recoveryProcessWitness) error
	})
	if !ok {
		return errors.New("exact owner retirement unavailable")
	}
	var targets []recoveryProcessWitness
	servers := map[int]SessionServerIdentity{}
	parents := map[int]int{}
	for _, s := range plan.servers {
		targets = append(targets, s.Witness)
		targets = append(targets, s.Descendants...)
		servers[s.Server.PID] = s.Server
	}
	for _, p := range targets {
		parents[p.PID] = p.PPID
	}
	depth := func(pid int) int {
		n := 0
		for seen := map[int]bool{}; !seen[pid]; {
			seen[pid] = true
			parent, ok := parents[pid]
			if !ok {
				break
			}
			n++
			pid = parent
		}
		return n
	}
	sort.Slice(targets, func(i, j int) bool {
		a, b := depth(targets[i].PID), depth(targets[j].PID)
		if a != b {
			return a > b
		}
		return targets[i].PID < targets[j].PID
	})
	observe := func() (map[int]procutil.Process, error) {
		snapshot, err := d.ops.Observe(ctx, plan.session)
		if err != nil {
			return nil, err
		}
		if _, err = d.checkConfirmed(ctx, plan, snapshot, true); err != nil {
			return nil, err
		}
		return recoveryRows(snapshot)
	}

	for _, target := range targets {
		rows, err := observe()
		if err != nil {
			return err
		}
		if _, alive := rows[target.PID]; !alive {
			continue
		}
		if server, ok := servers[target.PID]; ok {
			err = d.ops.KillServer(server)
		} else {
			err = signal.KillProcess(ctx, target)
		}
		if err != nil {
			return err
		}
		for {
			rows, err = observe()
			if err != nil {
				return err
			}
			if _, alive := rows[target.PID]; !alive {
				break
			}
			if err = waitRecoveryTerminalPoll(ctx, d.poll); err != nil {
				return fmt.Errorf("accepted terminal process %d remains: %w", target.PID, err)
			}
		}
	}
	for {
		rows, err := observe()
		if err != nil {
			return err
		}
		for _, p := range targets {
			if _, alive := rows[p.PID]; alive {
				return errors.New("accepted terminal owner still present")
			}
		}
		present, err := d.ops.SessionPresent(ctx, plan.session)
		if err != nil {
			return err
		}
		if !present {
			// Listing is another IO interval: repeat the process proof after it.
			final, err := observe()
			if err != nil {
				return err
			}
			for _, p := range targets {
				if _, alive := final[p.PID]; alive {
					return errors.New("accepted terminal owner reappeared")
				}
			}
			return ctx.Err()
		}
		if err = waitRecoveryTerminalPoll(ctx, d.poll); err != nil {
			return fmt.Errorf("terminal listing remains: %w", err)
		}
	}
}
