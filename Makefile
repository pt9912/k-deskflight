# k-deskflight — OpenDesk Preflight Operator
#
# Docker-only workflow per docs/plan/planning/in-progress/
# slice-M1-repo-skeleton.md §2.1. Build/lint/test/coverage rufen
# `docker build --target <stage>`; das Repository hat keine
# host-seitige Go-Toolchain-Anforderung.
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

.PHONY: help build compile deps lint test coverage coverage-gate doc-refs \
        govulncheck image-build run gates security-gates clean

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

# ---- gate bundles ----------------------------------------------------------

gates: lint test coverage-gate doc-refs ## Inner-loop Pflicht-Gates (ADR 0012 §2.11).
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

# ---- ops -------------------------------------------------------------------

image-build: build ## Alias for `build` — slice-plan-Konvention.

run: build ## Run the runtime image in the foreground.
	docker run --rm --name $(IMAGE)-smoke $(IMAGE):go

clean: ## Remove all skeleton-related images.
	-docker rmi $(IMAGE):go $(IMAGE):go-test $(IMAGE):go-lint \
	    $(IMAGE):go-compile $(IMAGE):go-deps $(IMAGE):go-coverage 2>/dev/null
