# Trigger — Domain für Kubernetes-API-Gruppe

**Trigger für:** `LH-OP-002` (Namensraum und API-Gruppe final entscheiden)
**Eröffnet:** 2026-05-16
**Bezug:** [Lastenheft `LH-OP-002`, `LH-PROD-002`, `LH-DAT-007`](../../../../spec/lastenheft.md),
[ADR 0001](../../adr/0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](../../adr/0002-adr-lifecycle.md)

---

## 1. Kontext

`LH-PROD-002` führt aktuell den Beispielwert
`preflight.k-deskflight.dev/v1alpha1` für die API-Gruppe. Kubernetes
folgt der Konvention, API-Gruppen als Reverse-DNS einer Domain zu
bilden, die der Projekt-Betreiber kontrolliert, damit es nicht zu
Konflikten mit anderen Operatoren kommt.

Die Domain `k-deskflight.dev` ist noch nicht gesichert. Bis diese
Klärung erfolgt, kann `LH-OP-002` nicht als `Accepted`-ADR
abgeschlossen werden: Eine akzeptierte ADR würde nach `ADR 0002 §6`
CRD-Manifeste, Go-Paketpfade, Helm-Werte und die initiale CRD-Version
auf den Wert festschreiben — ein späterer Domainwechsel wäre dann nur
über eine `v1alpha2`-Migration oder eine ablösende ADR möglich.

---

## 2. Zu entscheiden

- **Welche Basisdomain** trägt die API-Gruppe?
  - `k-deskflight.dev` registrieren?
  - alternativer Suffix (z. B. `.io`, `.org`, `.app`)?
  - Subdomain unter einer bereits kontrollierten Domain?
- **Wer kontrolliert/registriert** die Domain und auf welchen Halter?
- **Bis wann** muss die Entscheidung spätestens stehen?
  Spätester Zeitpunkt: vor dem ersten CRD-Manifest bzw. dem
  Pflichtenheft (`LH-VM-002`), da das Pflichtenheft die API-Gruppe
  als Architekturartefakt aufnimmt.

---

## 3. Optionen-Skizze

| Option | Vorteil | Risiko |
| ------ | ------- | ------ |
| `k-deskflight.dev` registrieren | passt zum Projektnamen, kurz, eindeutig | `.dev` ist Google-TLD mit HSTS-Preload — unkritisch für API-Gruppen, aber Registrierungs- und Verlängerungspflicht |
| `k-deskflight.io` registrieren | etabliertes Tech-TLD | höhere Kosten, Verfügbarkeit prüfen |
| Subdomain unter Halterdomain (z. B. `preflight.<halter>.example`) | keine zusätzliche Registrierung | API-Gruppe ist an Halter gebunden; bei Trägerwechsel kostet Migration einen API-Gruppenwechsel |
| `preflight.k-deskflight.dev` ohne Registrierung | passt zu LH-PROD-002 | Konventionsbruch (Domain nicht kontrolliert) — bei späterer Fremdregistrierung droht Konflikt |

Die Tabelle ist Diskussionsgrundlage, keine Vorentscheidung. Eine ADR
schließt `LH-OP-002` erst, wenn der Halter und der konkrete Wert
festliegen.

---

## 4. Nächste Schritte

1. Verfügbarkeit der Kandidatendomains prüfen (`whois`,
   Registrar-Verfügbarkeit).
2. Halterentscheidung mit dem Projektowner finalisieren.
3. ADR für `LH-OP-002` schreiben, sobald Domain und Halter feststehen.
4. Lastenheft `LH-PROD-002` ggf. anpassen, falls die Beispiel-Domain
   im Lastenheft sich vom finalen Wert unterscheidet.

---

## 5. Status

Offen. Wandert nach `next/`, sobald ein Halterkandidat steht, oder
direkt nach `in-progress/`, falls die Domainregistrierung ohne weitere
Vorklärung möglich ist. Wird mit Akzeptanz der zugehörigen ADR
abgeschlossen und in `done/` als Closure-Notiz dokumentiert.
