#!/usr/bin/env bash
# Lints every commit since lesson-00 (or since the first commit, before it exists).
#
#   scripts/check-commits.sh            messages only
#   scripts/check-commits.sh --replay   also checks out each commit and runs make check
#
# The rules are the house style:
#   subject   type(scope): imperative summary, lowercase, at most 50 characters, no period
#   body      blank line after the subject, lines at most 72 characters, says WHY
#   replay    every commit builds and passes on its own, so git bisect works
set -euo pipefail

types='feat|fix|refactor|test|docs|chore|perf|build|ci'
base=$(git rev-parse -q --verify refs/tags/lesson-00 || git rev-list --max-parents=0 HEAD)
fail=0

for c in $(git rev-list --reverse "$base..HEAD"); do
  short=$(git rev-parse --short "$c")
  subject=$(git log -1 --format=%s "$c")
  problems=""

  printf '%s' "$subject" | grep -Eq "^($types)\([a-z0-9,-]+\): [a-z0-9]" \
    || problems="$problems\n    subject is not type(scope): lowercase summary"
  [ "${#subject}" -le 50 ] || problems="$problems\n    subject is ${#subject} chars, max 50"
  case "$subject" in *.) problems="$problems\n    subject ends with a period" ;; esac

  second=$(git log -1 --format=%B "$c" | sed -n 2p)
  [ -z "$second" ] || problems="$problems\n    no blank line after the subject"

  long=$(git log -1 --format=%b "$c" | grep -Ev '^[A-Za-z-]+: ' | awk 'length > 72' | head -1 || true)
  [ -z "$long" ] || problems="$problems\n    body line over 72 chars: ${long:0:40}..."

  if [ -n "$problems" ]; then
    printf '✗ %s %s%b\n' "$short" "$subject" "$problems"
    fail=1
  else
    printf '✓ %s %s\n' "$short" "$subject"
  fi
done

if [ "${1:-}" = "--replay" ]; then
  work=$(mktemp -d)
  trap 'git worktree remove --force "$work" >/dev/null 2>&1 || true' EXIT
  git worktree add -q --detach "$work" HEAD
  for c in $(git rev-list --reverse "$base..HEAD"); do
    git -C "$work" checkout -q "$c"
    if make -C "$work" -s check >/dev/null 2>&1; then
      printf '✓ green %s\n' "$(git rev-parse --short "$c")"
    else
      printf '✗ red   %s %s\n' "$(git rev-parse --short "$c")" "$(git log -1 --format=%s "$c")"
      fail=1
    fi
  done
fi

exit "$fail"
