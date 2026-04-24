import { useMemo, useState } from 'react';
import { Plus, Tags, X } from 'lucide-react';
import Button from './Button';
import DialogShell from './DialogShell';
import { Input } from './Input';
import { useT } from '../i18n';

function normalizeGroups(groups: string[]): string[] {
  return Array.from(new Set(groups.map((group) => group.trim()).filter(Boolean)))
    .sort((a, b) => a.localeCompare(b));
}

interface SkillGroupsEditorProps {
  open: boolean;
  skillName: string;
  groups: string[];
  knownGroups: string[];
  saving: boolean;
  onSave: (groups: string[]) => void;
  onClose: () => void;
}

interface DraftState {
  key: string;
  groups: string[];
  input: string;
}

export default function SkillGroupsEditor({
  open,
  skillName,
  groups,
  knownGroups,
  saving,
  onSave,
  onClose,
}: SkillGroupsEditorProps) {
  const t = useT();
  const normalizedGroups = useMemo(() => normalizeGroups(groups), [groups]);
  const draftKey = JSON.stringify([skillName, normalizedGroups]);
  const [draftState, setDraftState] = useState<DraftState>(() => ({
    key: draftKey,
    groups: normalizedGroups,
    input: '',
  }));
  const draftGroups = draftState.key === draftKey ? draftState.groups : normalizedGroups;
  const input = draftState.key === draftKey ? draftState.input : '';

  const suggestions = useMemo(
    () => normalizeGroups(knownGroups).filter((group) => !draftGroups.includes(group)),
    [knownGroups, draftGroups],
  );

  const addGroup = (value: string) => {
    const name = value.trim();
    if (!name) return;
    setDraftState({
      key: draftKey,
      groups: normalizeGroups([...draftGroups, name]),
      input: '',
    });
  };

  const removeGroup = (name: string) => {
    setDraftState({
      key: draftKey,
      groups: draftGroups.filter((group) => group !== name),
      input,
    });
  };

  return (
    <DialogShell open={open} onClose={onClose} maxWidth="lg" preventClose={saving}>
      <div className="flex items-start gap-3 mb-4">
        <div className="p-2 bg-muted text-pencil rounded-[var(--radius-sm)]">
          <Tags size={18} strokeWidth={2.5} />
        </div>
        <div className="min-w-0">
          <h3 className="text-lg font-bold text-pencil">{t('skillGroupsEditor.title')}</h3>
          <p className="text-sm text-pencil-light truncate">{skillName}</p>
        </div>
      </div>

      <div className="space-y-4">
        <div className="flex gap-2">
          <Input
            label={t('skillGroupsEditor.inputLabel')}
            value={input}
            onChange={(e) => {
              setDraftState({
                key: draftKey,
                groups: draftGroups,
                input: e.target.value,
              });
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                addGroup(input);
              }
            }}
            placeholder={t('skillGroupsEditor.inputPlaceholder')}
            disabled={saving}
          />
          <Button
            variant="secondary"
            size="md"
            onClick={() => addGroup(input)}
            disabled={saving || input.trim() === ''}
            className="self-end"
          >
            <Plus size={16} strokeWidth={2.5} />
            {t('skillGroupsEditor.add')}
          </Button>
        </div>

        <div>
          <p className="text-sm font-medium text-pencil-light mb-2">{t('skillGroupsEditor.current')}</p>
          {draftGroups.length > 0 ? (
            <div className="flex flex-wrap gap-2">
              {draftGroups.map((group) => (
                <span
                  key={group}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-muted text-pencil text-sm rounded-[var(--radius-sm)]"
                >
                  {group}
                  <button
                    type="button"
                    className="text-pencil-light hover:text-pencil cursor-pointer disabled:cursor-not-allowed"
                    onClick={() => removeGroup(group)}
                    disabled={saving}
                    aria-label={t('skillGroupsEditor.removeGroup', { group })}
                  >
                    <X size={13} strokeWidth={2.5} />
                  </button>
                </span>
              ))}
            </div>
          ) : (
            <p className="text-sm text-pencil-light">{t('skillGroupsEditor.empty')}</p>
          )}
        </div>

        {suggestions.length > 0 && (
          <div>
            <p className="text-sm font-medium text-pencil-light mb-2">{t('skillGroupsEditor.suggestions')}</p>
            <div className="flex flex-wrap gap-2">
              {suggestions.map((group) => (
                <Button
                  key={group}
                  variant="ghost"
                  size="sm"
                  onClick={() => addGroup(group)}
                  disabled={saving}
                >
                  {group}
                </Button>
              ))}
            </div>
          </div>
        )}
      </div>

      <div className="flex justify-end gap-3 mt-6">
        <Button variant="secondary" size="md" onClick={onClose} disabled={saving}>
          {t('common.cancel')}
        </Button>
        <Button variant="primary" size="md" onClick={() => onSave(draftGroups)} loading={saving}>
          {t('skillGroupsEditor.save')}
        </Button>
      </div>
    </DialogShell>
  );
}
