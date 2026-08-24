// SPDX-License-Identifier: MIT
//go:build integration

package compatibility

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
)

const MaxDocumentBytes = 1 << 20

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Declaration struct {
	SchemaVersion   int      `json:"schema_version"`
	SupportedMinors []string `json:"supported_minors"`
	Cells           []Cell   `json:"cells"`
}

type Cell struct {
	Environment string      `json:"environment"`
	RunnerLabel string      `json:"runner_label"`
	GitMinor    string      `json:"git_minor"`
	GitPatch    string      `json:"git_patch"`
	Provisioner Provisioner `json:"provisioner"`
	RootFS      *RootFS     `json:"rootfs,omitempty"`
	RoutineCI   bool        `json:"routine_ci"`
}

type Provisioner struct {
	Kind      string `json:"kind"`
	SourceURL string `json:"source_url"`
	SHA256    string `json:"sha256"`
}
type RootFS struct {
	Release            string        `json:"release"`
	ImageDate          string        `json:"image_date"`
	SourceURL          string        `json:"source_url"`
	SHA256             string        `json:"sha256"`
	WSLVersion         int           `json:"wsl_version"`
	Snapshot           string        `json:"snapshot"`
	BuildPrerequisites []PinnedInput `json:"build_prerequisites"`
}
type PinnedInput struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MatrixResult struct {
	Scope string       `json:"scope"`
	Cells []MatrixCell `json:"cells"`
}
type MatrixCell struct {
	Key string `json:"key"`
	Cell
}

func Load(path string) (Declaration, error) {
	file, err := os.Open(path)
	if err != nil {
		return Declaration{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, MaxDocumentBytes+1))
	decoder.DisallowUnknownFields()
	var declaration Declaration
	if err := decoder.Decode(&declaration); err != nil {
		return declaration, fmt.Errorf("decode compatibility declaration: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return declaration, fmt.Errorf("decode compatibility declaration trailing JSON: %w", err)
	}
	if err := Validate(declaration); err != nil {
		return declaration, err
	}
	return declaration, nil
}

func Validate(declaration Declaration) error {
	if declaration.SchemaVersion != 1 {
		return fmt.Errorf("schema_version: got %d, want 1", declaration.SchemaVersion)
	}
	wantedMinors := []string{"2.53", "2.54", "2.55"}
	if !equalStrings(declaration.SupportedMinors, wantedMinors) {
		return fmt.Errorf("supported_minors: got %v, want ordered closed window %v", declaration.SupportedMinors, wantedMinors)
	}
	wantedRunners := map[string]string{"linux": "ubuntu-24.04", "macos": "macos-15", "windows": "windows-2025", "wsl": "windows-2025"}
	wantedKinds := map[string]string{"linux": "source-build", "macos": "source-build", "windows": "mingit-archive", "wsl": "source-build"}
	seen, routine := map[string]bool{}, map[string]int{}
	for index, cell := range declaration.Cells {
		field := fmt.Sprintf("cells[%d]", index)
		runner, ok := wantedRunners[cell.Environment]
		if !ok {
			return fmt.Errorf("%s.environment: unsupported %q", field, cell.Environment)
		}
		if cell.RunnerLabel != runner || strings.Contains(cell.RunnerLabel, "latest") {
			return fmt.Errorf("%s.runner_label: got %q, want %q", field, cell.RunnerLabel, runner)
		}
		if cell.Provisioner.Kind != wantedKinds[cell.Environment] {
			return fmt.Errorf("%s.provisioner.kind: got %q", field, cell.Provisioner.Kind)
		}
		if !contains(declaration.SupportedMinors, cell.GitMinor) || !strings.HasPrefix(cell.GitPatch, cell.GitMinor+".") {
			return fmt.Errorf("%s.git_patch: %q does not match minor %q", field, cell.GitPatch, cell.GitMinor)
		}
		if err := validatePinnedInput(field+".provisioner", cell.Provisioner.SourceURL, cell.Provisioner.SHA256); err != nil {
			return err
		}
		key := CellKey(cell)
		if seen[key] {
			return fmt.Errorf("%s: duplicate cell %s", field, key)
		}
		seen[key] = true
		if cell.RoutineCI {
			routine[cell.Environment]++
		}
		if cell.Environment == "wsl" {
			if cell.RootFS == nil {
				return fmt.Errorf("%s.rootfs: required for WSL", field)
			}
			if cell.RootFS.Release != "ubuntu-24.04" || cell.RootFS.ImageDate != "20240423" || cell.RootFS.WSLVersion != 1 {
				return fmt.Errorf("%s.rootfs: requires Canonical Noble 20240423 WSL1", field)
			}
			if err := validatePinnedInput(field+".rootfs", cell.RootFS.SourceURL, cell.RootFS.SHA256); err != nil {
				return err
			}
			if cell.RootFS.Snapshot != "https://snapshot.ubuntu.com/ubuntu/20240423T000000Z/" {
				return fmt.Errorf("%s.rootfs.snapshot: must use the signed 20240423 snapshot", field)
			}
			if len(cell.RootFS.BuildPrerequisites) == 0 {
				return fmt.Errorf("%s.rootfs.build_prerequisites: exact signed-snapshot packages are required", field)
			}
			for inputIndex, input := range cell.RootFS.BuildPrerequisites {
				if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Version) == "" || strings.Contains(input.Version, "*") {
					return fmt.Errorf("%s.rootfs.build_prerequisites[%d]: exact package name and version are required", field, inputIndex)
				}
			}
		} else if cell.RootFS != nil {
			return fmt.Errorf("%s.rootfs: only WSL cells may declare rootfs", field)
		}
	}
	if len(seen) != 12 {
		return fmt.Errorf("cells: got %d unique cells, want complete 4x3 matrix", len(seen))
	}
	for environment := range wantedRunners {
		if routine[environment] != 1 {
			return fmt.Errorf("routine_ci: environment %s has %d representatives, want 1", environment, routine[environment])
		}
		for _, minor := range declaration.SupportedMinors {
			if !seen[environment+"-git-"+minor] {
				return fmt.Errorf("cells: missing %s-git-%s", environment, minor)
			}
		}
	}
	return nil
}

func Expand(declaration Declaration, scope string) (MatrixResult, error) {
	if err := Validate(declaration); err != nil {
		return MatrixResult{}, err
	}
	if scope != "routine" && scope != "release" {
		return MatrixResult{}, fmt.Errorf("scope: got %q, want routine or release", scope)
	}
	result := MatrixResult{Scope: scope}
	for _, cell := range declaration.Cells {
		if scope == "release" || cell.RoutineCI {
			result.Cells = append(result.Cells, MatrixCell{Key: CellKey(cell), Cell: cell})
		}
	}
	sort.Slice(result.Cells, func(i, j int) bool { return result.Cells[i].Key < result.Cells[j].Key })
	return result, nil
}

func FindCell(declaration Declaration, key string) (Cell, error) {
	for _, cell := range declaration.Cells {
		if CellKey(cell) == key {
			return cell, nil
		}
	}
	return Cell{}, fmt.Errorf("cell %q not found", key)
}
func CellKey(cell Cell) string { return cell.Environment + "-git-" + cell.GitMinor }

func VerifyDocs(declaration Declaration, design []byte) error {
	if err := Validate(declaration); err != nil {
		return err
	}
	for _, required := range []string{"2.53", "2.54", "2.55", "Linux", "macOS", "Windows", "WSL", "closed"} {
		if !bytes.Contains(design, []byte(required)) {
			return fmt.Errorf("DESIGN.md compatibility summary missing %q", required)
		}
	}
	return nil
}

func validatePinnedInput(field, rawURL, digest string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s.source_url: must be an absolute HTTPS URL", field)
	}
	// The declaration is committed, so userinfo in a pinned source URL would
	// publish whatever credential it carries.
	if parsed.User != nil {
		return fmt.Errorf("%s.source_url: must not embed credentials", field)
	}
	if !sha256Pattern.MatchString(digest) {
		return fmt.Errorf("%s.sha256: must be 64 lowercase hexadecimal characters", field)
	}
	return nil
}
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
