# Graph Report - repokeeper  (2026-09-05)

## Corpus Check
- 394 files · ~344,473 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3597 nodes · 7216 edges · 257 communities (243 shown, 14 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 750 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `914dccd2`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Save
- GitAdapter
- coverage_boost_test.go
- github.com/spf13/cobra.Command
- MultiAdapter
- Implementation Plan: Recipe-Driven End-to-End Test Harness
- withInstallEnv
- github.com/mark3labs/mcp-go/mcp.CallToolRequest
- codex.go
- ApplyPlans
- ADR-0008: MCP Install Tooling for Supported Agent Runtimes
- server.go
- ADR-0017: Retire the TUI
- runDescribeRepo
- testing.T
- .agents/skills/speckit-analyze/SKILL.md
- context.Context
- status.go
- model.go
- multiStubAdapter
- common.sh
- tools_mutation.go
- index.go
- export_test.go
- .handleBuildWorkspaceInventory
- Execution Steps
- Worktree
- runtime_test.go
- NewGitAdapter
- Entry
- Tasks: [FEATURE NAME]
- Milestones
- Tasks: Recipe-Driven End-to-End Test Harness
- import_test.go
- repometa_test.go
- Config
- registry_test.go
- import_clone.go
- materializeRecipe
- ADR-0002: Branch Switch and Checkout Workflow Boundaries
- sync.go
- RepoStatus
- Tasks: Distribution Channel Conformance
- Registry
- repometa.go
- perf/main.go
- README.md
- ExecutionResult
- RepoKeeper
- 5.1 CLI commands (v1)
- ADR-0001: MCP Server for Agent-Native Repository Querying and Planning
- writeRepairUpstreamTable
- addFormatFlag
- index_test.go
- MCPToolCase
- ADR-0006: Adapter-Facing Contract Stability and Versioning
- RepoKeeper
- Manpage Plan
- github.com/mark3labs/mcp-go/mcp.CallToolResult
- discovery.go
- Changelog
- planAdapter
- newPlanExecEngine
- 9. Future: Cross-Machine Registry Sync
- helpers_test.go
- writeEmptyConfig
- ADR-0003: Sync Policy and Execution Modes
- ADR-0004: Prune Workflow Boundaries and Safety Model
- Specification Quality Checklist: GitHub Remote End-to-End Expansion
- WriteTable
- ADR-0007: Release Binary Publishing and Homebrew Distribution
- NewGitErrorClassifier
- opencodeAdapter
- Feature Specification: Distribution Channel Conformance
- ADR-0005: Workspace Config vs Repo-Local Metadata Ownership
- buildIndexProposal
- ADR-0011: Credential and Auth Handling Deferred
- Verification Checklist (with Evidence Placeholders)
- SplitCSV
- Phase 0 Research: Recipe-Driven End-to-End Test Harness
- Feature Specification: [FEATURE NAME]
- cloneMetadataMap
- HgAdapter
- RepoKeeper — Design Spec
- ADR-0010: Repo ID Normalization Stability
- ADR-0014: Local Branch Prune-Safety Classification Model
- ADR-0015: Branch Retention and Protection Policy
- .claude/skills/speckit-analyze/SKILL.md
- LocalBranches
- Decision
- RepoKeeper Adapter Contract
- GitHub Copilot Instructions for RepoKeeper
- SyncResult
- StatusReport
- Core Principles
- Core Principles
- Repository Guidelines
- install.go
- 5.3 Kubectl-Style CLI Alignment (Milestone 6+)
- Command Notes
- .agents/skills/speckit-plan/SKILL.md
- hintForErrorClass
- .handleGetWorkspaceConfig
- Release Process
- repair_upstream_test.go
- .agents/skills/speckit-specify/SKILL.md
- Decision
- repairResolveTargetBranch
- Scope
- Contributing Guidelines
- Quickstart: Building an Adapter Against the RepoKeeper Contract
- Implementation Plan: [FEATURE]
- 6. Data Model
- Execution Steps
- SortRegistryEntries
- MCP Server Setup
- mcpSession
- newToolError
- dropIgnoredImportEntries
- Phase 1 Data Model: Recipe-Driven End-to-End Test Harness
- readJSONDoc
- .agents/skills/speckit-tasks/SKILL.md
- os/exec.Cmd
- 9. Stretch Goals
- Phase 0 Research: Distribution Channel Conformance
- grokAdapter
- ADR-0013: GoReleaser Owns the GitHub Release
- Decision
- ADR-0016: No Self-Update Subcommand
- benchmarkEngineWithRepos
- recipe_test.go
- Execution Contract
- sync.Mutex
- Decision
- newEngineWith
- Installation
- ClassifyError
- Quickstart: Recipe-Driven End-to-End Harness
- Contract: `ghcr.io/skaphos/repokeeper`
- 4. Safety & Policy
- 8. Architecture
- NormalizeURL
- Phase 1 Data Model: Distribution Channel Conformance
- MockRunner
- Feature Specification: Stable Plugin Adapter Contract
- declaration.go
- [CHECKLIST TYPE] Checklist: [FEATURE NAME]
- Alternatives Considered
- WriteAtomic
- .agents/skills/speckit-checklist/SKILL.md
- check-coverage.sh
- .agents/skills/speckit-clarify/SKILL.md
- .DeleteRepo
- .agents/skills/speckit-implement/SKILL.md
- 001-distribution-channels/spec.md
- .agents/skills/speckit-constitution/SKILL.md
- importTargetRelativePath
- .agents/skills/speckit-taskstoissues/SKILL.md
- NopLogger
- runtime.go
- tools_metadata.go
- 004-adapter-contract/spec.md
- TestRemoteMismatchReconcileHelpers
- nopLogger
- Git Compatibility Declaration Contract
- Tasks: Stable Plugin Adapter Contract
- Recipe Contract
- assertions_test.go
- TestWorkerChannelBufferSize
- MCP Tool Matrix Contract
- environment_test.go
- buildStatusJSONOutput
- run_generation
- version_test.go
- .claude/skills/speckit-plan/SKILL.md
- .claude/skills/speckit-specify/SKILL.md
- perf/README.md
- coverage-report.sh script
- verify-mcp.sh
- .claude/skills/speckit-tasks/SKILL.md
- Implementation Plan: Stable Plugin Adapter Contract
- github.com/skaphos/repokeeper/tools
- comment_guard_test.go
- Implementation Plan: Distribution Channel Conformance
- Quickstart: Validating Distribution Channel Conformance
- server.json
- Contract: `server.json` — MCP Registry Entry
- Research: Stable Plugin Adapter Contract
- WorkspaceRecipe
- vcs/localbranch_test.go
- Contract: Release Artifacts and Channel Verification
- TestResolveEditorCommand
- Contract: Adapter-Facing Surface Inventory
- 5. Container behavior (FR-024 – FR-029)
- TestResolveAbsoluteTargetPath
- Requirements Quality Checklist: Stable Plugin Adapter Contract
- Contract: `repokeeper version`
- Human-readable output
- .claude/skills/speckit-checklist/SKILL.md
- NewHgAdapter
- Contract: Adapter Response Envelope
- .claude/skills/speckit-clarify/SKILL.md
- .claude/skills/speckit-implement/SKILL.md
- Data Model: Stable Plugin Adapter Contract
- e2e_suite_test.go
- .claude/skills/speckit-constitution/SKILL.md
- edit_test.go
- normalizedCLIOutcome
- Specification Quality Checklist: Distribution Channel Conformance
- .claude/skills/speckit-taskstoissues/SKILL.md
- status_prune_test.go
- adapter_test.go
- toSyncResultJSON
- Registry
- github.com/skaphos/repokeeper/v2

## God Nodes (most connected - your core abstractions)
1. `Entry` - 70 edges
2. `Save()` - 62 edges
3. `DefaultConfig()` - 61 edges
4. `Registry` - 59 edges
5. `RepoStatus` - 55 edges
6. `SyncResult` - 45 edges
7. `Engine` - 44 edges
8. `NewGitAdapter()` - 44 edges
9. `withTestConfig()` - 43 edges
10. `Config` - 41 edges

## Surprising Connections (you probably didn't know these)
- `writeTestConfigAndRegistry()` --calls--> `Save()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/registry/registry.go
- `TestSyncRunEPersistsRegistryAfterCheckoutMissingClone()` --calls--> `Load()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/config/config.go
- `TestStatusRunEIncludesRepoMetadata()` --calls--> `Load()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/config/config.go
- `TestIndexRunEWriteRefreshesRepoMetadataSnapshotInConfig()` --calls--> `Load()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/config/config.go
- `TestIndexRunECanPromoteLocalLabels()` --calls--> `Load()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/repometa/repometa.go

## Import Cycles
- None detected.

## Communities (257 total, 14 thin omitted)

### Community 0 - "Save"
Cohesion: 0.12
Nodes (51): TestGetReposReportsErrors(), TestDescribeRunEIncludesRepoMetadata(), TestDescribeRunEPaths(), TestIndexReposRunEPreviewsAndWritesSelectedRepos(), TestIndexReposRunERequiresPromoteFlag(), TestIndexRunECanPromoteLocalLabels(), TestIndexRunEFailsEarlyWhenMetadataExistsWithoutForce(), TestIndexRunEForceResolvesDualMetadataFiles() (+43 more)

### Community 1 - "GitAdapter"
Cohesion: 0.08
Nodes (35): GitRunner, CleanFD(), Clone(), Fetch(), Runner, HasSubmodules(), Head(), TestTrackingFromShort() (+27 more)

### Community 2 - "coverage_boost_test.go"
Cohesion: 0.08
Nodes (30): TestLogOutputWriteFailureLogsError(), TestLogOutputWriteFailureNilError(), TestMarshalToGenericMarshalErrorPath(), TestMarshalToGenericUnmarshalErrorPath(), TestNewSyncProgressWriter(), TestResolveCustomColumnValueEdgeCases(), TestRowsForCustomColumnsFallbackPaths(), TestSyncProgressWriterAdditionalBranches() (+22 more)

### Community 3 - "github.com/spf13/cobra.Command"
Cohesion: 0.11
Nodes (38): TestSyncProgressMessageKinds(), TestFlagGettersBranchCoverage(), TestLogHelpers(), assumeYes(), configOverride(), debugf(), Execute(), ExecuteWithExitCode() (+30 more)

### Community 4 - "MultiAdapter"
Cohesion: 0.11
Nodes (7): stubLBAdapter, syncFetchAction(), Adapter, LocalBranchSignal, LocalBranchInspector, MultiAdapter, RemoteTrackingRefInspector

### Community 5 - "Implementation Plan: Recipe-Driven End-to-End Test Harness"
Cohesion: 0.06
Nodes (31): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Recipe-Driven End-to-End Test Harness, Compatibility and Provisioning Design, Constitution Check, Delivery Sequence (+23 more)

### Community 6 - "withInstallEnv"
Cohesion: 0.08
Nodes (49): resetInstallListFlags(), runInstallListWithFlags(), TestInstallListAllNotRegistered(), TestInstallListInvalidScope(), TestInstallListJSON(), TestInstallListProjectCodexUnsupported(), TestInstallListRegisteredStateMatchesExecutable(), TestInstallListStaleWhenCommandDiffers() (+41 more)

### Community 7 - "github.com/mark3labs/mcp-go/mcp.CallToolRequest"
Cohesion: 0.19
Nodes (11): github.com/mark3labs/mcp-go/mcp.CallToolRequest, resolveRepo(), T, newEnvelope(), newStructuredListResult(), newStructuredResult(), MCPServer, MCPServer (+3 more)

### Community 8 - "codex.go"
Cohesion: 0.16
Nodes (14): codexServersMap(), Entry, init(), readTOMLDoc(), refuseIfTOMLComments(), skipTOMLDelim(), skipTOMLSingleLine(), tomlHasComments() (+6 more)

### Community 9 - "ApplyPlans"
Cohesion: 0.19
Nodes (15): Engine, RemoteMismatchPlan, RemoteMismatchReconcileMode, ApplyPlans(), BuildPlans(), findRegistryEntryIndexForStatus(), ParseReconcileMode(), primaryRemoteURL() (+7 more)

### Community 10 - "ADR-0008: MCP Install Tooling for Supported Agent Runtimes"
Cohesion: 0.10
Nodes (19): 1. CLI surface, 1. Keep `skill install/uninstall`, add parallel `mcp install/uninstall/list`, 2. Architecture: per-runtime adapter interface, 2. `repokeeper install` as a leaf verb, with a separate top-level `repokeeper status` for state, 3. Data-table architecture instead of per-runtime adapters, 3. Documentation changes, 4. Silent fallback for `--scope project --codex` to user scope, 5. Prompt before every overwrite (+11 more)

### Community 11 - "server.go"
Cohesion: 0.05
Nodes (66): assertInventory(), inventoryRows(), inventorySection(), TestPublishedAdapterInventoryMatchesRegistrations(), github.com/mark3labs/mcp-go/mcp.ReadResourceRequest, github.com/mark3labs/mcp-go/mcp.Resource, github.com/mark3labs/mcp-go/mcp.ResourceContents, github.com/mark3labs/mcp-go/mcp.ResourceTemplate (+58 more)

### Community 12 - "ADR-0017: Retire the TUI"
Cohesion: 0.12
Nodes (16): ADR-0017: Retire the TUI, Alternatives Considered, Consequences, Context, Decision, Deleting beats deprecating, Deprecate for one release cycle, then remove, Half-built surfaces are a liability, not an option value (+8 more)

### Community 13 - "runDescribeRepo"
Cohesion: 0.11
Nodes (35): TestCheckoutSelectorMissingPathUnderSymlinkRoot(), TestCheckoutSelectorPrecedenceAndAmbiguity(), TestCheckoutSelectorSymlinkAndAtSignPaths(), TestStatusTableErrorColumn(), canonicalPathForMatch(), describeCheckoutID(), pathWithinBase(), runDescribeRepo() (+27 more)

### Community 14 - "testing.T"
Cohesion: 0.03
Nodes (98): TestRepokeeperSuite(), testing.T, TestCliio(), TestConfig(), TestDiscovery(), TestEngine(), TestGitx(), copyFixture() (+90 more)

### Community 15 - ".agents/skills/speckit-analyze/SKILL.md"
Cohesion: 0.08
Nodes (25): 1. Initialize Analysis Context, 2. Load Artifacts (Progressive Disclosure), 3. Build Semantic Models, 4. Detection Passes (Token-Efficient Analysis), 5. Severity Assignment, 6. Produce Compact Analysis Report, 7. Provide Next Actions, 8. Offer Remediation (+17 more)

### Community 16 - "context.Context"
Cohesion: 0.05
Nodes (8): stubAdapter, benchAdapter, context.Context, Head, Remote, Submodules, repoContextResponse, adapterStub

### Community 17 - "status.go"
Cohesion: 0.14
Nodes (28): TestWriteRemoteMismatchPlan(), TestDivergedAdviceAndTable(), TestRelWithinResolvesSymlinkedBase(), TestWriteStatusDetailsAndHelpers(), buildDivergedAdvice(), countGoneRepos(), displayRepoPath(), displayTrackingStatusNoColor() (+20 more)

### Community 18 - "model.go"
Cohesion: 0.13
Nodes (21): localBranchNamesByCategory(), Engine, TestUpstreamStatusFromSignal(), upstreamStatusFromSignal(), LocalBranch, LocalBranchStatus, PruneCategory, PruneReason (+13 more)

### Community 19 - "multiStubAdapter"
Cohesion: 0.11
Nodes (8): NewAdapterForSelection(), ParseAdapterSelection(), TestMultiAdapterDelegatesAllMethods(), TestMultiAdapterRoutesByPath(), TestMultiAdapterRoutesCapabilityMethodsByPath(), TestNewAdapterForSelection(), TestParseAdapterSelection(), multiStubAdapter

### Community 20 - "common.sh"
Cohesion: 0.08
Nodes (17): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+9 more)

### Community 21 - "tools_mutation.go"
Cohesion: 0.15
Nodes (14): matchesStatusFilter(), countNewRegistryEntries(), countRegistryEntriesWithStatus(), registryEntryKey(), registryEntrySet(), EntryStatus, addRepoResponse, listRepoEntry (+6 more)

### Community 22 - "index.go"
Cohesion: 0.18
Nodes (15): detectAuthoritativePaths(), detectLowValuePaths(), detectReadmeEntrypoint(), fallbackMetadataPath(), guessRepoMetadataDefaults(), humanizeRepoName(), parseIndexAssignments(), parseIndexList() (+7 more)

### Community 23 - "export_test.go"
Cohesion: 0.13
Nodes (19): TestCommonPathRoot(), commonPathRoot(), exportEntriesWithEmbeddedCredentials(), exportEntryPath(), inferRegistrySharedRoot(), populateExportBranches(), prepareRegistryForExport(), TestCloneRegistry() (+11 more)

### Community 24 - ".handleBuildWorkspaceInventory"
Cohesion: 0.13
Nodes (18): filterBulkIndexEntriesByLabels(), filterStatusReportByLabels(), filterStatusReportByLocalLabels(), MCPServer, mergeLabels(), MCPServer, buildMatchReason(), enrichAnnotations() (+10 more)

### Community 25 - "Execution Steps"
Cohesion: 0.12
Nodes (15): 1. Initialize Convergence Context, 2. Load Artifacts (Progressive Disclosure), 3. Build the Intent Inventory, 4. Assess the Codebase and Classify Findings, 5. Assign Severity, 6. Present the In-Session Findings Summary, 7. Append Convergence Tasks (or report converged), 8. Provide Next Actions (Handoff) (+7 more)

### Community 26 - "Worktree"
Cohesion: 0.18
Nodes (6): ForEachRefEntry, ParseForEachRef(), ParsePorcelainStatus(), ParseRemotePruneDryRun(), ParseRevListCount(), Worktree

### Community 27 - "runtime_test.go"
Cohesion: 0.35
Nodes (11): names(), TestAllReturnsSortedCopy(), TestAllSortsFakeRuntimes(), TestByNameHit(), TestByNameMiss(), TestByNameMissAgainstNonEmpty(), TestResolveAutoDetect(), TestResolveAutoDetectError() (+3 more)

### Community 28 - "NewGitAdapter"
Cohesion: 0.20
Nodes (22): TestEngineGuardErrors(), TestExecuteSyncPlanAppliesActions(), TestExecuteSyncPlanCloneAction(), TestExecuteSyncPlanStopsOnFailure(), TestExecuteSyncPlanStopsOnNonDryRunFailure(), TestExecuteSyncPlanWithCallbackInvokesPerResult(), TestHandleMissingSyncEntry(), TestHandleMissingSyncEntrySkipsNoUpstreamBranch() (+14 more)

### Community 29 - "Entry"
Cohesion: 0.13
Nodes (18): persistDescribeMetadataSnapshot(), editRegistryEntryWithEditor(), findRegistryEntryIndex(), resolveEditorCommand(), trackingBranchFromUpstream(), resolveUpstreamTargetBranch(), findRegistryEntryForSyncResult(), canonicalRegistryPath() (+10 more)

### Community 30 - "Tasks: [FEATURE NAME]"
Cohesion: 0.07
Nodes (26): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation Strategy, Incremental Delivery, MVP First (User Story 1 Only) (+18 more)

### Community 31 - "Milestones"
Cohesion: 0.07
Nodes (27): CI pipeline (GitHub Actions), Coverage requirements, Integration tests, Linting & code quality, Milestone 0 — Repo skeleton, Milestone 10 — 1.0 Readiness & Release Reset, Milestone 11 — MCP Server for Agent-Native Querying, Milestone 1 — Discovery + registry (+19 more)

### Community 32 - "Tasks: Recipe-Driven End-to-End Test Harness"
Cohesion: 0.06
Nodes (31): Compatibility Contract Specs, Declaration and Reusable Compatibility Interface, Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Foundation, Foundational Implementation, Foundational Validation Specs, Implementation Strategy (+23 more)

### Community 33 - "import_test.go"
Cohesion: 0.11
Nodes (26): cloneImportedEntriesWithProgress(), cloneImportedRepos(), cloneImportedReposWithProgress(), normalizeImportedBundle(), TestCloneImportedReposMarksFailedCloneAsMissing(), TestCloneImportedReposNoopWithoutRegistry(), TestCloneImportedReposRejectsDuplicateTargets(), TestCloneImportedReposRejectsUnsafeTargets() (+18 more)

### Community 34 - "repometa_test.go"
Cohesion: 0.17
Nodes (24): TestLoadInvalidYAMLErrors(), Apply(), mustMarshalRepoMetadata(), mustMetadataFingerprint(), rewriteFilePreservingFingerprint(), testAbsolutePath(), TestApplyCachesDualFileConflict(), TestApplyCachesInvalidMetadataError() (+16 more)

### Community 35 - "Config"
Cohesion: 0.10
Nodes (34): loadExistingConfig(), BranchPolicy, Defaults, configDir(), ConfigPath(), ConfigRoot(), EffectiveRoot(), findNearestConfigPath() (+26 more)

### Community 36 - "registry_test.go"
Cohesion: 0.12
Nodes (16): Load(), TestFindByRepoIDAndCheckoutIDBackfillsLegacyEntries(), TestFindEntriesByRepoID(), TestFindEntriesByRepoIDReturnsAllCheckoutMatches(), TestLegacyEntryBackfillsCheckoutID(), TestLegacySameBasenameCheckoutsKeepTheirMetadata(), TestLookupsDoNotMutateEntries(), TestUpsertAllowsDuplicateRepoIDWithDistinctCheckoutID() (+8 more)

### Community 37 - "import_clone.go"
Cohesion: 0.23
Nodes (12): ImportCloneCallbacks, importCloneConflict, ImportCloneOptions, ImportCloneSkip, ImportCloneTarget, TestImportCloneHelperFunctions(), findImportCloneConflicts(), Engine (+4 more)

### Community 38 - "materializeRecipe"
Cohesion: 0.33
Nodes (10): MaterializedMissingEntry, MaterializedRepository, MaterializedWorkspace, cloneMap(), commitAll(), fileURL(), gitOutput(), materializeRecipe() (+2 more)

### Community 39 - "ADR-0002: Branch Switch and Checkout Workflow Boundaries"
Cohesion: 0.09
Nodes (22): 1. Fold checkout into sync, 2. Keep checkout permanently out of scope, 3. Allow branch switching through MCP, ADR-0002: Branch Switch and Checkout Workflow Boundaries, Alternatives Considered, Branch switch / checkout, CLI, Consequences (+14 more)

### Community 40 - "sync.go"
Cohesion: 0.11
Nodes (24): TestRepairUpstreamMatchesFilterTable(), TestShouldStreamSyncResults(), TestSyncResultNeedsConfirmationTable(), TestShouldStreamSyncResultsBranches(), TestDescribeSyncAction(), TestDescribeSyncActionAdditionalBranches(), TestSyncPlanNeedsConfirmation(), TestWriteSyncFailureSummary() (+16 more)

### Community 41 - "RepoStatus"
Cohesion: 0.09
Nodes (34): syncPlanRecord, localUpdateCapable, PullRebasePolicyOptions, filterRegistryEntriesByIgnoredPaths(), filterRequiresInspect(), filterStatus(), findRegistryEntryForStatus(), FilterKind (+26 more)

### Community 42 - "Tasks: Distribution Channel Conformance"
Cohesion: 0.06
Nodes (33): Correction made during implementation, Dependencies & Execution Order, Documentation, Execution Notes, Format: `[ID] [P?] [Story] Description`, Format validation, Highest-risk tasks, Implementation for User Story 1 (+25 more)

### Community 43 - "Registry"
Cohesion: 0.15
Nodes (18): inferredCheckoutIDFromPath(), mergeImportedRegistry(), mergePolicyPreflightSkips(), mergeRegistryMatchIndex(), parseImportConflictPolicy(), parseImportMode(), prepareImportedConfig(), registryEntriesConflict() (+10 more)

### Community 44 - "repometa.go"
Cohesion: 0.20
Nodes (21): RepoMetadata, canonicalize(), discoverMetadataState(), discoverPath(), fileExists(), Load(), metadataConflictFingerprint(), metadataFileFingerprint() (+13 more)

### Community 45 - "perf/main.go"
Cohesion: 0.21
Nodes (16): benchmarkMetric, benchmarkRunRecord, appendRecord(), gitShortCommit(), loadLastRecord(), main(), parseBenchmarkMetrics(), printSummary() (+8 more)

### Community 46 - "README.md"
Cohesion: 0.16
Nodes (7): Manual installation, RepoKeeper Agent Skill, Git compatibility command, Recipe-driven end-to-end tests, Crediting libraries correctly, Regenerating notices, Third-Party Notices

### Community 47 - "ExecutionResult"
Cohesion: 0.15
Nodes (16): boundedBuffer, ExecutionResult, scanJSONEnvelope, scanJSONResponse, statusJSONRepository, statusJSONResponse, bytes.Buffer, time.Duration (+8 more)

### Community 48 - "RepoKeeper"
Cohesion: 0.09
Nodes (23): Commands, Configuration, Container (MCP server), Development, Documentation, Expected User Flow, Features, From release binaries (+15 more)

### Community 49 - "5.1 CLI commands (v1)"
Cohesion: 0.11
Nodes (18): 5.1 CLI commands (v1), Exit codes, Global flags (apply to all commands), `repokeeper add <path> <git-repo-url>`, `repokeeper delete <selector>`, `repokeeper describe <selector>`, `repokeeper edit <selector>`, `repokeeper export` (+10 more)

### Community 50 - "ADR-0001: MCP Server for Agent-Native Repository Querying and Planning"
Cohesion: 0.11
Nodes (18): 1. Keep MCP as a mixed read/write surface, 2. Remove MCP entirely and rely on CLI JSON only, 3. Allow execution only for "safe" mutations, ADR-0001: MCP Server for Agent-Native Repository Querying and Planning, Alternatives Considered, Architecture, Consequences, Context (+10 more)

### Community 51 - "writeRepairUpstreamTable"
Cohesion: 0.17
Nodes (12): needsUpstreamRepair(), parseUpstreamRepairFilter(), writeRepairUpstreamTable(), adaptiveCellLimit(), adaptiveCellLimitForWidth(), tableWidth(), github.com/liggitt/tabwriter.Writer, New() (+4 more)

### Community 52 - "addFormatFlag"
Cohesion: 0.29
Nodes (13): init(), addFormatFlag(), addLabelSelectorFlag(), addNoHeadersFlag(), addRepoFilterFlags(), addUpstreamRepairFilterFlag(), addVCSFlag(), init() (+5 more)

### Community 53 - "index_test.go"
Cohesion: 0.23
Nodes (13): formatAssignmentDefaults(), TestFormatAssignmentDefaultsSortsKeys(), TestUnifiedDiffEmptyWhenIdentical(), TestUnifiedDiffReportsChanges(), TestWriteMetadataPreviewDiffsUnparseableExisting(), TestWriteMetadataPreviewNewFile(), TestWriteMetadataPreviewShowsDiff(), TestWriteMetadataPreviewUnchanged() (+5 more)

### Community 54 - "MCPToolCase"
Cohesion: 0.16
Nodes (17): MCPToolCase, addedRepositoryPath(), mutationMCPToolCases(), emptyMCPArguments(), readMCPToolCases(), requireFields(), requireList(), safetyMCPToolCases() (+9 more)

### Community 55 - "ADR-0006: Adapter-Facing Contract Stability and Versioning"
Cohesion: 0.12
Nodes (16): 1. Treat CLI JSON as best-effort only, 2. Require adapters to import internal packages, 3. Use MCP as the only stable contract, ADR-0006: Adapter-Facing Contract Stability and Versioning, Alternatives Considered, Breaking changes must be explicit, Consequences, Context (+8 more)

### Community 56 - "RepoKeeper"
Cohesion: 0.11
Nodes (17): Avoid these mistakes, Check health first, Core rules, Discovery workflow, Execute safe updates, Good agent response pattern, Health errors and repository selectors, Initialization workflow (+9 more)

### Community 57 - "Manpage Plan"
Cohesion: 0.33
Nodes (5): Acceptance Criteria, Generation Approach, Manpage Plan, Release/CI Integration, Target

### Community 58 - "github.com/mark3labs/mcp-go/mcp.CallToolResult"
Cohesion: 0.18
Nodes (15): github.com/mark3labs/mcp-go/mcp.CallToolResult, github.com/mark3labs/mcp-go/mcp.JSONRPCMessage, expectResourceError(), expectResourceSuccess(), intPtr(), newTestConfig(), newTestRegistry(), newTestStatusReport() (+7 more)

### Community 59 - "discovery.go"
Cohesion: 0.19
Nodes (18): Options, Result, buildResult(), detectRepo(), gitdirFromFile(), TestBuildResultBranches(), TestDetectRepoBranches(), TestGitdirFromFile() (+10 more)

### Community 60 - "Changelog"
Cohesion: 0.11
Nodes (17): [0.6.1](https://github.com/skaphos/repokeeper/compare/v0.6.0...v0.6.1) (2026-04-03), [0.7.0](https://github.com/skaphos/repokeeper/compare/v0.6.1...v0.7.0) (2026-04-09), [0.7.1](https://github.com/skaphos/repokeeper/compare/v0.7.0...v0.7.1) (2026-04-18), [1.2.0](https://github.com/skaphos/repokeeper/compare/v1.1.0...v1.2.0) (2026-05-31), [1.3.0](https://github.com/skaphos/repokeeper/compare/v1.2.0...v1.3.0) (2026-06-22), [1.3.1](https://github.com/skaphos/repokeeper/compare/v1.3.0...v1.3.1) (2026-07-12), [1.4.0](https://github.com/skaphos/repokeeper/compare/v1.3.1...v1.4.0) (2026-07-26), Bug Fixes (+9 more)

### Community 61 - "planAdapter"
Cohesion: 0.07
Nodes (4): dirtyBehindAdapter, planAdapter, unsupportedLocalUpdateAdapter, Tracking

### Community 62 - "newPlanExecEngine"
Cohesion: 0.24
Nodes (12): syncStep, Engine, Engine, SyncResult, newPlanExecEngine(), TestApplyRemoteMismatchPlansUsesInjectedAdapter(), TestExecutePlannedNonCloneUnknownStepFailsInvalid(), TestExecutePlannedSyncItemEmptyStepsFailsInvalid() (+4 more)

### Community 63 - "9. Future: Cross-Machine Registry Sync"
Cohesion: 0.50
Nodes (4): 9.1 Architecture considerations (factor in now), 9.2 Planned sync mechanisms (future, not v1), 9.3 Reconciliation (future), 9. Future: Cross-Machine Registry Sync

### Community 64 - "helpers_test.go"
Cohesion: 0.08
Nodes (33): TestColorizeAndTrackingDisplayBranches(), TestColorizeGuardBranches(), TestConfirmSyncExecution(), TestConfirmSyncExecutionEOF(), TestConfirmWithPrompt(), TestDisplayRepoPathPrefersCWDThenRoot(), TestFormatCellWrapControl(), TestStatusExitCode() (+25 more)

### Community 65 - "writeEmptyConfig"
Cohesion: 0.13
Nodes (29): TestAddCommandWithAbsoluteTargetDoesNotReRootUnderCWD(), mustRunGit(), TestAddDeleteWithRegistryOverride(), TestAddValidationMutuallyExclusiveFlags(), TestDeleteCancelledByPrompt(), TestDeleteTrackingOnlyAddsIgnoredPathAndKeepsRepo(), TestInitCommandForceBehavior(), TestScanJSONOutputAndUnsupportedFormat() (+21 more)

### Community 66 - "ADR-0003: Sync Policy and Execution Modes"
Cohesion: 0.13
Nodes (14): ADR-0003: Sync Policy and Execution Modes, CLI and TUI, Configuration Boundary, Consequences, Context, Decision, Execution Model, Interface Boundaries (+6 more)

### Community 67 - "ADR-0004: Prune Workflow Boundaries and Safety Model"
Cohesion: 0.13
Nodes (15): 1. Keep prune as an implicit part of sync, 2. Treat local branch prune as a simple merged/not-merged check, ADR-0004: Prune Workflow Boundaries and Safety Model, Alternatives Considered, CLI and TUI, Consequences, Context, Decision (+7 more)

### Community 68 - "Specification Quality Checklist: GitHub Remote End-to-End Expansion"
Cohesion: 0.10
Nodes (18): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: GitHub Remote End-to-End Expansion, Assumptions, Edge Cases, Feature Specification: GitHub Remote End-to-End Expansion (+10 more)

### Community 69 - "WriteTable"
Cohesion: 0.14
Nodes (15): errorWriter, TestHasRegistryWarnings(), TestWriters(), TestWriteScanTableNoHeaders(), hasRegistryWarnings(), writeScanTable(), writeSyncPlan(), PromptYesNo() (+7 more)

### Community 70 - "ADR-0007: Release Binary Publishing and Homebrew Distribution"
Cohesion: 0.14
Nodes (14): 1. Goreleaser owns the GitHub release and release body; release-please bumps version; the `release-please.yml` workflow pushes the tag, 2. Keep `homebrew_casks:`, but make it install cleanly on Apple Silicon, 3. Backfill v0.7.0 assets (one-off), ADR-0007: Release Binary Publishing and Homebrew Distribution, Consequences, Context, Decision, Implementation plan (+6 more)

### Community 71 - "NewGitErrorClassifier"
Cohesion: 0.18
Nodes (15): executeImportClonePlanWithProgress(), planImportedEntries(), TestActionsResetDeleteCloneAndRegister(), TestExecuteImportClonesSuccessFailureAndSkips(), TestPlanImportClonesGuardsAndErrors(), TestPlanImportClonesSkipsAndSuccess(), TestRepairUpstreamScenarios(), TestExecuteSyncPlanStopsOnFailureWhenConfigured() (+7 more)

### Community 72 - "opencodeAdapter"
Cohesion: 0.24
Nodes (7): checkJsonc(), Entry, init(), opencodeDir(), opencodeServersMap(), opencodeAdapter, opencodeServer

### Community 73 - "Feature Specification: Distribution Channel Conformance"
Cohesion: 0.09
Nodes (22): Assumptions, Clarifications, Constitution Alignment, Dependencies, Edge Cases, Feature Specification: Distribution Channel Conformance, Functional Requirements, Key Entities (+14 more)

### Community 74 - "ADR-0005: Workspace Config vs Repo-Local Metadata Ownership"
Cohesion: 0.15
Nodes (13): 1. Use repo-local metadata as the home for most policy, 2. Keep repo-local metadata narrowly informational with no agent/runtime fields, ADR-0005: Workspace Config vs Repo-Local Metadata Ownership, Alternatives Considered, Consequences, Context, Decision, Negative (+5 more)

### Community 75 - "buildIndexProposal"
Cohesion: 0.23
Nodes (11): buildIndexProposal(), formatRelatedRepoDefaults(), newIndexQuestioner(), parseRelatedRepos(), TestFormatRelatedRepoDefaultsSortsValues(), TestIndexQuestionerUsesDefaultsAndParsers(), bufio.Reader, io.Reader (+3 more)

### Community 76 - "ADR-0011: Credential and Auth Handling Deferred"
Cohesion: 0.15
Nodes (12): 1. Add a thin credential helper wrapper, 2. Document specific recommended setups in this ADR, 3. Build a `repokeeper auth doctor` command, 4. Encrypt the registry at rest, ADR-0011: Credential and Auth Handling Deferred, Alternatives Considered, Consequences, Context (+4 more)

### Community 77 - "Verification Checklist (with Evidence Placeholders)"
Cohesion: 0.15
Nodes (12): 1. Tool Discovery, 2. Read-only Tools, 3. Planning Tools, 4. Mutation Tools + Safety Gates, 5. Structured Content & Error Quality, 6. Overall Claude Experience, How to Run the Verification (for the user), MCP Manual Verification Results (SKA-201) (+4 more)

### Community 78 - "SplitCSV"
Cohesion: 0.32
Nodes (5): TestSplitCSV(), ParseFieldSelectorFilter(), ResolveRepoFilter(), SplitCSV(), TestSplitCSV()

### Community 79 - "Phase 0 Research: Recipe-Driven End-to-End Test Harness"
Cohesion: 0.11
Nodes (18): Decision 10: Use separate CLI and MCP materializations and deterministic MCP order, Decision 11: Compare semantic normalized outcomes five times, Decision 12: Reuse existing modules and add no ADR, Decision 13: Run the E2E boundary in four supported environments, Decision 14: Separate representative CI from exhaustive release qualification, Decision 15: Centralize matrix, provisioning, and evidence behavior in a tagged command, Decision 16: Provision immutable Git inputs per environment, Decision 17: Make race coverage and release-candidate recovery explicit (+10 more)

### Community 80 - "Feature Specification: [FEATURE NAME]"
Cohesion: 0.15
Nodes (12): Assumptions, Edge Cases, Feature Specification: [FEATURE NAME], Functional Requirements, Key Entities *(include if feature involves data)*, Measurable Outcomes, Requirements *(mandatory)*, Success Criteria *(mandatory)* (+4 more)

### Community 81 - "cloneMetadataMap"
Cohesion: 0.23
Nodes (9): validateEditedRegistryEntry(), mergePromotedLabels(), cloneMetadataMap(), normalizeMetadataMap(), parseMetadataAssignments(), parseMetadataKeys(), TestParseMetadataAssignments(), TestParseMetadataKeys() (+1 more)

### Community 82 - "HgAdapter"
Cohesion: 0.11
Nodes (3): rejectFlagLike(), runCommand(), HgAdapter

### Community 83 - "RepoKeeper — Design Spec"
Cohesion: 0.20
Nodes (10): 10. Open Questions (explicitly deferred), 1. Summary, 2. Problem Statement, 3.1 Functional goals (v1 / "80%"), 3.2 Non-goals (defer), 3. Goals, 7.0 Git CLI Strategy & Compatibility Matrix, 7. Git Operations (Engine Contract) (+2 more)

### Community 84 - "ADR-0010: Repo ID Normalization Stability"
Cohesion: 0.17
Nodes (11): 1. Treat normalization as implementation detail, 2. Version every normalization rule from day one, 3. Hash the normalized URL instead of storing it, ADR-0010: Repo ID Normalization Stability, Alternatives Considered, Consequences, Context, Decision (+3 more)

### Community 85 - "ADR-0014: Local Branch Prune-Safety Classification Model"
Cohesion: 0.17
Nodes (12): 1. Treat "upstream gone" as `probably_safe` without patch-equivalence, 2. Use `git branch --merged` (reachability) as the sole integration signal, 3. Put the enums in a dependency-free `internal/prune` package, 4. Classify only the current branch, reusing existing state, ADR-0014: Local Branch Prune-Safety Classification Model, Alternatives Considered, Consequences, Context (+4 more)

### Community 86 - "ADR-0015: Branch Retention and Protection Policy"
Cohesion: 0.17
Nodes (12): 1. Workspace-global `base_branch` (empty ⇒ `Defaults.MainBranch`), 2. Reuse `--protected-branches` for prune and union it in, 3. Store branch policy in `.repokeeper-repo.yaml`, 4. Define the policy type inside the classifier and have config embed it, ADR-0015: Branch Retention and Protection Policy, Alternatives Considered, Consequences, Context (+4 more)

### Community 87 - ".claude/skills/speckit-analyze/SKILL.md"
Cohesion: 0.08
Nodes (25): 1. Initialize Analysis Context, 2. Load Artifacts (Progressive Disclosure), 3. Build Semantic Models, 4. Detection Passes (Token-Efficient Analysis), 5. Severity Assignment, 6. Produce Compact Analysis Report, 7. Provide Next Actions, 8. Offer Remediation (+17 more)

### Community 88 - "LocalBranches"
Cohesion: 0.21
Nodes (13): LocalBranchInfo, stubRunner, LocalBranches(), MergedBranches(), ParseCherryEquivalent(), ParseLocalBranches(), PatchEquivalentToBase(), TestLocalBranchesRunner() (+5 more)

### Community 89 - "Decision"
Cohesion: 0.09
Nodes (23): 10. The contract version stays independent of the config and repo-metadata schemas, 1. Every adapter-facing response is wrapped in a versioned envelope, 1. Out-of-band version discovery, 2. Per-surface mechanisms, 2. The version is promoted from `v1beta1` to `v1`, 3. CLI and MCP are enveloped in the same change, 3. Inline `apiVersion` into existing objects rather than nesting a named payload, 4. One shared constant (+15 more)

### Community 90 - "RepoKeeper Adapter Contract"
Cohesion: 0.12
Nodes (17): 10. CLI or MCP?, 11. Declaring which core versions you support, 1. Check compatibility from the first response, 2. Ignore fields you do not recognise, 3. The envelope, 4. Surface inventory, 5. Know which calls are safe to make unprompted, 6. URLs are redacted — do not clone from them (+9 more)

### Community 91 - "GitHub Copilot Instructions for RepoKeeper"
Cohesion: 0.18
Nodes (10): Codebase Shape, Commit and Branch Guidance, Documentation Expectations, GitHub Copilot Instructions for RepoKeeper, Go and Repository Conventions, Pull Request Instructions, Safety Rules, Testing Expectations (+2 more)

### Community 92 - "SyncResult"
Cohesion: 0.20
Nodes (13): OutcomeKind, executedNonCloneOutcome(), Engine, SyncOptions, SyncResult, SyncResultCallback, SyncStartCallback, outcomeForRebase() (+5 more)

### Community 93 - "StatusReport"
Cohesion: 0.17
Nodes (8): enrichReportWithRegistryMetadata(), findRegistryMetadataEntry(), time.Time, ScanOptions, StatusOptions, StatusReport, mockEngine, SyncResult

### Community 94 - "Core Principles"
Cohesion: 0.11
Nodes (17): Core Principles, Engineering Constraints, Governance, I. Explicit State Over Implicit Behavior, II. Git Is the Durable Desired-State Boundary, III. Deterministic, Reconstructible Operation, IV. Kubernetes-Native, Never Obscured, IX. Technical Precision, Honest Scope (+9 more)

### Community 95 - "Core Principles"
Cohesion: 0.18
Nodes (10): Core Principles, Governance, [PRINCIPLE_1_NAME], [PRINCIPLE_2_NAME], [PRINCIPLE_3_NAME], [PRINCIPLE_4_NAME], [PRINCIPLE_5_NAME], [PROJECT_NAME] Constitution (+2 more)

### Community 96 - "Repository Guidelines"
Cohesion: 0.18
Nodes (10): Build, Test, and Development Commands, Coding Style & Naming Conventions, Commit & Pull Request Guidelines, Configuration & Safety Notes, Engineering Guardrails, graphify, Project Structure & Module Organization, Repository Docs & Agent Notes (+2 more)

### Community 97 - "install.go"
Cohesion: 0.24
Nodes (9): desiredInstallEntry(), runInstallList(), writeInstallListTable(), parseInstallScope(), printClaudePermissionsBlock(), printManualSnippets(), runInstall(), Entry (+1 more)

### Community 98 - "5.3 Kubectl-Style CLI Alignment (Milestone 6+)"
Cohesion: 0.22
Nodes (9): 5.2 TUI command (withdrawn), 5.3.1 Command shape, 5.3.2 Output contracts, 5.3.3 Styling and color policy (intentional delta vs kubectl), 5.3.4 Filter and selector direction, 5.3.5 Migration strategy, 5.3 Kubectl-Style CLI Alignment (Milestone 6+), 5. User Experience (+1 more)

### Community 99 - "Command Notes"
Cohesion: 0.13
Nodes (15): Command Notes, Global Flags, `repokeeper add`, RepoKeeper Commands, `repokeeper describe`, `repokeeper edit`, `repokeeper get`, `repokeeper index` (+7 more)

### Community 100 - ".agents/skills/speckit-plan/SKILL.md"
Cohesion: 0.18
Nodes (10): Completion Report, Done When, Key rules, Mandatory Post-Execution Hooks, Outline, Phase 0: Outline & Research, Phase 1: Design & Contracts, Phases (+2 more)

### Community 101 - "hintForErrorClass"
Cohesion: 0.28
Nodes (6): Engine, SyncResult, hintForErrorClass(), TestHintForErrorClass_Deterministic(), TestHintForErrorClass_KnownClasses(), TestHintForErrorClass_UnknownClasses()

### Community 102 - ".handleGetWorkspaceConfig"
Cohesion: 0.36
Nodes (6): cfgDefault(), MCPServer, intDefault(), positiveIntDefault(), configDefaults, workspaceConfigResponse

### Community 103 - "Release Process"
Cohesion: 0.15
Nodes (13): 1. Land releasable commits on `main`, 2. Run local release checks, 3. Review and merge the release PR, 4. Tag push + GitHub release automation, 5. Verify the release, Channel verification, Compatibility qualification boundary, Credentials fail the release; they never skip a channel (+5 more)

### Community 104 - "repair_upstream_test.go"
Cohesion: 0.25
Nodes (7): TestNeedsUpstreamRepair(), TestParseUpstreamRepairFilter(), TestRepairUpstreamMatchesFilter(), TestResolveUpstreamTargetBranch(), TestWriteRepairUpstreamTable(), TestWriteRepairUpstreamTableCompactsColumnsOnTinyTTY(), TestWriteRepairUpstreamTableNoHeaders()

### Community 105 - ".agents/skills/speckit-specify/SKILL.md"
Cohesion: 0.18
Nodes (10): Completion Report, Done When, For AI Generation, Mandatory Post-Execution Hooks, Outline, Pre-Execution Checks, Quick Guidelines, Section Requirements (+2 more)

### Community 106 - "Decision"
Cohesion: 0.12
Nodes (16): Action pin, ADR-0009: Replace Release Please with skaphos/actions, Alternatives considered, Bot identity, Consequences, Context, Decision, Files added (+8 more)

### Community 107 - "repairResolveTargetBranch"
Cohesion: 0.43
Nodes (5): RepairUpstreamResult, TestRepairHelpers(), Engine, repairNeedsUpstream(), repairResolveTargetBranch()

### Community 108 - "Scope"
Cohesion: 0.22
Nodes (4): Scope, Entry, TestScopeString(), fakeRuntime

### Community 109 - "Contributing Guidelines"
Cohesion: 0.20
Nodes (9): Branching and Commits, Coding Standards, Contributing Guidelines, Development Setup, Graphify, Pull Requests, Release Process, Safety Expectations (+1 more)

### Community 110 - "Quickstart: Building an Adapter Against the RepoKeeper Contract"
Cohesion: 0.12
Nodes (14): Consequence for this feature, Contract: CLI / MCP Parity, The rule, Verification, Why this is a contract and not an implementation detail, 1. Check compatibility from the first response, 2. Ignore fields you do not recognise, 3. Know which calls are safe to make unprompted (+6 more)

### Community 111 - "Implementation Plan: [FEATURE]"
Cohesion: 0.22
Nodes (8): Complexity Tracking, Constitution Check, Documentation (this feature), Implementation Plan: [FEATURE], Project Structure, Source Code (repository root), Summary, Technical Context

### Community 112 - "6. Data Model"
Cohesion: 0.25
Nodes (8): 6.1 Repo identity, 6.2.1 Machine config, 6.2.2 Registry (embedded in machine config by default), 6.2 Config files, 6.3 Status JSON schema (v1), 6.4 Sync (reconcile) JSON schema, 6. Data Model, JSON output schema stability policy

### Community 113 - "Execution Steps"
Cohesion: 0.12
Nodes (15): 1. Initialize Convergence Context, 2. Load Artifacts (Progressive Disclosure), 3. Build the Intent Inventory, 4. Assess the Codebase and Classify Findings, 5. Assign Severity, 6. Present the In-Session Findings Summary, 7. Append Convergence Tasks (or report converged), 8. Provide Next Actions (Handoff) (+7 more)

### Community 114 - "SortRegistryEntries"
Cohesion: 0.39
Nodes (6): LessRepoIDPath(), SortRegistryEntries(), SortRepoStatuses(), TestLessRepoIDPath(), TestSortRegistryEntries(), TestSortRepoStatuses()

### Community 115 - "MCP Server Setup"
Cohesion: 0.07
Nodes (29): 1. Tool Discovery (all 14 tools visible), 2. Read-only Tools (safe, no side effects), 3. Planning Tools (dry-run only), 4. Mutation Tools + Safety Gates, 5. Structured Content + Error Handling, 6. Client Compatibility Smoke Test, Available tools, CLI skill fallback (+21 more)

### Community 116 - "mcpSession"
Cohesion: 0.17
Nodes (11): mcpSession, context.CancelFunc, github.com/mark3labs/mcp-go/client.Client, io.WriteCloser, sync.Once, newInitializedClient(), mcpProcessExitError(), startMCPSession() (+3 more)

### Community 117 - "newToolError"
Cohesion: 0.23
Nodes (14): explainReadOnly(), isReadOnly(), mentionsReadOnlyFilesystem(), resultErrorText(), TestExplainReadOnlyDoesNotMisdiagnoseOtherGitFailures(), TestExplainReadOnlyNamesCauseAndRemedy(), TestExplainReadOnlyPassesThroughUnrelatedErrors(), TestExplainReadOnlyRecognisesGitSubprocessFailures() (+6 more)

### Community 118 - "dropIgnoredImportEntries"
Cohesion: 0.16
Nodes (17): dropIgnoredImportEntries(), ignoredPathSet(), TestDropIgnoredImportEntriesCaseInsensitiveOnWindows(), TestDropIgnoredImportEntriesRemovesIgnoredTargets(), pathCleanCanonical(), CanonicalNormalize(), CleanNormalize(), IgnoredPathSet() (+9 more)

### Community 119 - "Phase 1 Data Model: Recipe-Driven End-to-End Test Harness"
Cohesion: 0.14
Nodes (13): BranchRecipe and CommitRecipe, CompatibilityCommandResult, CompatibilityResult, ConfigRecipe, ExecutionResult, ExpectedOutcome and NormalizedOutcome, GitCompatibilityMatrix and CompatibilityCell, MaterializedWorkspace (+5 more)

### Community 120 - "readJSONDoc"
Cohesion: 0.23
Nodes (6): Entry, init(), readJSONDoc(), writeJSONDoc(), claudeAdapter, claudeServer

### Community 121 - ".agents/skills/speckit-tasks/SKILL.md"
Cohesion: 0.18
Nodes (10): Checklist Format (REQUIRED), Completion Report, Done When, Mandatory Post-Execution Hooks, Outline, Phase Structure, Pre-Execution Checks, Task Generation Rules (+2 more)

### Community 122 - "os/exec.Cmd"
Cohesion: 0.32
Nodes (10): os/exec.Cmd, attachProcessTree(), configureProcessTree(), forceTerminateProcessTree(), releaseProcessTree(), attachProcessTree(), closeProcessJob(), configureProcessTree() (+2 more)

### Community 123 - "9. Stretch Goals"
Cohesion: 0.29
Nodes (7): 7.1 Detection commands, 7.2 Tracking status (preferred), 7.3 Sync operation (core action), 7.4 Error handling guidelines, 9.1 Multi-VCS Support (Hg), 9. Stretch Goals, Bare repo handling

### Community 124 - "Phase 0 Research: Distribution Channel Conformance"
Cohesion: 0.14
Nodes (14): Phase 0 Research: Distribution Channel Conformance, R10 — Mercurial in the container, R11 — ADR-0016 and the deviation record, R12 — Zero dependency growth, R1 — Version identity: three-tier resolution, R2 — Container base image ⚠ DIVERGENCE, R3 — Git "dubious ownership" in the container ⚠ DIVERGENCE, measured, R4 — Workspace discovery: cwd-relative crawl vs. a frozen container ⚠ DIVERGENCE (+6 more)

### Community 125 - "grokAdapter"
Cohesion: 0.32
Nodes (4): grokDir(), init(), grokAdapter, grokServer

### Community 126 - "ADR-0013: GoReleaser Owns the GitHub Release"
Cohesion: 0.29
Nodes (7): ADR-0013: GoReleaser Owns the GitHub Release, Alternatives Considered, Consequences, Context, Decision, Negative / risks, Positive

### Community 127 - "Decision"
Cohesion: 0.29
Nodes (7): Boundary, Decision, Defaults must survive the zero-value backfill idiom, Per-repo base resolution, Protection is prune-scoped and independent of rebase protection, Schema, Validation fails closed

### Community 128 - "ADR-0016: No Self-Update Subcommand"
Cohesion: 0.15
Nodes (13): ADR-0016: No Self-Update Subcommand, Alternatives Considered, Consequences, Context, Decision, Links, Negative, Positive (+5 more)

### Community 129 - "benchmarkEngineWithRepos"
Cohesion: 0.53
Nodes (5): testing.B, benchmarkEngineWithRepos(), BenchmarkStatusReport(), BenchmarkSyncDryRunPlan(), Engine

### Community 130 - "recipe_test.go"
Cohesion: 0.22
Nodes (17): BranchRecipe, CommitRecipe, MetadataRecipe, RelationshipRecipe, RepositoryRecipe, UpstreamRecipe, containedPath(), hasDriveOrUNCPath() (+9 more)

### Community 131 - "Execution Contract"
Cohesion: 0.20
Nodes (9): CLI Process Boundary, CLI Scenario Contract, Determinism Contract, Exact Environment, Execution Contract, Full-release qualification, Git Compatibility Boundary, MCP Stdio Boundary (+1 more)

### Community 132 - "sync.Mutex"
Cohesion: 0.25
Nodes (6): blockingRunner, countingRunner, mockResponse, mockRunner, sync.Mutex, joinArgs()

### Community 133 - "Decision"
Cohesion: 0.33
Nodes (6): Categories and decision precedence, Decision, Enum placement (no import cycle), Integration evidence, Per-branch model, Surfacing

### Community 134 - "newEngineWith"
Cohesion: 0.33
Nodes (9): plainAdapter, Engine, newEngineWith(), TestInspectLocalBranchesBaseUnresolved(), TestInspectLocalBranchesClassifies(), TestInspectLocalBranchesPatchEquivalenceGatedByRequireMerged(), TestInspectLocalBranchesRemoteQualifiedBaseOverride(), TestInspectLocalBranchesUnsupportedAndBareAndError() (+1 more)

### Community 135 - "Installation"
Cohesion: 0.25
Nodes (8): Container image, From local source checkout, From release binaries, From source, Homebrew (cask), Installation, Linux packages, Migration from old Homebrew formula install

### Community 136 - "ClassifyError"
Cohesion: 0.33
Nodes (4): ClassifyError(), containsAny(), TestClassifyError(), gitErrorClassifier

### Community 137 - "Quickstart: Recipe-Driven End-to-End Harness"
Cohesion: 0.20
Nodes (9): Add a Scenario, Diagnose a Failure, Prerequisites, Quickstart: Recipe-Driven End-to-End Harness, Release Qualification, Repeatability Qualification, Run the E2E Package, Scope Reminder (+1 more)

### Community 138 - "Contract: `ghcr.io/skaphos/repokeeper`"
Cohesion: 0.17
Nodes (12): Behavioral contract, Contract: `ghcr.io/skaphos/repokeeper`, Divergences from the sting reference, Image properties, Invocation contract, Multiple workspace roots, OCI labels, Read-only inspection — the documented default (+4 more)

### Community 139 - "4. Safety & Policy"
Cohesion: 0.40
Nodes (5): 4.1 Submodules: do not update, 4.2 Working tree safety, 4.3 MCP boundary, 4.4 Mutating command safety, 4. Safety & Policy

### Community 140 - "8. Architecture"
Cohesion: 0.40
Nodes (5): 8.1 Packages (Go), 8.2 CLI framework, 8.3 Concurrency model, 8.4 TUI model (withdrawn), 8. Architecture

### Community 141 - "NormalizeURL"
Cohesion: 0.25
Nodes (4): NormalizeURL(), PrimaryRemote(), TestGitURLNormalizerMatchesGitx(), gitURLNormalizer

### Community 142 - "Phase 1 Data Model: Distribution Channel Conformance"
Cohesion: 0.17
Nodes (11): 1. Version Identity, 2. Release Artifact Set, 3. Server Description, 4. Container Workspace Contract, `Info`, Phase 1 Data Model: Distribution Channel Conformance, Resolution rules — FR-001 through FR-005, `Source` (+3 more)

### Community 144 - "Feature Specification: Stable Plugin Adapter Contract"
Cohesion: 0.12
Nodes (16): Assumptions, Clarifications, Context and Prior Art, Edge Cases, Feature Specification: Stable Plugin Adapter Contract, Functional Requirements, Key Entities, Measurable Outcomes (+8 more)

### Community 145 - "declaration.go"
Cohesion: 0.07
Nodes (49): CompatibilityResult, EvidenceSummary, MatrixCell, MatrixResult, PinnedInput, Provisioner, ProvisionResult, RootFS (+41 more)

### Community 146 - "[CHECKLIST TYPE] Checklist: [FEATURE NAME]"
Cohesion: 0.40
Nodes (4): [Category 1], [Category 2], [CHECKLIST TYPE] Checklist: [FEATURE NAME], Notes

### Community 147 - "Alternatives Considered"
Cohesion: 0.50
Nodes (4): 1. Keep sync permanently fetch/prune only, 2. Treat configured `update_local` as silent execution permission, 3. Store sync policy in repo-local metadata, Alternatives Considered

### Community 149 - "WriteAtomic"
Cohesion: 0.33
Nodes (8): io/fs.FileMode, resolveTarget(), TestWriteAtomicCreatesFile(), TestWriteAtomicFollowsSymlink(), TestWriteAtomicNoFileAtRequestedMode(), TestWriteAtomicOverwritesExisting(), TestWriteAtomicPreservesExistingMode(), WriteAtomic()

### Community 150 - ".agents/skills/speckit-checklist/SKILL.md"
Cohesion: 0.25
Nodes (7): Anti-Examples: What NOT To Do, Checklist Purpose: "Unit Tests for English", Example Checklist Types & Sample Items, Execution Steps, Post-Execution Checks, Pre-Execution Checks, User Input

### Community 152 - ".agents/skills/speckit-clarify/SKILL.md"
Cohesion: 0.29
Nodes (6): Completion Report, Done When, Mandatory Post-Execution Hooks, Outline, Pre-Execution Checks, User Input

### Community 153 - ".DeleteRepo"
Cohesion: 0.33
Nodes (5): Engine, repoDefaultBranch(), resolveDeleteEntry(), safeRemoveAll(), validateRemoveAllTarget()

### Community 154 - ".agents/skills/speckit-implement/SKILL.md"
Cohesion: 0.29
Nodes (6): Completion Report, Done When, Mandatory Post-Execution Hooks, Outline, Pre-Execution Checks, User Input

### Community 155 - "001-distribution-channels/spec.md"
Cohesion: 0.16
Nodes (7): ADR-0012: Release Please Owns Release Notes, Alternatives Considered, Consequences, Context, Decision, Negative / risks, Positive

### Community 156 - ".agents/skills/speckit-constitution/SKILL.md"
Cohesion: 0.33
Nodes (5): Outline, Post-Execution Checks, Pre-Execution Checks, Scope Guard, User Input

### Community 157 - "importTargetRelativePath"
Cohesion: 0.44
Nodes (8): importTargetRelativePath(), relativeFromAbsolutePath(), relFromRootBasename(), TestImportTargetRelativePath(), TestImportTargetRelativePathWindowsDriveAbsolute(), cleanRelativePath(), isAbsoluteLikePath(), normalizePathLikeInput()

### Community 158 - ".agents/skills/speckit-taskstoissues/SKILL.md"
Cohesion: 0.40
Nodes (4): Outline, Post-Execution Checks, Pre-Execution Checks, User Input

### Community 159 - "NopLogger"
Cohesion: 0.13
Nodes (14): buildMCPLogger(), newMCPEngine(), TestBuildMCPLoggerReturnsOpenFileErrors(), TestBuildMCPLoggerWithoutPathReturnsNopLogger(), TestBuildMCPLoggerWritesFormattedLines(), TestNewMCPEngineWiresLoggerIntoGitAdapter(), log.Logger, TestSyncSkipsUnsupportedLocalUpdateByAdapterCapability() (+6 more)

### Community 160 - "runtime.go"
Cohesion: 0.24
Nodes (10): collectUninstallTargets(), All(), ByName(), Runtime, register(), SelectionFromFlags(), TestSelectionFromFlagsAll(), TestSelectionFromFlagsEmpty() (+2 more)

### Community 162 - "004-adapter-contract/spec.md"
Cohesion: 0.27
Nodes (4): Coverage summary, Cross-Artifact Analysis: Stable Plugin Adapter Contract, Findings, Verdict

### Community 163 - "TestRemoteMismatchReconcileHelpers"
Cohesion: 0.29
Nodes (6): TestParseRemoteMismatchReconcileModeTable(), TestRemoteMismatchReconcileHelpers(), remoteMismatchReconcileMode, parseRemoteMismatchReconcileMode(), TestRemoteMismatchWrapperFunctions(), ParseRemoteMismatchReconcileMode()

### Community 165 - "Git Compatibility Declaration Contract"
Cohesion: 0.25
Nodes (7): Closed support claim, Document shape, Environment and provisioner rules, Git Compatibility Declaration Contract, Release evidence, Reusable executable interface, Tag and recovery semantics

### Community 166 - "Tasks: Stable Plugin Adapter Contract"
Cohesion: 0.15
Nodes (13): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation Strategy, Parallel opportunities, Phase 0: Gates (blocking), Phase 1: Foundational (blocking prerequisites), Phase 2: User Story 1 — uniform version detection (P1) 🎯 MVP, Phase 3: User Story 2 — published contract (P2) (+5 more)

### Community 167 - "Recipe Contract"
Cohesion: 0.25
Nodes (7): Canonical Topology, Materialization Contract, Preflight Contract, Purpose, Ready-State Invariants, Recipe Contract, Reuse Contract

### Community 168 - "assertions_test.go"
Cohesion: 0.36
Nodes (6): RepositorySnapshot, WorkspaceSnapshot, captureWorkspaceSnapshot(), semanticPath(), sortedRegistryPaths(), hashBytes()

### Community 169 - "TestWorkerChannelBufferSize"
Cohesion: 0.50
Nodes (3): testResponse, testRunner, TestWorkerChannelBufferSize()

### Community 170 - "MCP Tool Matrix Contract"
Cohesion: 0.29
Nodes (6): Canonical Fixtures, Coverage Gate, Current Canonical Cases, Deterministic Call Order, MCP Tool Matrix Contract, Result Shape Rules

### Community 171 - "environment_test.go"
Cohesion: 0.52
Nodes (6): buildChildEnvironment(), copyHostEnvironment(), environmentMap(), executableNullDevice(), lookupEnvironmentFold(), setEnvironmentValue()

### Community 172 - "buildStatusJSONOutput"
Cohesion: 0.06
Nodes (49): T, newLabelsEnvelope(), newRepoEnvelope(), newReposEnvelope(), newResultsEnvelope(), newVersionEnvelope(), assertCarriesAPIVersion(), decodeTopLevel() (+41 more)

### Community 179 - "version_test.go"
Cohesion: 0.10
Nodes (33): Source, advertisedVersion(), orUnavailable(), resolvedBuildInfo(), revisionField(), runVersion(), TestAdvertisedVersion(), TestRevisionField() (+25 more)

### Community 181 - ".claude/skills/speckit-plan/SKILL.md"
Cohesion: 0.18
Nodes (10): Completion Report, Done When, Key rules, Mandatory Post-Execution Hooks, Outline, Phase 0: Outline & Research, Phase 1: Design & Contracts, Phases (+2 more)

### Community 182 - ".claude/skills/speckit-specify/SKILL.md"
Cohesion: 0.18
Nodes (10): Completion Report, Done When, For AI Generation, Mandatory Post-Execution Hooks, Outline, Pre-Execution Checks, Quick Guidelines, Section Requirements (+2 more)

### Community 200 - ".claude/skills/speckit-tasks/SKILL.md"
Cohesion: 0.18
Nodes (10): Checklist Format (REQUIRED), Completion Report, Done When, Mandatory Post-Execution Hooks, Outline, Phase Structure, Pre-Execution Checks, Task Generation Rules (+2 more)

### Community 201 - "Implementation Plan: Stable Plugin Adapter Contract"
Cohesion: 0.18
Nodes (11): Clarify outcomes folded into this plan (session 2026-09-05), Complexity Tracking, Constitution Check, Documentation (this feature), Implementation Plan: Stable Plugin Adapter Contract, Open Risks, Phasing, Project Structure (+3 more)

### Community 206 - "comment_guard_test.go"
Cohesion: 0.50
Nodes (8): mustReadFile(), TestCodexRemoveEntryAbsentCommentedFileSucceeds(), TestCodexRemoveEntryPresentCommentedFileRefuses(), TestCodexWriteEntryAllowsUncommentedFile(), TestCodexWriteEntryRefusesCommentedFile(), TestGrokWriteEntryRefusesCommentedFile(), TestNewConfigFilesAreOwnerOnly(), writeTemp()

### Community 207 - "Implementation Plan: Distribution Channel Conformance"
Cohesion: 0.18
Nodes (11): Complexity Tracking, Constitution Check, Documentation (this feature), Implementation Phasing, Implementation Plan: Distribution Channel Conformance, Post-Phase 1 re-check, Project Structure, Risks (+3 more)

### Community 208 - "Quickstart: Validating Distribution Channel Conformance"
Cohesion: 0.20
Nodes (10): 1. Version identity across install paths (FR-001 – FR-006), 2. Unit tests, 3. Manifests validate (FR-017), 4. Build the image locally, 6. Installed packages (needs a release), 7. Published channels (needs a release), 8. Zero dependency growth (FR-037), Acceptance summary (+2 more)

### Community 209 - "server.json"
Cohesion: 0.22
Nodes (8): description, name, packages, repository, source, url, $schema, version

### Community 210 - "Contract: `server.json` — MCP Registry Entry"
Cohesion: 0.22
Nodes (8): Contract: `server.json` — MCP Registry Entry, Describing a mixed tool surface — the RepoKeeper divergence, Identity, Layer 1 — schema, in CI, Layer 2 — drift, in Go tests, Publishing (FR-018, FR-019), Shape, Validation: two layers

### Community 211 - "Research: Stable Plugin Adapter Contract"
Cohesion: 0.18
Nodes (11): 1. Settled upstream: ADR-0006, 2.1 What is already versioned, 2.2 What is not versioned, 2.3 MCP surface, 2. Current implementation state, 3. Decision 1 — how adapters detect contract version, 4. Decision 2 — promote `v1beta1` to `v1`, 5. Constraints carried into planning (+3 more)

### Community 212 - "WorkspaceRecipe"
Cohesion: 0.29
Nodes (8): ConfigRecipe, MissingEntryRecipe, RegistrySeedMode, WorkspaceRecipe, extensionRecipe(), materializeExtension(), canonicalRecipe(), materializeCanonical()

### Community 213 - "vcs/localbranch_test.go"
Cohesion: 0.48
Nodes (6): enumLine(), TestGitAdapterInspectLocalBranches(), TestGitAdapterInspectLocalBranchesEnumError(), TestGitAdapterInspectLocalBranchesMergedCheckFails(), TestGitAdapterInspectLocalBranchesNoBase(), TestGitAdapterInspectLocalBranchesSkipsPatchWhenDisabled()

### Community 214 - "Contract: Release Artifacts and Channel Verification"
Cohesion: 0.25
Nodes (8): CI additions, Contract: Release Artifacts and Channel Verification, Credential pre-flight (FR-032) — a behavior change, Documentation obligation (FR-015), Linux package contents (FR-012, FR-013), Published artifacts, Single-invocation invariant (FR-031), Verification job (FR-033 – FR-035)

### Community 215 - "TestResolveEditorCommand"
Cohesion: 0.47
Nodes (4): ResolveEditorCommand(), contains(), findSubstring(), TestResolveEditorCommand()

### Community 216 - "Contract: Adapter-Facing Surface Inventory"
Cohesion: 0.20
Nodes (10): CLI surfaces, Contract: Adapter-Facing Surface Inventory, Credentials in URL fields, Drift, Explicitly NOT contractual, Flag defaults that invert the obvious reading, MCP resources, MCP surfaces (+2 more)

### Community 217 - "5. Container behavior (FR-024 – FR-029)"
Cohesion: 0.29
Nodes (7): 5. Container behavior (FR-024 – FR-029), 5a. Read-only inspection works, 5b. Mutating tools refuse, and explain, 5c. Read-only tools are unaffected by 5b, 5d. No workspace, 5e. Multi-root pinning, 5f. Native multi-root is unchanged

### Community 219 - "Requirements Quality Checklist: Stable Plugin Adapter Contract"
Cohesion: 0.22
Nodes (8): Assumption risk, Consistency, Constitutional alignment, Requirement clarity, Requirement completeness, Requirements Quality Checklist: Stable Plugin Adapter Contract, Scope boundaries, Testability

### Community 220 - "Contract: `repokeeper version`"
Cohesion: 0.33
Nodes (6): Consistency invariant (FR-005), Contract: `repokeeper version`, Exit codes, Invocation, Machine-readable output (FR-006), Test obligations

### Community 221 - "Human-readable output"
Cohesion: 0.33
Nodes (6): Human-readable output, Local build from a clean tree (FR-002), Local build from a modified tree (FR-002), Module install — `go install ...@v0.8.0` (FR-001), Nothing recorded (FR-004), Release build — ldflags stamped (FR-003)

### Community 222 - ".claude/skills/speckit-checklist/SKILL.md"
Cohesion: 0.25
Nodes (7): Anti-Examples: What NOT To Do, Checklist Purpose: "Unit Tests for English", Example Checklist Types & Sample Items, Execution Steps, Post-Execution Checks, Pre-Execution Checks, User Input

### Community 228 - "NewHgAdapter"
Cohesion: 0.39
Nodes (6): NewHgAdapter(), TestHgAdapterCloneRejectsFlagLikeArgs(), TestHgAdapterEndToEndWithFakeBinary(), TestHgAdapterIsRepoGracefullyHandlesCommandError(), TestHgAdapterSyncCapabilityMetadata(), TestHgAdapterUnsupportedOperations()

### Community 229 - "Contract: Adapter Response Envelope"
Cohesion: 0.25
Nodes (7): Contract: Adapter Response Envelope, Evolution, Failure behaviour, Identity, Rules, Shape, Verification

### Community 230 - ".claude/skills/speckit-clarify/SKILL.md"
Cohesion: 0.29
Nodes (6): Completion Report, Done When, Mandatory Post-Execution Hooks, Outline, Pre-Execution Checks, User Input

### Community 231 - ".claude/skills/speckit-implement/SKILL.md"
Cohesion: 0.29
Nodes (6): Completion Report, Done When, Mandatory Post-Execution Hooks, Outline, Pre-Execution Checks, User Input

### Community 232 - "Data Model: Stable Plugin Adapter Contract"
Cohesion: 0.29
Nodes (7): 1. Contract envelope, 2. Adapter-facing surface, 3. Stability class, 4. Access classification, 5. Contract version, 6. Per-record shapes, Data Model: Stable Plugin Adapter Contract

### Community 234 - ".claude/skills/speckit-constitution/SKILL.md"
Cohesion: 0.33
Nodes (5): Outline, Post-Execution Checks, Pre-Execution Checks, Scope Guard, User Input

### Community 236 - "edit_test.go"
Cohesion: 0.40
Nodes (5): TestEditRunERejectsInvalidEditedYAML(), TestResolveEditorCommandParsesQuotedExecutable(), TestTrackingBranchFromUpstream(), TestValidateEditedRegistryEntryUniqueness(), writeEditorFixtureScript()

### Community 238 - "Specification Quality Checklist: Distribution Channel Conformance"
Cohesion: 0.33
Nodes (5): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Distribution Channel Conformance

### Community 249 - ".claude/skills/speckit-taskstoissues/SKILL.md"
Cohesion: 0.40
Nodes (4): Outline, Post-Execution Checks, Pre-Execution Checks, User Input

### Community 250 - "status_prune_test.go"
Cohesion: 0.60
Nodes (4): pruneRepoFixture(), TestStatusJSONIncludesLocalBranches(), TestWriteStatusDetailsPruneClassification(), TestWriteStatusDetailsPruneInspectionError()

### Community 251 - "adapter_test.go"
Cohesion: 0.40
Nodes (3): TestGitAdapterMethods(), TestNewGitAdapterDefaultsRunnerAndCloneErrors(), runnerStub

### Community 252 - "toSyncResultJSON"
Cohesion: 1.00
Nodes (3): toSyncResultJSON(), toSyncResultJSONs(), syncResultJSON

## Knowledge Gaps
- **1038 isolated node(s):** `common.sh script`, `runtimeStateKey`, `versionJSON`, `github.com/skaphos/repokeeper/v2`, `localUpdateCapable` (+1033 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **14 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Entry` connect `Entry` to `github.com/mark3labs/mcp-go/mcp.CallToolRequest`, `server.go`, `runDescribeRepo`, `tools_mutation.go`, `index.go`, `.handleBuildWorkspaceInventory`, `.DeleteRepo`, `importTargetRelativePath`, `import_test.go`, `import_clone.go`, `RepoStatus`, `Registry`, `repometa.go`, `newPlanExecEngine`, `NewGitErrorClassifier`, `buildIndexProposal`, `cloneMetadataMap`, `SyncResult`, `StatusReport`, `repairResolveTargetBranch`, `SortRegistryEntries`?**
  _High betweenness centrality (0.012) - this node is a cross-community bridge._
- **Why does `TestEnvelopesNameTheirPayload()` connect `buildStatusJSONOutput` to `testing.T`?**
  _High betweenness centrality (0.011) - this node is a cross-community bridge._
- **Why does `DefaultConfig()` connect `Save` to `writeEmptyConfig`, `Config`, `newEngineWith`, `.handleGetWorkspaceConfig`, `materializeRecipe`, `runDescribeRepo`, `github.com/mark3labs/mcp-go/mcp.CallToolResult`, `SyncResult`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `Save()` (e.g. with `TestSaveErrorsWhenParentIsFile()` and `TestSaveNilConfigErrors()`) actually correct?**
  _`Save()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `DefaultConfig()` (e.g. with `TestSaveErrorsWhenParentIsFile()` and `TestValidateSavedConfigGVKErrors()`) actually correct?**
  _`DefaultConfig()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `common.sh script`, `runtimeStateKey`, `versionJSON` to the rest of the system?**
  _1038 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Save` be split into smaller, more focused modules?**
  _Cohesion score 0.11973875181422351 - nodes in this community are weakly interconnected._