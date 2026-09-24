// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useCallback, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  DockviewReact, themeDark, themeLight,
  type DockviewApi, type DockviewReadyEvent, type IDockviewPanelHeaderProps, type IDockviewPanelProps,
} from 'dockview-react';
import { CodePanel } from '../panels/CodePanel';
import { RegistersPanel } from '../panels/RegistersPanel';
import { DataPanel } from '../panels/DataPanel';
import { StatsPanel } from '../panels/StatsPanel';
import { PipelinePanel } from '../panels/PipelinePanel';
import { CyclesPanel } from '../panels/CyclesPanel';
import { TerminalPanel } from '../panels/TerminalPanel';
import { EditorPanel } from '../panels/EditorPanel';
import { storage } from '../lib/storage';
import { useUi } from '../state/ui';

export const PANEL_IDS = ['editor', 'code', 'registers', 'data', 'statistics', 'pipeline', 'cycles', 'terminal'] as const;

const wrap = (C: () => React.ReactElement) => function Panel(_: IDockviewPanelProps) {
  return <div className="panel-root"><C /></div>;
};

const components = {
  editor: wrap(EditorPanel),
  code: wrap(CodePanel),
  registers: wrap(RegistersPanel),
  data: wrap(DataPanel),
  statistics: wrap(StatsPanel),
  pipeline: wrap(PipelinePanel),
  cycles: wrap(CyclesPanel),
  terminal: wrap(TerminalPanel),
};

function Tab(props: IDockviewPanelHeaderProps) {
  const { t } = useTranslation();
  return <div className="dv-tab-label" data-testid={`tab-${props.api.id}`}>{t(`panel.${props.api.id}`)}</div>;
}

const LAYOUT_KEY = 'layout.v1';

function defaultLayout(api: DockviewApi) {
  api.clear();
  api.addPanel({ id: 'editor', component: 'editor' });
  api.addPanel({ id: 'code', component: 'code', position: { referencePanel: 'editor', direction: 'within' } });
  const w = api.width || 1280;
  const h = api.height || 700;
  const side = Math.round(Math.min(460, Math.max(300, w * 0.3)));
  api.addPanel({ id: 'pipeline', component: 'pipeline', position: { direction: 'right' }, initialWidth: w - 2 * side });
  api.addPanel({ id: 'registers', component: 'registers', position: { direction: 'right' }, initialWidth: side });
  api.addPanel({ id: 'cycles', component: 'cycles', position: { referencePanel: 'pipeline', direction: 'below' }, initialHeight: Math.round(h * 0.6) });
  api.addPanel({ id: 'data', component: 'data', position: { referencePanel: 'registers', direction: 'below' }, initialHeight: Math.round(h * 0.42) });
  api.addPanel({ id: 'statistics', component: 'statistics', position: { referencePanel: 'data', direction: 'within' } });
  api.addPanel({ id: 'terminal', component: 'terminal', position: { referencePanel: 'cycles', direction: 'within' } });
  api.getPanel('editor')?.api.setActive();
  api.getPanel('data')?.api.setActive();
  api.getPanel('cycles')?.api.setActive();
  // Sizing is best-effort: apply once the grid has been laid out.
  setTimeout(() => {
    try {
      api.getPanel('editor')?.group.api.setSize({ width: side });
      api.getPanel('registers')?.group.api.setSize({ width: side });
      api.getPanel('data')?.group.api.setSize({ height: Math.round(h * 0.42) });
      api.getPanel('pipeline')?.group.api.setSize({ height: Math.round(h * 0.4) });
    } catch {
      /* ignore */
    }
  }, 0);
}

export function Dock({ dark }: { dark: boolean }) {
  const apiRef = useRef<DockviewApi | null>(null);

  const onReady = useCallback((e: DockviewReadyEvent) => {
    apiRef.current = e.api;
    const saved = storage.get<unknown>(LAYOUT_KEY);
    let restored = false;
    if (saved) {
      try {
        e.api.fromJSON(saved as Parameters<DockviewApi['fromJSON']>[0]);
        restored = PANEL_IDS.every((id) => e.api.getPanel(id));
      } catch {
        restored = false;
      }
    }
    if (!restored) defaultLayout(e.api);
    e.api.onDidLayoutChange(() => {
      try {
        storage.set(LAYOUT_KEY, e.api.toJSON());
      } catch {
        /* ignore */
      }
    });
    useUi.setState({
      resetLayout: () => {
        defaultLayout(e.api);
        storage.set(LAYOUT_KEY, e.api.toJSON());
      },
    });
  }, []);

  useEffect(() => () => useUi.setState({ resetLayout: null }), []);

  return (
    <DockviewReact
      className="dock"
      components={components}
      defaultTabComponent={Tab}
      onReady={onReady}
      theme={dark ? themeDark : themeLight}
    />
  );
}
