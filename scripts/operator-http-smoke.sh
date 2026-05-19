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
    local check="$3"   # "ok-body" oder "prometheus-text"

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
            # slice-M6 §2.2: drei Asserts gegen die /metrics-Response.
            # (a) Format-Marker: # HELP/# TYPE — generisch, robust gegen
            #     controller-runtime-library-Drift einer einzelnen Metrik.
            if ! grep -qE '^# (HELP|TYPE) ' <<<"${body}"; then
                log "FAIL ${label}: response missing '# HELP' / '# TYPE' marker"
                echo "${body}" | head -10 >&2
                return 1
            fi
            # (b) Inhalts-Beweis: mindestens eine der drei controller-
            #     runtime-Standard-Metriken muss da sein. OR-verknüpft —
            #     wenn eine Library-Version eine Metrik umbenennt, gewinnt
            #     der Test trotzdem solange ≥ 1 da ist.
            if ! grep -qE '^(workqueue_depth|rest_client_requests_total|controller_runtime_reconcile_total)( |\{)' <<<"${body}"; then
                log "FAIL ${label}: response missing any expected controller-runtime metric"
                echo "${body}" | head -20 >&2
                return 1
            fi
            # (c) Sanity-Mindestlänge: controller-runtime exponiert typisch
            #     > 80 nicht-leere Zeilen; 20 als konservative Untergrenze
            #     gegen den "HTTP 200 mit leerem Body, Manager-Init noch
            #     nicht durch"-Race.
            local nonempty_lines
            nonempty_lines="$(grep -vc '^$' <<<"${body}" || true)"
            if [[ "${nonempty_lines}" -lt 20 ]]; then
                log "FAIL ${label}: only ${nonempty_lines} non-empty lines, want >= 20"
                echo "${body}" >&2
                return 1
            fi
            log "  /metrics body ${nonempty_lines} non-empty lines (format+content+sanity)"
            ;;
    esac
    log "OK ${label}"
}

probe "/healthz" "http://localhost:${LOCAL_HEALTHZ}/healthz" ok-body
probe "/readyz"  "http://localhost:${LOCAL_HEALTHZ}/readyz"  ok-body
probe "/metrics" "http://localhost:${LOCAL_METRICS}/metrics" prometheus-text

log "PASSED — healthz, readyz, /metrics reachable + format+content+sanity OK"
