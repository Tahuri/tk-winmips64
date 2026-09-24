// localStorage wrapper: every access is guarded (private mode, blocked storage, SSR/tests).
const PREFIX = 'wmips.';

export const storage = {
  get<T>(key: string): T | undefined {
    try {
      const raw = globalThis.localStorage?.getItem(PREFIX + key);
      return raw == null ? undefined : (JSON.parse(raw) as T);
    } catch {
      return undefined;
    }
  },
  set(key: string, value: unknown): void {
    try {
      globalThis.localStorage?.setItem(PREFIX + key, JSON.stringify(value));
    } catch {
      /* storage unavailable: ignore */
    }
  },
  remove(key: string): void {
    try {
      globalThis.localStorage?.removeItem(PREFIX + key);
    } catch {
      /* ignore */
    }
  },
};
