#!/usr/bin/env bash
# Verify that local markdown link targets resolve.
#
# Adaption von /Development/m-trace/scripts/verify-doc-refs.sh
# (ADR 0012 §2.10, LH-QG-008, slice-M1-repo-skeleton.md §3).
#
# Geltungsbereich für k-deskflight:
#   - alle *.md unter docs/ und spec/
#   - Top-Level: README.md, README.de.md, CONTRIBUTING.md,
#     CODE_OF_CONDUCT.md, SECURITY.md, CHANGELOG.md (sobald vorhanden)
#
# Was geprüft wird (ADR 0012 §2.10):
#   - Standard-Inline-Linkform (eckiges Klammerpaar für den Text,
#     rundes Klammerpaar für den Pfad), auch spitzklammerumschlossen.
#   - Relative Pfade werden gegen den Speicherort der enthaltenden
#     MD-Datei aufgelöst, absolute gegen Repository-Root.
#   - Image-Links (![…](…)) werden übersprungen.
#   - Externe Links mit URL-Schema (http://, https://, mailto:, …)
#     werden ignoriert.
#   - Fragment-Anker (#section) werden vor der Auflösung abgeschnitten.
#
# Usage:
#   scripts/verify-doc-refs.sh [root-dir]
#
# Exit codes:
#   0  passed
#   1  broken local link target detected
#   2  environment error
set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
root="${1:-$repo_root}"

if [[ ! -d "$root" ]]; then
    echo "ERROR: root directory not found: $root" >&2
    exit 2
fi

extract_local_markdown_links() {
    awk '
        {
            line = $0
            while (match(line, /!?\[[^]]*\]\([^)]*\)/)) {
                link = substr(line, RSTART, RLENGTH)
                line = substr(line, RSTART + RLENGTH)

                if (substr(link, 1, 1) == "!") {
                    continue
                }

                sub(/^!?\[[^]]*\]\(/, "", link)
                sub(/\)$/, "", link)

                if (link ~ /^</) {
                    sub(/^</, "", link)
                    sub(/>.*/, "", link)
                } else {
                    sub(/[[:space:]].*/, "", link)
                }

                sub(/#.*/, "", link)

                if (link == "" ||
                    link ~ /^[a-zA-Z][a-zA-Z0-9+.-]*:/) {
                    continue
                }

                print link
            }
        }
    ' "$1" | sort -u
}

broken=0

while IFS= read -r md; do
    rel="${md#"$root"/}"
    while IFS= read -r target; do
        if [[ "$target" == /* ]]; then
            resolved="$target"
        else
            resolved="$(dirname "$md")/$target"
        fi
        if [[ ! -e "$resolved" ]]; then
            echo "BROKEN: $rel -> $target"
            # Why: $(( … )) statt ((broken++)) — bash post-increment
            # liefert beim Übergang 0→1 Exit 1, was set -e abbrechen
            # würde. Die committete Form ist robust gegen späteres
            # Refactoring.
            broken=$((broken + 1))
        fi
    done < <(extract_local_markdown_links "$md")
done < <(
    {
        for docs_dir in "$root/docs" "$root/spec"; do
            if [[ -d "$docs_dir" ]]; then
                find "$docs_dir" -name '*.md' -type f
            fi
        done
        for top_level_doc in \
                "$root/README.md" \
                "$root/README.de.md" \
                "$root/CONTRIBUTING.md" \
                "$root/CODE_OF_CONDUCT.md" \
                "$root/SECURITY.md" \
                "$root/CHANGELOG.md"; do
            if [[ -f "$top_level_doc" ]]; then
                printf '%s\n' "$top_level_doc"
            fi
        done
    } | sort
)

if [[ "$broken" -gt 0 ]]; then
    echo "$broken broken documentation link(s)"
    exit 1
fi
echo "All documentation links OK."
