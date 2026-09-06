// SPDX-License-Identifier: MIT
package mcpserver

import (
	"time"

	"github.com/skaphos/repokeeper/v2/internal/config"
	"github.com/skaphos/repokeeper/v2/internal/model"
	"github.com/skaphos/repokeeper/v2/internal/registry"
)

// Resource DTOs keep the JSON contract independent of YAML persistence structs.
// In particular, serving Config directly would expose an unredacted registry
// and make an internal Go field rename a breaking adapter change.
type registryResourceResponse struct {
	UpdatedAt string                  `json:"updated_at,omitempty"`
	Repos     []registryEntryResponse `json:"repos"`
}

type registryEntryResponse struct {
	RepoID                  string               `json:"repo_id"`
	CheckoutID              string               `json:"checkout_id,omitempty"`
	Path                    string               `json:"path"`
	RemoteURL               string               `json:"remote_url"`
	Type                    string               `json:"type,omitempty"`
	Branch                  string               `json:"branch,omitempty"`
	Labels                  map[string]string    `json:"labels,omitempty"`
	Annotations             map[string]string    `json:"annotations,omitempty"`
	RepoMetadataFile        string               `json:"repo_metadata_file,omitempty"`
	RepoMetadataError       string               `json:"repo_metadata_error,omitempty"`
	RepoMetadataFingerprint string               `json:"repo_metadata_fingerprint,omitempty"`
	RepoMetadata            *model.RepoMetadata  `json:"repo_metadata,omitempty"`
	LastSeen                string               `json:"last_seen,omitempty"`
	Status                  registry.EntryStatus `json:"status"`
}

type configResourceResponse struct {
	APIVersion        string                    `json:"apiVersion"`
	Kind              string                    `json:"kind"`
	Exclude           []string                  `json:"exclude"`
	IgnoredPaths      []string                  `json:"ignored_paths,omitempty"`
	RegistryPath      string                    `json:"registry_path,omitempty"`
	Registry          *registryResourceResponse `json:"registry,omitempty"`
	RegistryStaleDays int                       `json:"registry_stale_days"`
	Defaults          configDefaults            `json:"defaults"`
	BranchPolicy      branchPolicyResponse      `json:"branch_policy"`
}

type branchPolicyResponse struct {
	ProtectedPatterns []string `json:"protected_patterns"`
	BaseBranch        string   `json:"base_branch,omitempty"`
	StaleDays         int      `json:"stale_days"`
	RequireMerged     bool     `json:"require_merged"`
}

func newRegistryEntryResponse(entry registry.Entry) registryEntryResponse {
	entry = entry.Redacted()
	return registryEntryResponse{
		RepoID: entry.RepoID, CheckoutID: entry.CheckoutID, Path: entry.Path,
		RemoteURL: entry.RemoteURL, Type: entry.Type, Branch: entry.Branch,
		Labels: entry.Labels, Annotations: entry.Annotations,
		RepoMetadataFile: entry.RepoMetadataFile, RepoMetadataError: entry.RepoMetadataError,
		RepoMetadataFingerprint: entry.RepoMetadataFingerprint, RepoMetadata: entry.RepoMetadata,
		LastSeen: observedTime(entry.LastSeen), Status: entry.Status,
	}
}

func newRegistryResourceResponse(reg *registry.Registry) *registryResourceResponse {
	if reg == nil {
		return nil
	}
	response := &registryResourceResponse{
		UpdatedAt: observedTime(reg.UpdatedAt), Repos: make([]registryEntryResponse, 0, len(reg.Entries)),
	}
	for _, entry := range reg.Entries {
		response.Repos = append(response.Repos, newRegistryEntryResponse(entry))
	}
	return response
}

func newConfigResourceResponse(cfg *config.Config) configResourceResponse {
	return configResourceResponse{
		APIVersion: cfg.APIVersion, Kind: cfg.Kind, Exclude: nonNilStrings(cfg.Exclude),
		IgnoredPaths: cfg.IgnoredPaths, RegistryPath: cfg.RegistryPath,
		Registry: newRegistryResourceResponse(cfg.Registry), RegistryStaleDays: cfg.RegistryStaleDays,
		Defaults: configDefaults{
			RemoteName: cfg.Defaults.RemoteName, MainBranch: cfg.Defaults.MainBranch,
			Concurrency: cfg.Defaults.Concurrency, TimeoutSeconds: cfg.Defaults.TimeoutSeconds,
		},
		BranchPolicy: newBranchPolicyResponse(cfg.BranchPolicy),
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// Unknown observation times are absent, never a fabricated year-one date.
func observedTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

// Both config surfaces report the same policy, including explicit false and zero values.
func newBranchPolicyResponse(policy config.BranchPolicy) branchPolicyResponse {
	return branchPolicyResponse{
		ProtectedPatterns: nonNilStrings(policy.ProtectedPatterns),
		BaseBranch:        policy.BaseBranch, StaleDays: policy.StaleDays,
		RequireMerged: policy.RequireMerged,
	}
}
