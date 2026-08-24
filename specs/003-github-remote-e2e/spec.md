# Feature Specification: GitHub Remote End-to-End Expansion

**Feature Branch**: `test/github-remote-e2e`

**Created**: 2026-08-23

**Status**: Draft

**Input**: User description: "Create a future expansion specification for using GitHub-hosted sample repositories to test RepoKeeper's real network access." Tracked by issue #329.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Verify GitHub Network Workflows (Priority: P1)

A RepoKeeper maintainer can run a network-enabled end-to-end scenario against dedicated public GitHub fixture repositories and verify that RepoKeeper can discover, clone, fetch, identify, and inspect remote-backed repositories through the same executable paths used by real users.

**Why this priority**: The hermetic harness from feature 002 proves executable, protocol, configuration, and local repository behavior, but local remotes cannot expose failures in DNS resolution, TLS, HTTP transport, GitHub redirects, remote URL handling, or real hosted fetch behavior.

**Independent Test**: From an isolated empty workspace, run RepoKeeper against a declared set of public GitHub fixtures, obtain their pinned expected state through normal network operations, and verify repository identity, remotes, refs, tracking, structured output, and resulting local checkout state.

**Acceptance Scenarios**:

1. **Given** an isolated workspace and a reachable public fixture repository, **When** RepoKeeper acquires and inspects the repository through its actual executable, **Then** the resulting checkout matches the declared repository identity, default branch, pinned object, remote URL, and tracking relationship.
2. **Given** multiple public fixtures representing different remote states, **When** RepoKeeper scans and reports them, **Then** each fixture is classified according to its declared expected state and no host-level repository or credential configuration affects the result.
3. **Given** a previously acquired fixture whose remote contains a declared update, **When** RepoKeeper performs its supported fetch-oriented synchronization, **Then** the expected remote-tracking state is observed without changing protected local worktree contents.

---

### User Story 2 - Keep Hosted Fixtures Stable and Auditable (Priority: P2)

A RepoKeeper maintainer can determine exactly which hosted repositories and refs a network scenario depends on, who controls them, what state they promise, and whether an observed difference is an intentional fixture revision or unexpected drift.

**Why this priority**: A network test is only trustworthy when the remote inputs are controlled and versioned. Depending on arbitrary third-party repositories or mutable undocumented branches would turn upstream activity into nondeterministic failures.

**Independent Test**: Compare the declared fixture catalog with the live GitHub repositories and verify ownership, allowed host, expected default branches, pinned object identities, and scenario-specific refs before running RepoKeeper assertions.

**Acceptance Scenarios**:

1. **Given** the network test suite, **When** a maintainer inspects its fixture catalog, **Then** every remote URL, expected repository identity, pinned object, required branch or tag, and scenario purpose is explicit.
2. **Given** a fixture branch or default branch changes without a corresponding reviewed catalog update, **When** validation runs, **Then** the suite reports fixture drift separately from a RepoKeeper behavior failure.
3. **Given** a proposed fixture points outside the approved GitHub organization or includes embedded credentials, **When** the catalog is validated, **Then** it is rejected before any network request occurs.

---

### User Story 3 - Diagnose Network Failures Without Destabilizing Pull Requests (Priority: P3)

A RepoKeeper maintainer can distinguish a RepoKeeper regression from unavailable DNS, TLS, GitHub, proxy, authentication, or rate-limit infrastructure and can see enough evidence to act without exposing secrets.

**Why this priority**: External dependencies inevitably fail independently of RepoKeeper. A network tier that reports every outage as a product regression will be ignored or disabled and therefore provide no lasting value.

**Independent Test**: Exercise successful and controlled failing network cases, confirm that failures are categorized with bounded diagnostics, and run the tier independently of the hermetic pull-request gate until measured reliability supports promotion.

**Acceptance Scenarios**:

1. **Given** GitHub or the network is unavailable, **When** a scenario cannot reach its fixture, **Then** the result identifies the failed network phase and does not claim that a repository classification assertion failed.
2. **Given** a transient transport failure, **When** the configured retry policy is exhausted, **Then** the test stops within its deadline and reports every attempt without retrying deterministic configuration or assertion failures.
3. **Given** the network tier has not yet met its reliability threshold, **When** it fails during scheduled or manually requested execution, **Then** the hermetic pull-request suite remains authoritative and unaffected.

### Edge Cases

- GitHub can return redirects, rate limits, transient server errors, connection resets, or maintenance responses; diagnostics must retain the category and terminal outcome without dumping headers that may contain sensitive data.
- DNS, TLS trust, IPv4/IPv6 routing, and corporate proxy configuration can fail before Git starts transferring repository data; the failing phase must remain identifiable.
- A fixture repository can be renamed, transferred, archived, made private, deleted, or have its default branch changed; catalog validation must surface the specific drift.
- A branch can be force-pushed while a pinned object remains reachable or becomes unavailable; the scenario must distinguish branch drift from missing pinned history.
- A repository can enable Git LFS or add submodules; fixtures must declare those characteristics and the initial scope must not silently download undeclared secondary content.
- A credential helper or ambient token can make a public scenario pass differently on one machine; the network tier must run without inheriting credentials.
- A clone or fetch can partially transfer data and then time out; cleanup and retry behavior must not reuse a corrupt partial checkout as a successful fixture.
- GitHub availability can recover during a run; retry behavior must be bounded and results must record which attempt succeeded.
- Network access can be intentionally disabled in a developer environment; the network tier must have an explicit invocation and must not run as part of the ordinary hermetic test command.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The network tier MUST depend only on dedicated public GitHub fixture repositories controlled by the project maintainers or their organization.
- **FR-002**: The suite MUST maintain a version-controlled fixture catalog declaring each approved URL, repository identity, purpose, expected default branch, pinned object identity, required refs, and expected observable state.
- **FR-003**: Catalog validation MUST reject unapproved hosts, repositories outside the approved ownership boundary, URLs containing credentials, duplicate fixture identities, and incomplete expected state before making network requests.
- **FR-004**: Initial remote access MUST use unauthenticated public HTTPS and MUST NOT require personal tokens, SSH keys, credential helpers, or hosted-service write permissions.
- **FR-005**: Network scenarios MUST reuse the disposable recipe harness and actual RepoKeeper executable established by feature 002 rather than creating a second process or workspace framework.
- **FR-006**: All local configuration, caches, temporary data, clones, and worktrees created by a network scenario MUST remain beneath its disposable root.
- **FR-007**: The network tier MUST exercise actual hosted clone or acquisition, scan, status, and fetch-oriented synchronization behavior and MUST verify structured output plus resulting repository state.
- **FR-008**: At least two hosted fixture repositories or independently declared fixture states MUST be exercised so the suite covers more than a single happy-path remote shape.
- **FR-009**: Assertions MUST cover normalized repository identity, configured remote URL, primary remote, default or checked-out branch, upstream tracking, pinned object identity, and preservation of local worktree contents.
- **FR-010**: At least one CLI network workflow and one MCP network workflow MUST cross the real executable boundary; the MCP workflow MUST exercise every registered tool whose documented behavior performs or depends on remote network access.
- **FR-011**: Network scenarios MUST NOT push, delete, create, rename, change visibility, or otherwise mutate a hosted fixture repository.
- **FR-012**: The suite MUST run without ambient credentials and MUST redact credential-bearing URLs, headers, proxy data, and environment values from all diagnostics.
- **FR-013**: Failures MUST distinguish fixture drift, DNS resolution, connection, TLS, proxy, redirect, authentication, rate limiting, timeout, transfer, process, parsing, and RepoKeeper assertion phases when the underlying evidence permits.
- **FR-014**: Retries MUST be bounded, MUST apply only to explicitly classified transient network failures, and MUST NOT retry invalid configuration, fixture drift, safety refusal, malformed output, or assertion failures.
- **FR-015**: Every network operation and child process MUST have a deadline and deterministic cleanup, including removal of partial clones and termination of stream readers.
- **FR-016**: The network tier MUST be invoked separately from the hermetic test commands and MUST begin as a scheduled or manually requested workflow rather than a required pull-request check.
- **FR-017**: Promotion to a required pull-request check MUST require recorded reliability evidence and an explicit repository decision; it MUST NOT occur implicitly as part of implementing this feature.
- **FR-018**: Fixture changes MUST be reviewed like test-contract changes and MUST update the catalog expectations in the same change that modifies hosted fixture state.
- **FR-019**: The suite MUST document ownership, invocation, expected runtime, network prerequisites, failure categories, fixture-change procedure, and the boundary between provider incidents and RepoKeeper regressions.
- **FR-020**: Arbitrary third-party repositories, authenticated private repositories, SSH transport, remote write testing, hosted API mutation, and destructive remote scenarios MUST remain outside the initial network tier.

### Key Entities

- **Remote Fixture Catalog**: The version-controlled inventory of approved GitHub repositories, pinned objects, expected refs, ownership, scenario purpose, and observable state.
- **Hosted Repository Fixture**: A dedicated public GitHub repository whose controlled state supports one or more network scenarios without accepting writes from the test suite.
- **Network Scenario**: A declared CLI or MCP workflow that combines one or more hosted fixtures with expected process, structured-output, and local repository outcomes.
- **Availability Result**: The categorized outcome of reaching and transferring from a fixture, distinct from RepoKeeper's subsequent behavioral assertions.
- **Fixture Drift Result**: Evidence that a live hosted fixture no longer matches its reviewed catalog contract.
- **Reliability Record**: Aggregated scheduled-run results used to decide whether the network tier is stable enough for stronger CI gating.

### Scope Boundaries

- This feature extends, but does not replace, the hermetic end-to-end suite from feature 002.
- Initial coverage is public, unauthenticated, read-only GitHub HTTPS access.
- The suite may create and mutate local disposable clones but never the hosted fixture repositories.
- Only RepoKeeper behavior that genuinely crosses the network boundary needs duplication here; local-only MCP tools remain fully covered by feature 002.
- GitHub API writes, private repositories, credential lifecycle, SSH authentication, Git LFS content transfer, submodule recursion, release artifacts, and provider portability are future work.
- Scheduled or manual execution is the initial operating mode; required pull-request gating is a separate decision based on reliability evidence.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A network-enabled run acquires and inspects at least two approved public GitHub fixture states through the actual RepoKeeper executable without using credentials.
- **SC-002**: 100% of network fixtures are declared in the reviewed catalog with an approved owner, HTTPS URL, expected repository identity, default branch, pinned object, required refs, and scenario purpose.
- **SC-003**: 100% of files, configuration, caches, clones, and worktree changes created during a run remain beneath its disposable root, and zero hosted repository mutations occur.
- **SC-004**: The CLI and MCP network workflows both verify normalized identity, remote configuration, tracking state, pinned content, and preservation of protected local worktree state.
- **SC-005**: Each individual network attempt terminates within 30 seconds, the complete network tier terminates within two minutes under normal fixture sizes, and no scenario performs more than one transient retry.
- **SC-006**: Every failed network run reports a fixture identifier, operation phase, categorized cause, attempt count, elapsed time, and redacted process diagnostics.
- **SC-007**: Fixture drift is identified before RepoKeeper behavioral assertions and names every mismatched catalog field.
- **SC-008**: The network tier completes at least 30 scheduled runs with at least 95% infrastructure-adjusted success before maintainers consider proposing it as a required pull-request check.
- **SC-009**: Ordinary unit, contract, and hermetic end-to-end commands complete successfully when network access is unavailable or intentionally disabled.
- **SC-010**: A maintainer can update or add a hosted fixture by changing its reviewed catalog entry and documented fixture state without modifying the shared process and isolation harness.

## Assumptions

- Feature 002, the recipe-driven hermetic end-to-end harness, is implemented and stable before this expansion begins.
- The maintainers can create or designate dedicated public repositories under an approved GitHub organization and protect them against accidental test writes.
- Public HTTPS clone and fetch behavior is the most valuable first network boundary because it requires no secret distribution and matches a common user path.
- Fixture repositories remain deliberately small so network tests measure correctness rather than bandwidth or large-repository performance.
- GitHub and network availability are external dependencies; scheduled results retain provider-failure classification rather than treating every unavailable run as a RepoKeeper defect.
- Promotion into required pull-request CI, authenticated access, private repositories, and remote mutation testing require separate review after this read-only tier demonstrates value and reliability.
