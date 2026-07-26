# Graph Report - feature+distribution-channels  (2026-07-26)

## Corpus Check
- 319 files · ~254,035 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 3288 nodes · 6709 edges · 228 communities (198 shown, 30 thin omitted)
- Extraction: 84% EXTRACTED · 16% INFERRED · 0% AMBIGUOUS · INFERRED: 1085 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `6a2f3f40`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- DefaultConfig
- GitAdapter
- copyFixture
- root.go
- MultiAdapter
- Config
- withInstallEnv
- resolveRepo
- WriteAtomic
- model_internal_test.go
- tuiModel
- server.go
- metadata_forms.go
- runDescribeRepo
- actions_internal_test.go
- columns.go
- HgAdapter
- status.go
- import_test.go
- multiStubAdapter
- common.sh
- renderListView
- index.go
- export_test.go
- helpers_test.go
- repokeeper/sync.go
- benchAdapter
- model/model.go
- New
- Registry
- Tasks: [FEATURE NAME]
- Milestones
- withConfigAndCWD
- RepoStatus
- repometa_test.go
- NewGitErrorClassifier
- ApplyPlans
- renderDetailView
- stubAdapter
- ADR-0002: Branch Switch and Checkout Workflow Boundaries
- planAdapter
- engine/engine.go
- adapterStub
- import.go
- repometa.go
- perf/main.go
- README.md
- engine_more_internal_test.go
- RepoKeeper
- 5.1 CLI commands (v1)
- ADR-0001: MCP Server for Agent-Native Repository Querying and Planning
- filterRows
- addFormatFlag
- index_test.go
- writeStatusTable
- ADR-0006: Adapter-Facing Contract Stability and Versioning
- RepoKeeper
- runtime_test.go
- mcpserver_test.go
- .handleDeleteConfirmKey
- Changelog
- MCP Server Setup
- newPlanExecEngine
- sync_test.go
- WriteTable
- runUninstall
- ADR-0003: Sync Policy and Execution Modes
- ADR-0004: Prune Workflow Boundaries and Safety Model
- Classify
- LabelsMatchSelector
- ADR-0007: Release Binary Publishing and Homebrew Distribution
- Engine
- readJSONDoc
- opencodeAdapter
- ADR-0005: Workspace Config vs Repo-Local Metadata Ownership
- ADR-0008: MCP Install Tooling for Supported Agent Runtimes
- ADR-0011: Credential and Auth Handling Deferred
- Verification Checklist (with Evidence Placeholders)
- .handleSelectRepositories
- mockEngine
- Feature Specification: [FEATURE NAME]
- cloneMetadataMap
- status_helpers_test.go
- RepoKeeper — Design Spec
- ADR-0010: Repo ID Normalization Stability
- ADR-0014: Local Branch Prune-Safety Classification Model
- ADR-0015: Branch Retention and Protection Policy
- countingRunner
- wrappers_test.go
- registry_test.go
- newMCPEngine
- GitHub Copilot Instructions for RepoKeeper
- mockEngine
- prepareEditCmd
- Core Principles
- Core Principles
- Repository Guidelines
- install.go
- 5.3 Kubectl-Style CLI Alignment (Milestone 6+)
- Command Notes
- SyncResult
- hintForErrorClass
- .handleGetWorkspaceConfig
- Release Process
- command_parsing_test.go
- Contributing Guidelines
- ADR-0009: Replace Release Please with skaphos/actions
- repairResolveTargetBranch
- Scope
- inspectEntryCmd
- tui/sync.go
- Implementation Plan: [FEATURE]
- 6. Data Model
- Manual Verification Checklist (MCP Phase 4 / SKA-470)
- engine_actions_import_repair_internal_test.go
- SplitCSV
- PrintHeaders
- vcs/localbranch_test.go
- TestAddCommandWithAbsoluteTargetDoesNotReRootUnderCWD
- Colorize
- cloneImportedEntriesWithProgress
- importTargetRelativePath
- runInstallList
- 9. Stretch Goals
- Decision
- ADR-0012: Release Please Owns Release Notes
- ADR-0013: GoReleaser Owns the GitHub Release
- Decision
- ResolveEditorCommand
- adapter_test.go
- exportEntriesWithEmbeddedCredentials
- status_prune_test.go
- Alternatives Considered
- Decision
- Manpage Plan
- Installation
- ClassifyError
- resetRepoCmd
- TestGetReconcileRemoteMismatchFlagIsWired
- 4. Safety & Policy
- 8. Architecture
- Available tools
- MockRunner
- Run
- TestMainInvokesExecute
- [CHECKLIST TYPE] Checklist: [FEATURE NAME]
- Alternatives Considered
- unsupportedLocalUpdateAdapter
- matchesStatusFilter
- check-coverage.sh
- TestRepokeeperSuite
- 3. Goals
- TestCliio
- TestConfig
- TestDiscovery
- TestEngine
- TestGitx
- TestMCPServer
- tools_metadata.go
- TestModel
- TestRegistry
- TestRemoteMismatch
- TestSelector
- TestSortutil
- TestStrutil
- TestTableutil
- TestTermstyle
- TestTui
- TestRunWithNilConfig
- TestVcs
- run_generation
- version.go
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
- repokeeper/label_test.go
- grokAdapter
- vcs/localbranch_test.go
- Contract: Release Artifacts and Channel Verification
- ResolveEditorCommand
- adapter_test.go
- 5. Container behavior (FR-024 – FR-029)
- TestResolveAbsoluteTargetPath
- Specification Quality Checklist: Distribution Channel Conformance
- Contract: `repokeeper version`
- Human-readable output
- Run

## God Nodes (most connected - your core abstractions)
1. `RepoStatus` - 88 edges
2. `Registry` - 77 edges
3. `DefaultConfig()` - 67 edges
4. `New()` - 46 edges
5. `Config` - 44 edges
6. `withTestConfig()` - 43 edges
7. `NewGitAdapter()` - 43 edges
8. `Engine` - 42 edges
9. `copyFixture()` - 41 edges
10. `tuiModel` - 38 edges

## Surprising Connections (you probably didn't know these)
- `writeEmptyConfig()` --calls--> `DefaultConfig()`  [INFERRED]
  cmd/repokeeper/commands_more_test.go → internal/config/config.go
- `TestRootRunEInteractiveMissingRegistry()` --calls--> `DefaultConfig()`  [INFERRED]
  cmd/repokeeper/coverage_boost_test.go → internal/config/config.go
- `runDescribeRepo()` --calls--> `EffectiveRoot()`  [INFERRED]
  cmd/repokeeper/describe.go → internal/config/config.go
- `runDescribeRepo()` --calls--> `ResolveConfigPath()`  [INFERRED]
  cmd/repokeeper/describe.go → internal/config/config.go
- `runDescribeRepo()` --calls--> `SeedRepoMetadataStatus()`  [INFERRED]
  cmd/repokeeper/describe.go → internal/registry/registry.go

## Import Cycles
- None detected.

## Communities (228 total, 30 thin omitted)

### Community 0 - "DefaultConfig"
Cohesion: 0.23
Nodes (33): T, TestDescribeRunEIncludesRepoMetadata(), TestDescribeRunEPaths(), TestIndexReposRunEPreviewsAndWritesSelectedRepos(), TestIndexReposRunERequiresPromoteFlag(), TestIndexRunECanPromoteLocalLabels(), TestIndexRunEFailsEarlyWhenMetadataExistsWithoutForce(), TestIndexRunEForceResolvesDualMetadataFiles() (+25 more)

### Community 1 - "GitAdapter"
Cohesion: 0.05
Nodes (53): ForEachRefEntry, GitRunner, LocalBranchInfo, Runner, stubRunner, CleanFD(), Clone(), Fetch() (+45 more)

### Community 2 - "copyFixture"
Cohesion: 0.05
Nodes (80): copyFixture(), T, TestClaudeConfigPathProject(), TestClaudeConfigPathUser(), TestClaudeDetectFalseWhenNeither(), TestClaudeDetectTrueWhenDotClaudeDir(), TestClaudeDetectTrueWhenDotClaudeJson(), TestClaudeNameAndDetect() (+72 more)

### Community 3 - "root.go"
Cohesion: 0.06
Nodes (70): T, TestCommonPathRoot(), TestFlagGettersBranchCoverage(), TestLogOutputWriteFailureLogsError(), TestLogOutputWriteFailureNilError(), TestMarshalToGenericMarshalErrorPath(), TestMarshalToGenericUnmarshalErrorPath(), TestNewSyncProgressWriter() (+62 more)

### Community 4 - "MultiAdapter"
Cohesion: 0.06
Nodes (38): Command, selectedAdapterForCommand(), Options, Result, stubLBAdapter, buildResult(), detectRepo(), gitdirFromFile() (+30 more)

### Community 5 - "Config"
Cohesion: 0.21
Nodes (18): findNearestConfigPath(), isSharedDir(), canonicalPath(), T, TestConfigDirVariants(), TestFindNearestConfigPathDoesNotWalkIntoSharedDir(), TestFindNearestConfigPathErrorsOnInvalidCWD(), TestFindNearestConfigPathIgnoresDirectoryNamedLikeConfig() (+10 more)

### Community 6 - "withInstallEnv"
Cohesion: 0.08
Nodes (58): Buffer, T, resetInstallListFlags(), runInstallListWithFlags(), TestInstallListAllNotRegistered(), TestInstallListInvalidScope(), TestInstallListJSON(), TestInstallListProjectCodexUnsupported() (+50 more)

### Community 7 - "resolveRepo"
Cohesion: 0.40
Nodes (7): CallToolRequest, CallToolResult, Context, MCPServer, optionalStringSliceArg(), parseSyncOptions(), requireStrictBoolArg()

### Community 8 - "WriteAtomic"
Cohesion: 0.17
Nodes (13): codexServersMap(), Entry, FileMode, readTOMLDoc(), refuseIfTOMLComments(), skipTOMLDelim(), skipTOMLSingleLine(), tomlHasComments() (+5 more)

### Community 9 - "model_internal_test.go"
Cohesion: 0.06
Nodes (53): Cmd, Context, EngineAPI, Entry, SyncResult, tuiModel, T, TestEscInListClearsFilter() (+45 more)

### Community 10 - "tuiModel"
Cohesion: 0.20
Nodes (8): repoIDRowCount(), executeSyncCmd(), Cmd, KeyPressMsg, Model, tuiModel, startStatusRefresh(), Msg

### Community 11 - "server.go"
Cohesion: 0.07
Nodes (53): explainReadOnly(), T, TestExplainReadOnlyNamesCauseAndRemedy(), TestExplainReadOnlyPassesThroughUnrelatedErrors(), TestMutatingToolsRemainAdvertised(), TestSerializeToolLeavesSuccessfulCallsAlone(), TestSerializeToolTranslatesReadOnlyFailures(), T (+45 more)

### Community 12 - "metadata_forms.go"
Cohesion: 0.15
Nodes (31): cloneMetadataStringMap(), currentRegistryEntry(), currentVisibleRepo(), defaultRepoMetadataForTUI(), detectNamedDirsForTUI(), detectReadmeEntrypointForTUI(), formatRelatedReposCSV(), formatStringMapCSV() (+23 more)

### Community 13 - "runDescribeRepo"
Cohesion: 0.13
Nodes (33): canonicalPathForMatch(), describeCheckoutID(), Command, Entry, pathWithinBase(), persistDescribeMetadataSnapshot(), runDescribeRepo(), samePathForMatch() (+25 more)

### Community 14 - "actions_internal_test.go"
Cohesion: 0.16
Nodes (23): T, TestEditRepairResetDeleteAddDoneHandlers(), TestHandleAddKey(), TestHandleDeleteConfirmKey(), TestHandleDeleteConfirmKeyResetsOffset(), TestHandleDetailKey(), TestHandleDetailKeyOpensLabelAndMetadataEditors(), TestHandleFilterKey() (+15 more)

### Community 15 - "columns.go"
Cohesion: 0.13
Nodes (31): colStyledBranch(), colStyledDirty(), colStyledError(), colStyledRepo(), colStyledStatus(), colStyledSynced(), colValueBranch(), colValueDirty() (+23 more)

### Community 16 - "HgAdapter"
Cohesion: 0.11
Nodes (12): Context, rejectFlagLike(), runCommand(), Context, NewHgAdapter(), T, TestHgAdapterCloneRejectsFlagLikeArgs(), TestHgAdapterEndToEndWithFakeBinary() (+4 more)

### Community 17 - "status.go"
Cohesion: 0.12
Nodes (32): TestDivergedAdviceAndTable(), TestWriteStatusDetailsAndHelpers(), buildDivergedAdvice(), countGoneRepos(), displayTrackingStatusNoColor(), divergedReasonAndAction(), enrichReportWithRegistryMetadata(), findRegistryMetadataEntry() (+24 more)

### Community 18 - "import_test.go"
Cohesion: 0.13
Nodes (31): cloneImportedRepos(), T, TestCloneImportedReposMarksFailedCloneAsMissing(), TestCloneImportedReposNoopWithoutRegistry(), TestCloneImportedReposRejectsDuplicateTargets(), TestCloneImportedReposRejectsUnsafeTargets(), TestCloneImportedReposReportsSpecificTargetConflicts(), TestCloneImportedReposSkipsLocalEntriesWithoutRemoteURL() (+23 more)

### Community 19 - "multiStubAdapter"
Cohesion: 0.16
Nodes (8): Context, T, TestMultiAdapterDelegatesAllMethods(), TestMultiAdapterRoutesByPath(), TestMultiAdapterRoutesCapabilityMethodsByPath(), TestNewAdapterForSelection(), TestParseAdapterSelection(), multiStubAdapter

### Community 20 - "common.sh"
Cohesion: 0.09
Nodes (15): check-prerequisites.sh script, check_dir(), check_file(), get_feature_paths(), get_repo_root(), has_jq(), _persist_feature_json(), resolve_specify_init_dir() (+7 more)

### Community 21 - "renderListView"
Cohesion: 0.15
Nodes (20): TestViewsAndRendering(), renderDivider(), tuiModel, renderDeleteConfirmView(), duplicateSyncRepoCounts(), findRepoStatusIndex(), SyncResult, sameRepoCheckout() (+12 more)

### Community 22 - "index.go"
Cohesion: 0.14
Nodes (23): buildIndexProposal(), detectAuthoritativePaths(), detectLowValuePaths(), detectReadmeEntrypoint(), fallbackMetadataPath(), formatAssignmentDefaults(), formatRelatedRepoDefaults(), Command (+15 more)

### Community 23 - "export_test.go"
Cohesion: 0.10
Nodes (29): commonPathRoot(), exportEntriesWithEmbeddedCredentials(), exportEntryPath(), Context, inferRegistrySharedRoot(), populateExportBranches(), prepareRegistryForExport(), T (+21 more)

### Community 24 - "helpers_test.go"
Cohesion: 0.12
Nodes (30): T, TestColorizeGuardBranches(), TestConfirmSyncExecution(), TestConfirmSyncExecutionEOF(), TestConfirmWithPrompt(), TestDescribeSyncAction(), TestDescribeSyncActionAdditionalBranches(), TestDisplayRepoPathPrefersCWDThenRoot() (+22 more)

### Community 25 - "repokeeper/sync.go"
Cohesion: 0.21
Nodes (21): displayRepoPath(), describeSyncAction(), Command, Mutex, SyncResult, persistSyncRegistryAfterCheckoutMissing(), syncLocalUpdateSkipReason(), syncPlanNeedsConfirmation() (+13 more)

### Community 27 - "model/model.go"
Cohesion: 0.15
Nodes (18): localBranchNamesByCategory(), Context, Engine, upstreamStatusFromSignal(), Time, HintForReason(), ParsePruneCategory(), repoContextResponse (+10 more)

### Community 28 - "New"
Cohesion: 0.17
Nodes (30): testResponse, testRunner, unsupportedLocalUpdateAdapter, Context, T, TestEngineGuardErrors(), TestExecuteSyncPlanAppliesActions(), TestExecuteSyncPlanCloneAction() (+22 more)

### Community 29 - "Registry"
Cohesion: 0.14
Nodes (18): matchesStatusFilter(), canonicalRegistryPath(), checkoutIDFromEntry(), cloneRepoMetadata(), cloneStringMap(), cloneStringSlice(), defaultCheckoutIDFromPath(), Time (+10 more)

### Community 30 - "Tasks: [FEATURE NAME]"
Cohesion: 0.07
Nodes (26): Dependencies & Execution Order, Format: `[ID] [P?] [Story] Description`, Implementation for User Story 1, Implementation for User Story 2, Implementation for User Story 3, Implementation Strategy, Incremental Delivery, MVP First (User Story 1 Only) (+18 more)

### Community 31 - "Milestones"
Cohesion: 0.07
Nodes (27): CI pipeline (GitHub Actions), Coverage requirements, Integration tests, Linting & code quality, Milestone 0 — Repo skeleton, Milestone 10 — 1.0 Readiness & Release Reset, Milestone 11 — MCP Server for Agent-Native Querying, Milestone 1 — Discovery + registry (+19 more)

### Community 32 - "withConfigAndCWD"
Cohesion: 0.50
Nodes (3): dirtyBehindAdapter, Context, Worktree

### Community 33 - "RepoStatus"
Cohesion: 0.14
Nodes (23): Duration, ImportCloneCallbacks, OutcomeKind, SyncOptions, SyncResult, SyncResultCallback, SyncStartCallback, executedNonCloneOutcome() (+15 more)

### Community 34 - "repometa_test.go"
Cohesion: 0.23
Nodes (25): Apply(), Load(), T, mustMarshalRepoMetadata(), mustMetadataFingerprint(), rewriteFilePreservingFingerprint(), testAbsolutePath(), TestApplyCachesDualFileConflict() (+17 more)

### Community 35 - "NewGitErrorClassifier"
Cohesion: 0.11
Nodes (29): importCloneConflict, ImportCloneOptions, ImportClonePlan, ImportCloneSkip, ImportCloneTarget, pathCleanCanonical(), TestImportCloneHelperFunctions(), findImportCloneConflicts() (+21 more)

### Community 36 - "ApplyPlans"
Cohesion: 0.08
Nodes (22): Context, Engine, RemoteMismatchPlan, RemoteMismatchReconcileMode, ParseRemoteMismatchReconcileMode(), ApplyPlans(), BuildPlans(), findRegistryEntryIndexForStatus() (+14 more)

### Community 37 - "renderDetailView"
Cohesion: 0.17
Nodes (22): TestDetailViewRendersUnknownStaleMarkerOnInspectionError(), TestHelpersAndPathResolution(), TestRenderDetailViewDeterministicMapOrder(), TestRenderDetailViewStripsControlCharactersFromRepoMetadata(), TestSanitizeMetadataText(), TestTUIUsesSharedAndLocalLabelsDistinctly(), colStyledPlain(), colValueDelta() (+14 more)

### Community 38 - "stubAdapter"
Cohesion: 0.16
Nodes (3): stubAdapter, Context, Remote

### Community 39 - "ADR-0002: Branch Switch and Checkout Workflow Boundaries"
Cohesion: 0.09
Nodes (22): 1. Fold checkout into sync, 2. Keep checkout permanently out of scope, 3. Allow branch switching through MCP, ADR-0002: Branch Switch and Checkout Workflow Boundaries, Alternatives Considered, Branch switch / checkout, CLI, Consequences (+14 more)

### Community 40 - "planAdapter"
Cohesion: 0.15
Nodes (3): planAdapter, Context, Mutex

### Community 41 - "engine/engine.go"
Cohesion: 0.16
Nodes (17): FilterKind, localUpdateCapable, PullRebasePolicyOptions, ScanOptions, filterRegistryEntriesByIgnoredPaths(), filterRequiresInspect(), filterStatus(), findRegistryEntryForStatus() (+9 more)

### Community 42 - "adapterStub"
Cohesion: 0.06
Nodes (33): Correction made during implementation, Dependencies & Execution Order, Documentation, Execution Notes, Format: `[ID] [P?] [Story] Description`, Format validation, Highest-risk tasks, Implementation for User Story 1 (+25 more)

### Community 43 - "import.go"
Cohesion: 0.20
Nodes (19): Entry, inferredCheckoutIDFromPath(), loadExistingConfig(), mergeImportedRegistry(), mergePolicyPreflightSkips(), mergeRegistryMatchIndex(), normalizeImportedBundle(), parseImportConflictPolicy() (+11 more)

### Community 44 - "repometa.go"
Cohesion: 0.19
Nodes (20): canonicalize(), discoverMetadataState(), discoverPath(), fileExists(), metadataConflictFingerprint(), metadataFileFingerprint(), normalize(), normalizeMap() (+12 more)

### Community 45 - "perf/main.go"
Cohesion: 0.22
Nodes (17): benchmarkMetric, benchmarkRunRecord, appendRecord(), gitShortCommit(), loadLastRecord(), main(), parseBenchmarkMetrics(), printSummary() (+9 more)

### Community 46 - "README.md"
Cohesion: 0.12
Nodes (10): Acceptance Criteria, Generation Approach, Manpage Plan, Release/CI Integration, Target, Manual installation, RepoKeeper Agent Skill, Crediting libraries correctly (+2 more)

### Community 47 - "engine_more_internal_test.go"
Cohesion: 0.18
Nodes (18): matchesProtectedBranch(), T, TestExecuteSyncPlanAppliesPlannedActions(), TestExecuteSyncPlanStopsOnFailureWhenConfigured(), TestFilterStatusDefaultKind(), TestFilterStatusKindsAndSorts(), TestFindRegistryEntryForStatusAndReplace(), TestHasRemoteMismatchCases() (+10 more)

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
Cohesion: 0.27
Nodes (16): filterRows(), matchesFilter(), T, TestFilterRowsByBranch(), TestFilterRowsByDisplayLabel(), TestFilterRowsByErrorClass(), TestFilterRowsByLabelValue(), TestFilterRowsByPath() (+8 more)

### Community 52 - "addFormatFlag"
Cohesion: 0.25
Nodes (15): init(), addFormatFlag(), addLabelSelectorFlag(), addNoHeadersFlag(), addRepoFilterFlags(), addUpstreamRepairFilterFlag(), addVCSFlag(), Command (+7 more)

### Community 53 - "index_test.go"
Cohesion: 0.26
Nodes (16): T, TestFormatAssignmentDefaultsSortsKeys(), TestFormatRelatedRepoDefaultsSortsValues(), TestGuessRepoMetadataDefaultsClonesExistingMetadata(), TestIndexQuestionerUsesDefaultsAndParsers(), TestParseIndexAssignments(), TestParseIndexListSortsAndTrims(), TestUnifiedDiffEmptyWhenIdentical() (+8 more)

### Community 54 - "writeStatusTable"
Cohesion: 0.25
Nodes (15): writeStatusTable(), adaptiveCellLimit(), adaptiveCellLimitForWidth(), Command, tableWidth(), captureStatusTableOutputAtWidth(), captureSyncTableOutputAtWidth(), T (+7 more)

### Community 55 - "ADR-0006: Adapter-Facing Contract Stability and Versioning"
Cohesion: 0.12
Nodes (16): 1. Treat CLI JSON as best-effort only, 2. Require adapters to import internal packages, 3. Use MCP as the only stable contract, ADR-0006: Adapter-Facing Contract Stability and Versioning, Alternatives Considered, Breaking changes must be explicit, Consequences, Context (+8 more)

### Community 56 - "RepoKeeper"
Cohesion: 0.12
Nodes (16): Avoid these mistakes, Check health first, Core rules, Discovery workflow, Execute safe updates, Good agent response pattern, Initialization workflow, Labeling workflow (+8 more)

### Community 57 - "runtime_test.go"
Cohesion: 0.19
Nodes (23): init(), All(), ByName(), register(), SelectionFromFlags(), T, names(), TestAllReturnsSortedCopy() (+15 more)

### Community 58 - "mcpserver_test.go"
Cohesion: 0.16
Nodes (14): callTool(), expectResourceError(), expectResourceSuccess(), CallToolResult, intPtr(), newTestConfig(), newTestRegistry(), newTestStatusReport() (+6 more)

### Community 59 - ".handleDeleteConfirmKey"
Cohesion: 0.40
Nodes (6): TestModalHelpers(), KeyPressMsg, tuiModel, isModalNav(), modalMoveLeft(), modalMoveRight()

### Community 60 - "Changelog"
Cohesion: 0.12
Nodes (15): [0.6.1](https://github.com/skaphos/repokeeper/compare/v0.6.0...v0.6.1) (2026-04-03), [0.7.0](https://github.com/skaphos/repokeeper/compare/v0.6.1...v0.7.0) (2026-04-09), [0.7.1](https://github.com/skaphos/repokeeper/compare/v0.7.0...v0.7.1) (2026-04-18), [1.2.0](https://github.com/skaphos/repokeeper/compare/v1.1.0...v1.2.0) (2026-05-31), [1.3.0](https://github.com/skaphos/repokeeper/compare/v1.2.0...v1.3.0) (2026-06-22), [1.3.1](https://github.com/skaphos/repokeeper/compare/v1.3.0...v1.3.1) (2026-07-12), Bug Fixes, Bug Fixes (+7 more)

### Community 61 - "MCP Server Setup"
Cohesion: 0.07
Nodes (29): 1. Tool Discovery (all 14 tools visible), 2. Read-only Tools (safe, no side effects), 3. Planning Tools (dry-run only), 4. Mutation Tools + Safety Gates, 5. Structured Content + Error Handling, 6. Client Compatibility Smoke Test, Available tools, CLI skill fallback (+21 more)

### Community 62 - "newPlanExecEngine"
Cohesion: 0.26
Nodes (14): syncStep, Engine, Engine, Entry, SyncResult, T, newPlanExecEngine(), TestApplyRemoteMismatchPlansUsesInjectedAdapter() (+6 more)

### Community 63 - "sync_test.go"
Cohesion: 0.21
Nodes (17): T, TestHandleSyncDone(), TestHandleSyncPlanError(), TestHandleSyncPlanKeyCancel(), TestHandleSyncPlanKeyCancelViaEnterOnCancel(), TestHandleSyncPlanKeyConfirm(), TestHandleSyncPlanSuccess(), TestHandleSyncProgressKey_WhenDoneReturnsToList() (+9 more)

### Community 64 - "WriteTable"
Cohesion: 0.24
Nodes (12): errorWriter, Reader, Writer, PromptYesNo(), T, TestPromptYesNo(), TestPromptYesNoNoAndEOF(), TestPromptYesNoWriteError() (+4 more)

### Community 65 - "runUninstall"
Cohesion: 0.17
Nodes (27): TestAddCommandWithAbsoluteTargetDoesNotReRootUnderCWD(), T, mustRunGit(), TestAddDeleteWithRegistryOverride(), TestAddValidationMutuallyExclusiveFlags(), TestDeleteCancelledByPrompt(), TestDeleteTrackingOnlyAddsIgnoredPathAndKeepsRepo(), TestEditRequiresEditorConfiguration() (+19 more)

### Community 66 - "ADR-0003: Sync Policy and Execution Modes"
Cohesion: 0.13
Nodes (14): ADR-0003: Sync Policy and Execution Modes, CLI and TUI, Configuration Boundary, Consequences, Context, Decision, Execution Model, Interface Boundaries (+6 more)

### Community 67 - "ADR-0004: Prune Workflow Boundaries and Safety Model"
Cohesion: 0.13
Nodes (15): 1. Keep prune as an implicit part of sync, 2. Treat local branch prune as a simple merged/not-merged check, ADR-0004: Prune Workflow Boundaries and Safety Model, Alternatives Considered, CLI and TUI, Consequences, Context, Decision (+7 more)

### Community 68 - "Classify"
Cohesion: 0.22
Nodes (13): Classify(), Time, isBaseBranch(), isStale(), MatchesProtected(), boolPtr(), T, Time (+5 more)

### Community 69 - "LabelsMatchSelector"
Cohesion: 0.24
Nodes (12): filterBulkIndexEntriesByLabels(), filterStatusReportByLabels(), filterStatusReportByLocalLabels(), CallToolRequest, CallToolResult, Context, MCPServer, LabelsMatchSelector() (+4 more)

### Community 70 - "ADR-0007: Release Binary Publishing and Homebrew Distribution"
Cohesion: 0.14
Nodes (14): 1. Goreleaser owns the GitHub release and release body; release-please bumps version; the `release-please.yml` workflow pushes the tag, 2. Keep `homebrew_casks:`, but make it install cleanly on Apple Silicon, 3. Backfill v0.7.0 assets (one-off), ADR-0007: Release Binary Publishing and Homebrew Distribution, Consequences, Context, Decision, Implementation plan (+6 more)

### Community 71 - "Engine"
Cohesion: 0.15
Nodes (19): T, TestActionsResetDeleteCloneAndRegister(), TestExecuteImportClonesSuccessFailureAndSkips(), TestPlanImportClonesGuardsAndErrors(), TestPlanImportClonesSkipsAndSuccess(), TestRemoteMismatchWrapperFunctions(), TestRepairHelpers(), TestRepairUpstreamScenarios() (+11 more)

### Community 72 - "readJSONDoc"
Cohesion: 0.12
Nodes (14): Entry, FileMode, init(), readJSONDoc(), writeJSONDoc(), checkJsonc(), Entry, init() (+6 more)

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
Cohesion: 0.14
Nodes (15): CallToolRequest, CallToolResult, Context, MCPServer, mergeLabels(), buildMatchReason(), enrichAnnotations(), enrichLabels() (+7 more)

### Community 79 - "mockEngine"
Cohesion: 0.27
Nodes (3): Context, SyncResult, mockEngine

### Community 80 - "Feature Specification: [FEATURE NAME]"
Cohesion: 0.15
Nodes (12): Assumptions, Edge Cases, Feature Specification: [FEATURE NAME], Functional Requirements, Key Entities *(include if feature involves data)*, Measurable Outcomes, Requirements *(mandatory)*, Success Criteria *(mandatory)* (+4 more)

### Community 81 - "cloneMetadataMap"
Cohesion: 0.24
Nodes (9): mergePromotedLabels(), cloneMetadataMap(), normalizeMetadataMap(), parseMetadataAssignments(), parseMetadataKeys(), T, TestParseMetadataAssignments(), TestParseMetadataKeys() (+1 more)

### Community 82 - "status_helpers_test.go"
Cohesion: 0.32
Nodes (11): buildStatusJSONOutput(), T, TestDesignDocNamesStatusJSONAPIVersion(), TestRelatedReposString(), TestSanitizeForDisplayStripsControlAndANSISequences(), TestStatusJSONOutputFlagsGoneRepairSuggestion(), TestStatusJSONOutputIncludesAPIVersion(), TestStatusJSONOutputNilReportIsSafe() (+3 more)

### Community 83 - "RepoKeeper — Design Spec"
Cohesion: 0.17
Nodes (12): 10. Open Questions (explicitly deferred), 1. Summary, 2. Problem Statement, 7.0 Git CLI Strategy & Compatibility Matrix, 7. Git Operations (Engine Contract), 9.1 Architecture considerations (factor in now), 9.2 Planned sync mechanisms (future, not v1), 9.3 Reconciliation (future) (+4 more)

### Community 84 - "ADR-0010: Repo ID Normalization Stability"
Cohesion: 0.17
Nodes (11): 1. Treat normalization as implementation detail, 2. Version every normalization rule from day one, 3. Hash the normalized URL instead of storing it, ADR-0010: Repo ID Normalization Stability, Alternatives Considered, Consequences, Context, Decision (+3 more)

### Community 85 - "ADR-0014: Local Branch Prune-Safety Classification Model"
Cohesion: 0.17
Nodes (12): 1. Treat "upstream gone" as `probably_safe` without patch-equivalence, 2. Use `git branch --merged` (reachability) as the sole integration signal, 3. Put the enums in a dependency-free `internal/prune` package, 4. Classify only the current branch, reusing existing state, ADR-0014: Local Branch Prune-Safety Classification Model, Alternatives Considered, Consequences, Context (+4 more)

### Community 86 - "ADR-0015: Branch Retention and Protection Policy"
Cohesion: 0.17
Nodes (12): 1. Workspace-global `base_branch` (empty ⇒ `Defaults.MainBranch`), 2. Reuse `--protected-branches` for prune and union it in, 3. Store branch policy in `.repokeeper-repo.yaml`, 4. Define the policy type inside the classifier and have config embed it, ADR-0015: Branch Retention and Protection Policy, Alternatives Considered, Consequences, Context (+4 more)

### Community 87 - "countingRunner"
Cohesion: 0.17
Nodes (17): editRegistryEntryWithEditor(), findRegistryEntryIndex(), Command, Entry, resolveEditorCommand(), T, TestEditRunEFailsWithoutEditor(), TestEditRunERejectsInvalidEditedYAML() (+9 more)

### Community 88 - "wrappers_test.go"
Cohesion: 0.30
Nodes (11): T, TestCloneDoesNotTrimURLOrPath(), TestCloneRejectsFlagLikeArgs(), TestCloneWrapper(), TestPushWrapper(), TestSetRemoteURLRejectsFlagLikeArgs(), TestSetRemoteURLWrapper(), TestSetUpstreamRejectsFlagLikeArgs() (+3 more)

### Community 89 - "registry_test.go"
Cohesion: 0.30
Nodes (11): T, TestFindByRepoIDAndCheckoutIDBackfillsLegacyEntries(), TestFindEntriesByRepoID(), TestFindEntriesByRepoIDReturnsAllCheckoutMatches(), TestLegacyEntryBackfillsCheckoutID(), TestLookupsDoNotMutateEntries(), TestUpsertAllowsDuplicateRepoIDWithDistinctCheckoutID(), TestUpsertCollapsesDuplicatePathAcrossRepoIDs() (+3 more)

### Community 90 - "newMCPEngine"
Cohesion: 0.18
Nodes (13): T, TestEditKeyHandlersAndMetadataFieldMutationHelpers(), TestSaveLabelEditCmdDoesNotRaceWithConcurrentRegistryReads(), TestSaveRepoMetadataEditCmdDoesNotRaceWithConcurrentRegistryReads(), TestSaveRepoMetadataEditCmdSkipsUnchangedMetadata(), appendRepoMetadataField(), Context, EngineAPI (+5 more)

### Community 91 - "GitHub Copilot Instructions for RepoKeeper"
Cohesion: 0.18
Nodes (10): Codebase Shape, Commit and Branch Guidance, Documentation Expectations, GitHub Copilot Instructions for RepoKeeper, Go and Repository Conventions, Pull Request Instructions, Safety Rules, Testing Expectations (+2 more)

### Community 92 - "mockEngine"
Cohesion: 0.27
Nodes (4): StatusOptions, Context, SyncResult, mockEngine

### Community 93 - "prepareEditCmd"
Cohesion: 0.29
Nodes (10): Cmd, Entry, Model, tuiModel, handleEditReady(), prepareEditCmd(), validateEditEntry(), validateEntryKey() (+2 more)

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
Cohesion: 0.17
Nodes (13): desiredInstallEntry(), Command, Entry, Writer, parseInstallScope(), printClaudePermissionsBlock(), printManualSnippets(), runInstall() (+5 more)

### Community 98 - "5.3 Kubectl-Style CLI Alignment (Milestone 6+)"
Cohesion: 0.20
Nodes (10): 5.2 TUI command (phase 2), 5.3.1 Command shape, 5.3.2 Output contracts, 5.3.3 Styling and color policy (intentional delta vs kubectl), 5.3.4 Filter and selector direction, 5.3.5 Migration strategy, 5.3 Kubectl-Style CLI Alignment (Milestone 6+), 5. User Experience (+2 more)

### Community 99 - "Command Notes"
Cohesion: 0.15
Nodes (13): Command Notes, Global Flags, `repokeeper add`, RepoKeeper Commands, `repokeeper describe`, `repokeeper edit`, `repokeeper get`, `repokeeper index` (+5 more)

### Community 100 - "SyncResult"
Cohesion: 0.27
Nodes (10): cloneAndRegisterCmd(), defaultClonePath(), Cmd, Context, EngineAPI, tuiModel, renderAddView(), repoNameFromURL() (+2 more)

### Community 101 - "hintForErrorClass"
Cohesion: 0.29
Nodes (7): Engine, SyncResult, hintForErrorClass(), T, TestHintForErrorClass_Deterministic(), TestHintForErrorClass_KnownClasses(), TestHintForErrorClass_UnknownClasses()

### Community 102 - ".handleGetWorkspaceConfig"
Cohesion: 0.24
Nodes (8): cfgDefault(), CallToolRequest, CallToolResult, Context, MCPServer, intDefault(), configDefaults, workspaceConfigResponse

### Community 103 - "Release Process"
Cohesion: 0.17
Nodes (12): 1. Land releasable commits on `main`, 2. Run local release checks, 3. Review and merge the release PR, 4. Tag push + GitHub release automation, 5. Verify the release, Channel verification, Credentials fail the release; they never skip a channel, Notes (+4 more)

### Community 104 - "command_parsing_test.go"
Cohesion: 0.25
Nodes (10): T, TestParseRemoteMismatchReconcileModeTable(), TestRepairUpstreamMatchesFilterTable(), TestShouldStreamSyncResults(), TestSyncProgressMessageKinds(), TestSyncResultNeedsConfirmationTable(), repairUpstreamMatchesFilter(), remoteMismatchReconcileMode (+2 more)

### Community 105 - "Contributing Guidelines"
Cohesion: 0.20
Nodes (9): Branching and Commits, Coding Standards, Contributing Guidelines, Development Setup, Graphify, Pull Requests, Release Process, Safety Expectations (+1 more)

### Community 106 - "ADR-0009: Replace Release Please with skaphos/actions"
Cohesion: 0.12
Nodes (16): Action pin, ADR-0009: Replace Release Please with skaphos/actions, Alternatives considered, Bot identity, Consequences, Context, Decision, Files added (+8 more)

### Community 107 - "repairResolveTargetBranch"
Cohesion: 0.28
Nodes (7): RepairUpstreamResult, Context, Engine, Entry, repairNeedsUpstream(), repairResolveTargetBranch(), repairDoneMsg

### Community 109 - "inspectEntryCmd"
Cohesion: 0.23
Nodes (9): Entry, resolveRepo(), CallToolResult, T, newStructuredListResult(), CallToolRequest, CallToolResult, Context (+1 more)

### Community 110 - "tui/sync.go"
Cohesion: 0.28
Nodes (8): buildSyncPlanCmd(), Cmd, Context, EngineAPI, SyncResult, syncDoneMsg, syncPlanMsg, syncProgressMsg

### Community 111 - "Implementation Plan: [FEATURE]"
Cohesion: 0.22
Nodes (8): Complexity Tracking, Constitution Check, Documentation (this feature), Implementation Plan: [FEATURE], Project Structure, Source Code (repository root), Summary, Technical Context

### Community 112 - "6. Data Model"
Cohesion: 0.25
Nodes (8): 6.1 Repo identity, 6.2.1 Machine config, 6.2.2 Registry (embedded in machine config by default), 6.2 Config files, 6.3 Status JSON schema (v1), 6.4 Sync (reconcile) JSON schema, 6. Data Model, JSON output schema stability policy

### Community 113 - "Manual Verification Checklist (MCP Phase 4 / SKA-470)"
Cohesion: 0.18
Nodes (18): Command, needsUpstreamRepair(), parseUpstreamRepairFilter(), T, TestNeedsUpstreamRepair(), TestParseUpstreamRepairFilter(), TestRepairUpstreamMatchesFilter(), TestRepairUpstreamRunECancelledConfirmation() (+10 more)

### Community 114 - "engine_actions_import_repair_internal_test.go"
Cohesion: 0.33
Nodes (8): Entry, LessRepoIDPath(), SortRegistryEntries(), SortRepoStatuses(), T, TestLessRepoIDPath(), TestSortRegistryEntries(), TestSortRepoStatuses()

### Community 115 - "SplitCSV"
Cohesion: 0.32
Nodes (5): ParseFieldSelectorFilter(), ResolveRepoFilter(), SplitCSV(), T, TestSplitCSV()

### Community 116 - "PrintHeaders"
Cohesion: 0.36
Nodes (6): Writer, New(), PrintHeaders(), T, TestNew(), TestPrintHeaders()

### Community 117 - "vcs/localbranch_test.go"
Cohesion: 0.53
Nodes (4): Context, MCPServer, ReadResourceRequest, ResourceContents

### Community 118 - "TestAddCommandWithAbsoluteTargetDoesNotReRootUnderCWD"
Cohesion: 0.22
Nodes (18): BranchPolicy, Config, Defaults, configDir(), ConfigPath(), ConfigRoot(), EffectiveRoot(), InitConfigPath() (+10 more)

### Community 119 - "Colorize"
Cohesion: 0.33
Nodes (5): TestColorizeAndTrackingDisplayBranches(), displayTrackingStatus(), Colorize(), T, TestColorize()

### Community 120 - "cloneImportedEntriesWithProgress"
Cohesion: 0.26
Nodes (11): executeImportClonePlanWithProgress(), Command, SyncResult, cloneImportedEntriesWithProgress(), cloneImportedReposWithProgress(), Command, Entry, SyncResult (+3 more)

### Community 121 - "importTargetRelativePath"
Cohesion: 0.44
Nodes (8): dropIgnoredImportEntries(), ignoredPathSet(), importTargetRelativePath(), relativeFromAbsolutePath(), relFromRootBasename(), cleanRelativePath(), isAbsoluteLikePath(), normalizePathLikeInput()

### Community 122 - "runInstallList"
Cohesion: 0.38
Nodes (5): Command, Writer, runInstallList(), writeInstallListTable(), listRow

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
Nodes (5): B, benchmarkEngineWithRepos(), BenchmarkStatusReport(), BenchmarkSyncDryRunPlan(), Engine

### Community 130 - "exportEntriesWithEmbeddedCredentials"
Cohesion: 0.24
Nodes (8): TestViewDispatchAndOtherHelpers(), Time, relativeTime(), tuiModel, renderListView(), visibleRows(), trackingRowStyle(), Style

### Community 131 - "status_prune_test.go"
Cohesion: 0.60
Nodes (5): T, pruneRepoFixture(), TestStatusJSONIncludesLocalBranches(), TestWriteStatusDetailsPruneClassification(), TestWriteStatusDetailsPruneInspectionError()

### Community 132 - "Alternatives Considered"
Cohesion: 0.26
Nodes (7): blockingRunner, countingRunner, mockResponse, mockRunner, Context, Mutex, joinArgs()

### Community 133 - "Decision"
Cohesion: 0.33
Nodes (6): Categories and decision precedence, Decision, Enum placement (no import cycle), Integration evidence, Per-branch model, Surfacing

### Community 134 - "Manpage Plan"
Cohesion: 0.35
Nodes (11): plainAdapter, Engine, T, newEngineWith(), TestInspectLocalBranchesBaseUnresolved(), TestInspectLocalBranchesClassifies(), TestInspectLocalBranchesPatchEquivalenceGatedByRequireMerged(), TestInspectLocalBranchesRemoteQualifiedBaseOverride() (+3 more)

### Community 135 - "Installation"
Cohesion: 0.25
Nodes (8): Container image, From local source checkout, From release binaries, From source, Homebrew (cask), Installation, Linux packages, Migration from old Homebrew formula install

### Community 136 - "ClassifyError"
Cohesion: 0.40
Nodes (4): ClassifyError(), containsAny(), T, TestClassifyError()

### Community 137 - "resetRepoCmd"
Cohesion: 0.12
Nodes (15): TestActionCmdsPropagateProvidedContext(), deleteRepoCmd(), Cmd, Context, EngineAPI, Cmd, Context, EngineAPI (+7 more)

### Community 138 - "TestGetReconcileRemoteMismatchFlagIsWired"
Cohesion: 0.17
Nodes (12): Behavioral contract, Contract: `ghcr.io/skaphos/repokeeper`, Divergences from the sting reference, Image properties, Invocation contract, Multiple workspace roots, OCI labels, Read-only inspection — the documented default (+4 more)

### Community 139 - "4. Safety & Policy"
Cohesion: 0.40
Nodes (5): 4.1 Submodules: do not update, 4.2 Working tree safety, 4.3 MCP boundary, 4.4 Mutating command safety, 4. Safety & Policy

### Community 140 - "8. Architecture"
Cohesion: 0.40
Nodes (5): 8.1 Packages (Go), 8.2 CLI framework, 8.3 Concurrency model, 8.4 TUI model (phase 2), 8. Architecture

### Community 142 - "Available tools"
Cohesion: 0.17
Nodes (11): 1. Version Identity, 2. Release Artifact Set, 3. Server Description, 4. Container Workspace Contract, `Info`, Phase 1 Data Model: Distribution Channel Conformance, Resolution rules — FR-001 through FR-005, `Source` (+3 more)

### Community 143 - "MockRunner"
Cohesion: 0.50
Nodes (3): MockResponse, MockRunner, Context

### Community 144 - "Run"
Cohesion: 0.29
Nodes (8): buildMCPLogger(), Engine, newMCPEngine(), T, TestBuildMCPLoggerReturnsOpenFileErrors(), TestBuildMCPLoggerWithoutPathReturnsNopLogger(), TestBuildMCPLoggerWritesFormattedLines(), TestNewMCPEngineWiresLoggerIntoGitAdapter()

### Community 145 - "TestMainInvokesExecute"
Cohesion: 0.40
Nodes (3): main(), T, TestMainInvokesExecute()

### Community 146 - "[CHECKLIST TYPE] Checklist: [FEATURE NAME]"
Cohesion: 0.40
Nodes (4): [Category 1], [Category 2], [CHECKLIST TYPE] Checklist: [FEATURE NAME], Notes

### Community 147 - "Alternatives Considered"
Cohesion: 0.50
Nodes (4): 1. Keep sync permanently fetch/prune only, 2. Treat configured `update_local` as silent execution permission, 3. Store sync policy in repo-local metadata, Alternatives Considered

### Community 149 - "unsupportedLocalUpdateAdapter"
Cohesion: 0.36
Nodes (9): FileMode, resolveTarget(), T, TestWriteAtomicCreatesFile(), TestWriteAtomicFollowsSymlink(), TestWriteAtomicNoFileAtRequestedMode(), TestWriteAtomicOverwritesExisting(), TestWriteAtomicPreservesExistingMode() (+1 more)

### Community 150 - "matchesStatusFilter"
Cohesion: 0.24
Nodes (9): optionalStringMapArg(), addRepoResponse, removeRepoResponse, scanRepoEntry, scanResponse, setLabelsResponse, syncPlanEntry, syncResultEntry (+1 more)

### Community 153 - "3. Goals"
Cohesion: 0.67
Nodes (3): 3.1 Functional goals (v1 / "80%"), 3.2 Non-goals (defer), 3. Goals

### Community 179 - "version.go"
Cohesion: 0.10
Nodes (36): Info, Source, advertisedVersion(), Command, orUnavailable(), resolvedBuildInfo(), revisionField(), T (+28 more)

### Community 206 - "comment_guard_test.go"
Cohesion: 0.51
Nodes (10): T, mustReadFile(), TestCodexRemoveEntryAbsentCommentedFileSucceeds(), TestCodexRemoveEntryPresentCommentedFileRefuses(), TestCodexWriteEntryAllowsUncommentedFile(), TestCodexWriteEntryRefusesCommentedFile(), TestGrokWriteEntryRefusesCommentedFile(), TestNewConfigFilesAreOwnerOnly() (+2 more)

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

### Community 211 - "repokeeper/label_test.go"
Cohesion: 0.54
Nodes (7): T, TestLabelCommandAcceptsAbsolutePathSelector(), TestLabelCommandRejectsUnsupportedFormat(), TestLabelCommandSetAndRemove(), TestLabelCommandShowsLabels(), TestLabelCommandUpdatesLocalLabelsOnly(), writeLabelsTestConfig()

### Community 212 - "grokAdapter"
Cohesion: 0.32
Nodes (4): grokDir(), init(), grokAdapter, grokServer

### Community 213 - "vcs/localbranch_test.go"
Cohesion: 0.54
Nodes (7): enumLine(), T, TestGitAdapterInspectLocalBranches(), TestGitAdapterInspectLocalBranchesEnumError(), TestGitAdapterInspectLocalBranchesMergedCheckFails(), TestGitAdapterInspectLocalBranchesNoBase(), TestGitAdapterInspectLocalBranchesSkipsPatchWhenDisabled()

### Community 214 - "Contract: Release Artifacts and Channel Verification"
Cohesion: 0.25
Nodes (8): CI additions, Contract: Release Artifacts and Channel Verification, Credential pre-flight (FR-032) — a behavior change, Documentation obligation (FR-015), Linux package contents (FR-012, FR-013), Published artifacts, Single-invocation invariant (FR-031), Verification job (FR-033 – FR-035)

### Community 215 - "ResolveEditorCommand"
Cohesion: 0.38
Nodes (5): ResolveEditorCommand(), contains(), findSubstring(), T, TestResolveEditorCommand()

### Community 216 - "adapter_test.go"
Cohesion: 0.33
Nodes (5): Context, T, TestGitAdapterMethods(), TestNewGitAdapterDefaultsRunnerAndCloneErrors(), runnerStub

### Community 217 - "5. Container behavior (FR-024 – FR-029)"
Cohesion: 0.29
Nodes (7): 5. Container behavior (FR-024 – FR-029), 5a. Read-only inspection works, 5b. Mutating tools refuse, and explain, 5c. Read-only tools are unaffected by 5b, 5d. No workspace, 5e. Multi-root pinning, 5f. Native multi-root is unchanged

### Community 218 - "TestResolveAbsoluteTargetPath"
Cohesion: 0.33
Nodes (3): resolveAbsoluteTargetPath(), T, TestResolveAbsoluteTargetPath()

### Community 219 - "Specification Quality Checklist: Distribution Channel Conformance"
Cohesion: 0.33
Nodes (5): Content Quality, Feature Readiness, Notes, Requirement Completeness, Specification Quality Checklist: Distribution Channel Conformance

### Community 220 - "Contract: `repokeeper version`"
Cohesion: 0.33
Nodes (6): Consistency invariant (FR-005), Contract: `repokeeper version`, Exit codes, Invocation, Machine-readable output (FR-006), Test obligations

### Community 221 - "Human-readable output"
Cohesion: 0.33
Nodes (6): Human-readable output, Local build from a clean tree (FR-002), Local build from a modified tree (FR-002), Module install — `go install ...@v0.8.0` (FR-001), Nothing recorded (FR-004), Release build — ldflags stamped (FR-003)

### Community 222 - "Run"
Cohesion: 0.60
Nodes (4): Context, EngineAPI, Run(), RunWithEngine()

## Knowledge Gaps
- **610 isolated node(s):** `common.sh script`, `runtimeStateKey`, `versionJSON`, `github.com/skaphos/repokeeper`, `localUpdateCapable` (+605 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **30 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `RepoStatus` connect `renderDetailView` to `exportEntriesWithEmbeddedCredentials`, `status_prune_test.go`, `MultiAdapter`, `model_internal_test.go`, `metadata_forms.go`, `runDescribeRepo`, `columns.go`, `status.go`, `renderListView`, `matchesStatusFilter`, `helpers_test.go`, `model/model.go`, `Registry`, `withConfigAndCWD`, `RepoStatus`, `repometa_test.go`, `ApplyPlans`, `stubAdapter`, `engine/engine.go`, `repometa.go`, `engine_more_internal_test.go`, `filterRows`, `mockEngine`, `countingRunner`, `newMCPEngine`, `mockEngine`, `repairResolveTargetBranch`, `Manual Verification Checklist (MCP Phase 4 / SKA-470)`, `engine_actions_import_repair_internal_test.go`?**
  _High betweenness centrality (0.084) - this node is a cross-community bridge._
- **Why does `Registry` connect `Registry` to `model_internal_test.go`, `tuiModel`, `metadata_forms.go`, `runDescribeRepo`, `status.go`, `export_test.go`, `helpers_test.go`, `New`, `RepoStatus`, `NewGitErrorClassifier`, `ApplyPlans`, `engine/engine.go`, `import.go`, `addFormatFlag`, `mcpserver_test.go`, `.handleSelectRepositories`, `mockEngine`, `countingRunner`, `newMCPEngine`, `mockEngine`, `prepareEditCmd`, `Run`, `inspectEntryCmd`, `TestAddCommandWithAbsoluteTargetDoesNotReRootUnderCWD`, `cloneImportedEntriesWithProgress`?**
  _High betweenness centrality (0.082) - this node is a cross-community bridge._
- **Why does `Config` connect `TestAddCommandWithAbsoluteTargetDoesNotReRootUnderCWD` to `DefaultConfig`, `Manpage Plan`, `runDescribeRepo`, `Run`, `status.go`, `import_test.go`, `index.go`, `repokeeper/sync.go`, `New`, `Registry`, `RepoStatus`, `NewGitErrorClassifier`, `engine/engine.go`, `import.go`, `mcpserver_test.go`, `mockEngine`, `countingRunner`, `mockEngine`, `prepareEditCmd`, `Run`, `repairResolveTargetBranch`, `cloneImportedEntriesWithProgress`, `importTargetRelativePath`?**
  _High betweenness centrality (0.078) - this node is a cross-community bridge._
- **Are the 64 inferred relationships involving `DefaultConfig()` (e.g. with `TestDescribeRunEIncludesRepoMetadata()` and `TestIndexReposRunEPreviewsAndWritesSelectedRepos()`) actually correct?**
  _`DefaultConfig()` has 64 INFERRED edges - model-reasoned connections that need verification._
- **Are the 34 inferred relationships involving `New()` (e.g. with `.Run()` and `.Run()`) actually correct?**
  _`New()` has 34 INFERRED edges - model-reasoned connections that need verification._
- **What connects `common.sh script`, `runtimeStateKey`, `versionJSON` to the rest of the system?**
  _610 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `GitAdapter` be split into smaller, more focused modules?**
  _Cohesion score 0.050793650793650794 - nodes in this community are weakly interconnected._