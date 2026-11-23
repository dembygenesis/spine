#!/usr/bin/env bash
# Runs one lesson's tests against your working tree.
#
# The tests come from the lesson's tag, not from your files, so editing them
# cannot make a lesson pass: they are copied over a scratch copy of your work.
#
#   scripts/grade.sh 03
set -euo pipefail

lesson="${1:?usage: scripts/grade.sh NN   (e.g. 03)}"
lesson=$(printf '%02d' "$((10#$lesson))")
tag="lesson-$lesson"
prev="lesson-$(printf '%02d' "$((10#$lesson - 1))")"

git rev-parse -q --verify "refs/tags/$tag" >/dev/null || { echo "no tag $tag: run git fetch --tags"; exit 2; }
if [ "$lesson" = "00" ]; then
  echo "lesson 00 has no code to grade"
  exit 0
fi

files=$(git diff --name-only "$prev" "$tag" -- '*_test.go' '*/testdata/*')
if [ -z "$files" ]; then
  echo "$tag changes no tests"
  exit 0
fi

work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
git ls-files -co --exclude-standard -z | tar --null -T - -cf - | tar -xf - -C "$work"

pkgs=""
for f in $files; do
  mkdir -p "$work/$(dirname "$f")"
  git show "$tag:$f" > "$work/$f"
  pkgs="$pkgs ./$(dirname "${f%%/testdata/*}")"
done

echo "grading $tag:" $files
cd "$work"
log=$(mktemp)
# shellcheck disable=SC2086 # word splitting is the point: one arg per package
if go test -count=1 -v $(printf '%s\n' $pkgs | sort -u) >"$log" 2>&1; then rc=0; else rc=1; fi
grep -E '^(--- |ok|FAIL|panic)|_test.go:[0-9]+' "$log" || true
# go test exits 0 when it finds nothing to run; a grader that passes on zero
# tests proves nothing.
if ! grep -q -- '--- PASS' "$log" && [ "$rc" = 0 ]; then
  echo "FAIL $tag: zero tests ran"
  exit 1
fi
rm -f "$log"
if [ "$rc" = 0 ]; then
  echo "PASS $tag"
else
  echo "FAIL $tag"
  exit 1
fi
