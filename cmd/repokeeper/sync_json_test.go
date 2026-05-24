// SPDX-License-Identifier: MIT
package repokeeper

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/skaphos/repokeeper/internal/engine"
)

// These specs lock the stable `reconcile` / `sync -o json` contract (SKA-207):
// a top-level JSON array of per-repo objects keyed with snake_case fields that
// mirror the MCP plan_sync/execute_sync shape.
var _ = Describe("sync -o json result shape", func() {
	marshalOne := func(res engine.SyncResult) map[string]any {
		data, err := json.Marshal(toSyncResultJSONs([]engine.SyncResult{res}))
		Expect(err).NotTo(HaveOccurred())

		var arr []map[string]any
		Expect(json.Unmarshal(data, &arr)).To(Succeed())
		Expect(arr).To(HaveLen(1), "output must be a JSON array of result objects")
		return arr[0]
	}

	It("emits stable snake_case keys for a successful fetch", func() {
		obj := marshalOne(engine.SyncResult{
			RepoID:  "github.com/org/repo",
			Path:    "/work/org/repo",
			Action:  "git fetch --all --prune",
			Outcome: engine.SyncOutcomeFetched,
			OK:      true,
		})

		Expect(obj).To(HaveKeyWithValue("repo_id", "github.com/org/repo"))
		Expect(obj).To(HaveKeyWithValue("path", "/work/org/repo"))
		Expect(obj).To(HaveKeyWithValue("outcome", "fetched"))
		Expect(obj).To(HaveKeyWithValue("ok", true))
		Expect(obj).To(HaveKeyWithValue("action", "git fetch --all --prune"))
		// error/skip_reason/planned are omitted when empty for a clean success record.
		Expect(obj).NotTo(HaveKey("error"))
		Expect(obj).NotTo(HaveKey("skip_reason"))
		Expect(obj).NotTo(HaveKey("planned"))
	})

	It("emits outcome, ok and the error reason for a skipped repo", func() {
		obj := marshalOne(engine.SyncResult{
			RepoID:  "github.com/org/no-upstream",
			Path:    "/work/org/no-upstream",
			Outcome: engine.SyncOutcomeSkippedNoUpstream,
			OK:      true,
			Error:   engine.SyncErrorSkippedNoUpstream,
		})

		Expect(obj).To(HaveKeyWithValue("repo_id", "github.com/org/no-upstream"))
		Expect(obj).To(HaveKeyWithValue("outcome", "skipped_no_upstream"))
		Expect(obj).To(HaveKeyWithValue("ok", true))
		Expect(obj).To(HaveKeyWithValue("error", engine.SyncErrorSkippedNoUpstream))
	})

	It("emits the typed skip_reason for a skipped local update", func() {
		obj := marshalOne(engine.SyncResult{
			RepoID:     "github.com/org/dirty",
			Path:       "/work/org/dirty",
			Outcome:    engine.SyncOutcomeSkippedLocalUpdate,
			OK:         true,
			SkipReason: engine.SyncReasonDirtyWorkingTree,
		})

		Expect(obj).To(HaveKeyWithValue("outcome", "skipped_local_update"))
		Expect(obj).To(HaveKeyWithValue("ok", true))
		Expect(obj).To(HaveKeyWithValue("skip_reason", engine.SyncReasonDirtyWorkingTree))
	})

	It("marks dry-run entries planned and suppresses the dry-run sentinel", func() {
		// Plan entries from the engine carry both Planned=true and a dry-run
		// sentinel ("dry-run") in Error; that sentinel must not surface in JSON.
		obj := marshalOne(engine.SyncResult{
			RepoID:  "github.com/org/repo",
			Path:    "/work/org/repo",
			Outcome: engine.SyncOutcomePlannedFetch,
			OK:      true,
			Planned: true,
			Error:   "dry-run",
		})

		Expect(obj).To(HaveKeyWithValue("planned", true))
		Expect(obj).To(HaveKeyWithValue("outcome", "planned_fetch"))
		Expect(obj).NotTo(HaveKey("error"))
	})

	It("does not report executed results as planned", func() {
		// The execute path copies the plan item without clearing Planned, so an
		// executed "fetched" result still has Planned=true on the struct. The
		// JSON projection must derive planned from the outcome, not that flag.
		obj := marshalOne(engine.SyncResult{
			RepoID:  "github.com/org/repo",
			Path:    "/work/org/repo",
			Action:  "git fetch --all --prune",
			Outcome: engine.SyncOutcomeFetched,
			OK:      true,
			Planned: true,
		})

		Expect(obj).To(HaveKeyWithValue("outcome", "fetched"))
		Expect(obj).To(HaveKeyWithValue("ok", true))
		Expect(obj).NotTo(HaveKey("planned"))
	})
})
