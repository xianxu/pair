package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

// orphanNvimSection heads the #583 sweep's rows. The header states the reap's
// gate because the rows are the evidence an operator confirms against.
const orphanNvimSection = "Orphan nvim (untagged; reap needs --confirm)"

// orphanNvimRow is one sweep row in the one shape every surface prints: the
// report an operator reads, the refusal that names what a reap would have ended,
// and the outcome line of a reap that ran. Two renderings of one row would let
// the evidence and the thing it justified drift apart (#583).
func orphanNvimRow(o couchcore.OrphanNvim) string {
	return fmt.Sprintf("pid=%d age=%s RSS=%s cwd=%s root=%s %s", o.Key.PID, o.Age.Round(time.Minute), resourceBytes(o.RSSBytes), rowtext.Sanitize(o.Cwd), rowtext.Sanitize(o.Root), rowtext.Sanitize(o.Reason))
}

// renderOrphanNvimRows prints the sweep under its one header. An empty sweep
// still says it looked: an absent section reads as "not checked".
func renderOrphanNvimRows(out io.Writer, rows []couchcore.OrphanNvim) {
	fmt.Fprintf(out, "%s:\n", orphanNvimSection)
	if len(rows) == 0 {
		fmt.Fprintln(out, "  none observed")
	}
	for _, o := range rows {
		fmt.Fprintf(out, "  %s\n", orphanNvimRow(o))
	}
}

// leftRunningSection heads the registry's rows (pair#695): processes an agent
// started that had left its terminal when teardown ran. Teardown never kills
// them; the reap does, by exact identity, once confirmed.
const leftRunningSection = "Left running by teardown (reap needs --confirm)"

func leftRunningRow(o couchcore.LeftRunningOrphan) string {
	return fmt.Sprintf("pid=%d %s thread=%s/%s left=%s origin=%s", o.Process.PID, rowtext.Sanitize(o.Command), rowtext.Sanitize(o.Thread.RepoScope), rowtext.Sanitize(string(o.Thread.Tag)), o.LeftAt.Format(time.RFC3339), rowtext.Sanitize(o.Origin))
}

func renderLeftRunningRows(out io.Writer, rows []couchcore.LeftRunningOrphan) {
	fmt.Fprintf(out, "%s:\n", leftRunningSection)
	if len(rows) == 0 {
		fmt.Fprintln(out, "  none registered")
	}
	for _, o := range rows {
		fmt.Fprintf(out, "  %s\n", leftRunningRow(o))
	}
}

func runResourcesCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer, call messageCall) int {
	socket, err := couchmessage.SocketPath(rt.StoreDir(), "broker")
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), couchmessage.AdmissionTimeout)
	defer cancel()
	var result couchmessage.Response
	if err := call(ctx, socket, couchmessage.Request{Op: "resources"}, &result); err != nil {
		fmt.Fprintf(stderr, "couch: resource query failed (%v); inspect the running Couch\n", err)
		return 1
	}
	if result.Code != "ok" || result.Resources == nil {
		fmt.Fprintf(stderr, "couch: resources unavailable: %s; inspect the running Couch\n", rowtext.Sanitize(result.Error))
		return 1
	}
	if inv.jsonOutput {
		if err := json.NewEncoder(stdout).Encode(result.Resources); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	renderResources(stdout, *result.Resources, time.Now())
	return 0
}

func renderResources(out io.Writer, v couchcore.ResourceView, now time.Time) {
	if v.SampledAt.IsZero() {
		fmt.Fprintln(out, "Resources: waiting for first sample")
	} else {
		fmt.Fprintf(out, "Resources sampled %s (%s ago)\n", v.SampledAt.Format(time.RFC3339), now.Sub(v.SampledAt).Round(time.Second))
	}
	if v.HistoryFrames > 0 {
		fmt.Fprintf(out, "History: %d frames; oldest %s\n", v.HistoryFrames, v.HistoryOldest.Format(time.RFC3339))
	} else {
		fmt.Fprintln(out, "History: no retained frames")
	}
	if h := v.HostedIngest; h == nil {
		fmt.Fprintln(out, "Hosted ingest: unavailable (server does not provide diagnostics)")
	} else {
		fmt.Fprintf(out, "Hosted ingest: %d clients; %d omitted; sampled %s; server pid=%d started=%s\n", len(h.Clients), h.Omitted, h.SampledAt.Format(time.RFC3339Nano), h.ServerPID, h.ServerStartedAt.Format(time.RFC3339Nano))
		fmt.Fprintln(out, "ENDPOINT  THREAD  PROCESS  cumulative ingest bytes")
		for _, client := range h.Clients {
			fmt.Fprintf(out, "%s  %s/%s  pid=%d generation=%s  %d\n", rowtext.Sanitize(client.EndpointID), rowtext.Sanitize(client.Thread.RepoScope), rowtext.Sanitize(string(client.Thread.Tag)), client.Process.PID, rowtext.Sanitize(client.Process.Identity), client.IngestBytes)
		}
	}
	fmt.Fprintln(out, "SLOT  CPU%  RSS bytes  WRITE bytes/s  PROCESSES  OBSERVED CPU s / WRITE bytes")
	for _, s := range v.Slots {
		fmt.Fprintf(out, "%s  %s  %s  %s  %d", rowtext.Sanitize(s.Slot), resourceNumber(s.CPUPercent), resourceBytes(s.RSSBytes), resourceNumber(s.WriteBytesPerSecond), s.ProcessCount)
		fmt.Fprintf(out, "  %.3f / %d", float64(s.ObservedCPUTimeNS)/1e9, s.ObservedDiskWriteBytes)
		if s.Reason != "" {
			fmt.Fprintf(out, " (%s)", rowtext.Sanitize(s.Reason))
		}
		fmt.Fprintln(out)
	}
	for _, section := range []struct {
		name string
		rows []couchcore.ResourceProcessView
	}{{"Proven orphans", v.Orphans}, {"Unattributed (unproven; report only)", v.Unattributed}} {
		fmt.Fprintln(out, section.name+":")
		if len(section.rows) == 0 {
			fmt.Fprintln(out, "  none observed")
		}
		for _, p := range section.rows {
			fmt.Fprintf(out, "  %s pid=%d %s RSS=%s %s\n", rowtext.Sanitize(p.Slot), p.Key.PID, rowtext.Sanitize(p.Name), resourceBytes(p.RSSBytes), rowtext.Sanitize(p.Reason))
		}
	}
	renderOrphanNvimRows(out, v.OrphanNvim)
	renderLeftRunningRows(out, v.LeftRunning)
	fmt.Fprintf(out, "Observed since sampler start: CPU %.3fs; disk writes %d bytes (sampled, not lifetime billing)\n", float64(v.ObservedCPUTimeNS)/1e9, v.ObservedDiskWriteBytes)
	cpuCost := "unavailable"
	if v.Cost.CPUKnown {
		cpuCost = fmt.Sprintf("%.3fms", float64(v.Cost.CPUTimeNS)/1e6)
	}
	fmt.Fprintf(out, "Sampler: %d native attempts (table %d, usage %d, command %d, cwd %d); CPU %s; elapsed %s\n", v.Cost.NativeCalls, v.Cost.TableCalls, v.Cost.UsageCalls, v.Cost.CommandCalls, v.Cost.CwdCalls, cpuCost, v.Cost.WallTime)
	if !v.Cost.CPUKnown {
		fmt.Fprintln(out, "Sampler CPU measurement unavailable")
	}
	for _, e := range v.Episodes {
		if e.Phase == "pending" && e.Effect != nil {
			fmt.Fprintf(out, "Alert %s: pending or delivery unknown; inspect this snapshot\n", rowtext.Sanitize(e.Effect.ID))
		}
	}
	for _, e := range v.Errors {
		fmt.Fprintln(out, "Coverage:", rowtext.Sanitize(e))
	}
}
func resourceNumber(v *float64) string {
	if v == nil {
		return "unavailable"
	}
	return fmt.Sprintf("%.1f", *v)
}
func resourceBytes(v *uint64) string {
	if v == nil {
		return "unavailable"
	}
	return fmt.Sprint(*v)
}
