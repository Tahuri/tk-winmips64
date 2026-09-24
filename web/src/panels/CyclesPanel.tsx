// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { stageColor, textOn, C } from '../lib/colors';
import { useScrollBox, visibleRange } from '../components/useVirtual';
import type { HistEntry } from '../sim/types';

const RH = 20; // row height
const CW = 38; // cycle column width
const LW = 190; // instruction column width
const HH = 22; // header height

export function cellLabel(e: HistEntry, j: number, causeLabel: (c: string) => string): string {
  const c = e.cells[j];
  if (!c) return '';
  if (c.cause) return causeLabel(c.cause);
  const prev = j > 0 ? e.cells[j - 1] : undefined;
  if (prev && prev.stage === c.stage && c.stage !== 'WB') return '';
  return c.stage === 'MEM' ? 'MEM' : c.stage;
}

export function CyclesPanel() {
  const { t } = useTranslation();
  const history = useSim((s) => s.snapshot?.history ?? null);
  const cycles = useSim((s) => s.snapshot?.cycles ?? 0);
  const ref = useRef<HTMLDivElement>(null);
  const box = useScrollBox(ref);
  const entries = history ?? [];

  const { base, ncols } = useMemo(() => {
    if (entries.length === 0) return { base: 0, ncols: 0 };
    let lo = Infinity;
    let hi = -Infinity;
    for (const e of entries) {
      lo = Math.min(lo, e.startCycle);
      hi = Math.max(hi, e.startCycle + e.cells.length - 1);
    }
    return { base: lo, ncols: hi - lo + 1 };
  }, [entries]);

  // Keep the newest cycle and instruction in view (as CCyclesView::OnUpdate does).
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const x = ncols * CW + LW;
    if (x > el.scrollLeft + el.clientWidth) el.scrollLeft = x - el.clientWidth + CW;
    const y = entries.length * RH + HH;
    if (y > el.scrollTop + el.clientHeight) el.scrollTop = y - el.clientHeight + RH;
  }, [cycles, ncols, entries.length]);

  if (entries.length === 0) return <div className="panel-empty">{t('cycles.empty')}</div>;

  const rows = visibleRange(Math.max(0, box.top - HH), box.height, RH, entries.length);
  const cols = visibleRange(Math.max(0, box.left - LW), box.width, CW, ncols);
  const causeLabel = (c: string) => t(`cycles.cause.${c}`, { defaultValue: c });
  const totalH = HH + entries.length * RH;

  const headers = [];
  for (let c = cols.start; c < cols.end; c++) {
    headers.push(<div key={c} className="cy-head" style={{ left: c * CW, width: CW }}>{base + c}</div>);
  }
  const labels = [];
  const cells = [];
  for (let r = rows.start; r < rows.end; r++) {
    const e = entries[r];
    const last = e.cells[e.cells.length - 1];
    const stalled = last?.cause === 'raw' || last?.cause === 'waw' || last?.cause === 'structural';
    labels.push(
      <div key={r} className="cy-label" style={{ top: HH + r * RH, height: RH, color: stalled ? 'var(--stall-text)' : undefined }} title={e.mnemonic}>
        {e.mnemonic}
      </div>,
    );
    const off = e.startCycle - base;
    const from = Math.max(cols.start, off);
    const to = Math.min(cols.end, off + e.cells.length);
    for (let c = from; c < to; c++) {
      const j = c - off;
      const cell = e.cells[j];
      if (!cell || !cell.stage) continue;
      const bg = stageColor(cell.stage) ?? C.grey;
      cells.push(
        <div
          key={`${r}:${c}`}
          className="cy-cell"
          style={{ top: HH + r * RH, left: c * CW, width: CW, height: RH, background: bg, color: textOn(bg) }}
          title={`${cell.stage}${cell.cause ? ` (${cell.cause})` : ''}`}
        >
          {cellLabel(e, j, causeLabel)}
        </div>,
      );
    }
  }

  return (
    <div className="scroller cycles-view mono" ref={ref} data-testid="cycles-view">
      <div className="cy-inner" style={{ height: totalH, width: LW + ncols * CW }}>
        <div className="cy-left" style={{ width: LW, height: totalH }}>
          <div className="cy-corner" style={{ height: HH }}>{t('cycles.instruction')}</div>
          {labels}
        </div>
        <div className="cy-right" style={{ width: ncols * CW, height: totalH }}>
          <div className="cy-header" style={{ height: HH, width: ncols * CW }}>{headers}</div>
          {cells}
        </div>
      </div>
    </div>
  );
}
