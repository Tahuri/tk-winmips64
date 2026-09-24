import { useEffect, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { C, textOn } from '../lib/colors';
import { hex } from '../lib/format';
import type { Snapshot } from '../sim/types';
import { useScrollBox, visibleRange } from '../components/useVirtual';

const ROW = 18;

/** Background colour per code address, same precedence as WinMIPS64View::OnDraw (later wins). */
export function codeColors(s: Snapshot | null): Map<number, string> {
  const m = new Map<number, string>();
  if (!s || !s.loaded) return m;
  const p = s.pipeline;
  m.set(s.pc, C.yellow);
  const put = (active: boolean, addr: number, color: string) => active && m.set(addr, color);
  put(p.if.active, p.if.addr, C.cyan);
  put(p.ex.active, p.ex.addr, C.red);
  put(p.mem.active, p.mem.addr, C.green);
  put(p.wb.active, p.wb.addr, C.magenta);
  p.add.forEach((a) => put(a.active, a.addr, C.dgreen));
  p.mul.forEach((a) => put(a.active, a.addr, C.dcyan));
  put(p.div.active, p.div.addr, C.dyellow);
  return m;
}

export function CodePanel() {
  const { t } = useTranslation();
  const program = useSim((s) => s.program);
  const snapshot = useSim((s) => s.snapshot);
  const toggleBreakpoint = useSim((s) => s.toggleBreakpoint);
  const ref = useRef<HTMLDivElement>(null);
  const box = useScrollBox(ref);
  const colors = useMemo(() => codeColors(snapshot), [snapshot]);
  const bps = useMemo(() => new Set(snapshot?.breakpoints ?? []), [snapshot]);
  const predicted = useMemo(() => new Set(snapshot?.predicted ?? []), [snapshot]);
  const lines = program?.code ?? [];
  const digits = Math.max(4, Math.ceil(Math.log2(Math.max(16, lines.length * 4)) / 4));
  const pc = snapshot?.pc ?? 0;

  // Auto-scroll so that the PC stays visible.
  useEffect(() => {
    const el = ref.current;
    if (!el || !snapshot?.loaded) return;
    const y = (pc / 4) * ROW;
    if (y < el.scrollTop || y + ROW > el.scrollTop + el.clientHeight) {
      el.scrollTop = Math.max(0, y - el.clientHeight / 3);
    }
  }, [pc, snapshot?.loaded]);

  if (!program) return <div className="panel-empty">{t('code.empty')}</div>;
  const { start, end } = visibleRange(box.top, box.height, ROW, lines.length);
  return (
    <div className="scroller mono code-view" ref={ref} title={t('code.toggleHint')} data-testid="code-view">
      <div style={{ height: lines.length * ROW, position: 'relative' }}>
        {lines.slice(start, end).map((l) => {
          const bg = colors.get(l.addr);
          const bp = bps.has(l.addr);
          return (
            <div
              key={l.addr}
              className={`code-row${bp ? ' bp' : ''}${l.used ? '' : ' unused'}`}
              style={{ top: (l.addr / 4) * ROW, height: ROW, background: bg, color: bg && !bp ? textOn(bg) : undefined }}
              onDoubleClick={() => toggleBreakpoint(l.addr)}
              data-addr={l.addr}
              data-stage-color={bg ?? ''}
            >
              <span className="c-addr">{hex(l.addr, digits)}</span>
              <span className="c-pred" title={predicted.has(l.addr) ? t('code.predicted') : undefined}>{predicted.has(l.addr) ? '«' : ' '}</span>
              <span className="c-word">{l.word}</span>
              <span className="c-text">{l.text}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
