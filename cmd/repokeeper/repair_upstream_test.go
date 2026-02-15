package repokeeper

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/spf13/cobra"
)

func TestNeedsUpstreamRepair(t *testing.T) {
	repo := model.RepoStatus{
		Tracking: model.Tracking{
			Upstream: "origin/main",
			Status:   model.TrackingEqual,
		},
	}
	if needsUpstreamRepair(repo, "origin/main") {
		t.Fatal("expected matching upstream to not need repair")
	}
	if !needsUpstreamRepair(repo, "upstream/main") {
		t.Fatal("expected mismatched upstream to need repair")
	}
	repo.Tracking = model.Tracking{Status: model.TrackingNone}
	if !needsUpstreamRepair(repo, "origin/main") {
		t.Fatal("expected missing tracking to need repair")
	}
	if needsUpstreamRepair(repo, "  ") {
		t.Fatal("expected blank target upstream to skip repair")
	}
}

func TestRepairUpstreamMatchesFilter(t *testing.T) {
	if !repairUpstreamMatchesFilter("origin/main", "origin/main", "") {
		t.Fatal("expected empty filter to match")
	}
	if !repairUpstreamMatchesFilter("origin/main", "origin/main", "all") {
		t.Fatal("expected all filter to match")
	}
	if !repairUpstreamMatchesFilter("", "origin/main", "missing") {
		t.Fatal("expected missing filter match for empty upstream")
	}
	if repairUpstreamMatchesFilter("origin/main", "origin/main", "missing") {
		t.Fatal("did not expect missing filter match for non-empty upstream")
	}
	if !repairUpstreamMatchesFilter("origin/main", "upstream/main", "mismatch") {
		t.Fatal("expected mismatch filter match")
	}
	if repairUpstreamMatchesFilter("origin/main", "origin/main", "mismatch") {
		t.Fatal("did not expect mismatch filter match for equal upstream")
	}
	if !repairUpstreamMatchesFilter("origin/main", "origin/main", "unknown") {
		t.Fatal("expected unknown filters to default to match")
	}
}

func TestWriteRepairUpstreamTable(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	if err := writeRepairUpstreamTable(cmd, []repairUpstreamResult{
		{
			RepoID:          "github.com/org/repo",
			Path:            "/tmp/work/repo",
			LocalBranch:     "main",
			CurrentUpstream: "",
			TargetUpstream:  "origin/main",
			Action:          "would repair",
			OK:              true,
		},
	}, "/tmp/work", nil, false); err != nil {
		t.Fatalf("writeRepairUpstreamTable returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "PATH") || !strings.Contains(got, "TARGET") || !strings.Contains(got, "REPO") {
		t.Fatalf("expected repair table header, got: %q", got)
	}
	if !strings.Contains(got, "repo") || !strings.Contains(got, "would repair") || !strings.Contains(got, "origin/main") {
		t.Fatalf("expected repair row content, got: %q", got)
	}
}

func TestWriteRepairUpstreamTableNoHeaders(t *testing.T) {
	out := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(out)

	if err := writeRepairUpstreamTable(cmd, []repairUpstreamResult{
		{
			RepoID:          "github.com/org/repo",
			Path:            "/tmp/work/repo",
			LocalBranch:     "main",
			CurrentUpstream: "origin/main",
			TargetUpstream:  "origin/main",
			Action:          "unchanged",
			OK:              true,
		},
	}, "/tmp/work", nil, true); err != nil {
		t.Fatalf("writeRepairUpstreamTable returned error: %v", err)
	}

	if strings.Contains(out.String(), "ACTION") {
		t.Fatalf("expected no table headers, got: %q", out.String())
	}
}

func TestWriteRepairUpstreamTableCompactsColumnsOnTinyTTY(t *testing.T) {
	prevIsTerminalFD := isTerminalFD
	prevGetTerminalSize := getTerminalSize
	defer func() {
		isTerminalFD = prevIsTerminalFD
		getTerminalSize = prevGetTerminalSize
	}()
	isTerminalFD = func(int) bool { return true }
	getTerminalSize = func(int) (int, int, error) { return 70, 24, nil }

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe setup failed: %v", err)
	}
	defer reader.Close()

	cmd := &cobra.Command{}
	cmd.SetOut(writer)
	if err := writeRepairUpstreamTable(cmd, []repairUpstreamResult{
		{
			RepoID:          "github.com/org/repo",
			Path:            "/tmp/work/repo",
			LocalBranch:     "main",
			CurrentUpstream: "origin/main",
			TargetUpstream:  "origin/main",
			Action:          "unchanged",
			OK:              true,
		},
	}, "/tmp/work", nil, false); err != nil {
		t.Fatalf("writeRepairUpstreamTable returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "PATH") || !strings.Contains(got, "ACTION") || !strings.Contains(got, "OK") || !strings.Contains(got, "ERROR") {
		t.Fatalf("expected tiny repair headers, got: %q", got)
	}
	if strings.Contains(got, "BRANCH") || strings.Contains(got, "REPO") || strings.Contains(got, "CURRENT") {
		t.Fatalf("expected tiny mode to compact repair columns, got: %q", got)
	}
}

func TestRepairUpstreamRunEDryRunMissingRegistryEntry(t *testing.T) {
	cfgPath, _ := writeTestConfigAndRegistry(t)
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	repairUpstreamCmd.SetOut(out)
	repairUpstreamCmd.SetErr(errOut)
	repairUpstreamCmd.SetContext(context.Background())
	defer repairUpstreamCmd.SetOut(nil)
	defer repairUpstreamCmd.SetErr(nil)

	_ = repairUpstreamCmd.Flags().Set("registry", "")
	_ = repairUpstreamCmd.Flags().Set("dry-run", "true")
	_ = repairUpstreamCmd.Flags().Set("only", "all")
	_ = repairUpstreamCmd.Flags().Set("format", "json")
	_ = repairUpstreamCmd.Flags().Set("no-headers", "false")

	if err := repairUpstreamCmd.RunE(repairUpstreamCmd, nil); err != nil {
		t.Fatalf("repair-upstream dry-run failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "\"repo_id\": \"github.com/org/repo-missing\"") {
		t.Fatalf("expected missing repo in output, got: %q", got)
	}
	if !strings.Contains(got, "\"action\": \"skip missing\"") {
		t.Fatalf("expected skip-missing action in output, got: %q", got)
	}
}

func TestRepairUpstreamRunEDryRunMixedRepoStates(t *testing.T) {
	tmp := t.TempDir()

	repoRepair := filepath.Join(tmp, "repo-repair")
	mustRunGit(t, filepath.Dir(repoRepair), "init", repoRepair)
	mustRunGit(t, repoRepair, "commit", "--allow-empty", "-m", "init")
	mustRunGit(t, repoRepair, "remote", "add", "origin", "git@github.com:org/repo-repair.git")

	repoNoRemote := filepath.Join(tmp, "repo-noremote")
	mustRunGit(t, filepath.Dir(repoNoRemote), "init", repoNoRemote)
	mustRunGit(t, repoNoRemote, "commit", "--allow-empty", "-m", "init")

	repoDetached := filepath.Join(tmp, "repo-detached")
	mustRunGit(t, filepath.Dir(repoDetached), "init", repoDetached)
	mustRunGit(t, repoDetached, "commit", "--allow-empty", "-m", "init")
	mustRunGit(t, repoDetached, "remote", "add", "origin", "git@github.com:org/repo-detached.git")
	mustRunGit(t, repoDetached, "checkout", "--detach", "HEAD")

	notRepoDir := filepath.Join(tmp, "not-repo")
	if err := os.MkdirAll(notRepoDir, 0o755); err != nil {
		t.Fatalf("mkdir non-repo dir: %v", err)
	}

	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-repair", Path: repoRepair, RemoteURL: "git@github.com:org/repo-repair.git", Status: registry.StatusPresent},
			{RepoID: "github.com/org/repo-noremote", Path: repoNoRemote, RemoteURL: "git@github.com:org/repo-noremote.git", Status: registry.StatusPresent},
			{RepoID: "github.com/org/repo-detached", Path: repoDetached, RemoteURL: "git@github.com:org/repo-detached.git", Status: registry.StatusPresent},
			{RepoID: "github.com/org/repo-bad", Path: notRepoDir, RemoteURL: "git@github.com:org/repo-bad.git", Status: registry.StatusPresent},
		},
	}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	repairUpstreamCmd.SetOut(out)
	repairUpstreamCmd.SetErr(errOut)
	repairUpstreamCmd.SetContext(context.Background())
	defer repairUpstreamCmd.SetOut(nil)
	defer repairUpstreamCmd.SetErr(nil)

	_ = repairUpstreamCmd.Flags().Set("registry", "")
	_ = repairUpstreamCmd.Flags().Set("dry-run", "true")
	_ = repairUpstreamCmd.Flags().Set("only", "all")
	_ = repairUpstreamCmd.Flags().Set("format", "json")
	_ = repairUpstreamCmd.Flags().Set("no-headers", "false")

	if err := repairUpstreamCmd.RunE(repairUpstreamCmd, nil); err != nil {
		t.Fatalf("repair-upstream dry-run failed: %v", err)
	}
	got := out.String()
	for _, want := range []string{"\"action\": \"would repair\"", "\"action\": \"skip no remote\"", "\"action\": \"skip detached\"", "\"action\": \"skip status error\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %s in output, got: %q", want, got)
		}
	}
}

func TestRepairUpstreamRunECancelledConfirmation(t *testing.T) {
	tmp := t.TempDir()
	repoRepair := filepath.Join(tmp, "repo-repair")
	mustRunGit(t, filepath.Dir(repoRepair), "init", repoRepair)
	mustRunGit(t, repoRepair, "commit", "--allow-empty", "-m", "init")
	mustRunGit(t, repoRepair, "remote", "add", "origin", "git@github.com:org/repo-repair.git")

	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{RepoID: "github.com/org/repo-repair", Path: repoRepair, RemoteURL: "git@github.com:org/repo-repair.git", Status: registry.StatusPresent},
		},
	}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	in := bytes.NewBufferString("n\n")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	repairUpstreamCmd.SetIn(in)
	repairUpstreamCmd.SetOut(out)
	repairUpstreamCmd.SetErr(errOut)
	repairUpstreamCmd.SetContext(context.Background())
	defer repairUpstreamCmd.SetIn(nil)
	defer repairUpstreamCmd.SetOut(nil)
	defer repairUpstreamCmd.SetErr(nil)

	_ = repairUpstreamCmd.Flags().Set("dry-run", "false")
	_ = repairUpstreamCmd.Flags().Set("only", "all")
	_ = repairUpstreamCmd.Flags().Set("format", "json")
	_ = repairUpstreamCmd.Flags().Set("registry", "")

	if err := repairUpstreamCmd.RunE(repairUpstreamCmd, nil); err != nil {
		t.Fatalf("repair-upstream non-dry-run failed: %v", err)
	}
	if !strings.Contains(errOut.String(), "Proceed with upstream tracking repairs?") {
		t.Fatalf("expected confirmation prompt, got: %q", errOut.String())
	}
}

func TestRepairUpstreamRunERepairedNonDryRun(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	mustRunGit(t, tmp, "init", "--bare", remote)

	work := filepath.Join(tmp, "work")
	mustRunGit(t, tmp, "clone", remote, work)
	mustRunGit(t, work, "checkout", "-b", "main")
	mustRunGit(t, work, "commit", "--allow-empty", "-m", "init")
	mustRunGit(t, work, "push", "-u", "origin", "main")
	mustRunGit(t, work, "branch", "--unset-upstream", "main")

	cfgPath := filepath.Join(tmp, ".repokeeper.yaml")
	cfg := config.DefaultConfig()
	cfg.Registry = &registry.Registry{
		Entries: []registry.Entry{
			{
				RepoID:    "github.com/org/repo-repair",
				Path:      work,
				RemoteURL: remote,
				Status:    registry.StatusPresent,
				Branch:    "main",
			},
		},
	}
	if err := config.Save(&cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	cleanup := withTestConfig(t, cfgPath)
	defer cleanup()

	out := &bytes.Buffer{}
	in := bytes.NewBufferString("y\n")
	repairUpstreamCmd.SetIn(in)
	repairUpstreamCmd.SetOut(out)
	repairUpstreamCmd.SetContext(context.Background())
	defer repairUpstreamCmd.SetIn(nil)
	defer repairUpstreamCmd.SetOut(nil)

	_ = repairUpstreamCmd.Flags().Set("registry", "")
	_ = repairUpstreamCmd.Flags().Set("dry-run", "false")
	_ = repairUpstreamCmd.Flags().Set("only", "all")
	_ = repairUpstreamCmd.Flags().Set("format", "json")
	_ = repairUpstreamCmd.Flags().Set("no-headers", "false")
	_ = repairUpstreamCmd.Flags().Set("yes", "false")

	if err := repairUpstreamCmd.RunE(repairUpstreamCmd, nil); err != nil {
		t.Fatalf("repair-upstream non-dry-run failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "\"action\": \"repaired\"") {
		t.Fatalf("expected repaired action in output, got: %q", got)
	}
}
