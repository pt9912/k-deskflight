# ADR 0002 — ADR-Lifecycle

**Status:** Accepted
**Datum:** 2026-05-15
**Bezug:** [ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[Lastenheft](../../../spec/lastenheft.md)
**Änderungstyp:** Ergänzung — `ADR 0001` bleibt inhaltlich gültig; diese
ADR fügt einen Lebenszyklus für ADR-Statuswerte hinzu, der in `ADR 0001`
nicht spezifiziert war.

---

## 1. Kontext

`ADR 0001 §3` und `§4` legen fest:

> ADRs dokumentieren **Entscheidungen**, nicht laufende Diskussionen.
> ADRs werden nach Erstellung nicht inhaltlich überschrieben; spätere
> Änderungen kommen als neue ADR.

Diese Regeln lassen drei Punkte offen, die ein konkreter ADR-Lifecycle
schließen muss:

- Strukturierte Entscheidungsfindung (z. B. ein Pre-Acceptance-Spike vor
  Verbindlichmachung) ist keine „laufende Diskussion", braucht aber
  einen eigenen Status zwischen Vorschlag und Beschluss.
- Eine akzeptierte ADR muss später als `Superseded` markierbar sein,
  ohne ihren Entscheidungstext inhaltlich umzuschreiben.
- `Rejected` und `Withdrawn` brauchen unterschiedliche Bedeutung:
  bewusste Negativentscheidung vs. Rückzug ohne Negativentscheidung.

Diese ADR schließt diese Lücken, ohne `ADR 0001` inhaltlich
umzuschreiben.

---

## 2. Entscheidung

ADRs in `docs/plan/adr/` durchlaufen den folgenden Lebenszyklus:

| Status        | Bedeutung | Wirkung auf abhängige Dokumente |
| ------------- | --------- | ------------------------------- |
| `Proposed`    | Empfehlung formuliert, **kein** Beschluss. Optionen und Bewertungskriterien sind dokumentiert. | Keine normative Wirkung. Lastenheft und spätere Architektur dürfen höchstens als Entwurfshinweis auf die ADR verweisen. |
| `Provisional` | Projektowner trägt die Empfehlung mit; ein begrenzter Validierungs-Spike läuft, dessen Vertrag in der ADR steht. | Eingeschränkte Wirkung. Abhängige Dokumente dürfen auf den laufenden Spike verweisen, aber keine `LH-OP-*`-Anforderung als geschlossen markieren. |
| `Accepted`    | Beschluss steht. Falls die ADR einen Validierungs-Spike enthielt, ist dieser nachweisbar grün abgeschlossen. | Volle Wirkung. Abhängige Dokumente werden gepflegt; offene Punkte dürfen als geschlossen markiert werden. |
| `Rejected`    | ADR wird nach Review, Spike oder Owner-Entscheid bewusst nicht übernommen. Die Negativentscheidung und ihre Begründung bleiben dauerhaft in der ADR. | Normative Schluss-Verweise werden entfernt; Folge-ADRs dürfen die Ablehnungsgründe referenzieren. |
| `Withdrawn`   | Autor oder Owner zieht den Vorschlag vor Beschluss zurück. Es liegt keine Negativentscheidung vor. | Laufende Hinweis-Verweise werden entfernt; der Vorschlag bleibt historisch sichtbar, bindet aber nicht. |
| `Superseded`  | ADR war akzeptiert, ist aber durch eine spätere ADR abgelöst. | Historisch. Die alte ADR bindet nicht mehr; abhängige Dokumente und operative Artefakte verweisen auf die Nachfolge-ADR. |

Erlaubte Übergänge:

```text
                Proposed
                  │  │
        ┌─────────┘  └─────────┐
        ▼                      ▼
    Provisional             Rejected / Withdrawn
        │
        ├──▶ Accepted ──▶ Superseded
        │
        └──▶ Rejected / Withdrawn
```

`Provisional` ist optional. Eine ADR ohne Validierungsbedarf darf direkt
`Proposed → Accepted` springen.

---

## 3. Verhältnis zu ADR 0001

- `ADR 0001 §3` („ADRs dokumentieren **Entscheidungen**, nicht laufende
  Diskussionen") wird so gelesen: **eine ADR im Status `Proposed`
  dokumentiert einen Entscheidungsvorschlag mit vollständigem
  Bewertungsrahmen und nimmt den Beschluss vorweg, ohne ihn zu
  treffen.** Sie ist keine offene Diskussion, sondern ein vorbereiteter
  Beschluss. Laufende Diskussionen, die noch keinen Vorschlag erlauben,
  gehören weiterhin nach `docs/plan/planning/open/`, nicht in `adr/`.
- `ADR 0001 §4` („ADRs werden nach Erstellung nicht inhaltlich
  überschrieben") gilt für **akzeptierte** Inhalte. Der Statuswechsel
  `Proposed → Provisional → Accepted` ist kein Inhaltsübergriff.
  Inhaltliche Verschärfungen oder Korrekturen vor `Accepted` (z. B. ein
  verschärfter Auflagenvertrag in einer Review-Runde) sind zulässig;
  sie werden im Header der ADR durch einen kurzen „Letzte inhaltliche
  Änderung"-Eintrag dokumentiert.
- Nach `Accepted` gilt das Änderungsverbot aus `ADR 0001 §4` strikt:
  jede Änderung kommt als neue ADR, die die vorhandene ablöst
  (`Superseded`).

---

## 4. Änderungsregeln

Nach `Accepted` ist der Entscheidungstext immutable. Fachliche
Änderungen kommen als neue ADR, die die bestehende ADR ablöst.

Zulässig bleiben nur Metadaten-Änderungen an der alten ADR:

- Statuswechsel auf `Superseded`,
- `Status geändert am`,
- `Superseded by`,
- ein kurzer Hinweis im Header, dass die ADR historisch ist.

Keine zulässige Metadaten-Änderung sind neue Begründungen, neue Regeln,
erweiterte Scope-Definitionen oder korrigierte Konsequenzen. Solche
Inhalte gehören in die Nachfolge-ADR.

---

## 5. Header-Schema

Jede ADR führt im Header:

- `Status`: Lifecycle-Status, optional mit kurzem Klartext-Zusatz.
- `Datum`: Erstellungsdatum der ADR.
- `Status geändert am`: Pflicht bei jedem Statuswechsel nach der
  Erstellung.
- `Letzte inhaltliche Änderung`: Pflicht bei inhaltlichen Änderungen
  vor `Accepted`; nach `Accepted` nur in der Nachfolge-ADR, nicht im
  abgelösten Entscheidungstext.
- `Superseded by`: Pflicht bei Status `Superseded`.

Das Feld `Letzte inhaltliche Änderung` ist kein Freibrief für
post-Acceptance-Korrekturen. Es dokumentiert nur erlaubte
Pre-Acceptance-Schärfungen.

---

## 6. Operative Artefakte

Die Status-Wirkung gilt für normative Dokumente (`spec/lastenheft.md`,
später `spec/architecture.md`) und für operative Artefakte (Makefiles,
Dockerfiles, CI-Jobs, Helm-Charts, Tool-Konfigurationen).

- Bei `Proposed` sind operative Artefakte nur als Spike- oder
  Prototyp-Artefakte erlaubt. Sie MÜSSEN als `Spike` oder `Prototyp` mit
  ADR-Verweis markiert sein und dürfen keinen beschlossenen Stack oder
  Prozess behaupten.
- Bei `Provisional` dürfen solche Artefakte den validierten Pfad des
  Spike-Vertrags bilden. Sie bleiben vorläufig und MÜSSEN bei
  `Rejected` oder `Withdrawn` entfernt, archiviert oder auf den
  Folgepfad umgestellt werden.
- Bei `Accepted` werden die Artefakte verbindliche Projektkonvention.
- Bei `Superseded` MÜSSEN betroffene Artefakte auf die Nachfolge-ADR
  umgestellt oder als historisch markiert werden. Sie dürfen nicht
  weiter eine abgelöste ADR als aktive Grundlage ausgeben.

---

## 7. Pflege-Regeln

`spec/lastenheft.md` (und später `spec/architecture.md`) nutzen
folgende Formelhilfe für ADR-Verweise:

- bei `Proposed`: kein Eintrag, höchstens „Entwurf in ADR XXXX".
- bei `Provisional`: „Vorgeschlagen, Spike laufend, siehe ADR XXXX".
- bei `Accepted`: „Geschlossen mit ADR XXXX".
- bei `Rejected`/`Withdrawn`: Eintrag entfernen.
- bei `Superseded`: Verweis auf Nachfolge-ADR.

---

## 8. Konsequenzen

- Künftige ADRs verwenden ausschließlich die hier definierten
  Statuswerte.
- `ADR 0001` bleibt inhaltlich unverändert; diese ADR ist Ergänzung,
  nicht Ablöser.

---

## 9. Nicht Gegenstand dieser ADR

- Review-Freigabeprozess für Statuswechsel.
- Automatisiertes Linting von ADR-Headern.
- Versionierung von ADR-Dateien (ADRs werden über `Superseded`-Ketten
  versioniert, nicht über Dateinamen-Suffixe).
