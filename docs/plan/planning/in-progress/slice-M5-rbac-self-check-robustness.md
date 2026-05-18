# Slice M5 — RBAC-Selbstprüfung & Robustheit

**Status:** In Progress
**Eröffnet:** 2026-05-18
**Vorgänger:** [M4 — Cluster-State-Prüfungen (Done)](../done/slice-M4-cluster-state-checks.md)
**Nachfolger:** [M6 — Metrics-Endpoint, Tests, Doku](roadmap.md#m6--metrics-endpoint-tests-doku)
**Bezug:**
[Roadmap §3 M5](roadmap.md#m5--rbac-selbstpr%C3%BCfung--robustheit),
[`spec/architecture.md` §5 (AR-009 Phase 4), §5 (AR-011), §7 (AR-015 / AR-018)](../../../../spec/architecture.md),
[ADR 0010 §2.3](../../adr/0010-externe-dienstpruefungen-und-secret-mechanik.md)

---

## 1. Lieferziel

Operator prüft vor jedem Check seine eigenen Cluster-Rechte; bei
fehlenden Rechten liefert der betroffene Check `Unknown` mit Reason
`RBACInsufficient`, andere Checks laufen weiter (`LH-AK-016`,
`LH-NF-005`). Reconciler ist gegen Panics in Check-Implementierungen
und gegen Hänger (per-Check-Timeout) abgesichert. Secret-Output-Filter
ist als Pflicht-Konvention im Status- und Log-Pfad verankert
(`LH-AK-012`, `LH-SEC-002`).

**Roadmap-§3-M5-Bullets, die in diesem Slice fallen:**

- [x] `SelfSubjectAccessReview` pro aktivierter Prüfung (`LH-F-024`).
- [x] Condition mit Reason `RBACInsufficient` falls Recht fehlt;
  betroffene Einzelprüfung wird `Unknown` (`LH-AK-016`).
  **Separates** Reason `RBACCheckFailed` für SAR-Subsystem-Ausfall
  (Review-Befund 1) — verhindert Verschleierung echter
  Infrastrukturprobleme.
- [x] Fehlertoleranz: per-Check-`defer/recover` umschließt den
  **gesamten** Per-Check-Pfad (SAR-Loop + Run, Review-Befund 3) plus
  Reconciler-Outer-Recover. Einzelne Check-Fehler erzeugen `Unknown`,
  nicht Abbruch (`LH-NF-005`, `LH-AK-010`).
- [x] Per-Check-Timeout (`AR-009 §4 Step 4`): Hänger einzelner Checks
  blockieren den Reconcile nicht. **Cause-aware** Klassifikation über
  `context.WithTimeoutCause` + `context.Cause(runCtx)`: unser eigener
  Per-Check-Deadline → Reason `Timeout` (critical), ererbter
  Parent-Deadline → Reason `ReconcileTimeout` (critical), Cancel →
  Reason `ReconcileCanceled` (info). **Race-Härtung** beim
  `<-resultCh`-Fall: bei gleichzeitigem Context-End gewinnt die
  Context-Klassifikation gegen das Check-Result (Folge-Review).
- [x] Secret-Output-Filter aktiv (`LH-SEC-002`, `LH-NF-007`,
  `LH-AK-012`) — **zwei** Hook-Funktionen: `SanitizeMessage` (String-
  Pfad für Status-/Event-Texte) und `SanitizeAttrs` (Attribut-Pfad
  für `slog.LogAttrs`, Folge-Review). Tests prüfen, dass kein
  Secret-Inhalt in Logs/Events/Status landet — inkl. strukturierter
  slog-Attrs. Im MVP gibt es noch keine externen Secrets
  (`ADR 0010`), die Hooks sind Identitäts-Stubs für v0.2.
- [x] Keine destruktiven Aktionen (`LH-SEC-005`) — verbleibt
  Konventions-Item, Tests verifizieren via depguard-Profil und
  RBAC-Audit.

**Was M5 noch nicht macht (Roadmap-§3-eng, slice-M5-Scope-Diskussion):**

- **Wiederholintervall** (`AR-010`, `LH-F-025`) — verschoben auf M6
  zusammen mit Anwender-Doku, weil `Spec.Interval` Anwender-Default-
  Erwartungen prägt und die Doku parallel entstehen sollte.
- **Leader-Election** (`AR-026`) — verschoben auf M7-Release-
  Hardening; Single-Pod-MVP-Smoke reicht für v0.1.0-Verifikation.
  RBAC-Pre-Grant für Leases ist bereits in M2 verankert.
- **Worker-Pool** (`AR-009 §4` voll) — verschoben auf v0.2. Fünf
  MVP-Checks sequenziell mit 30s-Timeout summieren sich auf höchstens
  150 s pro Lauf; Parallelisierung rechtfertigt erst sich, wenn
  externe Service-Checks (`LH-F-018..021`) dazukommen.
- **`OPERATOR_STRICT_CONFIG` + `CHECK_TIMEOUT_SECONDS`-Env-Override**
  (`AR-010.1`) — verschoben auf v0.2. M5 nutzt einen hart kodierten
  30 s-Default; Env-Override kommt mit der Configuration-Klassifizierungs-
  ADR zusammen.

---

## 2. Slice-Entscheidungen

### 2.1 Port-Segregation für SelfSubjectAccessReview

Konsistent mit slice-M4 §2.1 bekommt SAR einen eigenen, schmal
geschnittenen Port. `port.KubernetesAPI`/`port.APIGroupDiscovery`
bleiben unverändert.

| Port-Interface | Methoden |
| -------------- | -------- |
| `port.AccessReviewer` | `CanI(ctx context.Context, req PermissionRequest) (bool, error)` |

`PermissionRequest` ist ein **Domain-Struct** (`internal/hexagon/domain/`)
ohne k8s-Imports — analog zu `NodeInfo`. Der Adapter mappt es auf
`authorizationv1.SelfSubjectAccessReview.Spec.ResourceAttributes`.

```go
// domain/permission.go
type PermissionRequest struct {
    Group       string
    Resource    string
    Subresource string
    Verb        string
    Namespace   string // leer = cluster-scoped
}
```

### 2.2 RequiredPermissions als Check-Interface-Methode

Jeder Check deklariert seine Rechte selbst. Das verhindert, dass die
SAR-Liste im Reconciler (Application-Layer) hardcoded landet und
Adapter-Wissen sickert.

**Erweiterung** von `domain.Check` (architecture.md AR-012):

```go
type Check interface {
    Name() string
    SpecKind() string
    RequiredPermissions() []PermissionRequest  // neu (slice-M5 §2.2)
    Run(ctx context.Context, spec CheckSpec) Result
}
```

**RequiredPermissions pro M4-Check:**

| Check | RequiredPermissions |
| ----- | ------------------- |
| `KubernetesVersion` | leer — Discovery-Endpoint braucht keine zusätzlichen Rechte (über `system:discovery`-ClusterRole automatisch gewährt) |
| `CertManager` | leer — Discovery-Pfad analog |
| `StorageClass` | `{Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list"}` |
| `IngressClass` | `{Group: "networking.k8s.io", Resource: "ingressclasses", Verb: "list"}` |
| `ClusterResources` | `{Group: "", Resource: "nodes", Verb: "list"}` |

Diese Werte sind 1:1 deckungsgleich mit den `+kubebuilder:rbac:`-
Markern am Reconciler (`AR-007`), die `config/rbac/role.yaml`
erzeugen. **Konsistenz-Drift wird automatisch geprüft** (Review-
Befund 4): der neue
`internal/hexagon/application/rbac_consistency_test.go` parst die
`+kubebuilder:rbac:`-Marker am Reconciler via `go/parser` und
sammelt die `RequiredPermissions()` aus einer expliziten Check-
Konstruktor-Tabelle. Test schlägt fehl, wenn:

- ein Check ein Recht deklariert, das im Reconciler-Marker fehlt
  (würde zur Smoke-Laufzeit als `RBACInsufficient` erscheinen),
- oder der Marker ein Recht enthält, das von keinem registrierten
  Check beansprucht wird (Hinweis auf Drift / Reste alter Slices).

Die Check-Konstruktor-Tabelle ist Pflicht-Wartung: ein neuer Check
muss dort eingetragen werden. Eine vollständige Reflection-basierte
Lösung wäre möglich, aber für fünf Checks Overengineering.

### 2.3 SAR-Aufrufpunkt: Pre-Execution pro Check, dedupliziert pro Run

Der Reconciler ruft `CanI` für jede `RequiredPermission` einmal pro
Reconcile-Lauf auf. Mehrere Checks mit derselben Permission teilen
sich das Ergebnis über einen Run-lokalen Cache. Damit landen wir
nicht bei `O(checks × permissions)` SAR-Calls, sondern bei `O(unique
permissions)`.

**Drei verschiedene Outcomes pro `CanI`-Call** (Review-Befund 1):

| Outcome | Cache-Wert | Result-Klassifikation |
| ------- | ---------- | --------------------- |
| `CanI` → `(true, nil)` | `allowed=true, err=nil` | Check läuft. |
| `CanI` → `(false, nil)` | `allowed=false, err=nil` | Check liefert `Unknown` + Reason `RBACInsufficient` — echte ClusterRole-Lücke, Cluster-Admin muss handeln. |
| `CanI` → `(_, err≠nil)` | `allowed=false, err≠nil` | Check liefert `Unknown` + Reason `RBACCheckFailed` — Auth-Subsystem-Ausfall (transient oder Konfig), OnCall/Operator-Owner muss handeln. **NICHT** als RBACInsufficient klassifizieren, weil das echte Infrastrukturprobleme verschleiern würde. |

Pseudocode (kommt in `internal/hexagon/application/runner.go`):

```go
type permOutcome struct {
    allowed bool
    err     error // nicht-nil → SAR-Call selbst gescheitert
}
permCache := make(map[PermissionRequest]permOutcome)

for _, check := range active {
    var (
        insufficient []PermissionRequest // CanI: false, nil
        checkFailed  []PermissionRequest // CanI: error
    )
    for _, p := range check.RequiredPermissions() {
        outcome, cached := permCache[p]
        if !cached {
            allowed, err := reviewer.CanI(ctx, p)
            outcome = permOutcome{allowed: allowed, err: err}
            permCache[p] = outcome
        }
        switch {
        case outcome.err != nil:
            checkFailed = append(checkFailed, p)
        case !outcome.allowed:
            insufficient = append(insufficient, p)
        }
    }
    // RBACCheckFailed gewinnt vor RBACInsufficient — der Auth-Subsystem-
    // Ausfall ist die wichtigere Fehlermeldung; ein Cluster-Admin muss
    // erst die Auth-Subsystem-Lücke schließen, bevor RBACInsufficient
    // überhaupt belastbar diagnostizierbar wäre.
    switch {
    case len(checkFailed) > 0:
        results = append(results, rbacCheckFailedResult(check, checkFailed))
    case len(insufficient) > 0:
        results = append(results, rbacInsufficientResult(check, insufficient))
    default:
        results = append(results, runCheckSafely(ctx, check, spec))
    }
}
```

`rbacInsufficientResult` baut ein `domain.Result` mit:

- `Name`: ConditionType des Checks (z. B. `StorageClassReady`).
- `Status`: `Unknown`.
- `Reason`: `RBACInsufficient`.
- `Severity`: `critical` (Anwender muss handeln, sonst kein
  belastbares Ergebnis).
- `Message`: enthält die fehlenden Permissions in CamelCase, ohne
  Secret-Material (siehe §2.6).

`rbacCheckFailedResult` ist strukturell analog, aber:

- `Reason`: `RBACCheckFailed`.
- `Severity`: `critical` (Operations-Owner muss handeln; transient
  oder permanent ist aus Operator-Sicht ununterscheidbar).
- `Message`: nennt den SAR-Endpoint und die fehlgeschlagene
  Permission-Liste, **ohne** Error-Detail (Error-Detail landet im
  Logger via `slog.Error` mit `slog.Any("err", err)` — `LH-SEC-002`
  / `LH-NF-007`).

**Logger** schreibt `WARN`/`ERROR` mit dem Original-Error-Wrap; das
ermöglicht Oncall, transiente von permanenten Auth-Subsystem-
Problemen zu unterscheiden.

`runCheckSafely` umschließt die per-Check-Recover-Logic (§2.4) plus
den Per-Check-Timeout (§2.5). Damit ist der SAR-Loop **nicht** unter
dem per-Check-Recover (Review-Befund 3: ein Panic in `CanI` darf
nicht den ganzen Reconciler abreißen) — siehe §2.4 für die explizite
Lösung.

### 2.4 Panic-Härtung: per-Check + Reconciler-Outer

Zwei Recovery-Layer (AR-011):

1. **Per-Check-Recover** umschließt den **gesamten Per-Check-Pfad**,
   nicht nur `check.Run` (Review-Befund 3). Die Helper-Funktion
   `runCheckSafely(ctx, reviewer, cache, check, spec)` hat einen
   `defer/recover` als äußerste Anweisung und enthält:
   - SAR-Loop (`reviewer.CanI` pro `RequiredPermission`).
   - Cache-Lookup und -Update.
   - Run-mit-Timeout (`runWithTimeout(ctx, check, spec)`).
   Damit wird **jeder Panic** im Per-Check-Pfad lokal gefangen — egal
   ob er aus dem AccessReviewAdapter (nil-pointer, Marshal-Fehler)
   oder aus der Check-Implementierung selbst stammt. Result bei
   Panic: `Status:Unknown, Reason:InternalError, Severity:critical,
   Message:"internal check error (panic recovered)"`. Stack-Trace
   landet via `slog.Error("panic recovered", slog.Any("stack",
   debug.Stack()))` im Logger, **nicht im Status** (LH-SEC-002,
   LH-NF-007).

2. **Reconciler-Outer-Recover** als oberste `defer/recover` in der
   `Reconcile`-Methode. Greift nur, wenn ein Panic außerhalb der
   per-Check-Pipeline geworfen wird (z. B. in `buildSpecMap`, im
   Aggregator, in `writeStatus`). Bei Panic → Status-Update auf
   Phase=`Unknown`, Condition=`ReconcileError`,
   Reason=`ReconcilePanic`. Fehler-Präzedenz: `ReconcileTimeout`
   gewinnt vor `ReconcilePanic` (AR-011).

Pseudocode für `runCheckSafely`:

```go
func runCheckSafely(
    ctx context.Context,
    reviewer port.AccessReviewer,
    cache map[domain.PermissionRequest]permOutcome,
    check domain.Check,
    spec domain.CheckSpec,
) (res domain.Result) {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("panic recovered in check pipeline",
                slog.String("check", check.Name()),
                slog.Any("recover", r),
                slog.Any("stack", debug.Stack()),
            )
            res = internalErrorResult(check)
        }
    }()

    // SAR-Pre-Execution (§2.3) — Panic hier landet in obigem recover.
    if outcome := classifyPermissions(reviewer, cache, check); outcome != nil {
        return *outcome // RBACInsufficient oder RBACCheckFailed
    }

    // Per-Check-Timeout-Wrapper (§2.5) — Panic in check.Run wird ebenfalls
    // hier gefangen (innerhalb derselben Goroutine, falls timeout nicht
    // greift; bei Timeout-Goroutine-Panic siehe §2.5).
    return runWithTimeout(ctx, check, spec)
}
```

### 2.5 Per-Check-Timeout: hartkodiert 30 s

Slice-M5 nutzt einen **hartkodierten 30 s-Default** (AR-009 §4 Step 4
Default-Wert). Kein Env-Override in M5 — der gehört zu `AR-010.1`
zusammen mit `OPERATOR_STRICT_CONFIG` und kommt v0.2.

**Drei Ende-Ursachen** unterscheiden (Review-Befund 2 + Folge-
Review): `runCtx = context.WithTimeoutCause(parentCtx, 30s,
errPerCheckTimeout)` erbt vom Parent und kann auf drei Wegen
beendet werden — unseren eigenen Per-Check-Deadline (Sentinel
`errPerCheckTimeout`), eine ererbte Parent-Deadline
(`RECONCILE_TIMEOUT_SECONDS` oder ein orchestrierter
Aufrufer-Deadline), oder ein expliziter Parent-Cancel
(Operator-Shutdown, Manager-Stop). Wir differenzieren über
`context.Cause(runCtx)` plus `runCtx.Err()`:

| `runCtx.Err()` | `context.Cause(runCtx)` | Reason | Severity | Begründung |
| -------------- | ----------------------- | ------ | -------- | ---------- |
| `DeadlineExceeded` | `errPerCheckTimeout` | `Timeout` | `critical` | echter Hänger im Check, Anwender muss handeln |
| `DeadlineExceeded` | sonst (Parent-Deadline) | `ReconcileTimeout` | `critical` | übergeordnete Reconcile-Frist ausgelaufen, Operator/Cluster zu langsam |
| `Canceled` | beliebig | `ReconcileCanceled` | `info` | Operator-Shutdown / Manager-Stop; kein Fehler im Check, Reconcile wird neu geplant |

`Severity` `info` bei `ReconcileCanceled` heißt: das Phase-Mapping
über `AR-014` schwenkt auf `Warning` (mindestens ein non-passed-
Result), aber der Aggregator-Eintrag in `Summary.Warning` macht es
nicht zu einem `Failed`. Operatoren sehen damit explizit
„abgebrochen vor Abschluss" statt „durchgelaufen mit Fehler".

`Severity` `critical` bei `ReconcileTimeout` ist absichtlich gleich
zu `Timeout` — die Differenzierung dient der Diagnose (welcher Layer
hat das Limit ausgelöst), nicht dem Phase-Mapping.

**Race-Härtung beim `<-resultCh`-Fall** (Review-Befund 1): wenn
`check.Run` exakt zum Timeout-Zeitpunkt fertig wird, hat Go-`select`
nicht-deterministische Auswahl zwischen `resultCh` und
`runCtx.Done()`. Wir validieren nach `<-resultCh` deshalb zusätzlich
`runCtx.Err()` — wenn der Context bereits in `Done`-Zustand ist,
gewinnt die Context-Klassifikation (`Timeout` /
`ReconcileTimeout` / `ReconcileCanceled`) gegen das vermeintliche
Check-Result. Damit ist die Klassifikation stabil unabhängig vom
exakten Scheduler-Verhalten.

Mechanik (`runWithTimeout` in `runner.go`):

```go
var errPerCheckTimeout = errors.New("per-check timeout exceeded")

func runWithTimeout(
    parentCtx context.Context,
    check domain.Check,
    spec domain.CheckSpec,
    checkTimeout time.Duration,
) domain.Result {
    runCtx, cancel := context.WithTimeoutCause(parentCtx, checkTimeout, errPerCheckTimeout)
    defer cancel()
    resultCh := make(chan domain.Result, 1)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                // Late-Recover-Pfad — Goroutine darf nicht abrauchen.
                // runCheckSafely fängt den Hauptpfad; dieser ist
                // defensive Tiefe.
                resultCh <- internalErrorResult(check)
            }
        }()
        resultCh <- check.Run(runCtx, spec)
    }()
    select {
    case res := <-resultCh:
        // Race-Härtung: wenn Context-End genau zur gleichen Zeit kam,
        // gewinnt die Context-Klassifikation gegen das Check-Result.
        if runCtx.Err() != nil {
            return classifyContextEnd(check, runCtx, checkTimeout)
        }
        return res
    case <-runCtx.Done():
        return classifyContextEnd(check, runCtx, checkTimeout)
    }
}

func classifyContextEnd(check domain.Check, runCtx context.Context, t time.Duration) domain.Result {
    switch {
    case errors.Is(runCtx.Err(), context.DeadlineExceeded):
        if errors.Is(context.Cause(runCtx), errPerCheckTimeout) {
            return timeoutResult(check, t)
        }
        return reconcileTimeoutResult(check)
    case errors.Is(runCtx.Err(), context.Canceled):
        return reconcileCanceledResult(check)
    default:
        // Soll nicht passieren — defensiv als InternalError.
        return internalErrorResult(check)
    }
}
```

Late-Result-Drops sind harmlos: `resultCh` ist gebuffered (Größe 1),
schreibender Goroutine blockt nicht; Reconciler hat das Result-
Ownership längst übergeben. Diese Mechanik ist Vorstufe der vollen
AR-009 §4 Worker-Pool-Variante; M5 macht sequenziell pro Check.

Reason-Code-Konstanten:

- `reasonCheckTimeout = "Timeout"` (eigener Per-Check-Deadline)
- `reasonReconcileTimeout = "ReconcileTimeout"` (Parent-Deadline)
- `reasonReconcileCanceled = "ReconcileCanceled"` (Cancel)
- `reasonInternalError = "InternalError"` (Panic-Pfad, §2.4)

### 2.6 Secret-Output-Filter als Pflicht-Hook

Verankerung statt Implementation: im MVP gibt es keine externen
Secrets (ADR 0010), also gibt es nichts zu filtern. Das
`application/secret_filter.go`-Modul exponiert **zwei** Hook-
Funktionen, die in M5 jeweils Identitäts-Stubs sind, in v0.2 aber
echte Maskierung übernehmen:

- `SanitizeMessage(msg string) string` — String-Pfad für
  `Result.Message`, `Condition.Message`, Event-Message-Texte.
- `SanitizeAttrs(attrs ...slog.Attr) []slog.Attr` — Attribut-Pfad
  für `slog.LogAttrs`. Iteriert die Attrs und maskiert
  Value-Inhalte (`slog.AnyValue`, `slog.StringValue` etc.) gemäß
  derselben Pattern wie `SanitizeMessage`. **Wichtig** (Review-
  Befund 3): strukturierte Observability darf nicht am Filter
  vorbei vertrauliche Werte loggen, daher braucht es einen
  Attribut-bewussten Hook neben dem reinen String-Hook.

Plus ein Wrapper für den heißen Pfad:

- `LogResult(logger *slog.Logger, level slog.Level, msg string,
  result domain.Result, extra ...slog.Attr)` — kombiniert beide
  Hooks: sanitized den Message-String, sanitized die Attrs (sowohl
  abgeleitete aus `Result` als auch zusätzliche aus `extra`), und
  ruft dann `logger.LogAttrs(level, …)`. Pflicht für jeden Log-Call,
  der Check-Result-Daten oder externe Inputs durchreicht (z. B.
  Panic-Stack, SAR-Error-Wrap, RBACCheckFailed-Detail).

**Pflicht-Aufrufstellen in M5:**

| Stelle | Hook | Begründung |
| ------ | ---- | ---------- |
| Vor jedem `Status().Update`-Call im Reconciler | `SanitizeMessage` auf `Condition.Message` und `Summary`-Felder | Status landet im CR und wird vom User gelesen |
| Panic-Recovery in `runCheckSafely` und Goroutine in `runWithTimeout` | `SanitizeAttrs` auf `slog.Any("recover", r)` und `slog.Any("stack", debug.Stack())` | recover-Wert ist `interface{}` aus dem Check, könnte beliebige Werte enthalten |
| RBACCheckFailed-Logger in §2.3 | `SanitizeAttrs` auf `slog.Any("err", err)` | SAR-Error könnte API-Response-Body mit Token-Echo enthalten |
| Per-Reconcile-Summary-Log am Ende von `Reconcile` | `LogResult` pro Result, sanitized Message + Attrs | konsolidiert die Result-Sicht für Oncall |
| Künftige K8s-Events (außerhalb M5) | `SanitizeMessage` | Events sind über `kubectl describe` sichtbar |

Tests in `secret_filter_test.go` fixieren das Aufruf-Pattern (Mock-
Sanitizer zählt Invocations und prüft, dass alle Pflicht-Stellen
durchlaufen). Zusätzlich verifiziert ein Konventions-Test, dass
keine direkten `logger.Info(..., "key", value)`-Calls mit
Result-/Check-Daten im neuen Code stehen — alle gehen über
`LogResult` oder explizite Sanitize-Calls.

**Erwartung an aktuellen M4-Reconciler-Logger-Stand:** der
existierende `logger.Info("reconciled", "name", req.Name, …)`-Call
in `Reconcile()` ist sicher (Namespace/Name sind keine Secrets,
Phase/Counters sind aggregierte Counts). M5 lässt diesen Call
unverändert; die Sanitize-Pflicht greift für die **neuen** Log-
Calls aus §2.3 (RBACCheckFailed), §2.4 (Panic-Recovery) und §2.6
(Per-Result-Log).

v0.2 ersetzt die Identität durch echte Secret-Pattern-Maskierung
(z. B. Bearer-Token-Muster, Base64-Blob-Heuristik); das Pflicht-
Hook-Pattern verhindert, dass Aufrufstellen vergessen werden.

### 2.7 Keine destruktiven Aktionen als depguard-Test

`LH-SEC-005` ist Konvention, kein Code. Slice-M5 verankert sie via
Code-Audit-Test in `internal/hexagon/application/destructive_audit_test.go`:
das Test-File parst die `+kubebuilder:rbac:`-Marker im Reconciler und
prüft, dass keine destruktiven Verben (`delete`, `deletecollection`,
`patch` auf fremde Ressourcen) gegen produktive OpenDesk-Ressourcen
existieren. CR/Status-Update-Verben sind whitelisted (LH-F-004).

### 2.8 Versionspins

| Komponente | Pin in M5 | Quelle |
| ---------- | --------- | ------ |
| `k8s.io/api/authorization/v1` | identisch zur transitiven Auflösung aus controller-runtime v0.24.1 | unverändert |
| `k8s.io/client-go/kubernetes` (Typed Clientset) | unverändert seit M4 | bereits Direct-Require |

---

## 3. Datei-Inventar

### 3.1 Neue Code-Dateien

| Pfad | Zweck |
| ---- | ----- |
| `internal/hexagon/domain/permission.go` | `PermissionRequest`-Struct (k8s-frei) + Helper `CanonicalString()` für Logging/Test-Vergleich. |
| `internal/hexagon/port/access_review.go` | `port.AccessReviewer`-Interface mit `CanI(ctx, PermissionRequest) (bool, error)`. |
| `internal/adapter/k8s/access_review.go` | `AccessReviewAdapter` implementiert `port.AccessReviewer` via `kubernetes.Interface.AuthorizationV1().SelfSubjectAccessReviews().Create`. Mapping `PermissionRequest` → `authorizationv1.ResourceAttributes`. |
| `internal/hexagon/application/runner.go` | Sequentielle Check-Runner-Helper `runWithTimeout(ctx, check, spec, timeout)` mit per-Check-Recover (slice-M5 §2.4) und 30s-Default-Timeout (§2.5). Verlagert die `runChecks`-Logik aus `reconciler.go` und ergänzt SAR-Pre-Execution-Pfad (§2.3). |
| `internal/hexagon/application/secret_filter.go` | Zwei Identitäts-Hooks plus ein Wrapper für M5 (§2.6): `SanitizeMessage(msg string) string` für String-Pfade (Status, Events), `SanitizeAttrs(attrs ...slog.Attr) []slog.Attr` für strukturierte `slog`-Attribute, und `LogResult(logger, level, msg, result, extra...)` als Pflicht-Aufruf für Result-bezogene Logs. Implementierungen sind in M5 Identität, v0.2 hebt sie auf echte Pattern-Maskierung. |
| `internal/hexagon/application/destructive_audit_test.go` | Code-Audit-Test gegen `+kubebuilder:rbac:`-Marker im Reconciler, verifiziert `LH-SEC-005` (§2.7). |
| `internal/hexagon/application/rbac_consistency_test.go` | Konsistenz-Test (Review-Befund 4): vergleicht die `+kubebuilder:rbac:`-Marker am Reconciler gegen die `RequiredPermissions()` aller registrierten Checks. Fehler bei Drift in beide Richtungen. Shared `go/parser`-Helper mit `destructive_audit_test.go`. |

### 3.2 Erweiterte Code-Dateien

| Pfad | Änderung |
| ---- | -------- |
| `internal/hexagon/domain/check.go` | `Check`-Interface um `RequiredPermissions() []PermissionRequest` erweitert (§2.2). |
| `internal/adapter/check/kubernetesversion.go` | `RequiredPermissions()` returniert leere Slice (Discovery). |
| `internal/adapter/check/certmanager.go` | analog leer. |
| `internal/adapter/check/storageclass.go` | `RequiredPermissions()` returniert `[{Group:"storage.k8s.io", Resource:"storageclasses", Verb:"list"}]`. |
| `internal/adapter/check/ingressclass.go` | analog `networking.k8s.io/ingressclasses`. |
| `internal/adapter/check/clusterresources.go` | analog core/nodes. |
| `internal/hexagon/application/reconciler.go` | `Reconciler` bekommt neue Felder `AccessReviewer port.AccessReviewer` und `CheckTimeout time.Duration` (Default 30s wenn 0). `runChecks` ruft auf den neuen `runner`. Outer-defer-Recover am Anfang von `Reconcile` (§2.4). `SanitizeMessage`-Aufrufe an den drei Pflicht-Stellen (§2.6). |
| `cmd/operator/main.go` | Wiring: `k8s.NewAccessReviewAdapter(clients.Clientset)` und Inject in den Reconciler. |
| `internal/adapter/check/registry_test.go` (Erweiterung) | Tests aktualisiert, da `Check`-Interface eine neue Methode bekommt. Stubs implementieren `RequiredPermissions()` (meist leer). |
| `internal/hexagon/application/reconciler_test.go` (Erweiterung) | Vier neue Fälle: RBACInsufficient-Pfad, Panic-Recovery-per-Check, Per-Check-Timeout, Reconciler-Outer-Panic. |

### 3.3 Neue Test-Dateien

| Pfad | Coverage |
| ---- | -------- |
| `internal/hexagon/domain/permission_test.go` | `PermissionRequest`-Konstruktion, `CanonicalString()`-Output stabil über Field-Permutation. |
| `internal/adapter/k8s/access_review_test.go` | `AccessReviewAdapter` mit `fake.NewSimpleClientset` + `PrependReactor` auf `create selfsubjectaccessreviews`: allowed/denied/error-Pfade. |
| `internal/hexagon/application/runner_test.go` | Tabelle: passed (Check liefert True), failed-Check (False/critical), panic-im-Check (recover → Unknown/InternalError), timeout (kontextueller Hänger → Unknown/Timeout). |
| `internal/hexagon/application/secret_filter_test.go` | Identitäts-Verhalten in M5; Aufruf-Pflicht-Pattern via wrapped-Sanitizer-Mock. |
| `internal/adapter/check/{name}_permission_test.go` (5 Stück oder ein gemeinsames `permissions_test.go`) | Tabellen-Test pro Check: erwartete RequiredPermissions exakt. |

### 3.4 RBAC-Manifest-Refresh

`+kubebuilder:rbac:`-Marker am Reconciler haben bereits in M2 alle
M5-relevanten Verben:

- `authorization.k8s.io/selfsubjectaccessreviews,verbs=create`
- `authorization.k8s.io/selfsubjectrulesreviews,verbs=create`
- alle list-Verben für die fünf MVP-Check-Ressourcen

`make manifests` muss laufen, sollte aber keinen Drift erzeugen
(`config/rbac/role.yaml` bleibt unverändert). Generated-Drift-Gate
verifiziert.

---

## 4. Reihenfolge der Umsetzung

Jeder Schritt = ein eigener Commit; lokal `make gates` grün ziehen.

1. **Domain + Port + RequiredPermissions-Interface.** `domain/permission.go`,
   `port/access_review.go`. `Check`-Interface erweitert; alle fünf
   M4-Checks bekommen `RequiredPermissions()`. Compile-Bruch erwartbar
   (Stubs in Tests müssen die neue Methode implementieren) — gleicher
   Commit aktualisiert die Test-Stubs.
2. **Adapter.** `adapter/k8s/access_review.go` mit Mapping
   `PermissionRequest` → `ResourceAttributes`. Adapter-Test gegen fake
   clientset.
3. **Application-Runner + Secret-Filter.** Neuer
   `application/runner.go` mit `runWithTimeout` + Per-Check-Recover
   + SAR-Pre-Execution. Neuer `application/secret_filter.go`
   (Identität). Tests.
4. **Reconciler-Erweiterung.** Outer-Recover, SAR-Wiring,
   `runChecks` ruft `runner`. SanitizeMessage-Aufrufe.
   Reconciler-Tests für die vier neuen Pfade (RBACInsufficient,
   Per-Check-Panic, Per-Check-Timeout, Outer-Panic).
5. **Destructive-Audit-Test.** Code-Audit-Test parst Reconciler-RBAC-
   Marker und verifiziert Allowlist.
6. **Wiring.** `cmd/operator/main.go` zieht `AccessReviewAdapter`
   ein. Lokal `make cluster-smoke` grün — der existierende Smoke-CR
   (M4) muss weiter `Phase=Passed` liefern, weil unsere
   ClusterRole-RBAC alle nötigen Rechte gewährt.
7. **Slice-Closure.** Nach `done/` ziehen, Roadmap-Status M5 = Done,
   Closure-Notiz mit Verifikations-Ergebnis und CI-Run-URLs.

---

## 5. Lastenheft-Kennungen

`LH-F-024` (RBAC-Selbstprüfung), `LH-F-031` (Schweregrad — pro
Result), `LH-F-032` (Ergebnis-Inhalt), `LH-NF-004` (Stabilität —
Operator stürzt nicht ab), `LH-NF-005` (Fehlertoleranz —
Einzelausfälle stoppen nicht), `LH-NF-006` (Minimalrechte —
operativ verankert via §2.2), `LH-NF-007` (Datenschutz — Stack-
Traces bleiben im Logger), `LH-SEC-001` (Minimalrechte), `LH-SEC-002`
(Keine Secret-Ausgabe), `LH-SEC-005` (Keine destruktiven
Aktionen), `LH-DAT-007` (Secret-Referenzierung — Konvention
vorbereitet), `LH-AK-010` (Fehlerfall robust), `LH-AK-012` (Keine
Secret-Leaks), `LH-AK-015` (Minimalrechte dokumentiert), `LH-AK-016`
(RBAC-Selbstprüfung wirksam).

---

## 6. Architekturartefakte

Erfüllt aus `spec/architecture.md`:

- `AR-009 §4 Step 4` Per-Check-Timeout (sequentiell, ohne
  Worker-Pool) — scharf.
- `AR-011` Error-Handling und Fehlertoleranz, Fehler-Präzedenz
  `ReconcileTimeout > ReconcilePanic > Check-InternalError` — scharf.
- `AR-018` SelfSubjectAccessReview-Right operativ verankert.
- `AR-014` Severity-Aggregation für Unknown-Results (M3 bereits
  vorhanden, M5 testet den Unknown-Pfad mit RBACInsufficient).

Vorbereitet, aktiv ab späterer Slice:

- `AR-009 §4` Worker-Pool voll — v0.2.
- `AR-010` Wiederholintervall — M6.
- `AR-010.1` OPERATOR_STRICT_CONFIG + Env-Override für Timeouts —
  v0.2.
- `AR-026` Leader-Election — M7.

---

## 7. Verifikation (Abnahmekriterien)

1. **`make build`** baut den erweiterten Reconciler + Runner +
   AccessReviewAdapter.
2. **`make lint`** grün — `Check`-Interface-Erweiterung bricht keine
   `AR-005`-depguard-Regel (domain/permission.go bleibt k8s-frei).
3. **`make test`** grün; alle neuen Unit-Tests bestehen.
4. **`make coverage-gate`** grün bei Threshold 90 % (slice-M4
   vorgezogen). Erwartung: Coverage bleibt ≥ 90 %; der neue
   `adapter/k8s/access_review.go` wird via `fake.NewSimpleClientset`
   getestet, fällt also nicht aus dem strikten Schnitt.
5. **`make doc-refs`** grün.
6. **`make generated-drift-check`** grün — keine RBAC-Marker-
   Änderungen (M2-Pre-Grant deckt M5-Bedarf 1:1).
7. **`make gates`** grün (Bundle).
8. **`make security-gates`** grün — `govulncheck` ohne Findings
   nach `k8s.io/api/authorization/v1` Direct-Require-Promotion.
9. **`LH-AK-010` Fehlerfall robust** — `application/runner_test.go`
   deckt Panic-Pfad ab (Check panickt → Result Unknown/InternalError;
   andere Checks im selben Reconcile laufen weiter).
   Reconciler-Outer-Panic via `reconciler_test.go` (Phase=Unknown,
   Condition=ReconcileError, Reason=ReconcilePanic).
10. **`LH-AK-012` Keine Secret-Leaks** — `secret_filter_test.go`
    fixiert das Aufruf-Pattern für **beide** Hooks: `SanitizeMessage`
    auf Status-/Event-Pfaden, `SanitizeAttrs` auf allen neuen
    `slog.LogAttrs`-Calls (Panic-Recovery in §2.4, RBACCheckFailed in
    §2.3, Per-Result-Summary in §2.6). Mock-Sanitizer zählt
    Invocations und prüft, dass alle Pflicht-Stellen durchlaufen.
    Konventions-Test verifiziert zusätzlich, dass keine
    `logger.Info(..., key, value)`-Calls mit Result-/Check-Daten am
    Filter vorbei gehen.
11. **`LH-AK-015` Minimalrechte dokumentiert** — Konsistenz zwischen
    den `RequiredPermissions`-Returns (§2.2) und den
    `+kubebuilder:rbac:`-Markern am Reconciler ist via
    `rbac_consistency_test.go` **automatisiert** geprüft
    (Review-Befund 4). Drift in beide Richtungen bricht den Build.
12. **`LH-AK-016` RBAC-Selbstprüfung wirksam** — `runner_test.go`
    deckt **beide** SAR-Outcomes ab (Review-Befund 1):
    - `CanI`→`(false, nil)` → Result Unknown/RBACInsufficient,
      andere Checks laufen weiter.
    - `CanI`→`(_, err)` → Result Unknown/**RBACCheckFailed**, mit
      Pflicht-Logger-`ERROR`-Eintrag.
    Plus: Panic in `CanI` → Result Unknown/InternalError (Review-
    Befund 3), Reconciler-Outer-Recover läuft NICHT.
    Plus: Per-Check-Timeout-Diskriminierung (Review-Befund 2) —
    `runCtx.Err()==DeadlineExceeded` → Reason `Timeout`,
    `runCtx.Err()==Canceled` → Reason `ReconcileCanceled` mit
    Severity `info`.
    Real attestiert via Cluster-Smoke: der Smoke-CR bleibt
    `Phase=Passed`, weil die ClusterRole alle nötigen Rechte
    gewährt; ein gesondertes Smoke-Szenario mit künstlich
    eingeschränkter RBAC ist nicht im Scope (M6-Manual-Test).

---

## 8. Out-of-Scope (geht in M6/M7/v0.2)

- **Wiederholintervall** (`AR-010`, `LH-F-025`) — M6 zusammen mit
  Anwender-Doku.
- **Leader-Election** (`AR-026`) — M7-Release-Hardening.
- **Worker-Pool voll** (`AR-009 §4`) — v0.2.
- **`OPERATOR_STRICT_CONFIG` + `CHECK_TIMEOUT_SECONDS`-Env-Override**
  (`AR-010.1`) — v0.2 zusammen mit der Configuration-Klassifizierungs-
  ADR.
- **Echte Secret-Pattern-Maskierung** in `SanitizeMessage` — v0.2,
  sobald externe Service-Checks (`LH-F-018..021`) reale Secrets
  liefern (`ADR 0010`).
- **Cluster-Smoke-Szenario mit eingeschränkter RBAC** (Operator-
  ServiceAccount ohne `storage.k8s.io/storageclasses,verbs=list`) —
  M6-Manual-Test.
- **K8s-Events schreiben** (`LH-F-027`) — v0.2.

---

## 9. Risiken und Mitigation

- **SAR-Endpoint-Verfügbarkeit / transienter Subsystem-Ausfall**
  (Review-Befund 1): Cluster ohne `authorization.k8s.io`-API
  (theoretisch — alle k8s-Versionen ≥ 1.10 haben sie) oder mit
  transientem Auth-Webhook-Fehler würden ohne saubere Unterscheidung
  als „RBAC fehlt" erscheinen und echte Infrastrukturprobleme
  verschleiern. Mitigation: `CanI`-Fehler werden als separater
  Reason `RBACCheckFailed` klassifiziert (§2.3), nicht als
  `RBACInsufficient`. Operator-Log schreibt `ERROR` mit Error-Wrap,
  damit Oncall transient von permanent unterscheiden kann.
- **Cache-Miss bei Permission-Mutation während eines Runs:** wenn
  der Cluster-Admin mitten im Reconcile RBAC-Rechte entzieht,
  könnten gecachte `true`-Werte stale sein. Akzeptiert für den MVP
  — die Reconcile-Dauer ist ≤ 150 s; wird der Pfad in v0.2-Worker-
  Pool relevant, kann ein TTL eingezogen werden.
- **Per-Check-Goroutine leakt nach Timeout:** AR-009 §4
  beschreibt, dass nicht-kooperative Checks ihre Goroutine
  weiterlaufen lassen. Im MVP sind alle Checks kooperativ (sie
  reichen `ctx` durch). Defensive Maßnahme: `runCheckWithTimeout`
  bricht den parent-Context ab, der Goroutine sieht das. Worker-
  Leaks bleiben theoretisch möglich, sind aber von M5-`runner`-
  Tests abgedeckt.
- **Go-`select`-Race im `runWithTimeout`-Wrapper** (Folge-Review-
  Befund 1): Wenn `check.Run` exakt zum Timeout-Zeitpunkt fertig
  wird, ist die Auswahl zwischen `<-resultCh` und `<-runCtx.Done()`
  nicht-deterministisch. Mitigation: nach `<-resultCh` validieren
  wir zusätzlich `runCtx.Err()` — bei Context-End gewinnt die
  Context-Klassifikation. Damit ist die Reason-Klassifikation
  unabhängig vom exakten Scheduler-Verhalten.
- **Strukturierte slog-Attrs am Sanitize-Filter vorbei** (Folge-
  Review-Befund 3): `SanitizeMessage(msg string) string` alleine
  fängt keine Werte aus `slog.Any(...)`-Attrs. Mitigation: zweiter
  Hook `SanitizeAttrs(...slog.Attr) []slog.Attr` plus ein
  `LogResult`-Wrapper als Pflicht-Aufruf. Konventions-Test verbietet
  direkte `logger.Info(..., key, value)`-Calls mit
  Result-/Check-Daten.
- **Panic-Recovery maskiert echte Programmierfehler:** ein recover
  in `runner.go` verhindert, dass ein Panic den Operator beendet
  — gut für `LH-AK-010` Robustheit, schlecht für
  Programmierfehler-Sichtbarkeit. Mitigation: Stack-Trace ins
  Logger schreiben (mit slog.Error + slog.Any("stack", ...)); ein
  externer Operator-Watcher (oncall) sieht das.
- **`destructive_audit_test.go`-False-Positive bei
  Marker-Whitespace:** das Test-File parst Go-Comments mit Regex
  — Whitespace-Varianten könnten zu Fehl-Hits führen. Mitigation:
  Test-File verwendet `go/parser` für AST-genaues Marker-Lesen,
  nicht regex.
- **`RequiredPermissions()`-Drift gegen `kubebuilder:rbac`-Marker:**
  ein neu hinzugefügter Check könnte ein Recht deklarieren, das im
  Reconciler-Marker fehlt — dann erteilt die ClusterRole das Recht
  nicht, und SAR liefert false. Mitigation: Slice-Plan §2.2-Tabelle
  ist Pflicht-Doku; M6-Anwender-Doku-Eintrag erinnert daran.

---

## 10. Closure

— bleibt leer bis zum Schließen der Slice.
