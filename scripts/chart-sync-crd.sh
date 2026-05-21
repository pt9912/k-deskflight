#!/usr/bin/env bash
#
# Slice-M8 step-5-Review H-2: synchronisiert die Chart-Inline-Copy
# der CRD (`deploy/charts/k-deskflight/templates/crd.yaml`) mit dem
# controller-gen-Output (`config/crd/k-deskflight.geo-terrain.net_
# opendeskpreflightchecks.yaml`).
#
# Aufruf-Konvention: `make chart-sync-crd`. Läuft auf dem Host (kein
# Tooling-Bedarf außer bash, identisch zur doc-refs-Carveout im
# Makefile-Header) — Source und Ziel sind beide im Repo.
#
# Wann verwenden:
#   Nach jedem `make manifests`, wenn das config/crd/-File durch
#   neue/geänderte `+kubebuilder:`-Marker am Reconcile-Receiver
#   regeneriert wurde. Das `helm-manifests-sync`-Gate würde sonst
#   einen CRD-Drift melden, und die Fehler-Meldung verweist explizit
#   auf dieses Skript (slice-M8 step-5-Review H-2).

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$REPO_ROOT/config/crd/k-deskflight.geo-terrain.net_opendeskpreflightchecks.yaml"
DST="$REPO_ROOT/deploy/charts/k-deskflight/templates/crd.yaml"

if [[ ! -f "$SRC" ]]; then
    echo "[chart-sync-crd] source CRD not found: $SRC" >&2
    echo "[chart-sync-crd] hint: run \`make manifests\` first." >&2
    exit 1
fi

# Wrapper-Header + CRD-Body + Wrapper-Footer. Pattern identisch zum
# Step-2-Aufbau von templates/crd.yaml (vgl. Commit 3b5cae2).
{
    echo '{{- if .Values.crd.install -}}'
    echo '# CRD inline-copy von config/crd/k-deskflight.geo-terrain.net_opendeskpreflightchecks.yaml'
    echo '# (controller-gen-Output). Sync via `make chart-sync-crd` nach jedem'
    echo '# `make manifests`-Lauf. Drift-Gate `make helm-manifests-sync` (slice-M8 §2.5)'
    echo '# prüft, dass diese Datei mit der Quelle synchron bleibt.'
    cat "$SRC"
    echo '{{- end }}'
} > "$DST"

echo "[chart-sync-crd] $DST refreshed from $SRC"
