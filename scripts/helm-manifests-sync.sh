#!/usr/bin/env bash
#
# Slice-M8 §2.5 — structural drift detection between Helm-Chart-Rendering
# und der kanonischen Quelle in `deploy/manifests/` plus `config/crd/`.
#
# Aufruf-Konvention: über `make helm-manifests-sync` (Docker-only,
# K-2-Konvention). Der Make-Wrapper mountet das Repo nach /src und
# benutzt die `helm-tools`-Stage (helm + kubectl + yq).
#
# Algorithmus (slice-M8 step 5):
#   1. `helm template release-sync deploy/charts/k-deskflight/
#      --kube-version $HELM_KUBE_VERSION --namespace k-deskflight-system
#      -f test-values/default.yaml` rendern (= Cluster-Wide Mode,
#      Default-Pfad, der dem `deploy/manifests/`-Apply entspricht).
#   2. `kubectl kustomize deploy/manifests/` rendern (expandiert
#      `kustomization.yaml` inklusive ../../config/crd/-/rbac/).
#   3. Beide normalisieren:
#        - Helm-Meta-Labels entfernen (`helm.sh/chart`,
#          `app.kubernetes.io/version`, `app.kubernetes.io/managed-by`)
#          — Step-2-Review-Heads-up.
#        - controller-gen-Annotation entfernen
#          (`controller-gen.kubebuilder.io/version`) — Pin der Generator-
#          Version, irrelevant für Chart-Drift.
#        - `metadata.creationTimestamp` entfernen falls null.
#        - Resources sortieren nach (kind, namespace, name).
#        - YAML-Keys deterministisch sortieren.
#   4. `diff -u` vergleichen; bei Drift exit 1 mit Marker-Header und
#      gepatchter Diff-Ausgabe nach stderr.
#
# Drift-Politik (slice-M8 §2.2):
#   - `deploy/manifests/` ist die kanonische Quelle. Ein Drift bedeutet
#     in der Regel: Chart-Template muss nachgezogen werden.
#   - Ausnahme: wenn `make manifests` (controller-gen) eine CRD-
#     Änderung erzeugt, muss die Chart-Inline-Copy in templates/crd.yaml
#     synchronisiert werden. Heutige Lösung: Chart-CRD ist ein
#     manueller Sync; bei häufigeren CRD-Änderungen entsteht
#     `make chart-sync-crd` als Folge-Arbeit (siehe Closure-Notiz).

set -euo pipefail

CHART_DIR="deploy/charts/k-deskflight"
MANIFESTS_DIR="deploy/manifests"
KUBE_VERSION="${HELM_KUBE_VERSION:-v1.34.0}"

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# Normalisierungs-Pipeline für yq.
# Per-Document:
#   - Helm-Meta-Labels überall entfernen (sowohl auf top-level
#     metadata.labels als auch auf spec.template.metadata.labels via
#     `..`-Recursive-Descent).
#   - controller-gen-Versions-Annotation entfernen.
#   - creationTimestamp entfernen falls vorhanden (null oder gesetzt).
# Dann ein zweites yq mit `ea` (eval-all): alle Docs in Array sammeln,
# nach (kind, namespace, name) sortieren, als YAML-Array emittieren.
# Beide Files normalisiert ergeben vergleichbare Arrays.
normalize() {
    local input=$1
    local output=$2
    yq '
        # 1. Drop all comments (Helm-Template-Kommentare wie
        #    `# Source: …` würden sonst als Knoten-Annotationen
        #    in der Diff erscheinen).
        ... comments=""
        # 2. Helm-Meta-Labels überall entfernen.
        | (.. | select(tag == "!!map" and has("labels")) | .labels) |= (
            del(."helm.sh/chart")
          | del(."app.kubernetes.io/version")
          | del(."app.kubernetes.io/managed-by")
        )
        # 3. controller-gen-Versions-Annotation überall entfernen.
        | (.. | select(tag == "!!map" and has("annotations")) | .annotations) |= del(."controller-gen.kubebuilder.io/version")
        # 4. creationTimestamp entfernen falls vorhanden.
        | del(.metadata.creationTimestamp)
        # 5. Alle Map-Keys rekursiv sortieren — kustomize und helm
        #    emittieren unterschiedliche YAML-Key-Reihenfolgen.
        | (.. | select(tag == "!!map")) |= sort_keys(.)
    ' "$input" \
    | yq ea '
        [.]
        | map(select(. != null and (.kind // "") != ""))
        | sort_by(.kind, (.metadata.namespace // ""), .metadata.name)
    ' > "$output"
    # Step-5-Review-N-1: sanity check. Eine leere normalize-Output-
    # Datei würde den späteren `diff` zu False-Positive (oder False-
    # Negative gegen eine ebenfalls leere Seite) machen.
    [[ -s "$output" ]] || {
        echo "[helm-manifests-sync] normalize produced empty output for $input" >&2
        exit 2
    }
}
# Step-5-Review-N-2: Sort-Tiebreaker ist `metadata.name`. Cluster-
# Resources (ClusterRole/ClusterRoleBinding/CRD/Namespace) haben kein
# `metadata.namespace` und fallen via `// ""` deterministisch in eine
# gemeinsame Bucket; Tiebreaker `metadata.name` ist eindeutig pro
# Kind+Namespace im Kubernetes-Identitätsmodell.

# 1. Render the Helm chart against the default test-values overlay
#    (matches the Cluster-Wide Mode that deploy/manifests/ also targets).
echo "[helm-manifests-sync] rendering chart with test-values/default.yaml ..." >&2
helm template release-sync "$CHART_DIR" \
    --namespace k-deskflight-system \
    --kube-version "$KUBE_VERSION" \
    -f "$CHART_DIR/test-values/default.yaml" \
    > "$WORK/chart-raw.yaml"

# 2. Render the canonical kustomize basis. `--load-restrictor=
# LoadRestrictionsNone` ist nötig, weil `deploy/manifests/
# kustomization.yaml` Resources aus `../../config/crd/` und
# `../../config/rbac/` einbindet — die controller-gen-Outputs leben
# bewusst nicht unter deploy/manifests/ (AR-007). Default-Kustomize
# blockt das aus Supply-Chain-Hygiene gegen Out-of-Tree-References.
#
# Step-5-Review-M-2-Trade-off: wir akzeptieren den Bypass bewusst,
# weil (a) der Aufruf im helm-tools-Container mit nur `/src` als
# Mount läuft, (b) alle in der kustomization referenzierten Pfade
# innerhalb von $CURDIR liegen, (c) die Alternative — symbolische
# Re-Layouts unter deploy/manifests/ oder eine separate Sync-
# Kustomization — fügt Indirektion ohne Sicherheitsgewinn hinzu.
# Diese Aushebelung gilt ausschließlich für das in-repo-Drift-Gate.
echo "[helm-manifests-sync] rendering canonical kustomize basis ..." >&2
kubectl kustomize --load-restrictor=LoadRestrictionsNone "$MANIFESTS_DIR" > "$WORK/manifests-raw.yaml"

# 3. Normalize both sides.
echo "[helm-manifests-sync] normalizing both sides ..." >&2
normalize "$WORK/chart-raw.yaml"     "$WORK/chart-norm.yaml"
normalize "$WORK/manifests-raw.yaml" "$WORK/manifests-norm.yaml"

# 4. Diff (manifests = expected, chart = actual; that's the canonical
#    convention from slice-M8 §2.2).
echo "[helm-manifests-sync] diffing chart vs deploy/manifests/ ..." >&2
if diff -u "$WORK/manifests-norm.yaml" "$WORK/chart-norm.yaml"; then
    echo "[helm-manifests-sync] passed" >&2
    exit 0
fi

cat <<EOF >&2

[helm-manifests-sync] FAILED — Chart-Render weicht von deploy/manifests/ ab.

Drift-Politik (slice-M8 §2.2): deploy/manifests/ ist die kanonische
Quelle. Zieh den Chart-Template nach. Wenn der Befund eine echte
Manifest-Korrektur ist, ändere deploy/manifests/ und commit beide
Seiten zusammen.

Spezialfall CRD: wenn der Diff in templates/crd.yaml steckt und du
gerade \`make manifests\` (controller-gen) gelaufen hast, ruf
\`make chart-sync-crd\` auf — das Skript kopiert den frischen
config/crd/...yaml-Output in templates/crd.yaml und re-applyt den
Helm-Wrapper.

Spezialfall Image-Tag: wenn das Manifest auf einem alten Tag
(z. B. \`:v0.1.0\`) festhängt und der Chart appVersion bereits weiter
zeigt, ist das ein Release-Sync-Pfad (slice-M8 §2.2 Image-Tag-Pin-
Klausel) — gehört in den jeweiligen Release-Slice (M16 für v0.2.0).

Normalisierungs-Schritte (für Reproduktion):
  - Helm-Meta-Labels (helm.sh/chart, app.kubernetes.io/version,
    app.kubernetes.io/managed-by) entfernt.
  - controller-gen.kubebuilder.io/version-Annotation entfernt.
  - metadata.creationTimestamp entfernt.
  - Resources nach (kind, namespace, name) sortiert.
EOF
exit 1
