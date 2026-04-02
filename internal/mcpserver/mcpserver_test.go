// SPDX-License-Identifier: MIT
package mcpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/skaphos/repokeeper/internal/config"
	"github.com/skaphos/repokeeper/internal/engine"
	"github.com/skaphos/repokeeper/internal/mcpserver"
	"github.com/skaphos/repokeeper/internal/model"
	"github.com/skaphos/repokeeper/internal/registry"
	"github.com/skaphos/repokeeper/internal/vcs"
)

// --- mock engine ---

type mockEngine struct {
	cfg           *config.Config
	reg           *registry.Registry
	inspectResult *model.RepoStatus
	inspectErr    error
	statusResult  *model.StatusReport
	statusErr     error
}

func (e *mockEngine) Status(_ context.Context, opts engine.StatusOptions) (*model.StatusReport, error) {
	if e.statusErr != nil {
		return nil, e.statusErr
	}
	if e.statusResult != nil {
		// Apply filter if FilterMissing requested — simulates engine behavior.
		if opts.Filter == engine.FilterDirty {
			var filtered []model.RepoStatus
			for _, r := range e.statusResult.Repos {
				if r.Worktree != nil && r.Worktree.Dirty {
					filtered = append(filtered, r)
				}
			}
			return &model.StatusReport{GeneratedAt: e.statusResult.GeneratedAt, Repos: filtered}, nil
		}
		return e.statusResult, nil
	}
	return &model.StatusReport{GeneratedAt: time.Now()}, nil
}
func (e *mockEngine) Sync(_ context.Context, _ engine.SyncOptions) ([]engine.SyncResult, error) {
	return nil, nil
}
func (e *mockEngine) ExecuteSyncPlanWithCallbacks(_ context.Context, _ []engine.SyncResult, _ engine.SyncOptions, _ engine.SyncStartCallback, _ engine.SyncResultCallback) ([]engine.SyncResult, error) {
	return nil, nil
}
func (e *mockEngine) InspectRepo(_ context.Context, path string) (*model.RepoStatus, error) {
	if e.inspectErr != nil {
		return nil, e.inspectErr
	}
	if e.inspectResult != nil {
		return e.inspectResult, nil
	}
	return &model.RepoStatus{
		RepoID: "github.com/example/alpha",
		Path:   path,
		Head:   model.Head{Branch: "main"},
		Tracking: model.Tracking{
			Upstream: "origin/main",
			Status:   model.TrackingEqual,
		},
	}, nil
}
func (e *mockEngine) RepairUpstream(_ context.Context, _, _ string) (engine.RepairUpstreamResult, error) {
	return engine.RepairUpstreamResult{}, nil
}
func (e *mockEngine) ResetRepo(_ context.Context, _, _ string) error                   { return nil }
func (e *mockEngine) DeleteRepo(_ context.Context, _, _ string, _ bool) error          { return nil }
func (e *mockEngine) CloneAndRegister(_ context.Context, _, _, _ string, _ bool) error { return nil }
func (e *mockEngine) Scan(_ context.Context, _ engine.ScanOptions) ([]model.RepoStatus, error) {
	return nil, nil
}
func (e *mockEngine) Registry() *registry.Registry { return e.reg }
func (e *mockEngine) Config() *config.Config       { return e.cfg }
func (e *mockEngine) Adapter() vcs.Adapter         { return nil }

var _ mcpserver.EngineAPI = (*mockEngine)(nil)

// --- helpers ---

func callTool(srv *mcpserver.MCPServer, name string, args map[string]any) (*mcp.CallToolResult, error) {
	st := srv.Inner().GetTool(name)
	Expect(st).NotTo(BeNil(), "tool %q not registered", name)

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	return st.Handler(context.Background(), req)
}

func resultJSON(result *mcp.CallToolResult) []byte {
	Expect(result).NotTo(BeNil())
	Expect(result.Content).NotTo(BeEmpty())
	tc, ok := result.Content[0].(mcp.TextContent)
	Expect(ok).To(BeTrue(), "expected TextContent, got %T", result.Content[0])
	return []byte(tc.Text)
}

func intPtr(v int) *int { return &v }

// --- test data ---

func newTestRegistry() *registry.Registry {
	return &registry.Registry{
		UpdatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
		Entries: []registry.Entry{
			{
				RepoID:      "github.com/example/alpha",
				Path:        "/home/user/repos/alpha",
				RemoteURL:   "git@github.com:example/alpha.git",
				Type:        "checkout",
				Labels:      map[string]string{"team": "platform", "env": "prod"},
				Annotations: map[string]string{"note": "primary service"},
				Status:      registry.StatusPresent,
				LastSeen:    time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
			},
			{
				RepoID:    "github.com/example/beta",
				Path:      "/home/user/repos/beta",
				RemoteURL: "git@github.com:example/beta.git",
				Type:      "checkout",
				Labels:    map[string]string{"team": "data"},
				Status:    registry.StatusPresent,
				LastSeen:  time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
			},
			{
				RepoID:   "github.com/example/gamma",
				Path:     "/home/user/repos/gamma",
				Type:     "mirror",
				Status:   registry.StatusMissing,
				LastSeen: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
			},
		},
	}
}

func newTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Exclude = []string{"**/node_modules/**"}
	return &cfg
}

func newTestStatusReport() *model.StatusReport {
	return &model.StatusReport{
		GeneratedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
		Repos: []model.RepoStatus{
			{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Type:   "checkout",
				Labels: map[string]string{"runtime": "go"},
				Head:   model.Head{Branch: "main"},
				Worktree: &model.Worktree{
					Dirty:     true,
					Untracked: 1,
				},
				Tracking: model.Tracking{
					Upstream: "origin/main",
					Status:   model.TrackingAhead,
					Ahead:    intPtr(2),
				},
				RepoMetadata: &model.RepoMetadata{
					Name:        "Alpha Service",
					Labels:      map[string]string{"runtime": "go"},
					Entrypoints: map[string]string{"main": "cmd/alpha/main.go", "config": "config/"},
					Paths: model.RepoMetadataPaths{
						Authoritative: []string{"cmd/", "internal/", "pkg/"},
						LowValue:      []string{"vendor/", "testdata/"},
					},
					RelatedRepos: []model.RepoMetadataRelatedRepo{
						{RepoID: "github.com/example/beta", Relationship: "dependency"},
						{RepoID: "github.com/example/unknown", Relationship: "upstream"},
					},
				},
			},
			{
				RepoID: "github.com/example/beta",
				Path:   "/home/user/repos/beta",
				Type:   "checkout",
				Head:   model.Head{Branch: "develop"},
				Worktree: &model.Worktree{
					Dirty: false,
				},
				Tracking: model.Tracking{
					Upstream: "origin/develop",
					Status:   model.TrackingEqual,
				},
			},
		},
	}
}

// --- specs ---

var _ = Describe("MCPServer", func() {
	var (
		eng *mockEngine
		srv *mcpserver.MCPServer
	)

	BeforeEach(func() {
		eng = &mockEngine{
			cfg: newTestConfig(),
			reg: newTestRegistry(),
		}
		srv = mcpserver.New(eng, "/home/user/.repokeeper.yaml", "0.1.0-test", nil)
	})

	It("creates a server with all registered tools", func() {
		Expect(srv.Inner()).NotTo(BeNil())
		tools := srv.Inner().ListTools()
		// Phase 1 tools
		Expect(tools).To(HaveKey("list_repositories"))
		Expect(tools).To(HaveKey("get_repository_context"))
		Expect(tools).To(HaveKey("get_workspace_config"))
		// Phase 2 tools
		Expect(tools).To(HaveKey("build_workspace_inventory"))
		Expect(tools).To(HaveKey("select_repositories"))
		Expect(tools).To(HaveKey("get_repo_metadata"))
		Expect(tools).To(HaveKey("get_authoritative_paths"))
		Expect(tools).To(HaveKey("get_related_repositories"))
	})

	// --- Phase 1 tools ---

	Describe("list_repositories", func() {
		It("returns all repos when no filters given", func() {
			result, err := callTool(srv, "list_repositories", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(HaveLen(3))
		})

		It("filters by status", func() {
			result, err := callTool(srv, "list_repositories", map[string]any{
				"status": "missing",
			})
			Expect(err).NotTo(HaveOccurred())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(HaveLen(1))
			Expect(repos[0]["repo_id"]).To(Equal("github.com/example/gamma"))
		})

		It("filters by label selector", func() {
			result, err := callTool(srv, "list_repositories", map[string]any{
				"label_selector": "team=platform",
			})
			Expect(err).NotTo(HaveOccurred())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(HaveLen(1))
			Expect(repos[0]["repo_id"]).To(Equal("github.com/example/alpha"))
		})

		It("returns error for invalid label selector", func() {
			result, err := callTool(srv, "list_repositories", map[string]any{
				"label_selector": "bad key=1",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error when registry is nil", func() {
			eng.reg = nil
			result, err := callTool(srv, "list_repositories", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	Describe("get_repository_context", func() {
		It("returns repo context for valid repo_id", func() {
			result, err := callTool(srv, "get_repository_context", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			Expect(resp["repo_id"]).To(Equal("github.com/example/alpha"))
			Expect(resp["path"]).To(Equal("/home/user/repos/alpha"))
		})

		It("returns error when repo parameter is missing", func() {
			result, err := callTool(srv, "get_repository_context", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error for unknown repo", func() {
			result, err := callTool(srv, "get_repository_context", map[string]any{
				"repo": "nonexistent/repo",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("merges registry labels with status labels", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Labels: map[string]string{"runtime": "go"},
				Head:   model.Head{Branch: "main"},
			}

			result, err := callTool(srv, "get_repository_context", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			labels := resp["labels"].(map[string]any)
			// Registry labels (team, env) + status label (runtime)
			Expect(labels).To(HaveKey("team"))
			Expect(labels).To(HaveKey("env"))
			Expect(labels).To(HaveKey("runtime"))
		})
	})

	Describe("get_workspace_config", func() {
		It("returns config with defaults", func() {
			result, err := callTool(srv, "get_workspace_config", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			Expect(resp["config_path"]).To(Equal("/home/user/.repokeeper.yaml"))
			Expect(resp["repo_count"]).To(BeNumerically("==", 3))

			defaults := resp["defaults"].(map[string]any)
			Expect(defaults["remote_name"]).To(Equal("origin"))
			Expect(defaults["main_branch"]).To(Equal("main"))
		})

		It("returns error when config is nil", func() {
			eng.cfg = nil
			result, err := callTool(srv, "get_workspace_config", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	// --- Phase 2 tools ---

	Describe("build_workspace_inventory", func() {
		BeforeEach(func() {
			eng.statusResult = newTestStatusReport()
		})

		It("returns all repos when no filter given", func() {
			result, err := callTool(srv, "build_workspace_inventory", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			Expect(resp["generated_at"]).NotTo(BeEmpty())
			repos := resp["repos"].([]any)
			Expect(repos).To(HaveLen(2))
		})

		It("filters by health filter", func() {
			result, err := callTool(srv, "build_workspace_inventory", map[string]any{
				"filter": "dirty",
			})
			Expect(err).NotTo(HaveOccurred())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			repos := resp["repos"].([]any)
			Expect(repos).To(HaveLen(1))
			first := repos[0].(map[string]any)
			Expect(first["repo_id"]).To(Equal("github.com/example/alpha"))
		})

		It("filters by label selector post-status", func() {
			result, err := callTool(srv, "build_workspace_inventory", map[string]any{
				"label_selector": "team=platform",
			})
			Expect(err).NotTo(HaveOccurred())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			// alpha has team=platform in registry, enriched onto status labels
			repos := resp["repos"].([]any)
			Expect(repos).To(HaveLen(1))
			first := repos[0].(map[string]any)
			Expect(first["repo_id"]).To(Equal("github.com/example/alpha"))
		})

		It("enriches with registry annotations", func() {
			result, err := callTool(srv, "build_workspace_inventory", map[string]any{
				"label_selector": "team=platform",
			})
			Expect(err).NotTo(HaveOccurred())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			repos := resp["repos"].([]any)
			first := repos[0].(map[string]any)
			annotations := first["annotations"].(map[string]any)
			Expect(annotations["note"]).To(Equal("primary service"))
		})

		It("returns error when engine status fails", func() {
			eng.statusErr = fmt.Errorf("engine failure")
			result, err := callTool(srv, "build_workspace_inventory", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error for invalid label selector", func() {
			result, err := callTool(srv, "build_workspace_inventory", map[string]any{
				"label_selector": "bad key=1",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	Describe("select_repositories", func() {
		BeforeEach(func() {
			eng.statusResult = newTestStatusReport()
		})

		It("returns all repos when no selectors given", func() {
			result, err := callTool(srv, "select_repositories", nil)
			Expect(err).NotTo(HaveOccurred())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(HaveLen(2))
			Expect(repos[0]["match_reason"]).To(Equal("all"))
		})

		It("filters by label selector", func() {
			result, err := callTool(srv, "select_repositories", map[string]any{
				"label_selector": "team=data",
			})
			Expect(err).NotTo(HaveOccurred())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(HaveLen(1))
			Expect(repos[0]["repo_id"]).To(Equal("github.com/example/beta"))
		})

		It("filters by name match", func() {
			result, err := callTool(srv, "select_repositories", map[string]any{
				"name_match": "alpha",
			})
			Expect(err).NotTo(HaveOccurred())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(HaveLen(1))
			Expect(repos[0]["repo_id"]).To(Equal("github.com/example/alpha"))
			Expect(repos[0]["match_reason"]).To(ContainSubstring("name:alpha"))
		})

		It("combines label and name filters", func() {
			result, err := callTool(srv, "select_repositories", map[string]any{
				"label_selector": "team=platform",
				"name_match":     "alpha",
			})
			Expect(err).NotTo(HaveOccurred())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(HaveLen(1))
			Expect(repos[0]["match_reason"]).To(ContainSubstring("label:"))
			Expect(repos[0]["match_reason"]).To(ContainSubstring("name:"))
		})

		It("returns error for invalid field selector", func() {
			result, err := callTool(srv, "select_repositories", map[string]any{
				"field_selector": "invalid.field=x",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error when engine status fails", func() {
			eng.statusErr = fmt.Errorf("engine failure")
			result, err := callTool(srv, "select_repositories", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	Describe("get_repo_metadata", func() {
		It("returns metadata when present", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Head:   model.Head{Branch: "main"},
				RepoMetadata: &model.RepoMetadata{
					Name:   "Alpha Service",
					Labels: map[string]string{"runtime": "go"},
				},
			}

			result, err := callTool(srv, "get_repo_metadata", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			Expect(resp["name"]).To(Equal("Alpha Service"))
		})

		It("returns null when no metadata exists", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Head:   model.Head{Branch: "main"},
			}

			result, err := callTool(srv, "get_repo_metadata", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			text := string(resultJSON(result))
			Expect(text).To(Equal("null"))
		})

		It("returns error when repo parameter is missing", func() {
			result, err := callTool(srv, "get_repo_metadata", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error for unknown repo", func() {
			result, err := callTool(srv, "get_repo_metadata", map[string]any{
				"repo": "nonexistent/repo",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	Describe("get_authoritative_paths", func() {
		It("returns paths and entrypoints from metadata", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Head:   model.Head{Branch: "main"},
				RepoMetadata: &model.RepoMetadata{
					Entrypoints: map[string]string{"main": "cmd/alpha/main.go"},
					Paths: model.RepoMetadataPaths{
						Authoritative: []string{"cmd/", "internal/"},
						LowValue:      []string{"vendor/"},
					},
				},
			}

			result, err := callTool(srv, "get_authoritative_paths", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			auth := resp["authoritative"].([]any)
			Expect(auth).To(ContainElement("cmd/"))
			Expect(auth).To(ContainElement("internal/"))

			low := resp["low_value"].([]any)
			Expect(low).To(ContainElement("vendor/"))

			entry := resp["entrypoints"].(map[string]any)
			Expect(entry["main"]).To(Equal("cmd/alpha/main.go"))
		})

		It("returns error when no metadata exists", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Head:   model.Head{Branch: "main"},
			}

			result, err := callTool(srv, "get_authoritative_paths", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error when repo parameter is missing", func() {
			result, err := callTool(srv, "get_authoritative_paths", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	Describe("get_related_repositories", func() {
		It("returns related repos with registry cross-reference", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Head:   model.Head{Branch: "main"},
				RepoMetadata: &model.RepoMetadata{
					RelatedRepos: []model.RepoMetadataRelatedRepo{
						{RepoID: "github.com/example/beta", Relationship: "dependency"},
						{RepoID: "github.com/example/unknown", Relationship: "upstream"},
					},
				},
			}

			result, err := callTool(srv, "get_related_repositories", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(HaveLen(2))

			// beta is in the registry — should have path and status
			Expect(repos[0]["repo_id"]).To(Equal("github.com/example/beta"))
			Expect(repos[0]["relationship"]).To(Equal("dependency"))
			Expect(repos[0]["path"]).To(Equal("/home/user/repos/beta"))
			Expect(repos[0]["status"]).To(Equal("present"))

			// unknown is not in the registry — path and status should be empty
			Expect(repos[1]["repo_id"]).To(Equal("github.com/example/unknown"))
			Expect(repos[1]["relationship"]).To(Equal("upstream"))
			Expect(repos[1]).NotTo(HaveKey("path"))
		})

		It("returns empty array when no metadata", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Head:   model.Head{Branch: "main"},
			}

			result, err := callTool(srv, "get_related_repositories", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(BeEmpty())
		})

		It("returns empty array when metadata has no related repos", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID:       "github.com/example/alpha",
				Path:         "/home/user/repos/alpha",
				Head:         model.Head{Branch: "main"},
				RepoMetadata: &model.RepoMetadata{Name: "Alpha"},
			}

			result, err := callTool(srv, "get_related_repositories", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(BeEmpty())
		})

		It("returns error when repo parameter is missing", func() {
			result, err := callTool(srv, "get_related_repositories", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	// --- MCP Resources ---

	Describe("resources", func() {
		It("serves config resource", func() {
			response := srv.Inner().HandleMessage(
				context.Background(),
				resourceReadMessage("repokeeper://config"),
			)

			text := expectResourceSuccess(response)
			var cfg map[string]any
			Expect(json.Unmarshal([]byte(text), &cfg)).To(Succeed())
			Expect(cfg).To(HaveKey("Exclude"))
		})

		It("serves registry resource", func() {
			response := srv.Inner().HandleMessage(
				context.Background(),
				resourceReadMessage("repokeeper://registry"),
			)

			text := expectResourceSuccess(response)
			var reg map[string]any
			Expect(json.Unmarshal([]byte(text), &reg)).To(Succeed())
			Expect(reg).To(HaveKey("Entries"))
		})

		It("serves repo entry resource by repo_id", func() {
			response := srv.Inner().HandleMessage(
				context.Background(),
				resourceReadMessage("repokeeper://repo/github.com/example/alpha"),
			)

			text := expectResourceSuccess(response)
			var entry map[string]any
			Expect(json.Unmarshal([]byte(text), &entry)).To(Succeed())
			Expect(entry["RepoID"]).To(Equal("github.com/example/alpha"))
			Expect(entry["Path"]).To(Equal("/home/user/repos/alpha"))
		})

		It("returns error for unknown repo resource", func() {
			response := srv.Inner().HandleMessage(
				context.Background(),
				resourceReadMessage("repokeeper://repo/nonexistent/repo"),
			)
			expectResourceError(response)
		})

		It("serves repo metadata resource", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Head:   model.Head{Branch: "main"},
				RepoMetadata: &model.RepoMetadata{
					Name: "Alpha Service",
				},
			}

			response := srv.Inner().HandleMessage(
				context.Background(),
				resourceReadMessage("repokeeper://repo/github.com/example/alpha/metadata"),
			)

			text := expectResourceSuccess(response)
			var meta map[string]any
			Expect(json.Unmarshal([]byte(text), &meta)).To(Succeed())
			Expect(meta["name"]).To(Equal("Alpha Service"))
		})

		It("returns error for repo metadata when no metadata exists", func() {
			eng.inspectResult = &model.RepoStatus{
				RepoID: "github.com/example/alpha",
				Path:   "/home/user/repos/alpha",
				Head:   model.Head{Branch: "main"},
			}

			response := srv.Inner().HandleMessage(
				context.Background(),
				resourceReadMessage("repokeeper://repo/github.com/example/alpha/metadata"),
			)
			expectResourceError(response)
		})

		It("returns error when config is nil for config resource", func() {
			eng.cfg = nil
			response := srv.Inner().HandleMessage(
				context.Background(),
				resourceReadMessage("repokeeper://config"),
			)
			expectResourceError(response)
		})

		It("returns error when registry is nil for registry resource", func() {
			eng.reg = nil
			response := srv.Inner().HandleMessage(
				context.Background(),
				resourceReadMessage("repokeeper://registry"),
			)
			expectResourceError(response)
		})
	})
})

var _ = Describe("resolveRepo", Ordered, func() {
	// resolveRepo is tested indirectly through get_repository_context above.
	// These additional specs cover edge cases.

	var (
		eng *mockEngine
		srv *mcpserver.MCPServer
	)

	BeforeAll(func() {
		eng = &mockEngine{
			cfg: newTestConfig(),
			reg: newTestRegistry(),
		}
		srv = mcpserver.New(eng, "/tmp/test.yaml", "0.1.0-test", nil)
	})

	It("resolves by absolute path", func() {
		result, err := callTool(srv, "get_repository_context", map[string]any{
			"repo": "/home/user/repos/beta",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeFalse())
	})
})

// --- resource test helpers ---

// resourceReadMessage builds a JSON-RPC resources/read request as []byte.
func resourceReadMessage(uri string) []byte {
	return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":%q}}`, uri))
}

// expectResourceSuccess asserts the response is a successful resource read and
// returns the text content of the first resource.
func expectResourceSuccess(response mcp.JSONRPCMessage) string {
	resp, ok := response.(mcp.JSONRPCResponse)
	Expect(ok).To(BeTrue(), "expected JSONRPCResponse, got %T", response)

	result, ok := resp.Result.(mcp.ReadResourceResult)
	Expect(ok).To(BeTrue(), "expected ReadResourceResult, got %T", resp.Result)
	Expect(result.Contents).NotTo(BeEmpty())

	tc, ok := result.Contents[0].(mcp.TextResourceContents)
	Expect(ok).To(BeTrue(), "expected TextResourceContents, got %T", result.Contents[0])
	return tc.Text
}

// expectResourceError asserts the response is a JSON-RPC error.
func expectResourceError(response mcp.JSONRPCMessage) {
	_, ok := response.(mcp.JSONRPCError)
	Expect(ok).To(BeTrue(), "expected JSONRPCError, got %T", response)
}
