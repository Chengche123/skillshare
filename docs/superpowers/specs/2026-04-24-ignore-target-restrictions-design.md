# Ignore Skill And Agent Target Restrictions

Date: 2026-04-24

## Summary

Change target-scoped result computation so skill and agent frontmatter `targets` are treated as metadata only, not as sync eligibility constraints.

After this change:

- Skills continue to parse `targets` from `SKILL.md`.
- Agents continue to parse `targets` from their frontmatter.
- Sync, diff, status, analyze, and sync-matrix style target views no longer exclude resources when the declared `targets` do not match the current target.
- Validation and visibility of the declared `targets` metadata remain intact.

## Problem

Today the repo has two different behaviors around resource `targets` metadata:

- Skills: real target computation filters by `DiscoveredSkill.Targets`, so a skill can be excluded from sync when its declared `targets` do not match the target being processed.
- Agents: real sync does not enforce frontmatter `targets`, but sync-matrix style preview still reports `skill_target_mismatch` for agents.

This produces two problems:

1. Users cannot sync a skill to an arbitrary target if its frontmatter declares a different target.
2. Different product surfaces disagree about effective behavior, especially for agents.

The intended behavior is simpler: ignore resource-level `targets` during target result computation while keeping the field parsed and visible.

## Goals

- Make skill frontmatter `targets` non-blocking for sync.
- Make agent frontmatter `targets` non-blocking as well.
- Keep parsing and storing declared `targets` metadata.
- Keep doctor-style metadata validation available.
- Minimize code churn by changing shared filtering/classification points instead of many callers.

## Non-Goals

- Removing the `targets` field from frontmatter.
- Changing config schema or migration behavior.
- Removing doctor warnings for unknown declared target names.
- Refactoring target naming, include/exclude filters, or sync modes.

## Confirmed Behavior

### Skills

- `targets` remains parsed during discovery.
- Any target-level result computation must ignore `DiscoveredSkill.Targets`.
- Include/exclude filters still apply.
- Target naming validation and collision handling still apply.

### Agents

- `targets` remains parsed during discovery.
- Any target-level result computation and preview must ignore `DiscoveredResource.Targets`.
- Include/exclude filters still apply.
- Existing agent path and mode resolution stay unchanged.

### Metadata Visibility

- UIs, diagnostics, and internal models may continue to display declared `targets`.
- Doctor may continue validating whether declared target names are known.

## Recommended Design

Use the existing shared helper boundaries and change behavior there.

### 1. Make skill target filtering a no-op

Current shared target filtering for skills happens through:

- `internal/sync/filter.go` → `FilterSkillsByTarget()`

This function is already reused by:

- sync result computation
- prune result computation
- diff
- status
- analyze
- doctor
- server handlers that derive per-target skill views

Change `FilterSkillsByTarget()` so it returns the input slice unchanged. This preserves all existing call sites while removing target-based exclusion for skills.

### 2. Remove target mismatch classification from preview/matrix flows

Current sync-matrix and preview classification goes through:

- `internal/sync/classify.go` → `ClassifySkillForTarget()`

This function is used for both skills and agents in sync-matrix handlers.

Change `ClassifySkillForTarget()` so it no longer emits `skill_target_mismatch`. Classification should only reflect:

- `synced`
- `excluded`
- `not_included`

This keeps sync-matrix and preview behavior aligned with actual sync behavior.

### 3. Leave discovery unchanged

Do not change:

- `internal/sync/discover_walk.go`
- `internal/resource/agent.go`

Those discovery paths should continue parsing frontmatter `targets` so metadata remains available to validation and presentation layers.

## Why This Is The Smallest Safe Change

Two tempting alternatives were rejected:

### Parse-time removal

Stopping parsing in discovery would also remove metadata visibility and doctor validation. That does not match the requirement.

### Call-site-by-call-site removal

Individually deleting target filtering from sync, diff, status, analyze, and server handlers would create larger code churn and a higher chance of inconsistent behavior.

Changing the shared helper points produces the desired behavior with the smallest coherent edit set.

## Expected Code Touch Points

### Primary behavior changes

- `internal/sync/filter.go`
- `internal/sync/classify.go`

### Tests to update

- `internal/sync/filter_target_test.go`
- `internal/sync/classify_test.go`
- `internal/server/handler_sync_matrix_test.go`
- one skill sync integration test
- one agent sync integration test

## Testing Strategy

### Unit tests

Update `FilterSkillsByTarget()` tests so all inputs pass through regardless of declared `Targets`.

Update `ClassifySkillForTarget()` tests so declared `targets` no longer affect status. Add cases proving that mismatched declared targets still classify as:

- `synced` when no include/exclude rule rejects them
- `excluded` when exclude matches
- `not_included` when include does not match

### Handler tests

Add or update sync-matrix tests so:

- a skill with declared `targets: [cursor]` is still `synced` for `claude`
- an agent with declared `targets: [cursor]` is still `synced` for `claude`
- no `skill_target_mismatch` status is returned

### Integration tests

Add one skill sync integration test:

- configure only a `claude` target
- create a skill declaring `targets: [cursor]`
- run sync
- verify it still appears in the `claude` target

Add one agent sync integration test with the same shape for agent sync.

## Compatibility And Risk

This is an intentional behavior change.

### Main behavior change

Resources that previously relied on frontmatter `targets` for isolation will now sync to any eligible target unless blocked by include/exclude filters or other normal validation.

### Low-risk areas

- No config migration is needed.
- No stored data shape changes.
- No discovery format changes.

### Residual compatibility note

`sync_matrix.skill_target_mismatch` may remain in reason-code mapping code for now, but it should become unreachable after the behavior change. Keeping the mapping temporarily is acceptable for a minimal patch.

## Acceptance Criteria

1. A skill declaring any `targets` value can still sync to any configured target.
2. An agent declaring any `targets` value can still sync to any configured target.
3. Sync, diff, status, analyze, and sync-matrix style views no longer disagree due to resource `targets`.
4. Declared `targets` remain parsed and visible in discovery-derived data.
5. Doctor-style validation of declared target names still works.

## Implementation Notes For Planning

The implementation plan should preserve the narrow change set:

1. Update shared filtering/classification helpers.
2. Adjust unit tests first.
3. Update sync-matrix handler tests.
4. Add one skill integration test and one agent integration test.
5. Run targeted Go tests for touched packages, then broader verification if needed.
