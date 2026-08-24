// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"path/filepath"
)

func extensionRecipe() (WorkspaceRecipe, error) {
	return loadRecipe(filepath.Join(moduleRoot, "test", "e2e", "testdata", "recipes", "extension.json"))
}

func materializeExtension(ctx context.Context, root string) (*MaterializedWorkspace, WorkspaceRecipe, error) {
	recipe, err := extensionRecipe()
	if err != nil {
		return nil, WorkspaceRecipe{}, err
	}
	workspace, err := materializeRecipe(ctx, recipe, root, SeedAllEntries)
	return workspace, recipe, err
}
