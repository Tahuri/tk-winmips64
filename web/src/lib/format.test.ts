// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import {
  base64ToBytes, bigToHex16, bytesToBase64, doubleToHex16, formatCpi, formatDouble, hex, hex16ToBig,
  hex16ToDouble, hex16ToSigned, parseFloatInput, parseHexInput, parseIntInput,
} from './format';

describe('format helpers', () => {
  it('hex pads and truncates', () => {
    expect(hex(0x1c, 4)).toBe('001c');
    expect(hex(0xdeadbeef, 8)).toBe('deadbeef');
    expect(hex(0x12345, 4)).toBe('2345');
  });

  it('64-bit round trips', () => {
    expect(bigToHex16(-1n)).toBe('ffffffffffffffff');
    expect(bigToHex16(255n)).toBe('00000000000000ff');
    expect(hex16ToBig('ffffffffffffffff')).toBe((1n << 64n) - 1n);
    expect(hex16ToSigned('ffffffffffffffff')).toBe(-1n);
    expect(hex16ToSigned('7fffffffffffffff')).toBe((1n << 63n) - 1n);
    expect(() => hex16ToBig('xyz')).toThrow();
  });

  it('double <-> hex', () => {
    expect(doubleToHex16(1)).toBe('3ff0000000000000');
    expect(doubleToHex16(-2.5)).toBe('c004000000000000');
    expect(hex16ToDouble('4002000000000000')).toBe(2.25);
    expect(hex16ToDouble(doubleToHex16(Math.PI))).toBe(Math.PI);
  });

  it('parses user input', () => {
    expect(parseHexInput('ff')).toBe('00000000000000ff');
    expect(parseHexInput('0xFF')).toBe('00000000000000ff');
    expect(parseHexInput('12345678901234567')).toBeNull();
    expect(parseHexInput('zz')).toBeNull();
    expect(parseIntInput('-1')).toBe('ffffffffffffffff');
    expect(parseIntInput('10')).toBe('000000000000000a');
    expect(parseIntInput('0x10')).toBe('0000000000000010');
    expect(parseIntInput('1.5')).toBeNull();
    expect(parseIntInput('99999999999999999999999')).toBeNull();
    expect(parseFloatInput('1.5')).toBe(1.5);
    expect(parseFloatInput('-3e2')).toBe(-300);
    expect(parseFloatInput('abc')).toBeNull();
    expect(parseFloatInput('')).toBeNull();
  });

  it('formats doubles and CPI', () => {
    expect(formatDouble(1.5)).toBe('1.500000');
    expect(formatDouble(0)).toBe('0.000000');
    expect(formatDouble(1e20)).toBe('1.000000e+20');
    expect(formatDouble(NaN)).toBe('NaN');
    expect(formatCpi(1.23456)).toBe('1.235');
    expect(formatCpi(Infinity)).toBe('0.000');
  });

  it('base64 round trip', () => {
    const b = new Uint8Array([0, 1, 2, 250, 255]);
    expect([...base64ToBytes(bytesToBase64(b))]).toEqual([...b]);
    expect(base64ToBytes('').length).toBe(0);
  });
});
