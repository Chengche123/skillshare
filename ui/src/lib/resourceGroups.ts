import type { Skill } from '../api/client';

export interface SkillGroupOption {
  value: string;
  label: string;
  count: number;
}

export function skillGroups(skill: Pick<Skill, 'groups'>): string[] {
  return skill.groups ?? [];
}

export function buildSkillGroupOptions(skills: Pick<Skill, 'kind' | 'groups'>[]): SkillGroupOption[] {
  const counts = new Map<string, number>();
  for (const skill of skills) {
    if (skill.kind === 'agent') continue;
    for (const group of skill.groups ?? []) {
      counts.set(group, (counts.get(group) ?? 0) + 1);
    }
  }
  return Array.from(counts.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([group, count]) => ({ value: group, label: group, count }));
}

export function matchesSelectedGroup(skill: Pick<Skill, 'groups'>, selectedGroup: string): boolean {
  return selectedGroup === '' || (skill.groups ?? []).includes(selectedGroup);
}

export function formatGroupBadgeText(groups: string[] | undefined, limit = 2): { visible: string[]; overflow: number } {
  const safeGroups = groups ?? [];
  return {
    visible: safeGroups.slice(0, limit),
    overflow: Math.max(0, safeGroups.length - limit),
  };
}
