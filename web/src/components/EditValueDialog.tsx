// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from './Modal';

export interface EditRequest {
  title: string;
  hint: string;
  initial: string;
  /** Returns false (or a promise of false) when the value is invalid. */
  submit: (text: string) => boolean | Promise<boolean>;
}

export function EditValueDialog({ req, onClose }: { req: EditRequest; onClose: () => void }) {
  const { t } = useTranslation();
  const [text, setText] = useState(req.initial);
  const [error, setError] = useState(false);
  const ok = async () => {
    if (await req.submit(text)) onClose();
    else setError(true);
  };
  return (
    <Modal
      title={req.title}
      onClose={onClose}
      testId="edit-dialog"
      footer={
        <>
          <button onClick={onClose}>{t('common.cancel')}</button>
          <button className="primary" onClick={ok}>{t('common.ok')}</button>
        </>
      }
    >
      <label className="field">
        <span>{req.hint}</span>
        <input
          className="mono"
          value={text}
          autoFocus
          onChange={(e) => { setText(e.target.value); setError(false); }}
          onKeyDown={(e) => e.key === 'Enter' && ok()}
        />
      </label>
      {error && <div className="error-text">{t('edit.invalid')}</div>}
    </Modal>
  );
}
