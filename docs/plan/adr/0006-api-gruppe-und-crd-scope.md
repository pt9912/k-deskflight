# ADR 0006 — API-Gruppe und CRD-Scope

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md)

---

## 1. Kontext

`LH-OP-002` forderte die finale Entscheidung über Namensraum und
Kubernetes-API-Gruppe der CRD `OpenDeskPreflightCheck`. Das Lastenheft
führte bis dato den Beispielwert `preflight.k-deskflight.dev`
(`LH-PROD-002`), ohne dass die Domain gesichert war. Der
Vorklärungs-Trigger `docs/plan/planning/open/api-gruppe-domain.md`
(mit Verabschiedung dieser ADR nach `docs/archive/api-gruppe-domain.md`
verschoben) hat die Optionen dokumentiert. `ADR 0002 §2` verbot eine
`Accepted`-ADR auf instabiler Grundlage — diese Sperre ist mit der
Domain-Klärung aufgehoben.

Zusätzlich klärt diese ADR den **CRD-Scope** (cluster-scoped vs.
namespaced), den `LH-PROD-002` und `LH-PROD-003a`/`-b` bisher nicht
festgelegt haben.

**Begleitende Lastenheft-Anpassungen** (im selben Commit):
`LH-PROD-002`, `LH-PROD-003a` und `LH-PROD-003b` werden auf den hier
beschlossenen Stand gehoben. Die dortigen Beispielwerte
(`preflight.k-deskflight.dev`) waren Platzhalter ohne gesicherte
Domain.

---

## 2. Entscheidung

### 2.1 Halter-Domain

Halter-Domain für die API-Gruppe ist **`geo-terrain.net`**. Die Domain
wird vom Projektowner (`LH-PROD-001`-Autor) kontrolliert.

### 2.2 API-Gruppe

Die Kubernetes-API-Gruppe ist:

```text
k-deskflight.geo-terrain.net
```

Pattern: `<projekt>.<owner-domain>`. Alle CRDs des Projekts —
einschließlich der in `LH-ZB-004` vorgesehenen Erweiterungen
(`OpenDeskTenant`, `OpenDeskMaintenanceWindow`, `OpenDeskBackupPolicy`,
`OpenDeskModuleHealth`, `OpenDeskUpgradePlan`) — teilen sich diese
Gruppe.

### 2.3 API-Version

Die initiale API-Version ist `v1alpha1`. Damit gelten die
Kubernetes-Konventionen für Alpha-Versionen: Schema-Brüche zwischen
Releases sind zulässig (vgl. `ADR 0005` zur MVP-Begründung).

### 2.4 CRD-Kind und Vollreferenz

Der CRD-Kind ist `OpenDeskPreflightCheck` (`LH-PROD-002`). Die
vollständige API-Referenz für die MVP-CR lautet:

```text
apiVersion: k-deskflight.geo-terrain.net/v1alpha1
kind: OpenDeskPreflightCheck
```

### 2.5 CRD-Scope

Die CRD `OpenDeskPreflightCheck` ist **namespaced**. Begründung:

- `LH-DAT-007` verlangt `secretRef`-Referenzen auf Secrets im selben
  Namespace; ein namespaced-Scope macht diese Auflösung natürlich und
  vermeidet die unschöne Pflicht, im Spec einen Namespace zusätzlich
  zum Secret-Namen anzugeben.
- `LH-BA-004` erlaubt mehrere CRs parallel; je Namespace lassen sich
  Verantwortungsbereiche (z. B. Plattform-Team vs. OpenDesk-Team)
  abgrenzen.
- Die cluster-weiten Prüfgegenstände (StorageClass, IngressClass,
  cert-manager) werden vom Operator ausgelesen — der CR-Scope hat
  darauf keinen Einfluss; die Prüfungen funktionieren unabhängig
  davon, wo die CR liegt.
- RBAC ist mit namespaced-CRs feiner steuerbar (Read/Write am
  CR-Objekt pro Namespace).

---

## 3. Konsequenzen

- `LH-PROD-002`, `LH-PROD-003a` und `LH-PROD-003b` werden im selben
  Commit auf `k-deskflight.geo-terrain.net/v1alpha1` aktualisiert.
- `LH-OP-002` wird im Lastenheft als geschlossen mit dieser ADR
  markiert (Formelhilfe aus `ADR 0002`).
- Der Vorklärungs-Trigger wandert aus `docs/plan/planning/open/` nach
  `docs/archive/` mit einer Closure-Notiz (überholt, weil entschieden,
  `ADR 0001 §3`). `done/` bleibt für abgeschlossene Pläne und Slices
  reserviert; Trigger-Einträge gehören dort nicht hin.
- Künftige k-deskflight-CRDs nutzen dieselbe API-Gruppe. Ein Wechsel
  der Halter-Domain würde alle gleichzeitig brechen; das ist ein
  bewusst akzeptierter Tradeoff der Wahl A1 (Owner-Domain-Pattern)
  über A2 (eigene Service-Subdomain je CRD).
- Domain-Verlängerungspflicht für `geo-terrain.net` ist operative
  Verantwortung des Owners. Ein Verlust der Domain würde alle
  k-deskflight-CRDs invalidieren — analog zu `LH-RISK-006`
  (Providerabhängigkeit), hier auf Domain-Ebene.

---

## 4. Nicht Gegenstand dieser ADR

- **Konkrete Schema-Felder der CRD `OpenDeskPreflightCheck`** — die
  Lastenheft-Beispiele `LH-PROD-003a/b` skizzieren die Felder, das
  vollständige Schema entsteht mit dem Pflichtenheft (`LH-VM-002`).
- **CRD-Conversion-Webhooks** und **Upgrade-Pfad zu späteren
  API-Versionen** (`v1alpha2`, `v1beta1`, `v1`) — spätere ADRs zum
  Zeitpunkt der jeweiligen Versionshebung.
- **DNS-Zone-Konfiguration für `*.k-deskflight.geo-terrain.net`** —
  diese ADR bindet nur den Reverse-DNS-String der API-Gruppe; ob unter
  der Domain tatsächlich Web-Inhalte ausgeliefert werden (z. B. ein
  Dokumentations-Subdomain `docs.k-deskflight.geo-terrain.net`) ist
  operative Folgearbeit.
- **Container-Image-Registry-Pfad** und **Go-Modul-Hostname** — nicht
  Gegenstand dieser ADR (siehe `ADR 0004 §4`).
- **Schwester-API-Gruppen für gänzlich andere Projekte unter
  `geo-terrain.net`** — die Wahl bindet `k-deskflight.geo-terrain.net`,
  nicht den restlichen Domain-Namensraum.
