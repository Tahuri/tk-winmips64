import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { useUi } from '../state/ui';
import { formatDouble, hex, parseFloatInput, parseIntInput, hex16ToSigned } from '../lib/format';
import type { DataWord } from '../sim/types';
import { storage } from '../lib/storage';

export function DataPanel() {
  const { t } = useTranslation();
  const snapshot = useSim((s) => s.snapshot);
  const setMem = useSim((s) => s.setMem);
  const setMemDouble = useSim((s) => s.setMemDouble);
  const openEdit = useUi((s) => s.openEdit);
  const [onlyUsed, setOnlyUsed] = useState<boolean>(() => storage.get<boolean>('dataOnlyUsed') ?? false);
  const data = snapshot?.data ?? [];
  const digits = Math.max(4, Math.ceil(Math.log2(Math.max(16, data.length * 8)) / 4));
  const rows = onlyUsed ? data.filter((d) => d.written || d.label || d.text) : data;

  const editInt = (d: DataWord) =>
    openEdit({
      title: t('edit.memory', { addr: hex(d.addr, digits) }),
      hint: t('edit.intHint'),
      initial: hex16ToSigned(d.value).toString(),
      submit: async (text) => {
        const h = parseIntInput(text);
        return h !== null && (await setMem(d.addr, h));
      },
    });
  const editDouble = (d: DataWord) =>
    openEdit({
      title: t('edit.memory', { addr: hex(d.addr, digits) }),
      hint: t('edit.floatHint'),
      initial: String(d.double),
      submit: async (text) => {
        const f = parseFloatInput(text);
        return f !== null && (await setMemDouble(d.addr, f));
      },
    });

  return (
    <div className="panel-col">
      <div className="panel-toolbar">
        <label className="check small">
          <input
            type="checkbox"
            checked={onlyUsed}
            onChange={(e) => { setOnlyUsed(e.target.checked); storage.set('dataOnlyUsed', e.target.checked); }}
          />
          {t('data.onlyWritten')}
        </label>
        <span className="muted small">{t('data.editHint')}</span>
      </div>
      <div className="scroller mono" data-testid="data-view">
        <table className="grid-table">
          <thead>
            <tr><th>{t('data.address')}</th><th>{t('data.value')}</th><th>{t('data.label')}</th><th>{t('data.source')}</th></tr>
          </thead>
          <tbody>
            {rows.map((d) => (
              <tr
                key={d.addr}
                className={d.written ? 'written' : ''}
                onDoubleClick={() => editInt(d)}
                onContextMenu={(e) => { e.preventDefault(); editDouble(d); }}
                title={`${formatDouble(d.double)}`}
              >
                <td>{hex(d.addr, digits)}</td>
                <td>{d.value}</td>
                <td>{d.label ? `${d.label}:` : ''}</td>
                <td className="pre">{d.text}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
