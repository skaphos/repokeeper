# Graph Report - repokeeper  (2026-08-23)

## Corpus Check
- 372 files · ~298,345 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3445 nodes · 7529 edges · 233 communities (211 shown, 22 thin omitted)
- Extraction: 88% EXTRACTED · 12% INFERRED · 0% AMBIGUOUS · INFERRED: 880 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `266168e7`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Save
- GitAdapter
- testing.T
- github.com/spf13/cobra.Command
- MultiAdapter
- Implementation Plan: Recipe-Driven End-to-End Test Harness
- withInstallEnv
- github.com/mark3labs/mcp-go/mcp.CallToolRequest
- codex.go
- model_internal_test.go
- charm.land/bubbletea/v2.Cmd
- server.go
- metadata_forms.go
- runDescribeRepo
- actions_internal_test.go
- columns.go
- context.Context
- status.go
- import_test.go
- multiStubAdapter
- common.sh
- renderDivider
- index.go
- export_test.go
- writeStatusTable
- repokeeper/sync.go
- Tracking
- RepoStatus
- NewGitAdapter
- Entry
- Tasks: [FEATURE NAME]
- Milestones
- Tasks: Recipe-Driven End-to-End Test Harness
- SyncResult
- repometa_test.go
- dropIgnoredImportEntries
- model/model.go
- ApplyPlans
- materializeRecipe
- ADR-0002: Branch Switch and Checkout Workflow Boundaries
- helpers_test.go
- engine/engine.go
- Tasks: Distribution Channel Conformance
- import.go
- RepoMetadata
- perf/main.go
- README.md
- ExecutionResult
- RepoKeeper
- 5.1 CLI commands (v1)
- ADR-0001: MCP Server for Agent-Native Repository Querying and Planning
- filterRows
- addFormatFlag
- index_test.go
- github.com/mark3labs/mcp-go/mcp.CallToolResult
- ADR-0006: Adapter-Facing Contract Stability and Versioning
- RepoKeeper
- runtime_test.go
- mcpserver_test.go
- Adapter
- Changelog
- MCP Server Setup
- newPlanExecEngine
- .handleKey
- WriteTable
- Load
- ADR-0003: Sync Policy and Execution Modes
- ADR-0004: Prune Workflow Boundaries and Safety Model
- Specification Quality Checklist: GitHub Remote End-to-End Expansion
- ADR-0007: Release Binary Publishing and Homebrew Distribution
- Engine
- readJSONDoc
- opencodeAdapter
- ADR-0005: Workspace Config vs Repo-Local Metadata Ownership
- ADR-0008: MCP Install Tooling for Supported Agent Runtimes
- ADR-0011: Credential and Auth Handling Deferred
- Verification Checklist (with Evidence Placeholders)
- .handleSelectRepositories
- Phase 0 Research: Recipe-Driven End-to-End Test Harness
- Feature Specification: [FEATURE NAME]
- cloneMetadataMap
- HgAdapter
- RepoKeeper — Design Spec
- ADR-0010: Repo ID Normalization Stability
- ADR-0014: Local Branch Prune-Safety Classification Model
- ADR-0015: Branch Retention and Protection Policy
- edit_test.go
- LocalBranches
- update.go
- tuiModel
- GitHub Copilot Instructions for RepoKeeper
- mockEngine
- command_parsing_test.go
- Core Principles
- Core Principles
- Repository Guidelines
- install.go
- 5.3 Kubectl-Style CLI Alignment (Milestone 6+)
- Command Notes
- TestHelpersAndPathResolution
- hintForErrorClass
- .handleGetWorkspaceConfig
- Release Process
- import_clone.go
- importTargetRelativePath
- ADR-0009: Replace Release Please with skaphos/actions
- mockEngine
- Scope
- resolveRepo
- parseRemoteMismatchReconcileMode
- Implementation Plan: [FEATURE]
- 6. Data Model
- repair_upstream_test.go
- engine_actions_import_repair_internal_test.go
- SplitCSV
- mcpSession
- newToolError
- Config
- Phase 1 Data Model: Recipe-Driven End-to-End Test Harness
- coverage_boost_test.go
- TestCliio
- os/exec.Cmd
- 9. Stretch Goals
- Decision
- ADR-0012: Release Please Owns Release Notes
- ADR-0013: GoReleaser Owns the GitHub Release
- Decision
- ResolveEditorCommand
- adapter_test.go
- recipe_test.go
- Execution Contract
- Alternatives Considered
- Decision
- Manpage Plan
- Installation
- ClassifyError
- Quickstart: Recipe-Driven End-to-End Harness
- TestGetReconcileRemoteMismatchFlagIsWired
- 4. Safety & Policy
- 8. Architecture
- 001-distribution-channels/spec.md
- Phase 1 Data Model: Distribution Channel Conformance
- MockRunner
- Run
- declaration.go
- [CHECKLIST TYPE] Checklist: [FEATURE NAME]
- Alternatives Considered
- unsupportedLocalUpdateAdapter
- TestTableutil
- check-coverage.sh
- TestRepokeeperSuite
- .DeleteRepo
- TestTui
- editReadyMsg
- normalize.go
- tools_metadata.go
- Git Compatibility Declaration Contract
- Recipe Contract
- assertions_test.go
- MCP Tool Matrix Contract
- environment_test.go
- TestVcs
- run_generation
- version_test.go
- mcpserver/engine.go
- tui/engine.go
- perf/README.md
- coverage-report.sh script
- verify-mcp.sh
- github.com/skaphos/repokeeper
- github.com/skaphos/repokeeper/tools
- comment_guard_test.go
- Implementation Plan: Distribution Channel Conformance
- Quickstart: Validating Distribution Channel Conformance
- server.json
- Contract: `server.json` — MCP Registry Entry
- grokAdapter
- vcs/localbranch_test.go
- Contract: Release Artifacts and Channel Verification
- ResolveEditorCommand
- adapter_test.go
- 5. Container behavior (FR-024 – FR-029)
- TestResolveAbsoluteTargetPath
- Manpage Plan
- Contract: `repokeeper version`
- Human-readable output
- Run
- Q: ok what else is on the docket for repokeeper that is achievable in one day
- nopLogger
- e2e_suite_test.go
- 9. Future: Cross-Machine Registry Sync
- normalizedCLIOutcome

## God Nodes (most connected - your core abstractions)
1. `RepoStatus` - 89 edges
2. `Entry` - 74 edges
3. `Save()` - 70 edges
4. `DefaultConfig()` - 69 edges
5. `Registry` - 68 edges
6. `SyncResult` - 54 edges
7. `Config` - 44 edges
8. `Engine` - 44 edges
9. `NewGitAdapter()` - 44 edges
10. `withTestConfig()` - 43 edges

## Surprising Connections (you probably didn't know these)
- `TestSaveWritesCanonicalFile()` --calls--> `Load()`  [INFERRED]
  internal/repometa/repometa_test.go → test/e2e/internal/compatibility/declaration.go
- `writeTestConfigAndRegistry()` --calls--> `Save()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/registry/registry.go
- `TestSyncRunEPersistsRegistryAfterCheckoutMissingClone()` --calls--> `Load()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/config/config.go
- `TestStatusRunEIncludesRepoMetadata()` --calls--> `Load()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/config/config.go
- `TestIndexRunEWriteRefreshesRepoMetadataSnapshotInConfig()` --calls--> `Load()`  [EXTRACTED]
  cmd/repokeeper/command_run_test.go → internal/config/config.go

## Import Cycles
- None detected.

## Communities (233 total, 22 thin omitted)

### Community 0 - "Save"
Cohesion: 0.10
Nodes (58): TestDescribeRunEIncludesRepoMetadata(), TestDescribeRunEPaths(), TestIndexReposRunEPreviewsAndWritesSelectedRepos(), TestIndexReposRunERequiresPromoteFlag(), TestIndexRunECanPromoteLocalLabels(), TestIndexRunEFailsEarlyWhenMetadataExistsWithoutForce(), TestIndexRunEForceResolvesDualMetadataFiles(), TestIndexRunEPreviewOnlyDoesNotWrite() (+50 more)

### Community 1 - "GitAdapter"
Cohesion: 0.09
Nodes (32): GitRunner, CleanFD(), Clone(), Fetch(), Runner, HasSubmodules(), Head(), IsBare() (+24 more)

### Community 2 - "testing.T"
Cohesion: 0.03
Nodes (115): testing.T, TestSaveNilConfigErrors(), TestValidateSavedConfigGVKErrors(), TestConfig(), TestDiscovery(), TestEngine(), TestGitx(), copyFixture() (+107 more)

### Community 3 - "github.com/spf13/cobra.Command"
Cohesion: 0.19
Nodes (24): TestFlagGettersBranchCoverage(), assumeYes(), configOverride(), debugf(), getBoolFlag(), getCountFlag(), getStringFlag(), infof() (+16 more)

### Community 5 - "Implementation Plan: Recipe-Driven End-to-End Test Harness"
Cohesion: 0.06
Nodes (31): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Recipe-Driven End-to-End Test Harness, Compatibility and Provisioning Design, Constitution Check, Delivery Sequence (+23 more)

### Community 6 - "withInstallEnv"
Cohesion: 0.08
Nodes (49): resetInstallListFlags(), runInstallListWithFlags(), TestInstallListAllNotRegistered(), TestInstallListInvalidScope(), TestInstallListJSON(), TestInstallListProjectCodexUnsupported(), TestInstallListRegisteredStateMatchesExecutable(), TestInstallListStaleWhenCommandDiffers() (+41 more)

### Community 7 - "github.com/mark3labs/mcp-go/mcp.CallToolRequest"
Cohesion: 0.13
Nodes (20): github.com/mark3labs/mcp-go/mcp.CallToolRequest, T, newStructuredListResult(), countNewRegistryEntries(), countRegistryEntriesWithStatus(), MCPServer, optionalStringMapArg(), optionalStringSliceArg() (+12 more)

### Community 8 - "codex.go"
Cohesion: 0.18
Nodes (12): codexServersMap(), Entry, readTOMLDoc(), refuseIfTOMLComments(), skipTOMLDelim(), skipTOMLSingleLine(), tomlHasComments(), writeTOMLDoc() (+4 more)

### Community 9 - "model_internal_test.go"
Cohesion: 0.07
Nodes (30): TestEscInListClearsFilter(), TestF5TriggersRefresh(), TestFilterModeActivation(), TestFilterModeEscClears(), TestFilterModeTyping(), TestHandleKeyQuitCtrlCGlobal(), TestHandleKeyQuitQ(), TestHandleKeyUnhandled() (+22 more)

### Community 10 - "charm.land/bubbletea/v2.Cmd"
Cohesion: 0.24
Nodes (6): charm.land/bubbletea/v2.Cmd, charm.land/bubbletea/v2.Model, charm.land/bubbletea/v2.Msg, repoIDRowCount(), tuiModel, startStatusRefresh()

### Community 11 - "server.go"
Cohesion: 0.07
Nodes (49): github.com/mark3labs/mcp-go/mcp.ReadResourceRequest, github.com/mark3labs/mcp-go/mcp.Resource, github.com/mark3labs/mcp-go/mcp.ResourceContents, github.com/mark3labs/mcp-go/mcp.ResourceTemplate, github.com/mark3labs/mcp-go/mcp.Tool, github.com/mark3labs/mcp-go/server.MCPServer, github.com/mark3labs/mcp-go/server.ToolHandlerFunc, TestMutatingToolsRemainAdvertised() (+41 more)

### Community 12 - "metadata_forms.go"
Cohesion: 0.14
Nodes (28): tuiModel, prepareEditCmd(), cloneMetadataStringMap(), currentRegistryEntry(), currentVisibleRepo(), defaultRepoMetadataForTUI(), detectNamedDirsForTUI(), detectReadmeEntrypointForTUI() (+20 more)

### Community 13 - "runDescribeRepo"
Cohesion: 0.15
Nodes (28): canonicalPathForMatch(), describeCheckoutID(), pathWithinBase(), runDescribeRepo(), samePathForMatch(), selectRegistryEntryForDescribe(), splitRepoAndCheckoutSelector(), TestDescribeRepoSubcommandExists() (+20 more)

### Community 14 - "actions_internal_test.go"
Cohesion: 0.13
Nodes (14): TestEditRepairResetDeleteAddDoneHandlers(), TestHandleAddKey(), TestHandleDeleteConfirmKey(), TestHandleDeleteConfirmKeyResetsOffset(), TestHandleDetailKey(), TestHandleFilterKey(), TestHandleListKey(), TestHandleRepairConfirmKey() (+6 more)

### Community 15 - "columns.go"
Cohesion: 0.06
Nodes (56): TestDetailViewRendersUnknownStaleMarkerOnInspectionError(), TestRenderDetailViewDeterministicMapOrder(), TestRenderDetailViewStripsControlCharactersFromRepoMetadata(), TestSanitizeMetadataText(), TestTUIUsesSharedAndLocalLabelsDistinctly(), colStyledBranch(), colStyledDirty(), colStyledError() (+48 more)

### Community 16 - "context.Context"
Cohesion: 0.04
Nodes (7): stubAdapter, benchAdapter, planAdapter, unsupportedLocalUpdateAdapter, context.Context, Remote, adapterStub

### Community 17 - "status.go"
Cohesion: 0.08
Nodes (44): TestWriteRemoteMismatchPlan(), TestDivergedAdviceAndTable(), TestFormatCellWrapControl(), TestTruncateASCIIBranches(), TestWriteStatusDetailsAndHelpers(), buildDivergedAdvice(), buildStatusJSONOutput(), countGoneRepos() (+36 more)

### Community 18 - "import_test.go"
Cohesion: 0.15
Nodes (21): cloneImportedEntriesWithProgress(), cloneImportedRepos(), cloneImportedReposWithProgress(), normalizeImportedBundle(), TestCloneImportedReposMarksFailedCloneAsMissing(), TestCloneImportedReposNoopWithoutRegistry(), TestCloneImportedReposRejectsDuplicateTargets(), TestCloneImportedReposRejectsUnsafeTargets() (+13 more)

### Community 19 - "multiStubAdapter"
Cohesion: 0.11
Nodes (8): NewAdapterForSelection(), ParseAdapterSelection(), TestMultiAdapterDelegatesAllMethods(), TestMultiAdapterRoutesByPath(), TestMultiAdapterRoutesCapabilityMethodsByPath(), TestNewAdapterForSelection(), TestParseAdapterSelection(), multiStubAdapter

### Community 20 - "common.sh"
Cohesion: 0.08
Nodes (17): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+9 more)

### Community 21 - "renderDivider"
Cohesion: 0.09
Nodes (29): charm.land/bubbletea/v2.View, TestResolveRepairTarget(), TestResolveRepairTargetUsesPathForDuplicateRepoID(), TestViewsAndRendering(), renderDivider(), tuiModel, renderDeleteConfirmView(), duplicateSyncRepoCounts() (+21 more)

### Community 22 - "index.go"
Cohesion: 0.10
Nodes (28): buildIndexProposal(), detectAuthoritativePaths(), detectLowValuePaths(), detectReadmeEntrypoint(), fallbackMetadataPath(), formatRelatedRepoDefaults(), guessRepoMetadataDefaults(), humanizeRepoName() (+20 more)

### Community 23 - "export_test.go"
Cohesion: 0.11
Nodes (20): TestCommonPathRoot(), commonPathRoot(), exportEntriesWithEmbeddedCredentials(), exportEntryPath(), inferRegistrySharedRoot(), populateExportBranches(), prepareRegistryForExport(), TestExportCommandRunELoadsRegistryFromRegistryPath() (+12 more)

### Community 24 - "writeStatusTable"
Cohesion: 0.24
Nodes (12): writeStatusTable(), adaptiveCellLimit(), adaptiveCellLimitForWidth(), tableWidth(), captureStatusTableOutputAtWidth(), captureSyncTableOutputAtWidth(), TestAdaptiveCellLimitForWidth(), TestStatusTableHeaderSnapshotsAcrossWidths() (+4 more)

### Community 25 - "repokeeper/sync.go"
Cohesion: 0.13
Nodes (24): TestDescribeSyncAction(), TestDescribeSyncActionAdditionalBranches(), TestSyncPlanNeedsConfirmation(), TestWriters(), TestWriteSyncFailureSummary(), TestWriteSyncTableNoHeaders(), TestWriteSyncTableWideBranches(), describeSyncAction() (+16 more)

### Community 26 - "Tracking"
Cohesion: 0.08
Nodes (17): dirtyBehindAdapter, ForEachRefEntry, TestTrackingFromShort(), trackingFromShort(), TrackingStatus(), WorktreeStatus(), ParseForEachRef(), ParsePorcelainStatus() (+9 more)

### Community 27 - "RepoStatus"
Cohesion: 0.21
Nodes (6): inspectFailureResult(), supportsLocalUpdate(), RemoteTrackingRefStatus, RepoStatus, SeedRepoMetadataStatus(), registryEntryToPartialStatus()

### Community 28 - "NewGitAdapter"
Cohesion: 0.16
Nodes (26): TestRemoteMismatchReconcileHelpers(), testResponse, testRunner, TestEngineGuardErrors(), TestExecuteSyncPlanAppliesActions(), TestExecuteSyncPlanCloneAction(), TestExecuteSyncPlanStopsOnFailure(), TestExecuteSyncPlanStopsOnNonDryRunFailure() (+18 more)

### Community 29 - "Entry"
Cohesion: 0.12
Nodes (22): persistDescribeMetadataSnapshot(), editRegistryEntryWithEditor(), findRegistryEntryIndex(), resolveEditorCommand(), trackingBranchFromUpstream(), validateEditedRegistryEntry(), resolveUpstreamTargetBranch(), findRegistryMetadataEntry() (+14 more)

### Community 30 - "Tasks: [FEATURE NAME]"
Cohesion: 0.07
Nodes (26): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation Strategy, Incremental Delivery, MVP First (User Story 1 Only) (+18 more)

### Community 31 - "Milestones"
Cohesion: 0.07
Nodes (27): CI pipeline (GitHub Actions), Coverage requirements, Integration tests, Linting & code quality, Milestone 0 — Repo skeleton, Milestone 10 — 1.0 Readiness & Release Reset, Milestone 11 — MCP Server for Agent-Native Querying, Milestone 1 — Discovery + registry (+19 more)

### Community 32 - "Tasks: Recipe-Driven End-to-End Test Harness"
Cohesion: 0.06
Nodes (31): Compatibility Contract Specs, Declaration and Reusable Compatibility Interface, Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Foundation, Foundational Implementation, Foundational Validation Specs, Implementation Strategy (+23 more)

### Community 33 - "SyncResult"
Cohesion: 0.21
Nodes (10): ImportCloneCallbacks, findRegistryEntryForSyncResult(), Engine, SyncOptions, SyncResult, SyncResultCallback, SyncStartCallback, replaceRegistryEntry() (+2 more)

### Community 34 - "repometa_test.go"
Cohesion: 0.19
Nodes (22): TestLoadInvalidYAMLErrors(), Apply(), mustMetadataFingerprint(), rewriteFilePreservingFingerprint(), testAbsolutePath(), TestApplyCachesDualFileConflict(), TestApplyCachesInvalidMetadataError(), TestApplyHandlesMetadataDeletion() (+14 more)

### Community 35 - "dropIgnoredImportEntries"
Cohesion: 0.16
Nodes (17): dropIgnoredImportEntries(), ignoredPathSet(), TestDropIgnoredImportEntriesCaseInsensitiveOnWindows(), TestDropIgnoredImportEntriesRemovesIgnoredTargets(), pathCleanCanonical(), CanonicalNormalize(), CleanNormalize(), IgnoredPathSet() (+9 more)

### Community 36 - "model/model.go"
Cohesion: 0.13
Nodes (23): localBranchNamesByCategory(), time.Time, Engine, TestUpstreamStatusFromSignal(), upstreamStatusFromSignal(), LocalBranch, LocalBranchStatus, PruneCategory (+15 more)

### Community 37 - "ApplyPlans"
Cohesion: 0.19
Nodes (15): Engine, RemoteMismatchPlan, RemoteMismatchReconcileMode, ApplyPlans(), BuildPlans(), findRegistryEntryIndexForStatus(), ParseReconcileMode(), primaryRemoteURL() (+7 more)

### Community 38 - "materializeRecipe"
Cohesion: 0.18
Nodes (17): MaterializedMissingEntry, MaterializedRepository, MaterializedWorkspace, RegistrySeedMode, cloneMap(), commitAll(), containedPath(), fileURL() (+9 more)

### Community 39 - "ADR-0002: Branch Switch and Checkout Workflow Boundaries"
Cohesion: 0.09
Nodes (22): 1. Fold checkout into sync, 2. Keep checkout permanently out of scope, 3. Allow branch switching through MCP, ADR-0002: Branch Switch and Checkout Workflow Boundaries, Alternatives Considered, Branch switch / checkout, CLI, Consequences (+14 more)

### Community 40 - "helpers_test.go"
Cohesion: 0.11
Nodes (19): TestColorizeAndTrackingDisplayBranches(), TestColorizeGuardBranches(), TestConfirmSyncExecution(), TestConfirmSyncExecutionEOF(), TestConfirmWithPrompt(), TestDisplayRepoPathPrefersCWDThenRoot(), TestHasRegistryWarnings(), TestStatusExitCode() (+11 more)

### Community 41 - "engine/engine.go"
Cohesion: 0.10
Nodes (33): localUpdateCapable, OutcomeKind, PullRebasePolicyOptions, syncStep, executedNonCloneOutcome(), filterRegistryEntriesByIgnoredPaths(), filterRequiresInspect(), filterStatus() (+25 more)

### Community 42 - "Tasks: Distribution Channel Conformance"
Cohesion: 0.06
Nodes (33): Correction made during implementation, Dependencies & Execution Order, Documentation, Execution Notes, Format: `[ID] [P?] [Story] Description`, Format validation, Highest-risk tasks, Implementation for User Story 1 (+25 more)

### Community 43 - "import.go"
Cohesion: 0.13
Nodes (25): TestCloneRegistry(), inferredCheckoutIDFromPath(), loadExistingConfig(), mergeImportedRegistry(), mergePolicyPreflightSkips(), mergeRegistryMatchIndex(), parseImportConflictPolicy(), parseImportMode() (+17 more)

### Community 44 - "RepoMetadata"
Cohesion: 0.14
Nodes (25): RepoMetadata, canonicalize(), discoverMetadataState(), discoverPath(), fileExists(), metadataConflictFingerprint(), metadataFileFingerprint(), normalize() (+17 more)

### Community 45 - "perf/main.go"
Cohesion: 0.21
Nodes (16): benchmarkMetric, benchmarkRunRecord, appendRecord(), gitShortCommit(), loadLastRecord(), main(), parseBenchmarkMetrics(), printSummary() (+8 more)

### Community 46 - "README.md"
Cohesion: 0.09
Nodes (16): Branching and Commits, Coding Standards, Contributing Guidelines, Development Setup, Graphify, Pull Requests, Release Process, Safety Expectations (+8 more)

### Community 47 - "ExecutionResult"
Cohesion: 0.17
Nodes (14): boundedBuffer, ExecutionResult, scanJSONResponse, statusJSONRepository, statusJSONResponse, bytes.Buffer, time.Duration, decodeScanJSON() (+6 more)

### Community 48 - "RepoKeeper"
Cohesion: 0.09
Nodes (23): Commands, Configuration, Container (MCP server), Development, Documentation, Expected User Flow, Features, From release binaries (+15 more)

### Community 49 - "5.1 CLI commands (v1)"
Cohesion: 0.11
Nodes (18): 5.1 CLI commands (v1), Exit codes, Global flags (apply to all commands), `repokeeper add <path> <git-repo-url>`, `repokeeper delete <repo-id-or-path>`, `repokeeper describe <repo-id-or-path>`, `repokeeper edit <repo-id-or-path>`, `repokeeper export` (+10 more)

### Community 50 - "ADR-0001: MCP Server for Agent-Native Repository Querying and Planning"
Cohesion: 0.11
Nodes (18): 1. Keep MCP as a mixed read/write surface, 2. Remove MCP entirely and rely on CLI JSON only, 3. Allow execution only for "safe" mutations, ADR-0001: MCP Server for Agent-Native Repository Querying and Planning, Alternatives Considered, Architecture, Consequences, Context (+10 more)

### Community 51 - "filterRows"
Cohesion: 0.21
Nodes (15): filterRows(), matchesFilter(), TestFilterRowsByBranch(), TestFilterRowsByDisplayLabel(), TestFilterRowsByErrorClass(), TestFilterRowsByLabelValue(), TestFilterRowsByPath(), TestFilterRowsByRepoID() (+7 more)

### Community 52 - "addFormatFlag"
Cohesion: 0.29
Nodes (13): init(), addFormatFlag(), addLabelSelectorFlag(), addNoHeadersFlag(), addRepoFilterFlags(), addUpstreamRepairFilterFlag(), addVCSFlag(), init() (+5 more)

### Community 53 - "index_test.go"
Cohesion: 0.23
Nodes (13): formatAssignmentDefaults(), TestFormatAssignmentDefaultsSortsKeys(), TestGuessRepoMetadataDefaultsClonesExistingMetadata(), TestIndexQuestionerUsesDefaultsAndParsers(), TestUnifiedDiffEmptyWhenIdentical(), TestUnifiedDiffReportsChanges(), TestWriteMetadataPreviewDiffsUnparseableExisting(), TestWriteMetadataPreviewNewFile() (+5 more)

### Community 54 - "github.com/mark3labs/mcp-go/mcp.CallToolResult"
Cohesion: 0.16
Nodes (19): MCPToolCase, github.com/mark3labs/mcp-go/mcp.CallToolResult, addedRepositoryPath(), mutationMCPToolCases(), emptyMCPArguments(), readMCPToolCases(), requireFields(), requireList() (+11 more)

### Community 55 - "ADR-0006: Adapter-Facing Contract Stability and Versioning"
Cohesion: 0.12
Nodes (16): 1. Treat CLI JSON as best-effort only, 2. Require adapters to import internal packages, 3. Use MCP as the only stable contract, ADR-0006: Adapter-Facing Contract Stability and Versioning, Alternatives Considered, Breaking changes must be explicit, Consequences, Context (+8 more)

### Community 56 - "RepoKeeper"
Cohesion: 0.12
Nodes (16): Avoid these mistakes, Check health first, Core rules, Discovery workflow, Execute safe updates, Good agent response pattern, Initialization workflow, Labeling workflow (+8 more)

### Community 57 - "runtime_test.go"
Cohesion: 0.19
Nodes (20): init(), All(), ByName(), Runtime, register(), SelectionFromFlags(), names(), TestAllReturnsSortedCopy() (+12 more)

### Community 58 - "mcpserver_test.go"
Cohesion: 0.16
Nodes (12): github.com/mark3labs/mcp-go/mcp.JSONRPCMessage, callTool(), expectResourceError(), expectResourceSuccess(), intPtr(), newTestConfig(), newTestRegistry(), newTestStatusReport() (+4 more)

### Community 59 - "Adapter"
Cohesion: 0.12
Nodes (24): Options, Result, stubLBAdapter, buildResult(), detectRepo(), gitdirFromFile(), TestBuildResultBranches(), TestDetectRepoBranches() (+16 more)

### Community 60 - "Changelog"
Cohesion: 0.11
Nodes (17): [0.6.1](https://github.com/skaphos/repokeeper/compare/v0.6.0...v0.6.1) (2026-04-03), [0.7.0](https://github.com/skaphos/repokeeper/compare/v0.6.1...v0.7.0) (2026-04-09), [0.7.1](https://github.com/skaphos/repokeeper/compare/v0.7.0...v0.7.1) (2026-04-18), [1.2.0](https://github.com/skaphos/repokeeper/compare/v1.1.0...v1.2.0) (2026-05-31), [1.3.0](https://github.com/skaphos/repokeeper/compare/v1.2.0...v1.3.0) (2026-06-22), [1.3.1](https://github.com/skaphos/repokeeper/compare/v1.3.0...v1.3.1) (2026-07-12), [1.4.0](https://github.com/skaphos/repokeeper/compare/v1.3.1...v1.4.0) (2026-07-26), Bug Fixes (+9 more)

### Community 61 - "MCP Server Setup"
Cohesion: 0.07
Nodes (29): 1. Tool Discovery (all 14 tools visible), 2. Read-only Tools (safe, no side effects), 3. Planning Tools (dry-run only), 4. Mutation Tools + Safety Gates, 5. Structured Content + Error Handling, 6. Client Compatibility Smoke Test, Available tools, CLI skill fallback (+21 more)

### Community 62 - "newPlanExecEngine"
Cohesion: 0.24
Nodes (12): Engine, Engine, SyncResult, newPlanExecEngine(), TestApplyRemoteMismatchPlansUsesInjectedAdapter(), TestExecutePlannedNonCloneUnknownStepFailsInvalid(), TestExecutePlannedSyncItemEmptyStepsFailsInvalid(), TestParseFilterKind() (+4 more)

### Community 63 - ".handleKey"
Cohesion: 0.34
Nodes (7): charm.land/bubbletea/v2.KeyPressMsg, TestModalHelpers(), tuiModel, isModalNav(), modalMoveLeft(), modalMoveRight(), renderModalButtons()

### Community 64 - "WriteTable"
Cohesion: 0.10
Nodes (22): errorWriter, runInstallList(), writeInstallListTable(), needsUpstreamRepair(), parseUpstreamRepairFilter(), writeRepairUpstreamTable(), github.com/liggitt/tabwriter.Writer, io.Writer (+14 more)

### Community 65 - "Load"
Cohesion: 0.09
Nodes (43): TestAddCommandWithAbsoluteTargetDoesNotReRootUnderCWD(), mustRunGit(), TestAddDeleteWithRegistryOverride(), TestAddValidationMutuallyExclusiveFlags(), TestDeleteCancelledByPrompt(), TestDeleteTrackingOnlyAddsIgnoredPathAndKeepsRepo(), TestInitCommandForceBehavior(), TestScanJSONOutputAndUnsupportedFormat() (+35 more)

### Community 66 - "ADR-0003: Sync Policy and Execution Modes"
Cohesion: 0.13
Nodes (14): ADR-0003: Sync Policy and Execution Modes, CLI and TUI, Configuration Boundary, Consequences, Context, Decision, Execution Model, Interface Boundaries (+6 more)

### Community 67 - "ADR-0004: Prune Workflow Boundaries and Safety Model"
Cohesion: 0.13
Nodes (15): 1. Keep prune as an implicit part of sync, 2. Treat local branch prune as a simple merged/not-merged check, ADR-0004: Prune Workflow Boundaries and Safety Model, Alternatives Considered, CLI and TUI, Consequences, Context, Decision (+7 more)

### Community 68 - "Specification Quality Checklist: GitHub Remote End-to-End Expansion"
Cohesion: 0.10
Nodes (18): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: GitHub Remote End-to-End Expansion, Assumptions, Edge Cases, Feature Specification: GitHub Remote End-to-End Expansion (+10 more)

### Community 70 - "ADR-0007: Release Binary Publishing and Homebrew Distribution"
Cohesion: 0.14
Nodes (14): 1. Goreleaser owns the GitHub release and release body; release-please bumps version; the `release-please.yml` workflow pushes the tag, 2. Keep `homebrew_casks:`, but make it install cleanly on Apple Silicon, 3. Backfill v0.7.0 assets (one-off), ADR-0007: Release Binary Publishing and Homebrew Distribution, Consequences, Context, Decision, Implementation plan (+6 more)

### Community 71 - "Engine"
Cohesion: 0.14
Nodes (20): executeImportClonePlanWithProgress(), planImportedEntries(), TestActionsResetDeleteCloneAndRegister(), TestExecuteImportClonesSuccessFailureAndSkips(), TestPlanImportClonesGuardsAndErrors(), TestPlanImportClonesSkipsAndSuccess(), TestRepairUpstreamScenarios(), TestSyncSkipsUnsupportedLocalUpdateByAdapterCapability() (+12 more)

### Community 72 - "readJSONDoc"
Cohesion: 0.13
Nodes (13): Entry, init(), readJSONDoc(), writeJSONDoc(), checkJsonc(), Entry, init(), opencodeDir() (+5 more)

### Community 73 - "opencodeAdapter"
Cohesion: 0.09
Nodes (22): Assumptions, Clarifications, Constitution Alignment, Dependencies, Edge Cases, Feature Specification: Distribution Channel Conformance, Functional Requirements, Key Entities (+14 more)

### Community 74 - "ADR-0005: Workspace Config vs Repo-Local Metadata Ownership"
Cohesion: 0.15
Nodes (13): 1. Use repo-local metadata as the home for most policy, 2. Keep repo-local metadata narrowly informational with no agent/runtime fields, ADR-0005: Workspace Config vs Repo-Local Metadata Ownership, Alternatives Considered, Consequences, Context, Decision, Negative (+5 more)

### Community 75 - "ADR-0008: MCP Install Tooling for Supported Agent Runtimes"
Cohesion: 0.11
Nodes (19): 1. CLI surface, 1. Keep `skill install/uninstall`, add parallel `mcp install/uninstall/list`, 2. Architecture: per-runtime adapter interface, 2. `repokeeper install` as a leaf verb, with a separate top-level `repokeeper status` for state, 3. Data-table architecture instead of per-runtime adapters, 3. Documentation changes, 4. Silent fallback for `--scope project --codex` to user scope, 5. Prompt before every overwrite (+11 more)

### Community 76 - "ADR-0011: Credential and Auth Handling Deferred"
Cohesion: 0.15
Nodes (12): 1. Add a thin credential helper wrapper, 2. Document specific recommended setups in this ADR, 3. Build a `repokeeper auth doctor` command, 4. Encrypt the registry at rest, ADR-0011: Credential and Auth Handling Deferred, Alternatives Considered, Consequences, Context (+4 more)

### Community 77 - "Verification Checklist (with Evidence Placeholders)"
Cohesion: 0.15
Nodes (12): 1. Tool Discovery, 2. Read-only Tools, 3. Planning Tools, 4. Mutation Tools + Safety Gates, 5. Structured Content & Error Quality, 6. Overall Claude Experience, How to Run the Verification (for the user), MCP Manual Verification Results (SKA-201) (+4 more)

### Community 78 - ".handleSelectRepositories"
Cohesion: 0.16
Nodes (16): filterBulkIndexEntriesByLabels(), filterStatusReportByLabels(), filterStatusReportByLocalLabels(), MCPServer, matchesStatusFilter(), buildMatchReason(), enrichAnnotations(), enrichLabels() (+8 more)

### Community 79 - "Phase 0 Research: Recipe-Driven End-to-End Test Harness"
Cohesion: 0.11
Nodes (18): Decision 10: Use separate CLI and MCP materializations and deterministic MCP order, Decision 11: Compare semantic normalized outcomes five times, Decision 12: Reuse existing modules and add no ADR, Decision 13: Run the E2E boundary in four supported environments, Decision 14: Separate representative CI from exhaustive release qualification, Decision 15: Centralize matrix, provisioning, and evidence behavior in a tagged command, Decision 16: Provision immutable Git inputs per environment, Decision 17: Make race coverage and release-candidate recovery explicit (+10 more)

### Community 80 - "Feature Specification: [FEATURE NAME]"
Cohesion: 0.15
Nodes (12): Assumptions, Edge Cases, Feature Specification: [FEATURE NAME], Functional Requirements, Key Entities *(include if feature involves data)*, Measurable Outcomes, Requirements *(mandatory)*, Success Criteria *(mandatory)* (+4 more)

### Community 81 - "cloneMetadataMap"
Cohesion: 0.43
Nodes (5): parseMetadataAssignments(), parseMetadataKeys(), TestParseMetadataAssignments(), TestParseMetadataKeys(), validateMetadataKey()

### Community 82 - "HgAdapter"
Cohesion: 0.08
Nodes (9): rejectFlagLike(), runCommand(), NewHgAdapter(), TestHgAdapterCloneRejectsFlagLikeArgs(), TestHgAdapterEndToEndWithFakeBinary(), TestHgAdapterIsRepoGracefullyHandlesCommandError(), TestHgAdapterSyncCapabilityMetadata(), TestHgAdapterUnsupportedOperations() (+1 more)

### Community 83 - "RepoKeeper — Design Spec"
Cohesion: 0.17
Nodes (11): 10. Open Questions (explicitly deferred), 1. Summary, 2. Problem Statement, 3.1 Functional goals (v1 / "80%"), 3.2 Non-goals (defer), 3. Goals, 7.0 Git CLI Strategy & Compatibility Matrix, 7. Git Operations (Engine Contract) (+3 more)

### Community 84 - "ADR-0010: Repo ID Normalization Stability"
Cohesion: 0.17
Nodes (11): 1. Treat normalization as implementation detail, 2. Version every normalization rule from day one, 3. Hash the normalized URL instead of storing it, ADR-0010: Repo ID Normalization Stability, Alternatives Considered, Consequences, Context, Decision (+3 more)

### Community 85 - "ADR-0014: Local Branch Prune-Safety Classification Model"
Cohesion: 0.17
Nodes (12): 1. Treat "upstream gone" as `probably_safe` without patch-equivalence, 2. Use `git branch --merged` (reachability) as the sole integration signal, 3. Put the enums in a dependency-free `internal/prune` package, 4. Classify only the current branch, reusing existing state, ADR-0014: Local Branch Prune-Safety Classification Model, Alternatives Considered, Consequences, Context (+4 more)

### Community 86 - "ADR-0015: Branch Retention and Protection Policy"
Cohesion: 0.17
Nodes (12): 1. Workspace-global `base_branch` (empty ⇒ `Defaults.MainBranch`), 2. Reuse `--protected-branches` for prune and union it in, 3. Store branch policy in `.repokeeper-repo.yaml`, 4. Define the policy type inside the classifier and have config embed it, ADR-0015: Branch Retention and Protection Policy, Alternatives Considered, Consequences, Context (+4 more)

### Community 87 - "edit_test.go"
Cohesion: 0.38
Nodes (6): TestEditRunERejectsInvalidEditedYAML(), TestEditRunEUpdatesSingleRepoFromEditor(), TestResolveEditorCommandParsesQuotedExecutable(), TestTrackingBranchFromUpstream(), TestValidateEditedRegistryEntryUniqueness(), writeEditorFixtureScript()

### Community 88 - "LocalBranches"
Cohesion: 0.21
Nodes (13): LocalBranchInfo, stubRunner, LocalBranches(), MergedBranches(), ParseCherryEquivalent(), ParseLocalBranches(), PatchEquivalentToBase(), TestLocalBranchesRunner() (+5 more)

### Community 89 - "update.go"
Cohesion: 0.21
Nodes (10): TestEditKeyHandlersAndMetadataFieldMutationHelpers(), appendRepoMetadataField(), executeSyncCmd(), EngineAPI, removeRepoByID(), repoStatusStillTracked(), resetRepoMetadataEditState(), trimRepoMetadataField() (+2 more)

### Community 90 - "tuiModel"
Cohesion: 0.18
Nodes (13): charm.land/bubbletea/v2.Program, sync/atomic.Pointer, TestViewDispatchAndOtherHelpers(), EngineAPI, tuiModel, loadStatusCmd(), cloneRegistryMetadataMap(), EngineAPI (+5 more)

### Community 91 - "GitHub Copilot Instructions for RepoKeeper"
Cohesion: 0.18
Nodes (10): Codebase Shape, Commit and Branch Guidance, Documentation Expectations, GitHub Copilot Instructions for RepoKeeper, Go and Repository Conventions, Pull Request Instructions, Safety Rules, Testing Expectations (+2 more)

### Community 93 - "command_parsing_test.go"
Cohesion: 0.25
Nodes (7): TestRepairUpstreamMatchesFilterTable(), TestShouldStreamSyncResults(), TestSyncProgressMessageKinds(), TestSyncResultNeedsConfirmationTable(), TestShouldStreamSyncResultsBranches(), repairUpstreamMatchesFilter(), shouldStreamSyncResults()

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
Cohesion: 0.43
Nodes (6): desiredInstallEntry(), parseInstallScope(), printClaudePermissionsBlock(), printManualSnippets(), runInstall(), Entry

### Community 98 - "5.3 Kubectl-Style CLI Alignment (Milestone 6+)"
Cohesion: 0.20
Nodes (10): 5.2 TUI command (phase 2), 5.3.1 Command shape, 5.3.2 Output contracts, 5.3.3 Styling and color policy (intentional delta vs kubectl), 5.3.4 Filter and selector direction, 5.3.5 Migration strategy, 5.3 Kubectl-Style CLI Alignment (Milestone 6+), 5. User Experience (+2 more)

### Community 99 - "Command Notes"
Cohesion: 0.15
Nodes (13): Command Notes, Global Flags, `repokeeper add`, RepoKeeper Commands, `repokeeper describe`, `repokeeper edit`, `repokeeper get`, `repokeeper index` (+5 more)

### Community 100 - "TestHelpersAndPathResolution"
Cohesion: 0.12
Nodes (18): TestActionCmdsPropagateProvidedContext(), TestHelpersAndPathResolution(), cloneAndRegisterCmd(), defaultClonePath(), EngineAPI, tuiModel, renderAddView(), repoNameFromURL() (+10 more)

### Community 101 - "hintForErrorClass"
Cohesion: 0.28
Nodes (6): Engine, SyncResult, hintForErrorClass(), TestHintForErrorClass_Deterministic(), TestHintForErrorClass_KnownClasses(), TestHintForErrorClass_UnknownClasses()

### Community 102 - ".handleGetWorkspaceConfig"
Cohesion: 0.36
Nodes (6): cfgDefault(), MCPServer, intDefault(), positiveIntDefault(), configDefaults, workspaceConfigResponse

### Community 103 - "Release Process"
Cohesion: 0.15
Nodes (13): 1. Land releasable commits on `main`, 2. Run local release checks, 3. Review and merge the release PR, 4. Tag push + GitHub release automation, 5. Verify the release, Channel verification, Compatibility qualification boundary, Credentials fail the release; they never skip a channel (+5 more)

### Community 104 - "import_clone.go"
Cohesion: 0.24
Nodes (11): importCloneConflict, ImportCloneOptions, ImportCloneSkip, ImportCloneTarget, TestImportCloneHelperFunctions(), findImportCloneConflicts(), Engine, ImportClonePlan (+3 more)

### Community 105 - "importTargetRelativePath"
Cohesion: 0.44
Nodes (8): importTargetRelativePath(), relativeFromAbsolutePath(), relFromRootBasename(), TestImportTargetRelativePath(), TestImportTargetRelativePathWindowsDriveAbsolute(), cleanRelativePath(), isAbsoluteLikePath(), normalizePathLikeInput()

### Community 106 - "ADR-0009: Replace Release Please with skaphos/actions"
Cohesion: 0.12
Nodes (16): Action pin, ADR-0009: Replace Release Please with skaphos/actions, Alternatives considered, Bot identity, Consequences, Context, Decision, Files added (+8 more)

### Community 107 - "mockEngine"
Cohesion: 0.15
Nodes (7): TestRepairHelpers(), Engine, RepairUpstreamResult, repairNeedsUpstream(), repairResolveTargetBranch(), mockEngine, repairDoneMsg

### Community 108 - "Scope"
Cohesion: 0.16
Nodes (6): collectUninstallTargets(), Scope, Entry, TestScopeString(), fakeRuntime, uninstallTarget

### Community 109 - "resolveRepo"
Cohesion: 0.28
Nodes (4): resolveRepo(), MCPServer, mergeLabels(), MCPServer

### Community 110 - "parseRemoteMismatchReconcileMode"
Cohesion: 0.33
Nodes (5): TestParseRemoteMismatchReconcileModeTable(), remoteMismatchReconcileMode, parseRemoteMismatchReconcileMode(), TestRemoteMismatchWrapperFunctions(), ParseRemoteMismatchReconcileMode()

### Community 111 - "Implementation Plan: [FEATURE]"
Cohesion: 0.22
Nodes (8): Complexity Tracking, Constitution Check, Documentation (this feature), Implementation Plan: [FEATURE], Project Structure, Source Code (repository root), Summary, Technical Context

### Community 112 - "6. Data Model"
Cohesion: 0.25
Nodes (8): 6.1 Repo identity, 6.2.1 Machine config, 6.2.2 Registry (embedded in machine config by default), 6.2 Config files, 6.3 Status JSON schema (v1), 6.4 Sync (reconcile) JSON schema, 6. Data Model, JSON output schema stability policy

### Community 113 - "repair_upstream_test.go"
Cohesion: 0.25
Nodes (7): TestNeedsUpstreamRepair(), TestParseUpstreamRepairFilter(), TestRepairUpstreamMatchesFilter(), TestResolveUpstreamTargetBranch(), TestWriteRepairUpstreamTable(), TestWriteRepairUpstreamTableCompactsColumnsOnTinyTTY(), TestWriteRepairUpstreamTableNoHeaders()

### Community 114 - "engine_actions_import_repair_internal_test.go"
Cohesion: 0.39
Nodes (6): LessRepoIDPath(), SortRegistryEntries(), SortRepoStatuses(), TestLessRepoIDPath(), TestSortRegistryEntries(), TestSortRepoStatuses()

### Community 115 - "SplitCSV"
Cohesion: 0.32
Nodes (5): TestSplitCSV(), ParseFieldSelectorFilter(), ResolveRepoFilter(), SplitCSV(), TestSplitCSV()

### Community 116 - "mcpSession"
Cohesion: 0.24
Nodes (7): mcpSession, context.CancelFunc, github.com/mark3labs/mcp-go/client.Client, io.WriteCloser, sync.Once, startMCPSession(), validateJSONRPCFrames()

### Community 117 - "newToolError"
Cohesion: 0.19
Nodes (16): explainReadOnly(), isReadOnly(), mentionsReadOnlyFilesystem(), resultErrorText(), TestExplainReadOnlyDoesNotMisdiagnoseOtherGitFailures(), TestExplainReadOnlyNamesCauseAndRemedy(), TestExplainReadOnlyPassesThroughUnrelatedErrors(), TestExplainReadOnlyRecognisesGitSubprocessFailures() (+8 more)

### Community 118 - "Config"
Cohesion: 0.11
Nodes (30): persistSyncRegistryAfterCheckoutMissing(), BranchPolicy, Defaults, configDir(), ConfigPath(), ConfigRoot(), EffectiveRoot(), findNearestConfigPath() (+22 more)

### Community 119 - "Phase 1 Data Model: Recipe-Driven End-to-End Test Harness"
Cohesion: 0.14
Nodes (13): BranchRecipe and CommitRecipe, CompatibilityCommandResult, CompatibilityResult, ConfigRecipe, ExecutionResult, ExpectedOutcome and NormalizedOutcome, GitCompatibilityMatrix and CompatibilityCell, MaterializedWorkspace (+5 more)

### Community 120 - "coverage_boost_test.go"
Cohesion: 0.08
Nodes (32): TestLogOutputWriteFailureLogsError(), TestLogOutputWriteFailureNilError(), TestMarshalToGenericMarshalErrorPath(), TestMarshalToGenericUnmarshalErrorPath(), TestNewSyncProgressWriter(), TestResolveCustomColumnValueEdgeCases(), TestRootRunEHelpFallbackForNonTerminalOutput(), TestRowsForCustomColumnsFallbackPaths() (+24 more)

### Community 122 - "os/exec.Cmd"
Cohesion: 0.32
Nodes (10): os/exec.Cmd, attachProcessTree(), configureProcessTree(), forceTerminateProcessTree(), releaseProcessTree(), attachProcessTree(), closeProcessJob(), configureProcessTree() (+2 more)

### Community 123 - "9. Stretch Goals"
Cohesion: 0.29
Nodes (7): 7.1 Detection commands, 7.2 Tracking status (preferred), 7.3 Sync operation (core action), 7.4 Error handling guidelines, 9.1 Multi-VCS Support (Hg), 9. Stretch Goals, Bare repo handling

### Community 124 - "Decision"
Cohesion: 0.14
Nodes (14): Phase 0 Research: Distribution Channel Conformance, R10 — Mercurial in the container, R11 — ADR-0016 and the deviation record, R12 — Zero dependency growth, R1 — Version identity: three-tier resolution, R2 — Container base image ⚠ DIVERGENCE, R3 — Git "dubious ownership" in the container ⚠ DIVERGENCE, measured, R4 — Workspace discovery: cwd-relative crawl vs. a frozen container ⚠ DIVERGENCE (+6 more)

### Community 125 - "ADR-0012: Release Please Owns Release Notes"
Cohesion: 0.29
Nodes (7): ADR-0012: Release Please Owns Release Notes, Alternatives Considered, Consequences, Context, Decision, Negative / risks, Positive

### Community 126 - "ADR-0013: GoReleaser Owns the GitHub Release"
Cohesion: 0.29
Nodes (7): ADR-0013: GoReleaser Owns the GitHub Release, Alternatives Considered, Consequences, Context, Decision, Negative / risks, Positive

### Community 127 - "Decision"
Cohesion: 0.29
Nodes (7): Boundary, Decision, Defaults must survive the zero-value backfill idiom, Per-repo base resolution, Protection is prune-scoped and independent of rebase protection, Schema, Validation fails closed

### Community 128 - "ResolveEditorCommand"
Cohesion: 0.15
Nodes (13): ADR-0016: No Self-Update Subcommand, Alternatives Considered, Consequences, Context, Decision, Links, Negative, Positive (+5 more)

### Community 129 - "adapter_test.go"
Cohesion: 0.53
Nodes (5): testing.B, benchmarkEngineWithRepos(), BenchmarkStatusReport(), BenchmarkSyncDryRunPlan(), Engine

### Community 130 - "recipe_test.go"
Cohesion: 0.22
Nodes (18): BranchRecipe, CommitRecipe, ConfigRecipe, MetadataRecipe, MissingEntryRecipe, RelationshipRecipe, RepositoryRecipe, UpstreamRecipe (+10 more)

### Community 131 - "Execution Contract"
Cohesion: 0.20
Nodes (9): CLI Process Boundary, CLI Scenario Contract, Determinism Contract, Exact Environment, Execution Contract, Full-release qualification, Git Compatibility Boundary, MCP Stdio Boundary (+1 more)

### Community 132 - "Alternatives Considered"
Cohesion: 0.25
Nodes (6): blockingRunner, countingRunner, mockResponse, mockRunner, sync.Mutex, joinArgs()

### Community 133 - "Decision"
Cohesion: 0.33
Nodes (6): Categories and decision precedence, Decision, Enum placement (no import cycle), Integration evidence, Per-branch model, Surfacing

### Community 134 - "Manpage Plan"
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

### Community 138 - "TestGetReconcileRemoteMismatchFlagIsWired"
Cohesion: 0.17
Nodes (12): Behavioral contract, Contract: `ghcr.io/skaphos/repokeeper`, Divergences from the sting reference, Image properties, Invocation contract, Multiple workspace roots, OCI labels, Read-only inspection — the documented default (+4 more)

### Community 139 - "4. Safety & Policy"
Cohesion: 0.40
Nodes (5): 4.1 Submodules: do not update, 4.2 Working tree safety, 4.3 MCP boundary, 4.4 Mutating command safety, 4. Safety & Policy

### Community 140 - "8. Architecture"
Cohesion: 0.40
Nodes (5): 8.1 Packages (Go), 8.2 CLI framework, 8.3 Concurrency model, 8.4 TUI model (phase 2), 8. Architecture

### Community 141 - "001-distribution-channels/spec.md"
Cohesion: 0.17
Nodes (5): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Distribution Channel Conformance

### Community 142 - "Phase 1 Data Model: Distribution Channel Conformance"
Cohesion: 0.17
Nodes (11): 1. Version Identity, 2. Release Artifact Set, 3. Server Description, 4. Container Workspace Contract, `Info`, Phase 1 Data Model: Distribution Channel Conformance, Resolution rules — FR-001 through FR-005, `Source` (+3 more)

### Community 144 - "Run"
Cohesion: 0.18
Nodes (8): buildMCPLogger(), newMCPEngine(), TestBuildMCPLoggerReturnsOpenFileErrors(), TestBuildMCPLoggerWithoutPathReturnsNopLogger(), TestBuildMCPLoggerWritesFormattedLines(), TestNewMCPEngineWiresLoggerIntoGitAdapter(), log.Logger, fileLogger

### Community 145 - "declaration.go"
Cohesion: 0.10
Nodes (41): CompatibilityResult, EvidenceSummary, MatrixCell, MatrixResult, PinnedInput, Provisioner, ProvisionResult, RootFS (+33 more)

### Community 146 - "[CHECKLIST TYPE] Checklist: [FEATURE NAME]"
Cohesion: 0.40
Nodes (4): [Category 1], [Category 2], [CHECKLIST TYPE] Checklist: [FEATURE NAME], Notes

### Community 147 - "Alternatives Considered"
Cohesion: 0.50
Nodes (4): 1. Keep sync permanently fetch/prune only, 2. Treat configured `update_local` as silent execution permission, 3. Store sync policy in repo-local metadata, Alternatives Considered

### Community 149 - "unsupportedLocalUpdateAdapter"
Cohesion: 0.33
Nodes (8): io/fs.FileMode, resolveTarget(), TestWriteAtomicCreatesFile(), TestWriteAtomicFollowsSymlink(), TestWriteAtomicNoFileAtRequestedMode(), TestWriteAtomicOverwritesExisting(), TestWriteAtomicPreservesExistingMode(), WriteAtomic()

### Community 153 - ".DeleteRepo"
Cohesion: 0.33
Nodes (5): Engine, repoDefaultBranch(), resolveDeleteEntry(), safeRemoveAll(), validateRemoveAllTarget()

### Community 155 - "editReadyMsg"
Cohesion: 0.53
Nodes (5): handleEditReady(), validateEditEntry(), validateEntryKey(), editDoneMsg, editReadyMsg

### Community 159 - "normalize.go"
Cohesion: 0.25
Nodes (4): NormalizeURL(), PrimaryRemote(), TestGitURLNormalizerMatchesGitx(), gitURLNormalizer

### Community 165 - "Git Compatibility Declaration Contract"
Cohesion: 0.25
Nodes (7): Closed support claim, Document shape, Environment and provisioner rules, Git Compatibility Declaration Contract, Release evidence, Reusable executable interface, Tag and recovery semantics

### Community 167 - "Recipe Contract"
Cohesion: 0.25
Nodes (7): Canonical Topology, Materialization Contract, Preflight Contract, Purpose, Ready-State Invariants, Recipe Contract, Reuse Contract

### Community 168 - "assertions_test.go"
Cohesion: 0.31
Nodes (7): RepositorySnapshot, WorkspaceSnapshot, captureWorkspaceSnapshot(), reloadWorkspaceState(), semanticPath(), sortedRegistryPaths(), hashBytes()

### Community 170 - "MCP Tool Matrix Contract"
Cohesion: 0.29
Nodes (6): Canonical Fixtures, Coverage Gate, Current Canonical Cases, Deterministic Call Order, MCP Tool Matrix Contract, Result Shape Rules

### Community 171 - "environment_test.go"
Cohesion: 0.52
Nodes (6): buildChildEnvironment(), copyHostEnvironment(), environmentMap(), executableNullDevice(), lookupEnvironmentFold(), setEnvironmentValue()

### Community 179 - "version_test.go"
Cohesion: 0.11
Nodes (32): Source, advertisedVersion(), orUnavailable(), resolvedBuildInfo(), revisionField(), runVersion(), TestAdvertisedVersion(), TestRevisionField() (+24 more)

### Community 206 - "comment_guard_test.go"
Cohesion: 0.42
Nodes (9): mustReadFile(), TestCodexRemoveEntryAbsentCommentedFileSucceeds(), TestCodexRemoveEntryPresentCommentedFileRefuses(), TestCodexWriteEntryAllowsUncommentedFile(), TestCodexWriteEntryRefusesCommentedFile(), TestGrokWriteEntryRefusesCommentedFile(), TestNewConfigFilesAreOwnerOnly(), TestTomlHasComments() (+1 more)

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

### Community 212 - "grokAdapter"
Cohesion: 0.32
Nodes (4): grokDir(), init(), grokAdapter, grokServer

### Community 213 - "vcs/localbranch_test.go"
Cohesion: 0.48
Nodes (6): enumLine(), TestGitAdapterInspectLocalBranches(), TestGitAdapterInspectLocalBranchesEnumError(), TestGitAdapterInspectLocalBranchesMergedCheckFails(), TestGitAdapterInspectLocalBranchesNoBase(), TestGitAdapterInspectLocalBranchesSkipsPatchWhenDisabled()

### Community 214 - "Contract: Release Artifacts and Channel Verification"
Cohesion: 0.25
Nodes (8): CI additions, Contract: Release Artifacts and Channel Verification, Credential pre-flight (FR-032) — a behavior change, Documentation obligation (FR-015), Linux package contents (FR-012, FR-013), Published artifacts, Single-invocation invariant (FR-031), Verification job (FR-033 – FR-035)

### Community 215 - "ResolveEditorCommand"
Cohesion: 0.47
Nodes (4): ResolveEditorCommand(), contains(), findSubstring(), TestResolveEditorCommand()

### Community 216 - "adapter_test.go"
Cohesion: 0.40
Nodes (3): TestGitAdapterMethods(), TestNewGitAdapterDefaultsRunnerAndCloneErrors(), runnerStub

### Community 217 - "5. Container behavior (FR-024 – FR-029)"
Cohesion: 0.29
Nodes (7): 5. Container behavior (FR-024 – FR-029), 5a. Read-only inspection works, 5b. Mutating tools refuse, and explain, 5c. Read-only tools are unaffected by 5b, 5d. No workspace, 5e. Multi-root pinning, 5f. Native multi-root is unchanged

### Community 219 - "Manpage Plan"
Cohesion: 0.33
Nodes (5): Acceptance Criteria, Generation Approach, Manpage Plan, Release/CI Integration, Target

### Community 220 - "Contract: `repokeeper version`"
Cohesion: 0.33
Nodes (6): Consistency invariant (FR-005), Contract: `repokeeper version`, Exit codes, Invocation, Machine-readable output (FR-006), Test obligations

### Community 221 - "Human-readable output"
Cohesion: 0.33
Nodes (6): Human-readable output, Local build from a clean tree (FR-002), Local build from a modified tree (FR-002), Module install — `go install ...@v0.8.0` (FR-001), Nothing recorded (FR-004), Release build — ldflags stamped (FR-003)

### Community 222 - "Run"
Cohesion: 0.40
Nodes (4): EngineAPI, Run(), RunWithEngine(), TestRunWithNilConfig()

### Community 230 - "Q: ok what else is on the docket for repokeeper that is achievable in one day"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: ok what else is on the docket for repokeeper that is achievable in one day, Source Nodes

### Community 234 - "9. Future: Cross-Machine Registry Sync"
Cohesion: 0.50
Nodes (4): 9.1 Architecture considerations (factor in now), 9.2 Planned sync mechanisms (future, not v1), 9.3 Reconciliation (future), 9. Future: Cross-Machine Registry Sync

## Knowledge Gaps
- **734 isolated node(s):** `common.sh script`, `runtimeStateKey`, `versionJSON`, `github.com/skaphos/repokeeper`, `localUpdateCapable` (+729 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **22 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Entry` connect `Entry` to `github.com/mark3labs/mcp-go/mcp.CallToolRequest`, `metadata_forms.go`, `runDescribeRepo`, `import_test.go`, `index.go`, `.DeleteRepo`, `RepoStatus`, `editReadyMsg`, `SyncResult`, `model/model.go`, `engine/engine.go`, `import.go`, `RepoMetadata`, `newPlanExecEngine`, `Engine`, `.handleSelectRepositories`, `tuiModel`, `import_clone.go`, `importTargetRelativePath`, `mockEngine`, `resolveRepo`, `engine_actions_import_repair_internal_test.go`?**
  _High betweenness centrality (0.027) - this node is a cross-community bridge._
- **Why does `materializeRecipe()` connect `materializeRecipe` to `Save`, `recipe_test.go`, `assertions_test.go`, `environment_test.go`, `context.Context`, `normalize.go`?**
  _High betweenness centrality (0.017) - this node is a cross-community bridge._
- **Why does `Save()` connect `Save` to `Load`, `testing.T`, `dropIgnoredImportEntries`, `materializeRecipe`, `github.com/mark3labs/mcp-go/mcp.CallToolRequest`, `metadata_forms.go`, `runDescribeRepo`, `edit_test.go`, `export_test.go`, `Config`, `.DeleteRepo`, `editReadyMsg`, `Entry`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `Save()` (e.g. with `TestSaveErrorsWhenParentIsFile()` and `TestSaveNilConfigErrors()`) actually correct?**
  _`Save()` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `common.sh script`, `runtimeStateKey`, `versionJSON` to the rest of the system?**
  _734 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Save` be split into smaller, more focused modules?**
  _Cohesion score 0.09890710382513661 - nodes in this community are weakly interconnected._
- **Should `GitAdapter` be split into smaller, more focused modules?**
  _Cohesion score 0.09200603318250378 - nodes in this community are weakly interconnected._