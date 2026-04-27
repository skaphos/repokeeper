# ADR-0011: Credential and Auth Handling Deferred

**Status:** Accepted
**Date:** 2026-04-26
**Author:** Shawn Stratton

## Context

DESIGN.md §3.2 lists "auto-clone or credential management" as an explicit non-goal. That deferral is a passing line in a non-goals list, not a documented decision with rationale, and it shows up adjacent to several future-work areas that *could* reopen the question:

* `repokeeper add` clones repositories and inherits whatever git's credential helper is configured.
* `repokeeper reconcile` runs `git fetch` and similarly inherits credentials.
* DESIGN.md §9 plans cross-machine registry sync, which has its own auth model question separate from git credential handling.

Without an ADR, this non-goal is easily re-contested every time someone hits a private-repo clone failure or starts thinking about network sync transports. The deferral has rationale worth pinning.

## Decision

RepoKeeper does not own credential management. This applies to:

* SSH keys, personal access tokens, OAuth tokens, GitHub App credentials, or any other credential material for git remotes.
* Credential storage at rest (no RepoKeeper credential store, no encrypted credential file in `.repokeeper.yaml`).
* Credential prompting (no RepoKeeper-owned interactive auth flow).

All git operations RepoKeeper invokes — `clone`, `fetch`, `remote get-url`, etc. — rely entirely on the operator's existing git credential infrastructure: SSH agent, `git-credential-osxkeychain`, `git-credential-libsecret`, `gh auth git-credential`, 1Password SSH agent, etc.

When auth fails, RepoKeeper surfaces the failure as a per-repo error class and continues with other repos. It does not retry, prompt, or attempt fallback credentials.

The future scope where this ADR may be revisited is bounded:

1. **RepoKeeper-to-RepoKeeper network sync** (DESIGN.md §9.2). If RepoKeeper grows a `serve` mode or peer transport for cross-machine sync, a separate ADR must define how nodes authenticate to each other. That is a different problem from git credential handling and should not pull RepoKeeper into the git-credential business.

2. **Registry-in-a-git-repo sync** (DESIGN.md §9.2 alternative). Storing the registry itself in a git repository for sync would inherit the operator's git credentials with no new credential surface. This is the preferred future path precisely because it stays out of credential management.

Both of those future ADRs must reaffirm or explicitly override this decision; neither implicitly grants RepoKeeper a credential surface.

## Consequences

### Positive

* No credential code means no credential-handling vulnerabilities. RepoKeeper has zero attack surface for credential leakage.
* Operators with working git credential setups (the overwhelming common case) need no additional configuration.
* The contract with `git` is narrow: shell out and trust the user's git environment.

### Negative

* First-time clone failures on private repos surface as raw git error output. RepoKeeper cannot help an operator set up credentials and cannot offer a "guided fix" path.
* Operators new to git credential helpers may find the cryptic auth-failure output unhelpful. Mitigation belongs in CONTRIBUTING / setup docs, not in RepoKeeper code.

### Neutral

* This formalizes an existing non-goal rather than changing behavior.
* Error-class output (`auth_failure`, `permission_denied`) remains the integration point for tooling that wants to surface auth issues; the classification itself is not in scope here.

## Alternatives Considered

### 1. Add a thin credential helper wrapper

Wrap one or more git credential helpers behind a RepoKeeper abstraction to give consistent error messages and setup hints.

**Rejected because:** every credential helper RepoKeeper would wrap already exists upstream and is well-documented. Adding indirection adds bug surface, version-skew risk between RepoKeeper and the wrapped helper, and a maintenance cost for no user benefit beyond marginally nicer error text. Setup documentation in CONTRIBUTING covers the same ground without code.

### 2. Document specific recommended setups in this ADR

E.g., "use 1Password SSH agent on macOS, libsecret on Linux."

**Rejected because:** ADRs document decisions, not setup recipes. Recommended setups belong in CONTRIBUTING or `docs/setup/` and can change with the ecosystem without churning this ADR.

### 3. Build a `repokeeper auth doctor` command

A diagnostic that inspects credential helper config and SSH agent state and suggests fixes.

**Rejected for v1 because:** it pulls RepoKeeper into the credential ecosystem in a soft way (no credential storage, but it now has opinions about credential setup). Worth revisiting if first-time-clone friction becomes a measurable problem; until then, deferred.

### 4. Encrypt the registry at rest

Add at-rest encryption for `.repokeeper.yaml` to protect remote URLs and labels.

**Rejected because:** the registry contains no credentials and no secrets — only remote URLs (which are not secrets) and machine-local labels. Encryption would provide no protection against the threat models RepoKeeper is concerned with, while adding setup complexity. If a future feature stores secrets in the registry, that feature's ADR must justify the encryption surface.
