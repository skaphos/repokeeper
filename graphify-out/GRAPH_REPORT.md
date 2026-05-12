# Graph Report - /home/sstratton/work/skaphos/repokeeper  (2026-04-29)

## Corpus Check
- 166 files · ~271,294 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1474 nodes · 3408 edges · 61 communities detected
- Extraction: 55% EXTRACTED · 45% INFERRED · 0% AMBIGUOUS · INFERRED: 1539 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]

## God Nodes (most connected - your core abstractions)
1. `contains()` - 130 edges
2. `New()` - 84 edges
3. `Save()` - 57 edges
4. `DefaultConfig()` - 46 edges
5. `Engine` - 46 edges
6. `tuiModel` - 41 edges
7. `writeFile()` - 41 edges
8. `NewGitAdapter()` - 36 edges
9. `withTestConfig()` - 34 edges
10. `runDescribeRepo()` - 29 edges

## Surprising Connections (you probably didn't know these)
- `TestSplitCSV()` --calls--> `SplitCSV()`  [INFERRED]
  /home/sstratton/work/skaphos/repokeeper/scripts/perf/main_test.go → /home/sstratton/work/skaphos/repokeeper/internal/strutil/csv.go
- `rootRunE()` --calls--> `TestRootRunEHelpFallbackForNonTerminalOutput()`  [INFERRED]
  /home/sstratton/work/skaphos/repokeeper/cmd/repokeeper/root.go → /home/sstratton/work/skaphos/repokeeper/cmd/repokeeper/coverage_boost_test.go
- `persistStatusRegistrySnapshots()` --calls--> `Save()`  [INFERRED]
  /home/sstratton/work/skaphos/repokeeper/cmd/repokeeper/status.go → /home/sstratton/work/skaphos/repokeeper/internal/registry/registry.go
- `loadExistingConfig()` --calls--> `Load()`  [INFERRED]
  /home/sstratton/work/skaphos/repokeeper/cmd/repokeeper/import.go → /home/sstratton/work/skaphos/repokeeper/internal/registry/registry.go
- `TestImportCommandRunERejectsBlankBundleArg()` --calls--> `contains()`  [INFERRED]
  /home/sstratton/work/skaphos/repokeeper/cmd/repokeeper/import_test.go → /home/sstratton/work/skaphos/repokeeper/internal/editor/editor_test.go

## Communities

### Community 0 - "Community 0"
Cohesion: 0.02
Nodes (117): NewGitAdapter(), TestNewGitAdapterDefaultsRunnerAndCloneErrors(), benchmarkEngineWithRepos(), BenchmarkStatusReport(), BenchmarkSyncDryRunPlan(), NewGitErrorClassifier(), NewGitURLNormalizer(), TestGitErrorClassifier() (+109 more)

### Community 1 - "Community 1"
Cohesion: 0.03
Nodes (115): TestEditRepairResetDeleteAddDoneHandlers(), TestHandleAddKey(), TestHandleDeleteConfirmKey(), TestHandleDetailKey(), TestHandleFilterKey(), TestHandleListKey(), TestHandleRepairConfirmKey(), TestHandleResetConfirmKey() (+107 more)

### Community 2 - "Community 2"
Cohesion: 0.03
Nodes (129): PromptYesNo(), TestPromptYesNo(), TestPromptYesNoNoAndEOF(), TestPromptYesNoWriteError(), TestWriteTable(), TestWriteTableNoHeaders(), TestWriteTableWriteError(), WriteTable() (+121 more)

### Community 3 - "Community 3"
Cohesion: 0.07
Nodes (93): TestHandleDetailKeyOpensLabelAndMetadataEditors(), TestHandleLabelEditSavePersistsConfig(), TestHandleLabelEditSaveUsesPathLookupForLocalOnlyRepoID(), TestHandleRepoMetadataEditSaveUsesPathLookupForLocalOnlyRepoID(), TestHandleRepoMetadataEditSaveWritesRepoMetadata(), TestDescribeRunEIncludesRepoMetadata(), TestDescribeRunEPaths(), TestIndexRunEFailsEarlyWhenMetadataExistsWithoutForce() (+85 more)

### Community 4 - "Community 4"
Cohesion: 0.04
Nodes (20): TestGitAdapterMethods(), TestGitURLNormalizer(), TestGitURLNormalizerMatchesGitx(), benchAdapter, NewHgAdapter(), TestHgAdapterEndToEndWithFakeBinary(), TestHgAdapterIsRepoGracefullyHandlesCommandError(), TestHgAdapterSyncCapabilityMetadata() (+12 more)

### Community 5 - "Community 5"
Cohesion: 0.04
Nodes (80): TestCommonPathRoot(), canonicalPathForMatch(), describeCheckoutID(), pathWithinBase(), samePathForMatch(), selectRegistryEntryForDescribe(), splitRepoAndCheckoutSelector(), TestSelectRegistryEntryForDescribe() (+72 more)

### Community 6 - "Community 6"
Cohesion: 0.06
Nodes (38): persistDescribeMetadataSnapshot(), pathCleanCanonical(), TestImportCloneHelperFunctions(), TestPlanImportClonesSkipsAndSuccess(), ignoredPathSet(), ImportCloneCallbacks, importCloneConflict, ImportCloneOptions (+30 more)

### Community 7 - "Community 7"
Cohesion: 0.06
Nodes (23): SplitCSV(), TestSplitCSV(), NewAdapterForSelection(), ParseAdapterSelection(), TestMultiAdapterDelegatesAllMethods(), TestMultiAdapterRoutesByPath(), TestMultiAdapterRoutesCapabilityMethodsByPath(), TestNewAdapterForSelection() (+15 more)

### Community 8 - "Community 8"
Cohesion: 0.08
Nodes (42): colStyledBranch(), colStyledDirty(), colStyledError(), colStyledPlain(), colStyledRepo(), colStyledStatus(), colStyledSynced(), colValueBranch() (+34 more)

### Community 9 - "Community 9"
Cohesion: 0.09
Nodes (32): TestDescribeRepoSubcommandExists(), buildIndexProposal(), detectAuthoritativePaths(), detectLowValuePaths(), detectReadmeEntrypoint(), fallbackMetadataPath(), formatAssignmentDefaults(), formatRelatedRepoDefaults() (+24 more)

### Community 10 - "Community 10"
Cohesion: 0.08
Nodes (30): TestLogOutputWriteFailureLogsError(), TestLogOutputWriteFailureNilError(), TestMarshalToGenericMarshalErrorPath(), TestMarshalToGenericUnmarshalErrorPath(), TestRemoveRegistryEntryByRepoID(), TestResolveCustomColumnValueEdgeCases(), TestRootRunEHelpFallbackForNonTerminalOutput(), TestRowsForCustomColumnsFallbackPaths() (+22 more)

### Community 11 - "Community 11"
Cohesion: 0.08
Nodes (26): errorWriter, editRegistryEntryWithEditor(), resolveEditorCommand(), TestResolveEditorCommandParsesQuotedExecutable(), validateEditedRegistryEntry(), ResolveEditorCommand(), appendRecord(), gitShortCommit() (+18 more)

### Community 12 - "Community 12"
Cohesion: 0.13
Nodes (34): Apply(), discoverMetadataState(), discoverPath(), fileExists(), Load(), metadataConflictFingerprint(), metadataFileFingerprint(), metadataState (+26 more)

### Community 13 - "Community 13"
Cohesion: 0.07
Nodes (13): adapterStub, ApplyPlans(), BuildPlans(), findRegistryEntryIndexForStatus(), ParseReconcileMode(), Plan, primaryRemoteURL(), ReconcileMode (+5 more)

### Community 14 - "Community 14"
Cohesion: 0.07
Nodes (13): buildResult(), detectRepo(), gitdirFromFile(), TestBuildResultBranches(), TestGitdirFromFile(), TestMatchesExcludeWithInvalidPattern(), TestScanDefaultsAndEmptyRoots(), MatchesExclude() (+5 more)

### Community 15 - "Community 15"
Cohesion: 0.12
Nodes (28): CleanFD(), Clone(), ErrorClass, Fetch(), ForEachRefEntry, GitRunner, HasSubmodules(), Head() (+20 more)

### Community 16 - "Community 16"
Cohesion: 0.11
Nodes (28): validateEditEntry(), validateEntryKey(), containsAny(), cloneMetadataStringMap(), currentRegistryEntry(), currentVisibleRepo(), defaultRepoMetadataForTUI(), detectNamedDirsForTUI() (+20 more)

### Community 17 - "Community 17"
Cohesion: 0.13
Nodes (25): Config, ConfigDir(), ConfigPath(), ConfigRoot(), Defaults, EffectiveRoot(), FindNearestConfigPath(), InitConfigPath() (+17 more)

### Community 18 - "Community 18"
Cohesion: 0.1
Nodes (2): runCommand(), HgAdapter

### Community 19 - "Community 19"
Cohesion: 0.18
Nodes (22): allSupportedSkillRoots(), dedupeSortedStrings(), dirExists(), existingSkillRoots(), installedSkillRoots(), requestedSkillRoots(), resolveSkillInstallRoots(), resolveSkillUninstallRoots() (+14 more)

### Community 20 - "Community 20"
Cohesion: 0.13
Nodes (22): TestRepairUpstreamMatchesFilterTable(), init(), TestTrackingBranchFromUpstream(), trackingBranchFromUpstream(), addFormatFlag(), addLabelSelectorFlag(), addNoHeadersFlag(), addRepoFilterFlags() (+14 more)

### Community 21 - "Community 21"
Cohesion: 0.21
Nodes (15): filterRows(), matchesFilter(), TestFilterRowsByBranch(), TestFilterRowsByDisplayLabel(), TestFilterRowsByErrorClass(), TestFilterRowsByLabelValue(), TestFilterRowsByPath(), TestFilterRowsByRepoID() (+7 more)

### Community 22 - "Community 22"
Cohesion: 0.15
Nodes (12): Head, Remote, RepoMetadata, RepoMetadataPaths, RepoMetadataRelatedRepo, RepoStatus, StatusReport, Submodules (+4 more)

### Community 23 - "Community 23"
Cohesion: 0.18
Nodes (1): mockEngine

### Community 24 - "Community 24"
Cohesion: 0.47
Nodes (4): hintForErrorClass(), TestHintForErrorClass_Deterministic(), TestHintForErrorClass_KnownClasses(), TestHintForErrorClass_UnknownClasses()

### Community 25 - "Community 25"
Cohesion: 0.4
Nodes (4): blockingRunner, mockResponse, mockRunner, joinArgs()

### Community 26 - "Community 26"
Cohesion: 0.5
Nodes (2): MockResponse, MockRunner

### Community 27 - "Community 27"
Cohesion: 0.5
Nodes (2): RepoKeeperSkill(), TestRepoKeeperSkillContainsEmbeddedContent()

### Community 28 - "Community 28"
Cohesion: 1.0
Nodes (0): 

### Community 29 - "Community 29"
Cohesion: 1.0
Nodes (0): 

### Community 30 - "Community 30"
Cohesion: 1.0
Nodes (0): 

### Community 31 - "Community 31"
Cohesion: 1.0
Nodes (0): 

### Community 32 - "Community 32"
Cohesion: 1.0
Nodes (0): 

### Community 33 - "Community 33"
Cohesion: 1.0
Nodes (0): 

### Community 34 - "Community 34"
Cohesion: 1.0
Nodes (0): 

### Community 35 - "Community 35"
Cohesion: 1.0
Nodes (0): 

### Community 36 - "Community 36"
Cohesion: 1.0
Nodes (0): 

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (0): 

### Community 38 - "Community 38"
Cohesion: 1.0
Nodes (1): EngineAPI

### Community 39 - "Community 39"
Cohesion: 1.0
Nodes (0): 

### Community 40 - "Community 40"
Cohesion: 1.0
Nodes (0): 

### Community 41 - "Community 41"
Cohesion: 1.0
Nodes (0): 

### Community 42 - "Community 42"
Cohesion: 1.0
Nodes (0): 

### Community 43 - "Community 43"
Cohesion: 1.0
Nodes (0): 

### Community 44 - "Community 44"
Cohesion: 1.0
Nodes (0): 

### Community 45 - "Community 45"
Cohesion: 1.0
Nodes (0): 

### Community 46 - "Community 46"
Cohesion: 1.0
Nodes (0): 

### Community 47 - "Community 47"
Cohesion: 1.0
Nodes (0): 

### Community 48 - "Community 48"
Cohesion: 1.0
Nodes (0): 

### Community 49 - "Community 49"
Cohesion: 1.0
Nodes (0): 

### Community 50 - "Community 50"
Cohesion: 1.0
Nodes (0): 

### Community 51 - "Community 51"
Cohesion: 1.0
Nodes (0): 

### Community 52 - "Community 52"
Cohesion: 1.0
Nodes (0): 

### Community 53 - "Community 53"
Cohesion: 1.0
Nodes (0): 

### Community 54 - "Community 54"
Cohesion: 1.0
Nodes (0): 

### Community 55 - "Community 55"
Cohesion: 1.0
Nodes (0): 

### Community 56 - "Community 56"
Cohesion: 1.0
Nodes (0): 

### Community 57 - "Community 57"
Cohesion: 1.0
Nodes (0): 

### Community 58 - "Community 58"
Cohesion: 1.0
Nodes (0): 

### Community 59 - "Community 59"
Cohesion: 1.0
Nodes (0): 

### Community 60 - "Community 60"
Cohesion: 1.0
Nodes (0): 

## Knowledge Gaps
- **79 isolated node(s):** `benchmarkMetric`, `benchmarkRunRecord`, `labelRequirement`, `runtimeStateKey`, `runtimeState` (+74 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 28`** (2 nodes): `move.go`, `init()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 29`** (2 nodes): `init.go`, `init()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 30`** (2 nodes): `init()`, `delete.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (2 nodes): `label.go`, `init()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (2 nodes): `init()`, `add.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (2 nodes): `repokeeper_suite_test.go`, `TestRepokeeperSuite()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (2 nodes): `version.go`, `init()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (2 nodes): `status_characterization_test.go`, `statusIntPtr()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (2 nodes): `vcs_suite_test.go`, `TestVcs()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (2 nodes): `TestConfig()`, `config_suite_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (2 nodes): `engine.go`, `EngineAPI`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (2 nodes): `tui_suite_test.go`, `TestTui()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (2 nodes): `termstyle_suite_test.go`, `TestTermstyle()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (2 nodes): `TestGitx()`, `gitx_suite_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (2 nodes): `registry_suite_test.go`, `TestRegistry()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (2 nodes): `model_suite_test.go`, `TestModel()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (2 nodes): `remotemismatch_suite_test.go`, `TestRemoteMismatch()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (2 nodes): `tableutil_suite_test.go`, `TestTableutil()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (2 nodes): `sortutil_suite_test.go`, `TestSortutil()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (2 nodes): `TestDiscovery()`, `discovery_suite_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (2 nodes): `TestEngine()`, `engine_suite_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (2 nodes): `TestCliio()`, `cliio_suite_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (2 nodes): `strutil_suite_test.go`, `TestStrutil()`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (1 nodes): `sync_characterization_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (1 nodes): `export_characterization_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 53`** (1 nodes): `import_characterization_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (1 nodes): `config_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (1 nodes): `view.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (1 nodes): `parse_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 57`** (1 nodes): `normalize_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (1 nodes): `gitx_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (1 nodes): `model_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (1 nodes): `discovery_test.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 0` to `Community 1`, `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 7`, `Community 10`, `Community 11`, `Community 13`, `Community 14`, `Community 17`, `Community 25`?**
  _High betweenness centrality (0.277) - this node is a cross-community bridge._
- **Why does `contains()` connect `Community 3` to `Community 0`, `Community 1`, `Community 2`, `Community 5`, `Community 8`, `Community 9`, `Community 10`, `Community 12`, `Community 15`, `Community 16`, `Community 19`, `Community 21`, `Community 27`?**
  _High betweenness centrality (0.230) - this node is a cross-community bridge._
- **Why does `Save()` connect `Community 3` to `Community 0`, `Community 1`, `Community 2`, `Community 6`, `Community 12`, `Community 16`, `Community 17`?**
  _High betweenness centrality (0.085) - this node is a cross-community bridge._
- **Are the 127 inferred relationships involving `contains()` (e.g. with `TestLabelCommandShowsLabels()` and `TestLabelCommandSetAndRemove()`) actually correct?**
  _`contains()` has 127 INFERRED edges - model-reasoned connections that need verification._
- **Are the 79 inferred relationships involving `New()` (e.g. with `TestRemoteMismatchReconcileHelpers()` and `TestPopulateExportBranches()`) actually correct?**
  _`New()` has 79 INFERRED edges - model-reasoned connections that need verification._
- **Are the 56 inferred relationships involving `Save()` (e.g. with `writeLabelsTestConfig()` and `TestLabelCommandUpdatesLocalLabelsOnly()`) actually correct?**
  _`Save()` has 56 INFERRED edges - model-reasoned connections that need verification._
- **Are the 44 inferred relationships involving `DefaultConfig()` (e.g. with `writeLabelsTestConfig()` and `TestLabelCommandUpdatesLocalLabelsOnly()`) actually correct?**
  _`DefaultConfig()` has 44 INFERRED edges - model-reasoned connections that need verification._