# Trigger — Aktivierung der externen Dienstprüfungen (v0.3+)

**Trigger für:** Folge-ADR zur Aktivierung von `LH-F-020` (PostgreSQL)
und `LH-F-021` (S3-kompatibles Object Storage).
**Eröffnet:** 2026-05-16
**Bezug:** [Lastenheft `LH-F-020`, `LH-F-021`, `LH-PRI-003`](../../../../spec/lastenheft.md),
[ADR 0010](../../adr/0010-externe-dienstpruefungen-und-secret-mechanik.md),
[ADR 0001](../../adr/0001-dokumentations-und-planungsstruktur.md)

---

## 1. Kontext

`ADR 0010` legt den Phasenplan für externe Dienstprüfungen fest:
ohne-Auth-Block (`LH-F-018`, `LH-F-019`, `LH-F-022`) kommt v0.2,
mit-Auth-Block (`LH-F-020`, `LH-F-021`) kommt v0.3+. Die konkrete
Aktivierung der mit-Auth-Prüfungen ist in `ADR 0010 §3` als „eigene
Folge-ADR" markiert und in `§4` als Nicht-Gegenstand der bestehenden
ADR ausgewiesen.

Ohne diesen Trigger-Eintrag bliebe die Folge-ADR im Cross-Verweisnetz
des Lastenhefts unsichtbar — `LH-OP-005` ist als geschlossen markiert,
und ohne Trigger gäbe es keinen Anker mehr für die noch zu treffende
Aktivierungs-Entscheidung.

---

## 2. Zu entscheiden in der Folge-ADR

- **Aktivierungs-Zeitpunkt**: Beginn der v0.3-Roadmap (`LH-REL-003`)
  oder spätere Version?
- **Optionale Erweiterungen der Auth-Methoden** über die in
  `ADR 0010 §2.5` festgelegte Minimal-Liste hinaus:
  - PostgreSQL: `cert` (Client-Cert-Auth)
  - S3: IRSA / Workload-Identity-Federation
  - S3: anonymer Read-Only-Modus für public Buckets
- **Pflichtenheft-Konkretisierungen**: konkretes CR-Spec-Schema für
  `spec.checks.externalServices.postgres` und `.objectStorage`,
  Timeouts, Retry-Budget, Probe-Implementierung.
- **Smoketest-Strategie** für mit-Auth-Pfade. Pendant zum m-trace-
  Pattern (`make smoke-<service>` mit aufgesetzten Test-Containern)
  ist Adaptionsvorlage; ob lokale Compose-Stacks für PostgreSQL/MinIO
  im Test-Lab leben oder reine Mocks reichen, entscheidet die
  Folge-ADR.
- **Re-Klassifizierung in `LH-PRI-*`**: `LH-F-020`/`LH-F-021` stehen
  aktuell in `LH-PRI-003` (Kann-Anforderungen). Mit der Aktivierung
  wären sie in einer v0.3-Soll-Liste (analog `LH-PRI-002` für v0.2)
  zu führen.

---

## 3. Nächste Schritte

1. Bei Beginn der v0.3-Roadmap (`docs/plan/planning/in-progress/`)
   wandert dieser Eintrag nach `next/` oder direkt nach
   `in-progress/`.
2. Folge-ADR schreiben, die `LH-F-020` und `LH-F-021` aktiviert. Sie
   zitiert `ADR 0010` als verbindliche Grundlage (Key-Konventionen
   §2.2, Conditions §2.3, TLS-Trust §2.4, Auth-Methoden §2.5,
   Output-Vertrag §2.6).
3. Bei Acceptance der Folge-ADR Lastenheft anpassen:
   `LH-F-020`/`LH-F-021` aus `LH-PRI-003` in eine v0.3-Soll-Liste
   überführen (entweder `LH-PRI-002` erweitern, falls die
   Phasenstaffelung sich verändert, oder einen neuen
   `LH-PRI-*`-Eintrag für v0.3 anlegen).

---

## 4. Status

Offen. Wandert nach `next/`, sobald die v0.3-Roadmap aktiviert wird;
direkt nach `in-progress/`, falls die Aktivierung ohne weitere
Vorklärung möglich ist; nach `docs/archive/` mit Closure-Notiz,
sobald die Folge-ADR akzeptiert ist (analog dem Pfad von
`docs/archive/api-gruppe-domain.md` für `LH-OP-002`).
