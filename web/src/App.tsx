// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { MenuBar } from './components/MenuBar';
import { Toolbar } from './components/Toolbar';
import { StatusBar } from './components/StatusBar';
import { Dock } from './components/Dock';
import { ConfigDialog } from './components/ConfigDialog';
import { MultiDialog } from './components/MultiDialog';
import { EditValueDialog } from './components/EditValueDialog';
import { Modal } from './components/Modal';
import { actionOpen, actionSave } from './components/actions';
import { useSim } from './state/store';
import { applyTheme, useUi } from './state/ui';
import { createWorkerClient } from './sim/client';

function useDarkMode(): boolean {
  const theme = useUi((s) => s.theme);
  const [systemDark, setSystemDark] = useState(() => !!globalThis.matchMedia?.('(prefers-color-scheme: dark)').matches);
  useEffect(() => {
    const mq = globalThis.matchMedia?.('(prefers-color-scheme: dark)');
    if (!mq) return;
    const on = () => setSystemDark(mq.matches);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, []);
  useEffect(() => applyTheme(theme), [theme, systemDark]);
  return theme === 'dark' || (theme === 'system' && systemDark);
}

function useShortcuts() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const sim = useSim.getState();
      const ui = useUi.getState();
      if (ui.dialog || ui.edit) return;
      const inEditor = (e.target as HTMLElement | null)?.closest?.('.cm-editor, input, textarea, select');
      const ctrl = e.ctrlKey || e.metaKey;
      if (e.key === 'F7') { e.preventDefault(); void sim.step(); return; }
      if (e.key === 'F8') { e.preventDefault(); void sim.stepN(); return; }
      if (e.key === 'F4') { e.preventDefault(); void sim.runTo(); return; }
      if (e.key === 'F10') { e.preventDefault(); void sim.reload(); return; }
      if (e.key === 'Escape' && sim.running) { sim.stop(); return; }
      if (ctrl && !e.altKey && e.key.toLowerCase() === 'o') { e.preventDefault(); void actionOpen(); return; }
      if (ctrl && !e.altKey && e.key.toLowerCase() === 's') { e.preventDefault(); void actionSave(); return; }
      if (ctrl && e.altKey && !inEditor) {
        const k = e.code;
        if (k === 'KeyR') { e.preventDefault(); void sim.reset(false); }
        else if (k === 'KeyF') { e.preventDefault(); void sim.reset(true); }
        else if (k === 'KeyA') { e.preventDefault(); ui.openDialog('config'); }
        else if (k === 'KeyT') { e.preventDefault(); ui.openDialog('multi'); }
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);
}

export function App() {
  const { t } = useTranslation();
  const dark = useDarkMode();
  const dialog = useUi((s) => s.dialog);
  const edit = useUi((s) => s.edit);
  const openDialog = useUi((s) => s.openDialog);
  const openEdit = useUi((s) => s.openEdit);
  useShortcuts();

  useEffect(() => {
    const mock = new URLSearchParams(window.location.search).get('mock') === '1';
    const client = createWorkerClient();
    void useSim.getState().boot(client, mock);
    return () => client.terminate();
  }, []);

  return (
    <div className="app">
      <MenuBar />
      <Toolbar />
      <div className="dock-host">
        <Dock dark={dark} />
      </div>
      <StatusBar />
      {dialog === 'config' && <ConfigDialog onClose={() => openDialog(null)} />}
      {dialog === 'multi' && <MultiDialog onClose={() => openDialog(null)} />}
      {dialog === 'about' && (
        <Modal title={t('about.title')} onClose={() => openDialog(null)} footer={<button className="primary" onClick={() => openDialog(null)}>{t('common.ok')}</button>}>
          <p><strong>WinMIPS64 V1.60</strong></p>
          <p>{t('about.text')}</p>
          <h4>{t('about.courseTitle')}</h4>
          <p>{t('about.course')}</p>
          <ul className="about-links">
            <li><a href="https://weblidi.info.unlp.edu.ar/catedras/arquitectura/" target="_blank" rel="noopener noreferrer">{t('about.courseSite')}</a></li>
            <li><a href="https://weblidi.info.unlp.edu.ar/catedras/arquitectura/?page=programa" target="_blank" rel="noopener noreferrer">{t('about.courseProgram')}</a></li>
            <li><a href="http://sedici.unlp.edu.ar/handle/10915/122754" target="_blank" rel="noopener noreferrer">{t('about.coursePaper')}</a></li>
            <li><a href="https://github.com/Tahuri/tk-winmips64" target="_blank" rel="noopener noreferrer">Tahuri/tk-winmips64</a> — {t('about.repo')}</li>
          </ul>
          <h4>{t('about.creditsTitle')}</h4>
          <p>{t('about.credits')}</p>
          <ul className="about-links">
            <li><a href="https://github.com/mcarrickscott/WinMIPS64" target="_blank" rel="noopener noreferrer">mcarrickscott/WinMIPS64</a> — Mike Scott (Apache-2.0)</li>
            <li><a href="https://github.com/AndoniZubimendi/WinMIPS64" target="_blank" rel="noopener noreferrer">AndoniZubimendi/WinMIPS64</a> — Andoni Zubimendi</li>
            <li><a href="https://www.apache.org/licenses/LICENSE-2.0" target="_blank" rel="noopener noreferrer">Apache License 2.0</a></li>
          </ul>
        </Modal>
      )}
      {edit && <EditValueDialog req={edit} onClose={() => openEdit(null)} />}
    </div>
  );
}
