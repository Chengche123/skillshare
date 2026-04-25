# Filter Studio Group Preview Design

## Summary

Add a skill-only custom group filter to the right-side Live Preview on `/targets/:name/filters?kind=skill`.

This filter is preview-only:

- It does not change target config.
- It does not affect saved `include` / `exclude` rules.
- It does not appear on `kind=agent`.

## Problem

Filter Studio already lets users edit target include/exclude patterns, but when the source contains many skills, the right-side preview is hard to scan. Skills already support custom groups in metadata, and the Resources UI can read those groups, but Filter Studio does not expose them.

The user wants to preview skills by custom group without changing persisted target filters.

## Goals

- Let users narrow the Live Preview to one custom skill group.
- Keep current save behavior unchanged.
- Reuse existing skill group metadata and UI helpers where possible.
- Avoid expanding server-side target filter semantics.

## Non-Goals

- Persisting selected preview groups to target config.
- Changing sync behavior outside the page preview.
- Adding custom-group filtering for agents.
- Replacing or generating `include` / `exclude` patterns from groups.

## User Experience

On `/targets/:name/filters?kind=skill`:

- The right-side Live Preview gains a group filter control.
- The control appears only when at least one custom skill group exists.
- Default selection is `All groups`.
- Choosing a group filters the preview list to skills that belong to that group.
- The preview search box still works on top of the selected group.
- The synced/total counts reflect the filtered preview currently shown for the selected group, before the text search is applied.

On `/targets/:name/filters?kind=agent`:

- No group filter control is shown.

When saving:

- Only the existing `include` / `exclude` payload is sent.
- The selected preview group is ignored by save and navigation.

## Data Flow

### Existing data

- `api.previewSyncMatrix(...)` returns preview entries for the target.
- `api.listSkills('skill')` returns all skill resources, including `groups`.

### New page-level flow

1. Filter Studio loads the target config as it does today.
2. When `kind === 'skill'`, the page also loads skill resources.
3. The page builds group options from resource metadata using the existing `resourceGroups` helper.
4. The preview entries are joined with skill metadata by `flatName`.
5. A selected group filters only the right-side preview data.
6. Save still calls `api.updateTarget(name, { include, exclude })`.

## Implementation Approach

### Frontend

Update `ui/src/pages/FilterStudioPage.tsx`:

- Add a `listSkills('skill')` query, enabled only for skill mode.
- Build preview group options from returned skills.
- Track local `selectedGroup` state, defaulting to empty string for `All groups`.
- Filter `kindPreview` by selected group before applying the search box.
- Hide the control when there are no available custom groups.
- Leave `handleSave` and preview request payload unchanged.

Reuse existing helper logic from `ui/src/lib/resourceGroups.ts` rather than duplicating group counting or matching rules.

### API / Server

No API or server contract changes.

This keeps the feature strictly presentational and avoids modifying target config, sync matrix handlers, or persisted schema.

## Testing Strategy

Add or update frontend tests to cover:

- The group filter appears in skill mode when groups exist.
- Selecting a group narrows preview rows.
- Save still calls `updateTarget` with only the current include/exclude fields.
- Agent mode does not show the group filter.

No backend test changes should be needed because behavior stays client-side.

## Risks

- Preview entries and skill metadata must match by the same identifier. The implementation should use skill `flatName` consistently.
- If a preview entry has no matching metadata, it should remain visible under `All groups` and be excluded from named-group views.
- Query loading states should not block the existing preview flow; missing group data should degrade to no extra control.

## Minimal File Scope

- Modify: `ui/src/pages/FilterStudioPage.tsx`
- Reuse: `ui/src/lib/resourceGroups.ts`
- Add or modify: related UI test file under `ui/src/pages/`

## Acceptance Criteria

- Visiting `/targets/<name>/filters?kind=skill` shows a preview-only custom group filter when grouped skills exist.
- Selecting a custom group changes only the right-side preview and its counts.
- Saving does not persist the selected group and does not alter the request shape.
- `kind=agent` behavior is unchanged.
