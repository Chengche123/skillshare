import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ComponentProps } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import BulkSkillGroupsEditor, { buildBulkGroupUpdates } from './BulkSkillGroupsEditor';
import { I18nProvider, LOCALE_STORAGE_KEY } from '../i18n';

describe('buildBulkGroupUpdates', () => {
  it('computes add, remove, and replace results from selected skills', () => {
    const selected = [
      { flatName: 'Alpha', name: 'Alpha', groups: ['Archive'] },
      { flatName: 'Beta', name: 'Beta', groups: ['Cold', 'Reference'] },
    ];

    expect(buildBulkGroupUpdates(selected, 'add', ['Cold'])).toEqual([
      { name: 'Alpha', groups: ['Archive', 'Cold'] },
      { name: 'Beta', groups: ['Cold', 'Reference'] },
    ]);
    expect(buildBulkGroupUpdates(selected, 'remove', ['Cold'])).toEqual([
      { name: 'Alpha', groups: ['Archive'] },
      { name: 'Beta', groups: ['Reference'] },
    ]);
    expect(buildBulkGroupUpdates(selected, 'replace', ['Shared'])).toEqual([
      { name: 'Alpha', groups: ['Shared'] },
      { name: 'Beta', groups: ['Shared'] },
    ]);
  });
});

describe('BulkSkillGroupsEditor', () => {
  beforeEach(() => {
    localStorage.setItem(LOCALE_STORAGE_KEY, 'en');
  });

  function renderEditor(props?: Partial<ComponentProps<typeof BulkSkillGroupsEditor>>) {
    const onSave = vi.fn();
    const onClose = vi.fn();
    render(
      <I18nProvider>
        <BulkSkillGroupsEditor
          open
          selectedSkills={[
            { flatName: 'Alpha', name: 'Alpha', groups: ['Archive'] },
          ]}
          knownGroups={['Archive', 'Reference']}
          saving={false}
          error={null}
          onSave={onSave}
          onClose={onClose}
          {...props}
        />
      </I18nProvider>,
    );
    return { onSave, onClose };
  }

  it('disables add and remove submit when no groups are entered', async () => {
    const user = userEvent.setup();
    renderEditor();

    const submit = screen.getByRole('button', { name: 'Apply changes' });
    expect(submit).toBeDisabled();

    await user.click(screen.getByRole('radio', { name: 'Remove groups' }));
    expect(submit).toBeDisabled();

    await user.click(screen.getByRole('radio', { name: 'Replace all groups' }));
    expect(submit).toBeEnabled();
  });

  it('submits replace mode with an empty group set', async () => {
    const user = userEvent.setup();
    const { onSave } = renderEditor();

    await user.click(screen.getByRole('radio', { name: 'Replace all groups' }));
    await user.click(screen.getByRole('button', { name: 'Apply changes' }));

    expect(onSave).toHaveBeenCalledWith('replace', []);
  });

  it('adds a group and submits add mode', async () => {
    const user = userEvent.setup();
    const { onSave } = renderEditor();

    await user.type(screen.getByLabelText('Group name'), 'Cold{Enter}');
    await user.click(screen.getByRole('button', { name: 'Apply changes' }));

    expect(onSave).toHaveBeenCalledWith('add', ['Cold']);
  });
});
