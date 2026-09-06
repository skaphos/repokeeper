// SPDX-License-Identifier: MIT
package mcpserver_test

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/v2/internal/config"
	"github.com/skaphos/repokeeper/v2/internal/mcpserver"
	"github.com/skaphos/repokeeper/v2/internal/registry"
)

var _ = Describe("Resource JSON contract", func() {
	var eng *mockEngine
	var srv *mcpserver.MCPServer
	BeforeEach(func() {
		cfg := config.DefaultConfig()
		eng = &mockEngine{cfg: &cfg, reg: &registry.Registry{Entries: []registry.Entry{{
			RepoID: "example/repo", Path: "/repos/repo", Status: registry.StatusPresent,
			RemoteURL: "https://user:secret@example.com/repo.git",
		}}}}
		cfg.Registry = eng.reg
		srv = mcpserver.New(eng, "", "test", nil)
	})

	read := func(srv *mcpserver.MCPServer, uri string) map[string]any {
		text := expectResourceSuccess(srv.Inner().HandleMessage(context.Background(), resourceReadMessage(uri)))
		var payload map[string]any
		Expect(json.Unmarshal([]byte(text), &payload)).To(Succeed())
		Expect(text).NotTo(ContainSubstring("secret"))
		return payload
	}

	It("uses the same named fields for direct and config-embedded registry snapshots", func() {
		reg := read(srv, "repokeeper://registry")
		Expect(reg).To(HaveLen(1))
		Expect(reg).To(HaveKey("repos"))
		entry := reg["repos"].([]any)[0].(map[string]any)
		Expect(entry).To(Equal(map[string]any{
			"repo_id": "example/repo", "path": "/repos/repo", "status": "present",
			"remote_url": "https://***@example.com/repo.git",
		}))
		Expect(read(srv, "repokeeper://repo/example/repo")).To(Equal(entry))
		cfg := read(srv, "repokeeper://config")
		Expect(cfg["registry"]).To(Equal(reg))
		Expect(cfg).To(HaveKeyWithValue("apiVersion", config.ConfigAPIVersion))
		Expect(cfg).To(HaveKey("defaults"))
		Expect(cfg["defaults"]).To(HaveKeyWithValue("remote_name", "origin"))
		Expect(cfg["branch_policy"]).To(HaveKeyWithValue("require_merged", true))
		Expect(eng.reg.Entries[0].RemoteURL).To(Equal("https://user:secret@example.com/repo.git"))
	})

	It("omits unknown times and emits observed times in UTC", func() {
		observed := time.Date(2026, 9, 5, 12, 0, 0, 0, time.FixedZone("CDT", -5*60*60))
		eng.reg.UpdatedAt = observed
		eng.reg.Entries[0].LastSeen = observed
		reg := read(srv, "repokeeper://registry")
		Expect(reg).To(HaveKeyWithValue("updated_at", "2026-09-05T17:00:00Z"))
		Expect(reg["repos"].([]any)[0]).To(HaveKeyWithValue("last_seen", "2026-09-05T17:00:00Z"))
	})

	It("distinguishes an empty registry from an absent config registry", func() {
		eng.reg.Entries = nil
		Expect(read(srv, "repokeeper://registry")["repos"]).To(Equal([]any{}))
		eng.cfg.Registry = nil
		eng.cfg.Exclude = nil
		eng.cfg.BranchPolicy.ProtectedPatterns = nil
		cfg := read(srv, "repokeeper://config")
		Expect(cfg).NotTo(HaveKey("registry"))
		Expect(cfg["exclude"]).To(Equal([]any{}))
		Expect(cfg["branch_policy"]).To(HaveKeyWithValue("protected_patterns", []any{}))
	})

	It("redacts repository listings and omits unknown last_seen", func() {
		result, err := callTool(srv, "list_repositories", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.IsError).To(BeFalse())
		var entries []map[string]any
		Expect(json.Unmarshal(resultJSON(result), &entries)).To(Succeed())
		Expect(entries[0]).To(HaveKeyWithValue("remote_url", "https://***@example.com/repo.git"))
		Expect(entries[0]).NotTo(HaveKey("last_seen"))
		Expect(eng.reg.Entries[0].RemoteURL).To(ContainSubstring("secret"))
	})
})
