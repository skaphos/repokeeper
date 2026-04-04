# ADR-0004: Prune Workflow Boundaries and Safety Model

**Status:** Accepted
**Date:** 2026-04-03
**Author:** Shawn Stratton

## Context

RepoKeeper already treats stale remote-tracking refs and local branch cleanup as related hygiene concerns, but they do not have the same risk profile.

- remote-tracking prune is mostly mechanical
- local branch deletion is a real destructive action that requires classification and operator review

If both are treated as a single hidden cleanup behavior inside sync, the safety model becomes opaque.

## Decision

RepoKeeper defines prune as a first-class workflow separate from sync.

Prune planning may include two categories:

1. **remote-tracking prune**
2. **local branch prune**

These categories must be presented distinctly in plans, output, and execution UX.

## Remote-Tracking Prune

Remote-tracking prune removes stale refs that no longer exist upstream.

This is comparatively mechanical and should be treated as a hygiene action with explainable detection.

Plan output should include:

- count of stale refs
- stale ref names where practical
- the repository they belong to

## Local Branch Prune

Local branch prune is not mechanical. It requires explicit classification before execution.

Local branches must be classified using a reusable model such as:

- `safe_to_prune`
- `probably_safe`
- `keep`
- `needs_review`

Each classification must include explicit reason codes.

## Safety Model

Prune follows:

- plan first
- explain proposed actions and blockers
- require explicit confirmation before execution

Additional defaults:

- local branch deletion defaults to safe delete semantics, not force delete
- protected branches are conservatively retained by default
- weak signals alone must not justify deletion
- prune execution must never be a hidden side effect of sync

## Interface Boundaries

### CLI and TUI

CLI and TUI own prune execution.

They may:

- preview prune plans
- distinguish remote-tracking prune from local branch deletion
- collect confirmation
- execute the chosen prune actions

### MCP

MCP may expose prune planning only.

It must not execute prune actions.

## Consequences

### Positive

- Remote hygiene and destructive local cleanup are no longer conflated.
- Users get explainable branch cleanup behavior.
- Branch hygiene views and prune plans can share the same classification engine.

### Negative

- RepoKeeper needs additional planning/output surface rather than hiding cleanup inside existing sync paths.

## Alternatives Considered

### 1. Keep prune as an implicit part of sync

**Rejected because:** remote-tracking cleanup and local branch deletion need different operator expectations and safety treatment.

### 2. Treat local branch prune as a simple merged/not-merged check

**Rejected because:** branch safety depends on multiple signals, not a single merge condition.
