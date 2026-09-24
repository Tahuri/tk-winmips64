#!/usr/bin/env bash
# Copyright 2026 tk-winmips64 contributors
# SPDX-License-Identifier: Apache-2.0
#
# Regenerates the golden traces and checks that their *decompressed* content
# matches the committed ones. gzip output bytes differ between gzip builds
# (macOS vs Linux), so a byte-level `git diff` of the .gz files is not valid.
set -euo pipefail
cd "$(dirname "$0")/../.."

tools/oracle/gen-golden.sh >/dev/null
bad=0
for f in testdata/golden/*.trace.gz; do
  if ! git cat-file -e "HEAD:$f" 2>/dev/null; then
    echo "new trace not committed: $f"; bad=1; continue
  fi
  if ! cmp -s <(git show "HEAD:$f" | gunzip -c) <(gunzip -c "$f"); then
    echo "trace content differs: $f"; bad=1
  fi
done
git checkout -- testdata/golden   # restore committed bytes
exit $bad
