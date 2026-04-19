import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

class MemoryStorage implements Storage {
  #items = new Map<string, string>();

  get length(): number {
    return this.#items.size;
  }

  clear(): void {
    this.#items.clear();
  }

  getItem(key: string): string | null {
    return this.#items.get(key) ?? null;
  }

  key(index: number): string | null {
    return [...this.#items.keys()][index] ?? null;
  }

  removeItem(key: string): void {
    this.#items.delete(key);
  }

  setItem(key: string, value: string): void {
    this.#items.set(key, String(value));
  }
}

// Node 25 exposes a non-browser Web Storage stub on globalThis that Vitest's jsdom
// environment does not override. Use an in-memory Storage shim for stable tests.
for (const key of ['localStorage', 'sessionStorage'] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: new MemoryStorage(),
  });
}

afterEach(() => {
  cleanup();
});
