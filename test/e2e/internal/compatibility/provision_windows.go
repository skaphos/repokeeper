// SPDX-License-Identifier: MIT
//go:build integration && windows

package compatibility

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Provision(parent context.Context, cell Cell, prefix string) (ProvisionResult, error) {
	ctx, cancel := context.WithTimeout(parent, ProvisionTimeout)
	defer cancel()
	absPrefix, err := filepath.Abs(prefix)
	if err != nil {
		return ProvisionResult{}, err
	}
	if err := os.MkdirAll(absPrefix, 0o750); err != nil {
		return ProvisionResult{}, err
	}
	if cell.Environment == "windows" {
		return provisionMinGit(ctx, cell, absPrefix)
	}
	if cell.Environment == "wsl" {
		return provisionWSL(ctx, cell, absPrefix)
	}
	return ProvisionResult{}, fmt.Errorf("cell %s cannot run on Windows", CellKey(cell))
}

func provisionMinGit(ctx context.Context, cell Cell, prefix string) (ProvisionResult, error) {
	archive := filepath.Join(prefix, "mingit.zip")
	if err := downloadVerified(ctx, cell.Provisioner.SourceURL, cell.Provisioner.SHA256, archive); err != nil {
		return ProvisionResult{}, err
	}
	install := filepath.Join(prefix, "install")
	if err := extractZip(archive, install); err != nil {
		return ProvisionResult{}, err
	}
	gitPath := filepath.Join(install, "cmd", "git.exe")
	version, err := VerifyVersion(ctx, cell, gitPath)
	return ProvisionResult{CellKey: CellKey(cell), Prefix: prefix, GitPath: gitPath, ExpectedVersion: version.ExpectedVersion, ActualVersion: version.ActualVersion, SourceURL: cell.Provisioner.SourceURL, SHA256: cell.Provisioner.SHA256}, err
}

func provisionWSL(ctx context.Context, cell Cell, prefix string) (ProvisionResult, error) {
	if cell.RootFS == nil {
		return ProvisionResult{}, fmt.Errorf("cell %s has no WSL rootfs", CellKey(cell))
	}
	rootfs := filepath.Join(prefix, "rootfs.tar.gz")
	if err := downloadVerified(ctx, cell.RootFS.SourceURL, cell.RootFS.SHA256, rootfs); err != nil {
		return ProvisionResult{}, err
	}
	name := "repokeeper-" + strings.ReplaceAll(CellKey(cell), ".", "-")
	installRoot := filepath.Join(prefix, "distribution")
	if err := runProvisionCommand(ctx, prefix, "wsl.exe", "--import", name, installRoot, rootfs, "--version", "1"); err != nil {
		return ProvisionResult{}, err
	}
	failed := true
	defer func() {
		if failed {
			_ = runProvisionCommand(context.Background(), prefix, "wsl.exe", "--unregister", name)
		}
	}()
	packages := make([]string, 0, len(cell.RootFS.BuildPrerequisites))
	for _, prerequisite := range cell.RootFS.BuildPrerequisites {
		packages = append(packages, prerequisite.Name+"="+prerequisite.Version)
	}
	installPrerequisites := fmt.Sprintf("set -eu; printf 'deb %s noble main universe\\n' > /etc/apt/sources.list; apt-get -o Acquire::Check-Valid-Until=false update; DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends %s", cell.RootFS.Snapshot, strings.Join(packages, " "))
	if err := runProvisionCommand(ctx, prefix, "wsl.exe", "-d", name, "-u", "root", "--", "sh", "-lc", installPrerequisites); err != nil {
		return ProvisionResult{}, fmt.Errorf("install exact signed-snapshot WSL build prerequisites: %w", err)
	}
	archive := filepath.Join(prefix, "git-source.tar.xz")
	if err := downloadVerified(ctx, cell.Provisioner.SourceURL, cell.Provisioner.SHA256, archive); err != nil {
		return ProvisionResult{}, err
	}
	pathOutput := exec.CommandContext(ctx, "wsl.exe", "-d", name, "--", "wslpath", "-a", archive)
	raw, err := pathOutput.Output()
	if err != nil {
		return ProvisionResult{}, err
	}
	linuxArchive := strings.TrimSpace(string(raw))
	script := fmt.Sprintf("set -eu; rm -rf /tmp/git-source /opt/repokeeper-git; mkdir -p /tmp/git-source /opt/repokeeper-git; tar -xJf %q -C /tmp/git-source --strip-components=1; cd /tmp/git-source; make -j2 prefix=/opt/repokeeper-git NO_CURL=YesPlease NO_EXPAT=YesPlease NO_GETTEXT=YesPlease all; make prefix=/opt/repokeeper-git NO_CURL=YesPlease NO_EXPAT=YesPlease NO_GETTEXT=YesPlease install", linuxArchive)
	if err := runProvisionCommand(ctx, prefix, "wsl.exe", "-d", name, "--", "sh", "-lc", script); err != nil {
		return ProvisionResult{}, err
	}
	gitPath := "wsl://" + name + "/opt/repokeeper-git/bin/git"
	version, err := VerifyVersion(ctx, cell, gitPath)
	if err != nil {
		return ProvisionResult{}, err
	}
	failed = false
	return ProvisionResult{CellKey: CellKey(cell), Prefix: prefix, GitPath: gitPath, ExpectedVersion: version.ExpectedVersion, ActualVersion: version.ActualVersion, SourceURL: cell.Provisioner.SourceURL, SHA256: cell.Provisioner.SHA256, WSLDistribution: name}, nil
}

func extractZip(path, destination string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	if err := os.MkdirAll(destination, 0o750); err != nil {
		return err
	}
	for _, file := range reader.File {
		target := filepath.Join(destination, filepath.FromSlash(file.Name))
		relative, relErr := filepath.Rel(destination, target)
		if relErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("zip entry escapes destination: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, file.Mode())
		if err != nil {
			source.Close()
			return err
		}
		_, copyErr := io.Copy(output, source)
		closeErr := output.Close()
		sourceErr := source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if sourceErr != nil {
			return sourceErr
		}
	}
	return nil
}
