import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSim } from '../state/store';
import { base64ToBytes } from '../lib/format';

export function TerminalPanel() {
  const { t } = useTranslation();
  const terminal = useSim((s) => s.snapshot?.terminal ?? '');
  const status = useSim((s) => s.snapshot?.status);
  const inputKind = useSim((s) => s.snapshot?.inputKind);
  const screen = useSim((s) => s.snapshot?.screen);
  const sendInput = useSim((s) => s.sendInput);
  const outRef = useRef<HTMLPreElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [text, setText] = useState('');
  const [error, setError] = useState(false);
  const waiting = status === 'waiting_input';

  useEffect(() => {
    const el = outRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [terminal]);

  useEffect(() => {
    if (waiting) inputRef.current?.focus();
  }, [waiting]);

  useEffect(() => {
    const cv = canvasRef.current;
    if (!cv || !screen) return;
    const w = screen.width || 50;
    const h = screen.height || 50;
    cv.width = w;
    cv.height = h;
    const ctx = cv.getContext('2d');
    if (!ctx) return;
    const rgb = base64ToBytes(screen.pixels);
    const img = ctx.createImageData(w, h);
    for (let i = 0, j = 0; i < w * h; i++, j += 3) {
      img.data[i * 4] = rgb[j] ?? 0;
      img.data[i * 4 + 1] = rgb[j + 1] ?? 0;
      img.data[i * 4 + 2] = rgb[j + 2] ?? 0;
      img.data[i * 4 + 3] = 255;
    }
    ctx.putImageData(img, 0, 0);
  }, [screen]);

  const submit = async () => {
    const v = inputKind === 'char' ? text.slice(0, 1) : text.trim();
    if (inputKind !== 'char' && !/^[-+]?(\d+(\.\d*)?|\.\d+)([eE][-+]?\d+)?$/.test(v)) {
      setError(true);
      return;
    }
    if (inputKind === 'char' && v.length === 0) return;
    if (await sendInput(v)) {
      setText('');
      setError(false);
    }
  };

  return (
    <div className="terminal-view" data-testid="terminal-view">
      <div className="term-main">
        <pre className="term-out mono" ref={outRef} data-testid="terminal-output">{terminal}</pre>
        {waiting && (
          <div className="term-input">
            <label>{t('terminal.input')}</label>
            <input
              ref={inputRef}
              className="mono"
              value={text}
              maxLength={inputKind === 'char' ? 1 : 64}
              placeholder={inputKind === 'char' ? t('terminal.charPlaceholder') : t('terminal.numberPlaceholder')}
              onChange={(e) => { setText(e.target.value); setError(false); }}
              onKeyDown={(e) => { if (e.key === 'Enter') void submit(); e.stopPropagation(); }}
              data-testid="terminal-input"
            />
            <button onClick={submit}>{t('terminal.send')}</button>
            {error && <span className="error-text">{t('terminal.invalidNumber')}</span>}
          </div>
        )}
      </div>
      <div className="term-screen">
        <div className="muted small">{t('terminal.graphics')}</div>
        <canvas ref={canvasRef} width={50} height={50} className="pixel-canvas" data-testid="terminal-canvas" />
      </div>
    </div>
  );
}
