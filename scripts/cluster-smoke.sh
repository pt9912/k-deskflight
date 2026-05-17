#!/usr/bin/env bash
# End-to-end cluster smoke against a kind cluster.
#
# Bezug: ADR 0013 (kind als Default-Plattform für Cluster-Smoke).
# Läuft INSIDE der `smoke`-Dockerfile-Stage (Makefile `cluster-smoke`-
# Target) mit Docker-Socket-Mount (DooD) und `--network host`.
#
# Verifiziert in Sequenz:
#   - LH-AK-001 — CRD installierbar (`kubectl apply -f config/crd/…`).
#   - LH-AK-002 — Operator startbar (`kubectl wait Available`).
#   - LH-AK-005 — KubernetesVersion-Check gegen reale Server-Version
#     (Status.Phase=Passed, Condition KubernetesVersionReady=True).
#
# Env-Overrides:
#   CLUSTER_NAME=name           — kind-Cluster-Name (Default: k-deskflight-smoke)
#   K8S_VERSION=v1.34.0         — kindest/node-Tag (Default synchron mit ADR 0009 §2.2)
#   IMAGE_TAG=k-deskflight:go   — Operator-Image, das in kind geladen wird
#   CLUSTER_KEEP=0|1            — bei 1 wird der Cluster nach Erfolg behalten
#   CLUSTER_SMOKE_ATTEST_FILE   — optional: Pfad für YAML-Status-Dump als Attest-Artefakt
#
# Exit codes:
#   0 — alle Assertions passed
#   1 — Status-Assertion failed (Phase, Condition, oder Timeout)
#   2 — Setup-Fehler (Cluster-Build, CRD-Install, Deployment-Wait)
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-k-deskflight-smoke}"
K8S_VERSION="${K8S_VERSION:-v1.34.0}"
IMAGE_TAG="${IMAGE_TAG:-k-deskflight:go}"
CLUSTER_KEEP="${CLUSTER_KEEP:-0}"

KIND_CONTEXT="kind-${CLUSTER_NAME}"
SAMPLE_NAMESPACE="default"
SAMPLE_NAME="smoke"
WAIT_TIMEOUT_PHASE=60
DEPLOYMENT_WAIT_TIMEOUT=120s

log() { echo "[cluster-smoke] $*"; }

cleanup() {
    rm -rf /src/.cluster-smoke-overlay 2>/dev/null || true
    if [[ "${CLUSTER_KEEP}" != "1" ]]; then
        log "delete kind cluster ${CLUSTER_NAME}"
        kind delete cluster --name "${CLUSTER_NAME}" >/dev/null 2>&1 || true
    else
        log "CLUSTER_KEEP=1 — cluster bleibt erhalten (kubectl --context ${KIND_CONTEXT})"
    fi
}
trap cleanup EXIT

# Step 1 — kind cluster (idempotent).
log "Step 1: kind create cluster ${CLUSTER_NAME} (k8s ${K8S_VERSION})"
if kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
    log "  cluster ${CLUSTER_NAME} existiert bereits — wiederverwenden"
else
    kind create cluster \
        --name "${CLUSTER_NAME}" \
        --image "kindest/node:${K8S_VERSION}" \
        --wait 60s
fi

# Step 2 — operator-Image in kind laden (lokal gebautes Image, kein Registry-Pull).
log "Step 2: load ${IMAGE_TAG} into ${CLUSTER_NAME}"
kind load docker-image --name "${CLUSTER_NAME}" "${IMAGE_TAG}"

# Step 3 — CRD installieren (LH-AK-001).
log "Step 3: apply CRD"
kubectl --context "${KIND_CONTEXT}" apply -f config/crd/
kubectl --context "${KIND_CONTEXT}" wait --for=condition=Established \
    crd/opendeskpreflightchecks.k-deskflight.geo-terrain.net --timeout=30s

# Step 4 — Operator-Manifeste mit Image-Substitution.
log "Step 4: apply operator manifests (image-override → ${IMAGE_TAG})"
# kustomize akzeptiert keine absoluten Pfade im `resources:`-Feld
# (`new root … cannot be absolute`). Wir legen das Overlay neben das
# Workspace ab und referenzieren relativ.
TMP_OVERLAY="/src/.cluster-smoke-overlay"
rm -rf "${TMP_OVERLAY}"
mkdir -p "${TMP_OVERLAY}"
cat > "${TMP_OVERLAY}/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../deploy/manifests
images:
  - name: ghcr.io/pt9912/k-deskflight
    newName: ${IMAGE_TAG%:*}
    newTag: ${IMAGE_TAG##*:}
EOF
# deploy/manifests/kustomization.yaml referenziert ../../config/crd/
# und ../../config/rbac/ (Generated-Drift-Konsistenz). Das verstößt
# gegen kustomize's Default-load-restrictor; wir nutzen
# `kubectl kustomize --load-restrictor=LoadRestrictionsNone` und
# pipen das Ergebnis in `kubectl apply -f -`. (Pure `kubectl apply -k`
# kennt das Flag nicht.)
kubectl --context "${KIND_CONTEXT}" kustomize "${TMP_OVERLAY}" \
    --load-restrictor=LoadRestrictionsNone \
    | kubectl --context "${KIND_CONTEXT}" apply -f -
rm -rf "${TMP_OVERLAY}"

# Step 5 — Deployment ready (LH-AK-002).
log "Step 5: wait for Deployment Available (timeout ${DEPLOYMENT_WAIT_TIMEOUT})"
kubectl --context "${KIND_CONTEXT}" -n k-deskflight-system wait \
    --for=condition=Available deployment/k-deskflight-operator \
    --timeout="${DEPLOYMENT_WAIT_TIMEOUT}"

# Step 6 — Sample-CR.
log "Step 6: apply sample CR"
kubectl --context "${KIND_CONTEXT}" apply -f config/samples/

# Step 7 — auf finale Phase warten.
log "Step 7: wait for status.phase != Pending/Running (max ${WAIT_TIMEOUT_PHASE}s)"
phase=""
for _ in $(seq 1 "${WAIT_TIMEOUT_PHASE}"); do
    phase="$(kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
        get opendeskpreflightcheck "${SAMPLE_NAME}" \
        -o jsonpath='{.status.phase}' 2>/dev/null || true)"
    if [[ -n "${phase}" && "${phase}" != "Pending" && "${phase}" != "Running" ]]; then
        break
    fi
    sleep 1
done

# Step 8 — Assertions (LH-AK-005).
log "Step 8: assert final state"
log "  status.phase = ${phase:-<empty>}"

if [[ "${phase}" != "Passed" ]]; then
    log "FAIL: expected Passed, got '${phase}'"
    kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
        get opendeskpreflightcheck "${SAMPLE_NAME}" -o yaml >&2 || true
    exit 1
fi

condition_type="$(kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
    get opendeskpreflightcheck "${SAMPLE_NAME}" \
    -o jsonpath='{.status.conditions[0].type}' 2>/dev/null || true)"
condition_status="$(kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
    get opendeskpreflightcheck "${SAMPLE_NAME}" \
    -o jsonpath='{.status.conditions[0].status}' 2>/dev/null || true)"

if [[ "${condition_type}" != "KubernetesVersionReady" ]]; then
    log "FAIL: expected KubernetesVersionReady condition, got '${condition_type}'"
    kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
        get opendeskpreflightcheck "${SAMPLE_NAME}" -o yaml >&2 || true
    exit 1
fi
if [[ "${condition_status}" != "True" ]]; then
    log "FAIL: expected condition status True, got '${condition_status}'"
    kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
        get opendeskpreflightcheck "${SAMPLE_NAME}" -o yaml >&2 || true
    exit 1
fi

log "PASSED — phase=${phase}, condition=${condition_type}=${condition_status}"

# Attest-Artefakt: Status-Dump unter /src/.cluster-smoke-attest.yaml
# (Workspace-Mount); CI-Workflow `cluster-smoke.yml` lädt das als
# Run-Artefakt hoch, lokal gitignored. Override via
# CLUSTER_SMOKE_ATTEST_FILE=<pfad>.
ATTEST_FILE="${CLUSTER_SMOKE_ATTEST_FILE:-/src/.cluster-smoke-attest.yaml}"
kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
    get opendeskpreflightcheck "${SAMPLE_NAME}" -o yaml \
    > "${ATTEST_FILE}"
log "attest written to ${ATTEST_FILE}"
