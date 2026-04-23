package sync

import "strings"

// ClassifySkillForTarget determines whether a skill should sync to a target,
// returning a status string and an optional reason.
//
// Statuses:
//   - "synced": skill will be synced to this target
//   - "excluded": skill matched an exclude pattern (reason = the pattern)
//   - "not_included": include patterns exist but skill matched none
func ClassifySkillForTarget(flatName string, include, exclude []string) (status, reason string) {
	// Normalize patterns (trim whitespace, validate syntax) — same path as shouldSyncFlatName
	incNorm, excNorm, _ := normalizedFilterPatterns(include, exclude)

	// Check include first (matches shouldSyncFlatName precedence)
	if len(incNorm) > 0 && !matchesAnyPattern(flatName, incNorm) {
		return "not_included", ""
	}

	// Check exclude
	if pattern, matched := firstMatchingPattern(flatName, excNorm); matched {
		return "excluded", pattern
	}

	return "synced", ""
}

// ClassifyAgentForTarget determines whether an agent should sync to a target.
// Agent filter patterns are matched against the flat name without the .md
// extension so server previews match FilterAgents and real sync behavior.
func ClassifyAgentForTarget(flatName string, include, exclude []string) (status, reason string) {
	return ClassifySkillForTarget(strings.TrimSuffix(flatName, ".md"), include, exclude)
}
