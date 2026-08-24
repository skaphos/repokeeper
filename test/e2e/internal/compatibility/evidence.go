// SPDX-License-Identifier: MIT
//go:build integration

package compatibility

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type CompatibilityResult struct {
	SchemaVersion  int               `json:"schema_version"`
	CellKey        string            `json:"cell_key"`
	CandidateTag   string            `json:"candidate_tag"`
	Commit         string            `json:"commit"`
	Environment    string            `json:"environment"`
	RunnerLabel    string            `json:"runner_label"`
	GitMinor       string            `json:"git_minor"`
	ExpectedGit    string            `json:"expected_git"`
	ActualGit      string            `json:"actual_git"`
	InputDigests   map[string]string `json:"input_digests"`
	Status         string            `json:"status"`
	ArtifactDigest string            `json:"artifact_digest"`
}

type EvidenceSummary struct {
	Complete   bool     `json:"complete"`
	Missing    []string `json:"missing"`
	Duplicate  []string `json:"duplicate"`
	Unexpected []string `json:"unexpected"`
	Failed     []string `json:"failed"`
	Skipped    []string `json:"skipped"`
	Mismatched []string `json:"mismatched"`
}

func WriteEvidence(path string, result CompatibilityResult) (CompatibilityResult, error) {
	result.SchemaVersion = 1
	result.ArtifactDigest = ""
	payload, err := json.Marshal(result)
	if err != nil {
		return result, err
	}
	result.ArtifactDigest = digestBytes(payload)
	payload, err = json.MarshalIndent(result, "", "  ")
	if err != nil {
		return result, err
	}
	payload = append(payload, '\n')
	if len(payload) > MaxDocumentBytes {
		return result, fmt.Errorf("evidence document exceeds %d bytes", MaxDocumentBytes)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return result, err
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return result, err
	}
	return result, nil
}

func VerifyEvidence(declaration Declaration, directory, candidateTag, commit string) (EvidenceSummary, error) {
	if err := Validate(declaration); err != nil {
		return EvidenceSummary{}, err
	}
	expected := map[string]Cell{}
	for _, cell := range declaration.Cells {
		expected[CellKey(cell)] = cell
	}
	seen := map[string]int{}
	summary := EvidenceSummary{Missing: []string{}, Duplicate: []string{}, Unexpected: []string{}, Failed: []string{}, Skipped: []string{}, Mismatched: []string{}}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return summary, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(directory, entry.Name()))
		if readErr != nil {
			return summary, readErr
		}
		if len(data) > MaxDocumentBytes {
			return summary, fmt.Errorf("evidence %s exceeds size limit", entry.Name())
		}
		var result CompatibilityResult
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if decodeErr := decoder.Decode(&result); decodeErr != nil {
			return summary, fmt.Errorf("decode evidence %s: %w", entry.Name(), decodeErr)
		}
		if decodeErr := decoder.Decode(&struct{}{}); decodeErr != io.EOF {
			return summary, fmt.Errorf("decode evidence %s trailing JSON: %w", entry.Name(), decodeErr)
		}
		seen[result.CellKey]++
		cell, exists := expected[result.CellKey]
		if !exists {
			summary.Unexpected = append(summary.Unexpected, result.CellKey)
			continue
		}
		if result.Status == "skipped" {
			summary.Skipped = append(summary.Skipped, result.CellKey)
		} else if result.Status != "passed" {
			summary.Failed = append(summary.Failed, result.CellKey)
		}
		copyForDigest := result
		copyForDigest.ArtifactDigest = ""
		digestPayload, _ := json.Marshal(copyForDigest)
		mismatch := result.SchemaVersion != 1 || result.CandidateTag != candidateTag || result.Commit != commit || result.Environment != cell.Environment || result.RunnerLabel != cell.RunnerLabel || result.GitMinor != cell.GitMinor || result.ExpectedGit != cell.GitPatch || result.ActualGit != "git version "+cell.GitPatch || result.InputDigests["provisioner"] != cell.Provisioner.SHA256 || result.ArtifactDigest != digestBytes(digestPayload)
		if cell.RootFS != nil && result.InputDigests["rootfs"] != cell.RootFS.SHA256 {
			mismatch = true
		}
		if cell.RootFS != nil {
			for _, prerequisite := range cell.RootFS.BuildPrerequisites {
				if result.InputDigests["package:"+prerequisite.Name] != prerequisite.Version {
					mismatch = true
				}
			}
		}
		if mismatch {
			summary.Mismatched = append(summary.Mismatched, result.CellKey)
		}
	}
	for key := range expected {
		if seen[key] == 0 {
			summary.Missing = append(summary.Missing, key)
		}
		if seen[key] > 1 {
			summary.Duplicate = append(summary.Duplicate, key)
		}
	}
	for _, values := range [][]string{summary.Missing, summary.Duplicate, summary.Unexpected, summary.Failed, summary.Skipped, summary.Mismatched} {
		sort.Strings(values)
	}
	summary.Complete = len(summary.Missing)+len(summary.Duplicate)+len(summary.Unexpected)+len(summary.Failed)+len(summary.Skipped)+len(summary.Mismatched) == 0
	if !summary.Complete {
		return summary, fmt.Errorf("compatibility evidence incomplete: missing=%v duplicate=%v unexpected=%v failed=%v skipped=%v mismatched=%v", summary.Missing, summary.Duplicate, summary.Unexpected, summary.Failed, summary.Skipped, summary.Mismatched)
	}
	return summary, nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
