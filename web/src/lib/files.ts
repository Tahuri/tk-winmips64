// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

// Open/save .s files: File System Access API when available, <input type=file> / download otherwise.

interface FsFileHandle {
  name: string;
  getFile(): Promise<File>;
  createWritable(): Promise<{ write(data: string): Promise<void>; close(): Promise<void> }>;
}
interface FsWindow {
  showOpenFilePicker?: (opts: unknown) => Promise<FsFileHandle[]>;
  showSaveFilePicker?: (opts: unknown) => Promise<FsFileHandle>;
}

const PICKER_TYPES = [{ description: 'MIPS64 assembly', accept: { 'text/plain': ['.s', '.asm', '.txt'] } }];
let currentHandle: FsFileHandle | null = null;

export async function openSourceFile(): Promise<{ name: string; text: string } | null> {
  const w = window as unknown as FsWindow;
  if (w.showOpenFilePicker) {
    try {
      const [h] = await w.showOpenFilePicker({ types: PICKER_TYPES, multiple: false });
      const f = await h.getFile();
      currentHandle = h;
      return { name: f.name, text: await f.text() };
    } catch (e) {
      if (e instanceof DOMException && e.name === 'AbortError') return null;
      // fall through to the classic input
    }
  }
  return new Promise((resolve) => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.s,.asm,.txt,text/plain';
    input.onchange = async () => {
      const f = input.files?.[0];
      currentHandle = null;
      resolve(f ? { name: f.name, text: await f.text() } : null);
    };
    input.click();
  });
}

export async function saveSourceFile(name: string, text: string): Promise<string | null> {
  const w = window as unknown as FsWindow;
  if (w.showSaveFilePicker) {
    try {
      const h = currentHandle ?? (await w.showSaveFilePicker({ suggestedName: name, types: PICKER_TYPES }));
      const ws = await h.createWritable();
      await ws.write(text);
      await ws.close();
      currentHandle = h;
      return h.name;
    } catch (e) {
      if (e instanceof DOMException && e.name === 'AbortError') return null;
    }
  }
  const blob = new Blob([text], { type: 'text/plain' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = name || 'program.s';
  a.click();
  setTimeout(() => URL.revokeObjectURL(a.href), 1000);
  return name;
}

export interface ExampleInfo { name: string }

const BASE: string = import.meta.env?.BASE_URL ?? '/';

async function json(url: string): Promise<unknown> {
  const r = await fetch(url, { headers: { accept: 'application/json' } });
  if (!r.ok || !(r.headers.get('content-type') ?? '').includes('json')) throw new Error(`${url}: ${r.status}`);
  return r.json();
}

/**
 * Example programs: a static index generated at build time (works on GitHub
 * Pages) with the Go server API as fallback.
 */
export async function fetchExamples(): Promise<ExampleInfo[]> {
  let list: unknown;
  try {
    list = await json(`${BASE}examples/index.json`);
  } catch {
    list = await json('/api/examples');
  }
  if (!Array.isArray(list)) throw new Error('examples: bad payload');
  return list.filter((x): x is ExampleInfo => typeof x?.name === 'string');
}

export async function fetchExample(name: string): Promise<string> {
  const file = encodeURIComponent(name);
  let r = await fetch(`${BASE}examples/${file}`);
  if (!r.ok || (r.headers.get('content-type') ?? '').includes('text/html')) r = await fetch(`/api/examples/${file}`);
  if (!r.ok) throw new Error(`example ${name}: ${r.status}`);
  return r.text();
}
