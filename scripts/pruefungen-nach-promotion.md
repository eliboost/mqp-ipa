# Prüfungen nach einer Kargo-Promotion: Prioritäten

Was nach dem Promoten neuer Versionen auf eine Umgebung geprüft wird, nach Priorität. Grundlage: [§8 und Anhang A](../DETAILBESCHRIEB-VORLAGE.md#8-testarten) (A.1, A.3 bis A.5), ergänzt mit Funden aus Lab-D vom 26.10.2026 (Befehle: [pruefungen-befehle.md](pruefungen-befehle.md)).

## Reihenfolge nach einer Promotion

```
K (vor dem Commit) → ArgoCD-Sync → Kargo wartet auf Healthy → AnalysisRun:
   V → S → (C)      als ein Job mit Phasen
   F, O             Prometheus, mit initialDelay
```

## Priorität 1: unbedingt

### K: Kompatibilität, vor dem Commit (`fail`-Step)

- Init und Broker haben dieselbe Major.Minor (`2.57.2026…` ↔ `artemis.2.57.0`).
- Der Broker ist die neueste Version in der Liste des ArkMQ-Operators.

Damit wird nichts deployt, bevor der PF-Operator mit `os.Exit(1)` in einen Crash-Loop geht (`pf-activemq-operator/controller/resource/artemismq.go:50-54`).

### V: Version angekommen, Rollout fertig (Job)

Ohne V prüfen S, F und O eventuell noch den alten Stand. Die Conditions haben kein `observedGeneration` (auf Lab-D belegt).

| Was | Feld |
|---|---|
| Operator-Pods mit dem Image aus der Freight | `pf-activemq-operator`, `activemq-artemis-controller-manager` |
| Jedes `ActiveMQArtemis` hat die neue Version | `status.version.image`, `.brokerVersion`, `.initImage` |
| Alle StatefulSets fertig ausgerollt | `updatedReplicas == replicas`, `currentRevision == updateRevision` |
| Alle Broker-Pods laufen mit dem neuen Image | Pods mit `-l ActiveMQArtemis`, Init- und Broker-Image |

### S: Status aller Instanzen (Job)

- `PFActiveMQ`: `Ready` und `ActiveMQBrokersReady` sind `True`.
- `ActiveMQArtemis`: `Ready`, `Valid`, `BrokerVersionAligned` und `BrokerPropertiesApplied` sind `True`.
- Nur `status` prüfen, **nie `reason`**. Auf Lab-D steht `False` mit Reason `Ready`.
- Nur **neue** Fehler zählen: `lastTransitionTime` liegt nach dem Start des neuen Operator-Pods. Auf Lab-D haben `e1-mqp-selmaniarbe-a` und `e1-mqp-selmaniarbe2-a` schon heute `Mismatch` und würden das Gate sonst dauerhaft blockieren.

### F: Nachrichtenfluss (Prometheus, artemis-monitor)

- Verbrauchte Nachrichten > 0, `delayed` = 0, Fehler = 0, p99-Latenz unter dem Schwellwert.
- **Fester Cluster-Filter** `kubernetes_cluster_name`, weil Lab-D und E1 dieselben Namespace- und Instanznamen haben.
- `initialDelay` muss grösser sein als die Rollout-Dauer plus das Messfenster. Während des Rolling Updates sind Verspätungen normal.

## Priorität 2: sehr gut

### O: Operator stabil (Prometheus)

- Keine Neustarts (`kube_pod_container_status_restarts_total`).
- Keine Reconcile-Fehler, keine steigende Reconcile-Rate.
- Auf Lab-D belegt: Der PF-Operator schreibt `Mismatch` **nicht ins Log**. Ein Dauer-`Mismatch` ist also nur über die Reconcile-Rate sichtbar.
- Der ArkMQ-Operator hatte schon 2 Neustarts. Darum im Messfenster mit `increase(...)` messen, nicht den absoluten Zähler.

### C: Canary-Instanz (Job, nur Lab-D und Lab-C)

Ablauf: anlegen, warten auf `Ready=True`, Nachricht senden und empfangen, löschen. Danach prüfen, dass Zertifikat, Secrets, ConfigMap und Finalizer weg sind.

## Auf Kargo-Ebene prüfen

| Was | Worauf achten |
|---|---|
| AnalysisRun-Phase | Nur `Successful` verifiziert die Freight. `Failed`, `Error`, `Aborted` und **`Inconclusive`** blockieren. |
| Kein `Inconclusive` | Jede Prometheus-Metrik braucht `successCondition` **und** `failureCondition`. Jeder Job braucht `activeDeadlineSeconds` und `backoffLimit: 0/1`. |
| Sperre wirkt | Eine Freight, die auf `lab-d` gescheitert ist, darf sich **nicht** auf `lab-c` promoten lassen. Das ist der Nachweis für die Demo. |
| Aufräumen | `ttlSecondsAfterFinished` in den Jobs setzen, weil der Kargo-GC AnalysisRuns, Jobs und Pods nicht aufräumt. |
| ArgoCD | Apps `Healthy` und `Synced`, Annotation `kargo.akuity.io/authorized-stage` vorhanden (P-91). |

## Zahlen von Lab-D für die Timeouts

- 16 Broker-Pods in 8 StatefulSets mit je 2 Replicas, alle `RollingUpdate`.
- Readiness-Probe: alle 10 s, `failureThreshold` 10, also bis zu ~100 s pro Pod.
- Daraus folgen `activeDeadlineSeconds` für V und `initialDelay` für F und O. Den echten Wert liefert erst P-31, die gemessene Dauer eines Rolling Restarts.

## Pro Stage in der Probe-IPA

| Stage | K | V | S | F | O | C |
|---|---|---|---|---|---|---|
| lab-d | ✓ | ✓ | ✓ | dryRun | dryRun | ✓ |
| lab-c | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

Welche Kombination umgesetzt wird (Variante 1 = K+V+S+O oder Variante 2 = K+V+F+C), entscheidet die Nutzwertanalyse (§9).
