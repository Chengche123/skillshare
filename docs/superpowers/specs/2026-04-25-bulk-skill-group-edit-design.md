# Bulk Skill Group Edit Design

## Summary

Add bulk group editing to the `/resources?tab=skills` UI so a user can select multiple skills and apply one of three group operations in one flow:

- add groups
- remove groups
- replace all groups

The feature should stay within the existing Resources page patterns, reuse the current single-skill group editor concepts where possible, and avoid backend API expansion for this first version.

## Problem

The current UI supports editing groups for one skill at a time from the context menu. That works for occasional cleanup but becomes slow when reorganizing many skills at once. Users need a batch workflow directly on the skills resources page.

## Goals

- Let users select multiple visible skills on `/resources?tab=skills`
- Let users open one bulk editor from the current page
- Support these operations:
  - add one or more groups to all selected skills
  - remove one or more groups from all selected skills
  - replace each selected skill's groups with one normalized group set
- Reuse the existing `PATCH /api/resources/{name}/groups` API for persistence
- Keep the UI resilient when one or more saves fail

## Non-Goals

- No batch group editing for agents
- No new backend batch groups endpoint in this change
- No change to the existing folder-based batch target editing flow
- No persistent cross-session selection state
- No GitHub Actions workflow changes

## User Experience

### Selection model

Selection is available only in the skills tab.

- Each visible skill row or card gets a checkbox
- The page gets a bulk-selection action area when at least one skill is selected
- Users can select:
  - a single skill
  - all currently visible filtered skills
  - none

Selection applies only to currently visible skills in the active tab and current filters. The UI should not silently operate on hidden or filtered-out skills.

### Bulk action entry point

When one or more skills are selected, the page shows a compact bulk action bar near the existing filters and controls. It includes:

- selected count
- select all visible action
- clear selection action
- bulk edit groups action

This keeps the workflow obvious without hiding it behind a context menu.

### Bulk groups editor

Opening bulk edit shows a dedicated dialog for multi-skill editing. The dialog should show:

- selected skill count
- operation mode selector
  - add groups
  - remove groups
  - replace all groups
- group input area
- known-group suggestions derived from all skills

The input behavior should match the current group editor as closely as possible:

- trim whitespace
- deduplicate values
- store normalized sorted groups

### Save behavior

On submit, the client computes the target group list for each selected skill based on the chosen mode and that skill's current groups.

Rules:

- add: union of current groups and entered groups
- remove: current groups minus entered groups
- replace: entered groups only

The UI then calls the existing single-resource group update API once per selected skill.

### Result handling

If all requests succeed:

- close the dialog
- keep or clear selection based on the final selected list choice; for v1, clear selection after success
- show one success toast with updated count

If any request fails:

- keep the dialog open
- keep selection intact
- show a summary error state in the dialog
- allow retry without losing the drafted groups and chosen mode

## Architecture

## Frontend

Primary work stays in the UI layer:

- `ui/src/pages/ResourcesPage.tsx`
  - add selection state for visible skills
  - render bulk action controls
  - open the new bulk editor dialog
  - perform optimistic or near-optimistic cache updates after successful writes
- `ui/src/components/SkillGroupsEditor.tsx`
  - keep single-skill editor behavior unchanged
- new bulk editor component
  - dedicated dialog component for batch semantics rather than overloading the single-skill dialog with many conditional branches

### Why a new bulk component

The single-skill editor models one final group set. Bulk editing adds an operation mode and result summary, so folding both concerns into one component would make the existing single-item flow harder to reason about. A separate component keeps responsibilities clear while still reusing normalization logic and suggestion patterns.

## Data flow

1. Resources page loads skills as it does today.
2. User selects visible skills.
3. User opens bulk groups editor.
4. User chooses operation mode and group inputs.
5. Client derives per-skill target groups.
6. Client sends one `setSkillGroups(name, groups)` request per selected skill.
7. Page refreshes resource queries and updates toast/error state.

## Error handling

- If no skills are selected, the bulk editor cannot open.
- If the dialog has no entered groups:
  - add/remove should stay disabled
  - replace is allowed and means clear all groups only if the user explicitly submits an empty set via the replace mode
- Any failed API call should be surfaced in a compact summary
- Partial success should not be reported as full success

## Testing

### Frontend tests

Add or update React tests to cover:

- selecting multiple visible skills
- opening the bulk groups editor from the page
- add mode computes the merged groups per skill
- remove mode computes the reduced groups per skill
- replace mode sends the exact final groups per skill
- failed save keeps the dialog open with draft values preserved
- selection only applies to visible skills in the current filter state

### Backend tests

No new backend behavior is introduced. Existing handler tests for single-skill group updates remain the verification baseline for the API contract.

## Files likely involved

- `ui/src/pages/ResourcesPage.tsx`
- `ui/src/pages/ResourcesPage.test.tsx`
- `ui/src/components/SkillGroupsEditor.tsx` if shared helpers are extracted
- `ui/src/components/SkillGroupsEditor.test.tsx` if helper behavior moves there
- `ui/src/components/BulkSkillGroupsEditor.tsx` new
- related i18n locale files for new text

## Risks

- Large selections mean many sequential or parallel requests because v1 reuses the single-item endpoint
- Selection state could become confusing if filters change mid-edit
- Overloading existing page controls could add visual clutter if the bulk bar is not scoped carefully

## Mitigations

- Limit scope to visible skills only
- Clear selection after full success
- Keep the bulk bar compact and conditional
- Use a dedicated bulk editor instead of complicating the single-item dialog

## Acceptance Criteria

- On `/resources?tab=skills`, a user can select multiple visible skills
- The page exposes a bulk groups edit action when selection is non-empty
- The bulk editor supports add, remove, and replace modes
- Saving updates each selected skill through the existing single-skill groups API
- Partial or full failures keep the dialog open and preserve user input
- Existing single-skill group editing still works
