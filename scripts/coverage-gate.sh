#!/usr/bin/env bash
# Coverage threshold gate.
#
# Adaption von /Development/m-trace/apps/api/scripts/coverage-gate.sh
# (ADR 0012 §2.5, LH-QG-003, slice-M1-repo-skeleton.md §3).
#
# Erwartet eine Datei mit dem Output-Format von `go tool cover -func`.
# Die letzte Zeile beginnt im Normalfall mit `total:` und trägt im
# dritten Whitespace-Token den Prozentwert (Format `89.8%`).
#
# Bootstrap-Modus (slice M1, internal/ noch leer): wenn die Eingabe-
# Datei leer ist oder keine `total:`-Zeile enthält, exitet das Gate
# sauber mit 0. M6 hebt die Schwelle auf 90 % und macht das Gate
# PR-blockierend (Roadmap §3 M6).
#
# Verwendung:
#   bash scripts/coverage-gate.sh <go-tool-cover-func-file> [<threshold-percent>]
#
# Exit codes:
#   0 — Coverage >= Threshold (oder Bootstrap-Modus)
#   1 — Coverage < Threshold
#   2 — Eingabe-Fehler (Datei fehlt)
set -euo pipefail

usage() {
    echo "usage: $(basename "$0") <coverage-func-file> [<threshold-percent>]" >&2
    echo "  Default threshold: 0 (Slice M1; M6 hebt auf 90)" >&2
    exit 2
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
    usage
fi

func_file="$1"
threshold="${2:-0}"

if [[ ! -f "$func_file" ]]; then
    echo "coverage-gate: input file not found: $func_file" >&2
    exit 2
fi

# Bootstrap-Modus: leere Datei (kein Test-Output) oder Datei ohne
# `total:`-Zeile. Tritt in Slice M1 auf, solange internal/ keine
# Go-Pakete enthält.
if [[ ! -s "$func_file" ]] || ! grep -q '^total:' "$func_file"; then
    echo "coverage-gate: no coverage data yet — bootstrap mode (threshold=${threshold}% accepted)"
    exit 0
fi

total_line="$(grep '^total:' "$func_file" | tail -n1)"
total_pct="$(awk '{print $NF}' <<<"$total_line" | tr -d '%')"

if [[ -z "$total_pct" ]]; then
    echo "coverage-gate: could not parse total from: $total_line" >&2
    exit 2
fi

# Floating-Point-Vergleich via awk (bash-Builtin kann nur int).
if awk -v have="$total_pct" -v want="$threshold" 'BEGIN { exit (have+0 >= want+0) ? 0 : 1 }'; then
    echo "coverage-gate: ${total_pct}% >= ${threshold}% — OK"
    exit 0
fi

echo "coverage-gate: ${total_pct}% < ${threshold}% — FAIL" >&2
exit 1
