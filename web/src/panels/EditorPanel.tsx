// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { EditorView, basicSetup } from 'codemirror';
import { EditorState } from '@codemirror/state';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { lintGutter, setDiagnostics, type Diagnostic } from '@codemirror/lint';
import { tags } from '@lezer/highlight';
import { useSim } from '../state/store';
import { useUi } from '../state/ui';
import { mipsLanguage } from '../lib/mipsLanguage';
import { translateAsmError } from '../i18n';

const highlight = HighlightStyle.define([
  { tag: tags.comment, color: 'var(--cm-comment)', fontStyle: 'italic' },
  { tag: tags.labelName, color: 'var(--cm-label)', fontWeight: 'bold' },
  { tag: tags.keyword, color: 'var(--cm-directive)' },
  { tag: tags.controlKeyword, color: 'var(--cm-opcode)', fontWeight: 'bold' },
  { tag: tags.number, color: 'var(--cm-number)' },
  { tag: tags.string, color: 'var(--cm-string)' },
  { tag: tags.special(tags.variableName), color: 'var(--cm-register)' },
]);

const theme = EditorView.theme({
  '&': { height: '100%', backgroundColor: 'var(--bg)', color: 'var(--fg)' },
  '.cm-scroller': { fontFamily: 'var(--mono)', fontSize: '13px' },
  '.cm-gutters': { backgroundColor: 'var(--bg-2)', color: 'var(--fg-muted)', borderRight: '1px solid var(--border)' },
  '.cm-activeLine': { backgroundColor: 'var(--active-line)' },
  '.cm-activeLineGutter': { backgroundColor: 'var(--active-line)' },
  '.cm-cursor': { borderLeftColor: 'var(--fg)' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': { backgroundColor: 'var(--selection) !important' },
  '.cm-tooltip': { backgroundColor: 'var(--bg-2)', color: 'var(--fg)', border: '1px solid var(--border)' },
});

export function EditorPanel() {
  const { t, i18n } = useTranslation();
  const source = useSim((s) => s.source);
  const fileName = useSim((s) => s.fileName);
  const asmErrors = useSim((s) => s.asmErrors);
  const assembledSource = useSim((s) => s.assembledSource);
  const setSource = useSim((s) => s.setSource);
  const assemble = useSim((s) => s.assemble);
  const running = useSim((s) => s.running);
  const hostRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const saveTimer = useRef<number | undefined>(undefined);
  const lastLocal = useRef<string>(source);

  useEffect(() => {
    if (!hostRef.current) return;
    const view = new EditorView({
      parent: hostRef.current,
      state: EditorState.create({
        doc: useSim.getState().source,
        extensions: [
          basicSetup,
          mipsLanguage,
          syntaxHighlighting(highlight),
          lintGutter(),
          theme,
          EditorView.updateListener.of((u) => {
            if (!u.docChanged) return;
            const text = u.state.doc.toString();
            window.clearTimeout(saveTimer.current);
            saveTimer.current = window.setTimeout(() => {
              lastLocal.current = text;
              setSource(text);
            }, 150);
          }),
        ],
      }),
    });
    viewRef.current = view;
    useUi.setState({
      gotoLine: (line) => {
        const v = viewRef.current;
        if (!v) return;
        const l = v.state.doc.line(Math.min(Math.max(1, line), v.state.doc.lines));
        v.dispatch({ selection: { anchor: l.from }, scrollIntoView: true });
        v.focus();
      },
    });
    return () => {
      window.clearTimeout(saveTimer.current);
      const text = view.state.doc.toString();
      if (text !== useSim.getState().source) setSource(text);
      view.destroy();
      viewRef.current = null;
      useUi.setState({ gotoLine: null });
    };
  }, [setSource]);

  // External source changes (open file, example, reset): replace the document.
  useEffect(() => {
    const v = viewRef.current;
    if (!v) return;
    const cur = v.state.doc.toString();
    if (cur !== source && source !== lastLocal.current) {
      lastLocal.current = source;
      v.dispatch({ changes: { from: 0, to: cur.length, insert: source } });
    }
  }, [source]);

  // Diagnostics from the assembler.
  useEffect(() => {
    const v = viewRef.current;
    if (!v) return;
    const doc = v.state.doc;
    const diags: Diagnostic[] = asmErrors
      .filter((e) => e.line >= 1 && e.line <= doc.lines)
      .map((e) => {
        const l = doc.line(e.line);
        return { from: l.from, to: l.to, severity: 'error', message: translateAsmError(t, e) };
      });
    v.dispatch(setDiagnostics(v.state, diags));
  }, [asmErrors, t, i18n.language]);

  const flushAndAssemble = async () => {
    const v = viewRef.current;
    window.clearTimeout(saveTimer.current);
    if (v) {
      lastLocal.current = v.state.doc.toString();
      setSource(lastLocal.current);
    }
    await assemble();
  };

  const modified = assembledSource !== null && assembledSource !== source;
  const gotoLine = useUi((s) => s.gotoLine);

  return (
    <div className="panel-col editor-panel">
      <div className="panel-toolbar">
        <button className="primary" onClick={flushAndAssemble} disabled={running} data-testid="assemble">
          {t('editor.assemble')}
        </button>
        <span className="mono small">{fileName}{modified ? ` (${t('editor.modified')})` : ''}</span>
      </div>
      <div className="editor-host" ref={hostRef} data-testid="editor" />
      <div className="asm-errors" data-testid="asm-errors">
        {asmErrors.length === 0 ? (
          <span className="muted small">{t('editor.noErrors')}</span>
        ) : (
          <>
            <div className="small"><strong>{t('editor.errors')}</strong></div>
            <ul>
              {asmErrors.map((e, i) => (
                <li key={i}>
                  <button className="link" onClick={() => gotoLine?.(e.line)}>{t('editor.line', { line: e.line })}</button>
                  {': '}{translateAsmError(t, e)} — <code>{e.text.trim()}</code>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
    </div>
  );
}
