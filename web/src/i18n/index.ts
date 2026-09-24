import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { en } from './en';
import { es } from './es';
import { MESSAGE_CODES, ASM_ERROR_CODES, type AsmError, type Message } from '../sim/types';
import { storage } from '../lib/storage';

export const LANGUAGES = ['es', 'en'] as const;
export type Lang = (typeof LANGUAGES)[number];

export function detectLanguage(): Lang {
  const saved = storage.get<string>('lang');
  if (saved === 'es' || saved === 'en') return saved;
  const nav = typeof navigator !== 'undefined' ? navigator.languages ?? [navigator.language] : [];
  for (const l of nav) {
    const base = (l ?? '').slice(0, 2).toLowerCase();
    if (base === 'es' || base === 'en') return base;
  }
  return 'es';
}

export function initI18n(lng: Lang = detectLanguage()) {
  if (!i18n.isInitialized) {
    void i18n.use(initReactI18next).init({
      resources: { es: { translation: es }, en: { translation: en } },
      lng,
      fallbackLng: 'es',
      interpolation: { escapeValue: false },
      returnNull: false,
    });
  }
  return i18n;
}

export function setLanguage(lng: Lang) {
  storage.set('lang', lng);
  void i18n.changeLanguage(lng);
  if (typeof document !== 'undefined') document.documentElement.lang = lng;
}

type T = (key: string, opts?: Record<string, unknown>) => string;

export function translateMessage(t: T, m: Message): string {
  if ((MESSAGE_CODES as readonly string[]).includes(m.code)) {
    return t(`msg.${m.code}`, { stage: m.stage ?? '', reg: m.reg ?? '' });
  }
  return t('msg.unknown', { code: m.code });
}

export function translateAsmError(t: T, e: AsmError): string {
  if ((ASM_ERROR_CODES as readonly string[]).includes(e.code)) return t(`asm.${e.code}`);
  return t('asm.unknown', { code: e.code });
}

export default i18n;
