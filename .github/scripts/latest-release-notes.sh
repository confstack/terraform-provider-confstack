#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
changelog_file="${1:-$repo_root/CHANGELOG.md}"

if [[ ! -f "$changelog_file" ]]; then
  echo "missing changelog: $changelog_file" >&2
  exit 1
fi

notes="$(awk '
  /^## / {
    if (in_section) exit
    in_section = 1
    next
  }

  in_section { print }
' "$changelog_file")"

if [[ -z "$notes" ]]; then
  echo "failed to extract latest release notes from $changelog_file" >&2
  exit 1
fi

printf '%s\n' "$notes"
