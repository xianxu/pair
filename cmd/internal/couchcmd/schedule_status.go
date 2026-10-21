package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

func scheduleCLIError(stderr io.Writer, err error) int {
	fmt.Fprintln(stderr, "couch: schedule:", rowtext.Sanitize(err.Error()))
	return 1
}

func runScheduleCLI(a *scheduleArgs, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	if a == nil || (a.verb != "status" && a.verb != "history" && a.verb != "run") || (a.verb != "status" && !couchschedule.ValidTaskName(a.task)) || (a.runID != "" && !validScheduleRunID(a.runID)) {
		fmt.Fprintln(stderr, scheduleUsage)
		return 2
	}
	if rt == nil || call == nil {
		return scheduleCLIError(stderr, fmt.Errorf("status transport is unavailable"))
	}
	socket, err := couchmessage.SocketPath(rt.StoreDir(), "broker")
	if err != nil {
		return scheduleCLIError(stderr, err)
	}
	// The query deadline belongs only to transport, never to a manual task.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	var response couchmessage.Response
	err = call(ctx, socket, couchmessage.Request{Op: "schedule-status"}, &response)
	cancel()
	if err != nil {
		return scheduleCLIError(stderr, fmt.Errorf("status query failed: %w", err))
	}
	if response.Code != "ok" || response.Schedule == nil {
		return scheduleCLIError(stderr, fmt.Errorf("status unavailable (%s): %s", response.Code, response.Error))
	}
	snapshot := response.Schedule
	if err = snapshot.Validate(); err != nil {
		return scheduleCLIError(stderr, fmt.Errorf("invalid status response: %w", err))
	}
	if a.verb == "status" {
		if a.json {
			err = json.NewEncoder(stdout).Encode(snapshot)
		} else {
			err = renderScheduleStatus(stdout, *snapshot)
		}
		if err != nil {
			return scheduleCLIError(stderr, err)
		}
		return 0
	}
	if a.verb == "run" && snapshot.Error != "" {
		return scheduleCLIError(stderr, fmt.Errorf("scheduler is blocked: %s", snapshot.Error))
	}
	cwd, err := os.Getwd()
	if err != nil {
		return scheduleCLIError(stderr, err)
	}
	// Keep manual cancellation independent of the completed status RPC.
	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root, err := resolveScheduleHub(runCtx, cwd, snapshot.Hubs, couchcore.ExecGit{})
	if err != nil {
		return scheduleCLIError(stderr, err)
	}
	if a.verb == "history" {
		return runScheduleHistory(runCtx, a, rt.StoreDir(), root, stdout, stderr)
	}
	return runScheduleManual(runCtx, a, rt, root, stdout, stderr)
}

type scheduleHubGitRunner interface {
	RunContext(context.Context, string, ...string) (string, error)
}

// resolveScheduleHub only accepts the current Git checkout's configured hub.
// Claims by a product checkout do not confer scheduler-root authority.
func resolveScheduleHub(ctx context.Context, cwd string, hubs []string, git scheduleHubGitRunner) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("hub resolution requires context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if git == nil {
		git = couchcore.ExecGit{}
	}
	reported, err := git.RunContext(ctx, cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve current Git checkout: %w", err)
	}
	reported = strings.TrimSpace(reported)
	if !filepath.IsAbs(reported) || filepath.Clean(reported) != reported {
		return "", fmt.Errorf("Git returned a noncanonical worktree root")
	}
	root, err := filepath.EvalSymlinks(reported)
	if err != nil {
		return "", fmt.Errorf("canonicalize current checkout: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	for _, hub := range hubs {
		if root == hub {
			return root, nil
		}
	}
	return "", fmt.Errorf("current Git checkout is not a configured scheduler hub")
}

func scheduleTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

func renderScheduleStatus(out io.Writer, s couchschedule.StatusSnapshot) error {
	var b strings.Builder
	if s.Error != "" {
		fmt.Fprintf(&b, "Scheduler blocked: %s\n", rowtext.Sanitize(s.Error))
	}
	if len(s.Tasks) == 0 {
		fmt.Fprintln(&b, "No scheduled tasks.")
	}
	for _, t := range s.Tasks {
		fmt.Fprintf(&b, "%s\t%s\tscope=%s owner=%s\tcron=%s location=%s\t%s next=%s", rowtext.Sanitize(t.Hub), rowtext.Sanitize(t.Task), rowtext.Sanitize(t.Scope), rowtext.Sanitize(t.Owner), rowtext.Sanitize(t.Cron), rowtext.Sanitize(t.Location), rowtext.Sanitize(t.State), scheduleTime(t.NextDue))
		if t.Reason != "" {
			fmt.Fprintf(&b, " reason=%s", rowtext.Sanitize(t.Reason))
		}
		if t.Last != nil {
			fmt.Fprintf(&b, " last=%s source=%s outcome=%s ended=%s notice=%s", rowtext.Sanitize(t.Last.ID), rowtext.Sanitize(t.Last.Source), rowtext.Sanitize(t.Last.Outcome), scheduleTime(t.Last.EndedAt), rowtext.Sanitize(t.Last.Notice))
			if t.Last.Exit != nil {
				fmt.Fprintf(&b, " exit=%d", *t.Last.Exit)
			}
		}
		if t.Active != nil {
			fmt.Fprintf(&b, " active=%s stage=%s deadline=%s", rowtext.Sanitize(t.Active.ID), rowtext.Sanitize(t.Active.Stage), scheduleTime(t.Active.Deadline))
		}
		b.WriteByte('\n')
	}
	_, err := io.WriteString(out, b.String())
	return err
}
