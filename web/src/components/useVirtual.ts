// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useCallback, useEffect, useState, type RefObject } from 'react';

/** Minimal fixed-size virtualisation: tracks scroll position and viewport size of a scroller. */
export function useScrollBox(ref: RefObject<HTMLElement | null>) {
  const [box, setBox] = useState({ top: 0, left: 0, width: 800, height: 600 });
  const update = useCallback(() => {
    const el = ref.current;
    if (!el) return;
    setBox({ top: el.scrollTop, left: el.scrollLeft, width: el.clientWidth || 800, height: el.clientHeight || 600 });
  }, [ref]);
  // The scroller may mount after the hook (panels render an "empty" placeholder
  // first), so re-attach whenever the underlying element changes.
  const [el, setEl] = useState<HTMLElement | null>(null);
  useEffect(() => {
    if (ref.current !== el) setEl(ref.current);
  });
  useEffect(() => {
    if (!el) return;
    update();
    el.addEventListener('scroll', update, { passive: true });
    const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(update) : null;
    ro?.observe(el);
    return () => {
      el.removeEventListener('scroll', update);
      ro?.disconnect();
    };
  }, [el, update]);
  return box;
}

export function visibleRange(offset: number, size: number, item: number, count: number, overscan = 4) {
  const start = Math.max(0, Math.floor(offset / item) - overscan);
  const end = Math.min(count, Math.ceil((offset + size) / item) + overscan);
  return { start, end };
}
