# Trigger — `chart-testing` (`ct`) als Quality-Gate aktivieren

**Trigger für:** Erweiterung von `make gates` um `ct lint` und
optional `ct install` zur strukturellen + funktionalen Helm-Chart-
Verifikation jenseits dessen, was `helm lint` und der
`helm-manifests-sync`-Drift-Gate heute leisten.
**Eröffnet:** 2026-05-21 (aus slice-M8 §8 Out-of-Scope verschoben)
**Bezug:** [Slice M8 — Helm-Chart](../in-progress/slice-M8-helm-chart.md),
[`spec/lastenheft.md` `LH-QG-*`](../../../../spec/lastenheft.md),
[ADR 0012 §2.11](../../adr/0012-quality-gates.md),
[ADR 0013](../../adr/0013-cluster-smoke-platform.md)

---

## 1. Kontext

Slice M8 (Helm-Chart) hat drei Helm-Gates aktiviert:

- `make helm-lint` — `helm lint` über Chart-Syntax + Best-Practice-
  Hinweise.
- `make helm-template` — drei Test-Values-Overlays rendern.
- `make helm-manifests-sync` — strukturelle Drift-Detektion gegen
  `deploy/manifests/`.

Das deckt **statische** Chart-Korrektheit ab. `chart-testing`
(Repository [`helm/chart-testing`](https://github.com/helm/chart-testing),
CLI `ct`) ergänzt:

- **`ct lint`** — Chart-Versionsprüfung (`Chart.yaml.version`-Bump
  bei Änderungen am Chart-Verzeichnis), YAML-Lint, Maintainer-Pflege,
  README-Validierung gegen `values.yaml`-Schema.
- **`ct install`** — Install/Upgrade-Test gegen einen echten kind-
  Cluster mit den `ci/test-values/*.yaml`-Overlays (Pendant zu
  unseren `deploy/charts/k-deskflight/test-values/*.yaml`). Helm-
  Test-Hooks werden ausgeführt.

Slice M8 §8 hat das explizit auf später verschoben: „Quality-of-Life-
Tools; ihre Aktivierung wird auf M16 (Release-Slice) verschoben,
sobald die Distributions-Form geklärt ist." Step-8-Diskussion
(2026-05-21) hat den M16-Pin als zu spät identifiziert (Release-Tag-
Slice darf nicht zum Polish-Sammler werden); deshalb dieser
`open/`-Trigger.

---

## 2. Aktivierungs-Anlass

Aktivieren, wenn einer der folgenden Anlässe eintritt:

- **Chart-Drift, die der `helm-manifests-sync`-Gate nicht abfängt**
  (z. B. semantische Schema-Drift, fehlender Version-Bump in
  `Chart.yaml`, README-vs-`values.yaml`-Drift). `ct lint` deckt
  Versions-Bump-Disziplin und Maintainer-Hygiene ab, die heutige
  Gates nicht überwachen.
- **Vor erstem Chart-OCI-Publish nach `ghcr.io/pt9912/charts/`**
  (M16 oder eigener Publish-Slice). `ct install` als
  Pre-Publish-Verifikation reduziert das Risiko, einen unfunktionalen
  Chart hochzuladen.
- **Erweiterung des Charts um Subcharts oder Hooks** (heute
  out-of-scope, aber bei Bedarf in v0.3+). Subchart-Auflösung
  testet `ct install` zuverlässiger als `helm template` alleine.

---

## 3. Zu entscheiden bei Aktivierung

- **Tool-Stage:** Erweiterung der `helm-tools`-Stage um den
  `quay.io/helmpack/chart-testing`-Container, oder eigene
  `chart-testing-tools`-Stage? Letzteres ist sauberer, weil
  chart-testing ein Wrapper-Image mit eigenen Pins ist.
- **CI-Wiring:** `ct lint` als zusätzliches Gate in `make gates`
  einhängen; `ct install` läuft im `cluster-smoke`-Workflow als
  zusätzlicher Matrix-Eintrag (`install-mode: [manifests, helm,
  chart-testing]`) — oder als eigener Workflow?
- **`ct.yaml`-Konfiguration:** `chart-dirs: [deploy/charts]`,
  `chart-repos:` falls Subcharts; `lint-conf`-Anpassungen für
  k-deskflight-spezifische Konventionen (z. B. das fehlende `icon`
  in Chart.yaml als `[INFO]` ignorieren).
- **Versionierung:** `Chart.yaml.version`-Bump-Disziplin formal
  einführen — `ct` verlangt einen Version-Bump bei jeder Chart-
  Änderung. Heute pflegen wir das manuell; mit `ct` wird's enforced.

---

## 4. Nächste Schritte

1. Bei Aktivierung wandert dieser Eintrag nach `next/` (mit
   Scope-Skizze) oder direkt nach `in-progress/` (mit Slice-Plan).
2. Slice-Plan zieht die `ct lint`/`ct install`-Wiring durch:
   Dockerfile-Stage, Makefile-Targets (`make chart-test-lint`,
   `make chart-test-install`), `ct.yaml`-Config, CI-Anbindung.
3. Slice-Closure aktualisiert `slice-M8-helm-chart.md §8` (Out-of-
   Scope-Punkt streichen) und ggf. die v0.2-Roadmap.

---

## 5. Status

Offen. Niedrige Priorität — die heutigen drei Helm-Gates decken die
strukturelle Chart-Korrektheit für v0.2 ausreichend ab. Aktivierung
nicht vor dem ersten Chart-OCI-Publish (M16) sinnvoll, kann aber
auch parallel zu späteren M9–M15-Slices passieren, falls einer der
oben genannten Anlässe eintritt.
