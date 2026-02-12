package repokeeper

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" a, ,b,c ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected split result: %#v", got)
	}
}

func TestHasRegistryWarnings(t *testing.T) {
	reg := &registry.Registry{Entries: []registry.Entry{{Status: registry.StatusPresent}}}
	if hasRegistryWarnings(reg) {
		t.Fatal("did not expect warnings")
	}
	reg.Entries[0].Status = registry.StatusMissing
	if !hasRegistryWarnings(reg) {
		t.Fatal("expected warnings")
	}
}

func TestStatusHasWarningsOrErrors(t *testing.T) {
	report := &model.StatusReport{Repos: []model.RepoStatus{{RepoID: "r1"}}}
	reg := &registry.Registry{}
	if statusHasWarningsOrErrors(report, reg) {
		t.Fatal("did not expect warnings")
	}
	report.Repos[0].Error = "boom"
	if !statusHasWarningsOrErrors(report, reg) {
		t.Fatal("expected warnings")
	}
}

func TestWriters(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	writeScanTable(cmd, []model.RepoStatus{{RepoID: "r1", Path: "/repo", Bare: true, PrimaryRemote: "origin"}})
	if !strings.Contains(out.String(), "PRIMARY_REMOTE") {
		t.Fatal("expected scan header")
	}

	out.Reset()
	writeStatusTable(cmd, &model.StatusReport{Repos: []model.RepoStatus{{RepoID: "r1", Path: "/repo", Tracking: model.Tracking{Status: model.TrackingNone}}}})
	if !strings.Contains(out.String(), "TRACKING") {
		t.Fatal("expected status header")
	}

	out.Reset()
	writeSyncTable(cmd, []engine.SyncResult{{RepoID: "r1", OK: false, ErrorClass: "network", Error: "x"}})
	if !strings.Contains(out.String(), "ERROR_CLASS") {
		t.Fatal("expected sync header")
	}
}

func TestLogHelpers(t *testing.T) {
	cmd := &cobra.Command{}
	errOut := &bytes.Buffer{}
	cmd.SetErr(errOut)

	flagQuiet = false
	flagVerbose = 1
	infof(cmd, "hello %s", "info")
	debugf(cmd, "hello %s", "debug")
	if !strings.Contains(errOut.String(), "hello info") || !strings.Contains(errOut.String(), "hello debug") {
		t.Fatal("expected both info and debug logs")
	}
}

func TestPrompt(t *testing.T) {
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	reader := bufio.NewReader(strings.NewReader("custom\n"))
	got := prompt(reader, cmd, "Label", "default")
	if got != "custom" {
		t.Fatalf("unexpected prompt value: %q", got)
	}

	reader = bufio.NewReader(strings.NewReader("\n"))
	got = prompt(reader, cmd, "Label", "default")
	if got != "default" {
		t.Fatalf("expected default, got %q", got)
	}
}
