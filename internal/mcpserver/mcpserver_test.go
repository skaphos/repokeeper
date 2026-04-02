// SPDX-License-Identifier: MIT
package mcpserver_test

import (
	"context"
	"encoding/json"
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
}

func (e *mockEngine) Status(_ context.Context, _ engine.StatusOptions) (*model.StatusReport, error) {
	return nil, nil
}
func (e *mockEngine) Sync(_ context.Context, _ engine.SyncOptions) ([]engine.SyncResult, error) {
	return nil, nil
}
func (e *mockEngine) ExecuteSyncPlanWithCallbacks(_ context.Context, _ []engine.SyncResult, _ engine.SyncOptions, _ engine.SyncStartCallback, _ engine.SyncResultCallback) ([]engine.SyncResult, error) {
	return nil, nil
}
func (e *mockEngine) InspectRepo(_ context.Context, path string) (*model.RepoStatus, error) {
	if e.inspectResult != nil {
		return e.inspectResult, e.inspectErr
	}
	return &model.RepoStatus{
		RepoID: "github.com/example/alpha",
		Path:   path,
		Head:   model.Head{Branch: "main"},
		Tracking: model.Tracking{
			Upstream: "origin/main",
			Status:   model.TrackingEqual,
		},
	}, e.inspectErr
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

// --- test data ---

func newTestRegistry() *registry.Registry {
	return &registry.Registry{
		UpdatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
		Entries: []registry.Entry{
			{
				RepoID:    "github.com/example/alpha",
				Path:      "/home/user/repos/alpha",
				RemoteURL: "git@github.com:example/alpha.git",
				Type:      "checkout",
				Labels:    map[string]string{"team": "platform", "env": "prod"},
				Status:    registry.StatusPresent,
				LastSeen:  time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
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

	It("creates a server with registered tools", func() {
		Expect(srv.Inner()).NotTo(BeNil())
		tools := srv.Inner().ListTools()
		Expect(tools).To(HaveKey("list_repositories"))
		Expect(tools).To(HaveKey("get_repository_context"))
		Expect(tools).To(HaveKey("get_workspace_config"))
	})

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
