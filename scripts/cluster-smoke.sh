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
#   - LH-AK-005 — KubernetesVersion-Check gegen reale Server-Version.
#   - LH-AK-006 — StorageClass-Check gegen kind-`standard` (default).
#   - LH-AK-007 — IngressClass-Check gegen den nginx-Stub aus
#     hack/cluster-smoke/cluster-state-stubs.yaml.
#   - LH-AK-008 — cert-manager-Check gegen die Stub-CRD-Registrierung.
#   - LH-AK-009 — ClusterResources-Check gegen kind-Worker-Allocatable
#     mit MinCPU=1/MinMemory=1Gi.
#   - LH-NF-016 — Helm-Installierbarkeit (nur INSTALL_MODE=helm,
#     slice-M8). Manifests-Pfad attestiert sie nicht.
#
# Cluster-State-Stubs (cert-manager-API-Gruppe und nginx-IngressClass)
# werden vor dem Operator-Deploy appliziert, sind aber kein Upstream-
# Mirror — handgeschrieben unter hack/cluster-smoke/ (slice-M4 §2.6).
#
# Env-Overrides:
#   CLUSTER_NAME=name           — kind-Cluster-Name (Default: k-deskflight-smoke)
#   K8S_VERSION=v1.34.0         — kindest/node-Tag (Default synchron mit ADR 0009 §2.2)
#   IMAGE_TAG=k-deskflight:go   — Operator-Image, das in kind geladen wird
#   CLUSTER_KEEP=0|1            — bei 1 wird der Cluster nach Erfolg behalten
#   INSTALL_MODE=manifests|helm — Default `manifests` (kubectl apply via kustomize-overlay
#                                 wie seit M2). `helm` installiert den Operator über das
#                                 Chart unter deploy/charts/k-deskflight/ (slice-M8 §2.7).
#                                 Beide Modi rendern dasselbe Funktionsset; CI fährt beide
#                                 parallel als Matrix-Job.
#   CLUSTER_SMOKE_ATTEST_FILE   — optional: Pfad für YAML-Status-Dump als Attest-Artefakt
#   PROBE_NAMESPACE=ns          — **Symbol-Konstante**, kein echter Override (Default: default).
#                                 Ein Override erfordert paralleles Patchen der drei
#                                 `namespace:`-Felder im Probe-Yaml — wird sonst zu
#                                 120s-Wait-Timeout (siehe Inline-Comment + Probe-Yaml-Header).
#   METRICS_SERVICE_FQDN=…      — Service-DNS-FQDN für Step-9b /metrics-Scrape (Default:
#                                 k-deskflight-operator-metrics.k-deskflight-system.svc.cluster.local).
#                                 Override sinnvoll z. B. bei custom-Cluster-Domain.
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
# Slice-M8 §2.7 + step 6: Install-Modus.
#   manifests (Default) — Operator via `kubectl kustomize deploy/manifests/`
#                          (Cluster-Wide Mode), wie seit M2.
#   helm                 — Operator via `helm install` aus dem Chart unter
#                          deploy/charts/k-deskflight/; CRD+RBAC+Deployment+
#                          Service+metrics-ClusterRole als single chart-release.
INSTALL_MODE="${INSTALL_MODE:-manifests}"
if [[ "${INSTALL_MODE}" != "manifests" && "${INSTALL_MODE}" != "helm" ]]; then
    echo "[cluster-smoke] FAIL: INSTALL_MODE must be 'manifests' or 'helm', got '${INSTALL_MODE}'" >&2
    exit 2
fi

KIND_CONTEXT="kind-${CLUSTER_NAME}"
# Step-6-Review-M-1: zentrale Skript-Variable für den Operator-
# Namespace, sodass Step-3+4 (helm/manifests install) und Step 5
# (Deployment-Available-Wait) dieselbe Quelle nutzen. Default folgt
# dem Chart-`values.yaml`-Default und dem Manifest unter
# deploy/manifests/namespace.yaml.
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-k-deskflight-system}"
SAMPLE_NAMESPACE="default"
SAMPLE_NAME="smoke"
WAIT_TIMEOUT_PHASE=60
DEPLOYMENT_WAIT_TIMEOUT=120s
# Befund 9+10 (Step-2-Review) + Round-2-Befund 2:
#   - METRICS_SERVICE_FQDN: echter Env-Override für custom-Cluster-Domains,
#     Default Plan-konform mit `.svc.cluster.local` (statt .svc-Kurzform).
#   - PROBE_NAMESPACE: **Symbol-Konstante**, kein echter Override. Die drei
#     `namespace:`-Felder im Probe-Yaml (SA, Pod, CRB-subject) sind
#     hardcoded auf `default`; ein Override auf einen anderen Namespace
#     führt zu „pods 'metrics-scrape-probe' not found" im Wait-Aufruf
#     unten und läuft 120s lang ins Leere. Für echte Namespace-Wahl
#     braucht es sed-Templating oder kustomize-Overlay des Yamls —
#     v0.2-Scope, wenn überhaupt nötig. Aktuell ist `default` die einzige
#     getestete Konfiguration.
PROBE_NAMESPACE="${PROBE_NAMESPACE:-default}"
METRICS_SERVICE_FQDN="${METRICS_SERVICE_FQDN:-k-deskflight-operator-metrics.k-deskflight-system.svc.cluster.local}"

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

log "INSTALL_MODE=${INSTALL_MODE}"

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

# Step 2b — Cluster-State-Stubs (slice-M4 §2.6): Minimal-Resources, die
# die Existence-Checks für cert-manager (API-Gruppe via Stub-CRD) und
# IngressClass `nginx` erfüllen. Keine Controller-Pods, kein `wait`.
log "Step 2b: apply cluster-state stubs (cert-manager.io CRD + IngressClass nginx)"
kubectl --context "${KIND_CONTEXT}" apply -f hack/cluster-smoke/cluster-state-stubs.yaml

if [[ "${INSTALL_MODE}" == "helm" ]]; then
    # Step 3+4 (helm) — CRD und Operator in einem Helm-Release.
    # `--wait` blockiert bis das Deployment Available ist; das macht
    # Step 5 strukturell redundant, wir behalten ihn aber als
    # expliziten Sanity-Check (kostet wenige Sekunden, hält Symmetrie
    # zum manifests-Pfad).
    log "Step 3+4 (helm): helm install k-deskflight (image-override → ${IMAGE_TAG})"
    # Step-6-Review-N-2 Anmerkung: die `--create-namespace` +
    # `--set namespace.create=false`-Kombination ist ein
    # **Smoke-spezifischer Workaround**, keine Operations-Empfehlung.
    # Anwender, die das Chart produktiv installieren, wählen genau
    # eine der zwei sauberen Optionen:
    #   (a) Chart-Default beibehalten (`namespace.create=true`) und
    #       das `--create-namespace`-Flag weglassen — der Chart
    #       verwaltet den Namespace.
    #   (b) `--set namespace.create=false` UND `--create-namespace`
    #       benutzen — Helm verwaltet den Namespace, Chart nicht.
    # Smoke nimmt (b) hier nur, weil wir den Chart-Default
    # `namespace.create=true` unverändert ausrollen wollen (um Step 5
    # idempotent zu halten gegen wiederholte Smoke-Läufe in CI), aber
    # `helm install --namespace ohne-Existenz` ohne `--create-namespace`
    # mit Chart-Namespace-Resource trotzdem failt ("namespaces … not
    # found"). Step-6-Review-N-2 + Step-8-Doku werden das Pattern
    # in der Anwender-Doku explizit ausgrenzen.
    #
    # Step-6-Review-N-1: `--atomic` rollt einen Failed-Install
    # automatisch zurück; Cleanup-Trap löscht ohnehin den Cluster,
    # aber `--atomic` sorgt für klareres Helm-Side-Logging im
    # Fehlerfall (CI-Run-Output wird lesbarer).
    helm --kube-context "${KIND_CONTEXT}" install k-deskflight \
        deploy/charts/k-deskflight/ \
        --namespace "${OPERATOR_NAMESPACE}" \
        --create-namespace \
        --set namespace.create=false \
        --set image.repository="${IMAGE_TAG%:*}" \
        --set image.tag="${IMAGE_TAG##*:}" \
        --atomic --wait --timeout=120s
    # CRD-Establishment-Wait als Belt-and-Suspenders: helm --wait
    # blockt nur auf Deployment-Ready (kind_sorter sortiert die CRD
    # zwar vor das Deployment, aber --wait observiert die CRD-
    # `Established`-Condition nicht). Wenn die CRD wider Erwarten
    # nicht Established wäre, würde der Operator-Pod im
    # Reconcile-Loop straucheln und helm --wait timeouten — dieser
    # explizite Wait gibt eine 30s-Window für die saubere Failure-
    # Detektion (Step-6-Review M-2).
    kubectl --context "${KIND_CONTEXT}" wait --for=condition=Established \
        crd/opendeskpreflightchecks.k-deskflight.geo-terrain.net --timeout=30s
else
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
fi

# Step 5 — Deployment ready (LH-AK-002). Im Helm-Pfad strukturell
# durch `helm install --wait` redundant; bewusst beibehalten für
# Symmetrie und als zusätzlicher Sanity-Check.
log "Step 5: wait for Deployment Available (timeout ${DEPLOYMENT_WAIT_TIMEOUT})"
kubectl --context "${KIND_CONTEXT}" -n "${OPERATOR_NAMESPACE}" wait \
    --for=condition=Available deployment/k-deskflight-operator \
    --timeout="${DEPLOYMENT_WAIT_TIMEOUT}"

# Step 6 — Sample-CR (passed-Pfad).
log "Step 6: apply sample CR"
kubectl --context "${KIND_CONTEXT}" apply -f config/samples/

# Step 6b — Failed-CRs (slice-M6 §2.8): vier weitere OPDC-Ressourcen
# konfigurieren je einen Check auf failed (KubernetesVersion,
# StorageClass, IngressClass, ClusterResources). Eindeutige
# `metadata.name`-Werte verhindern Status-Mischung im
# controller-runtime-Watch; serielles Apply + Wait pro CR garantiert,
# dass Status-Konvergenz pro Reconcile-Zyklus abgeschlossen ist, bevor
# die nächste CR appliziert wird.
#
# **cert-manager-Ausnahme** (Plan §1 + §2.8): kein failed-CR im
# automatischen Smoke, weil cluster-state-stubs.yaml den cert-manager-
# Stub cluster-global setzt; ein per-CR-Ausschalten ist ohne
# M4-Smoke-Refactor nicht möglich. LH-AK-008 bleibt durch M4 attestiert.
log "Step 6b: apply failed-CRs (seriell + wait jsonpath=phase=Failed je 60s)"
failed_crs=(
    smoke-failed-version
    smoke-failed-storage
    smoke-failed-ingress
    smoke-failed-resources
)
for cr in "${failed_crs[@]}"; do
    log "  apply ${cr}"
    kubectl --context "${KIND_CONTEXT}" apply \
        -f "hack/cluster-smoke/failed-crs/${cr#smoke-failed-}.yaml"
    if ! kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
        wait "opendeskpreflightcheck/${cr}" \
        --for=jsonpath='{.status.phase}'=Failed --timeout=60s; then
        log "FAIL Step 6b: ${cr} did not reach phase=Failed within 60s"
        kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
            get "opendeskpreflightcheck/${cr}" -o yaml >&2 || true
        exit 1
    fi
done

# Step 7 — auf finale Phase warten (passed-CR `smoke`).
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

# Iteriert die fünf erwarteten MVP-Conditions (LH-AK-005..009) und
# verifiziert jede einzeln auf status=True. Die Aggregator-Sortierung
# (AR-014) liefert sie alphabetisch — wir nutzen den Type-Filter im
# jsonpath, damit die Assertion ordnungs-unabhängig ist.
expected_conditions=(
    CertManagerInstalled
    ClusterResourcesReady
    IngressClassReady
    KubernetesVersionReady
    StorageClassReady
)
for ct in "${expected_conditions[@]}"; do
    cs="$(kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
        get opendeskpreflightcheck "${SAMPLE_NAME}" \
        -o jsonpath="{.status.conditions[?(@.type=='${ct}')].status}" 2>/dev/null || true)"
    if [[ "${cs}" != "True" ]]; then
        log "FAIL: expected condition ${ct}=True, got '${cs:-<missing>}'"
        kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
            get opendeskpreflightcheck "${SAMPLE_NAME}" -o yaml >&2 || true
        exit 1
    fi
    log "  ${ct}=True"
done

# Step 8 Phase 2 — Failed-CR-Assertions (slice-M6 §2.8). Pro CR:
#   - phase=Failed (bereits via Wait in Step 6b verifiziert)
#   - die EINE erwartete Failed-Condition mit konkretem Reason
#   - die ANDEREN vier Conditions auf status=True (Co-Aktivierung;
#     verhindert, dass eine versehentliche Mehrfach-Failed-Konfiguration
#     unbemerkt durchrutscht).
log "Step 8 Phase 2: failed-CR conditions + reasons"
# Mapping CR-Name → (Failed-Condition, erwarteter Reason).
declare -A failed_cr_expectation=(
    [smoke-failed-version]="KubernetesVersionReady:KubernetesVersionTooOld"
    [smoke-failed-storage]="StorageClassReady:StorageClassMissing"
    [smoke-failed-ingress]="IngressClassReady:IngressClassMissing"
    [smoke-failed-resources]="ClusterResourcesReady:InsufficientCPU"
)
for cr in "${failed_crs[@]}"; do
    expected="${failed_cr_expectation[${cr}]}"
    failed_type="${expected%%:*}"
    failed_reason="${expected##*:}"

    # (a) Die erwartete Failed-Condition: status=False + matching Reason.
    cs="$(kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
        get "opendeskpreflightcheck/${cr}" \
        -o jsonpath="{.status.conditions[?(@.type=='${failed_type}')].status}" 2>/dev/null || true)"
    cr_reason="$(kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
        get "opendeskpreflightcheck/${cr}" \
        -o jsonpath="{.status.conditions[?(@.type=='${failed_type}')].reason}" 2>/dev/null || true)"
    if [[ "${cs}" != "False" || "${cr_reason}" != "${failed_reason}" ]]; then
        log "FAIL: ${cr}: expected ${failed_type}=False/${failed_reason}, got ${cs:-<missing>}/${cr_reason:-<missing>}"
        kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
            get "opendeskpreflightcheck/${cr}" -o yaml >&2 || true
        exit 1
    fi
    log "  ${cr}: ${failed_type}=False, reason=${failed_reason} ✓"

    # (b) Die anderen vier Conditions müssen True bleiben — sichert,
    # dass nur der konfigurierte Check fehlschlägt.
    for ct in "${expected_conditions[@]}"; do
        if [[ "${ct}" == "${failed_type}" ]]; then
            continue
        fi
        other_cs="$(kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
            get "opendeskpreflightcheck/${cr}" \
            -o jsonpath="{.status.conditions[?(@.type=='${ct}')].status}" 2>/dev/null || true)"
        if [[ "${other_cs}" != "True" ]]; then
            log "FAIL: ${cr}: collateral condition ${ct} expected True, got '${other_cs:-<missing>}'"
            kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
                get "opendeskpreflightcheck/${cr}" -o yaml >&2 || true
            exit 1
        fi
    done
    log "  ${cr}: other 4 conditions=True (no collateral failure)"
done

# Backward-Compat-Variablen für die abschließende Status-Zeile.
condition_type="all 5 MVP conditions"
condition_status="True"

# Step 9 — HTTP-Smoke gegen Operator-Endpoints (healthz/readyz/metrics).
# Identische Toolchain (smoke-Stage); aufgerufen als embedded Sub-Step,
# damit ein einziger `make cluster-smoke`-Run alle Slice-§7-Items
# attestiert (CR-Phase + HTTP-Endpoints).
log "Step 9: HTTP-Smoke gegen Operator-Endpoints"
bash scripts/operator-http-smoke.sh

# Step 9b — E2E-Verifikation für das neue /metrics-Service-Objekt
# (slice-M6 §2.2.2). Der M3-Stand in operator-http-smoke.sh prüft
# /metrics nur via Port-Forward direkt zum Pod — das Service-Routing
# bleibt damit ungetestet. Hier scraped ein Probe-Pod via Service-DNS-
# FQDN, und drei Asserts attestieren Format + Inhalt + Sanity.
log "Step 9b: /metrics via Service-DNS aus Probe-Pod (FQDN-Routing)"
kubectl --context "${KIND_CONTEXT}" apply -f hack/cluster-smoke/metrics-scrape-probe.yaml
# Befund 3 (Step-2-Review): Timeout 120s (analog DEPLOYMENT_WAIT_TIMEOUT)
# + describe-Fallback bei Wait-Timeout, damit der nächste Maintainer
# nicht raten muss, ob Pod-Pull oder Scheduling klemmt.
if ! kubectl --context "${KIND_CONTEXT}" -n "${PROBE_NAMESPACE}" \
    wait pod/metrics-scrape-probe --for=condition=Ready --timeout=120s; then
    log "FAIL Step 9b: probe pod did not become Ready within 120s"
    kubectl --context "${KIND_CONTEXT}" -n "${PROBE_NAMESPACE}" describe pod metrics-scrape-probe >&2 || true
    # Round-2-Befund 3: Events auf den Probe-Pod gefiltert, damit die
    # relevanten Pull-/Schedule-Fails nicht in kind-Bootstrap-Events
    # vergraben sind. Schmaler Field-Selector — wenn Scheduler-Events
    # ohne involvedObject.name kommen (selten), ist der Pod-Describe-
    # Aufruf darüber bereits die Backup-Diagnose.
    kubectl --context "${KIND_CONTEXT}" -n "${PROBE_NAMESPACE}" \
        get events --sort-by=.lastTimestamp \
        --field-selector involvedObject.name=metrics-scrape-probe >&2 || true
    exit 1
fi
# Befund 1+2 (Step-2-Review): Retry-Loop analog operator-http-smoke.sh
# (Manager-Init-Race) + explizites if-let-Capture, damit curl-Fehler
# eine sinnvolle Diagnose liefern statt `set -e` stumm zu triggern.
METRICS_BODY=""
for _ in $(seq 1 10); do
    if METRICS_BODY="$(kubectl --context "${KIND_CONTEXT}" -n "${PROBE_NAMESPACE}" \
        exec metrics-scrape-probe -- \
        curl -sf --max-time 5 \
        "http://${METRICS_SERVICE_FQDN}:8080/metrics" 2>/dev/null)"; then
        break
    fi
    METRICS_BODY=""
    sleep 1
done
if [[ -z "${METRICS_BODY}" ]]; then
    log "FAIL Step 9b: curl via Service-DNS failed after 10 retries"
    log "  verbose retry for diagnostics:"
    kubectl --context "${KIND_CONTEXT}" -n "${PROBE_NAMESPACE}" exec metrics-scrape-probe -- \
        curl -v --max-time 5 "http://${METRICS_SERVICE_FQDN}:8080/metrics" >&2 || true
    exit 1
fi
# Assert (a): Prometheus-Format-Marker.
if ! grep -qE '^# (HELP|TYPE) ' <<<"${METRICS_BODY}"; then
    log "FAIL Step 9b: response missing '# HELP' / '# TYPE' marker"
    echo "${METRICS_BODY}" | head -10 >&2
    exit 1
fi
# Assert (b): Inhalts-Beweis — mindestens eine library-unabhängige
# Standard-Metrik (OR-verknüpft). `go_goroutines` und
# `process_cpu_seconds_total` kommen aus dem Prometheus-Client-Go-
# Default-Set und sind library-unabhängig garantiert (Befund 5
# Step-2-Review); die drei controller-runtime-Kandidaten sind
# zusätzlich library-spezifisch.
if ! grep -qE '^(go_goroutines|process_cpu_seconds_total|workqueue_depth|rest_client_requests_total|controller_runtime_reconcile_total)( |\{)' <<<"${METRICS_BODY}"; then
    log "FAIL Step 9b: response missing any expected baseline metric"
    echo "${METRICS_BODY}" | head -20 >&2
    exit 1
fi
# Assert (c): Sanity-Mindestlänge (controller-runtime exponiert typisch
# > 80 nicht-leere Zeilen; 20 als konservative Untergrenze gegen den
# „HTTP 200 mit leerem Body, Manager-Init noch nicht durch"-Race).
nonempty_lines="$(grep -vc '^$' <<<"${METRICS_BODY}" || true)"
if [[ "${nonempty_lines}" -lt 20 ]]; then
    log "FAIL Step 9b: response only ${nonempty_lines} non-empty lines, want >= 20"
    echo "${METRICS_BODY}" >&2
    exit 1
fi
log "  /metrics via Service-DNS OK (${nonempty_lines} non-empty lines, format+content+sanity)"

# Step 9c — Struktureller Inhalts-Check der metrics-scrape-ClusterRole
# (slice-M6 §2.2.2). Schützt gegen Drift wie zusätzliche Verben
# (`["get", "list"]`), erweiterte Pfade (`["/metrics", "/admin"]`)
# oder Wildcard (`["*"]`) — auch wenn der Endpoint in M6 unauthen-
# tisiert ist und die Rolle funktional wirkungslos bleibt, soll das
# Pattern-Asset semantisch sauber bleiben.
#
# **v0.2-Hinweis (Befund 6 Step-2-Review):** Bei v0.2-Auth-Filter-
# Aktivierung (ADR-Folge zu ADR 0007) muss dieser jq-Check
# mit-evolvieren. Falls der Auth-Filter `head` für Liveness-Probes
# zusätzlich braucht, lautet die Lockerung z. B.
# `select((.verbs - ["get", "head"]) == [])` oder
# `select(.verbs | sort == ["get", "head"])`. Anpassung gehört in
# **denselben Commit** wie die Auth-Filter-Aktivierung — sonst bricht
# cluster-smoke nach dem Merge.
log "Step 9c: jq-Inhaltscheck der metrics-scrape-ClusterRole"
if ! kubectl --context "${KIND_CONTEXT}" get clusterrole k-deskflight-metrics-scrape -o json \
    | jq -e '.rules[] | select(.nonResourceURLs[]? == "/metrics") | select(.verbs == ["get"])' \
    > /dev/null; then
    log "FAIL Step 9c: ClusterRole missing '/metrics nonResourceURL with verbs=[get]' rule"
    kubectl --context "${KIND_CONTEXT}" get clusterrole k-deskflight-metrics-scrape -o yaml >&2 || true
    exit 1
fi
log "  ClusterRole rule structurally valid (nonResourceURLs=[/metrics], verbs=[get])"

# Step 9d — Lease-Existenz-Check (AR-026, slice-M7 §2.8).
# Attestiert, dass die in M7 scharfgeschaltete Leader-Election im echten
# Cluster funktioniert: das `coordination.k8s.io/leases`-Objekt
# `k-deskflight-operator` wurde vom Operator-Pod erstellt und trägt
# einen nicht-leeren `holderIdentity` (Pod-Name + Renew-UUID).
#
# Retry-Schleife (bis zu 30s in 5s-Schritten) gegen den AR-026-Risiko-
# pfad „Lease-Race beim ersten Reconcile" (siehe slice-M7 §9): der
# Operator-Pod ist via `kubectl wait --for=condition=Available`
# (Step 5) bereits up, aber die Lease-Erst-Acquisition kann ein paar
# Sekunden brauchen, weil `LeaderElectionReleaseOnCancel=true` (siehe
# cmd/operator/main.go) das Lease im Worst-Case-Vorgänger-Zyklus erst
# nach Manager-Start aufnimmt.
log "Step 9d: Lease-Existenz-Check für Leader-Election (AR-026)"
lease_holder=""
for attempt in 1 2 3 4 5 6; do
    lease_holder="$(kubectl --context "${KIND_CONTEXT}" -n k-deskflight-system \
        get leases.coordination.k8s.io k-deskflight-operator \
        -o jsonpath='{.spec.holderIdentity}' 2>/dev/null || true)"
    if [[ -n "${lease_holder}" ]]; then
        break
    fi
    sleep 5
done
if [[ -z "${lease_holder}" ]]; then
    log "FAIL Step 9d: lease k-deskflight-operator missing or holderIdentity empty after 30s"
    kubectl --context "${KIND_CONTEXT}" -n k-deskflight-system get leases.coordination.k8s.io >&2 || true
    kubectl --context "${KIND_CONTEXT}" -n k-deskflight-system describe lease k-deskflight-operator >&2 || true
    exit 1
fi
log "  Lease k-deskflight-operator vorhanden, holderIdentity=${lease_holder}"

log "PASSED — passed-CR (smoke): phase=${phase}, ${condition_type}=${condition_status}; failed-CRs: 4×Phase=Failed with expected reason; HTTP probes OK, Service-DNS OK, ClusterRole structurally valid, Lease scharf"

# Attest-Artefakt: Status-Dump unter /src/.cluster-smoke-attest.yaml
# (Workspace-Mount); CI-Workflow `cluster-smoke.yml` lädt das als
# Run-Artefakt hoch, lokal gitignored. Override via
# CLUSTER_SMOKE_ATTEST_FILE=<pfad>.
#
# Seit slice-M6 §2.8 enthält das Attest **alle fünf** CRs (1 passed +
# 4 failed) — `-l app=…` würde greifen, aber die Smoke-CRs tragen
# kein gemeinsames Label, also einfache `--field-selector`-loses
# `get opendeskpreflightcheck` über den ganzen Namespace.
ATTEST_FILE="${CLUSTER_SMOKE_ATTEST_FILE:-/src/.cluster-smoke-attest.yaml}"
kubectl --context "${KIND_CONTEXT}" -n "${SAMPLE_NAMESPACE}" \
    get opendeskpreflightcheck -o yaml \
    > "${ATTEST_FILE}"
log "attest written to ${ATTEST_FILE} (${#failed_crs[@]} failed + 1 passed CR)"
