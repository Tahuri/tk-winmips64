// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { translateMessage } from '../i18n';
import { formatCpi, hex } from '../lib/format';

export function StatusBar() {
  const { t } = useTranslation();
  const status = useSim((s) => s.status);
  const snap = useSim((s) => s.snapshot);
  const text =
    status.kind === 'key'
      ? t(status.key, status.params)
      : status.messages.map((m) => translateMessage(t, m)).join('  ');
  return (
    <div className="statusbar" data-testid="statusbar">
      <span className="status-text" data-testid="status-text">{text}</span>
      {snap?.loaded && (
        <span className="status-right mono">
          <span data-testid="status-cycles">{t('stats.cycles')}: {snap.cycles}</span>
          <span>{t('stats.instructions')}: {snap.instructions}</span>
          <span>CPI: {formatCpi(snap.stats.cpi)}</span>
          <span>PC: {hex(snap.pc, 8)}</span>
        </span>
      )}
    </div>
  );
}
