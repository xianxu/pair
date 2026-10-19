package couchcmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/historyarchive"
)

// The subprocess fixture owns stdout/stderr descriptors and a separate control
// connection. A held descriptor is released explicitly, independent of timers.
func TestArchiveProtocolProcess(t *testing.T) {
	mode := os.Getenv("PAIR_ARCHIVE_PROTOCOL_MODE")
	if mode == "" {
		return
	}
	connect := func() net.Conn {
		conn, err := net.Dial("tcp", os.Getenv("PAIR_ARCHIVE_PROTOCOL_GATE"))
		if err != nil {
			os.Exit(81)
		}
		fmt.Fprintf(conn, "%d\n", os.Getpid())
		return conn
	}
	if mode == "child" {
		conn := connect()
		var token [1]byte
		_, _ = conn.Read(token[:])
		os.Stdout.Close()
		os.Stderr.Close()
		conn.Close()
		os.Exit(0)
	}
	fixture := os.Getenv("PAIR_ARCHIVE_PROTOCOL_FIXTURE")
	record := struct {
		Args []string
		Data string
	}{Data: os.Getenv("PAIR_DATA_DIR")}
	for i, arg := range os.Args {
		if arg == "--" {
			record.Args = os.Args[i+1:]
			break
		}
	}
	raw, _ := json.Marshal(record)
	if err := os.WriteFile(filepath.Join(fixture, "invocation"), raw, 0600); err != nil {
		os.Exit(82)
	}
	starts, err := os.OpenFile(filepath.Join(fixture, "starts"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		os.Exit(83)
	}
	starts.WriteString("start\n")
	starts.Close()
	if mode == "hold" {
		child := exec.Command(os.Args[0], "-test.run=^TestArchiveProtocolProcess$")
		child.Env = append(os.Environ(), "PAIR_ARCHIVE_PROTOCOL_MODE=child")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(84)
		}
	}
	raw, err = os.ReadFile(filepath.Join(fixture, "report"))
	if err != nil {
		os.Exit(85)
	}
	fmt.Fprint(os.Stderr, "fixture diagnostic: not JSON\n")
	if _, err := os.Stdout.Write(raw); err != nil {
		os.Exit(86)
	}
	if mode == "wait" {
		conn := connect()
		var token [1]byte
		_, _ = conn.Read(token[:])
		conn.Close()
	}
	code, _ := strconv.Atoi(os.Getenv("PAIR_ARCHIVE_PROTOCOL_EXIT"))
	os.Exit(code)
}

func archiveProtocolFixture(t *testing.T, raw []byte, mode string, exit int) (string, string) {
	t.Helper()
	fixture, data := t.TempDir(), t.TempDir()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_ARCHIVE_PROTOCOL_BINARY", binary)
	t.Setenv("PAIR_ARCHIVE_PROTOCOL_FIXTURE", fixture)
	t.Setenv("PAIR_ARCHIVE_PROTOCOL_MODE", mode)
	t.Setenv("PAIR_ARCHIVE_PROTOCOL_EXIT", strconv.Itoa(exit))
	t.Setenv("PAIR_DATA_DIR", "wrong-inherited-data")
	t.Setenv("PATH", fixture+string(os.PathListSeparator)+os.Getenv("PATH"))
	shim := "#!/bin/sh\nexec \"$PAIR_ARCHIVE_PROTOCOL_BINARY\" -test.run=^TestArchiveProtocolProcess$ -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(fixture, "pair"), []byte(shim), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "report"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	return fixture, data
}

func archiveProtocolReport(t *testing.T, outcome string) (historyarchive.ContinuousCommandReport, []byte) {
	t.Helper()
	want := historyarchive.ContinuousCommandReport{Version: historyarchive.ContinuousReportVersion, Outcome: outcome, MaxDatabaseAge: 48 * time.Hour, Report: historyarchive.ContinuousReport{Databases: []historyarchive.DatabaseBackupSummary{{Source: historyarchive.SourceIdentity{Host: "fixture", Root: "/native"}, Kind: "codex-database", Path: "partial.sqlite", Status: historyarchive.DatabaseDeferred, Detail: "busy"}}}}
	want.MaxDriveFiles = 20000
	want.Report.Drive = &historyarchive.DriveReport{Inventory: &historyarchive.DriveInventoryObservation{Destination: "remote:configured-root", FolderID: "root-1", Phase: "before", Files: 20001}}
	raw, err := historyarchive.EncodeContinuousReport(want)
	if err != nil {
		t.Fatal(err)
	}
	// Unknown fields and whitespace must survive logging byte-for-byte.
	raw = bytes.Replace(raw, []byte(`"report":{`), []byte(`"future":{"opaque":true}, "report":{`), 1)
	return want, append([]byte(" \n"), raw...)
}

func TestRunArchivePassProtocol(t *testing.T) {
	for _, tc := range []struct {
		name, outcome string
		exit          int
		wantErr       bool
	}{{"complete", historyarchive.ContinuousComplete, 0, false}, {"failed exit", historyarchive.ContinuousFailed, 7, true}, {"failed successful exit", historyarchive.ContinuousFailed, 0, true}} {
		t.Run(tc.name, func(t *testing.T) {
			want, raw := archiveProtocolReport(t, tc.outcome)
			fixture, data := archiveProtocolFixture(t, raw, "normal", tc.exit)
			got, err := runArchivePass(context.Background(), data, fixture)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("partial report changed: got %#v want %#v", got, want)
			}
			var invocation struct {
				Args []string
				Data string
			}
			record, err := os.ReadFile(filepath.Join(fixture, "invocation"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(record, &invocation); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(invocation.Args, []string{"archive", "continuous", "--json"}) || invocation.Data != data {
				t.Fatalf("invocation=%+v", invocation)
			}
			starts, err := os.ReadFile(filepath.Join(fixture, "starts"))
			if err != nil || string(starts) != "start\n" {
				t.Fatalf("state=%q error=%v", starts, err)
			}
			log, err := os.ReadFile(filepath.Join(fixture, "continuous.log"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(log, append([]byte("fixture diagnostic: not JSON\n"), raw...)) {
				t.Fatalf("raw protocol bytes or diagnostic lost: %q", log)
			}
		})
	}
}

func TestRunArchivePassRefusesInvalidProtocol(t *testing.T) {
	_, valid := archiveProtocolReport(t, historyarchive.ContinuousComplete)
	padded := append(append([]byte(nil), valid...), bytes.Repeat([]byte(" "), historyarchive.MaxContinuousReportBytes+1-len(valid))...)
	for name, raw := range map[string][]byte{"empty": nil, "malformed": []byte("{broken"), "trailing": append(append([]byte(nil), valid...), []byte("{}")...), "oversize": []byte(strings.Repeat("x", historyarchive.MaxContinuousReportBytes+1)), "oversize-valid-prefix": padded} {
		t.Run(name, func(t *testing.T) {
			fixture, data := archiveProtocolFixture(t, raw, "normal", 0)
			got, err := runArchivePass(context.Background(), data, fixture)
			if err == nil {
				t.Fatal("accepted invalid stdout")
			}
			if !reflect.DeepEqual(got, historyarchive.ContinuousCommandReport{}) {
				t.Fatalf("invalid stdout granted report authority: %#v", got)
			}
			log, readErr := os.ReadFile(filepath.Join(fixture, "continuous.log"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.HasPrefix(log, []byte("fixture diagnostic: not JSON\n")) {
				t.Fatal("diagnostic missing")
			}
			if !strings.HasPrefix(name, "oversize") && !bytes.HasSuffix(log, raw) {
				t.Fatal("original stdout missing")
			}
			if strings.HasPrefix(name, "oversize") && len(log) > historyarchive.MaxContinuousReportBytes+len("fixture diagnostic: not JSON\n") {
				t.Fatalf("oversize capture was unbounded: log bytes=%d, capture cap=%d", len(log), historyarchive.MaxContinuousReportBytes)
			}
		})
	}
}

func TestRunArchivePassCancellationAndInheritedDescriptors(t *testing.T) {
	for _, mode := range []string{"wait", "hold"} {
		t.Run(mode, func(t *testing.T) {
			want, raw := archiveProtocolReport(t, historyarchive.ContinuousFailed)
			fixture, data := archiveProtocolFixture(t, raw, mode, 0)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			t.Setenv("PAIR_ARCHIVE_PROTOCOL_GATE", listener.Addr().String())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			type result struct {
				report historyarchive.ContinuousCommandReport
				err    error
			}
			done := make(chan result, 1)
			go func() { report, err := runArchivePass(ctx, data, fixture); done <- result{report, err} }()
			// This deadline only guards a hung test. Readiness comes from the process.
			if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
				t.Fatal(err)
			}
			conn, err := listener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
				t.Fatal(err)
			}
			reader := bufio.NewReader(conn)
			ready, err := reader.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			pid, err := strconv.Atoi(strings.TrimSpace(ready))
			if err != nil || pid <= 0 {
				t.Fatalf("invalid owned process handshake %q", ready)
			}
			process, err := os.FindProcess(pid)
			if err != nil {
				t.Fatal(err)
			}
			defer process.Release()
			// On a test failure, kill only the exact child introduced by this fixture.
			stopped := false
			defer func() {
				if !stopped {
					_ = process.Kill()
				}
			}()
			if mode == "wait" {
				cancel()
			}
			var got result
			select {
			case got = <-done:
			case <-time.After(20 * time.Second):
				t.Fatal("command runner did not finish")
			}
			if got.err == nil {
				t.Fatal("expected cancellation or inherited-descriptor failure")
			}
			if mode == "hold" && !errors.Is(got.err, exec.ErrWaitDelay) {
				t.Fatalf("inherited descriptor error=%v", got.err)
			}
			if !reflect.DeepEqual(got.report, want) {
				t.Fatalf("valid partial report lost: %#v", got.report)
			}
			log, err := os.ReadFile(filepath.Join(fixture, "continuous.log"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasSuffix(log, raw) || !bytes.HasPrefix(log, []byte("fixture diagnostic: not JSON\n")) {
				t.Fatal("cancelled or inherited-descriptor result lost original stdout or diagnostics")
			}
			if mode == "hold" {
				if _, err := conn.Write([]byte{1}); err != nil {
					t.Fatal(err)
				}
			}
			if err := conn.SetReadDeadline(time.Now().Add(20 * time.Second)); err != nil {
				t.Fatal(err)
			}
			if _, err := reader.ReadByte(); !errors.Is(err, io.EOF) {
				t.Fatalf("fixture did not close its owned descriptors: %v", err)
			}
			stopped = true
		})
	}
}
