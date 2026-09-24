// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from './Modal';
import { useSim } from '../state/store';

export function MultiDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const multi = useSim((s) => s.multiCycles);
  const setMulti = useSim((s) => s.setMultiCycles);
  const [n, setN] = useState(String(multi));
  const ok = () => {
    setMulti(Number(n));
    onClose();
  };
  return (
    <Modal
      title={t('multi.title')}
      onClose={onClose}
      footer={
        <>
          <button onClick={onClose}>{t('common.cancel')}</button>
          <button className="primary" onClick={ok}>{t('common.ok')}</button>
        </>
      }
    >
      <label className="field-row">
        <span>{t('multi.cycles')}</span>
        <input type="number" min={1} max={10000} value={n} onChange={(e) => setN(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && ok()} />
      </label>
    </Modal>
  );
}
