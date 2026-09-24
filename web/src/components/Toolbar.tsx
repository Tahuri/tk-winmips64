import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { actionOpen, actionSave } from './actions';
import { LANGUAGES, setLanguage, type Lang } from '../i18n';

export function Toolbar() {
  const { t, i18n } = useTranslation();
  const sim = useSim();
  const disabled = sim.running || !sim.booted;
  return (
    <div className="toolbar" role="toolbar">
      <button onClick={() => void actionOpen()} title={t('menu.open')} data-testid="tb-open">📂 <span className="tb-text">{t('menu.open')}</span></button>
      <button onClick={() => void actionSave()} title={t('menu.save')} data-testid="tb-save">💾 <span className="tb-text">{t('menu.save')}</span></button>
      <span className="tb-sep" />
      <button onClick={() => void sim.assemble()} disabled={disabled} title={t('menu.assemble')} data-testid="tb-assemble">⚙ <span className="tb-text">{t('menu.assemble')}</span></button>
      <button onClick={() => void sim.reset(false)} disabled={disabled} title={t('menu.reset')} data-testid="tb-reset">⟲ <span className="tb-text">{t('menu.reset')}</span></button>
      <button onClick={() => void sim.reset(true)} disabled={disabled} title={t('menu.fullReset')} data-testid="tb-full-reset">⟲⟲</button>
      <span className="tb-sep" />
      <button onClick={() => void sim.step()} disabled={disabled} title={`${t('menu.single')} (F7)`} data-testid="tb-step">▶︎1 <span className="tb-text">F7</span></button>
      <button onClick={() => void sim.stepN()} disabled={disabled} title={`${t('menu.multi')} (F8)`} data-testid="tb-multi">▶︎{sim.multiCycles} <span className="tb-text">F8</span></button>
      <button onClick={() => void sim.runTo()} disabled={disabled} title={`${t('menu.runTo')} (F4)`} data-testid="tb-run">⏩ <span className="tb-text">F4</span></button>
      <button onClick={() => sim.stop()} disabled={!sim.running} title={t('menu.stop')} data-testid="tb-stop">⏹ <span className="tb-text">{t('menu.stop')}</span></button>
      <span className="tb-flex" />
      <span className={`engine-badge ${sim.engineKind ?? ''}`} data-testid="engine-kind">{sim.engineKind ?? '…'}</span>
      <select
        value={i18n.language}
        onChange={(e) => setLanguage(e.target.value as Lang)}
        aria-label={t('menu.language')}
        data-testid="lang-select"
      >
        {LANGUAGES.map((l) => <option key={l} value={l}>{t(`lang.${l}`)}</option>)}
      </select>
    </div>
  );
}
