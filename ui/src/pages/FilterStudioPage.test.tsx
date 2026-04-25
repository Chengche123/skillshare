import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import FilterStudioPage from './FilterStudioPage';
import { api, type Skill, type Target } from '../api/client';
import { ToastProvider } from '../components/Toast';
import { I18nProvider, LOCALE_STORAGE_KEY } from '../i18n';

vi.mock('react-virtuoso', () => ({
  Virtuoso: ({ totalCount, itemContent }: { totalCount: number; itemContent: (index: number) => React.ReactNode }) => (
    <div>{Array.from({ length: totalCount }, (_, index) => <div key={index}>{itemContent(index)}</div>)}</div>
  ),
}));

function makeTarget(overrides: Partial<Target> = {}): Target {
  return {
    name: 'claude',
    path: '/targets/claude',
    mode: 'merge',
    targetNaming: 'flat',
    status: 'synced',
    linkedCount: 0,
    localCount: 0,
    include: [],
    exclude: [],
    expectedSkillCount: 0,
    ...overrides,
  };
}

function makeSkill(overrides: Partial<Skill>): Skill {
  const flatName = overrides.flatName ?? overrides.name ?? 'skill';
  return {
    name: overrides.name ?? flatName,
    flatName,
    kind: 'skill',
    relPath: flatName,
    sourcePath: `/skills/${flatName}`,
    isInRepo: false,
    ...overrides,
  };
}

function renderPage(initialEntry = '/targets/claude/filters?kind=skill') {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <ToastProvider>
          <MemoryRouter initialEntries={[initialEntry]}>
            <Routes>
              <Route path="/targets/:name/filters" element={<FilterStudioPage />} />
              <Route path="/targets" element={<div>Targets</div>} />
            </Routes>
          </MemoryRouter>
        </ToastProvider>
      </I18nProvider>
    </QueryClientProvider>,
  );
}

describe('FilterStudioPage custom group preview filter', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
    localStorage.setItem(LOCALE_STORAGE_KEY, 'en');
    vi.spyOn(api, 'listTargets').mockResolvedValue({ targets: [makeTarget()], sourceSkillCount: 3 });
  });

  it('shows a skill group filter and narrows preview rows without changing save payload', async () => {
    vi.spyOn(api, 'listSkills').mockResolvedValue({
      resources: [
        makeSkill({ name: 'Alpha', flatName: 'alpha', groups: ['Archive'] }),
        makeSkill({ name: 'Beta', flatName: 'beta', groups: ['Archive'] }),
        makeSkill({ name: 'Gamma', flatName: 'gamma', groups: ['Reference'] }),
      ],
    });
    vi.spyOn(api, 'previewSyncMatrix').mockResolvedValue({
      entries: [
        { skill: 'alpha', target: 'claude', status: 'synced', reason: '' },
        { skill: 'beta', target: 'claude', status: 'excluded', reason: 'beta-*' },
        { skill: 'gamma', target: 'claude', status: 'synced', reason: '' },
      ],
    });
    const updateTarget = vi.spyOn(api, 'updateTarget').mockResolvedValue({ success: true });

    const user = userEvent.setup();
    renderPage();

    await screen.findByText('alpha');
    await user.click(screen.getByText('All groups'));
    await user.click(screen.getByRole('option', { name: /Archive \(2\)/ }));

    expect(screen.getByText('alpha')).toBeInTheDocument();
    expect(screen.getByText('beta')).toBeInTheDocument();
    expect(screen.queryByText('gamma')).not.toBeInTheDocument();

    const [includeInput] = screen.getAllByPlaceholderText('Type pattern + Enter');
    await user.type(includeInput, 'alpha{enter}');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(updateTarget).toHaveBeenCalledWith('claude', { include: ['alpha'], exclude: [] });
    });
  });

  it('hides the group filter in agent mode', async () => {
    const listSkills = vi.spyOn(api, 'listSkills').mockResolvedValue({ resources: [] });
    vi.spyOn(api, 'previewSyncMatrix').mockResolvedValue({ entries: [] });

    renderPage('/targets/claude/filters?kind=agent');

    await waitFor(() => {
      expect(listSkills).not.toHaveBeenCalled();
    });
    expect(screen.queryByText('All groups')).not.toBeInTheDocument();
  });
});