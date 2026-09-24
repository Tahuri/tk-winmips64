import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from './Modal';
import { CONFIG_LIMITS, type Config } from '../sim/types';
import { useSim } from '../state/store';

const NUM_FIELDS: { key: keyof typeof CONFIG_LIMITS; label: string; unit?: 'bits' }[] = [
  { key: 'codeBits', label: 'config.codeBits', unit: 'bits' },
  { key: 'dataBits', label: 'config.dataBits', unit: 'bits' },
  { key: 'addLatency', label: 'config.addLatency' },
  { key: 'mulLatency', label: 'config.mulLatency' },
  { key: 'divLatency', label: 'config.divLatency' },
];

export function ConfigDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const config = useSim((s) => s.config);
  const applyConfig = useSim((s) => s.applyConfig);
  const [cfg, setCfg] = useState<Config>(config);
  const setBool = (k: 'forwarding' | 'delaySlot' | 'btb' | 'registersAsNumbers', v: boolean) => {
    const next = { ...cfg, [k]: v };
    if (k === 'delaySlot' && v) next.btb = false;
    if (k === 'btb' && v) next.delaySlot = false;
    setCfg(next);
  };
  const ok = async () => {
    onClose();
    await applyConfig(cfg);
  };
  return (
    <Modal
      title={t('config.title')}
      onClose={onClose}
      testId="config-dialog"
      footer={
        <>
          <button onClick={onClose}>{t('common.cancel')}</button>
          <button className="primary" onClick={ok} data-testid="config-ok">{t('common.ok')}</button>
        </>
      }
    >
      <div className="config-grid">
        {NUM_FIELDS.map(({ key, label, unit }) => {
          const [min, max] = CONFIG_LIMITS[key];
          return (
            <label key={key} className="field-row">
              <span>{t(label)}</span>
              <select value={cfg[key]} onChange={(e) => setCfg({ ...cfg, [key]: Number(e.target.value) })} data-testid={`cfg-${key}`}>
                {Array.from({ length: max - min + 1 }, (_, i) => min + i).map((v) => (
                  <option key={v} value={v}>
                    {unit === 'bits' ? `${v} (${t('stats.bytes', { n: key === 'codeBits' || key === 'dataBits' ? 1 << v : v })})` : v}
                  </option>
                ))}
              </select>
              <small className="muted">{t('config.range', { min, max })}</small>
            </label>
          );
        })}
      </div>
      <div className="config-checks">
        {(['forwarding', 'delaySlot', 'btb', 'registersAsNumbers'] as const).map((k) => (
          <label key={k} className="check">
            <input type="checkbox" checked={cfg[k]} onChange={(e) => setBool(k, e.target.checked)} data-testid={`cfg-${k}`} />
            {t(k === 'registersAsNumbers' ? 'config.regsAsNumbers' : `config.${k}`)}
          </label>
        ))}
        <small className="muted">{t('config.exclusive')}</small>
      </div>
      <div className="warning-text">{t('config.warning')}</div>
    </Modal>
  );
}
