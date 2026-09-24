import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { formatCpi } from '../lib/format';

export function StatsPanel() {
  const { t } = useTranslation();
  const s = useSim((st) => st.snapshot);
  const cfg = useSim((st) => st.config);
  const st = s?.stats;
  const onOff = (b: boolean) => (b ? t('stats.enabled') : t('stats.disabled'));
  const rows: [string, (string | number)[]][] = [
    [t('stats.execution'), []],
    [t('stats.cycles'), [s?.cycles ?? 0]],
    [t('stats.instructions'), [s?.instructions ?? 0]],
    [t('stats.cpi'), [formatCpi(st?.cpi ?? 0)]],
    [t('stats.stalls'), []],
    [t('stats.raw'), [st?.rawStalls ?? 0]],
    [t('stats.waw'), [st?.wawStalls ?? 0]],
    [t('stats.war'), [st?.warStalls ?? 0]],
    [t('stats.structural'), [st?.structuralStalls ?? 0]],
    [t('stats.branchTaken'), [st?.branchTakenStalls ?? 0]],
    [t('stats.branchMispred'), [st?.branchMispredictionStalls ?? 0]],
    [t('stats.memory'), []],
    [t('stats.loads'), [st?.loads ?? 0]],
    [t('stats.stores'), [st?.stores ?? 0]],
    [t('stats.size'), []],
    [t('stats.codeSize'), [t('stats.bytes', { n: st?.codeSize ?? 0 })]],
    [t('stats.dataSize'), [t('stats.bytes', { n: st?.dataSize ?? 0 })]],
    [t('stats.config'), []],
    [t('stats.buses'), [`${t('stats.bits', { n: cfg.codeBits })} / ${t('stats.bits', { n: cfg.dataBits })}`]],
    [t('stats.latencies'), [`${cfg.addLatency} / ${cfg.mulLatency} / ${cfg.divLatency}`]],
    [t('config.forwarding'), [onOff(cfg.forwarding)]],
    [t('config.delaySlot'), [onOff(cfg.delaySlot)]],
    [t('config.btb'), [onOff(cfg.btb)]],
  ];
  return (
    <div className="scroller stats-view" data-testid="stats-view">
      <table className="stats-table">
        <tbody>
          {rows.map(([label, vals], i) =>
            vals.length === 0 ? (
              <tr key={i} className="section"><th colSpan={2}>{label}</th></tr>
            ) : (
              <tr key={i}><td>{label}</td><td className="mono num">{vals[0]}</td></tr>
            ),
          )}
        </tbody>
      </table>
    </div>
  );
}
