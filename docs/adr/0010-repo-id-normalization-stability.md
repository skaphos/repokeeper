# ADR-0010: Repo ID Normalization Stability

**Status:** Accepted
**Date:** 2026-04-26
**Author:** Shawn Stratton

## Context

`repo_id` is the cross-machine join key for the RepoKeeper registry. DESIGN.md §6.1 specifies how it is derived from a remote URL, and DESIGN.md §9 plans cross-machine reconciliation built on top of that join key. Export/import already exists in v1; future network-based sync extends the same data model.

For that data model to remain portable, the rules that turn a remote URL into a `repo_id` must produce the same answer on every machine and across versions. If two machines disagree about whether `git@github.com:Org/Repo.git` and `https://github.com/Org/Repo` are the same repo, every cross-machine workflow breaks silently — and silently is the worst failure mode for a registry-as-truth design.

The rules themselves are small (DESIGN.md §6.1):

* strip protocol and user prefix
* lowercase host
* preserve path case
* strip trailing `.git`
* strip trailing slashes

This ADR is about the *stability* of those rules, not their content.

## Decision

The normalization rules in DESIGN.md §6.1 are a contract surface, not an implementation detail. Tools, importers, exporters, and future network-sync transports are entitled to assume that `normalize(remote_url)` produces the same `repo_id` today and a year from now for the same input.

Concretely:

* The rules MUST NOT change in a way that would re-derive different `repo_id` values for the same remote URL across versions, except through one of the migration paths below.
* Bug fixes that *narrow* normalization (e.g., previously two URLs collided into one `repo_id` when they should not have) require a documented migration path.
* Normalization changes that broaden mapping (previously two URLs were distinct `repo_id`s and should be merged) require a documented migration path.

Acceptable migration paths for any future change:

1. **Additive only.** New rules apply only to new repo identities, never re-derive existing ones.
2. **Explicit migration tool.** A one-shot rewrite (`repokeeper migrate registry --normalize`) that rewrites entries with operator confirmation.
3. **Schema-version bump.** Registry gains a normalization-rules version field; importers detect mismatches and refuse or migrate explicitly.

Out of scope for this stability guarantee:

* `checkout_id`. It is machine-local by design and does not flow across machines as a join key.
* Path-based or `repo_id@checkout_id` selector resolution within a single registry. That is implementation detail and may change.
* Display formatting of `repo_id` in human-oriented output. The string stored in the registry is canonical; rendered output may differ.

## Consequences

### Positive

* Cross-machine export/import remains lossless without coordinating tool versions across machines.
* Future network sync (DESIGN.md §9.2) can treat `repo_id` as a stable join key without inventing its own identity layer.
* Registry files written today remain readable by future versions without forced migration.

### Negative

* Bug fixes to normalization rules now require migration tooling rather than a one-line code change.
* The current rule set is locked in earlier than other implementation choices, before extensive cross-machine usage.

### Neutral

* The rule set is small and intentionally conservative; the cost of locking it in is low because there is little surface to regret.
* This ADR does not preclude future versioning of the normalization rules; it requires that any version transition be explicit and migration-aware.

## Alternatives Considered

### 1. Treat normalization as implementation detail

Allow normalization to evolve freely with the implementation.

**Rejected because:** the registry is meant to be portable across machines and across time. Treating the join-key derivation as implementation detail makes cross-machine reconciliation depend on every machine running the same RepoKeeper version, which defeats the design.

### 2. Version every normalization rule from day one

Store a normalization-rules version with every `repo_id` entry, even when no rule has ever changed.

**Rejected for v1 because:** there is no current evidence that rules need to change. Adding the machinery now is speculative complexity; the migration paths in this ADR are sufficient when (and if) a change is actually proposed.

### 3. Hash the normalized URL instead of storing it

Use a content hash of the normalized URL as `repo_id` instead of the normalized URL itself.

**Rejected because:** human-readable `repo_id` is a usability win across CLI flags, TUI, log output, and metadata files. The stability of the *string* is the contract; hashing trades readability for no additional safety, since the hash is only as stable as the rules that feed it.
