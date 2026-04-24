import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ComponentProps } from 'react';
import SkillGroupsEditor from './SkillGroupsEditor';
import { I18nProvider, LOCALE_STORAGE_KEY } from '../i18n';

function renderEditor(props?: Partial<ComponentProps<typeof SkillGroupsEditor>>) {
  const onSave = vi.fn();
  const onClose = vi.fn();
  render(
    <I18nProvider>
      <SkillGroupsEditor
        open
        skillName="alpha"
        groups={['unused']}
        knownGroups={['reference', 'unused']}
        saving={false}
        onSave={onSave}
        onClose={onClose}
        {...props}
      />
    </I18nProvider>,
  );
  return { onSave, onClose };
}

describe('SkillGroupsEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.setItem(LOCALE_STORAGE_KEY, 'en');
  });

  it('adds a typed group and saves sorted values', async () => {
    const user = userEvent.setup();
    const { onSave } = renderEditor();

    await user.type(screen.getByLabelText('Group name'), 'reference{Enter}');
    await user.click(screen.getByRole('button', { name: 'Save groups' }));

    expect(onSave).toHaveBeenCalledWith(['reference', 'unused']);
  });

  it('removes an existing group', async () => {
    const user = userEvent.setup();
    const { onSave } = renderEditor();

    await user.click(screen.getByRole('button', { name: 'Remove unused' }));
    await user.click(screen.getByRole('button', { name: 'Save groups' }));

    expect(onSave).toHaveBeenCalledWith([]);
  });

  it('adds a known group from suggestions', async () => {
    const user = userEvent.setup();
    const { onSave } = renderEditor();

    await user.click(screen.getByRole('button', { name: 'reference' }));
    await user.click(screen.getByRole('button', { name: 'Save groups' }));

    expect(onSave).toHaveBeenCalledWith(['reference', 'unused']);
  });
});
