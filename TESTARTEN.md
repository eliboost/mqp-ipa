# Testarten für die Kargo-Verifikation von MQP

Welche Tests als Kargo-Verifikation für MQP wirklich etwas bringen (Stand 24.10.2026). Grundlage sind der Code der MQP-Repos, der Code von ArkMQ v2.1.7 / v2.2.2, Kargo v1.12.4 (Doku und Quellcode) und die Doku von Argo Rollouts. Ergänzt [NOTIZEN.md §9](NOTIZEN.md#9-tests-für-die-operatoren) und [VARIANTENENTSCHEIDE.md §2.1](VARIANTENENTSCHEIDE.md#21-testarten-bewerteter-variantenentscheid).

**Einordnung:** Die Teststrategie ist das Thema der **Probe-IPA**. Entwickelt und nachgewiesen wird sie auf den MQP-Testclustern **Lab-D → Lab-C** mit der Kargo-Test-Instanz. In der IPA wird sie aus Git für E1 und T1 übernommen und ist dort Ausgangslage. Was nur E1/T1 betrifft, ist unten als Ausblick markiert.

> Mit Unterstützung von KI (Claude) erstellt. Für die IPA deklarieren (A02 Punkt 3, Doc07 Punkt 6).

## Inhalt

1. [Kurzfassung](#1-kurzfassung)
2. [Was im Code auffällt und die Tests bestimmt](#2-was-im-code-auffällt-und-die-tests-bestimmt)
3. [Wie Kargo und Argo Rollouts verifizieren](#3-wie-kargo-und-argo-rollouts-verifizieren)
4. [Die Testarten im Detail](#4-die-testarten-im-detail)
5. [Zusammenspiel und Reihenfolge](#5-zusammenspiel-und-reihenfolge)
6. [Korrekturen an den bisherigen Notizen](#6-korrekturen-an-den-bisherigen-notizen)
7. [Offene Fragen](#7-offene-fragen)

---

## 1. Kurzfassung

| # | Testart | Wo in Kargo | Priorität | Warum |
|---|---|---|---|---|
| K | **Kompatibilität der Kombination** | vor dem Commit (`fail`-Step in der Promotion) | unbedingt | ArkMQ und Broker-Version sind enger gekoppelt als gedacht (§2.1) |
| V | **Version angekommen, Rollout fertig** | Verifikation, Job | unbedingt | Der Status des PF-Operators kann veraltet sein (§2.2). Ohne V prüfen alle anderen Tests eventuell noch den alten Stand |
| S | **Status aller Instanzen ohne Regression** | Verifikation, Job | unbedingt | Operator-Updates betreffen alle Broker im Cluster |
| F | **Nachrichtenfluss (artemis-monitor)** | Verifikation, Prometheus | unbedingt | Einziger Test, der zeigt, dass Broker wirklich Nachrichten verarbeiten |
| O | **Operator stabil (kein Dauer-Reconcile)** | Verifikation, Prometheus | sehr gut | Fängt den `Mismatch`-Dauerzustand und Crash-Loops ab, die bei „Ready“ durchrutschen können |
| C | **Canary-Instanz anlegen und löschen** | Verifikation, Job | sehr gut (Lab-D, Lab-C; später E1) | Einziger Test für den Weg „neue Instanz“ inklusive Zertifikat und Löschen |
| M | Management-Zugriff (Jolokia, OPS UI) | Verifikation, Web | optional | Teilweise schon durch S abgedeckt |
| U | Upgrade-Pfad n-1 → n | eigene Stage | Ausblick | Braucht den P1-Stand in einem Cluster |

**Kern der Empfehlung:** K, V, S und F sind das Minimum, das bei MQP wirklich schützt. O und C machen daraus eine starke Probe-IPA (Variante C). Gegenüber Variante C in den Notizen verschiebt sich der Schwerpunkt: Der neue Test **V** ist wichtiger als die Canary-Instanz.

---

## 2. Was im Code auffällt und die Tests bestimmt

### 2.1 ArkMQ und Broker-Version müssen exakt zusammenpassen

Der PF-Operator setzt im `ActiveMQArtemis` zwar `image` und `initImage`, aber **nicht `spec.version`** (`pf-activemq-operator/controller/resource/artemismq.go`). ArkMQ verhält sich dann so (Code v2.2.2):

1. `ResolveBrokerVersionFromCR` nimmt bei leerer Version die **neueste Version aus der eigenen Liste** (`pkg/utils/common/common.go:210-218`, `version/version.go`).
2. `AssertBrokerImageVersion` fragt über Jolokia die laufende Broker-Version ab und vergleicht sie **exakt** mit dieser Version (`controllers/activemqartemis_reconciler.go:3789-3808`). Bei einer Abweichung wird `BrokerVersionAligned=False`.
3. `Ready` ist nur `True`, wenn keine andere Condition `False` ist (`pkg/utils/common/conditions.go:32-46`).
4. Der PF-Operator setzt `ActiveMQBrokersReady` nur auf `True`, wenn ArkMQ `Ready=True` meldet.

Die neueste Version in der Liste von ArkMQ passt genau zum heutigen Stand:

| ArkMQ | neueste Artemis-Version in `version.go` | Broker heute |
|---|---|---|
| v2.1.7 | 2.53.0 | P1: `artemis.2.53.0` |
| v2.2.2 | 2.57.0 | E1/T1/Lab: `artemis.2.57.0` |

**Folge (aus dem Code abgeleitet, in Lab bestätigen):** Ein neuer Broker (z. B. 2.57.1 oder 2.58.0) ohne passendes ArkMQ-Release führt zu `BrokerVersionAligned=False` und damit zu `ActiveMQBrokersReady=False` auf **allen** Instanzen im Cluster. Umgekehrt gilt das auch für ein neues ArkMQ ohne neuen Broker. ArkMQ und Broker müssen also gemeinsam promotet werden. Das ist ein starkes Argument für Freight-Variante A und eine konkrete, automatisch ableitbare Kompatibilitätsregel.

### 2.2 Der Status des PF-Operators kann veraltet sein

- Der PF-Operator setzt in seinen Conditions **kein `observedGeneration`** (`controller/pfactivemq_controller.go`, alle `SetStatusCondition`-Aufrufe). Ein `Ready=True` direkt nach der Promotion kann noch vom alten Operator stammen.
- Die Condition `Ready` hat auch bei `Status: False` den Reason `Ready` (`pfactivemq_controller.go:351-356`, ähnlich `:445-450`). **Tests müssen `status` prüfen, nie `reason`.**
- `AuthSecretReady` und `StorageReady` fliessen nicht in `Ready` ein (`allConditionsReady`, `:371-391`).
- `os.Exit(1)` bei falschem Init-Image passiert erst beim Reconcile (`artemismq.go:52-55`). Ohne Instanzen im Cluster würde der Operator sogar „healthy“ bleiben.

Ein reiner Check „Ready=True“ reicht deshalb nicht. Es braucht zuerst den Nachweis, dass die neue Version tatsächlich läuft (Test V).

### 2.3 Gute Ansatzpunkte im Code

- `ActiveMQArtemis.status.version` enthält `brokerVersion`, `image` und `initImage` (ArkMQ `api/v1beta1/activemqartemis_types.go:731-741`). Damit lässt sich prüfen, ob die neue Version angekommen ist.
- PF- und Merge-Operator liefern Controller-Runtime-Metriken (`prometheus.io/port: 8080`, `/metrics`).
- Der `artemis-monitor` liefert `monitor_messages_sent_total`, `_consumed_total`, `_consumed_errors_total`, `_delayed_total` und `monitor_e2e_duration` (`artemis-monitor/.../Metrics.java`). `delayed` zählt auch Nachrichten, die gar nicht gesendet werden konnten: Sie landen zuerst im Cache und laufen nach 10 s ab (`MessageCache.java`, `application.message.sla.delay=10000`).
- Prometheus hat die Labels `kubernetes_cluster_name` und `kubernetes_namespace` (Beispiel in `documentation/docs/howto/alarming.md`).
- Der Namespace `appl-mqp-test` existiert auf Lab-D und E1 schon und ist leer (`deployments/appl-mqp-test`). Er eignet sich für die Canary-Instanz. Auf Lab-C muss er für die Probe-IPA angelegt werden.

---

## 3. Wie Kargo und Argo Rollouts verifizieren

Aus der Kargo-Doku und dem Kargo-Quellcode v1.12.4:

- **Ablauf:** Nach der Promotion wartet Kargo, bis die Argo-CD-Apps `Healthy` sind. Danach erzeugt Kargo aus allen `analysisTemplates` der Stage **eine** AnalysisRun. Ausgewertet wird sie vom **Argo-Rollouts-Controller** im Kargo-Cluster. Kargo übernimmt nur die Phase.
- **Phasen:** `Successful` markiert die Freight als verifiziert. `Failed`, `Error`, `Aborted` und **`Inconclusive`** sind Endzustände **ohne** Verifikation, die Freight bleibt also blockiert (`api/v1alpha1/stage_types.go:46-55`). Neu starten lässt sich die Verifikation mit `kargo verify stage`.
- **Metriken laufen parallel.** Alle Metriken einer AnalysisRun starten gleichzeitig. Eine Reihenfolge entsteht nur über `initialDelay` oder dadurch, dass ein Job selbst wartet.
- **Argumente aus der Freight:** In den `args` der Stage stehen auch in der Verifikation `imageFrom()`, `commitFrom()` und `warehouse()` zur Verfügung (`regular_stages.go`, `exprfn.FreightOperations`). Damit kann der Test die **erwartete** Version aus der Freight bekommen, z. B. `${{ imageFrom('middleware-docker-local.repo.pnet.ch/pf/middleware/pf-activemq-operator').Tag }}`.
- **`dryRun`** wird aus dem Template übernommen. So kann eine Metrik messen, ohne zu blockieren.
- **Provider in Open Source:** `job` (Erfolg = Exit-Code 0), `prometheus`, `web` (mit `jsonPath`, `successCondition`/`failureCondition`).
- **Aufräumen:** Laut den bestehenden Notizen zu Kargo (`test.txt`) räumt der Kargo-GC AnalysisRuns, Jobs und Pods **nicht** auf. In jedem Job `ttlSecondsAfterFinished` setzen.

**Regeln, damit kein `Inconclusive` entsteht:**

- Jede Prometheus- oder Web-Metrik braucht **beide** Bedingungen, `successCondition` und `failureCondition`.
- Jeder Job braucht `activeDeadlineSeconds` und `backoffLimit: 0` oder `1`. Ein Image-Pull-Fehler ergibt sonst `Inconclusive`.
- Bei `count` > 1 `failureLimit` bewusst setzen. Der Standard ist 0, also scheitert die Metrik schon beim ersten Fehlschlag.

---

## 4. Die Testarten im Detail

### K: Kompatibilität (vor der Promotion)

| | |
|---|---|
| Prüft | Broker = Init (Major.Minor); Broker-Version = neueste Version in der ArkMQ-Liste (§2.1); PF-Operator für diese ArkMQ-Version freigegeben |
| Fängt ab | `os.Exit(1)` des PF-Operators; `BrokerVersionAligned=False` auf allen Instanzen; PF-Operator gegen falsche ArkMQ-API |
| Umsetzung | Matrix als ConfigMap im Kargo-Projekt, am Anfang der Promotion ein `fail`-Step mit `if:` auf `configMap('mqp-compat-matrix')` (ArkMQ → Artemis) und auf Broker = Init (Major.Minor). Läuft in jeder Stage, auch bei `kargo approve` |
| Probe-IPA / IPA | In der Probe-IPA ist K die einzige Prüfung vor dem Commit, weil das vorläufige Warehouse keine Regel hat. In der IPA kommt die Regel ins Warehouse (Freight-Modell), K bleibt als zweite Sicherung |
| Vorteil | Es wird nichts deployt, wenn die Kombination ungültig ist. Eine Verifikation käme erst nach dem Schaden |
| Matrix pflegen | Die Spalte ArkMQ → Artemis lässt sich aus `version/version.go` des ArkMQ-Release ablesen und später automatisch erzeugen |

### V: Version angekommen, Rollout fertig (Job)

| | |
|---|---|
| Prüft | 1) Die Pods von PF-Operator und ArkMQ-Operator laufen mit dem Image aus der Freight. 2) Jedes `ActiveMQArtemis` mit Label `pf-activemq-operator.pnet.ch/operator-managed=true` hat `spec.deploymentPlan.image` und `status.version.image` = erwartetes Broker-Image sowie `status.version.brokerVersion` = erwartete Version. 3) Jedes StatefulSet ist fertig ausgerollt (`updatedReplicas == replicas`, `currentRevision == updateRevision`) |
| Fängt ab | veralteten Status (§2.2); hängende Rolling Updates; Broker, die noch mit dem alten Image laufen |
| Umsetzung | Job mit `kubectl`/`jq`, der **pollt**, bis alles stimmt, mit Timeout über `activeDeadlineSeconds` (z. B. 30 min, je nach Anzahl Broker). Erwartete Versionen als `args` aus der Freight |
| Hinweis | Das ist die Grundlage für S, F und O. Ohne V messen sie eventuell noch den alten Stand |

### S: Status aller Instanzen ohne Regression (Job)

| | |
|---|---|
| Prüft | Alle `PFActiveMQ`: `Ready` und `ActiveMQBrokersReady` stehen auf `status: "True"`. Alle `ActiveMQArtemis`: `Ready`, `Valid`, `BrokerVersionAligned`, `BrokerPropertiesApplied` stehen auf `True` |
| Fängt ab | CRD-/API-Inkompatibilität, Versionskonflikt ArkMQ/Broker, Fehler in der Broker-Konfiguration |
| Problem | Eine Instanz eines Teams, die schon **vorher** kaputt war, würde das Gate dauerhaft blockieren |
| Lösung | Als Regression zählt nur eine Condition, die **nach dem Start des neuen Operator-Pods** auf `False` gewechselt hat (`lastTransitionTime` > `startTime` des Pods). Bereits kaputte Instanzen werden geloggt, blockieren aber nicht. Die MQP-eigenen Instanzen (e2e, Canary) blockieren immer |
| Hinweis | Wegen §2.2 nur auf `status` prüfen, nie auf `reason` |

### F: Nachrichtenfluss mit artemis-monitor (Prometheus)

| | |
|---|---|
| Prüft | Nachrichten werden gesendet und empfangen, keine verspäteten oder verlorenen Nachrichten, Latenz im SLA |
| Abfragen (Skizze) | `sum(rate(monitor_messages_consumed_total{kubernetes_cluster_name="{{args.cluster}}"}[2m])) > 0`; `sum(increase(monitor_messages_delayed_total{...}[5m])) == 0`; `sum(increase(monitor_messages_consumed_errors_total{...}[5m])) == 0`; `monitor_e2e_duration{quantile="0.99"} < 2` |
| Einstellungen | `initialDelay` grösser als Rollout-Zeit + Messfenster (im Rolling Update sind verspätete Nachrichten normal); `interval: 1m`, `count: 5`, `failureLimit: 1` |
| Fängt ab | Broker läuft und ist „Ready“, verarbeitet aber nichts (TLS, Acceptor, Auth, Journal) |
| Voraussetzung | Prometheus muss vom Kargo-Cluster aus abfragbar sein, Cluster-Name als Stage-`vars` |
| **Pflicht: Cluster-Filter** | Jede Abfrage filtert fest auf `kubernetes_cluster_name="{{args.cluster}}"`. Lab-D und E1 haben denselben Namespace (`appl-mqp-e2e-e1`) und Instanznamen (`e1-mqp-e2e-a`). Das Grafana-Dashboard „Artemis E2E Monitor“ mischt deshalb nachweislich beide Cluster (§6, Punkt 6). Ohne Filter könnte eine Stage grün sein, weil die Werte von einem anderen Cluster kommen |
| Vorlage | Die PromQL der Panels im Dashboard „Artemis E2E Monitor“ (Grafana → Inspect → Query) übernehmen und um den Cluster-Filter ergänzen. Heutige Normalwerte auf E1: Verfügbarkeit 100 %, Delayed 0, Latenz p99 etwa 10–40 ms |

### O: Operator stabil (Prometheus)

| | |
|---|---|
| Prüft | Keine Neustarts der Operator-Pods und keine Reconcile-Fehler im Messfenster, und die Rate der Requeues steigt nicht |
| Abfragen (Skizze) | `sum(increase(kube_pod_container_status_restarts_total{namespace=~"kube-pf-activemq|kube-artemis-activemq",kubernetes_cluster_name="{{args.cluster}}"}[10m])) == 0`; `sum(increase(controller_runtime_reconcile_errors_total{controller="pfactivemq",...}[10m])) == 0` |
| Fängt ab | `Mismatch`-Dauerzustand (DeepEqual nie gleich, Requeue alle 5 s, `pfactivemq_controller.go:100-108`), Crash-Loop durch `os.Exit(1)`, Speicherlecks |
| Hinweis | Labelnamen (`controller`, Cluster-Label von kube-state-metrics) in Grafana prüfen. Zuerst in Lab mit `dryRun` laufen lassen |

### C: Canary-Instanz (Job, Lab-D und Lab-C; später E1)

| | |
|---|---|
| Ablauf | 1) `PFActiveMQ` mit Namen pro Freight in `appl-mqp-test` anlegen. 2) Warten auf `Ready=True`. 3) Eine Nachricht senden und empfangen (z. B. mit dem Artemis-CLI im Broker-Image). 4) Löschen und prüfen, dass Zertifikat, Secrets und ConfigMap weg sind und der Finalizer entfernt wurde |
| Fängt ab | Neue Instanzen entstehen nach dem Upgrade nicht mehr korrekt; Probleme mit Zertifikat/cert-manager; hängende Finalizer beim Löschen (`DeletionReconciler`) |
| Vorlage | Eine echte Instanz-Definition aus `activemq/instances` (z. B. die e2e-Instanz), **nicht** `testfiles/testrun/single.yaml` (siehe §6) |
| Dauer | Zertifikat + Broker-Start + Löschen, in Lab messen |
| Hinweis | Läuft am Merge-Operator vorbei. Das ist in einem MQP-eigenen Namespace vertretbar, gehört aber in die Abgrenzung. In der Probe-IPA auf beiden Testclustern ohne Absprache möglich. In der IPA auf E1 nur nach Absprache, auf T1 weglassen, siehe [VARIANTENENTSCHEIDE §2.1](VARIANTENENTSCHEIDE.md#21-testarten-bewerteter-variantenentscheid) |

### M: Management-Zugriff (optional)

ArkMQ braucht Jolokia für `BrokerVersionAligned` und `BrokerPropertiesApplied`, die OPS UI braucht Jolokia für alle Aktionen. Der Weg über Jolokia ist durch **S** schon indirekt abgedeckt. Ein zusätzlicher `web`-Test gegen `GET /v1/instance` der OPS UI (no-auth auf E1) würde zeigen, dass auch die OPS UI nach dem Upgrade noch funktioniert. Das ist sinnvoll, wenn bei einem Artemis-Update Änderungen an der Jolokia-API erwartet werden.

### U: Upgrade-Pfad (Ausblick)

Braucht eine Stage mit dem P1-Stand (2.53 / v2.1.7 / 0.17.1), z. B. einen vCluster oder eine frisch aufgesetzte Lab-Umgebung. Für die IPA nur als nächster Schritt (D03).

---

## 5. Zusammenspiel und Reihenfolge

```
Promotion:  K (fail-Step)  →  ArkMQ-App sync  →  argocd-wait  →  PF-App sync
                                                                   │
Kargo wartet auf Argo-CD-Health (Operator-Deployments)  ◄──────────┘
                                                                   │
AnalysisRun (alle Metriken starten parallel):                      ▼
  V  Job: pollt bis Version da + Rollout fertig      (0 … ~30 min)
  S  Job: wartet selbst auf V-Kriterien, prüft dann Conditions
  C  Job: legt Canary an, nachdem V-Kriterien erfüllt sind   (nicht auf T1)
  F  Prometheus: initialDelay ≥ Rollout + 5 min, dann 5 × 1 min
  O  Prometheus: initialDelay ≥ Rollout + 10 min, dann 1 Messung
```

Mein Vorschlag ist, V, S und C als **einen Job mit drei Phasen** umzusetzen. Dann ist die Reihenfolge garantiert, und das Protokoll liegt an einem Ort. Die beiden Prometheus-Metriken bleiben getrennt, damit sie in der Kargo-UI einzeln sichtbar sind.

**Pro Stage:**

| Stage | K | V | S | F | O | C | Arbeit |
|---|---|---|---|---|---|---|---|
| lab-d (erste Stage) | ✓ | ✓ | ✓ | dryRun | dryRun | ✓ | Probe-IPA |
| lab-c (zweite Stage, Modell für E1) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | Probe-IPA |
| e1 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ (nach Absprache) | IPA (Ausblick) |
| t1-nucleus / t1-pcilight | ✓ | ✓ | ✓ | ✓ | ✓ | – | IPA (Ausblick) |

Auf Lab-C laufen keine Instanzen anderer Teams. Damit S aussagekräftig ist, laufen dort mehrere eigene Test-Instanzen (mindestens eine mit 1 und eine mit 2 Replicas) und ein `artemis-monitor` mit beiden Brokern in der URL. Für Lab-C **eindeutige Namen** verwenden (z. B. Namespace `appl-mqp-e2e-labc`, Instanz `labc-mqp-e2e-a`), damit sich Metriken nicht mit anderen Clustern mischen.

**Fehlerfälle für die Demo der Probe-IPA (F&P04), je einer pro Testart:**

| Fehler provozieren | Erwartet: blockiert durch |
|---|---|
| Init-Image 2.56 zu Broker 2.57 | K (Promotion in `lab-d` bricht vor dem Commit ab) |
| Broker 2.57.0 mit ArkMQ v2.1.7 | K, sonst S (`BrokerVersionAligned=False`) |
| Falsches Image-Tag, Pull schlägt fehl | V (Timeout) |
| Test-Instanz: Consumer-Port blockieren | F (auf `lab-c`; in `lab-d` nur `dryRun`) |

In jedem Fall wird zusätzlich gezeigt, dass die Freight **nicht auf `lab-c` promotet werden kann**. Das ist der Nachweis der Sperre (README, Ergebnis 4).

---

## 6. Korrekturen an den bisherigen Notizen

Beim Lesen des Codes sind Punkte aufgefallen, die in den bisherigen Dateien falsch oder ungenau sind. Alle sind am 24.10.2026 in NOTIZEN.md nachgetragen.

1. **VARIANTENENTSCHEIDE.md §1:** Dort stand, für die Verifikation reichten die CRDs von Argo Rollouts. Das ist falsch, es braucht auch den Rollouts-Controller (§3). Ist korrigiert.
2. **NOTIZEN.md §12 „Lab-d und Renovate“:** Nicht nur Lab-d, sondern **Lab, E1 und beide T1** verwenden `arkmq-org-broker-kubernetes`. Nur P1 hat noch `activemq-artemis-broker-kubernetes`. Die Renovate-Regex erkennt Broker-Updates deshalb auf **keinem Nicht-Prod-Cluster**. Das ist für Kargo wichtig: Die Image-Subscription muss den neuen Namen abonnieren.
3. **NOTIZEN.md §9 „Vorlagen für die Canary-Instanz“:** `testfiles/testrun/single.yaml` und `multi.yaml` sind Fixtures für envtest (fake Secrets, `pause`-Image statt Broker). Als Canary taugen sie nicht.
4. **artemis-monitor auf E1** verbindet sich zweimal mit `ssl-0` statt mit `ssl-0` und `ssl-1` (`deployments/appl-mqp-e2e/e1-k8s-pfnet-a/deployment.yaml`). Auf E1 wird also nur ein Broker getestet. Vor der IPA korrigieren, sonst ist Test F auf E1 nur halb so aussagekräftig. Auf Lab-C gleich richtig aufsetzen. Dass im Dashboard bei `ss-1` ein Consumer verbunden ist, widerspricht dem nicht: Der Wert kann vom Monitor auf Lab-D stammen (Punkt 6).
5. **Go-Modul `v2.9.9` im PF-Operator** (offene Frage in NOTIZEN.md §4): Laut Go-Proxy stammt der Tag vom 11.06.2026, einen Tag vor ArkMQ v2.2.0, und zeigt auf einen Commit nahe v2.2.0. Im GitHub-Repo gibt es den Tag nicht mehr. Der PF-Operator kompiliert also gegen etwa den Stand von v2.2.0, nicht gegen eine „deutlich neuere API“.
6. **Grafana-Dashboard „Artemis E2E Monitor“ mischt E1 und Lab-D** (geprüft am 24.10.2026): Mit Cluster `e1-k8s-pfnet-a` und Namespace `appl-mqp-e2e-e1` zeigt es zwei Monitor-Pods, `artemis-monitor-687fd587-xhlwm` und `artemis-monitor-7c89dcd99c-xn4p5`. Der zweite läuft laut `kubectl` auf **Lab-D**. Ursache: Lab-D verwendet denselben Namespace und Instanznamen wie E1, und die Panels filtern nicht (alle) nach Cluster. Passend dazu zeigt das Panel „Log Events“ die Stufe `debug`, die nur der Monitor auf Lab-D schreibt (`LOGGING_LEVEL_CH_POSTFINANCE_ACTIVEMQ: Debug`, alle anderen Cluster `Info`).

---

## 7. Offene Fragen

- [x] Auf welcher Kargo-Control-Plane läuft das MQP-Projekt? → Probe-IPA auf der Test-Instanz (Lab-D), IPA auf management-a.
- [ ] Läuft auf Lab-D ein Argo-Rollouts-Controller, und welche Version hat er? `consecutiveSuccessLimit` gibt es erst ab Rollouts v1.8.
- [ ] Wie erreichen Verifikations-Jobs von Lab-D aus Lab-C (in der IPA von management-a aus E1/T1)? Option a: Kubeconfig mit Leserechten pro Cluster als Secret. Option b: eigene Rollouts-Instanz mit Job-Kubeconfig und `controllerInstanceID` pro Shard. Vor der Probe-IPA klären.
- [ ] Welche Prometheus-URL (direkt oder Thanos/Mimir) ist vom Kargo-Cluster aus erreichbar, und mit welcher Authentifizierung?
- [ ] §2.1 in Lab bestätigen: Broker 2.57.0 mit ArkMQ v2.1.7 deployen und `BrokerVersionAligned` beobachten.
- [ ] Wie lange dauert ein Rolling Update aller Broker auf Lab-D und Lab-C? Das bestimmt `activeDeadlineSeconds` und `initialDelay`. Für E1 in der IPA neu messen (mehr Instanzen).
- [ ] Für die IPA: Erlaubt das Team die Canary-Instanz in `appl-mqp-test` auf E1? Sonst dort ohne C.

---

## Quellen

- Kargo Docs, Verification: https://docs.kargo.io/user-guide/how-to-guides/verification
- Kargo Docs, Working with Stages: https://docs.kargo.io/user-guide/how-to-guides/working-with-stages
- Kargo Docs, Expression Language: https://docs.kargo.io/user-guide/reference-docs/expressions
- Kargo Quellcode v1.12.4: `pkg/rollouts/analysis_run_builder.go`, `pkg/controller/stages/regular_stages.go`, `api/v1alpha1/stage_types.go` (https://github.com/akuity/kargo/tree/v1.12.4)
- Argo Rollouts, Analysis: https://argo-rollouts.readthedocs.io/en/stable/features/analysis/
- Argo Rollouts, Job-Provider: https://argo-rollouts.readthedocs.io/en/stable/analysis/job/
- Argo Rollouts, Web-Provider: https://argo-rollouts.readthedocs.io/en/stable/analysis/web/
- ArkMQ v2.2.2 / v2.1.7: `api/v1beta1/activemqartemis_types.go`, `controllers/activemqartemis_reconciler.go`, `pkg/utils/common/{common,conditions}.go`, `version/version.go` (https://github.com/arkmq-org/activemq-artemis-operator)
