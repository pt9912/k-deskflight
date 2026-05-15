# ADR 0001 — Dokumentations- und Planungsstruktur

**Status:** Accepted
**Datum:** 2026-05-15
**Bezug:** [Lastenheft](../../../spec/lastenheft.md)

---

## 1. Kontext

`k-deskflight` startet in der Anforderungsphase. Neben dem Lastenheft
braucht das Projekt eine stabile Dokumentationsstruktur für:

- normative Spezifikation,
- Architekturentscheidungen,
- Roadmap und Umsetzungspläne,
- offene Folgearbeiten und Trigger-Watch-Punkte,
- anwender- und betreibernahe Erklärungen sowie Runbooks,
- archivierte Ideenskizzen.

Die Struktur soll klein genug für den Projektstart bleiben, aber später
Meilensteine, weitere ADRs und Umsetzungsslices aufnehmen können. Sie
muss zudem den V-Modell-Anforderungen aus Lastenheft §19 (`LH-VM-001`
bis `LH-VM-006`) Rechnung tragen, insbesondere der Nachverfolgbarkeit
nach `LH-VM-006`.

---

## 2. Entscheidung

Die Dokumentation wird wie folgt organisiert:

| Pfad                              | Zweck                                                                                    |
| --------------------------------- | ---------------------------------------------------------------------------------------- |
| `spec/`                           | normative Produkt- und Architekturvorgaben (Lastenheft, später Pflichtenheft, Architektur) |
| `docs/plan/adr/`                  | Architecture Decision Records                                                            |
| `docs/plan/planning/open/`        | Trigger-Watch, offene Folgearbeiten und Vorabklärungen                                   |
| `docs/plan/planning/next/`        | konkret geplante, aber noch nicht aktive Arbeit (Scope-Skizze)                           |
| `docs/plan/planning/in-progress/` | aktive Roadmap und laufende Slice-Pläne                                                  |
| `docs/plan/planning/done/`        | abgeschlossene Pläne und Closure-Notizen                                                 |
| `docs/user/`                      | anwender- und betreibernahe Dokumentation                                                |
| `docs/archive/`                   | verworfene oder historische Ideenskizzen                                                 |

ADR-Dateinamen folgen dem Schema `NNNN-kurz-titel.md` (vierstellige
Nummer, fortlaufend).

Lebenszyklus eines Plan-Eintrags:
`open/` (Trigger entsteht) → `next/` (Scope skizziert) →
`in-progress/` (Slice-Plan aktiv) → `done/` (geliefert).
Wird ein Eintrag verworfen, wandert er nach `docs/archive/`.

---

## 3. Konsequenzen

- Das Lastenheft bleibt die Quelle für Anforderungen (`LH-*`-Kennungen).
- Eine spätere Architekturbeschreibung (`spec/architecture.md`) ergänzt
  das Lastenheft und führt eigene Kennungen für Architekturartefakte
  ein. Diese Datei existiert noch nicht; sie wird mit dem
  Pflichtenheft bzw. spätestens vor der ersten Implementierungs-Slice
  angelegt.
- ADRs dokumentieren **Entscheidungen**, nicht laufende Diskussionen.
  Offene fachliche Punkte aus Lastenheft §22 (`LH-OP-*`) wandern bei
  Entscheidung in eine ADR und werden im Lastenheft mit ADR-Verweis als
  geschlossen markiert.
- Roadmap-Dokumente in `in-progress/` verfolgen Status, Reihenfolge und
  Abnahmeschnitte. Sie liefern später die Meilenstein-Marker (`M1`,
  `M2`, …) für die Lastenheft-Traceability nach `LH-VM-006`.
- Offene Punkte werden nicht in abgeschlossenen Plänen versteckt,
  sondern unter `docs/plan/planning/open/` sichtbar gehalten.
- `docs/user/` ist explizit getrennt von Plänen; Runbooks und
  Bedienanleitungen sind keine Architekturartefakte.
- `docs/archive/` ist explizit getrennt von `done/`: archiviert =
  verworfen oder überholt; done = umgesetzt.

---

## 4. Pflege-Regeln

- Neue fachliche Anforderungen erhalten eine `LH-*`-Kennung im
  Lastenheft.
- Neue Architekturartefakte erhalten eine eigene Kennung in der noch
  anzulegenden `spec/architecture.md` und werden mit dem zugehörigen
  Lastenheft-Eintrag verknüpft.
- Neue technische Entscheidungen erhalten eine ADR, wenn sie
  langfristige Auswirkungen haben oder einen offenen Punkt (`LH-OP-*`)
  schließen.
- Jeder Plan in `in-progress/` muss Akzeptanzkriterien und einen
  Verifikationspfad enthalten.
- Abgeschlossene Pläne wandern nach `done/` mit kurzer Closure-Notiz
  (was wurde geliefert, was bleibt offen).
- Offene Trigger bleiben in `open/`, bis sie zu einem skizzierten Scope
  werden (→ `next/`), direkt aktiviert (→ `in-progress/`) oder verworfen
  (→ `archive/`) werden.
- Einträge in `next/` werden aktiviert (→ `in-progress/`), zurückgestuft
  (→ `open/`) oder verworfen (→ `archive/`).
- ADRs werden nach Erstellung nicht inhaltlich überschrieben; spätere
  Änderungen kommen als neue ADR mit Verweis auf den abgelösten
  Vorgänger.

---

## 5. Nicht Gegenstand dieser ADR

- Lebenszyklus-Modell für ADR-Statuswerte (`Proposed`, `Provisional`,
  `Accepted`, `Rejected`, `Withdrawn`, `Superseded`) — eigene Folge-ADR.
- Konvention für Querverweise zwischen Dokumenten (Kennungen vs.
  Abschnittsnummern) — eigene Folge-ADR.
- Konkrete Sprache und Build-System der Implementierung — siehe
  `LH-NF-001` (Go) und `LH-NF-016` (Helm).
- Konkrete Pfade für Test-Artefakte, Container-Images oder
  Release-Pipelines.
