package install

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxCustomGroupsPerSkill = 20
	MaxCustomGroupNameRunes = 64
)

// NormalizeCustomGroups trims, deduplicates, validates, and sorts user-defined
// skill group names before they are stored in .metadata.json.
func NormalizeCustomGroups(groups []string) ([]string, error) {
	seen := make(map[string]struct{}, len(groups))
	normalized := make([]string, 0, len(groups))
	for _, raw := range groups {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > MaxCustomGroupNameRunes {
			return nil, fmt.Errorf("custom group %q is longer than %d characters", name, MaxCustomGroupNameRunes)
		}
		if !isValidCustomGroupName(name) {
			return nil, fmt.Errorf("custom group %q may contain only letters, numbers, spaces, dash, underscore, or dot", name)
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	if len(normalized) > MaxCustomGroupsPerSkill {
		return nil, fmt.Errorf("a skill can belong to at most %d custom groups", MaxCustomGroupsPerSkill)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func isValidCustomGroupName(name string) bool {
	for _, r := range name {
		switch {
		case r == ' ' || r == '-' || r == '_' || r == '.':
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			continue
		default:
			return false
		}
	}
	return true
}
