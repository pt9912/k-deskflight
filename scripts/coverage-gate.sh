#!/usr/bin/env bash
# Coverage threshold gate.
#
# Adaption von /Development/m-trace/apps/api/scripts/coverage-gate.sh
# (ADR 0012 §2.5, LH-QG-003, slice-M1-repo-skeleton.md §3).
#
# Erwartet eine Datei mit dem Output-Format von `go tool cover -func`.
# Die letzte Zeile beginnt im Normalfall mit `total:` und trägt im
# letzten Whitespace-Token den Prozentwert (Format `89.8%`).
#
# Bootstrap-Modus: nur akzeptiert, wenn das Env-Var COVERAGE_BOOTSTRAP
# explizit auf "1" gesetzt ist. Damit kann der Dockerfile-coverage-Stage
# in slice M1 (internal/ noch leer) sauber mit leerer Eingabe arbeiten,
# ohne dass ab M2 ein still maskierter Test-Failure das Gate grün
# erscheinen lässt. Sobald COVERPKG nicht leer ist, MUSS `go tool
# cover -func` eine `total:`-Zeile liefern; tut sie das nicht, bricht
# das Gate (Exit 2 = Format-/Pipeline-Fehler).
#
# Verwendung:
#   COVERAGE_BOOTSTRAP=0|1 bash scripts/coverage-gate.sh <func-file> [<threshold-percent>]
#
# Exit codes:
#   0 — Coverage >= Threshold (oder Bootstrap-Modus authorized)
#   1 — Coverage < Threshold
#   2 — Eingabe-/Format-Fehler (Datei fehlt; Bootstrap-Eingabe ohne
#       Authorisierung; total_pct nicht numerisch)
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

# Bootstrap-Modus: leere Datei oder Datei ohne `total:`-Zeile ist nur
# OK, wenn der Aufrufer COVERAGE_BOOTSTRAP=1 setzt. Sonst liegt ein
# echter Failure vor (Tests crashen, Pipefail im Build, neue Range-
# Selector-Drift), den wir nicht stillschweigend grün durchwinken.
bootstrap="${COVERAGE_BOOTSTRAP:-0}"

if [[ ! -s "$func_file" ]] || ! grep -q '^total:' "$func_file"; then
    if [[ "$bootstrap" == "1" ]]; then
        echo "coverage-gate: bootstrap-mode authorized (COVERAGE_BOOTSTRAP=1; threshold=${threshold}%)"
        exit 0
    fi
    echo "coverage-gate: input has no 'total:' line and COVERAGE_BOOTSTRAP!=1 — refusing to mask test failure" >&2
    exit 2
fi

total_line="$(grep '^total:' "$func_file" | tail -n1)"
total_pct="$(awk '{print $NF}' <<<"$total_line" | tr -d '%')"

if [[ ! "$total_pct" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
    echo "coverage-gate: total token is not numeric: '$total_pct' (line: $total_line)" >&2
    exit 2
fi

# Floating-Point-Vergleich via awk (bash-Builtin kann nur int).
if awk -v have="$total_pct" -v want="$threshold" 'BEGIN { exit (have+0 >= want+0) ? 0 : 1 }'; then
    echo "coverage-gate: ${total_pct}% >= ${threshold}% — OK"
    exit 0
fi

echo "coverage-gate: ${total_pct}% < ${threshold}% — FAIL" >&2
exit 1
