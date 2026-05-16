# ADR 0005 — Helm-Chart nicht im MVP

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md)

---

## 1. Kontext

`LH-OP-006` führt die Entscheidung über das Helm Chart im MVP als
offenen Punkt. Drei Lastenheftaussagen betreffen den Punkt direkt:

- `LH-NF-016`: „Das Produkt soll per Helm Chart installierbar sein."
- `LH-SST-010`: „Das Produkt soll optional über ein Helm Repository
  installierbar sein."
- `LH-AK-015`: „Der Operator wird mit einer dokumentierten, minimalen
  RBAC-Konfiguration ausgeliefert (z. B. als Bestandteil des Helm
  Charts oder als Manifeste im Repository)."

`LH-NF-016` und `LH-SST-010` stehen **nicht** in `LH-PRI-001`
(MVP-Pflicht). `LH-AK-015` ist MVP-Pflicht, lässt aber Helm und raw
manifests gleichberechtigt zu. `LH-MVP-002` führt „Container-Image" und
„Beispielmanifest" als MVP-Bestandteile; Helm wird dort nicht genannt.

Die CRD `OpenDeskPreflightCheck` startet gemäß `LH-PROD-002` in
`v1alpha1`. Diese Versionsstufe lässt nach Kubernetes-Konvention
Schema-Brüche zwischen Releases zu — jeder Bruch zöge eine
Helm-Chart-Migration mit. Ein Chart, das auf eine noch instabile CRD
referenziert, würde die Pflegekosten unnötig in den MVP-Zyklus
hineinziehen.

---

## 2. Entscheidung

**Im MVP (v0.1, `LH-REL-001`) wird kein Helm Chart geliefert.**

Das MVP erfüllt `LH-AK-015` und `LH-MVP-002`-„Beispielmanifest" über
rohe Kubernetes-Manifeste im Repository. Der verbindliche Ablageort
entsteht mit dem Pflichtenheft (`LH-VM-002`); wahrscheinlicher
Kandidat: `deploy/manifests/`. Die Manifeste umfassen mindestens:

- die CRD `OpenDeskPreflightCheck` (`LH-F-001`),
- ServiceAccount, ClusterRole/Role und zugehörige Bindings für die in
  `LH-PRI-001` aktivierten Prüfungen (`LH-AK-015`),
- das Deployment des Operators (`LH-AK-002`),
- ein Beispiel-`OpenDeskPreflightCheck` gemäß `LH-PROD-003a`.

`LH-NF-016` und `LH-SST-010` werden mit **v0.2** eingelöst. `LH-PRI-002`
(Soll-Anforderungen für v0.2) wird im selben Commit um den
Helm-Chart-Scope erweitert, damit das v0.2-Versprechen nicht durch die
MVP-Entscheidung verloren geht. Die Distributionsform (traditionelles
Helm-Repository vs. OCI-Registry) ist nicht Gegenstand dieser ADR
(siehe §4).

---

## 3. Konsequenzen

- Das MVP-Repository hat keinen `Chart.yaml`, kein `values.yaml`, keinen
  `helm-lint`-/`chart-testing`-Quality-Gate. Der MVP-Build bleibt um
  diese Dimension schlanker und adressiert damit `LH-RISK-002` (Zu
  großer Projektumfang).
- `LH-AK-015` wird im MVP durch raw manifests erfüllt; die Manifeste
  bilden später die fachliche Vorlage für die Chart-Templates.
- `LH-NF-017` (GitOps-Kompatibilität) wird im MVP über raw manifests
  und ihre `kustomize`-Eignung erfüllt; vollständige Helm-Integration
  folgt ab v0.2.
- Anwender, die im MVP zwingend Helm benötigen, können das Manifest-Set
  über `kubectl apply -k` (Kustomize) oder eigene Wrapper installieren
  — innerhalb des Lastenheftrahmens.
- `LH-OP-006` wird im Lastenheft als geschlossen mit dieser ADR
  markiert (Formelhilfe aus `ADR 0002`).

---

## 4. Nicht Gegenstand dieser ADR

- **Konkreter Ablageort der raw manifests** (`deploy/manifests/` oder
  anderer Pfad) — entsteht mit dem Pflichtenheft (`LH-VM-002`).
- **Distributionsform des Helm Charts ab v0.2** (traditionelles
  Helm-Repository vs. OCI-Registry; eigene Domain vs. GHCR) — entsteht
  mit der v0.2-Roadmap und ggf. einer eigenen ADR.
- **Chart-Format-Details** (Subcharts, hooks, `values.schema.json`,
  Naming-Konventionen für Releases, Upgrade-Pfad zwischen Versionen)
  — Folgearbeit für die v0.2-Roadmap.
- **Kustomize-Overlays** als dauerhafte Alternative oder Ergänzung zu
  Helm ab v0.2 — eigener Entscheidungspunkt, falls Bedarf entsteht.
- **Test-Strategie für die raw manifests** im MVP (z. B. ein
  Pendant zu `scripts/validate-k8s-examples.sh` aus dem
  Schwesterprojekt `/Development/m-trace`) — operative Folgearbeit
  für das Pflichtenheft.
