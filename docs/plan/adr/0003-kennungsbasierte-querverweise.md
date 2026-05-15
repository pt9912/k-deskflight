# ADR 0003 — Kennungsbasierte Querverweise

**Status:** Accepted
**Datum:** 2026-05-15
**Bezug:** [ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[Lastenheft](../../../spec/lastenheft.md)
**Änderungstyp:** Ergänzung — die Pflege-Regeln in `ADR 0001` werden
durch eine zusätzliche Konvention ergänzt; `ADR 0001` bleibt inhaltlich
unverändert.

---

## 1. Kontext

Das Lastenheft arbeitet bereits mit positionsunabhängigen
Kennungsräumen, u. a.:

- `LH-ZWE-*`, `LH-ZB-*`, `LH-AUS-*`, `LH-PK-*` — Zweck, Ziel, Kontext
- `LH-PROD-*` — Produktübersicht
- `LH-GL-*` — Glossar
- `LH-STK-*` — Stakeholder
- `LH-SYS-*`, `LH-BA-*` — Systemabgrenzung, Betriebsannahmen
- `LH-PROF-*` — Profile
- `LH-F-*` — funktionale Anforderungen
- `LH-NF-*` — nicht-funktionale Anforderungen
- `LH-SST-*` — Schnittstellen
- `LH-DAT-*` — Datenanforderungen
- `LH-SEC-*` — Sicherheitsanforderungen
- `LH-QA-*` — Qualitätsanforderungen
- `LH-AK-*` — Abnahmekriterien
- `LH-PRI-*`, `LH-MVP-*`, `LH-REL-*` — Priorisierung, MVP-Scope, Release
- `LH-VM-*` — V-Modell-Zuordnung
- `LH-OP-*` — offene Punkte
- `LH-RISK-*` — Risiken
- `LH-ERF-*` — Erfolgskriterien
- `LH-ZA-*`, `LH-UA-*` — Zielarchitektur und Benutzeranforderungen
- ADR-Nummern (`ADR 0001`, …)

Eine spätere `spec/architecture.md` wird einen analogen
Kennungsraum führen (vorgesehen: `LH-AR-*` oder eigene Präfixfamilie;
die konkrete Wahl entsteht mit der ersten Architekturversion).

Trotz der vorhandenen Kennungen besteht die Versuchung,
Querverweise wie „`lastenheft.md` §11" oder „§20 Traceability-Matrix"
zu nutzen. Solche Verweise sind:

- **positionsfragil:** Eine Umnummerierung der Abschnitte (z. B. beim
  Einfügen eines neuen Kapitels) bricht alle Verweise still.
- **semantisch arm:** „§11" sagt nicht, *was* gemeint ist; Leser müssen
  die Zielsektion lesen, um den Bezug zu verstehen.
- **renderer-abhängig:** Anker in Markdown sind nicht stabil
  (Slug-Bildung variiert zwischen Renderern und ändert sich bei
  Umbenennung der Überschrift).

Die etablierten Kennungen sind dagegen positionsunabhängig,
selbstbeschreibend und werden bereits in der Traceability-Matrix
(§20 Lastenheft) als stabile Referenz verwendet.

---

## 2. Entscheidung

Querverweise zwischen Spezifikations- und Planungsartefakten nutzen
**Kennungen als primäre Referenz**, nicht Abschnittsnummern.

### 2.1 Pflichtregel

Wenn das Referenzziel eine Kennung besitzt (`LH-*`, ADR-Nummer, später
`LH-AR-*` o. Ä.), MUSS die Kennung als Verweis verwendet werden. Eine
Abschnitts- oder Paragraphennummer ist als **Lesbarkeitshilfe in
Klammern** zulässig, trägt aber keine semantische Last.

Beispiele:

| Statt                                  | Besser                                                                              |
| -------------------------------------- | ----------------------------------------------------------------------------------- |
| „siehe `lastenheft.md` §11"            | „siehe `LH-F-001..035` (funktionale Anforderungen)"                                  |
| „`lastenheft.md` §20"                  | „`LH-VM-006` (Traceability-Matrix, §20)"                                              |
| „die Abnahmekriterien in §17"          | „`LH-AK-001..016`"                                                                   |
| „offene Punkte in §22"                 | „die konkret gemeinte Kennung, z. B. `LH-OP-002` (API-Gruppe)"                       |
| „MVP-Scope in §23"                     | „`LH-MVP-001..003` und `LH-PRI-001`"                                                  |

### 2.2 Wenn kein Kennungsraum existiert

Hat das Referenzziel keine etablierte Kennung, gilt folgende
Reihenfolge:

1. **Bevorzugt:** Eine Kennung im passenden Raum **anlegen**. Das ist
   im Rahmen der nächsten inhaltlichen Änderung des betroffenen
   Dokuments zu erledigen — nicht als eigener Big-Bang.
2. **Übergangsweise:** Inhaltliche Beschreibung plus Abschnittsnummer
   als Klammer-Hilfe (z. B. „die Begriffsdefinitionen in
   `lastenheft.md` §6"). Diese Form ist nur für Sektionen ohne
   Kennung erlaubt und nur übergangsweise, bis (1) erledigt ist.

### 2.3 Innerhalb desselben Dokuments

Innerhalb eines Dokuments gilt dieselbe Regel. Kennungen sind auch hier
gegenüber `§…`-Verweisen vorzuziehen. Die Klammer-Hilfe (z. B.
„`LH-F-024` (§11)") ist zulässig.

### 2.4 ADRs

ADR-zu-ADR-Verweise nutzen die ADR-Nummer (`ADR 0002`) als Kennung.
Bezugnahmen auf Unterabschnitte einer ADR werden inhaltlich benannt
(„Statuspfad in `ADR 0002`") statt positionsabhängig (`ADR 0002 §4`).
Kommen mehrere Verweise auf denselben Unterabschnitt vor, kann die
Ziel-ADR optional einen inhaltlichen Anker einführen (z. B.
`<!-- anchor:status-pfad -->` unmittelbar vor der Zielzeile). Diese
Anker-Konvention ist Konvention, kein Tooling-Vertrag, und nur dort
sinnvoll, wo der inhaltliche Name allein nicht eindeutig ist.

### 2.5 Externe Links

Markdown-Hyperlinks auf Dateien sind weiterhin erlaubt und erwünscht,
sofern sie die Kennung im Linktext führen:

```markdown
[`LH-OP-002`](../../../spec/lastenheft.md)
```

Der URL-Anker ist Konvention, nicht Vertrag — Verweis-Identität liegt
im Linktext.

---

## 3. Retrofit-Regel

Bestehende `§…`-Verweise werden nicht in einer Sammelaktion ersetzt.
Stattdessen gilt:

- Berührt eine Änderung ein Dokument inhaltlich, werden die in dieser
  Änderung sichtbaren `§…`-Verweise auf Kennungen umgestellt.
- `ADR 0001` und `ADR 0002` werden nicht nachträglich umgestellt — die
  Pflege-Regeln in `ADR 0001` verbieten inhaltliche Überschreibung
  akzeptierter ADRs.
- Das Lastenheft wird mit jeder inhaltlichen Änderung gemäß obiger
  Regel sukzessive umgestellt, spätestens jeweils bei der nächsten
  Minor-Versions-Hebung.

---

## 4. Konsequenzen

- Neue Spezifikations- und Planungsinhalte verwenden ausschließlich
  Kennungs-Verweise.
- Wenn beim Schreiben eines Verweises auffällt, dass die Zielsektion
  noch keine Kennung hat, wird die Kennung im selben Edit eingeführt
  (siehe Regel für Ziele ohne Kennung).
- Dokumentations-Tooling (z. B. ein möglicher `tools/check_refs` als
  Folgearbeit) kann später über die Kennungen einen Index erzeugen und
  nicht aufgelöste Verweise melden.
- Die Pflege-Regeln in `ADR 0001` bleiben unverändert; diese ADR fügt
  eine spezialisierte Pflege-Regel hinzu, überschreibt aber nichts.

---

## 5. Nicht Gegenstand dieser ADR

- Konkrete Linter-Implementierung für Querverweise (eigener Folge-ADR
  oder einfaches `tools/check_refs`-Skript als Routinearbeit).
- Renaming-Regeln für Kennungen (Kennungen sind unveränderlich, sobald
  veröffentlicht).
- HTML-/PDF-Rendering, Cross-Reference-Erzeugung in einem
  Dokumentations-Build.
