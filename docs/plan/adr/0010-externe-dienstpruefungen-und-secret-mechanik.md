# ADR 0010 — Externe Dienstprüfungen: Phasen und Secret-Mechanik

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[ADR 0006](0006-api-gruppe-und-crd-scope.md)

---

## 1. Kontext

Zwei verschränkte offene Punkte:

- `LH-OP-005` — Umfang der externen Dienstprüfungen festlegen.
- `LH-OP-011` — Behandlung von Authentifizierungs-Secrets für externe
  Dienste detaillieren.

Die fünf betroffenen funktionalen Anforderungen zerfallen in zwei
Gruppen:

| Kennung | Inhalt | Auth nötig? | Aktueller Lastenheft-Status |
| ------- | ------ | ----------- | --------------------------- |
| `LH-F-018` | DNS-Auflösung | nein | v0.2-Soll (`LH-PRI-002`) |
| `LH-F-019` | TLS-Zertifikate | nein | v0.2-Soll (`LH-PRI-002`) |
| `LH-F-022` | Netzwerk-Reachability allgemein | nein | unverortet |
| `LH-F-020` | PostgreSQL-Endpunkt | ja (Credentials) | Kann (`LH-PRI-003`) |
| `LH-F-021` | S3 Object Storage | ja (Credentials) | Kann (`LH-PRI-003`) |

Vorhandene Lastenheft-Festlegungen, die Secret-relevant sind:

- `LH-DAT-007`: `secretRef` auf bestehende Secrets im selben Namespace,
  nur Laufzeit-Lesen, keine Ausgabe.
- `LH-SEC-002`: keine Secret-Ausgabe in Logs, Events, Status, Reports.
- `LH-NF-007`: kein sensibler Inhalt in Status/Events.
- `LH-RISK-005`: Secret-Leaks als anerkanntes Risiko.
- `LH-SYS-006`: keine Secret-Erzeugung/-Speicherung durch den Operator.

`LH-OP-011` fragt nach Detail, das `LH-DAT-007` offen lässt:
Datenstruktur der Secrets, Key-Konventionen pro Dienst, Failure-
Conditions, TLS-CA-Trust, erlaubte Auth-Methoden.

---

## 2. Entscheidung

### 2.1 Phasenplan externe Dienstprüfungen (LH-OP-005)

| Phase | Inhalt | Auth | Bezug |
| ----- | ------ | ---- | ----- |
| MVP (v0.1, `LH-REL-001`) | **keine externen Dienstprüfungen** | — | `LH-PRI-001` enthält keine externen Prüfungen |
| v0.2 (`LH-REL-002`) | DNS (`LH-F-018`), TLS (`LH-F-019`), Netzwerk-Reachability (`LH-F-022`) | nein | `LH-PRI-002` |
| v0.3+ (`LH-REL-003` und später) | PostgreSQL (`LH-F-020`), S3 Object Storage (`LH-F-021`) | ja | konkrete Aktivierung in eigener Folge-ADR |

Begründung der Trennung: alle ohne-Auth-Prüfungen kommen v0.2
gemeinsam, weil sie keinerlei Secret-Handling brauchen und sich
unabhängig voneinander testen lassen. Die mit-Auth-Prüfungen folgen
v0.3+ als geschlossener Block, weil sie die in §2.2–§2.6 festgelegte
Secret-Mechanik gemeinsam erstmals aktivieren.

### 2.2 Secret-Datenstruktur (LH-OP-011)

Secrets sind Kubernetes-Standard-Secrets vom Typ `Opaque` im selben
Namespace wie die CR (`LH-DAT-007`). Der Operator liest sie nur zur
Laufzeit und cached sie nicht über einen Reconcile-Lauf hinaus.

**Key-Konventionen pro Dienst:**

PostgreSQL (`LH-F-020`):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-preflight-credentials
  namespace: <namespace-der-cr>
type: Opaque
data:
  username:     <base64>   # Pflicht
  password:     <base64>   # Pflicht
  # optional:
  database:     <base64>   # Default: postgres
  sslmode:      <base64>   # require | verify-ca | verify-full | disable | prefer
```

S3-kompatibles Object Storage (`LH-F-021`):

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: s3-preflight-credentials
  namespace: <namespace-der-cr>
type: Opaque
data:
  accessKeyId:      <base64>   # Pflicht
  secretAccessKey:  <base64>   # Pflicht
  # optional:
  sessionToken:     <base64>   # für temporäre STS-Credentials
  region:           <base64>   # z.B. eu-central-1
```

Key-Namen sind verbindlich (camelCase, ASCII). Andere Key-Namen oder
zusätzliche Keys werden vom Operator ignoriert.

### 2.3 Failure-Conditions

Pro extern geprüftem Dienst veröffentlicht der Operator höchstens
eine Condition vom folgenden Set (Reihenfolge = Prüfreihenfolge,
abgebrochen beim ersten Fehler):

| Condition-Type | Status | Reason | Severity | Bedeutung |
| -------------- | ------ | ------ | -------- | --------- |
| `SecretMissing` | `False` | `SecretNotFound` | `critical` | `secretRef` zeigt auf nicht-existentes Secret |
| `SecretMalformed` | `False` | `SecretKeyMissing` oder `SecretKeyEmpty` | `critical` | Pflicht-Key fehlt oder ist leer |
| `ConnectivityUnknown` | `Unknown` | `Timeout` oder `NetworkUnreachable` | `warning` | TCP-Verbindung scheitert vor Auth |
| `AuthFailed` | `False` | `AuthenticationFailed` | `critical` | Verbindung steht, Auth schlägt fehl |
| `ConnectionVerified` | `True` | `ConnectionEstablished` | — | Auth erfolgreich, Verbindung steht |

Die Schweregrade entsprechen `LH-F-031`: ein `critical`-Fail führt
zur Gesamtphase `Failed`, ein `warning` zu `Warning`, `Unknown`
ist nicht ermittelbar. `ConnectivityUnknown` bewusst nicht
`critical`: ein temporärer Netzwerk-Aussetzer soll keine Cluster-
Bereitschafts-Aussage hart umstoßen.

### 2.4 TLS-Vertrauensstellung

Für TLS-basierte externe Endpunkte (HTTPS-Probes für `LH-F-019`,
S3-HTTPS für `LH-F-021`, PostgreSQL mit `sslmode != disable`) gilt:

- **Default:** Der Operator nutzt das System-CA-Bundle seines
  Container-Images zur Zertifikatsverifikation. Das Bundle wird mit
  jedem Operator-Release aktualisiert.
- **Custom CA:** Anwender können ein selbst-signiertes oder
  eigenes-Root-CA-Zertifikat über einen optionalen CR-Spec-Verweis
  `caConfigMapRef` bereitstellen. Erwartet wird eine ConfigMap im
  selben Namespace mit dem Key `ca.crt` (PEM-kodiert). Konkretes
  Spec-Feld ist Pflichtenheft-Inhalt (`LH-VM-002`).
- **`insecureSkipVerify`:** Erlaubt als explizites CR-Spec-Feld pro
  Dienst, Default `false`. Bei `true` veröffentlicht der Operator
  eine zusätzliche Condition `TLSVerificationDisabled` (Severity
  `warning`) und ein Kubernetes-Event. Niemals Profile-Default.

### 2.5 Erlaubte Auth-Methoden

| Dienst | MVP/v0.2 | v0.3+ | Spätere Erweiterung |
| ------ | -------- | ----- | ------------------- |
| PostgreSQL | — | `password` (SCRAM-SHA-256 oder MD5 nach Server-Konfig) | `cert` (Client-Cert-Auth) |
| S3 | — | `accessKey` (statische Credentials) | IRSA, Workload-Identity-Federation, anonyme Reads |

Die `v0.3+`-Auth-Methoden sind das Minimum für die jeweils erste
Aktivierung der Prüfung. Erweiterungen kommen mit eigener ADR.

### 2.6 LH-SEC-002 strikt: was darf in Output

Erlaubt in Reason/Message/Logs:

- Host und Port des konfigurierten Endpunkts.
- TLS-Zertifikats-Fingerprint und Aussteller (für `LH-F-019`).
- Generische Fehlerklassifikation (`connection refused`,
  `timeout after 5s`, `tls handshake failed`).
- Bei `SecretMalformed`: der **Key-Name**, der fehlt oder leer ist
  (nicht der Wert).

Nicht erlaubt:

- Inhalt eines Secret-Keys (auch nicht in Hex, Base64 oder Hash).
- Vollständige Server-Antworten, wenn sie Credentials enthalten könnten.
- TLS-Client-Zertifikate (sofern später unterstützt).
- Verbindungs-Strings mit eingebetteten Passwörtern.

---

## 3. Konsequenzen

- `LH-PRI-002` wird im selben Commit um `LH-F-022` (Netzwerk-
  Reachability) erweitert; v0.2 enthält damit alle ohne-Auth-
  Prüfungen.
- `LH-PRI-003` wird konkretisiert: PostgreSQL und Object Storage
  bleiben dort, ihre Aktivierung wird mit eigener Folge-ADR (frühestens
  v0.3) eingeleitet. Die Phase wird verbindlich, sobald die
  Folge-ADR sie hebt.
- `LH-DAT-007` wird mit einem ADR-0010-Verweis ergänzt (Key-
  Konventionen, Conditions, TLS-Trust in §2.2–§2.6 dieser ADR).
- `LH-OP-005` und `LH-OP-011` werden im Lastenheft als geschlossen
  mit dieser ADR markiert (Formelhilfe aus `ADR 0002`).
- `LH-F-031` (Schweregrade): die Condition-Severities in §2.3 sind
  konform zu diesem Mapping.
- `LH-RISK-005` (Secret-Leaks): §2.6 macht den Output-Vertrag
  explizit; Pflichtenheft muss Test-Cases dafür einfordern.
- Pflichtenheft (`LH-VM-002`) konkretisiert: CR-Spec-Felder
  (`spec.checks.externalServices.postgres`/`.objectStorage`,
  `caConfigMapRef`, `insecureSkipVerify`), exakte Probe-Implementierung,
  Timeouts.

---

## 4. Nicht Gegenstand dieser ADR

- **Konkretes CR-Spec-Schema** der externen-Dienst-Felder
  (`spec.checks.externalServices.…`) — Pflichtenheft.
- **Aktivierungs-Zeitpunkt** der mit-Auth-Prüfungen (`LH-F-020`,
  `LH-F-021`) — eigene Folge-ADR vor v0.3.
- **PostgreSQL-Auth-Methode `cert` (Client-Cert)** und
  **S3-IRSA/Workload-Identity** — spätere ADRs, wenn Bedarf entsteht.
- **External-Secrets-Operator-Integration**, **Vault-Integration**,
  **CSI-Driver-basierter Secret-Bereitstellung** — operative
  Folgearbeit; alle diese Lösungen produzieren am Ende ein
  Kubernetes-Secret im richtigen Namespace und erfüllen damit
  automatisch das in §2.2 festgelegte Konventionsschema.
- **Connection-Pooling** und **Long-lived-Connections**: der Operator
  öffnet pro Prüflauf eine kurze Test-Verbindung und schließt sie
  wieder; keine Persistenz. Konkretes Timeout-Budget im Pflichtenheft.
- **DNSSEC-/DANE-Verifikation** — über `LH-F-018`/`LH-F-019` hinaus,
  spätere optionale Erweiterung.
- **Webhook-basierte Auth-Bridges** (z. B. MediaMTX-Style aus
  `m-trace`) — kein Bezug zu OpenDesk-Vorabprüfungen.
- **Custom-Probe-Skripte** in der CR — Out-of-Scope nach
  `LH-RISK-002` (Projektumfang).
