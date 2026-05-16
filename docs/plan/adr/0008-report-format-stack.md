# ADR 0008 — Report-Format-Stack

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[ADR 0005](0005-helm-chart-nicht-im-mvp.md),
[ADR 0007](0007-prometheus-metrik-scope-im-mvp.md)

---

## 1. Kontext

`LH-OP-008` forderte die Entscheidung über das Report-Format. Das
Lastenheft kennt mehrere Berichtsartefakte mit jeweils eigener
Phasenstaffelung:

| Lastenheft-Kennung | Artefakt | Phase | Format |
| ------------------ | -------- | ----- | ------ |
| `LH-F-007`, `LH-F-032`, `LH-DAT-002`, `LH-MVP-002` | Zusammenfassung im CR-Status | MVP | k8s-JSON-kanonisch — nicht entscheidungsoffen |
| `LH-F-027` | Kubernetes Events | v0.2 (`LH-PRI-002`) | Plain Text — k8s-Event-Konvention, nicht entscheidungsoffen |
| `LH-F-028`, `LH-DAT-004` | Optionaler ConfigMap-Report | v0.2 (`LH-PRI-002`) | **offen** — Gegenstand dieser ADR |
| `LH-PRI-003` | HTML-Report | später | offen — nicht Gegenstand dieser ADR |
| `LH-F-029` | Querschnitt: CI/CD-auswertbar | alle Phasen | strukturiert |

Die einzige format-offene Stelle ist der ConfigMap-Report aus
`LH-F-028`. Diese ADR beantwortet die Frage und macht den gesamten
Format-Stack einmalig explizit, damit für künftige Diskussionen ein
klarer Anker existiert.

**Begleitende Lastenheft-Anpassungen** (im selben Commit):
`LH-F-028` wird auf das zwei-Keys-Layout präzisiert, `LH-PRI-002`
Eintrag wird um die Format-Konkretisierung ergänzt.

---

## 2. Entscheidung

### 2.1 Phasen-Format-Karte

| Phase | Artefakt | Format | Quelle |
| ----- | -------- | ------ | ------ |
| MVP   | CR-Status (`LH-F-007`, `LH-F-032`) | JSON (Kubernetes-Status-Konvention) | nicht entscheidungsoffen, k8s-kanonisch |
| v0.2  | Kubernetes Events (`LH-F-027`) | Plain Text (`reason`/`message`) | k8s-Event-Konvention |
| v0.2  | ConfigMap (`LH-F-028`) | **zwei Daten-Keys**: `report.yaml` + `report.md` | diese ADR §2.2 |
| `LH-PRI-003` | HTML-Report | offen | spätere ADR |

### 2.2 ConfigMap-Layout für `LH-F-028`

Die optionale ConfigMap-Veröffentlichung ab v0.2 enthält **zwei**
Daten-Keys:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: <cr-name>-preflight-report
  namespace: <namespace-der-cr>
  ownerReferences:
    - apiVersion: k-deskflight.geo-terrain.net/v1alpha1
      kind: OpenDeskPreflightCheck
      name: <cr-name>
      uid: <cr-uid>
      controller: true
      blockOwnerDeletion: true
data:
  report.yaml: |
    # strukturierte, maschinenlesbare Repräsentation
    # erfüllt LH-F-029 (CI/CD-/GitOps-Auswertbarkeit)
    ...
  report.md: |
    # menschenlesbares Markdown
    # erfüllt LH-DAT-004 (menschenlesbare Reportdaten)
    ...
```

Begründung der zwei-Keys-Wahl:

- `report.yaml` bedient `LH-F-029` (CI/CD-Pipelines, GitOps-Tools
  können das deterministisch parsen).
- `report.md` bedient `LH-DAT-004` und ist optimal für PR-Reviews
  in GitOps-Workflows sowie für direkte Anzeige im Cluster
  (`kubectl get configmap <name> -o jsonpath='{.data.report\.md}'`).
- Beide leben in einer einzigen ConfigMap → ein API-Objekt, ein
  Lifecycle, eine Owner-Reference.

### 2.3 Naming und Lifecycle

- Name: `<cr-name>-preflight-report` im **selben Namespace** wie die
  CR. Folgt der namespaced-Scope-Entscheidung aus `ADR 0006`.
- `ownerReferences` auf die CR mit `controller: true` und
  `blockOwnerDeletion: true`. Damit garbage-collected Kubernetes die
  ConfigMap automatisch beim Löschen der CR.
- Die ConfigMap wird beim Abschluss jedes Prüflaufs **vollständig
  überschrieben**. Kein Append, keine History; Geschichtsdaten sind
  über Git/Backup-Pfade abzubilden, nicht im laufenden Cluster.

### 2.4 Opt-in, nicht Opt-out

`LH-F-028` ist „kann optional". Die ConfigMap wird nur erzeugt, wenn
der Anwender es im Spec aktiviert. Konkretes Spec-Feld
(`spec.report.configMap.enabled` o. ä.) ist nicht Gegenstand dieser
ADR und entsteht mit dem Pflichtenheft (`LH-VM-002`).

### 2.5 Trennung von CR-Status und ConfigMap

CR-Status (MVP) und ConfigMap-Report (v0.2) sind **redundante
Darstellungen** derselben Prüfdaten in unterschiedlichen Formaten.
Die ConfigMap leitet sich vom CR-Status ab; sie ist keine
Konkurrenz-Datenquelle und keine erweiterte Datenquelle. Das
verhindert Drift und reduziert Generator-Komplexität.

---

## 3. Konsequenzen

- `LH-F-028` wird im Lastenheft um den Format-Verweis auf diese ADR
  geschärft (zwei Keys statt eine pauschale „menschenlesbar"-Aussage).
- `LH-PRI-002` Eintrag „Report als ConfigMap (`LH-F-028`)" wird
  konkretisiert: Format YAML + Markdown gemäß `ADR 0008`.
- `LH-OP-008` wird im Lastenheft als geschlossen mit dieser ADR
  markiert (Formelhilfe aus `ADR 0002`).
- Pflichtenheft (`LH-VM-002`) konkretisiert: Spec-Felder für die
  Aktivierung der ConfigMap, exaktes YAML-Schema, Markdown-Layout
  (Heading-Hierarchie, Tabellen-Schema).
- HTML-Report (`LH-PRI-003`) bleibt formal offen; eine spätere ADR
  referenziert diese ADR als Vorlage für den Format-Stack und
  ergänzt die HTML-Variante.
- `LH-RISK-005` (Secret-Leaks): Beide Format-Generatoren müssen den
  Filter aus `LH-NF-007`/`LH-SEC-002` einhalten — das ist im
  Pflichtenheft als Generatoren-Vertrag zu verankern.

---

## 4. Nicht Gegenstand dieser ADR

- **HTML-Report-Format** und Generierungsstrategie — `LH-PRI-003`,
  eigene Folge-ADR. Die ADR sollte diese ADR 0008 als Vorlage des
  Phasen-Format-Stacks referenzieren und HTML konsistent einbetten.
- **JUnit-XML-Format** für CI-Pipelines: Jenkins, GitLab CI und
  GitHub Actions konsumieren JUnit-XML nativ als Test-Result-Format.
  Eine JUnit-Variante der Preflight-Ergebnisse wäre attraktiv, falls
  Anwender den Operator als Teil von CI-Test-Suites einsetzen
  (`LH-ZA-003` CI/CD-Sicht). Das Lastenheft nennt JUnit-XML aktuell
  nicht; eine spätere ADR kann das nachholen, wenn der Bedarf konkret
  wird. Argument *gegen* eine voreilige Aufnahme: ein Preflight-Check
  ist semantisch keine Unit-/Integrations-Test-Suite, sondern ein
  Cluster-Readiness-Bericht — JUnit als Format würde Anwender in eine
  Test-Framework-Erwartung drängen.
- **PDF-Report** und **CSV-Report** — nicht Lastenheft-Inhalt, keine
  derzeitige Notwendigkeit.
- **Exaktes YAML-Schema** der `report.yaml`-Datei (Field-Namen,
  Reihenfolge, Pflicht-Felder) — Pflichtenheft (`LH-VM-002`).
- **Markdown-Layout** (Heading-Hierarchie, Tabellen-Schema,
  Color-Coding via Emoji) — Pflichtenheft oder eigener Style-Guide.
- **Spec-Felder** zur Aktivierung der ConfigMap (`spec.report.…`) —
  Pflichtenheft.
- **Persistenz oder Versionierung** der Reports außerhalb des Clusters
  (Object Storage, Git-Push) — operative Folgearbeit, nicht
  Lastenheft-Inhalt.
- **Lokalisierung** des Markdown-Reports (Deutsch/Englisch). Die
  Operator-Ausgaben sind gemäß `LH-NF-021` Englisch; der Markdown-
  Report folgt dieser Linie, ohne dass diese ADR es zusätzlich
  bindet.
