// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { C, textOn } from '../lib/colors';
import type { StageSlot } from '../sim/types';

function Box({ slot, color, label, id, sub }: { slot: StageSlot | undefined; color: string; label: string; id: string; sub?: boolean }) {
  const active = !!slot?.active;
  return (
    <div
      className={`pipe-box${active ? ' active' : ''}${sub ? ' sub' : ''}`}
      style={active ? { background: color, color: textOn(color), borderColor: color } : undefined}
      data-testid={`pipe-${id}`}
      data-active={active ? '1' : '0'}
      title={active ? slot?.mnemonic : undefined}
    >
      <span className="pipe-label">{label}</span>
      <span className="pipe-mnem mono">{active ? slot?.mnemonic : ''}</span>
    </div>
  );
}

export function PipelinePanel() {
  const { t } = useTranslation();
  const s = useSim((st) => st.snapshot);
  const cfg = useSim((st) => st.config);
  const p = s?.pipeline;
  const add = p?.add ?? Array.from({ length: cfg.addLatency }, () => undefined);
  const mul = p?.mul ?? Array.from({ length: cfg.mulLatency }, () => undefined);
  const hostRef = useRef<HTMLDivElement>(null);
  const diagRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(1);

  // Scale the diagram down to fit narrow panels (never up).
  useLayoutEffect(() => {
    const host = hostRef.current;
    const diag = diagRef.current;
    if (!host || !diag || typeof ResizeObserver === 'undefined') return;
    const fit = () => {
      const avail = host.clientWidth - 20;
      const natural = diag.scrollWidth;
      setScale(natural > 0 ? Math.max(0.5, Math.min(1, avail / natural)) : 1);
    };
    const ro = new ResizeObserver(fit);
    ro.observe(host);
    fit();
    return () => ro.disconnect();
  }, [add.length, mul.length]);

  return (
    <div className="scroller pipeline-view" data-testid="pipeline-view" ref={hostRef}>
      <div className="pipe-diagram" ref={diagRef} style={{ transform: `scale(${scale})`, transformOrigin: 'top left' }}>
        <div className="pipe-col">
          <Box slot={p?.if} color={C.yellow} label="IF" id="if" />
        </div>
        <div className="pipe-arrow">→</div>
        <div className="pipe-col">
          <Box slot={p?.id} color={C.cyan} label="ID" id="id" />
        </div>
        <div className="pipe-arrow">→</div>
        <div className="pipe-units">
          <div className="pipe-unit">
            <Box slot={p?.ex} color={C.red} label="EX" id="ex" />
          </div>
          <div className="pipe-unit" title={t('pipeline.fpMul')}>
            {mul.map((m, i) => <Box key={i} slot={m} color={C.dcyan} label={`M${i + 1}`} id={`mul${i}`} sub />)}
          </div>
          <div className="pipe-unit" title={t('pipeline.fpAdd')}>
            {add.map((a, i) => <Box key={i} slot={a} color={C.dgreen} label={`A${i + 1}`} id={`add${i}`} sub />)}
          </div>
          <div className="pipe-unit" title={t('pipeline.fpDiv')}>
            <Box slot={p?.div} color={C.dyellow} label={`DIV (${cfg.divLatency})`} id="div" />
          </div>
        </div>
        <div className="pipe-arrow">→</div>
        <div className="pipe-col">
          <Box slot={p?.mem} color={C.green} label="MEM" id="mem" />
        </div>
        <div className="pipe-arrow">→</div>
        <div className="pipe-col">
          <Box slot={p?.wb} color={C.magenta} label="WB" id="wb" />
        </div>
      </div>
    </div>
  );
}
