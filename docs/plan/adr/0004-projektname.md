# ADR 0004 — Projektname `k-deskflight`

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md)

---

## 1. Kontext

`LH-OP-009` listet die Finalisierung des Projektnamens als offenen
Punkt. `LH-PROD-001` führt bereits `k-deskflight` als Produkt-,
Repository-, Container-Image- und Helm-Chart-Namen sowie
„OpenDesk Preflight Operator" als fachliche Produktbeschreibung. Der
Name wird zudem an mehreren weiteren Stellen des Lastenhefts konsistent
verwendet (u. a. Dokument-ID `LH-OPD-PFO-001`, `LH-PROD-002`).

Diese ADR formalisiert die im Lastenheft bereits konsistent verwendete
Wahl, schließt `LH-OP-009` und macht den Namen für künftige operative
Artefakte verbindlich (Wirkungspfad siehe `ADR 0002 §6`).

---

## 2. Entscheidung

Der Projektname lautet:

```text
k-deskflight
```

Er gilt einheitlich für die folgenden Artefakte:

| Artefakt                          | Wert                       |
| --------------------------------- | -------------------------- |
| Produktname                       | `k-deskflight`             |
| Repository-Name                   | `k-deskflight`             |
| Container-Image-Name (Kurzform)   | `k-deskflight`             |
| Helm-Chart-Name                   | `k-deskflight`             |
| Go-Modulname (Suffix)             | `k-deskflight`             |

Der Wert „Suffix" in der Zeile „Go-Modulname (Suffix)" bezeichnet den
letzten Pfadbestandteil des Modulnamens. Der vollständige Modulpfad
(`<host>/<owner>/k-deskflight`) ist nicht Gegenstand dieser ADR (siehe
§4) und wird mit dem Pflichtenheft (`LH-VM-002`) festgelegt.

Die fachliche Produktbeschreibung lautet:

```text
OpenDesk Preflight Operator
```

Sie beschreibt die Funktion des Operators und ist explizit **keine**
Aussage über eine offizielle Zugehörigkeit zum OpenDesk-Projekt.
Diese Klarstellung aus `LH-PROD-001` wird hiermit auf ADR-Ebene
bekräftigt.

---

## 3. Konsequenzen

- Repositorypfade, Image-Referenzen, Chart-Metadaten und Go-Modulnamen
  verwenden `k-deskflight` ohne Variantenschreibweisen (kein
  `KDeskFlight`, kein `k_deskflight`).
- Großschreibung und CamelCase erscheinen nur dort, wo Sprach- oder
  API-Konventionen es erzwingen (z. B. CRD-`Kind`
  `OpenDeskPreflightCheck` — siehe `LH-PROD-002` —, Go-Typnamen).
- Änderungen am Projektnamen erfolgen ausschließlich über eine
  ablösende ADR (`ADR 0002 §4`).
- `LH-OP-009` wird im Lastenheft §22 als geschlossen markiert, mit
  Verweis auf diese ADR (`ADR 0002 §7`).

---

## 4. Nicht Gegenstand dieser ADR

- **Wahl der DNS-Domain** für die Kubernetes-API-Gruppe und damit die
  endgültige API-Gruppe — siehe `LH-OP-002` und den Trigger-Eintrag
  [`docs/plan/planning/open/api-gruppe-domain.md`](../planning/open/api-gruppe-domain.md).
- **Go-Modulpfad** in voller Form (`<host>/<owner>/k-deskflight`) und
  **Hosting-Plattform** des Repositories — entsteht spätestens mit dem
  Pflichtenheft (`LH-VM-002`).
- **Container-Registry-Pfad** (`<registry>/<owner>/k-deskflight:<tag>`)
  und **Helm-Repository-URL** — operative Folgearbeit.
- **Namens- oder Markenklärung mit dem OpenDesk-Projekt** sowie
  generische Namensverfügbarkeit (Verfügbarkeit der GitHub-Organisation,
  Container-Registry-Namensraum, allgemeine Markenrecherche zu
  „deskflight"). Die deskriptive Nutzung des Begriffs „OpenDesk" in der
  fachlichen Produktbeschreibung bleibt von dieser ADR unberührt. Sollte
  sich Klärungsbedarf ergeben, wird er als eigener
  `planning/open/`-Eintrag oder ADR geführt.
