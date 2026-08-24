// SPDX-License-Identifier: MIT
//go:build integration && !windows

package compatibility

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func Provision(parent context.Context, cell Cell, prefix string) (ProvisionResult, error) {
	if cell.Provisioner.Kind != "source-build" {
		return ProvisionResult{}, fmt.Errorf("%s provisioner %q is not supported on %s", CellKey(cell), cell.Provisioner.Kind, runtime.GOOS)
	}
	wantedEnvironment := "linux"
	if runtime.GOOS == "darwin" {
		wantedEnvironment = "macos"
	}
	if cell.Environment != wantedEnvironment {
		return ProvisionResult{}, fmt.Errorf("cell %s requires %s, running on %s", CellKey(cell), cell.Environment, runtime.GOOS)
	}
	ctx, cancel := context.WithTimeout(parent, ProvisionTimeout)
	defer cancel()
	absPrefix, err := filepath.Abs(prefix)
	if err != nil {
		return ProvisionResult{}, err
	}
	if err := os.MkdirAll(absPrefix, 0o750); err != nil {
		return ProvisionResult{}, err
	}
	archive := filepath.Join(absPrefix, "git-source.tar.xz")
	if err := downloadVerified(ctx, cell.Provisioner.SourceURL, cell.Provisioner.SHA256, archive); err != nil {
		return ProvisionResult{}, err
	}
	sourceParent := filepath.Join(absPrefix, "source")
	if err := os.MkdirAll(sourceParent, 0o750); err != nil {
		return ProvisionResult{}, err
	}
	if err := runProvisionCommand(ctx, sourceParent, "tar", "-xJf", archive, "--strip-components=1"); err != nil {
		return ProvisionResult{}, err
	}
	install := filepath.Join(absPrefix, "install")
	makeArguments := []string{
		"prefix=" + install,
		"NO_CURL=YesPlease",
		"NO_EXPAT=YesPlease",
		"NO_GETTEXT=YesPlease",
		"NO_OPENSSL=YesPlease",
	}
	// Git's native Makefile accepts prefix directly. Avoid generating configure,
	// which would add an undeclared autoconf dependency to hosted runners.
	buildArguments := append([]string{"-j2"}, makeArguments...)
	buildArguments = append(buildArguments, "all")
	if err := runProvisionCommand(ctx, sourceParent, "make", buildArguments...); err != nil {
		return ProvisionResult{}, err
	}
	installArguments := append(makeArguments, "install")
	if err := runProvisionCommand(ctx, sourceParent, "make", installArguments...); err != nil {
		return ProvisionResult{}, err
	}
	gitPath := filepath.Join(install, "bin", "git")
	version, err := VerifyVersion(ctx, cell, gitPath)
	result := ProvisionResult{CellKey: CellKey(cell), Prefix: absPrefix, GitPath: gitPath, ExpectedVersion: version.ExpectedVersion, ActualVersion: version.ActualVersion, SourceURL: cell.Provisioner.SourceURL, SHA256: cell.Provisioner.SHA256}
	return result, err
}
