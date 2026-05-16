# ADR 0007 — Prometheus-Metrik-Scope im MVP

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[ADR 0005](0005-helm-chart-nicht-im-mvp.md)
**Schärft:** `LH-PRI-001` (Aufnahme von `LH-SST-004`),
`LH-PRI-002` (Verfeinerung des Metrik-Eintrags),
`LH-MVP-002` (Aufnahme des `/metrics`-Endpoints).

---

## 1. Kontext

`LH-OP-007` forderte die Entscheidung über Prometheus Metrics im MVP.
Drei betroffene Lastenheftaussagen:

- `LH-NF-008`: „Der Operator soll **eigene** Metriken bereitstellen
  können."
- `LH-SST-004`: „Das System soll Metriken in einem Prometheus-
  kompatiblen Format bereitstellen können."
- `LH-PRI-001` (MVP-Muss) enthält weder `LH-NF-008` noch `LH-SST-004`;
  `LH-PRI-002` (v0.2-Soll) führt „Prometheus Metrics (LH-NF-008,
  LH-SST-004)".

Das Standard-Framework für Go-Operators (`controller-runtime`)
exponiert „out of the box" einen `/metrics`-Endpoint mit
Framework-Defaults (Workqueue-Tiefe, Reconcile-Durations, REST-Client-
Counter). „Eigene" Domänenmetriken (z. B. `kdeskflight_preflight_-
checks_total`) müssten dagegen explizit implementiert werden und
brauchen ein stabiles Metrik-Schema, das die CRD-`v1alpha1`-Phase
übersteht.

Damit zerfällt `LH-NF-008`/`LH-SST-004` in zwei Teilfragen:

1. Prometheus-Format vorhanden? — Framework liefert das praktisch
   kostenlos.
2. Eigene Domänenmetriken vorhanden? — Eigener Design-Aufwand mit
   Schema-Stabilität als Voraussetzung.

---

## 2. Entscheidung

### 2.1 Scope im MVP (v0.1, `LH-REL-001`)

Das MVP exponiert genau einen `/metrics`-Endpoint in Prometheus-
kompatiblem Text-Format. Dieser Endpoint liefert ausschließlich die
**Framework-Default-Metriken** des `controller-runtime`-Stacks
(Workqueue, Reconcile-Latenzen, REST-Client). **Eigene k-deskflight-
Domänenmetriken sind nicht Bestandteil des MVP** und werden mit v0.2
nachgereicht.

Damit erfüllt das MVP `LH-SST-004` (Prometheus-kompatibles Format ist
gegeben), aber **nicht** `LH-NF-008` in dessen wörtlicher Lesart
„eigene Metriken" — `LH-NF-008` wird v0.2 vollständig eingelöst.

### 2.2 Ein Endpoint, gemeinsames Registry

Das MVP verwendet **einen** `/metrics`-Endpoint mit gemeinsamem
Prometheus-Registry. Eine Trennung in zwei Endpoints
(Framework-`/metrics` vs. Domain-`/metrics/preflight`) wird verworfen:

- Prometheus-Konvention im Kubernetes-Operator-Ökosystem ist ein
  Endpoint pro Operator (`cert-manager`, `external-dns`, `argo-cd`-
  Komponenten verfahren so).
- Zwei Endpoints würden Scrape-Konfiguration verkomplizieren (zwei
  Targets oder zwei Jobs) ohne Mehrwert.
- Die etablierte Trennung läuft über **Metric-Namen-Prefixes**, nicht
  über Endpoints.

Eine spätere Public/Private-Endpoint-Variante (öffentlich reduzierter
Endpoint, intern detailreich) bleibt eine Architektur-Frage und ist
nicht Gegenstand dieser ADR (siehe §4).

### 2.3 Namensraum-Trennung Framework vs. Domäne

Künftige v0.2-Domänenmetriken werden über Name-Prefix-Trennung von
den Framework-Metriken abgegrenzt:

- Framework-Metriken behalten ihre `controller-runtime`-/REST-Client-
  Prefixe (`controller_runtime_*`, `rest_client_*`, `workqueue_*`).
- k-deskflight-eigene Metriken werden mit `kdeskflight_*` präfixiert.

Das konkrete v0.2-Metrik-Set (Counter, Gauges, Histograms, Labels) ist
nicht Gegenstand dieser ADR und entsteht mit der v0.2-Roadmap.

### 2.4 RBAC, Probes, Doku

Der MVP-Endpoint wird über die mitgelieferten Manifeste
(`deploy/manifests/`, gemäß `ADR 0005`) scrapebar gemacht: passende
ServiceAccount, ClusterRole und Binding gemäß `LH-AK-015`. Die
Anwender-Dokumentation beschreibt, welches Metrik-Set erwartbar ist
(Framework-Defaults) und welches v0.2 dazukommt. Eine Aufnahme in
`/livez`-/`/readyz`-Probes ist nicht Gegenstand dieser ADR.

---

## 3. Konsequenzen

- `LH-SST-004` ist ab MVP erfüllt und wandert deshalb in `LH-PRI-001`.
- `LH-NF-008` bleibt v0.2-Soll (`LH-PRI-002`); der Eintrag wird auf
  „Eigene Domänen-Metriken" zugespitzt, weil das Format-Versprechen
  durch MVP bereits eingelöst ist.
- `LH-MVP-002` wird um den `/metrics`-Endpoint (Framework-Defaults)
  ergänzt.
- Es wird **kein** metrik-spezifisches Abnahmekriterium (`LH-AK-*`)
  eingeführt — die Framework-Default-Metriken werden vom
  `controller-runtime`-Stack selbst getestet. Das MVP-Pflichtenheft
  soll lediglich verifizieren, dass der Endpoint erreichbar und im
  Prometheus-Format ist (Smoketest-Pfad, analog
  `LH-AK-002` „Operator startbar").
- `LH-RISK-002` (Zu großer Projektumfang): Domänenmetriken im MVP
  hätten Schema-Design und Stabilisierungsaufwand bedeutet; Option C
  hält den MVP-Scope klein und liefert dennoch frühe
  Beobachtbarkeit.
- `LH-OP-007` wird in §22 als geschlossen mit dieser ADR markiert
  (`ADR 0002 §7` Formelhilfe).

---

## 4. Nicht Gegenstand dieser ADR

- **Konkrete v0.2-Metrik-Liste** (Name, Typ, Labels, Kardinalitäts-
  Budget) — Folgearbeit, eigene ADR oder v0.2-Roadmap.
- **Port- und Authentifizierungs-Layout** des MVP-Endpoints
  (`:8080/metrics` als `controller-runtime`-Default, TLS-Termination,
  Bearer-Token-Auth-Webhook) — entsteht mit dem Pflichtenheft
  (`LH-VM-002`).
- **Public/Private-Endpoint-Split** (öffentlich scrapebare Reduktion
  vs. interner Volldetails) — eigene Architektur-ADR, falls Bedarf
  entsteht.
- **OpenMetrics/Exemplar-Support** und **Native-Histograms** —
  spätere Capability-Hebung; nicht im MVP-Versprechen.
- **Alert-Regeln und Dashboard-Vorlagen** — operative Folgearbeit
  nach v0.2 (Anwender bauen selbst, oder optionales Beispielset
  später).
- **Aufnahme der Metriken in einen ServiceMonitor**
  (Prometheus-Operator-Pattern) — operative Folgearbeit, ggf.
  Bestandteil des Helm Charts ab v0.2 (siehe `ADR 0005`).
