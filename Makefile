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
#   make security-gates  — externe Gates (govulncheck; Image-Scan in M7).

IMAGE := k-deskflight

# Coverage-Schwelle für `make coverage-gate`. M1 startet mit 0 (Smoke-
# test, bootstrap-aware in coverage-gate.sh); M6 hebt auf 90 % gemäß
# ADR 0012 §2.5. Override via `make coverage-gate THRESHOLD=…`.
THRESHOLD ?= 0

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

.PHONY: help build compile deps tools lint test coverage coverage-gate doc-refs \
        manifests generated-drift-check govulncheck image-build run gates \
        security-gates cluster-smoke cluster-smoke-image cluster-smoke-cleanup \
        operator-http-smoke clean

# controller-gen-Pin (slice-M2 §2.4, ADR 0012 §2.8 Abs. 3). Hebung ist
# Routine ohne ADR; Override via `make manifests CONTROLLER_GEN_VERSION=…`.
CONTROLLER_GEN_VERSION ?= v0.21.0

# Cluster-Smoke-Pins (ADR 0013 §2.4). Hebung Routine, override via
# `make cluster-smoke KIND_VERSION=… KUBECTL_VERSION=… K8S_VERSION=…`.
KIND_VERSION    ?= v0.31.0
KUBECTL_VERSION ?= v1.34.0
K8S_VERSION     ?= v1.34.0
CLUSTER_NAME    ?= k-deskflight-smoke
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

security-gates: govulncheck ## Externe Pflicht-Gates (Vuln-Scan; Image-Scan in M7).
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

image-build: build ## Alias for `build` — slice-plan-Konvention.

run: build ## Run the runtime image in the foreground.
	docker run --rm --name $(IMAGE)-smoke $(IMAGE):go

clean: ## Remove all skeleton-related images.
	-docker rmi $(IMAGE):go $(IMAGE):go-test $(IMAGE):go-lint \
	    $(IMAGE):go-compile $(IMAGE):go-deps $(IMAGE):go-coverage 2>/dev/null
