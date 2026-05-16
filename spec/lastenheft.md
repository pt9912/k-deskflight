# Lastenheft: k-deskflight (OpenDesk Preflight Operator)

**Dokument-ID:** LH-OPD-PFO-001  
**Projektname / Repository:** k-deskflight  
**Produktname:** k-deskflight  
**Fachliche Produktbeschreibung:** OpenDesk Preflight Operator  
**Artefakt:** Lastenheft  
**Zielbild:** Kubernetes-native Vorabprüfung für OpenDesk-Bereitstellungen  
**Version:** 0.1.0  
**Status:** Entwurf  
**Autor:** Dietmar Burkard  
**Lizenzziel:** MIT  
**Sprache des Lastenhefts:** Deutsch  
**Sprache der Projektartefakte:** siehe LH-NF-021  

---

## 1. Zweck des Dokuments

### LH-ZWE-001 — Zweck des Lastenhefts

Dieses Lastenheft beschreibt die Anforderungen an einen Kubernetes Operator, der vor einer Installation oder Aktualisierung von OpenDesk prüft, ob ein Kubernetes-Cluster die notwendigen Voraussetzungen erfüllt.

Das Dokument beschreibt **was** erreicht werden soll, nicht **wie** es technisch umgesetzt wird. Die konkrete technische Umsetzung ist Gegenstand eines späteren Pflichtenhefts.

---

## 2. Zielbild

### LH-ZB-001 — Übergeordnetes Ziel

Es soll ein Open-Source-Projekt entstehen, das Kubernetes-Betreiber bei der Vorbereitung, Prüfung und Bewertung einer OpenDesk-Installation unterstützt.

Der Operator soll als Kubernetes-native Erweiterung bereitgestellt werden und über Custom Resource Definitions steuerbar sein.

### LH-ZB-002 — Zentrales Produktziel

Das Zielprodukt soll eine Custom Resource Definition (CRD) mit dem Kind `OpenDeskPreflightCheck` bereitstellen. Über konkrete Custom Resources dieses Kinds sollen Betreiber definieren können, welche Voraussetzungen für eine OpenDesk-Installation geprüft werden sollen.

Der Operator soll die Prüfungen ausführen und das Ergebnis strukturiert im Status der Custom Resource bereitstellen.

### LH-ZB-003 — Nutzenversprechen

Das Produkt soll Installationsfehler, Fehlkonfigurationen und unvollständige Cluster-Voraussetzungen frühzeitig sichtbar machen.

Der Operator soll insbesondere helfen bei:

- Vorbereitung von OpenDesk-Testumgebungen
- Vorbereitung produktiver OpenDesk-Installationen
- Dokumentation des Cluster-Zustands
- wiederholbarer Betriebsprüfung
- Automatisierung in GitOps- und CI/CD-Prozessen
- Unterstützung von Self-Hosting- und Plattformteams

### LH-ZB-004 — Strategisches Ziel

Das Projekt soll langfristig als Grundlage für eine Kubernetes-native OpenDesk-Betriebsunterstützung dienen.

Mögliche spätere Erweiterungen sind:

- `OpenDeskTenant`
- `OpenDeskMaintenanceWindow`
- `OpenDeskBackupPolicy`
- `OpenDeskModuleHealth`
- `OpenDeskUpgradePlan`

Diese Erweiterungen sind nicht Bestandteil des initialen Zielbilds, sollen aber durch eine saubere Architektur vorbereitet werden.

---

## 3. Ausgangssituation

### LH-AUS-001 — Problemstellung

OpenDesk besteht aus mehreren Komponenten und setzt eine geeignete Kubernetes-Plattform voraus. Betreiber müssen vor Installation und Betrieb sicherstellen, dass zentrale Cluster-Funktionen vorhanden und korrekt konfiguriert sind.

Typische Fehlerquellen sind:

- fehlende oder ungeeignete StorageClasses
- fehlende IngressClass
- fehlender oder falsch konfigurierter Zertifikatsmanager
- unklare Ressourcenverfügbarkeit
- fehlende DNS- oder TLS-Voraussetzungen
- unvollständige externe Dienste wie PostgreSQL oder Object Storage
- fehlende Netzwerk- oder Sicherheitsvoraussetzungen
- unklare Produktionsreife der Zielumgebung

### LH-AUS-002 — Motivation

Die Prüfung dieser Voraussetzungen erfolgt häufig manuell, dokumentationsbasiert oder durch Trial-and-Error während der Installation.

Das ist fehleranfällig und schlecht automatisierbar.

Der OpenDesk Preflight Operator soll diese Prüfungen deklarativ, wiederholbar und clusterintern verfügbar machen.

---

## 4. Produktkontext

### LH-PK-001 — Zielplattform

Das Produkt soll auf Kubernetes-Clustern betrieben werden.

### LH-PK-002 — Betriebsmodell

Das Produkt soll als Kubernetes Operator bereitgestellt werden.

### LH-PK-003 — Steuerung

Die Steuerung soll über Kubernetes Custom Resources erfolgen.

### LH-PK-004 — Zielnutzer

Zielnutzer sind:

- Kubernetes-Administratoren
- Plattformteams
- DevOps-Teams
- SRE-Teams
- OpenDesk-Betreiber
- Behördennahe IT-Dienstleister
- Self-Hosting-Betreiber mit Kubernetes-Erfahrung

### LH-PK-005 — Open-Source-Ausrichtung

Das Produkt soll als Open-Source-Projekt veröffentlicht werden.

Das Projekt soll für externe Mitwirkende verständlich, testbar und nachvollziehbar aufgebaut sein.

---

## 5. Produktübersicht

### LH-PROD-001 — Produktname

Der Produktname lautet:

```text
k-deskflight
```

Die fachliche Produktbeschreibung lautet:

```text
OpenDesk Preflight Operator
```

Projektname, Repository-Name, Container-Image-Name und Helm-Chart-Name sollen einheitlich `k-deskflight` verwenden. Der Begriff „OpenDesk Preflight Operator" beschreibt ausschließlich die fachliche Funktion und stellt keine offizielle Zugehörigkeit zum OpenDesk-Projekt dar.

### LH-PROD-002 — Hauptressource

Die zentrale Kubernetes-Ressource soll heißen:

```yaml
apiVersion: k-deskflight.geo-terrain.net/v1alpha1
kind: OpenDeskPreflightCheck
```

API-Gruppe, -Version, CRD-Kind und CRD-Scope (namespaced) sind in
`ADR 0006` final entschieden. Die Halter-Domain `geo-terrain.net` wird
vom Projektowner kontrolliert.

### LH-PROD-003a — MVP-Beispiel

Das folgende Beispiel deckt ausschließlich Prüfungen ab, die im MVP (LH-PRI-001, LH-MVP-002) enthalten sind.

```yaml
apiVersion: k-deskflight.geo-terrain.net/v1alpha1
kind: OpenDeskPreflightCheck
metadata:
  name: cluster-readiness
spec:
  profile: production

  checks:
    kubernetesVersion:
      min: "1.34"

    ingress:
      required: true
      className: nginx

    certManager:
      required: true

    storage:
      requiredClasses:
        - default
        - backup

    resources:
      minCpu: "16"
      minMemory: "64Gi"
```

### LH-PROD-003b — Zielbild-Beispiel (spätere Ausbaustufen)

Das folgende Beispiel zeigt das vollständige Zielbild inklusive Prüfungen, die in späteren Versionen vorgesehen sind (LH-PRI-002, LH-PRI-003).

Das Feld `domain` benennt den primären DNS-Namen der OpenDesk-Installation und dient als Eingabe für DNS- und TLS-Prüfungen (LH-F-018, LH-F-019).

Zugangsdaten externer Dienste werden nicht direkt im Spec abgelegt, sondern über `secretRef` auf bestehende Kubernetes-Secrets referenziert (siehe LH-DAT-007).

```yaml
apiVersion: k-deskflight.geo-terrain.net/v1alpha1
kind: OpenDeskPreflightCheck
metadata:
  name: cluster-readiness
spec:
  profile: production
  domain: example.org

  checks:
    kubernetesVersion:
      min: "1.34"

    ingress:
      required: true
      className: nginx

    certManager:
      required: true
      clusterIssuers:
        - letsencrypt-prod

    storage:
      requiredClasses:
        - default
        - backup

    resources:
      minCpu: "16"
      minMemory: "64Gi"

    externalServices:
      postgres:
        required: true
        endpoint: postgres.example.org:5432
        credentialsSecretRef:
          name: postgres-preflight-credentials
      objectStorage:
        required: true
        endpoint: https://s3.example.org
        credentialsSecretRef:
          name: s3-preflight-credentials
```

### LH-PROD-004 — Ergebnisdarstellung

Die Ergebnisse der Prüfungen sollen im `status` der Custom Resource dargestellt werden.

Beispiel:

```yaml
status:
  phase: Failed
  summary:
    passed: 5
    warning: 1
    failed: 2
  conditions:
    - type: KubernetesVersionReady
      status: "True"
      reason: VersionSupported
      message: "Kubernetes version satisfies the configured minimum version."
    - type: StorageReady
      status: "False"
      reason: StorageClassMissing
      message: "Required storage class 'backup' was not found."
```

---

## 6. Begriffe und Abkürzungen

| Kennung   | Begriff         | Beschreibung                                                                                |
| --------- | --------------- | ------------------------------------------------------------------------------------------- |
| LH-GL-001 | CRD             | Custom Resource Definition in Kubernetes                                                    |
| LH-GL-002 | Custom Resource | Konkretes Kubernetes-Objekt einer CRD                                                       |
| LH-GL-003 | Operator        | Kubernetes-Controller mit domänenspezifischer Betriebslogik                                 |
| LH-GL-004 | Preflight Check | Vorabprüfung vor Installation, Update oder Betrieb                                          |
| LH-GL-005 | OpenDesk        | Zielplattform, deren Betriebsumgebung geprüft werden soll                                   |
| LH-GL-006 | Condition       | Strukturierte Statusaussage innerhalb einer Kubernetes-Ressource                            |
| LH-GL-007 | Profile         | Vordefinierter Satz von Prüfanforderungen                                                   |
| LH-GL-008 | GitOps          | Deklaratives Betriebsmodell über versionierte Kubernetes-Manifeste                          |
| LH-GL-009 | RBAC            | Role-Based Access Control — rollenbasierte Zugriffssteuerung in Kubernetes                  |
| LH-GL-010 | DNS             | Domain Name System — Namensauflösung im Netzwerk                                            |
| LH-GL-011 | TLS             | Transport Layer Security — Verschlüsselungsprotokoll für Netzwerkkommunikation              |
| LH-GL-012 | MVP             | Minimum Viable Product — kleinster funktionsfähiger Produktumfang                           |
| LH-GL-013 | CI/CD           | Continuous Integration / Continuous Delivery — automatisierte Build- und Auslieferungskette |
| LH-GL-014 | SRE             | Site Reliability Engineering — Disziplin für zuverlässigen Betrieb verteilter Systeme       |
| LH-GL-015 | S3              | Object-Storage-API von Amazon, etabliertes Quasi-Standard-Protokoll                         |
| LH-GL-016 | SCS             | Sovereign Cloud Stack — offene Referenzplattform für souveräne Cloud-Infrastruktur          |
| LH-GL-017 | k3s             | Leichtgewichtige Kubernetes-Distribution                                                    |
| LH-GL-018 | airgapped       | Betriebsumgebung ohne direkte Verbindung zu öffentlichen Netzen                             |
| LH-GL-019 | ClusterIssuer   | cert-manager-Ressource zur clusterweiten Zertifikatsausstellung                             |
| LH-GL-020 | secretRef       | Verweis innerhalb einer Custom Resource auf ein bestehendes Kubernetes-Secret               |

---

## 7. Stakeholder

| Kennung    | Stakeholder              | Interesse                                            |
| ---------- | ------------------------ | ---------------------------------------------------- |
| LH-STK-001 | Kubernetes-Administrator | Sichere Bewertung der Cluster-Eignung                |
| LH-STK-002 | Plattformteam            | Wiederholbare Prüfung mehrerer Cluster               |
| LH-STK-003 | OpenDesk-Betreiber       | Reduzierung von Installationsfehlern                 |
| LH-STK-004 | SRE-Team                 | Frühe Erkennung von Betriebsrisiken                  |
| LH-STK-005 | Security-Team            | Sichtbarkeit sicherheitsrelevanter Voraussetzungen   |
| LH-STK-006 | Open-Source-Mitwirkende  | Verständliche Architektur und klare Einstiegspunkte  |
| LH-STK-007 | Projektverantwortliche   | Nachvollziehbare Entscheidungs- und Abnahmekriterien |

---

## 8. Systemabgrenzung

### LH-SYS-001 — Enthaltene Systemfunktion

Das System soll Kubernetes-Cluster-Voraussetzungen für OpenDesk prüfen und dokumentieren.

### LH-SYS-002 — Nicht enthaltene Systemfunktion

Das System soll OpenDesk nicht installieren.

### LH-SYS-003 — Nicht enthaltene Systemfunktion

Das System soll OpenDesk nicht konfigurieren.

### LH-SYS-004 — Nicht enthaltene Systemfunktion

Das System soll keine produktiven OpenDesk-Komponenten verändern.

### LH-SYS-005 — Nicht enthaltene Systemfunktion

Das System soll keine automatischen Reparaturen ohne explizite spätere Erweiterung durchführen.

### LH-SYS-006 — Nicht enthaltene Systemfunktion

Das System soll keine Geheimnisse wie Passwörter, Tokens oder private Schlüssel erzeugen oder speichern.

---

## 9. Betriebsannahmen

### LH-BA-001 — Kubernetes-Zugriff

Der Operator soll innerhalb eines Kubernetes-Clusters mit geeigneten RBAC-Rechten betrieben werden.

### LH-BA-002 — Lesender Standardbetrieb

Der initiale Standardbetrieb soll überwiegend lesend erfolgen.

### LH-BA-003 — Namespace

Der Operator soll in einem eigenen Namespace betrieben werden können.

### LH-BA-004 — Mehrere Checks

Es soll möglich sein, mehrere `OpenDeskPreflightCheck`-Ressourcen parallel zu betreiben.

### LH-BA-005 — Wiederholbarkeit

Prüfungen sollen wiederholt ausgeführt werden können.

### LH-BA-006 — GitOps-Eignung

Die Ressourcen sollen deklarativ über GitOps-Werkzeuge verwaltbar sein.

---

## 10. Profile

### LH-PROF-001 — Unterstützte Profile

Das Produkt soll vordefinierte Profile unterstützen.

Mindestens folgende Profile sind vorgesehen:

- `evaluation`
- `production`

Spätere Profile können sein:

- `k3s`
- `scs`
- `airgapped`
- `custom`

### LH-PROF-002 — Profil `evaluation`

Das Profil `evaluation` soll für Test- und Evaluierungsumgebungen gedacht sein.

Es soll weniger strenge Anforderungen prüfen als das Produktionsprofil.

Die K8s-Mindestversion-Vorbelegung folgt dem Operator-Floor gemäß `ADR 0009` (heute 1.34, rollend mit der Operator-Support-Matrix). Anwender können den Wert pro CR via `spec.checks.kubernetesVersion.min` überschreiben.

### LH-PROF-003 — Profil `production`

Das Profil `production` soll für produktionsnahe oder produktive Umgebungen gedacht sein.

Es soll strengere Anforderungen prüfen, insbesondere bezüglich Ressourcen, Storage, TLS, Ingress und externen Diensten.

Die K8s-Mindestversion-Vorbelegung ist identisch mit `evaluation` (Operator-Floor gemäß `ADR 0009`) — Profile-Differenzierung erfolgt nicht über die K8s-Version. Die OpenDesk-Doku-Untergrenze `≥ v1.24` ist niedriger als der Operator-Floor und wird in `ADR 0009 §2.3` eingeordnet.

### LH-PROF-004 — Modus `custom` (spätere Version)

Der Wert `custom` im Feld `profile` steht für vollständig benutzerdefinierte Prüfkonfigurationen. Er ist kein vordefinierter Prüfstandard, sondern markiert den Modus, in dem alle Prüfungen ausschließlich aus den Angaben der Custom Resource abgeleitet werden, ohne dass profil-spezifische Defaults angewendet werden.

Dieser Modus ist nicht Bestandteil des initialen Profilumfangs (siehe LH-PROF-001) und wird mit einer späteren Version eingeführt.

---

## 11. Funktionale Anforderungen

### LH-F-001 — Bereitstellung einer CRD

Das System muss eine Kubernetes Custom Resource Definition für `OpenDeskPreflightCheck` bereitstellen.

### LH-F-002 — Anlegen einer Preflight-Ressource

Ein Benutzer muss eine `OpenDeskPreflightCheck`-Ressource im Cluster anlegen können.

### LH-F-003 — Verarbeitung durch Controller

Der Operator muss angelegte `OpenDeskPreflightCheck`-Ressourcen erkennen und verarbeiten.

### LH-F-004 — Statusaktualisierung

Der Operator muss den Status einer `OpenDeskPreflightCheck`-Ressource aktualisieren.

### LH-F-005 — Conditions

Der Operator muss Prüfergebnisse als Kubernetes Conditions darstellen.

### LH-F-006 — Gesamtphase

Der Operator muss eine Gesamtphase für die Prüfung bereitstellen.

Zulässige Phasen sollen mindestens sein:

- `Pending`
- `Running`
- `Passed`
- `Warning`
- `Failed`
- `Unknown`

### LH-F-007 — Zusammenfassung

Der Operator soll eine Zusammenfassung der Prüfergebnisse bereitstellen.

Die Zusammenfassung soll mindestens enthalten:

- Anzahl erfolgreicher Prüfungen
- Anzahl fehlgeschlagener Prüfungen
- Anzahl von Warnungen
- Zeitpunkt der letzten Prüfung

### LH-F-008 — Kubernetes-Version prüfen

Der Operator muss prüfen können, ob die Kubernetes-Version des Clusters eine konfigurierte Mindestversion erfüllt.

### LH-F-009 — Kubernetes-API-Erreichbarkeit prüfen

Der Operator muss prüfen können, ob die Kubernetes-API aus Sicht des Operators erreichbar ist.

### LH-F-010 — StorageClass prüfen

Der Operator muss prüfen können, ob konfigurierte StorageClasses im Cluster vorhanden sind.

### LH-F-011 — Default StorageClass erkennen

Der Operator soll erkennen können, ob eine Default StorageClass vorhanden ist.

### LH-F-012 — IngressClass prüfen

Der Operator muss prüfen können, ob eine konfigurierte IngressClass vorhanden ist.

### LH-F-013 — cert-manager prüfen

Der Operator soll prüfen können, ob cert-manager im Cluster vorhanden ist.

### LH-F-014 — ClusterIssuer prüfen

Der Operator soll prüfen können, ob konfigurierte ClusterIssuer vorhanden sind.

### LH-F-015 — Ressourcen prüfen

Der Operator soll prüfen können, ob der Cluster über eine konfigurierte Mindestmenge an CPU und Arbeitsspeicher verfügt.

### LH-F-016 — Node-Anzahl prüfen

Der Operator soll prüfen können, ob eine konfigurierte Mindestanzahl an Nodes vorhanden ist.

### LH-F-017 — Node-Zustand prüfen

Der Operator soll prüfen können, ob Nodes im Zustand `Ready` sind.

### LH-F-018 — DNS-Prüfung

Der Operator soll prüfen können, ob konfigurierte DNS-Namen auflösbar sind.

### LH-F-019 — TLS-Prüfung

Der Operator soll prüfen können, ob konfigurierte HTTPS-Endpunkte gültige TLS-Zertifikate bereitstellen.

### LH-F-020 — PostgreSQL-Erreichbarkeit prüfen

Der Operator soll prüfen können, ob ein konfigurierter PostgreSQL-Endpunkt erreichbar ist.

### LH-F-021 — Object-Storage-Erreichbarkeit prüfen

Der Operator soll prüfen können, ob ein konfigurierter S3-kompatibler Object-Storage-Endpunkt erreichbar ist.

### LH-F-022 — Netzwerkzugriff prüfen

Der Operator soll prüfen können, ob aus dem Cluster heraus Netzwerkverbindungen zu konfigurierten Endpunkten möglich sind.

### LH-F-023 — Namespace-Prüfung

Der Operator soll prüfen können, ob benötigte Namespaces vorhanden sind.

### LH-F-024 — RBAC-Prüfung

Der Operator soll prüfen können, ob eigene Berechtigungen zur Durchführung der aktivierten Prüfungen ausreichen.

Die Prüfung soll auf Basis von Kubernetes-Mechanismen wie `SelfSubjectAccessReview` oder `SelfSubjectRulesReview` erfolgen.

Reichen die eigenen Rechte zur Durchführung einer aktivierten Prüfung nicht aus, soll dies über eine Condition vom Typ `RBACInsufficient` signalisiert werden. Die betroffene Einzelprüfung soll als nicht durchführbar markiert werden (Status `Unknown`), ohne die Gesamtausführung abzubrechen.

### LH-F-025 — Wiederholintervall

Ein Benutzer soll ein Wiederholintervall für Prüfungen konfigurieren können.

### LH-F-026 — Manuelle erneute Prüfung

Ein Benutzer soll eine erneute Prüfung durch Änderung der Custom Resource auslösen können.

### LH-F-027 — Events

Der Operator soll Kubernetes Events für wichtige Zustandsänderungen erzeugen. Format und Phasenstaffelung sind mit `ADR 0008` festgelegt (Plain-Text gemäß Kubernetes-Event-Konvention; v0.2-Soll-Anforderung gemäß `LH-PRI-002`).

### LH-F-028 — Report als ConfigMap

Der Operator kann optional einen Report als ConfigMap erzeugen. Das Layout ist mit `ADR 0008` festgelegt: zwei Daten-Keys — `report.yaml` (maschinenlesbar, erfüllt LH-F-029) und `report.md` (menschenlesbar, erfüllt LH-DAT-004). Naming und Lifecycle ebenfalls in `ADR 0008`.

### LH-F-029 — Exportierbare Ergebnisse

Die Prüfergebnisse sollen so strukturiert sein, dass sie von CI/CD- oder GitOps-Werkzeugen ausgewertet werden können.

### LH-F-030 — Deaktivierbare Prüfungen

Ein Benutzer soll einzelne Prüfungen aktivieren oder deaktivieren können.

### LH-F-031 — Schweregrad

Prüfungen sollen mit einem Schweregrad bewertet werden können.

Mögliche Schweregrade:

- `info`
- `warning`
- `critical`

Die Abbildung der Einzelergebnisse auf die Gesamtphase (LH-F-006) erfolgt nach folgender Regel:

| Höchster Schweregrad eines fehlgeschlagenen Checks | Resultierende Gesamtphase |
| -------------------------------------------------- | ------------------------- |
| kein Fehler                                        | `Passed`                  |
| `info`                                             | `Passed` (mit Hinweisen)  |
| `warning`                                          | `Warning`                 |
| `critical`                                         | `Failed`                  |
| nicht ermittelbar (z. B. RBAC, Timeout)            | `Unknown`                 |

Ein fehlgeschlagener `critical`-Check führt damit immer zur Gesamtphase `Failed`, auch wenn alle anderen Prüfungen erfolgreich waren.

### LH-F-032 — Ergebnis pro Prüfung

Jede einzelne Prüfung soll mindestens folgende Informationen liefern:

- Name der Prüfung
- Status
- Schweregrad
- Grund
- Beschreibung
- Zeitpunkt
- optional technische Details

### LH-F-033 — Mandantenunabhängigkeit

Der Operator soll in der initialen Version keine OpenDesk-Mandanten verwalten.

### LH-F-034 — Mehrere Instanzen

Der Operator soll mehrere `OpenDeskPreflightCheck`-Objekte im selben Cluster verwalten können.

### LH-F-035 — Lesender Betrieb

Der Operator muss im initialen Umfang ausschließlich lesend auf den Cluster zugreifen und darf produktive OpenDesk-Ressourcen nicht verändern.

Zulässige Schreibzugriffe beschränken sich auf eigene Ressourcen des Operators (Status der eigenen Custom Resources, eigene Events sowie optional vom Operator erzeugte ConfigMaps gemäß LH-F-028).

---

## 12. Nicht-funktionale Anforderungen

### LH-NF-001 — Programmiersprache

Das Produkt soll in Go entwickelt werden.

### LH-NF-002 — Kubernetes-Konventionen

Das Produkt soll sich an Kubernetes-Konventionen für Operatoren, CRDs, Conditions und RBAC orientieren.

### LH-NF-003 — Nachvollziehbarkeit

Prüfergebnisse müssen nachvollziehbar und verständlich formuliert sein.

### LH-NF-004 — Stabilität

Der Operator darf bei fehlschlagenden Prüfungen nicht abstürzen.

### LH-NF-005 — Fehlertoleranz

Fehler einzelner Prüfungen dürfen die Ausführung anderer Prüfungen nicht verhindern.

### LH-NF-006 — Sicherheit

Der Operator soll mit minimal notwendigen Berechtigungen betrieben werden können.

### LH-NF-007 — Datenschutz

Der Operator darf keine sensiblen Zugangsdaten im Status oder in Events ausgeben.

### LH-NF-008 — Observability

Der Operator soll eigene Metriken bereitstellen können.

### LH-NF-009 — Logging

Der Operator soll strukturierte Logs erzeugen.

### LH-NF-010 — Testbarkeit

Das Projekt soll automatisierte Tests für zentrale Prüfungen ermöglichen.

### LH-NF-011 — Erweiterbarkeit

Neue Prüfungen sollen mit vertretbarem Aufwand ergänzt werden können.

### LH-NF-012 — Wartbarkeit

Die Codebasis soll modular aufgebaut sein.

### LH-NF-013 — Dokumentation

Das Projekt muss eine verständliche Dokumentation für Installation, Nutzung und Entwicklung enthalten.

### LH-NF-014 — Lokale Entwicklung

Das Projekt soll lokal mit üblichen Kubernetes-Entwicklungswerkzeugen testbar sein.

### LH-NF-015 — Containerisierung

Der Operator soll als Container-Image bereitgestellt werden können.

### LH-NF-016 — Helm-Installierbarkeit

Das Produkt soll per Helm Chart installierbar sein.

### LH-NF-017 — GitOps-Kompatibilität

Das Produkt soll mit GitOps-Werkzeugen wie Argo CD oder Flux nutzbar sein.

### LH-NF-018 — Plattformneutralität

Der Operator soll nicht an einen bestimmten Kubernetes-Anbieter gebunden sein.

### LH-NF-019 — Ressourcenverbrauch

Der Operator soll ressourcenschonend betrieben werden können.

### LH-NF-020 — Rückwärtskompatibilität

Änderungen an der CRD sollen nach Möglichkeit rückwärtskompatibel gestaltet werden.

### LH-NF-021 — Projektsprache

Die Sprache der Projektartefakte ist wie folgt festgelegt:

- Quellcode, Bezeichner, Codekommentare: Englisch
- Issues, Pull Requests, Commit Messages: Englisch
- Benutzersichtbare Operator-Ausgaben (Conditions, Reasons, Messages, Events, Logs): Englisch
- CONTRIBUTING, Code of Conduct: Englisch
- README.md, Lastenheft, Pflichtenheft, fachliche Spezifikationsdokumente: Deutsch

Begründung: Englisch für Code und Community-Artefakte ermöglicht internationale Mitwirkende; Deutsch für die fachlichen Spezifikationsdokumente entspricht der Zielgruppe behördennaher und deutschsprachiger Betreiber (siehe LH-PK-004).

---

## 13. Schnittstellenanforderungen

### LH-SST-001 — Kubernetes API

Das System muss die Kubernetes API zur Abfrage von Clusterressourcen verwenden.

### LH-SST-002 — Custom Resource API

Das System muss eine Kubernetes Custom Resource API bereitstellen.

### LH-SST-003 — Kubernetes Events

Das System soll Kubernetes Events zur Signalisierung wichtiger Zustände verwenden.

### LH-SST-004 — Prometheus Metrics

Das System soll Metriken in einem Prometheus-kompatiblen Format bereitstellen können.

### LH-SST-005 — DNS

Das System soll DNS-Auflösung für konfigurierte Hostnamen durchführen können.

### LH-SST-006 — HTTPS/TLS

Das System soll TLS-Zertifikate konfigurierter HTTPS-Endpunkte prüfen können.

### LH-SST-007 — PostgreSQL

Das System soll PostgreSQL-Endpunkte auf Erreichbarkeit prüfen können.

### LH-SST-008 — S3-kompatibler Object Storage

Das System soll S3-kompatible Endpunkte auf Erreichbarkeit prüfen können.

### LH-SST-009 — Container Registry

Das Produkt soll als Container-Image aus einer Registry installierbar sein.

### LH-SST-010 — Helm Repository

Das Produkt soll optional über ein Helm Repository installierbar sein.

---

## 14. Datenanforderungen

### LH-DAT-001 — Keine Speicherung sensibler Daten

Das System darf keine sensiblen Zugangsdaten dauerhaft speichern.

### LH-DAT-002 — Statusdaten

Das System muss Prüfergebnisse im Status der Custom Resource speichern.

### LH-DAT-003 — Zeitstempel

Das System muss Zeitstempel für Prüfläufe speichern.

### LH-DAT-004 — Reportdaten

Das System kann zusätzlich menschenlesbare Reportdaten erzeugen.

### LH-DAT-005 — Technische Details

Technische Details dürfen nur gespeichert werden, wenn sie keine sensiblen Informationen enthalten.

### LH-DAT-006 — Konfigurationsdaten

Die durch Benutzer bereitgestellte Prüfkonfiguration muss in der Custom Resource abbildbar sein.

### LH-DAT-007 — Secret-Referenzierung

Zugangsdaten zur Prüfung externer Dienste (z. B. PostgreSQL, S3-kompatibler Object Storage) dürfen nicht direkt in der Custom Resource enthalten sein.

Solche Zugangsdaten müssen ausschließlich über `secretRef`-Felder auf bestehende Kubernetes-Secrets im selben Namespace referenziert werden. Der Operator soll die referenzierten Secrets nur zur Laufzeit lesen und ihre Werte weder im Status, in Events, in Logs noch in Reports ausgeben.

Key-Konventionen pro Dienst, Failure-Conditions, TLS-Vertrauensstellung und erlaubte Auth-Methoden sind in `ADR 0010` festgelegt.

---

## 15. Sicherheitsanforderungen

### LH-SEC-001 — Minimalrechte

Der Operator soll nach dem Least-Privilege-Prinzip betrieben werden.

### LH-SEC-002 — Keine Secret-Ausgabe

Der Operator darf keine Secret-Werte in Logs, Events, Status oder Reports ausgeben.

### LH-SEC-003 — RBAC-Transparenz

Die benötigten RBAC-Rechte müssen dokumentiert sein.

### LH-SEC-004 — Netzwerktests

Netzwerktests dürfen nur gegen konfigurierte Ziele ausgeführt werden.

### LH-SEC-005 — Keine destruktiven Aktionen

Der Operator darf im initialen Umfang keine destruktiven Aktionen im Cluster ausführen.

Als destruktive Aktionen gelten insbesondere:

- Löschen oder Patchen fremder Ressourcen (z. B. Deployments, StatefulSets, PersistentVolumeClaims, Secrets, ConfigMaps)
- Modifikation von Cluster-weiten Ressourcen (z. B. CRDs, ClusterRoles, ClusterIssuers, StorageClasses, IngressClasses)
- Drain, Cordon oder Neustart von Nodes
- Erzeugung von Workloads in fremden Namespaces
- Schreibende Zugriffe auf externe Dienste (z. B. Anlegen/Löschen von Datenbanken oder Buckets)

Erlaubt sind ausschließlich die in LH-F-035 genannten Schreibzugriffe auf eigene Ressourcen des Operators.

### LH-SEC-006 — Sichere Defaults

Standardwerte sollen konservativ und sicher gewählt werden.

### LH-SEC-007 — Auditierbarkeit

Wichtige Zustandsänderungen sollen über Kubernetes Events oder Logs nachvollziehbar sein.

---

## 16. Qualitätsanforderungen

### LH-QA-001 — Verständliche Fehlermeldungen

Fehlermeldungen müssen so formuliert sein, dass Betreiber daraus konkrete Maßnahmen ableiten können.

### LH-QA-002 — Reproduzierbare Ergebnisse

Bei unverändertem Clusterzustand und unveränderter Konfiguration sollen Prüfergebnisse reproduzierbar sein.

### LH-QA-003 — Keine falsche Sicherheit

Das System darf eine Umgebung nicht als produktionsbereit bewerten, wenn kritische Prüfungen fehlschlagen.

### LH-QA-004 — Transparente Bewertung

Die Bewertungskriterien der Prüfungen müssen dokumentiert sein.

### LH-QA-005 — Erweiterbare Prüfstruktur

Die Struktur der Prüfungen muss spätere OpenDesk-spezifische Erweiterungen ermöglichen.

### LH-QA-006 — Robuste Statuspflege

Der Status der Custom Resource muss auch bei Fehlern konsistent bleiben.

---

## 17. Abnahmekriterien

### LH-AK-001 — CRD installierbar

Die CRD `OpenDeskPreflightCheck` lässt sich in einem Kubernetes-Cluster installieren.

### LH-AK-002 — Operator startbar

Der Operator lässt sich in einem Kubernetes-Cluster starten.

### LH-AK-003 — Ressource verarbeitbar

Eine angelegte `OpenDeskPreflightCheck`-Ressource wird durch den Operator erkannt.

### LH-AK-004 — Status sichtbar

Der Operator schreibt einen Status in die Custom Resource.

### LH-AK-005 — Kubernetes-Version prüfbar

Der Operator kann die Kubernetes-Version prüfen und das Ergebnis im Status darstellen.

### LH-AK-006 — StorageClass prüfbar

Der Operator kann vorhandene und fehlende StorageClasses erkennen.

### LH-AK-007 — IngressClass prüfbar

Der Operator kann vorhandene und fehlende IngressClasses erkennen.

### LH-AK-008 — cert-manager prüfbar

Der Operator kann erkennen, ob cert-manager-Ressourcen im Cluster vorhanden sind.

### LH-AK-009 — Ressourcen prüfbar

Der Operator kann die allocatable CPU- und Arbeitsspeicherkapazität aller Nodes im Zustand `Ready` summieren und gegen die in der Custom Resource konfigurierten Mindestwerte für CPU und Speicher prüfen. Das Ergebnis wird im Status als Condition dargestellt.

### LH-AK-010 — Fehlerfall robust

Der Operator bleibt lauffähig, wenn einzelne Prüfungen fehlschlagen.

### LH-AK-011 — Conditions vorhanden

Die Ergebnisse werden als Conditions dargestellt.

### LH-AK-012 — Keine Secret-Leaks

Status, Events und Logs enthalten keine Secret-Werte.

### LH-AK-013 — Dokumentation vorhanden

Das Projekt enthält eine Dokumentation mit Installations- und Nutzungsbeispielen.

### LH-AK-014 — Open-Source-Veröffentlichung möglich

Das Projekt enthält eine MIT-Lizenzdatei, README und grundlegende Beitragsinformationen.

### LH-AK-015 — Minimalrechte dokumentiert

Der Operator wird mit einer dokumentierten, minimalen RBAC-Konfiguration ausgeliefert (z. B. als Bestandteil des Helm Charts oder als Manifeste im Repository). Die erteilten Rechte reichen für den deklarierten Funktionsumfang aus und gehen nicht darüber hinaus.

### LH-AK-016 — RBAC-Selbstprüfung wirksam

Der Operator erkennt eigene fehlende Berechtigungen, signalisiert dies über eine Condition `RBACInsufficient` und bricht die Gesamtausführung dadurch nicht ab.

---

## 18. Priorisierung

### LH-PRI-001 — Muss-Anforderungen für MVP

Für das MVP gelten folgende Anforderungen als zwingend:

- LH-F-001
- LH-F-002
- LH-F-003
- LH-F-004
- LH-F-005
- LH-F-006
- LH-F-007
- LH-F-008
- LH-F-010
- LH-F-012
- LH-F-013
- LH-F-015
- LH-F-024
- LH-F-035
- LH-NF-001
- LH-NF-002
- LH-NF-004
- LH-NF-005
- LH-NF-006
- LH-NF-013
- LH-SST-004 (Prometheus-Format, Framework-Defaults gemäß ADR 0007)
- LH-SEC-001
- LH-SEC-002
- LH-SEC-005
- LH-DAT-007
- LH-AK-001 bis LH-AK-016

### LH-PRI-002 — Soll-Anforderungen für Version 0.2

Für Version 0.2 sind vorgesehen:

- DNS-Prüfung (LH-F-018)
- TLS-Prüfung (LH-F-019)
- Netzwerk-Reachability (LH-F-022) — gemäß `ADR 0010` ohne-Auth-Block
- Node-Anzahl- und Zustandsprüfung (LH-F-016, LH-F-017)
- ClusterIssuer-Prüfung (LH-F-014)
- Events (LH-F-027)
- Report als ConfigMap (LH-F-028) — Format YAML + Markdown gemäß `ADR 0008`
- Eigene Domänen-Metriken (LH-NF-008) — Prometheus-Format (LH-SST-004) bereits im MVP erfüllt gemäß `ADR 0007`
- Helm Chart (LH-NF-016, LH-SST-010) — aus MVP ausgeklammert mit `ADR 0005`

### LH-PRI-003 — Kann-Anforderungen für spätere Versionen

Für spätere Versionen sind möglich:

- PostgreSQL-Prüfung (LH-F-020) — mit-Auth-Block gemäß `ADR 0010`; Aktivierung frühestens v0.3 in eigener Folge-ADR
- Object-Storage-Prüfung (LH-F-021) — mit-Auth-Block gemäß `ADR 0010`; Aktivierung frühestens v0.3 in eigener Folge-ADR
- vordefinierte Plattformprofile
- HTML-Report
- kubectl Plugin
- Integration in OpenDesk-spezifische Upgrade- oder Wartungsprozesse

---

## 19. V-Modell-Zuordnung

### LH-VM-001 — Lastenheft

Dieses Dokument stellt das Lastenheft dar.

### LH-VM-002 — Pflichtenheft

Aus diesem Lastenheft soll ein Pflichtenheft abgeleitet werden.

Das Pflichtenheft soll insbesondere enthalten:

- Architekturentscheidung
- CRD-Schema
- Controller-Design
- Paketstruktur
- RBAC-Konzept
- Testkonzept
- Build- und Release-Konzept

### LH-VM-003 — Systementwurf

Der Systementwurf soll aus dem Pflichtenheft abgeleitet werden.

### LH-VM-004 — Implementierung

Die Implementierung soll gegen die Anforderungen dieses Lastenhefts und die Spezifikation des Pflichtenhefts erfolgen.

### LH-VM-005 — Integrationstest

Integrationstests sollen die Abnahmekriterien dieses Lastenhefts abdecken.

### LH-VM-006 — Nachverfolgbarkeit

Jede wesentliche Implementierungsfunktion soll auf mindestens eine Lastenheftkennung zurückführbar sein.

---

## 20. Traceability-Matrix

| Lastenheftkennung | Thema                            | Abnahmekriterium |
| ----------------- | -------------------------------- | ---------------- |
| LH-F-001          | CRD bereitstellen                | LH-AK-001        |
| LH-F-002          | Preflight-Ressource anlegen      | LH-AK-003        |
| LH-F-003          | Controller verarbeitet Ressource | LH-AK-003        |
| LH-F-004          | Status aktualisieren             | LH-AK-004        |
| LH-F-005          | Conditions darstellen            | LH-AK-011        |
| LH-F-006          | Gesamtphase darstellen           | LH-AK-004        |
| LH-F-007          | Zusammenfassung der Ergebnisse   | LH-AK-004        |
| LH-F-008          | Kubernetes-Version prüfen        | LH-AK-005        |
| LH-F-009          | Kubernetes-API-Erreichbarkeit    | LH-AK-002        |
| LH-F-010          | StorageClass prüfen              | LH-AK-006        |
| LH-F-011          | Default StorageClass erkennen    | LH-AK-006        |
| LH-F-012          | IngressClass prüfen              | LH-AK-007        |
| LH-F-013          | cert-manager prüfen              | LH-AK-008        |
| LH-F-015          | Ressourcen prüfen                | LH-AK-009        |
| LH-F-024          | RBAC-Selbstprüfung               | LH-AK-016        |
| LH-F-035          | Lesender Betrieb                 | LH-AK-010        |
| LH-NF-004         | Stabilität                       | LH-AK-010        |
| LH-NF-005         | Fehlertoleranz                   | LH-AK-010        |
| LH-NF-013         | Dokumentation                    | LH-AK-013        |
| LH-SST-004        | Prometheus-Format (Framework-Defaults, ADR 0007) | LH-AK-002 |
| LH-SEC-001        | Minimalrechte                    | LH-AK-015        |
| LH-SEC-002        | Keine Secret-Ausgabe             | LH-AK-012        |
| LH-SEC-005        | Keine destruktiven Aktionen      | LH-AK-010        |
| LH-DAT-007        | Secret-Referenzierung            | LH-AK-012        |

---

## 21. Risiken

### LH-RISK-001 — Unklare OpenDesk-Anforderungen

Die konkreten Anforderungen von OpenDesk können sich ändern.

Gegenmaßnahme: Anforderungen als Profile modellieren und versionierbar halten.

### LH-RISK-002 — Zu großer Projektumfang

Das Projekt kann zu groß werden, wenn Installation, Konfiguration und Reparatur ebenfalls abgedeckt werden.

Gegenmaßnahme: Der MVP beschränkt sich auf lesende Preflight-Prüfungen.

### LH-RISK-003 — Falsche Produktionsbewertung

Eine fehlerhafte Prüfung könnte eine ungeeignete Umgebung als geeignet bewerten.

Gegenmaßnahme: Kritische Prüfungen konservativ bewerten und Bewertungskriterien dokumentieren.

### LH-RISK-004 — Zu hohe RBAC-Rechte

Der Operator könnte zu weitreichende Berechtigungen benötigen.

Gegenmaßnahme: Minimalrechte definieren und optionale Prüfungen an optionale Rechte koppeln.

### LH-RISK-005 — Secret-Leaks

Prüfungen externer Dienste könnten versehentlich sensible Informationen offenlegen.

Gegenmaßnahme: Status- und Log-Ausgaben strikt filtern.

### LH-RISK-006 — Providerabhängigkeit

Der Operator könnte unbeabsichtigt nur mit bestimmten Kubernetes-Distributionen funktionieren.

Gegenmaßnahme: Plattformneutrale Prüfungen bevorzugen und providerspezifische Profile getrennt behandeln.

### LH-RISK-007 — Domain-Verlust der API-Gruppen-Halter-Domain

Die Kubernetes-API-Gruppe `k-deskflight.geo-terrain.net` (`ADR 0006`) ist an die Halter-Domain `geo-terrain.net` gebunden. Ein Verlust der Domain (fehlende Verlängerung, Übertragung an Dritte, Registrar-Problem) würde alle k-deskflight-CRDs invalidieren — analog zu `LH-RISK-006`, hier auf Domain-Ebene.

Gegenmaßnahme: Domain-Verlängerung als operative Routine sichern (Verantwortung des Projektowners); bei längerfristigem Domainrisiko Migration über `v1alpha2` oder neue API-Gruppe in eigener ADR.

---

## 22. Offene Punkte

| Kennung   | Offener Punkt                                                              | Status                                                                                                              |
| --------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| LH-OP-001 | Exakte Mindestversionen für OpenDesk-Profile festlegen                     | Geschlossen mit [`ADR 0009`](../docs/plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md)             |
| LH-OP-002 | Namensraum und API-Gruppe final entscheiden                                | Geschlossen mit [`ADR 0006`](../docs/plan/adr/0006-api-gruppe-und-crd-scope.md) (Vorklärung archiviert unter [`docs/archive/api-gruppe-domain.md`](../docs/archive/api-gruppe-domain.md)) |
| LH-OP-003 | Lizenz auswählen                                                           | entfallen mit Commit `3be7b28` (MIT-Entscheidung; vor Einführung des ADR-Lifecycles)                                |
| LH-OP-004 | Unterstützte Kubernetes-Versionen definieren                               | Geschlossen mit [`ADR 0009`](../docs/plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md)             |
| LH-OP-005 | Umfang der externen Dienstprüfungen festlegen                              | Geschlossen mit [`ADR 0010`](../docs/plan/adr/0010-externe-dienstpruefungen-und-secret-mechanik.md)                  |
| LH-OP-006 | Entscheidung über Helm Chart im MVP treffen                                | Geschlossen mit [`ADR 0005`](../docs/plan/adr/0005-helm-chart-nicht-im-mvp.md)                                       |
| LH-OP-007 | Entscheidung über Prometheus Metrics im MVP treffen                        | Geschlossen mit [`ADR 0007`](../docs/plan/adr/0007-prometheus-metrik-scope-im-mvp.md)                                |
| LH-OP-008 | Entscheidung über Report-Format treffen                                    | Geschlossen mit [`ADR 0008`](../docs/plan/adr/0008-report-format-stack.md)                                          |
| LH-OP-009 | Projektname finalisieren                                                   | Geschlossen mit [`ADR 0004`](../docs/plan/adr/0004-projektname.md)                                                  |
| LH-OP-010 | Governance für Open-Source-Beiträge definieren                             | offen                                                                                                               |
| LH-OP-011 | Behandlung von Authentifizierungs-Secrets für externe Dienste detaillieren | Geschlossen mit [`ADR 0010`](../docs/plan/adr/0010-externe-dienstpruefungen-und-secret-mechanik.md)                  |

---

## 23. MVP-Zielumfang

### LH-MVP-001 — Ziel des MVP

Das MVP soll zeigen, dass ein Kubernetes Operator zuverlässig eine `OpenDeskPreflightCheck`-Ressource verarbeiten und grundlegende Cluster-Voraussetzungen prüfen kann.

### LH-MVP-002 — MVP-Funktionen

Das MVP umfasst:

- CRD `OpenDeskPreflightCheck`
- Controller für die Ressource
- Prüfung der Kubernetes-Version
- Prüfung von StorageClasses
- Prüfung von IngressClasses
- Prüfung von cert-manager (Vorhandensein)
- Prüfung grundlegender Cluster-Ressourcen (CPU, Speicher)
- Prüfung eigener RBAC-Berechtigungen
- Status Conditions
- Gesamtphase
- Zusammenfassung (passed/warning/failed/Zeitstempel)
- ausschließlich lesender Betrieb gemäß LH-F-035
- einfache Dokumentation
- Container-Image
- Beispielmanifest
- Prometheus-`/metrics`-Endpoint mit controller-runtime-Defaults (LH-SST-004 gemäß ADR 0007)

### LH-MVP-003 — Nicht Bestandteil des MVP

Nicht Bestandteil des MVP sind:

- Installation von OpenDesk
- Änderung bestehender OpenDesk-Komponenten
- automatische Reparaturen
- Verwaltung von Mandanten
- Backup-Orchestrierung
- Upgrade-Orchestrierung
- komplexe externe Dienstprüfungen

---

## 24. Zielarchitektur aus Anwendersicht

### LH-ZA-001 — Anwendersicht

Ein Betreiber installiert den Operator im Cluster.

Anschließend legt der Betreiber eine `OpenDeskPreflightCheck`-Ressource an.

Der Operator prüft den Cluster und schreibt die Ergebnisse in den Status der Ressource.

Der Betreiber kann das Ergebnis mit Kubernetes-Standardwerkzeugen abrufen.

Beispiel:

```bash
kubectl get opendeskpreflightcheck cluster-readiness
kubectl describe opendeskpreflightcheck cluster-readiness
kubectl get opendeskpreflightcheck cluster-readiness -o yaml
```

### LH-ZA-002 — GitOps-Sicht

In einem GitOps-Szenario wird die `OpenDeskPreflightCheck`-Ressource im Git-Repository abgelegt.

Der Operator aktualisiert den Status im Cluster.

Andere Werkzeuge können den Status auswerten.

### LH-ZA-003 — CI/CD-Sicht

In einem CI/CD-Szenario kann ein temporärer Cluster geprüft werden, bevor OpenDesk installiert oder aktualisiert wird.

---

## 25. Beispielhafte Benutzeranforderungen

### LH-UA-001 — Cluster readiness erkennen

Als Kubernetes-Administrator möchte ich sehen, ob mein Cluster grundsätzlich für eine OpenDesk-Installation geeignet ist, damit ich Installationsfehler früh erkenne.

### LH-UA-002 — Fehlende Voraussetzungen erkennen

Als Plattformbetreiber möchte ich fehlende StorageClasses, IngressClasses oder Zertifikatskomponenten erkennen, damit ich diese vor der Installation bereitstellen kann.

### LH-UA-003 — GitOps nutzen

Als DevOps-Team möchte ich Preflight Checks deklarativ in Git verwalten, damit die Prüfung reproduzierbar ist.

### LH-UA-004 — Status automatisiert auswerten

Als CI/CD-Verantwortlicher möchte ich Prüfergebnisse maschinenlesbar auswerten, damit Pipelines bei kritischen Fehlern abbrechen können.

### LH-UA-005 — Dokumentation erzeugen

Als Betreiber möchte ich einen nachvollziehbaren Prüfstatus dokumentieren, damit Entscheidungen über Produktionsreife belegbar sind.

---

## 26. Release-Zielbild

### LH-REL-001 — Version 0.1

Version 0.1 soll ein technisches MVP bereitstellen.

### LH-REL-002 — Version 0.2

Version 0.2 soll zusätzliche Prüfungen und bessere Auswertbarkeit bieten.

### LH-REL-003 — Version 0.3

Version 0.3 soll Reporting, Metriken und erste Profilvarianten verbessern.

### LH-REL-004 — Version 1.0

Version 1.0 soll eine stabile CRD-Version, dokumentierte Profile und produktionsnahe Nutzung ermöglichen.

---

## 27. Erfolgskriterien

### LH-ERF-001 — Technischer Erfolg

Das Projekt ist erfolgreich, wenn der Operator wiederholbar Preflight Checks ausführen und nachvollziehbar im Kubernetes-Status darstellen kann.

### LH-ERF-002 — Nutzwert

Das Projekt ist erfolgreich, wenn Betreiber vor der OpenDesk-Installation konkrete fehlende Voraussetzungen erkennen.

### LH-ERF-003 — Open-Source-Fähigkeit

Das Projekt ist erfolgreich, wenn externe Entwickler das Projekt lokal bauen, testen und erweitern können.

### LH-ERF-004 — Erweiterbarkeit

Das Projekt ist erfolgreich, wenn neue Prüfungen ohne grundlegende Architekturänderung ergänzt werden können.

---

## 28. Schlussbemerkung

Dieses Lastenheft beschreibt bewusst ein begrenztes, aber tragfähiges Zielbild.

Der OpenDesk Preflight Operator soll nicht versuchen, OpenDesk vollständig zu betreiben. Er soll zunächst ein sauberes, Kubernetes-natives Werkzeug zur Vorabprüfung von OpenDesk-Umgebungen werden.

Diese Begrenzung ist wichtig, damit das Projekt realistisch umsetzbar, testbar und als Open-Source-Projekt glaubwürdig bleibt.
