import { create } from 'zustand';
import type { EditRequest } from '../components/EditValueDialog';
import { storage } from '../lib/storage';

export type ThemePref = 'system' | 'light' | 'dark';
export type DialogKind = 'config' | 'multi' | 'about' | null;

interface UiState {
  dialog: DialogKind;
  edit: EditRequest | null;
  theme: ThemePref;
  /** Set by the dock layout; resets panels to the default arrangement. */
  resetLayout: (() => void) | null;
  /** Set by the editor panel; moves the cursor to a 1-based line. */
  gotoLine: ((line: number) => void) | null;
  openDialog(d: DialogKind): void;
  openEdit(r: EditRequest | null): void;
  setTheme(t: ThemePref): void;
}

export function applyTheme(t: ThemePref) {
  if (typeof document === 'undefined') return;
  const dark = t === 'dark' || (t === 'system' && globalThis.matchMedia?.('(prefers-color-scheme: dark)').matches);
  document.documentElement.dataset.theme = dark ? 'dark' : 'light';
}

export const useUi = create<UiState>((set) => ({
  dialog: null,
  edit: null,
  theme: storage.get<ThemePref>('theme') ?? 'system',
  resetLayout: null,
  gotoLine: null,
  openDialog: (dialog) => set({ dialog }),
  openEdit: (edit) => set({ edit }),
  setTheme: (theme) => {
    storage.set('theme', theme);
    applyTheme(theme);
    set({ theme });
  },
}));
