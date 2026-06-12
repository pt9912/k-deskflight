# ADR 0016 — Doc-Refs-Linter via d-check (ersetzt das §2.10-Skript)

**Status:** Accepted
**Datum:** 2026-06-12
**Bezug:** [Lastenheft](../../../spec/lastenheft.md) (`LH-QG-008`),
[ADR 0012](0012-quality-gates.md) (§2.10),
[ADR 0002](0002-adr-lifecycle.md)

---

## 1. Kontext

[ADR 0012](0012-quality-gates.md) §2.10 führte
`scripts/verify-doc-refs.sh` als Adaption des m-trace-Skripts ein —
eine von zwölf funktional überlappenden Tool-Kopien dreier Familien
im Entwicklungs-Workspace. Diese Kopien sind inzwischen in das
konfigurierbare Tool [d-check](https://github.com/pt9912/d-check)
konsolidiert (Container-Image auf GHCR, Digest-Pin); die
Schwester-Repos (u. a. m-trace, der Ursprung der hiesigen Adaption)
sind migriert. Das Skript war zudem die einzige dokumentierte
Carveout-Stelle der Docker-only-Konvention (Host-Bash statt
Container, Makefile-Kopf).

## 2. Entscheidung

`make doc-refs` (`LH-QG-008`) läuft über das digest-gepinnte
d-check-Image (v0.2.0); die repo-spezifische Konfiguration liegt in
`.d-check.yml` (Module `links` + `anchors`; die Default-Scan-Wurzeln
docs/, spec/ und Root-`*.md` decken den §2.10-Geltungsbereich ab).
`scripts/verify-doc-refs.sh` ist gelöscht. Diese ADR ersetzt
ADR 0012 §2.10; die übrigen Abschnitte von ADR 0012 bleiben
unverändert gültig.

## 3. Begründung

- **Vergleichslauf** (2026-06-12, derselbe Repo-Stand): Alt-Skript
  0 Befunde; d-check fand 2 echte Mehr-Befunde (veraltete
  Roadmap-Anker — das Alt-Skript schnitt Fragment-Anker vor der
  Prüfung ab), behoben; danach beidseitig 0 Befunde bei 49 Dateien.
  Keine False-Positives.
- **Mehr-Abdeckung:** Heading-Anker-Validierung und Bildreferenzen
  (das Alt-Skript übersprang `![…](…)`); identische Semantik für
  absolute Pfade (Repo-Root-relativ).
- **Carveout-Auflösung:** Der Doc-Refs-Check läuft jetzt als
  Container — die Host-Bash-Ausnahme der Docker-only-Konvention
  entfällt.
- **Eine Pflege-Linie** statt Drift der Kopien
  (Schwesterprojekt-Konsistenz, vgl. ADR 0012 §„Folge-Trigger").

## 4. Konsequenzen

- Digest-Hebungen (neue d-check-Releases) sind Routine-Pins im
  Makefile mit Begründung im Commit-Body.
- Repo-spezifische Erweiterungen des Prüfumfangs erfolgen deklarativ
  in `.d-check.yml`, nicht mehr per Skript-Fork.
