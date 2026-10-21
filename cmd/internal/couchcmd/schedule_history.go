package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

func runScheduleHistory(ctx context.Context, a *scheduleArgs, namespace, root string, stdout, stderr io.Writer) int {
	if a == nil || !couchschedule.ValidTaskName(a.task) || (a.runID != "" && !validScheduleRunID(a.runID)) {
		return scheduleCLIError(stderr, fmt.Errorf("invalid history task or run ID"))
	}
	key := operationalrun.RunKey{Hub: (couchschedule.TaskKey{Hub: root, Task: a.task}).HubID(), Task: a.task, ID: a.runID}
	store := operationalrun.Store{Namespace: namespace}
	var err error
	if a.runID != "" {
		var record operationalrun.Record
		var output []byte
		record, output, err = store.Read(ctx, key)
		if err == nil {
			if a.json {
				err = json.NewEncoder(stdout).Encode(struct {
					Record operationalrun.Record `json:"record"`
					Output string                `json:"output"`
				}{record, string(output)})
			} else {
				var b strings.Builder
				renderScheduleRecord(&b, record)
				for _, line := range strings.Split(string(output), "\n") {
					fmt.Fprintln(&b, rowtext.Sanitize(line))
				}
				_, err = io.WriteString(stdout, b.String())
			}
		}
	} else {
		var records []operationalrun.Record
		records, err = store.List(ctx, key.Hub, key.Task, 300)
		if err == nil {
			if records == nil {
				records = []operationalrun.Record{}
			}
			if a.json {
				err = json.NewEncoder(stdout).Encode(struct {
					Records []operationalrun.Record `json:"records"`
				}{records})
			} else {
				var b strings.Builder
				if len(records) == 0 {
					fmt.Fprintln(&b, "No retained scheduled runs.")
				}
				for _, record := range records {
					renderScheduleRecord(&b, record)
				}
				_, err = io.WriteString(stdout, b.String())
			}
		}
	}
	if err != nil {
		return scheduleCLIError(stderr, fmt.Errorf("history: %w", err))
	}
	return 0
}

func renderScheduleRecord(out io.Writer, r operationalrun.Record) {
	exit := "-"
	if r.Exit != nil {
		exit = fmt.Sprint(*r.Exit)
	}
	fmt.Fprintf(out, "hub=%s task=%s run=%s revision=%s source=%s scheduled=%s started=%s ended=%s exit=%s outcome=%s notice=%s\n", rowtext.Sanitize(r.Key.Hub), rowtext.Sanitize(r.Key.Task), rowtext.Sanitize(r.Key.ID), rowtext.Sanitize(r.DeclarationRevision), rowtext.Sanitize(r.Source), scheduleTime(r.ScheduledAt), scheduleTime(r.StartedAt), scheduleTime(r.EndedAt), exit, rowtext.Sanitize(r.Outcome), rowtext.Sanitize(r.Notice))
	var diagnostic string
	if raw := r.Unknown["error"]; len(raw) > 0 && json.Unmarshal(raw, &diagnostic) == nil {
		fmt.Fprintf(out, "error=%s\n", rowtext.Sanitize(diagnostic))
	}
}
