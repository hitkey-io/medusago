#!/bin/sh
# Conventional Commits check shared by the commit-msg and pre-push hooks.
# Usage: conventional.sh <subject line>
subject="$1"
case "$subject" in
  "Merge "*) exit 0 ;;
esac
if printf '%s\n' "$subject" | grep -Eq '^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9][a-z0-9._/-]*\))?!?: [^ ].{0,98}$'; then
  exit 0
fi
cat >&2 <<MSG
Commit message does not follow Conventional Commits:
  $subject
Expected: <type>(<scope>)?: <description>   (max 100 chars)
Types: feat fix docs style refactor perf test build ci chore revert
MSG
exit 1
