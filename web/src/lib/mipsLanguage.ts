import { StreamLanguage, type StreamParser } from '@codemirror/language';

const REG = /^(\$?(r([0-9]|[12][0-9]|3[01])|f([0-9]|[12][0-9]|3[01])|zero|at|v[01]|a[0-3]|t[0-9]|s[0-7]|k[01]|gp|sp|fp|ra))\b/i;

interface MipsState { lineStart: boolean; sawOp: boolean }

/** Very small MIPS64 (WinMIPS64 dialect) highlighter: comments, labels, directives, opcodes, registers, numbers, strings. */
export const mipsParser: StreamParser<MipsState> = {
  name: 'mips64',
  startState: () => ({ lineStart: true, sawOp: false }),
  token(stream, state) {
    if (stream.sol()) {
      state.lineStart = true;
      state.sawOp = false;
    }
    if (stream.eatSpace()) return null;
    if (stream.peek() === ';' || stream.peek() === '#') {
      stream.skipToEnd();
      return 'comment';
    }
    if (stream.match(/^"(?:[^"\\]|\\.)*"?/)) return 'string';
    if (stream.match(/^[A-Za-z_.$][\w.$]*\s*:/)) return 'labelName';
    if (stream.match(/^\.[A-Za-z0-9]+/)) {
      state.sawOp = true;
      return 'keyword';
    }
    if (stream.match(/^-?0x[0-9a-f]+/i) || stream.match(/^-?\d+(\.\d+)?([eE][-+]?\d+)?/)) return 'number';
    if (stream.match(REG)) return 'variableName.special';
    if (stream.match(/^[A-Za-z_][\w.]*/)) {
      if (!state.sawOp) {
        state.sawOp = true;
        return 'keyword.control';
      }
      return 'variableName';
    }
    stream.next();
    return 'punctuation';
  },
  languageData: { commentTokens: { line: ';' } },
};

export const mipsLanguage = StreamLanguage.define(mipsParser);
