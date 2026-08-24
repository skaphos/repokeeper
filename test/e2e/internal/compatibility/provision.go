// SPDX-License-Identifier: MIT
//go:build integration

package compatibility

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const ProvisionTimeout = 10 * time.Minute

type ProvisionResult struct {
	CellKey         string `json:"cell_key"`
	Prefix          string `json:"prefix"`
	GitPath         string `json:"git_path"`
	ExpectedVersion string `json:"expected_version"`
	ActualVersion   string `json:"actual_version"`
	SourceURL       string `json:"source_url"`
	SHA256          string `json:"sha256"`
	WSLDistribution string `json:"wsl_distribution,omitempty"`
}
type VersionResult struct {
	CellKey         string `json:"cell_key"`
	ExpectedVersion string `json:"expected_version"`
	ActualVersion   string `json:"actual_version"`
	Match           bool   `json:"match"`
}

func VerifyVersion(ctx context.Context, cell Cell, gitPath string) (VersionResult, error) {
	var command *exec.Cmd
	if strings.HasPrefix(gitPath, "wsl://") {
		parts := strings.SplitN(strings.TrimPrefix(gitPath, "wsl://"), "/", 2)
		if len(parts) != 2 {
			return VersionResult{}, fmt.Errorf("invalid WSL git path %q", gitPath)
		}
		command = exec.CommandContext(ctx, "wsl.exe", "-d", parts[0], "--", "/"+parts[1], "--version")
	} else {
		command = exec.CommandContext(ctx, gitPath, "--version")
	}
	output, err := command.CombinedOutput()
	actual := strings.TrimSpace(string(output))
	expected := "git version " + cell.GitPatch
	result := VersionResult{CellKey: CellKey(cell), ExpectedVersion: expected, ActualVersion: actual, Match: actual == expected}
	if err != nil {
		return result, fmt.Errorf("run %s --version: %w: %s", gitPath, err, output)
	}
	if !result.Match {
		return result, fmt.Errorf("cell %s Git version mismatch: got %q, want %q", result.CellKey, actual, expected)
	}
	return result, nil
}

func downloadVerified(ctx context.Context, sourceURL, expected, destination string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	response, err := (&http.Client{Timeout: ProvisionTimeout}).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %s", sourceURL, response.Status)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, 2<<30))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", sourceURL, actual, expected)
	}
	return nil
}

func runProvisionCommand(ctx context.Context, dir, executable string, arguments ...string) error {
	command := exec.CommandContext(ctx, executable, arguments...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %q: %w\n%s", executable, arguments, err, output)
	}
	return nil
}
