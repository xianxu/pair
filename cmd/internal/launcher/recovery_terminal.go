package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// RecoveryTerminalPlan is ephemeral exact-process authorization. Only Prepare
// can populate it; a session name or cached launch policy cannot manufacture it.
// The caller holds its thread gate and preserves the screen before Retire.
type RecoveryTerminalPlan struct {
	owner     artifactpath.StorageOwner
	session   string
	servers   []recoveryAttributedServer
	valid     bool
	confirmed bool
	ready     map[int]readiness.ReadyRecord
}
type recoveryProcessWitness struct {
	PID, PPID         int
	Identity, Command string
	Args              []string
	Role              string
}
type recoveryAttributedServer struct {
	Server      SessionServerIdentity
	Witness     recoveryProcessWitness
	Descendants []recoveryProcessWitness
}
type recoveryTerminalSnapshot struct {
	Servers   []SessionServerIdentity
	Processes []procutil.Process
	Commands  map[int]string
	Args      map[int][]string
}
type recoveryTerminalOps interface {
	Acquire(context.Context, string, []SessionServerIdentity) (io.Closer, error)
	Observe(context.Context, string) (recoveryTerminalSnapshot, error)
	SessionPresent(context.Context, string) (bool, error)
	KillServer(SessionServerIdentity) error
}
type recoveryTerminalDriver struct {
	ops           recoveryTerminalOps
	read          func(context.Context, string, int64) ([]byte, error)
	timeout, poll time.Duration
}

func PrepareRecoveryTerminal(ctx context.Context, root, scope, tag, session string) (RecoveryTerminalPlan, error) {
	return (recoveryTerminalDriver{ops: newRecoveryTerminalOS(), timeout: zjTimeout, poll: 25 * time.Millisecond}).prepare(ctx, root, scope, tag, session)
}
func RetireRecoveryTerminal(ctx context.Context, plan RecoveryTerminalPlan) error {
	return zellijquery.RetireReadyAfter(ctx, nil, plan.session, func(ctx context.Context) error {
		return (recoveryTerminalDriver{ops: newRecoveryTerminalOS(), timeout: zjTimeout, poll: 25 * time.Millisecond}).retire(ctx, plan)
	})
}

func (d recoveryTerminalDriver) prepare(ctx context.Context, root, scope, tag, session string) (RecoveryTerminalPlan, error) {
	return d.prepareMode(ctx, root, scope, tag, session, false)
}
func (d recoveryTerminalDriver) prepareMode(ctx context.Context, root, scope, tag, session string, confirmed bool) (plan RecoveryTerminalPlan, err error) {
	if ctx == nil || d.ops == nil || d.timeout <= 0 {
		return plan, errors.New("recovery terminal requires bounded context and IO")
	}
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	owner, err := sessionExpectedOwner(root, scope, tag)
	if err != nil {
		return plan, err
	}
	if session == "" || filepath.Base(session) != session || session == "." || session == ".." {
		return plan, errors.New("invalid recovery terminal name")
	}
	before, err := d.ops.Observe(ctx, session)
	if err != nil {
		return plan, err
	}
	before.Servers = append([]SessionServerIdentity(nil), before.Servers...)
	lease, err := d.ops.Acquire(ctx, session, before.Servers)
	if err != nil {
		return plan, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	after, err := d.ops.Observe(ctx, session)
	if err != nil {
		return plan, err
	}
	if !sameRecoveryServers(before.Servers, after.Servers) {
		return plan, errors.New("terminal servers changed while acquiring recovery lease")
	}
	var servers []recoveryAttributedServer
	var ready map[int]readiness.ReadyRecord
	if confirmed {
		servers, ready, err = d.attributeConfirmed(ctx, owner, session, after)
	} else {
		servers, err = attributeRecoveryServers(owner, session, after)
	}
	if err != nil {
		return plan, err
	}
	plan = RecoveryTerminalPlan{owner: owner, session: session, servers: servers, valid: true, confirmed: confirmed, ready: ready}
	if confirmed {
		final, observeErr := d.ops.Observe(ctx, session)
		if observeErr != nil {
			return RecoveryTerminalPlan{}, observeErr
		}
		if _, checkErr := d.checkConfirmed(ctx, plan, final, false); checkErr != nil {
			return RecoveryTerminalPlan{}, checkErr
		}
	}
	return plan, nil
}
func (d recoveryTerminalDriver) retire(ctx context.Context, plan RecoveryTerminalPlan) (err error) {
	if ctx == nil || d.ops == nil || d.timeout <= 0 || !plan.valid {
		return errors.New("retirement requires prepared exact terminal authorization")
	}
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	authorized := recoveryServerIdentities(plan.servers)
	lease, err := d.ops.Acquire(ctx, plan.session, authorized)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	snapshot, err := d.ops.Observe(ctx, plan.session)
	if err != nil {
		return err
	}
	if plan.confirmed {
		if _, err := d.checkConfirmed(ctx, plan, snapshot, true); err != nil {
			return err
		}
		return d.retireConfirmed(ctx, plan)
	}
	current, err := attributeRecoveryServers(plan.owner, plan.session, snapshot)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, plan.servers) {
		return errors.New("terminal ownership snapshot changed; refresh recovery preview")
	}
	if err := quiesceAuthorizedSession(ctx, plan.session, authorized, recoveryTerminalQuiescence{driver: d, plan: plan, ctx: ctx}, d.timeout, d.poll); err != nil {
		return err
	}
	// Strict authorization still signals only servers. Their disappearance
	// cannot establish that the independently named owners exited with them.
	for {
		snapshot, err := d.ops.Observe(ctx, plan.session)
		if err != nil {
			return err
		}
		if len(snapshot.Servers) != 0 {
			return errors.New("terminal server appeared after retirement")
		}
		rows, err := recoveryRows(snapshot)
		if err != nil {
			return err
		}
		alive := false
		for _, process := range plan.Processes() {
			if _, found := rows[process.PID]; found {
				alive = true
			}
		}
		if !alive {
			return ctx.Err()
		}
		if err := waitRecoveryTerminalPoll(ctx, d.poll); err != nil {
			return fmt.Errorf("accepted terminal owner remains: %w", err)
		}
	}
}

func waitRecoveryTerminalPoll(ctx context.Context, poll time.Duration) error {
	if poll <= 0 {
		poll = 25 * time.Millisecond
	}
	timer := time.NewTimer(poll)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func recoveryServerIdentities(servers []recoveryAttributedServer) []SessionServerIdentity {
	result := make([]SessionServerIdentity, 0, len(servers))
	for _, s := range servers {
		result = append(result, s.Server)
	}
	return result
}
func sameRecoveryServers(a, b []SessionServerIdentity) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[SessionServerIdentity]bool{}
	for _, s := range a {
		if seen[s] {
			return false
		}
		seen[s] = true
	}
	for _, s := range b {
		if !seen[s] {
			return false
		}
		delete(seen, s)
	}
	return len(seen) == 0
}

func attributeRecoveryServers(owner artifactpath.StorageOwner, session string, snapshot recoveryTerminalSnapshot) ([]recoveryAttributedServer, error) {
	var out []recoveryAttributedServer
	if len(snapshot.Servers) > 32 {
		return nil, errors.New("too many contested terminal generations; inspect manually")
	}
	rows := map[int]procutil.Process{}
	for _, p := range snapshot.Processes {
		if _, dup := rows[p.PID]; dup {
			return nil, errors.New("ambiguous process snapshot")
		}
		rows[p.PID] = p
	}
	pids := map[int]bool{}
	for _, server := range snapshot.Servers {
		if !validQuiescenceServer(server, session) || pids[server.PID] || rows[server.PID].Identity != server.Identity {
			return nil, errors.New("terminal server identity is incomplete or changed")
		}
		pids[server.PID] = true
		item := recoveryAttributedServer{Server: server}
		owned := false
		for _, p := range snapshot.Processes {
			if p.PID == server.PID || !recoveryDescendant(server.PID, p.PID, rows) {
				continue
			}
			if !AgentBelongsToServer(server, p.PID, p.Identity, snapshot.Processes) {
				return nil, errors.New("terminal descendant identity unavailable")
			}
			command, ok := snapshot.Commands[p.PID]
			if !ok || command == "" {
				return nil, errors.New("terminal descendant command unavailable; inspect before retirement")
			}
			args, err := splitObservedCommand(command)
			if err == nil && len(args) > 0 {
				if args[0] == "exec" {
					args = args[1:]
				}
				if len(args) > 0 && (filepath.Base(args[0]) == "pair-wrap" || filepath.Base(args[0]) == "pair" && len(args) > 1 && args[1] == "wrap") {
					return nil, errors.New("terminal still has a live Pair wrapper; use normal settled park before recovery")
				}
			}
			observed := ClassifySessionOwner(owner, []zellijpane.Pane{{PaneCommand: command}})
			if observed.State == SessionOwnerForeign {
				return nil, fmt.Errorf("terminal server %d has a foreign Pair owner; manual retirement prerequisite", server.PID)
			}
			if observed.State == SessionOwnerOwned {
				owned = true
			}
			item.Descendants = append(item.Descendants, recoveryProcessWitness{PID: p.PID, PPID: p.PPID, Identity: p.Identity, Command: command})
		}
		if !owned {
			return nil, fmt.Errorf("terminal server %d/%s has no proven live Pair owner; manually retire that exact server before retrying", server.PID, server.Identity)
		}
		sort.Slice(item.Descendants, func(i, j int) bool { return item.Descendants[i].PID < item.Descendants[j].PID })
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Server.PID < out[j].Server.PID })
	return out, nil
}

type recoveryTerminalQuiescence struct {
	driver recoveryTerminalDriver
	plan   RecoveryTerminalPlan
	ctx    context.Context
}

func (o recoveryTerminalQuiescence) SessionPresent(ctx context.Context, session string) (bool, error) {
	return o.driver.ops.SessionPresent(ctx, session)
}
func (o recoveryTerminalQuiescence) remaining(ctx context.Context) (servers []SessionServerIdentity, err error) {
	snapshot, err := o.driver.ops.Observe(ctx, o.plan.session)
	if err != nil {
		return nil, err
	}
	current, err := attributeRecoveryServers(o.plan.owner, o.plan.session, snapshot)
	if err != nil {
		return nil, err
	}
	for _, now := range current {
		found := false
		for _, prior := range o.plan.servers {
			if reflect.DeepEqual(now, prior) {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("terminal ownership changed during retirement; refusing further signals")
		}
	}
	return recoveryServerIdentities(current), nil
}
func (o recoveryTerminalQuiescence) SessionServers(ctx context.Context, session string) ([]SessionServerIdentity, error) {
	if session != o.plan.session {
		return nil, errors.New("recovery session mismatch")
	}
	return o.remaining(ctx)
}
func (o recoveryTerminalQuiescence) KillServer(server SessionServerIdentity) error {
	current, err := o.remaining(o.ctx)
	if err != nil {
		return err
	}
	for _, s := range current {
		if s == server {
			return o.driver.ops.KillServer(server)
		}
	}
	return nil // An authorized server that exited independently needs no signal.
}
func recoveryDescendant(server, pid int, rows map[int]procutil.Process) bool {
	seen := map[int]bool{}
	for pid > 0 && !seen[pid] {
		if pid == server {
			return true
		}
		seen[pid] = true
		p, ok := rows[pid]
		if !ok {
			return false
		}
		pid = p.PPID
	}
	return false
}
