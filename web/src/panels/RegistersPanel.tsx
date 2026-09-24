import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { useUi } from '../state/ui';
import { sourceColor } from '../lib/colors';
import { formatDouble, parseFloatInput, parseHexInput } from '../lib/format';
import { intRegName } from '../lib/regs';

export function RegistersPanel() {
  const { t } = useTranslation();
  const snapshot = useSim((s) => s.snapshot);
  const asNumbers = useSim((s) => s.config.registersAsNumbers);
  const setReg = useSim((s) => s.setReg);
  const setFReg = useSim((s) => s.setFReg);
  const openEdit = useUi((s) => s.openEdit);
  const regs = snapshot?.regs ?? [];
  const fregs = snapshot?.fregs ?? [];

  const editR = (i: number) => {
    if (i === 0) return;
    const name = intRegName(i, asNumbers);
    openEdit({
      title: t('edit.register', { name }),
      hint: t('edit.hexHint'),
      initial: regs[i]?.value ?? '0000000000000000',
      submit: async (text) => {
        const h = parseHexInput(text);
        return h !== null && (await setReg(i, h));
      },
    });
  };
  const editF = (i: number) => {
    openEdit({
      title: t('edit.register', { name: `F${i}` }),
      hint: t('edit.floatHint'),
      initial: String(fregs[i]?.value ?? 0),
      submit: async (text) => {
        const f = parseFloatInput(text);
        return f !== null && (await setFReg(i, f));
      },
    });
  };

  return (
    <div className="scroller regs-view mono" title={t('regs.editHint')} data-testid="regs-view">
      <div className="regs-grid">
        {Array.from({ length: 32 }, (_, i) => {
          const r = regs[i];
          const f = fregs[i];
          const rc = r ? sourceColor(r.source) : undefined;
          const fc = f ? sourceColor(f.source) : undefined;
          return (
            <div className="regs-row" key={i}>
              <span className={`reg${rc ? ' src' : ''}`} style={{ color: rc }} onDoubleClick={() => editR(i)} data-testid={`reg-r${i}`}>
                <span className="reg-name">{intRegName(i, asNumbers)}=</span>
                <span className="reg-val">{r?.value ?? '0000000000000000'}</span>
              </span>
              <span className={`reg${fc ? ' src' : ''}`} style={{ color: fc }} onDoubleClick={() => editF(i)} data-testid={`reg-f${i}`}>
                <span className="reg-name">f{i}=</span>
                <span className="reg-val">{formatDouble(f?.value ?? 0)}</span>
              </span>
            </div>
          );
        })}
      </div>
      {snapshot && <div className="muted small">FP CC = {snapshot.fpcc ? 1 : 0}</div>}
    </div>
  );
}
