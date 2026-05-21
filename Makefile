# k-deskflight — OpenDesk Preflight Operator
#
# Docker-only workflow per docs/plan/planning/done/slice-M1-repo-skeleton.md
# §2.1. Build/lint/test/coverage/govulncheck rufen alle
# `docker build --target <stage>` bzw. `docker run`; das Repository
# hat keine host-seitige Go-Toolchain-Anforderung.
#
# Ausnahme: `make doc-refs` ruft `bash scripts/verify-doc-refs.sh`
# direkt auf dem Host — ein 100-Zeilen-Bash-Skript ohne Go-Toolchain-
# Bedarf zu containerisieren wäre Overhead ohne Nutzen. Das ist die
# einzige Carveout-Stelle der Docker-only-Konvention.
#
# Quality-Gates (ADR 0012 §2.11):
#   make gates           — Pflicht-Gates der Inner-Loop, PR-blockierend.
#   make security-gates  — externe Gates (govulncheck). Mit VER=X.Y.Z
#                          zusätzlich Trivy image-scan (LH-QG-007).

IMAGE := k-deskflight

# Image-Publish-Pattern (slice-M7 §2.3, m-trace-Adaption). Default-Repo
# zeigt auf GHCR; override via IMAGE_REGISTRY/IMAGE_OWNER falls Anwender
# Mirror-Registries oder Fork-Repos nutzen. Der GHCR-Push-Pfad nutzt
# das Approval-Pattern aus ADR 0011 §2.5 (siehe `image-publish-guard`).
IMAGE_REGISTRY ?= ghcr.io
IMAGE_OWNER    ?= pt9912
IMAGE_REPO     := $(IMAGE_REGISTRY)/$(IMAGE_OWNER)/$(IMAGE)

# Coverage-Schwelle für `make coverage-gate`. M1 startete mit 0
# (Smoke-Pfad, bootstrap-aware in coverage-gate.sh); M4 zieht 90 %
# vor (slice-M4 §3.3 / §7 — Adapter sind ab M4 via fake clientset
# getestet, M6 erbt den strikten Wert). Override via
# `make coverage-gate THRESHOLD=…`.
THRESHOLD ?= 90

# `--no-cache-filter <stage>` zwingt BuildKit, die genannte Stage neu
# zu evaluieren, ohne die `deps`-Layer (Go-Module-Cache) zu verwerfen.
# Ohne diesen Filter kann ein Stale-Layer-Hash Test-/Lint-/Coverage-
# Failures maskieren.
NO_CACHE_FILTER_TEST     := --no-cache-filter test
NO_CACHE_FILTER_LINT     := --no-cache-filter lint
NO_CACHE_FILTER_COVERAGE := --no-cache-filter coverage

# govulncheck-Pin (ADR 0012 §2.8). Pin-Hebung ist Routine und benötigt
# keine ADR; Begründung gehört in den Commit-Body.
GOVULNCHECK_VERSION ?= v1.1.4

# Trivy-Pin (ADR 0012 §2.9, slice-M7 §2.4). m-trace-Match. Pin-Hebung
# Routine ohne ADR.
TRIVY_IMAGE ?= aquasec/trivy:0.59.1

.PHONY: help build compile deps tools lint test coverage coverage-gate doc-refs \
        manifests generated-drift-check govulncheck image-build run gates \
        security-gates cluster-smoke cluster-smoke-image cluster-smoke-cleanup \
        operator-http-smoke image-publish-dry-run image-publish-guard \
        image-publish image-scan release-guard release-guard-test clean \
        helm-tools-image helm-lint

# controller-gen-Pin (slice-M2 §2.4, ADR 0012 §2.8 Abs. 3). Hebung ist
# Routine ohne ADR; Override via `make manifests CONTROLLER_GEN_VERSION=…`.
CONTROLLER_GEN_VERSION ?= v0.21.0

# Cluster-Smoke-Pins (ADR 0013 §2.4). Hebung Routine, override via
# `make cluster-smoke KIND_VERSION=… KUBECTL_VERSION=… K8S_VERSION=…`.
KIND_VERSION    ?= v0.31.0
KUBECTL_VERSION ?= v1.34.0
K8S_VERSION     ?= v1.34.0
CLUSTER_NAME    ?= k-deskflight-smoke

# Helm-Pin (slice-M8 §2.5). Hebung Routine ohne ADR (ADR 0012 §2.8
# Abs. 3); Override via `make helm-lint HELM_VERSION=…`.
HELM_VERSION    ?= v3.16.4
CLUSTER_KEEP    ?= 0

help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ---- inner-loop ------------------------------------------------------------

build: ## Build the runtime image (distroless static, nonroot).
	docker build --target runtime -t $(IMAGE):go .

compile: ## Compile-only feedback loop (no tests, no lint).
	docker build --target compile -t $(IMAGE):go-compile .

deps: ## Resolve Go module dependencies (deps-cache layer).
	docker build --no-cache --target deps -t $(IMAGE):go-deps .

lint: ## golangci-lint with the SOLID-style mandatory profile.
	docker build $(NO_CACHE_FILTER_LINT) --target lint -t $(IMAGE):go-lint .

test: ## Run `go test ./...` inside Docker.
	docker build $(NO_CACHE_FILTER_TEST) --target test -t $(IMAGE):go-test .

coverage-gate: ## Coverage threshold gate (M1 = 0, M6 hebt auf 90).
	docker build $(NO_CACHE_FILTER_COVERAGE) \
	    --target coverage \
	    --build-arg COVERAGE_THRESHOLD=$(THRESHOLD) \
	    -t $(IMAGE):go-coverage .

# Alias: gleicher Pfad wie das Gate, aber als „informational"-Form
# für Tagworkflow lesbar — der Stage liefert per-Funktion-Report auf
# stdout, das Gate bleibt aktiv.
coverage: coverage-gate

doc-refs: ## Verify local markdown link targets (LH-QG-008).
	bash scripts/verify-doc-refs.sh

# ---- generators ------------------------------------------------------------

tools: ## Build the tools image with controller-gen pinned.
	docker build --target tools \
	    --build-arg CONTROLLER_GEN_VERSION=$(CONTROLLER_GEN_VERSION) \
	    -t $(IMAGE):go-tools .

# `make manifests` regeneriert die kubebuilder-Output-Artefakte
# (architecture.md AR-007, slice-M2 §2.3):
#   - api/v1alpha1/zz_generated.deepcopy.go (object-Generator)
#   - config/crd/<group>_<resource>.yaml (crd-Generator)
#   - config/rbac/role.yaml (rbac-Generator aus den Markern am
#     Reconcile-Receiver in internal/hexagon/application/)
# `--user $(id -u):$(id -g)` schreibt die Outputs als Caller, nicht
# als root. GOCACHE/GOMODCACHE auf /tmp, weil controller-gen intern
# `go list` aufruft und das Default-Cache-Verzeichnis (~/.cache/go-build)
# mit nicht-root-User nicht beschreibbar ist. controller-gen läuft
# idempotent.
manifests: tools
	docker run --rm \
	    --user "$$(id -u):$$(id -g)" \
	    -v "$(CURDIR):/src" \
	    -w /src \
	    -e GOCACHE=/tmp/gocache \
	    -e GOMODCACHE=/tmp/gomodcache \
	    $(IMAGE):go-tools \
	    /go/bin/controller-gen \
	        object:headerFile=hack/boilerplate.go.txt \
	        crd \
	        rbac:roleName=k-deskflight-operator-cluster \
	        paths=./api/... \
	        paths=./internal/hexagon/application/... \
	        output:crd:dir=config/crd \
	        output:rbac:dir=config/rbac

# Drift-Gate: regeneriert Manifeste und prüft via `git diff --exit-code`,
# dass nichts vom committeten Stand abweicht (ADR 0012 §2.7, LH-QG-005).
# Pfade: alle controller-gen-Outputs.
generated-drift-check: manifests
	@git diff --exit-code -- \
	    api/v1alpha1/zz_generated.deepcopy.go \
	    config/crd/ \
	    config/rbac/ \
	    >/dev/null 2>&1 || { \
	        echo "[generated-drift-check] FAILED — `make manifests`-Output weicht vom committeten Stand ab:" >&2; \
	        git diff --stat -- api/v1alpha1/zz_generated.deepcopy.go config/crd/ config/rbac/ >&2; \
	        exit 1; \
	    }
	@echo "[generated-drift-check] passed"

# ---- helm chart ------------------------------------------------------------
# Slice-M8 §2.5: helm-Tooling über die `helm-tools`-Stage des Dockerfiles.
# Konvention parallel zu `tools` / `cluster-smoke-image`: erst Image
# bauen, dann via `docker run` aufrufen — Docker-only, keine Host-helm-
# Anforderung.

helm-tools-image: ## Build the helm-tools image with helm pinned.
	docker build --target helm-tools \
	    --build-arg HELM_VERSION=$(HELM_VERSION) \
	    -t $(IMAGE):helm-tools .

helm-lint: helm-tools-image ## helm lint deploy/charts/k-deskflight/ (slice-M8 §2.5).
	docker run --rm \
	    -v "$(CURDIR):/src" \
	    -w /src \
	    $(IMAGE):helm-tools \
	    helm lint deploy/charts/k-deskflight/

# ---- gate bundles ----------------------------------------------------------

gates: build lint test coverage-gate doc-refs generated-drift-check ## Inner-loop Pflicht-Gates (ADR 0012 §2.11).
	@echo "[gates] passed"

# `govulncheck` läuft in einem golang:1.26.3-Container und installiert
# den Tool-Pin per `go install`. Funktionsbasiert — meldet nur tatsäch-
# lich aufgerufene Vulnerable-Funktionen (ADR 0012 §2.8). In M1 ohne
# externe Deps trivial grün; strikt PR-blockierend wird das Gate spätes-
# tens in M6 mit dem ersten echten Dependency-Tree (controller-runtime
# etc.).
govulncheck: ## Run govulncheck (function-based scanning).
	docker run --rm -v "$(CURDIR):/src" -w /src golang:1.26.3 \
	    bash -c 'go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) && \
	             "$$(go env GOPATH)/bin/govulncheck" ./...'

# security-gates bündelt die externen Pflicht-Gates (slice-M7 §2.4,
# ADR 0012 §2.11). govulncheck läuft immer (Inner-Loop-Pfad bleibt
# schnell); image-scan kommt nur dazu, wenn VER gesetzt ist, weil
# Trivy einen gebauten GHCR-Tagged-Image-Pfad braucht. Mit VER ist
# das Bündel der Release-Gate-Aufruf (`make security-gates VER=0.1.0`).
security-gates: govulncheck $(if $(strip $(VER)),image-scan) ## govulncheck immer; mit VER=X.Y.Z zusätzlich Trivy image-scan.
	@echo "[security-gates] passed"

# ---- cluster-smoke ---------------------------------------------------------
# ADR 0013: kind als Default-Plattform für LH-AK-001/-002/-005-Attest.
# Tools (kind, kubectl, docker-cli) leben in der `smoke`-Dockerfile-
# Stage; `cluster-smoke` ruft `docker run` mit Docker-out-of-Docker
# (Host-Socket-Mount) und `--network host`, damit der kind-apiserver
# (127.0.0.1:<port>) aus dem Container erreichbar bleibt.
# Nicht in `make gates` — opt-in pro ADR 0013 §2.5.

cluster-smoke-image: ## Build the smoke tool image (kind + kubectl + docker-cli).
	docker build --target smoke \
	    --build-arg KIND_VERSION=$(KIND_VERSION) \
	    --build-arg KUBECTL_VERSION=$(KUBECTL_VERSION) \
	    -t $(IMAGE):go-smoke .

cluster-smoke: build cluster-smoke-image ## End-to-end Cluster-Smoke gegen kind.
	docker run --rm \
	    --network host \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    -v "$(CURDIR):/src" \
	    -w /src \
	    -e CLUSTER_NAME=$(CLUSTER_NAME) \
	    -e K8S_VERSION=$(K8S_VERSION) \
	    -e IMAGE_TAG=$(IMAGE):go \
	    -e CLUSTER_KEEP=$(CLUSTER_KEEP) \
	    $(IMAGE):go-smoke \
	    bash scripts/cluster-smoke.sh

cluster-smoke-cleanup: cluster-smoke-image ## Lösche den smoke-Cluster manuell.
	docker run --rm \
	    --network host \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    $(IMAGE):go-smoke \
	    kind delete cluster --name $(CLUSTER_NAME)

# operator-http-smoke: standalone HTTP-Probe gegen /healthz, /readyz,
# /metrics. Setzt voraus, dass der smoke-Cluster läuft
# (z.B. `make cluster-smoke CLUSTER_KEEP=1` lokal). Wird sonst von
# cluster-smoke.sh als finale Stufe automatisch mitgerufen.
operator-http-smoke: cluster-smoke-image ## HTTP-Smoke gegen healthz/readyz/metrics.
	docker run --rm \
	    --network host \
	    -v /var/run/docker.sock:/var/run/docker.sock \
	    -v "$(CURDIR):/src" \
	    -w /src \
	    -e CLUSTER_NAME=$(CLUSTER_NAME) \
	    $(IMAGE):go-smoke \
	    bash scripts/operator-http-smoke.sh

# ---- ops -------------------------------------------------------------------

# image-build ist zweigleisig (slice-M7 §2.3, m-trace-Adaption):
#   - Ohne VER:    baut $(IMAGE):go  (Dev-Tag; gleichwertig zu `build`).
#   - Mit VER:     baut $(IMAGE_REPO):vX.Y.Z (GHCR-Tag für Release-Pfad).
# Migration ist additiv: der Inner-Loop-Aufruf `make image-build` bleibt
# funktional unverändert, die Tag-Variante kommt für M7-Release-Pfad dazu.
image-build: ## Build runtime image. With VER=X.Y.Z builds $(IMAGE_REPO):vX.Y.Z; otherwise builds $(IMAGE):go.
ifeq ($(strip $(VER)),)
	docker build --target runtime -t $(IMAGE):go .
else
	docker build --target runtime -t $(IMAGE_REPO):v$(VER) .
endif

# image-publish-dry-run baut das GHCR-tagged Image und verifiziert lokal
# via `docker image inspect`, ohne zu pushen. Release-Rehearsal-Pfad
# (slice-M7 §4 step 5). VER ist Pflicht — fail-fast vor `image-build`,
# damit ein vergessenes VER nicht den :go-Build triggert.
image-publish-dry-run: ## Build the GHCR-tagged image and announce the push target. Requires VER.
	@test -n "$(strip $(VER))" || { echo "image-publish-dry-run: VER=X.Y.Z is required (e.g. make image-publish-dry-run VER=0.1.0)" >&2; exit 2; }
	$(MAKE) --no-print-directory image-build VER=$(VER)
	docker image inspect $(IMAGE_REPO):v$(VER) >/dev/null
	@echo "[image-publish-dry-run] would push: $(IMAGE_REPO):v$(VER)"

# image-publish-guard verifiziert die operator-approved Approval-Variable
# (slice-M7 §2.3, ADR 0011 §2.5). Ohne sie schlägt `image-publish` mit
# Exit 2 fehl — kein versehentlicher GHCR-Push.
image-publish-guard:
	@test "$(K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED)" = "1" || { echo "Refusing to publish images without K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED=1" >&2; exit 2; }

# image-scan scannt das GHCR-tagged Image mit Trivy (slice-M7 §2.4,
# ADR 0012 §2.9). Die Recipe ist intentionell schlank — die Trivy-
# Logik (Two-Pass CRITICAL/HIGH + MEDIUM, Vulnignore-Render,
# Docker-Socket-Mount-Pattern) lebt in scripts/image-scan.sh und ist
# damit unabhängig vom Make-Wrapper testbar/aufrufbar.
# IMAGE_REPO + TRIVY_IMAGE werden via Env in die Skript-Schicht
# durchgereicht; ohne sie greifen die Skript-Defaults.
image-scan: ## Trivy-Scan des GHCR-tagged Image. Requires VER. CRITICAL/HIGH break; MEDIUM reported.
	IMAGE_REPO=$(IMAGE_REPO) TRIVY_IMAGE=$(TRIVY_IMAGE) bash scripts/image-scan.sh $(VER)

# image-publish pushed das GHCR-tagged Image. Pflicht: VER=X.Y.Z und
# K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED=1. Ein Push ohne Approval ist
# bewusst nicht möglich (ADR 0011 §2.5). Approval-Guard läuft als
# Prereq und exited fail-fast; VER-Check vor `image-build`-Sub-Make,
# damit ein vergessenes VER nicht zum :go-Build degeneriert.
image-publish: image-publish-guard ## Push the GHCR-tagged image. Requires VER and K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED=1.
	@test -n "$(strip $(VER))" || { echo "image-publish: VER=X.Y.Z is required (e.g. make image-publish VER=0.1.0)" >&2; exit 2; }
	$(MAKE) --no-print-directory image-build VER=$(VER)
	docker push $(IMAGE_REPO):v$(VER)

run: build ## Run the runtime image in the foreground.
	docker run --rm --name $(IMAGE)-smoke $(IMAGE):go

# release-guard ist die Pre-Release-Konsistenzprüfung (slice-M7 §2.5,
# ADR 0011 §2.5). Erforderlich: VER=X.Y.Z (ohne v-Prefix) und
# K_DESKFLIGHT_RELEASE_APPROVED=1. Das Skript validiert nur — die
# tatsächliche Tag-Erzeugung (`git tag -a vX.Y.Z`) ist ein separater
# Schritt (slice-M7 §4 step 15).
#
# Anders als das m-trace-Pendant ohne fest verdrahtetes DRY_RUN: der
# Guard hat ohnehin keine Side-Effects (kein Tag, kein Push, kein
# File-Edit), der DRY_RUN-Switch wechselt nur das Erfolgs-Label.
release-guard: ## Pre-release consistency guard. Requires VER and K_DESKFLIGHT_RELEASE_APPROVED=1.
	@test -n "$(strip $(VER))" || { echo "release-guard: VER=X.Y.Z is required (e.g. make release-guard VER=0.1.0)" >&2; exit 2; }
	bash scripts/release-guard.sh $(VER)

# release-guard-test exerciert die Failure-Paths des Guards gegen
# synthetische Fixtures unter /tmp. Nicht im CI (m-trace-Konvention).
release-guard-test: ## Local failure-path tests for release-guard.sh (not run in CI).
	bash scripts/test-release-guard.sh

clean: ## Remove all skeleton-related images.
	-docker rmi $(IMAGE):go $(IMAGE):go-test $(IMAGE):go-lint \
	    $(IMAGE):go-compile $(IMAGE):go-deps $(IMAGE):go-coverage 2>/dev/null
