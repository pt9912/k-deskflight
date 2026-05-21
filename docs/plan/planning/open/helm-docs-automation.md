# Trigger — `helm-docs`-Automatisierung der Chart-`README.md`

**Trigger für:** Aktivierung von [`helm-docs`](https://github.com/norwoodj/helm-docs)
zur automatischen Generierung der Chart-`README.md` aus
`values.yaml`-Kommentaren und `Chart.yaml`-Metadaten.
**Eröffnet:** 2026-05-21 (aus slice-M8 §8 Out-of-Scope verschoben)
**Bezug:** [Slice M8 — Helm-Chart](../in-progress/slice-M8-helm-chart.md),
[`deploy/charts/k-deskflight/values.yaml`](../../../../deploy/charts/k-deskflight/values.yaml),
[ADR 0012 §2.11](../../adr/0012-quality-gates.md)

---

## 1. Kontext

Slice M8 hat das Chart unter `deploy/charts/k-deskflight/` angelegt
mit `values.yaml` voller dokumentierter Kommentar-Blöcke (sieben
Top-Level-Slots, jeder mit Begründung der Default-Werte und der
Konfigurations-Optionen). Eine eigenständige
`deploy/charts/k-deskflight/README.md` ist im Step-8-Plan vorgesehen
(Anwender-Doku für den Chart-Pfad).

**Problem:** zwei Stellen mit denselben Informationen (`values.yaml`-
Kommentare und Chart-`README.md`) divergieren über die Zeit. Anwender
lesen tendenziell die `README.md`, Maintainer ändern den `values.yaml`.
Drift ist unausweichlich.

[`helm-docs`](https://github.com/norwoodj/helm-docs) löst das: ein
Tool, das die `README.md` aus `values.yaml`-Kommentaren und
`Chart.yaml`-Metadaten generiert. Pre-Commit-Hook oder Make-Target
verhindert Drift.

Slice M8 §8 hat das explizit auf später verschoben: „Quality-of-Life,
verschoben auf M16 oder eigener Slice." Step-8-Diskussion
(2026-05-21) hat den M16-Pin als zu spät identifiziert; deshalb
dieser `open/`-Trigger.

---

## 2. Aktivierungs-Anlass

Aktivieren, wenn einer der folgenden Anlässe eintritt:

- **Chart-`README.md` und `values.yaml` divergieren** (erste
  Anwender-Beschwerde oder Maintainer-Drift bemerkt). Bis Step 8
  wird die `README.md` initial handgeschrieben; die Drift entsteht
  erst bei der zweiten Pflege-Aktion.
- **Chart-`values.yaml` wächst um mehr als 3–5 Top-Level-Slots**
  (z. B. mit Subcharts oder neuen `LH-PRI-002`-Features in v0.3).
  Manuelle Pflege wird dann unzuverlässig.
- **Vor erstem Chart-OCI-Publish nach `ghcr.io/pt9912/charts/`**
  als optionale Hygiene-Maßnahme — eine generierte `README.md` ist
  in der Helm-Hub-Ansicht konsistenter.

---

## 3. Zu entscheiden bei Aktivierung

- **Tool-Stage:** `helm-docs` ist ein Go-Binary; analog zur yq-
  Integration in `helm-tools` per Binary-Download oder als
  Container `jnorwood/helm-docs`. Pin-Hebung Routine ohne ADR.
- **Make-Target-Pattern:**
  - `make chart-docs` — generiert `deploy/charts/k-deskflight/README.md`
    aus `values.yaml`.
  - `make chart-docs-check` — verifiziert Drift via `git diff
    --exit-code` (analog `generated-drift-check`).
  - Letzteres in `make gates` einhängen, damit PRs mit
    Doku-Drift abgewiesen werden.
- **`values.yaml`-Kommentar-Konvention:** `helm-docs` liest
  spezielle Markup-Pragmas (`# -- description`,
  `# @default -- ...`). Die heutigen `values.yaml`-Kommentare
  müssen migriert werden — eine Stunde Arbeit.
- **Template-Customisierung:** `helm-docs` hat eine
  `README.md.gotmpl`-Vorlage, die wir an unseren Stil anpassen
  können (z. B. Bezug zu `LH-*`-Kennungen, Verweise auf
  `spec/architecture.md`).
- **Verhältnis zur handgeschriebenen Operations-Doku** unter
  `docs/user/`: `helm-docs` generiert nur die Chart-`README.md`;
  `docs/user/installation.md` bleibt handgepflegt. Beide ergänzen
  sich.

---

## 4. Nächste Schritte

1. Bei Aktivierung wandert dieser Eintrag nach `next/` (mit
   Scope-Skizze) oder direkt nach `in-progress/`.
2. Slice-Plan zieht die `helm-docs`-Wiring durch: Dockerfile-Stage
   (oder Pin in helm-tools), Makefile-Targets, `values.yaml`-
   Kommentar-Migration, `README.md.gotmpl`-Custom-Template,
   `chart-docs-check` in `make gates`.
3. Slice-Closure aktualisiert `slice-M8-helm-chart.md §8` und ggf.
   die v0.2-Roadmap.

---

## 5. Status

Offen. Niedrige Priorität — die `values.yaml`-Kommentare sind heute
ausreichend selbstdokumentierend; die `README.md` in
`deploy/charts/k-deskflight/` entsteht handgeschrieben mit Step 8
und ist bei Slice-M8-Closure noch synchron. Drift entsteht
frühestens mit dem ersten Maintainer-Edit nach der M8-Closure.
