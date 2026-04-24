import { describe, expect, it } from 'vitest';
import type { Skill } from '../api/client';
import { buildSkillGroupOptions, formatGroupBadgeText, matchesSelectedGroup } from './resourceGroups';

function skill(name: string, groups?: string[], kind: Skill['kind'] = 'skill'): Skill {
  return {
    name,
    kind,
    flatName: name,
    relPath: name,
    sourcePath: `/tmp/${name}`,
    isInRepo: false,
    groups,
  };
}

describe('resourceGroups', () => {
  it('builds sorted skill-only group options with counts', () => {
    expect(buildSkillGroupOptions([
      skill('a', ['unused', 'reference']),
      skill('b', ['unused']),
      skill('agent', ['unused'], 'agent'),
      skill('c'),
    ])).toEqual([
      { value: 'reference', label: 'reference', count: 1 },
      { value: 'unused', label: 'unused', count: 2 },
    ]);
  });

  it('matches all skills when no group is selected', () => {
    expect(matchesSelectedGroup(skill('a'), '')).toBe(true);
    expect(matchesSelectedGroup(skill('a', ['unused']), '')).toBe(true);
  });

  it('matches only skills in the selected group', () => {
    expect(matchesSelectedGroup(skill('a', ['unused']), 'unused')).toBe(true);
    expect(matchesSelectedGroup(skill('b', ['reference']), 'unused')).toBe(false);
    expect(matchesSelectedGroup(skill('c'), 'unused')).toBe(false);
  });

  it('formats compact badge text', () => {
    expect(formatGroupBadgeText(['a'])).toEqual({ visible: ['a'], overflow: 0 });
    expect(formatGroupBadgeText(['a', 'b', 'c'])).toEqual({ visible: ['a', 'b'], overflow: 1 });
    expect(formatGroupBadgeText(undefined)).toEqual({ visible: [], overflow: 0 });
  });
});
