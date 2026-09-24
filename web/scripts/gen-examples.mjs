// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

// Copies testdata/programs/*.s into public/examples/ and writes index.json,
// so the example menu works on static hosting (GitHub Pages). Also copies
// LICENSE and NOTICE, which Apache-2.0 requires in every distribution.
import { copyFileSync, existsSync, mkdirSync, readdirSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const src = join(here, '..', '..', 'testdata', 'programs');
const out = join(here, '..', 'public', 'examples');

mkdirSync(join(here, '..', 'public'), { recursive: true });
for (const f of ['LICENSE', 'NOTICE']) {
  const p = join(here, '..', '..', f);
  if (existsSync(p)) copyFileSync(p, join(here, '..', 'public', f));
}

if (!existsSync(src)) {
  console.warn(`gen-examples: ${src} not found, skipping`);
  process.exit(0);
}
rmSync(out, { recursive: true, force: true });
mkdirSync(out, { recursive: true });
const list = readdirSync(src)
  .filter((f) => f.toLowerCase().endsWith('.s'))
  .sort()
  .map((name) => {
    copyFileSync(join(src, name), join(out, name));
    return { name, size: statSync(join(src, name)).size };
  });
writeFileSync(join(out, 'index.json'), JSON.stringify(list));
console.log(`gen-examples: ${list.length} programs`);
