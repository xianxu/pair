package historyarchive_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ha "github.com/xianxu/pair/cmd/internal/historyarchive"
	"github.com/xianxu/pair/cmd/internal/historyarchive/archivetest"
)

type restoreMissingDownloadRunner struct{ fake *archivetest.FakeRclone }

func (r restoreMissingDownloadRunner) Run(ctx context.Context, args []string, limit int) ([]byte, error) {
	if args[0] == "copy" {
		return nil, nil
	}
	return r.fake.Run(ctx, args, limit)
}
func TestRestoreMissingDownloadedCiphertextNamesExpectedPathAndSource(t *testing.T) {
	for _, month := range []bool{false, true} {
		t.Run(map[bool]string{false: "root", true: "month"}[month], func(t *testing.T) {
			r, req, records, fake, decoder := driveRestoreFixture(t, 1)
			if month {
				r, req, records, fake, decoder = monthRestoreFixture(t, 1)
			}
			index, err := ha.ReadLabelIndex(r.Config.Catalog + ".index.json")
			if err != nil {
				t.Fatal(err)
			}
			locator := ""
			for _, loc := range index.Records[records[0].Label.ID].Locations {
				locator = loc.Locator
			}
			r.Rclone.Runner = restoreMissingDownloadRunner{fake: fake}
			report, err := r.Run(context.Background(), req)
			assertDriveRestoreRefused(t, report, err, req, decoder)
			expected := filepath.Join(filepath.Dir(req.To), ".pair-restore-staging", "ciphertext", records[0].Label.ID+".tar.zst.gpg")
			if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), expected) || !strings.Contains(err.Error(), locator) || !strings.Contains(err.Error(), records[0].Label.ID) {
				t.Fatalf("missing acquisition context: %v; want path=%s source=%s", err, expected, locator)
			}
		})
	}
}

func TestRestoreDriveAcquisitionErrorsRetainOperationAndCause(t *testing.T) {
	for _, phase := range []string{"resolve-root", "list-root", "list-year", "list-month", "download"} {
		t.Run(phase, func(t *testing.T) {
			r, req, records, fake, decoder := monthRestoreFixture(t, 1)
			root, err := ha.ParseDriveDestination(r.Config.DriveRemote, r.Config.DriveFolder)
			if err != nil {
				t.Fatal(err)
			}
			month, err := (ha.DriveArchiveLayout{Root: root}).MonthDestination(records[0].Label)
			if err != nil {
				t.Fatal(err)
			}
			cause := errors.New("private injected acquisition failure")
			rootLists := 0
			fake.BeforeFolder = func(_ context.Context, operation, folder string) error {
				if operation == "lsjson" && folder == "folder-1" {
					rootLists++
				}
				fail := phase == "resolve-root" && operation == "lsjson" && folder == "folder-1" && rootLists == 1 || phase == "list-root" && operation == "lsjson" && folder == "folder-1" && rootLists == 2 || phase == "list-year" && operation == "lsjson" && folder == "year" || phase == "list-month" && operation == "lsjson" && folder == "month" || phase == "download" && operation == "copy"
				if fail {
					return cause
				}
				return nil
			}
			report, err := r.Run(context.Background(), req)
			assertDriveRestoreRefused(t, report, err, req, decoder)
			locator := month.String()
			operation := "list"
			if phase == "resolve-root" {
				locator = root.String()
				operation = "resolve"
			}
			if phase == "list-root" {
				locator = root.String()
			}
			if phase == "download" {
				operation = "download"
			}
			if !errors.Is(err, cause) || !strings.Contains(err.Error(), locator) || !strings.Contains(err.Error(), operation) {
				t.Fatalf("missing %s operation/location/cause: %v", phase, err)
			}
		})
	}
}

func TestRestoreMissingLocalCiphertextNamesExpectedPath(t *testing.T) {
	r, req, records, fake, decoder := driveRestoreFixture(t, 1)
	req.From = "local"
	report, err := r.Run(context.Background(), req)
	assertDriveRestoreRefused(t, report, err, req, decoder)
	path := filepath.Join(r.Config.Sink, records[0].BundleKey)
	if !errors.Is(err, os.ErrNotExist) || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), records[0].Label.ID) {
		t.Fatalf("missing local source context: %v; want path=%s", err, path)
	}
	if fake.Invocations != 0 {
		t.Fatal("missing local-only source reached Drive")
	}
}
