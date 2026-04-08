# ADR-0006: Adapter-Facing Contract Stability and Versioning

**Status:** Accepted
**Date:** 2026-04-03
**Author:** Shawn Stratton

## Context

RepoKeeper is moving toward standalone adapters such as IDE plugins, while also tightening MCP into a read-and-plan surface.

That means external consumers need stable contracts that do not depend on importing internal Go packages or scraping human text output.

The main adapter-facing surfaces are expected to be:

- CLI JSON output
- MCP schemas and payloads
- future documented local service surfaces, if added later

## Decision

RepoKeeper treats adapter-facing machine-readable output as an explicit contract surface.

That contract must be:

- deterministic
- documented
- versioned
- clearly separated from human-oriented table output

## Contract Rules

### Human output is not contractual

Human-oriented table, color, and prose output may evolve for usability.

External adapters must not depend on it.

### Machine-readable output is contractual

JSON output and MCP response schemas intended for adapters should be treated as stable interfaces.

Changes to field names, shape, semantics, requiredness, or classification values should be evaluated as contract changes rather than ordinary implementation details.

### Breaking changes must be explicit

Breaking changes to adapter-facing output require:

- documentation
- schema/version notes
- release-note visibility

### MCP and CLI contracts are related but not identical

RepoKeeper may expose similar information through CLI JSON and MCP, but the two surfaces do not need to be byte-for-byte identical.

They do need to be independently well-defined and stable.

## Versioning Guidance

RepoKeeper should provide a clear way for adapters to determine compatibility.

Acceptable approaches include:

- schema envelope version fields in JSON output
- documented compatibility policy tied to RepoKeeper releases
- MCP tool/schema compatibility notes for adapter consumers

The exact mechanism may vary by surface, but compatibility must not depend on guesswork.

## Consequences

### Positive

- Standalone adapters can be built without importing internal packages.
- RepoKeeper can evolve internal implementation details without breaking consumers unnecessarily.
- Contract changes become visible product decisions instead of accidental churn.

### Negative

- Some implementation changes will require extra documentation and compatibility review.
- Output/schema design needs more discipline than purely internal data structures.

## Alternatives Considered

### 1. Treat CLI JSON as best-effort only

**Rejected because:** standalone adapters need a reliable machine-readable contract.

### 2. Require adapters to import internal packages

**Rejected because:** it couples external repos to unstable internals and undermines the standalone adapter model.

### 3. Use MCP as the only stable contract

**Rejected because:** CLI JSON is also a valid adapter surface, and some integrations may prefer it over MCP.
