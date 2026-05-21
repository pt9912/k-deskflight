#!/usr/bin/env bash
#
# Slice-M8 §2.5 + §4 step 5 — structural drift detection between
# Helm-Chart-Rendering und der kanonischen Quelle in
# `deploy/manifests/` plus `config/crd/`.
#
# Aufruf-Konvention: über `make helm-manifests-sync` (Docker-only,
# K-2-Konvention). Direktaufruf vom Host setzt voraus dass `helm` und
# `kubectl` im PATH liegen — der Make-Wrapper benutzt die
# `helm-tools`-Docker-Stage, die helm bereitstellt; kubectl wird in
# Step 5 nachgezogen (entweder mit-image-built oder die `smoke`-Stage
# wiederverwendet).
#
# Pattern (in Step 5 umzusetzen):
#   1. `helm template release-test deploy/charts/k-deskflight/
#      --kube-version $HELM_KUBE_VERSION --namespace k-deskflight-system
#      -f test-values/default.yaml` rendern.
#   2. `kubectl kustomize deploy/manifests/` rendern (expandiert
#      `kustomization.yaml`-Resources inklusive ../../config/crd/-/rbac/).
#   3. Beide normalisieren (siehe Block "Normalisierung" unten).
#   4. `diff -u` vergleichen; bei Drift exit 1 mit klarem Hinweis.
#
# Normalisierung (Step-2-Review-Heads-up und allgemeine Helm-Boilerplate):
#   - Aus Chart-Output entfernen: Labels `helm.sh/chart`,
#     `app.kubernetes.io/version`, `app.kubernetes.io/managed-by`
#     (Helm-Meta, niemals in deploy/manifests/).
#   - Aus beiden entfernen: `metadata.annotations.kubectl.kubernetes.io/last-applied-configuration`
#     falls vorhanden (Cluster-Annotation, nicht Source).
#   - YAML-Stream beidseitig kanonisieren (z. B. `yq -P sort_keys(..)` oder
#     `kubectl apply --dry-run=client -o yaml --filename -`), damit
#     YAML-Key-Reihenfolge nicht zu False-Positives führt.
#   - Resource-Sortierung beidseitig nach (Kind, Namespace, Name).
#
# Drift-Politik:
#   - deploy/manifests/ ist die kanonische Quelle (slice-M8 §2.2);
#     ein Drift bedeutet meist: Chart-Template muss nachgezogen werden.
#   - Ausnahme: wenn `make manifests` (controller-gen) eine CRD-Änderung
#     erzeugt, muss die Chart-Inline-Copy in templates/crd.yaml
#     synchronisiert werden — ein eigenes `make chart-sync-crd` ist
#     dafür sinnvoll (offene Entscheidung für Step 5).
#
# Step 4 stub: das Gate steht strukturell zur Verfügung, ist aber
# bewusst noch nicht implementiert. Step 5 zieht die obigen Schritte
# konkret nach und entscheidet die kubectl-Tool-Stage-Frage.

set -euo pipefail

cat <<EOF >&2
[helm-manifests-sync] STUB — not implemented yet (slice-M8 step 5).
[helm-manifests-sync] This target exists structurally so that step 4
[helm-manifests-sync] can wire it into the Makefile and slice plan.
[helm-manifests-sync] Step 5 will:
[helm-manifests-sync]   1. render helm-chart + kubectl-kustomize
[helm-manifests-sync]   2. normalise both (drop helm-meta labels)
[helm-manifests-sync]   3. diff and report drift
[helm-manifests-sync] See scripts/helm-manifests-sync.sh header for
[helm-manifests-sync] the planned algorithm.
EOF

exit 1
