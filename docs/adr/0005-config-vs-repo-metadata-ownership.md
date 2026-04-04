# ADR-0005: Workspace Config vs Repo-Local Metadata Ownership

**Status:** Accepted
**Date:** 2026-04-03
**Author:** Shawn Stratton

## Context

RepoKeeper now has two important YAML surfaces:

- `.repokeeper.yaml`
- `.repokeeper-repo.yaml`

The distinction exists today, but the product is adding more policy, metadata, indexing, and adapter-facing behavior. Without a sharper ownership boundary, configuration and metadata will drift into each other.

## Decision

RepoKeeper uses these ownership rules:

### `.repokeeper.yaml`

Workspace config and machine-local state.

This file owns:

- workspace roots and exclusions
- ignored paths
- embedded registry state
- default execution policy
- machine-local labels and annotations
- concurrency, timeout, remote defaults, and similar operator policy
- future machine-local sync / prune / safety policy

### `.repokeeper-repo.yaml`

Source-controlled repository metadata.

This file owns:

- stable repo identity fields that belong with the repository
- labels/domains intended to be shared
- entrypoints and authoritative paths
- related repositories and relationship types
- provides/capabilities
- agent/runtime metadata intended to be shared as repository context

This file must not own:

- machine-local registry state
- local execution policy
- user-specific paths or workstation-only settings
- confirmation bypass behavior

## Why This Boundary Matters

- workspace config can legitimately differ by machine or operator
- repo-local metadata should be safe to commit and share
- future indexing and adapter work depends on stable source-controlled metadata that is not polluted by local execution concerns

## Write Paths

RepoKeeper should preserve the current write-path distinction:

- `label` and `edit` affect machine-local config / registry state
- `index --write` and the TUI metadata editor affect repo-local metadata

Promotion from machine-local labels into repo-local metadata must remain explicit.

## Consequences

### Positive

- New policy work has a clear home.
- Source-controlled metadata remains portable and reviewable.
- Adapter and indexing work can depend on `.repokeeper-repo.yaml` without inheriting machine-local noise.

### Negative

- Some concepts may need explicit duplication or promotion rather than automatic sharing.

## Alternatives Considered

### 1. Use repo-local metadata as the home for most policy

**Rejected because:** execution policy and registry state are machine-local concerns.

### 2. Keep repo-local metadata narrowly informational with no agent/runtime fields

**Rejected because:** agent/runtime context is a legitimate source-controlled repository concern when it is intended to be shared and generated deterministically.
