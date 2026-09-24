// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

// Hex / number formatting helpers. 64-bit values travel as 16-hex-digit strings.

const MASK64 = (1n << 64n) - 1n;

export function hex(n: number, digits: number): string {
  return (n >>> 0).toString(16).padStart(digits, '0').slice(-digits);
}

export function bigToHex16(v: bigint): string {
  return (v & MASK64).toString(16).padStart(16, '0');
}

export function hex16ToBig(h: string): bigint {
  const clean = h.trim().replace(/^0x/i, '');
  if (!/^[0-9a-f]{1,16}$/i.test(clean)) throw new Error(`bad hex: ${h}`);
  return BigInt('0x' + clean);
}

/** Interprets a 16-hex string as a signed 64-bit two's complement integer. */
export function hex16ToSigned(h: string): bigint {
  return BigInt.asIntN(64, hex16ToBig(h));
}

export function doubleToHex16(f: number): string {
  const dv = new DataView(new ArrayBuffer(8));
  dv.setFloat64(0, f);
  return bigToHex16(dv.getBigUint64(0));
}

export function hex16ToDouble(h: string): number {
  const dv = new DataView(new ArrayBuffer(8));
  dv.setBigUint64(0, hex16ToBig(h));
  return dv.getFloat64(0);
}

/**
 * Parses user input for a register (hex): accepts "ff", "0xff", up to 16 digits.
 * Returns a normalised 16-hex string or null.
 */
export function parseHexInput(s: string): string | null {
  const clean = s.trim().replace(/^0x/i, '');
  if (!/^[0-9a-f]{1,16}$/i.test(clean)) return null;
  return clean.toLowerCase().padStart(16, '0');
}

/**
 * Parses an integer typed by the user: decimal (optionally negative) or 0x-prefixed hex.
 * Returns the 16-hex two's complement string or null when invalid / out of range.
 */
export function parseIntInput(s: string): string | null {
  const t = s.trim();
  if (/^0x[0-9a-f]{1,16}$/i.test(t)) return parseHexInput(t);
  if (!/^[-+]?\d+$/.test(t)) return null;
  const v = BigInt(t);
  if (v < -(1n << 63n) || v > MASK64) return null;
  return bigToHex16(v);
}

export function parseFloatInput(s: string): number | null {
  const t = s.trim();
  if (t === '') return null;
  const n = Number(t);
  return Number.isFinite(n) || /^[-+]?(inf|infinity|nan)$/i.test(t) ? n : null;
}

/** Formats a double like the original views (fixed with 6 decimals, exponent for extremes). */
export function formatDouble(f: number): string {
  if (Number.isNaN(f)) return 'NaN';
  if (!Number.isFinite(f)) return f > 0 ? 'Inf' : '-Inf';
  const a = Math.abs(f);
  if (a !== 0 && (a >= 1e15 || a < 1e-6)) return f.toExponential(6);
  return f.toFixed(6);
}

export function formatCpi(cpi: number): string {
  return Number.isFinite(cpi) ? cpi.toFixed(3) : '0.000';
}

export function base64ToBytes(b64: string): Uint8Array {
  if (!b64) return new Uint8Array(0);
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

export function bytesToBase64(bytes: Uint8Array): string {
  let s = '';
  const CHUNK = 0x8000;
  for (let i = 0; i < bytes.length; i += CHUNK) {
    s += String.fromCharCode(...bytes.subarray(i, i + CHUNK));
  }
  return btoa(s);
}
