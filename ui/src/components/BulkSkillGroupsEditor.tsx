import { useMemo, useState } from 'react';
import { Plus, Tags, X } from 'lucide-react';
import Button from './Button';
import DialogShell from './DialogShell';
import { Input } from './Input';
import { useT } from '../i18n';
import { normalizeGroups } from './SkillGroupsEditor';

export type BulkGroupOperation = 'add' | 'remove' | 'replace';

export interface BulkSkillGroupTarget {
  flatName: string;
  name: string;
  groups: string[];
}

export function buildBulkGroupUpdates(
  selectedSkills: BulkSkillGroupTarget[],
  operation: BulkGroupOperation,
  enteredGroups: string[],
): Array<{ name: string; groups: string[] }> {
  const normalizedInput = normalizeGroups(enteredGroups);

  return selectedSkills.map((skill) => {
    const current = normalizeGroups(skill.groups ?? []);

    if (operation === 'add') {
      return { name: skill.flatName, groups: normalizeGroups([...current, ...normalizedInput]) };
    }
    if (operation === 'remove') {
      return { name: skill.flatName, groups: current.filter((group) => !normalizedInput.includes(group)) };
    }
    return { name: skill.flatName, groups: normalizedInput };
  });
}

interface BulkSkillGroupsEditorProps {
  open: boolean;
  selectedSkills: BulkSkillGroupTarget[];
  knownGroups: string[];
  saving: boolean;
  error: string | null;
  onSave: (operation: BulkGroupOperation, groups: string[]) => void;
  onClose: () => void;
}

interface DraftState {
  key: string;
  operation: BulkGroupOperation;
  groups: string[];
  input: string;
}

export default function BulkSkillGroupsEditor({
  open,
  selectedSkills,
  knownGroups,
  saving,
  error,
  onSave,
  onClose,
}: BulkSkillGroupsEditorProps) {
  const t = useT();
  const selectedCount = selectedSkills.length;
  const draftKey = JSON.stringify(
    selectedSkills.map((skill) => skill.flatName),
  );
  const [draftState, setDraftState] = useState<DraftState>(() => ({
    key: draftKey,
    operation: 'add',
    groups: [],
    input: '',
  }));

  const operation = draftState.key === draftKey ? draftState.operation : 'add';
  const draftGroups = draftState.key === draftKey ? draftState.groups : [];
  const input = draftState.key === draftKey ? draftState.input : '';

  const suggestions = useMemo(
    () => normalizeGroups(knownGroups).filter((group) => !draftGroups.includes(group)),
    [knownGroups, draftGroups],
  );

  const canSubmit = !saving && (operation === 'replace' || draftGroups.length > 0);

  const setOperation = (nextOperation: BulkGroupOperation) => {
    setDraftState({
      key: draftKey,
      operation: nextOperation,
      groups: draftGroups,
      input,
    });
  };

  const addGroup = (value: string) => {
    const name = value.trim();
    if (!name) return;
    setDraftState({
      key: draftKey,
      operation,
      groups: normalizeGroups([...draftGroups, name]),
      input: '',
    });
  };

  const removeGroup = (name: string) => {
    setDraftState({
      key: draftKey,
      operation,
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
          <h3 className="text-lg font-bold text-pencil">
            {t('bulkSkillGroupsEditor.title', undefined, 'Bulk edit groups')}
          </h3>
          <p className="text-sm text-pencil-light truncate">
            {t(
              'bulkSkillGroupsEditor.selectedCount',
              { count: selectedCount },
              `${selectedCount} skills selected`,
            )}
          </p>
        </div>
      </div>

      <fieldset className="space-y-2 mb-4">
        <legend className="text-sm font-medium text-pencil-light mb-2">
          {t('bulkSkillGroupsEditor.mode.label', undefined, 'Operation')}
        </legend>
        <label className="flex items-center gap-2 text-sm text-pencil cursor-pointer">
          <input
            type="radio"
            name="bulk-group-operation"
            checked={operation === 'add'}
            onChange={() => setOperation('add')}
            disabled={saving}
          />
          {t('bulkSkillGroupsEditor.mode.add', undefined, 'Add groups')}
        </label>
        <label className="flex items-center gap-2 text-sm text-pencil cursor-pointer">
          <input
            type="radio"
            name="bulk-group-operation"
            checked={operation === 'remove'}
            onChange={() => setOperation('remove')}
            disabled={saving}
          />
          {t('bulkSkillGroupsEditor.mode.remove', undefined, 'Remove groups')}
        </label>
        <label className="flex items-center gap-2 text-sm text-pencil cursor-pointer">
          <input
            type="radio"
            name="bulk-group-operation"
            checked={operation === 'replace'}
            onChange={() => setOperation('replace')}
            disabled={saving}
          />
          {t('bulkSkillGroupsEditor.mode.replace', undefined, 'Replace all groups')}
        </label>
      </fieldset>

      <div className="space-y-4">
        <div className="flex gap-2">
          <Input
            label={t('skillGroupsEditor.inputLabel')}
            value={input}
            onChange={(e) => {
              setDraftState({
                key: draftKey,
                operation,
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

        {error && (
          <p className="text-sm text-danger">
            {t('bulkSkillGroupsEditor.errorSummary', undefined, 'Some skills could not be updated')}: {error}
          </p>
        )}
      </div>

      <div className="flex justify-end gap-3 mt-6">
        <Button variant="secondary" size="md" onClick={onClose} disabled={saving}>
          {t('common.cancel')}
        </Button>
        <Button
          variant="primary"
          size="md"
          onClick={() => onSave(operation, draftGroups)}
          loading={saving}
          disabled={!canSubmit}
        >
          {t('bulkSkillGroupsEditor.save', undefined, 'Apply changes')}
        </Button>
      </div>
    </DialogShell>
  );
}
