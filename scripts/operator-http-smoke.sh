#!/usr/bin/env bash
# operator-http-smoke.sh — verifiziert die controller-runtime-HTTP-
# Endpoints (`/healthz`, `/readyz`, `/metrics`) am laufenden Operator-
# Pod.
#
# Voraussetzung: `make cluster-smoke` hat den Operator im
# kind-Cluster ${CLUSTER_NAME} ausgerollt (oder ein anderer Cluster
# wurde provisioniert; CLUSTER_KEEP=1 lässt den smoke-Cluster
# bestehen). Wird als eigene Stufe nach Step 6 in cluster-smoke.sh
# aufgerufen und ist zusätzlich standalone via `make operator-http-smoke`
# nutzbar.
#
# Implementierungs-Notiz: kubectl port-forward läuft im Hintergrund,
# wird per trap aufgeräumt. `--network host` (vom Makefile gesetzt)
# erlaubt curl gegen `localhost:<lokaler-port>`, wo port-forward die
# Pod-Ports anbietet.
#
# Env-Overrides:
#   CLUSTER_NAME      — kind-Kontext (Default: k-deskflight-smoke)
#   NAMESPACE         — Default: k-deskflight-system
#   DEPLOYMENT_LABEL  — Pod-Selector (Default: app.kubernetes.io/component=operator)
#   HEALTHZ_PORT      — Container-Port healthz/readyz (Default 8081)
#   METRICS_PORT      — Container-Port /metrics    (Default 8080)
#   LOCAL_HEALTHZ     — Host-seitiger forward-Port healthz (Default 9081)
#   LOCAL_METRICS     — Host-seitiger forward-Port metrics (Default 9080)
#
# Exit codes:
#   0 — alle drei Endpoints OK
#   1 — Endpoint-Probe fehlgeschlagen
#   2 — Setup-Fehler (kein Pod gefunden, port-forward kommt nicht hoch)
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-k-deskflight-smoke}"
NAMESPACE="${NAMESPACE:-k-deskflight-system}"
DEPLOYMENT_LABEL="${DEPLOYMENT_LABEL:-app.kubernetes.io/component=operator}"
HEALTHZ_PORT="${HEALTHZ_PORT:-8081}"
METRICS_PORT="${METRICS_PORT:-8080}"
LOCAL_HEALTHZ="${LOCAL_HEALTHZ:-9081}"
LOCAL_METRICS="${LOCAL_METRICS:-9080}"

KIND_CONTEXT="kind-${CLUSTER_NAME}"

log() { echo "[http-smoke] $*"; }

POD="$(kubectl --context "${KIND_CONTEXT}" -n "${NAMESPACE}" \
    get pods -l "${DEPLOYMENT_LABEL}" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
if [[ -z "${POD}" ]]; then
    log "FAIL: no pod found via label ${DEPLOYMENT_LABEL} in ${NAMESPACE}"
    exit 2
fi
log "target pod: ${POD}"

# Port-Forward im Hintergrund.
kubectl --context "${KIND_CONTEXT}" -n "${NAMESPACE}" \
    port-forward "pod/${POD}" \
    "${LOCAL_HEALTHZ}:${HEALTHZ_PORT}" \
    "${LOCAL_METRICS}:${METRICS_PORT}" \
    >/dev/null 2>&1 &
PF_PID=$!
cleanup() { kill "${PF_PID}" 2>/dev/null || true; }
trap cleanup EXIT

# Auf port-forward-Ready warten (kurzer Poll-Loop).
for _ in $(seq 1 15); do
    if curl -fsS -o /dev/null --max-time 1 "http://localhost:${LOCAL_HEALTHZ}/healthz" 2>/dev/null; then
        break
    fi
    sleep 1
done

probe() {
    local label="$1"
    local url="$2"
    local check="$3"   # entweder "ok-body" oder "any-200"

    log "GET ${url}"
    local body
    if ! body="$(curl -fsS --max-time 5 "${url}" 2>&1)"; then
        log "FAIL ${label}: curl returned non-2xx"
        return 1
    fi

    case "${check}" in
        ok-body)
            if [[ "${body}" != "ok" ]]; then
                log "FAIL ${label}: expected body 'ok', got '${body}'"
                return 1
            fi
            ;;
        prometheus-text)
            if ! grep -q '^# HELP' <<<"${body}"; then
                log "FAIL ${label}: response does not look like Prometheus exposition format"
                echo "${body}" | head -5 >&2
                return 1
            fi
            ;;
    esac
    log "OK ${label}"
}

probe "/healthz" "http://localhost:${LOCAL_HEALTHZ}/healthz" ok-body
probe "/readyz"  "http://localhost:${LOCAL_HEALTHZ}/readyz"  ok-body
probe "/metrics" "http://localhost:${LOCAL_METRICS}/metrics" prometheus-text

log "PASSED — healthz, readyz, /metrics reachable + valid"
