import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ResourcesPage from './ResourcesPage';
import { api, type Skill } from '../api/client';
import { ToastProvider } from '../components/Toast';
import { I18nProvider, LOCALE_STORAGE_KEY } from '../i18n';

vi.mock('react-virtuoso', () => ({
  VirtuosoGrid: ({ totalCount, itemContent }: { totalCount: number; itemContent: (index: number) => React.ReactNode }) => (
    <div>{Array.from({ length: totalCount }, (_, index) => <div key={index}>{itemContent(index)}</div>)}</div>
  ),
  Virtuoso: ({ totalCount, itemContent }: { totalCount: number; itemContent: (index: number) => React.ReactNode }) => (
    <div>{Array.from({ length: totalCount }, (_, index) => <div key={index}>{itemContent(index)}</div>)}</div>
  ),
}));

class ResizeObserverMock {
  observe() {}
  disconnect() {}
}

function makeSkill(overrides: Partial<Skill>): Skill {
  const name = overrides.name ?? overrides.flatName ?? 'skill';
  return {
    name,
    flatName: name,
    kind: 'skill',
    relPath: name,
    sourcePath: `/skills/${name}`,
    isInRepo: false,
    ...overrides,
  };
}

function renderResources(resources: Skill[]) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  vi.spyOn(api, 'listSkills').mockResolvedValue({ resources });
  vi.spyOn(api, 'availableTargets').mockResolvedValue({ targets: [] });

  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <ToastProvider>
          <MemoryRouter initialEntries={['/resources?tab=skills']}>
            <ResourcesPage />
          </MemoryRouter>
        </ToastProvider>
      </I18nProvider>
    </QueryClientProvider>,
  );
}

describe('ResourcesPage skill groups', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
    localStorage.setItem(LOCALE_STORAGE_KEY, 'en');
    Object.defineProperty(globalThis, 'ResizeObserver', {
      configurable: true,
      value: ResizeObserverMock,
    });
    Element.prototype.scrollIntoView = vi.fn();
  });

  it('filters skills by the selected custom group', async () => {
    const user = userEvent.setup();
    renderResources([
      makeSkill({ name: 'Alpha', groups: ['Archive', 'Unused'] }),
      makeSkill({ name: 'Beta', groups: ['Archive'] }),
      makeSkill({ name: 'Gamma', groups: ['Reference'] }),
      makeSkill({ name: 'Docs Agent', kind: 'agent', groups: ['Archive'] }),
    ]);

    expect((await screen.findAllByText('Alpha')).length).toBeGreaterThan(0);
    expect(screen.getAllByText('Beta').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Gamma').length).toBeGreaterThan(0);

    await user.click(screen.getByText('All groups'));
    await user.click(screen.getByRole('option', { name: /Archive \(2\)/ }));

    expect(screen.getAllByText('Alpha').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Beta').length).toBeGreaterThan(0);
    expect(screen.queryAllByText('Gamma')).toHaveLength(0);
  });

  it('saves groups from the skill context menu editor', async () => {
    const user = userEvent.setup();
    const setSkillGroups = vi.spyOn(api, 'setSkillGroups').mockResolvedValue({ success: true });
    renderResources([
      makeSkill({ name: 'Alpha', flatName: 'Alpha', groups: ['Archive'] }),
      makeSkill({ name: 'Beta', flatName: 'Beta', groups: ['Reference'] }),
    ]);

    const alpha = (await screen.findAllByText('Alpha'))[0];
    fireEvent.contextMenu(alpha);
    await user.click(await screen.findByRole('menuitem', { name: 'Edit groups' }));

    await user.type(screen.getByLabelText('Group name'), 'Cold{Enter}');
    await user.click(screen.getByRole('button', { name: 'Save groups' }));

    await waitFor(() => {
      expect(setSkillGroups).toHaveBeenCalledWith('Alpha', ['Archive', 'Cold']);
    });
  });

  it('keeps the groups editor open with draft values when saving fails', async () => {
    const user = userEvent.setup();
    const setSkillGroups = vi.spyOn(api, 'setSkillGroups').mockRejectedValue(new Error('save failed'));
    renderResources([
      makeSkill({ name: 'Alpha', flatName: 'Alpha', groups: ['Archive'] }),
    ]);

    const alpha = (await screen.findAllByText('Alpha'))[0];
    fireEvent.contextMenu(alpha);
    await user.click(await screen.findByRole('menuitem', { name: 'Edit groups' }));

    await user.type(screen.getByLabelText('Group name'), 'Cold{Enter}');
    await user.click(screen.getByRole('button', { name: 'Save groups' }));

    await waitFor(() => {
      expect(setSkillGroups).toHaveBeenCalledWith('Alpha', ['Archive', 'Cold']);
    });
    await screen.findByText('save failed');

    expect(screen.getByRole('button', { name: 'Save groups' })).toBeInTheDocument();
    expect(screen.getByText('Cold')).toBeInTheDocument();
  });

  it('selects visible skills and opens the bulk groups editor', async () => {
    const user = userEvent.setup();
    renderResources([
      makeSkill({ name: 'Alpha', flatName: 'Alpha', groups: ['Archive'] }),
      makeSkill({ name: 'Beta', flatName: 'Beta', groups: ['Reference'] }),
      makeSkill({ name: 'Gamma', flatName: 'Gamma', groups: ['Unused'] }),
    ]);

    await screen.findByText('Alpha');

    await user.click(screen.getByRole('checkbox', { name: 'Select Alpha' }));
    await user.click(screen.getByRole('checkbox', { name: 'Select Beta' }));

    expect(screen.getByText('2 selected')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Edit groups' }));

    expect(screen.getByText('Bulk edit groups')).toBeInTheDocument();
    expect(screen.getByText('2 skills selected')).toBeInTheDocument();
  });

  it('applies add mode to each selected skill', async () => {
    const user = userEvent.setup();
    const setSkillGroups = vi.spyOn(api, 'setSkillGroups').mockResolvedValue({ success: true });
    renderResources([
      makeSkill({ name: 'Alpha', flatName: 'Alpha', groups: ['Archive'] }),
      makeSkill({ name: 'Beta', flatName: 'Beta', groups: ['Reference'] }),
    ]);

    await screen.findByText('Alpha');
    await user.click(screen.getByRole('checkbox', { name: 'Select Alpha' }));
    await user.click(screen.getByRole('checkbox', { name: 'Select Beta' }));
    await user.click(screen.getByRole('button', { name: 'Edit groups' }));

    await user.type(screen.getByLabelText('Group name'), 'Cold{Enter}');
    await user.click(screen.getByRole('button', { name: 'Apply changes' }));

    await waitFor(() => {
      expect(setSkillGroups).toHaveBeenCalledWith('Alpha', ['Archive', 'Cold']);
      expect(setSkillGroups).toHaveBeenCalledWith('Beta', ['Cold', 'Reference']);
    });
  });

  it('applies remove mode to each selected skill', async () => {
    const user = userEvent.setup();
    const setSkillGroups = vi.spyOn(api, 'setSkillGroups').mockResolvedValue({ success: true });
    renderResources([
      makeSkill({ name: 'Alpha', flatName: 'Alpha', groups: ['Archive', 'Cold'] }),
      makeSkill({ name: 'Beta', flatName: 'Beta', groups: ['Cold', 'Reference'] }),
    ]);

    await screen.findByText('Alpha');
    await user.click(screen.getByRole('checkbox', { name: 'Select Alpha' }));
    await user.click(screen.getByRole('checkbox', { name: 'Select Beta' }));
    await user.click(screen.getByRole('button', { name: 'Edit groups' }));
    await user.click(screen.getByRole('radio', { name: 'Remove groups' }));
    await user.type(screen.getByLabelText('Group name'), 'Cold{Enter}');
    await user.click(screen.getByRole('button', { name: 'Apply changes' }));

    await waitFor(() => {
      expect(setSkillGroups).toHaveBeenCalledWith('Alpha', ['Archive']);
      expect(setSkillGroups).toHaveBeenCalledWith('Beta', ['Reference']);
    });
  });

  it('applies replace mode to each selected skill', async () => {
    const user = userEvent.setup();
    const setSkillGroups = vi.spyOn(api, 'setSkillGroups').mockResolvedValue({ success: true });
    renderResources([
      makeSkill({ name: 'Alpha', flatName: 'Alpha', groups: ['Archive'] }),
      makeSkill({ name: 'Beta', flatName: 'Beta', groups: ['Reference'] }),
    ]);

    await screen.findByText('Alpha');
    await user.click(screen.getByRole('checkbox', { name: 'Select Alpha' }));
    await user.click(screen.getByRole('checkbox', { name: 'Select Beta' }));
    await user.click(screen.getByRole('button', { name: 'Edit groups' }));
    await user.click(screen.getByRole('radio', { name: 'Replace all groups' }));
    await user.type(screen.getByLabelText('Group name'), 'Shared{Enter}');
    await user.click(screen.getByRole('button', { name: 'Apply changes' }));

    await waitFor(() => {
      expect(setSkillGroups).toHaveBeenCalledWith('Alpha', ['Shared']);
      expect(setSkillGroups).toHaveBeenCalledWith('Beta', ['Shared']);
    });
  });

  it('keeps the bulk groups editor open with draft values when a save fails', async () => {
    const user = userEvent.setup();
    vi.spyOn(api, 'setSkillGroups')
      .mockResolvedValueOnce({ success: true })
      .mockRejectedValueOnce(new Error('save failed'));
    renderResources([
      makeSkill({ name: 'Alpha', flatName: 'Alpha', groups: ['Archive'] }),
      makeSkill({ name: 'Beta', flatName: 'Beta', groups: ['Reference'] }),
    ]);

    await screen.findByText('Alpha');
    await user.click(screen.getByRole('checkbox', { name: 'Select Alpha' }));
    await user.click(screen.getByRole('checkbox', { name: 'Select Beta' }));
    await user.click(screen.getByRole('button', { name: 'Edit groups' }));
    await user.type(screen.getByLabelText('Group name'), 'Cold{Enter}');
    await user.click(screen.getByRole('button', { name: 'Apply changes' }));

    await screen.findByText('save failed');

    expect(screen.getByText('Bulk edit groups')).toBeInTheDocument();
    expect(screen.getByText('Cold')).toBeInTheDocument();
  });

  it('select all only targets currently visible filtered skills', async () => {
    const user = userEvent.setup();
    renderResources([
      makeSkill({ name: 'Alpha', flatName: 'Alpha', groups: ['Archive'] }),
      makeSkill({ name: 'Beta', flatName: 'Beta', groups: ['Archive'] }),
      makeSkill({ name: 'Gamma', flatName: 'Gamma', groups: ['Reference'] }),
    ]);

    await screen.findByText('Alpha');
    await user.click(screen.getByText('All groups'));
    await user.click(screen.getByRole('option', { name: /Archive \(2\)/ }));
    await user.click(screen.getByRole('button', { name: 'Select all visible' }));

    expect(screen.getByText('2 selected')).toBeInTheDocument();
    expect(screen.queryByRole('checkbox', { name: 'Select Gamma', checked: true })).not.toBeInTheDocument();
  });
});
