// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/skaphos/repokeeper/internal/model"
)

type scanJSONResponse []model.RepoStatus

type statusJSONResponse struct {
	APIVersion  string                 `json:"apiVersion"`
	GeneratedAt time.Time              `json:"generated_at"`
	Repos       []statusJSONRepository `json:"repos"`
}

type statusJSONRepository struct {
	model.RepoStatus
	LocalLabels map[string]string `json:"local_labels,omitempty"`
}

func runRepoKeeper(ctx context.Context, workspace *MaterializedWorkspace, operation string, arguments ...string) ExecutionResult {
	args := append([]string{"--config", workspace.ConfigPath}, arguments...)
	return runCommand(ctx, maximumScenarioTimeout, operation, repokeeperPath, args, workspace.WorkspaceRoot, workspace.Env, workspace.Root, workspace.ConfigPath)
}

func decodeScanJSON(result ExecutionResult) (scanJSONResponse, error) {
	var response scanJSONResponse
	if err := json.Unmarshal(result.Stdout, &response); err != nil {
		return nil, fmt.Errorf("decode scan JSON: %w\n%s", err, result.Diagnostics())
	}
	for index, repository := range response {
		if repository.RepoID == "" || repository.Path == "" || repository.PrimaryRemote == "" {
			return nil, fmt.Errorf("scan repos[%d] is missing required fields", index)
		}
	}
	return response, nil
}

func decodeStatusJSON(result ExecutionResult) (statusJSONResponse, error) {
	var response statusJSONResponse
	if err := json.Unmarshal(result.Stdout, &response); err != nil {
		return response, fmt.Errorf("decode status JSON: %w\n%s", err, result.Diagnostics())
	}
	if response.APIVersion == "" || response.GeneratedAt.IsZero() {
		return response, fmt.Errorf("status JSON is missing apiVersion or generated_at")
	}
	for index, repository := range response.Repos {
		if repository.RepoID == "" || repository.Path == "" {
			return response, fmt.Errorf("status repos[%d] is missing repo_id or path", index)
		}
	}
	return response, nil
}

func requireDomainExit(result ExecutionResult, allowed ...int) error {
	return expectExit(result, allowed...)
}
