package wrapcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/titlepoller"
)

// wrapperTitleProcess owns the sole writer and joins its exact direct child.
// The writer is CLOEXEC and never appears in ExtraFiles for any child.
type wrapperTitleProcess struct {
	writer   *os.File
	cmd      *exec.Cmd
	identity string
	done     chan error
}

var startTitleProcess = startWrapperTitleProcess

func startWrapperTitleProcess(agent, scrollbackLog string, stderr io.Writer) (*wrapperTitleProcess, error) {
	if launcher.CouchHostedEnv(os.Getenv) {
		return nil, nil
	}
	tag, scope, data, session := os.Getenv("PAIR_TAG"), os.Getenv("PAIR_SCOPE_KEY"), os.Getenv("PAIR_DATA_DIR"), os.Getenv("PAIR_SESSION_NAME")
	if !launcher.IsSupportedAgent(agent) || os.Getenv("PAIR_AGENT") != agent || tag == "" || scope == "" || data == "" || session == "" {
		return nil, nil
	}
	if err := launcher.ValidatePairTag(tag); err != nil {
		return nil, err
	}
	if err := launcher.ValidateRepoScopeKey(scope); err != nil {
		return nil, err
	}
	paths, err := artifactpath.ResolveScoped(data, tag)
	if err != nil {
		return nil, err
	}
	capture, err := paths.ScrollbackArtifacts(agent)
	if err != nil {
		return nil, err
	}
	if scrollbackLog == "" || scrollbackLog != capture.Raw {
		return nil, nil
	}
	owner, err := titlepoller.ParseTitleOwner(strconv.Itoa(os.Getpid()), procutil.Identity(strconv.Itoa(os.Getpid())))
	if err != nil {
		return nil, err
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	exe, err := os.Executable()
	if err != nil {
		_ = writer.Close()
		return nil, err
	}
	cmd := exec.Command(exe, "title", tag, agent, session)
	cmd.Env = os.Environ()
	for _, entry := range append(titlepoller.NewSessionEnv(data, scope).Environ(), owner.Environ()...) {
		// setEnv replaces inherited duplicates without shell interpretation.
		for i := 0; i < len(entry); i++ {
			if entry[i] == '=' {
				cmd.Env = setEnv(cmd.Env, entry[:i], entry[i+1:])
				break
			}
		}
	}
	cmd.ExtraFiles = []*os.File{reader}
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = writer.Close()
		return nil, err
	}
	p := &wrapperTitleProcess{writer: writer, cmd: cmd, identity: procutil.Identity(strconv.Itoa(cmd.Process.Pid)), done: make(chan error, 1)}
	go func() { p.done <- cmd.Wait() }()
	return p, nil
}

func (p *wrapperTitleProcess) Close() error {
	if p == nil {
		return nil
	}
	err := p.writer.Close()
	select {
	case <-p.done:
		return err
	case <-time.After(3 * time.Second):
	}
	// A title that cannot execute its watcher still owns the command group we
	// created with Setsid. Fence that group through our still-unjoined child;
	// numeric pidfiles grant no authority here.
	if p.identity == "" || procutil.Identity(strconv.Itoa(p.cmd.Process.Pid)) != p.identity {
		return errors.Join(err, errors.New("title shutdown identity unavailable"))
	}
	pgid, groupErr := syscall.Getpgid(p.cmd.Process.Pid)
	if groupErr != nil || pgid != p.cmd.Process.Pid || procutil.Identity(strconv.Itoa(p.cmd.Process.Pid)) != p.identity {
		return errors.Join(err, errors.New("title shutdown group ownership unavailable"), groupErr)
	}
	if killErr := syscall.Kill(-pgid, syscall.SIGKILL); killErr != nil {
		return errors.Join(err, killErr)
	}
	select {
	case <-p.done:
		return errors.Join(err, fmt.Errorf("title required forced shutdown"))
	case <-time.After(time.Second):
		return errors.Join(err, fmt.Errorf("title shutdown not confirmed"))
	}
}
