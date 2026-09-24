// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { useUi, type ThemePref } from '../state/ui';
import { actionLoadExample, actionOpen, actionSave, toggleConfigFlag } from './actions';
import { fetchExamples, type ExampleInfo } from '../lib/files';
import { LANGUAGES, setLanguage, type Lang } from '../i18n';

type Item =
  | { kind: 'item'; label: string; shortcut?: string; onClick: () => void; checked?: boolean; disabled?: boolean; testId?: string }
  | { kind: 'sep' }
  | { kind: 'note'; label: string };

function Menu({ name, label, items, open, setOpen }: { name: string; label: string; items: Item[]; open: string | null; setOpen: (n: string | null) => void }) {
  const isOpen = open === name;
  return (
    <div className="menu">
      <button
        className={`menu-button${isOpen ? ' open' : ''}`}
        onClick={() => setOpen(isOpen ? null : name)}
        onMouseEnter={() => open && open !== name && setOpen(name)}
        data-testid={`menu-${name}`}
      >
        {label}
      </button>
      {isOpen && (
        <div className="menu-list" role="menu">
          {items.map((it, i) =>
            it.kind === 'sep' ? (
              <div key={i} className="menu-sep" />
            ) : it.kind === 'note' ? (
              <div key={i} className="menu-note">{it.label}</div>
            ) : (
              <button
                key={i}
                role="menuitem"
                className="menu-item"
                disabled={it.disabled}
                data-testid={it.testId}
                onClick={() => { setOpen(null); it.onClick(); }}
              >
                <span className="menu-check">{it.checked ? '✓' : ''}</span>
                <span className="menu-label">{it.label}</span>
                <span className="menu-shortcut">{it.shortcut ?? ''}</span>
              </button>
            ),
          )}
        </div>
      )}
    </div>
  );
}

export function MenuBar() {
  const { t, i18n } = useTranslation();
  const [open, setOpen] = useState<string | null>(null);
  const [examples, setExamples] = useState<ExampleInfo[] | 'loading' | 'error' | null>(null);
  const ref = useRef<HTMLDivElement>(null);
  const sim = useSim();
  const ui = useUi();
  const cfg = sim.config;

  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(null);
    };
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, []);

  useEffect(() => {
    if (open !== 'examples' || examples !== null) return;
    setExamples('loading');
    fetchExamples().then(setExamples, () => setExamples('error'));
  }, [open, examples]);

  const file: Item[] = [
    { kind: 'item', label: t('menu.open'), shortcut: 'Ctrl+O', onClick: () => void actionOpen() },
    { kind: 'item', label: t('menu.save'), shortcut: 'Ctrl+S', onClick: () => void actionSave() },
    { kind: 'sep' },
    { kind: 'item', label: t('menu.assemble'), onClick: () => void sim.assemble() },
    { kind: 'item', label: t('menu.reset'), shortcut: 'Ctrl+Alt+R', onClick: () => void sim.reset(false) },
    { kind: 'item', label: t('menu.fullReset'), shortcut: 'Ctrl+Alt+F', onClick: () => void sim.reset(true) },
    { kind: 'item', label: t('menu.reload'), shortcut: 'F10', onClick: () => void sim.reload() },
  ];
  const exampleItems: Item[] =
    examples === 'loading' || examples === null
      ? [{ kind: 'note', label: t('examples.loading') }]
      : examples === 'error' || examples.length === 0
        ? [{ kind: 'note', label: t('examples.unavailable') }]
        : examples.map((ex) => ({ kind: 'item' as const, label: ex.name, onClick: () => void actionLoadExample(ex.name).catch(() => setExamples('error')) }));
  const exec: Item[] = [
    { kind: 'item', label: t('menu.single'), shortcut: 'F7', onClick: () => void sim.step(), disabled: sim.running },
    { kind: 'item', label: `${t('menu.multi')} (${sim.multiCycles})`, shortcut: 'F8', onClick: () => void sim.stepN(), disabled: sim.running },
    { kind: 'item', label: t('menu.runTo'), shortcut: 'F4', onClick: () => void sim.runTo(), disabled: sim.running },
    { kind: 'item', label: t('menu.stop'), onClick: () => sim.stop(), disabled: !sim.running },
  ];
  const conf: Item[] = [
    { kind: 'item', label: t('menu.architecture'), shortcut: 'Ctrl+Alt+A', onClick: () => ui.openDialog('config'), testId: 'menu-architecture' },
    { kind: 'item', label: t('menu.multiStep'), shortcut: 'Ctrl+Alt+T', onClick: () => ui.openDialog('multi') },
    { kind: 'sep' },
    { kind: 'item', label: t('menu.forwarding'), checked: cfg.forwarding, onClick: () => toggleConfigFlag('forwarding') },
    { kind: 'item', label: t('menu.btb'), checked: cfg.btb, onClick: () => toggleConfigFlag('btb') },
    { kind: 'item', label: t('menu.delaySlot'), checked: cfg.delaySlot, onClick: () => toggleConfigFlag('delaySlot') },
    { kind: 'item', label: t('menu.regsAsNumbers'), checked: cfg.registersAsNumbers, onClick: () => toggleConfigFlag('registersAsNumbers') },
  ];
  const win: Item[] = [
    { kind: 'item', label: t('menu.resetLayout'), onClick: () => ui.resetLayout?.(), testId: 'menu-reset-layout' },
    { kind: 'sep' },
    { kind: 'note', label: t('menu.theme') },
    ...(['system', 'light', 'dark'] as ThemePref[]).map((th) => ({
      kind: 'item' as const, label: t(`theme.${th}`), checked: ui.theme === th, onClick: () => ui.setTheme(th),
    })),
    { kind: 'sep' },
    { kind: 'note', label: t('menu.language') },
    ...LANGUAGES.map((l) => ({
      kind: 'item' as const, label: t(`lang.${l}`), checked: i18n.language === l, onClick: () => setLanguage(l as Lang),
    })),
  ];
  const help: Item[] = [{ kind: 'item', label: t('menu.about'), onClick: () => ui.openDialog('about') }];

  return (
    <div className="menubar" ref={ref}>
      <span className="brand">WinMIPS64</span>
      <Menu name="file" label={t('menu.file')} items={file} open={open} setOpen={setOpen} />
      <Menu name="examples" label={t('menu.examples')} items={exampleItems} open={open} setOpen={setOpen} />
      <Menu name="execute" label={t('menu.execute')} items={exec} open={open} setOpen={setOpen} />
      <Menu name="config" label={t('menu.config')} items={conf} open={open} setOpen={setOpen} />
      <Menu name="window" label={t('menu.window')} items={win} open={open} setOpen={setOpen} />
      <Menu name="help" label={t('menu.help')} items={help} open={open} setOpen={setOpen} />
    </div>
  );
}
