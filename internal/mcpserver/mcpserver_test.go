// SPDX-License-Identifier: MIT
package mcpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	// Mutation tracking
	syncResult       []engine.SyncResult
	syncErr          error
	scanResult       []model.RepoStatus
	scanErr          error
	deleteRepoCalled bool
	deleteRepoErr    error
	cloneCalled      bool
	cloneErr         error
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
func (e *mockEngine) Sync(_ context.Context, opts engine.SyncOptions) ([]engine.SyncResult, error) {
	if e.syncErr != nil {
		return nil, e.syncErr
	}
	if e.syncResult != nil {
		return e.syncResult, nil
	}
	return []engine.SyncResult{}, nil
}
func (e *mockEngine) ExecuteSyncPlanWithCallbacks(_ context.Context, plan []engine.SyncResult, _ engine.SyncOptions, _ engine.SyncStartCallback, _ engine.SyncResultCallback) ([]engine.SyncResult, error) {
	if e.syncErr != nil {
		return nil, e.syncErr
	}
	// Simulate execution: mark planned items as executed.
	results := make([]engine.SyncResult, len(plan))
	for i, p := range plan {
		results[i] = p
		results[i].Planned = false
		results[i].OK = true
		results[i].Outcome = engine.SyncOutcomeFetched
	}
	return results, nil
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
func (e *mockEngine) ResetRepo(_ context.Context, _, _ string) error { return nil }
func (e *mockEngine) DeleteRepo(_ context.Context, _, _ string, _ bool) error {
	e.deleteRepoCalled = true
	return e.deleteRepoErr
}
func (e *mockEngine) CloneAndRegister(_ context.Context, _, _, _ string, _ bool) error {
	e.cloneCalled = true
	return e.cloneErr
}
func (e *mockEngine) Scan(_ context.Context, _ engine.ScanOptions) ([]model.RepoStatus, error) {
	if e.scanErr != nil {
		return nil, e.scanErr
	}
	if e.scanResult != nil {
		return e.scanResult, nil
	}
	return []model.RepoStatus{}, nil
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

func structuredContentMap(result *mcp.CallToolResult) map[string]any {
	Expect(result).NotTo(BeNil())
	Expect(result.StructuredContent).NotTo(BeNil())
	structured, ok := result.StructuredContent.(map[string]any)
	Expect(ok).To(BeTrue(), "expected map structured content, got %T", result.StructuredContent)
	return structured
}

func structuredListJSON(result *mcp.CallToolResult, key string) []byte {
	structured := structuredContentMap(result)
	payload, ok := structured[key]
	Expect(ok).To(BeTrue(), "expected structured content key %q", key)
	b, err := json.Marshal(payload)
	Expect(err).NotTo(HaveOccurred())
	return b
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

	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "mcpserver-test-*")
		Expect(err).NotTo(HaveOccurred())
		eng = &mockEngine{
			cfg: newTestConfig(),
			reg: newTestRegistry(),
		}
		cfgPath := filepath.Join(tmpDir, ".repokeeper.yaml")
		srv = mcpserver.New(eng, cfgPath, "0.1.0-test", nil)
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
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
		// Phase 3 tools
		Expect(tools).To(HaveKey("scan_workspace"))
		Expect(tools).To(HaveKey("plan_sync"))
		Expect(tools).To(HaveKey("execute_sync"))
		Expect(tools).To(HaveKey("set_labels"))
		Expect(tools).To(HaveKey("add_repository"))
		Expect(tools).To(HaveKey("remove_repository"))
	})

	It("publishes explicit string item schemas for array mutation inputs", func() {
		tools := srv.Inner().ListTools()

		rootsSchema, ok := tools["scan_workspace"].Tool.InputSchema.Properties["roots"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(rootsSchema["type"]).To(Equal("array"))
		rootsItems, ok := rootsSchema["items"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(rootsItems["type"]).To(Equal("string"))

		removeSchema, ok := tools["set_labels"].Tool.InputSchema.Properties["remove"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(removeSchema["type"]).To(Equal("array"))
		removeItems, ok := removeSchema["items"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(removeItems["type"]).To(Equal("string"))

		setSchema, ok := tools["set_labels"].Tool.InputSchema.Properties["set"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(setSchema["type"]).To(Equal("object"))
		additionalProperties, ok := setSchema["additionalProperties"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(additionalProperties["type"]).To(Equal("string"))
	})

	It("publishes execute_sync confirm as a required boolean safety gate", func() {
		tool := srv.Inner().ListTools()["execute_sync"].Tool

		Expect(tool.InputSchema.Required).To(ContainElement("confirm"))
		confirmSchema, ok := tool.InputSchema.Properties["confirm"].(map[string]any)
		Expect(ok).To(BeTrue())
		Expect(confirmSchema["type"]).To(Equal("boolean"))
	})

	// --- Phase 1 tools ---

	Describe("list_repositories", func() {
		It("returns all repos when no filters given", func() {
			result, err := callTool(srv, "list_repositories", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())

			var structuredRepos []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "repositories"), &structuredRepos)).To(Succeed())
			Expect(structuredRepos).To(Equal(repos))
			Expect(repos).To(HaveLen(3))
			Expect(repos[0]["last_seen"]).To(Equal("2026-04-01T12:00:00Z"))
		})

		It("returns empty wrapped repositories when filters match nothing", func() {
			result, err := callTool(srv, "list_repositories", map[string]any{
				"label_selector": "team=missing",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(BeEmpty())

			var structuredRepos []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "repositories"), &structuredRepos)).To(Succeed())
			Expect(structuredRepos).To(Equal(repos))
		})

		It("formats last_seen in UTC RFC3339", func() {
			eng.reg.Entries[0].LastSeen = time.Date(2026, 4, 1, 7, 0, 0, 0, time.FixedZone("CDT", -5*60*60))

			result, err := callTool(srv, "list_repositories", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos[0]["last_seen"]).To(Equal("2026-04-01T12:00:00Z"))
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
			Expect(resp["config_path"]).To(HaveSuffix(".repokeeper.yaml"))
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

			var structuredRepos []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "repositories"), &structuredRepos)).To(Succeed())
			Expect(structuredRepos).To(Equal(repos))
			Expect(repos).To(HaveLen(2))
			Expect(repos[0]["match_reason"]).To(Equal("all"))
		})

		It("returns empty wrapped repositories when no repos match", func() {
			result, err := callTool(srv, "select_repositories", map[string]any{
				"name_match": "missing",
			})
			Expect(err).NotTo(HaveOccurred())

			var repos []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &repos)).To(Succeed())
			Expect(repos).To(BeEmpty())

			var structuredRepos []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "repositories"), &structuredRepos)).To(Succeed())
			Expect(structuredRepos).To(Equal(repos))
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

			var structuredRepos []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "repositories"), &structuredRepos)).To(Succeed())
			Expect(structuredRepos).To(Equal(repos))
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

			var structuredRepos []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "repositories"), &structuredRepos)).To(Succeed())
			Expect(structuredRepos).To(Equal(repos))
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

			var structuredRepos []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "repositories"), &structuredRepos)).To(Succeed())
			Expect(structuredRepos).To(Equal(repos))
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

	// --- Phase 3: Mutation tools ---

	Describe("scan_workspace", func() {
		It("returns scan results", func() {
			eng.scanResult = []model.RepoStatus{
				{RepoID: "github.com/example/alpha", Path: "/home/user/repos/alpha"},
				{RepoID: "github.com/example/beta", Path: "/home/user/repos/beta"},
			}

			result, err := callTool(srv, "scan_workspace", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			Expect(resp["discovered"]).To(BeNumerically("==", 2))
			repos := resp["repos"].([]any)
			Expect(repos).To(HaveLen(2))
		})

		It("returns error when scan fails", func() {
			eng.scanErr = fmt.Errorf("scan failure")
			result, err := callTool(srv, "scan_workspace", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("rejects non-string roots items", func() {
			result, err := callTool(srv, "scan_workspace", map[string]any{
				"roots": []any{"/home/user/repos", 1},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())

			text := string(resultJSON(result))
			Expect(text).To(ContainSubstring(`argument "roots" item 1 must be a string`))
		})
	})

	Describe("plan_sync", func() {
		BeforeEach(func() {
			eng.syncResult = []engine.SyncResult{
				{RepoID: "github.com/example/alpha", Path: "/home/user/repos/alpha", Action: "fetch --all --prune", Outcome: engine.SyncOutcomeFetched, Planned: true},
			}
		})

		It("returns dry-run plan", func() {
			result, err := callTool(srv, "plan_sync", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var entries []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &entries)).To(Succeed())

			var structuredEntries []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "plan"), &structuredEntries)).To(Succeed())
			Expect(structuredEntries).To(Equal(entries))
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]["planned"]).To(BeTrue())
			Expect(entries[0]["repo_id"]).To(Equal("github.com/example/alpha"))
		})

		It("returns empty wrapped plan when there are no sync actions", func() {
			eng.syncResult = nil

			result, err := callTool(srv, "plan_sync", nil)
			Expect(err).NotTo(HaveOccurred())

			var entries []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &entries)).To(Succeed())
			Expect(entries).To(BeEmpty())

			var structuredEntries []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "plan"), &structuredEntries)).To(Succeed())
			Expect(structuredEntries).To(Equal(entries))
		})

		It("returns error when sync fails", func() {
			eng.syncErr = fmt.Errorf("sync failure")
			result, err := callTool(srv, "plan_sync", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	Describe("execute_sync", func() {
		BeforeEach(func() {
			eng.syncResult = []engine.SyncResult{
				{RepoID: "github.com/example/alpha", Path: "/home/user/repos/alpha", Action: "fetch --all --prune", Outcome: engine.SyncOutcomeFetched, Planned: true},
			}
		})

		It("rejects without confirm=true", func() {
			result, err := callTool(srv, "execute_sync", map[string]any{
				"confirm": false,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())

			text := string(resultJSON(result))
			Expect(text).To(ContainSubstring("safety gate"))
		})

		It("rejects when confirm is missing", func() {
			result, err := callTool(srv, "execute_sync", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())

			text := string(resultJSON(result))
			Expect(text).To(ContainSubstring(`required argument "confirm" not found`))
		})

		It("rejects non-boolean confirm values", func() {
			result, err := callTool(srv, "execute_sync", map[string]any{
				"confirm": "true",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())

			text := string(resultJSON(result))
			Expect(text).To(ContainSubstring(`argument "confirm" must be a boolean`))
		})

		It("executes sync with confirm=true", func() {
			result, err := callTool(srv, "execute_sync", map[string]any{
				"confirm": true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var entries []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &entries)).To(Succeed())

			var structuredEntries []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "results"), &structuredEntries)).To(Succeed())
			Expect(structuredEntries).To(Equal(entries))
			Expect(entries).To(HaveLen(1))
			Expect(entries[0]["ok"]).To(BeTrue())
			Expect(entries[0]["outcome"]).To(Equal("fetched"))
		})

		It("returns empty wrapped results when execution plan is empty", func() {
			eng.syncResult = nil

			result, err := callTool(srv, "execute_sync", map[string]any{
				"confirm": true,
			})
			Expect(err).NotTo(HaveOccurred())

			var entries []map[string]any
			Expect(json.Unmarshal(resultJSON(result), &entries)).To(Succeed())
			Expect(entries).To(BeEmpty())

			var structuredEntries []map[string]any
			Expect(json.Unmarshal(structuredListJSON(result, "results"), &structuredEntries)).To(Succeed())
			Expect(structuredEntries).To(Equal(entries))
		})

		It("returns error when sync fails", func() {
			eng.syncErr = fmt.Errorf("sync failure")
			result, err := callTool(srv, "execute_sync", map[string]any{
				"confirm": true,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	Describe("set_labels", func() {
		It("sets labels on a repo", func() {
			result, err := callTool(srv, "set_labels", map[string]any{
				"repo": "github.com/example/alpha",
				"set":  map[string]any{"tier": "critical"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			Expect(resp["repo_id"]).To(Equal("github.com/example/alpha"))
			labels := resp["labels"].(map[string]any)
			Expect(labels["tier"]).To(Equal("critical"))
			// Original labels should still be present
			Expect(labels["team"]).To(Equal("platform"))
		})

		It("removes labels from a repo", func() {
			result, err := callTool(srv, "set_labels", map[string]any{
				"repo":   "github.com/example/alpha",
				"remove": []any{"env"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			labels := resp["labels"].(map[string]any)
			Expect(labels).NotTo(HaveKey("env"))
			Expect(labels["team"]).To(Equal("platform"))
		})

		It("returns error for unknown repo", func() {
			result, err := callTool(srv, "set_labels", map[string]any{
				"repo": "nonexistent/repo",
				"set":  map[string]any{"x": "y"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error when repo parameter is missing", func() {
			result, err := callTool(srv, "set_labels", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("rejects non-string remove items", func() {
			result, err := callTool(srv, "set_labels", map[string]any{
				"repo":   "github.com/example/alpha",
				"remove": []any{"env", 1},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())

			text := string(resultJSON(result))
			Expect(text).To(ContainSubstring(`argument "remove" item 1 must be a string`))
		})

		It("rejects non-string set values", func() {
			result, err := callTool(srv, "set_labels", map[string]any{
				"repo": "github.com/example/alpha",
				"set":  map[string]any{"tier": 1},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())

			text := string(resultJSON(result))
			Expect(text).To(ContainSubstring(`argument "set" key "tier" must have a string value`))
		})
	})

	Describe("add_repository", func() {
		It("clones and registers a repository", func() {
			result, err := callTool(srv, "add_repository", map[string]any{
				"url":  "git@github.com:example/new.git",
				"path": "/home/user/repos/new",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())
			Expect(eng.cloneCalled).To(BeTrue())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			Expect(resp["path"]).To(Equal("/home/user/repos/new"))
			Expect(resp["status"]).To(Equal("cloned"))
		})

		It("returns error when clone fails", func() {
			eng.cloneErr = fmt.Errorf("clone failed")
			result, err := callTool(srv, "add_repository", map[string]any{
				"url":  "git@github.com:example/fail.git",
				"path": "/home/user/repos/fail",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error when url is missing", func() {
			result, err := callTool(srv, "add_repository", map[string]any{
				"path": "/home/user/repos/new",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error when path is missing", func() {
			result, err := callTool(srv, "add_repository", map[string]any{
				"url": "git@github.com:example/new.git",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})
	})

	Describe("remove_repository", func() {
		It("removes a repo from registry (tracking-only)", func() {
			result, err := callTool(srv, "remove_repository", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeFalse())
			Expect(eng.deleteRepoCalled).To(BeTrue())

			var resp map[string]any
			Expect(json.Unmarshal(resultJSON(result), &resp)).To(Succeed())
			Expect(resp["repo_id"]).To(Equal("github.com/example/alpha"))
			Expect(resp["removed"]).To(BeTrue())
		})

		It("returns error when delete fails", func() {
			eng.deleteRepoErr = fmt.Errorf("delete failed")
			result, err := callTool(srv, "remove_repository", map[string]any{
				"repo": "github.com/example/alpha",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
		})

		It("returns error when repo parameter is missing", func() {
			result, err := callTool(srv, "remove_repository", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.IsError).To(BeTrue())
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
