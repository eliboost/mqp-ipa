# Prüfungen vor der Probe-IPA

Checkliste mit allem, was sich **heute** in den Clustern, in Grafana, in Artifactory und in Git prüfen lässt und Belege für die Teststrategie liefert (Stand 24.10.2026). Grundlage für Ist-Analyse (A13, D01), Testkonzept (F03, A12), Schwellwerte (E10) und Variantenentscheid (A16). Die Testarten K, V, S, F, O, C sind in [TESTARTEN.md](TESTARTEN.md) beschrieben.

> Mit Unterstützung von KI (Claude) erstellt. Für die IPA deklarieren (A02 Punkt 3, Doc07 Punkt 6).

## So benutzen

- **Ⓡ** = nur lesen, auf jedem Cluster unbedenklich. **Ⓦ** = verändert etwas, **nur auf Lab-D oder Lab-C**.
- Bei jeder Prüfung das Ergebnis in die Spalte „Ergebnis“ eintragen (Datum, Cluster, Wert). Ausgaben als Text oder Screenshot im Arbeitsjournal ablegen, damit sie als Beleg reproduzierbar sind (A12 Punkt 1).
- Namespaces in den Befehlen: `PFActiveMQ` liegen bei uns in **`kube-pf-activemq`** (nicht `kube-pf-activemq-instances`, wie die Plattform-Doku sagt, siehe P-03). ArkMQ-Operator in `kube-artemis-activemq`. Monitor in `appl-mqp-e2e-<env>`, auf Lab-D `appl-mqp-e2e-e1`.
- Cluster-Kontext vor jedem Befehl prüfen: `kubectl config current-context`.

## Inhalt

1. [Instanzen und Status (S)](#1-instanzen-und-status-s)
2. [Versionen und Kompatibilität (K, V)](#2-versionen-und-kompatibilität-k-v)
3. [Rollout und Laufzeiten (V, Demo)](#3-rollout-und-laufzeiten-v-demo)
4. [Operatoren (O)](#4-operatoren-o)
5. [Nachrichtenfluss und Monitoring (F)](#5-nachrichtenfluss-und-monitoring-f)
6. [Neue Instanz anlegen und löschen (C)](#6-neue-instanz-anlegen-und-löschen-c)
7. [Kargo und Argo Rollouts auf Lab-D](#7-kargo-und-argo-rollouts-auf-lab-d)
8. [ArgoCD](#8-argocd)
9. [Netzwerk und Rechte (C08)](#9-netzwerk-und-rechte-c08)
10. [Lab-C vorbereiten](#10-lab-c-vorbereiten)
11. [Artifactory und Versionen erkennen](#11-artifactory-und-versionen-erkennen)
12. [Git, Renovate und Geschichte](#12-git-renovate-und-geschichte)
13. [Fehlerfälle für die Demo vorab ausprobieren](#13-fehlerfälle-für-die-demo-vorab-ausprobieren)
14. [Zusammenfassung: Was welche Prüfung belegt](#14-zusammenfassung-was-welche-prüfung-belegt)

---

## 1. Instanzen und Status (S)

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-01 | Ⓡ | Anzahl `PFActiveMQ` pro Cluster | `kubectl get pfactivemq -A --no-headers \| wc -l` | Umfang von S, Dauer von V hochrechnen | |
| P-02 | Ⓡ | `Ready` und `ActiveMQBrokersReady` aller Instanzen | siehe Befehl unten (B1) | Ausgangslage: Wie viele sind **heute schon** nicht Ready? Begründet, dass S nur neue Fehler zählt | |
| P-03 | Ⓡ | In welchen Namespaces liegen `PFActiveMQ`? | `kubectl get pfactivemq -A -o custom-columns=NS:.metadata.namespace --no-headers \| sort -u` | Doku sagt `kube-pf-activemq-instances`, bei uns `kube-pf-activemq` → Fund für Ist-Analyse | |
| P-04 | Ⓡ | Alle Conditions einer nicht-Ready Instanz | Befehl B2 | Fehlerbilder verstehen, Testfälle ableiten | |
| P-05 | Ⓡ | Instanzen mit 0 Replicas | `kubectl get pfactivemq -A -o custom-columns=NAME:.metadata.name,REPL:.spec.instance.replicas \| awk '$2==0'` | Gelten als „Ready“ (`pfactivemq_controller.go:425-437`). S muss sie richtig behandeln | |
| P-06 | Ⓡ | `ActiveMQArtemis`: Ready, Valid, Version, Properties | Befehl B3 | Welche ArkMQ-Conditions S prüfen muss | |
| P-07 | Ⓡ | Steht irgendwo `Ready`-Reason `Ready` mit Status `False`? | in B1 Spalten READY und GRUND vergleichen | Beleg: Tests dürfen nur `status` prüfen, nie `reason` | |
| P-08 | Ⓡ | Haben die Conditions ein `observedGeneration`? | `kubectl get pfactivemq <name> -n kube-pf-activemq -o yaml \| grep -A6 conditions` | Beleg, warum V nötig ist (Status kann veraltet sein) | |
| P-09 | Ⓡ | `lastTransitionTime` der Conditions | in P-04 ansehen | Grundlage für S: „nach dem Start des neuen Operators auf False gewechselt“ | |
| P-10 | Ⓡ | `AuthSecretReady` und `StorageReady` vorhanden? | in P-04 ansehen | Fliessen nicht in `Ready` ein (`allConditionsReady`). S muss sie eventuell separat prüfen | |
| P-11 | Ⓡ | Gibt es `ActiveMQArtemis` ohne zugehöriges `PFActiveMQ`? | B3 mit B1 vergleichen | Verwaiste Ressourcen, sollen S nicht stören | |

**B1: Status aller Instanzen**

```bash
kubectl get pfactivemq -n kube-pf-activemq -o custom-columns=\
'NAME:.metadata.name,'\
'ZIEL:.spec.instance.namespace,'\
'REPL:.spec.instance.replicas,'\
'READY:.status.conditions[?(@.type=="Ready")].status,'\
'BROKERS:.status.conditions[?(@.type=="ActiveMQBrokersReady")].status,'\
'GRUND:.status.conditions[?(@.type=="ActiveMQBrokersReady")].reason'
```

**B2: alle Conditions einer Instanz**

```bash
kubectl get pfactivemq <name> -n kube-pf-activemq \
  -o jsonpath='{range .status.conditions[*]}{.type}{"\t"}{.status}{"\t"}{.reason}{"\t"}{.lastTransitionTime}{"\t"}{.message}{"\n"}{end}'
```

**B3: ArkMQ-Seite**

```bash
kubectl get activemqartemis -A -o custom-columns=\
'NS:.metadata.namespace,NAME:.metadata.name,'\
'READY:.status.conditions[?(@.type=="Ready")].status,'\
'VALID:.status.conditions[?(@.type=="Valid")].status,'\
'ALIGNED:.status.conditions[?(@.type=="BrokerVersionAligned")].status,'\
'PROPS:.status.conditions[?(@.type=="BrokerPropertiesApplied")].status,'\
'BROKER:.status.version.brokerVersion'
```

---

## 2. Versionen und Kompatibilität (K, V)

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-20 | Ⓡ | **Regel ArkMQ ↔ Broker live:** `ALIGNED=True` und `BROKER` = neueste Version der ArkMQ-Liste? | B3 auf E1 (erwartet `2.57.0`) und P1 (erwartet `2.53.0`) | **Wichtigster Beleg** für Test K und das Freight-Modell | |
| P-21 | Ⓡ | Laufende Images der Operatoren | `kubectl get deploy -n kube-pf-activemq -o jsonpath='{..image}'` und dasselbe für `kube-artemis-activemq` | Stimmt der Cluster mit Git überein? Grundlage für V | |
| P-22 | Ⓡ | Startargumente des PF-Operators | `kubectl get deploy pf-activemq-operator -n kube-pf-activemq -o jsonpath='{.spec.template.spec.containers[0].args}'` | Broker- und Init-Image, wie sie V erwarten muss | |
| P-23 | Ⓡ | Images in den Broker-Pods | `kubectl get pods -A -l ActiveMQArtemis -o custom-columns=NS:.metadata.namespace,POD:.metadata.name,INIT:.spec.initContainers[*].image,IMG:.spec.containers[*].image` | Laufen alle Broker mit dem Image aus den Argumenten? Gibt es Nachzügler? | |
| P-24 | Ⓡ | `status.version` von `ActiveMQArtemis` | in B3 Spalte BROKER, zusätzlich `.status.version.image` und `.initImage` | Feld, das V auswertet | |
| P-25 | Ⓡ | Liste unterstützter Versionen im laufenden ArkMQ | `kubectl get deploy -n kube-artemis-activemq -o yaml \| grep RELATED_IMAGE_ActiveMQ_Artemis_Broker_Kubernetes \| tail -3` | Letzter Eintrag = Sollwert für den Broker. Quelle für die Matrix (K) | |
| P-26 | Ⓡ | Installierte CRD-Versionen | `kubectl get crd pfactivemqs.pf-activemq-operator.pnet.ch activemqartemises.broker.amq.io -o custom-columns=NAME:.metadata.name,VERS:.spec.versions[*].name,CREATED:.metadata.creationTimestamp` | Grundlage für Rollback-Grenzen (CRD-Downgrade) | |
| P-27 | Ⓡ | Init-Image-Tag: Major.Minor + Zeitstempel | aus P-22 | Format für die Regel Broker = Init bestätigen | |
| P-28 | Ⓡ | Digest der laufenden Images | `kubectl get pods -n kube-pf-activemq -o jsonpath='{..imageID}'`, ebenso für einen Broker-Pod | Ob ein Tag neu gepusht wurde, sieht man nur am Digest | |

---

## 3. Rollout und Laufzeiten (V, Demo)

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-30 | Ⓡ | Anzahl Broker-Pods pro Cluster | `kubectl get pods -A -l ActiveMQArtemis --no-headers \| wc -l` | Dauer eines Rollouts auf E1/T1 hochrechnen | |
| P-31 | Ⓦ | **Dauer eines Rolling Restarts** auf Lab-D | `time kubectl rollout restart statefulset/e1-mqp-e2e-a-ss -n appl-mqp-e2e-e1 && kubectl rollout status statefulset/e1-mqp-e2e-a-ss -n appl-mqp-e2e-e1 --timeout=30m` | `activeDeadlineSeconds` von V, `initialDelay` von F und O, Demo-Plan | |
| P-32 | Ⓦ | Verspätungen während des Restarts | Dashboard „Artemis E2E Monitor“, Cluster Lab-D, während P-31 | Wie viele `delayed` sind im Rollout normal? Begründet das Messfenster von F | |
| P-33 | Ⓦ | Zeit, bis die Latenz nach dem Restart wieder normal ist | Dashboard, Panel Latency, nach P-31 | Länge von `initialDelay` | |
| P-34 | Ⓡ | Readiness- und Liveness-Probes der Broker | `kubectl get sts e1-mqp-e2e-a-ss -n appl-mqp-e2e-e1 -o jsonpath='{.spec.template.spec.containers[0].readinessProbe}'` | Wann gilt ein Broker als bereit? Wirkt auf die Dauer des Rollouts | |
| P-35 | Ⓡ | PodDisruptionBudgets | `kubectl get pdb -A \| grep -i mqp` | Kann ein Rollout hängen bleiben? | |
| P-36 | Ⓡ | Update-Strategie der StatefulSets | `kubectl get sts -A -l ActiveMQArtemis -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name,STRAT:.spec.updateStrategy.type` | Rolling oder OnDelete? Ändert, wie V „fertig“ erkennt | |
| P-37 | Ⓡ | `currentRevision` vs. `updateRevision` | `kubectl get sts -A -l ActiveMQArtemis -o custom-columns=NAME:.metadata.name,CUR:.status.currentRevision,UPD:.status.updateRevision,READY:.status.readyReplicas` | Felder für V. Stehen heute alle gleich? | |
| P-38 | Ⓡ | Dauer eines ArgoCD-Syncs der Operatoren | ArgoCD UI, Sync-Historie der MQP-Apps auf Lab-D | Teil der gesamten Promotion-Dauer | |

---

## 4. Operatoren (O)

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-40 | Ⓡ | Neustarts der Operator-Pods | `kubectl get pods -n kube-pf-activemq` und `-n kube-artemis-activemq` (Spalte RESTARTS) | Normalwert für O | |
| P-41 | Ⓡ | Reconcile-Rate im Normalbetrieb | Grafana Explore: `sum by (kubernetes_cluster_name) (rate(controller_runtime_reconcile_total{controller="pfactivemq"}[10m]))` | Schwellwert für O. Hohe Rate = Hinweis auf `Mismatch` | |
| P-42 | Ⓡ | Reconcile-Fehler | `sum by (kubernetes_cluster_name) (increase(controller_runtime_reconcile_errors_total{controller="pfactivemq"}[1h]))` | Normalwert für O | |
| P-43 | Ⓡ | Gibt es die Metriken überhaupt, und wie heissen die Labels? | Grafana Explore: `{__name__=~"controller_runtime_.*", kubernetes_namespace="kube-pf-activemq"}` | Machbarkeit von O; richtige Labelnamen | |
| P-44 | Ⓡ | Dasselbe für den ArkMQ-Operator | wie P-41 bis P-43 mit `kubernetes_namespace="kube-artemis-activemq"` | Sieht man Probleme auf der ArkMQ-Seite? | |
| P-45 | Ⓡ | Wechsel der Conditions in 24 h | `kubectl logs deploy/pf-activemq-operator -n kube-pf-activemq --since=24h \| grep -c "condition change"` | Flattert der Status? Fehlerbild für O | |
| P-46 | Ⓡ | Hinweise auf `Mismatch` | `kubectl logs deploy/pf-activemq-operator -n kube-pf-activemq --since=24h \| grep -i mismatch \| head` | Gibt es den Dauerzustand schon heute? | |
| P-47 | Ⓡ | Fehler im Log des ArkMQ-Operators | `kubectl logs deploy/activemq-artemis-controller-manager -n kube-artemis-activemq --since=24h \| grep -iE "error\|mismatch\|VersionMismatch" \| head` | Fehlerbilder auf der ArkMQ-Seite | |
| P-48 | Ⓡ | Speicher und CPU der Operatoren über 7 Tage | Grafana: `container_memory_working_set_bytes` / `container_cpu_usage_seconds_total` für beide Operator-Namespaces | Erkennt O Speicherlecks? Normalwerte | |
| P-49 | Ⓡ | Health-Endpunkte | `kubectl get deploy pf-activemq-operator -n kube-pf-activemq -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}'` | Was ArgoCD als „Healthy“ zählt | |

---

## 5. Nachrichtenfluss und Monitoring (F)

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-50 | Ⓡ | PromQL der Dashboard-Panels | Grafana „Artemis E2E Monitor“ → jedes Panel → Inspect → Query | Vorlage für die AnalysisTemplates von F | |
| P-51 | Ⓡ | Normalwerte auf E1, T1, P1 (7 Tage) | Dashboard, Zeitraum 7 Tage, pro Cluster | Schwellwerte für F (E10): Latenz p99, Verfügbarkeit, Delayed | |
| P-52 | Ⓡ | Filtert jedes Panel nach Cluster? | Panel-Queries aus P-50 auf `kubernetes_cluster_name` prüfen | Bestätigt den Fund „Dashboard mischt E1 und Lab-D“ pro Panel | |
| P-53 | Ⓡ | Monitor-Pods pro Cluster | `kubectl get pods -n appl-mqp-e2e-<env> -l app=artemis-monitor -o wide` | Läuft auf jedem Cluster genau einer? | |
| P-54 | Ⓡ | Broker-URL des Monitors pro Cluster | `kubectl get deploy artemis-monitor -n appl-mqp-e2e-<env> -o jsonpath='{..env[?(@.name=="SPRING_ARTEMIS_BROKER_URL")].value}'` | Auf E1 zweimal `ssl-0`? Werden alle Broker getestet? | |
| P-55 | Ⓡ | Consumer pro Broker, nur ein Cluster | Grafana: `artemis_consumer_count{kubernetes_cluster_name="e1-k8s-pfnet-a", queue="mqp.example.TEST"}` | Hat `ss-1` auf E1 wirklich keinen Consumer? | |
| P-56 | Ⓡ | Rohmetriken direkt am Monitor | `kubectl port-forward deploy/artemis-monitor -n appl-mqp-e2e-e1 8080` und `curl localhost:8080/actuator/prometheus \| grep monitor_` | Genaue Namen und Labels der Metriken | |
| P-57 | Ⓡ | Scrape-Intervall von Prometheus | Grafana: `scrape_interval` der Datenquelle oder Abstand der Datenpunkte | Mindestlänge des Messfensters (`[2m]`, `[5m]`) | |
| P-58 | Ⓡ | Prometheus-Endpunkt für Abfragen | Grafana → Datenquelle → URL (direkt Prometheus, Thanos oder Mimir?) | Adresse für den `prometheus`-Provider der AnalysisRun | |
| P-59 | Ⓡ | Broker-Metriken (`artemis_*`) vorhanden? | Grafana Explore: `{__name__=~"artemis_.*", kubernetes_cluster_name="e1-k8s-pfnet-lab-d"}` | Zusätzliche Signale für F (z. B. `artemis_message_count` in der DLQ) | |
| P-60 | Ⓡ | Bestehende Alarme für MQP | `documentation/docs/howto/alarming.md`, Alertmanager/Grafana Alerting | Schwellwerte, die es schon gibt (E10) | |
| P-61 | Ⓡ | Logging-Plugin-Ereignisse in Splunk | `index=midwitd kubernetes_namespace=appl-mqp-e2e-e1 "[PFEventLogging]"` | Kommt das Plugin nach einem Upgrade noch? Kandidat für einen Zusatztest | |
| P-62 | Ⓡ | Log-Level des Monitors pro Cluster | Env `LOGGING_LEVEL_CH_POSTFINANCE_ACTIVEMQ` (Lab-D `Debug`, sonst `Info`) | Konfigurationsunterschiede dokumentieren | |

---

## 6. Neue Instanz anlegen und löschen (C)

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-70 | Ⓡ | Vorlage für eine echte Instanz | Repo `activemq/instances`, die e2e-Instanz von Lab-D | Grundlage des Canary-Manifests | |
| P-71 | Ⓦ | **Zeit bis `Ready=True`** einer neuen Instanz auf Lab-D | Instanz anlegen, dann `kubectl wait --for=condition=Ready pfactivemq/<name> -n kube-pf-activemq --timeout=15m` mit `time` | Dauer von C, passt C in die Verifikation? | |
| P-72 | Ⓦ | Zeit für das Zertifikat | `kubectl get certificate -n <ziel> -w` während P-71 | Grösster Teil der Wartezeit? | |
| P-73 | Ⓦ | Nachricht senden und empfangen | im Broker-Pod: `/home/jboss/amq-broker/bin/artemis producer --message-count 1 ...` und `consumer` (Pfad im Image prüfen) | Befehl für den Job von C | |
| P-74 | Ⓦ | **Löschen und Aufräumen** | `kubectl delete pfactivemq <name> -n kube-pf-activemq` und danach Zertifikat, Secrets, ConfigMap, StatefulSet, PVC im Ziel-Namespace prüfen | Bleibt etwas liegen? Hängt der Finalizer? | |
| P-75 | Ⓡ | Läuft der Merge-Operator beim direkten Anlegen mit? | Logs des Merge-Operators auf management-a, Doku | Abgrenzung: C umgeht den Merge-Operator | |
| P-76 | Ⓡ | Welche Rechte braucht C? | ClusterRole des PF-Operators (`kube-pf-activemq-operator/base/clusterrole.yaml`) als Vergleich | Minimal-Rolle für den Canary-Job (C08) | |

---

## 7. Kargo und Argo Rollouts auf Lab-D

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-80 | Ⓡ | Kargo-Version der Test-Instanz | `kargo version` bzw. Image des Kargo-Controllers | Doku-Version festhalten (heute v1.12.4 aktuell) | |
| P-81 | Ⓡ | **Argo-Rollouts-CRDs** | `kubectl get crd analysistemplates.argoproj.io analysisruns.argoproj.io clusteranalysistemplates.argoproj.io` | Ohne sie keine Verifikation | |
| P-82 | Ⓡ | **Argo-Rollouts-Controller** und Version | `kubectl get pods -A \| grep -i rollouts` und Image-Tag | Wertet die AnalysisRuns aus. `consecutiveSuccessLimit` erst ab v1.8 | |
| P-83 | Ⓡ | Rollouts-Integration in Kargo aktiv | Helm-Values von Kargo: `controller.rollouts.integrationEnabled`, Log des Kargo-Controllers beim Start („Argo Rollouts integration is enabled“) | Schaltet sich sonst ohne Meldung ab | |
| P-84 | Ⓡ | `controllerInstanceID` gesetzt? | Helm-Values `controller.rollouts.controllerInstanceID` | Welcher Rollouts-Controller die AnalysisRuns übernimmt | |
| P-85 | Ⓡ | Shards: Controller auf Lab-D und Lab-C | `kubectl get pods -n <kargo-ns>` auf beiden Clustern, Shard-Namen | Welche Shard-Namen die Stages `lab-d`/`lab-c` brauchen | |
| P-86 | Ⓡ | Bestehende Projekte, Stages, AnalysisTemplates | `kargo get projects`, Repo `middleware/kargopromotion` | Muster wiederverwenden | |
| P-87 | Ⓡ | Kargo-GC: Was wird aufgeräumt? | Helm-Values `garbageCollector.*` | AnalysisRuns, Jobs, Pods bleiben liegen → `ttlSecondsAfterFinished` | |
| P-88 | Ⓡ | Wer kann Kargo-Version und Konfiguration ändern? | Team, Repo der Kargo-Installation | Einfrieren während der Probe-IPA absprechen | |
| P-89 | Ⓡ | Kargo-Metriken (`:9000`) und Grafana-Dashboard | laut `test.txt` vorhanden | Promotion- und Verifikationsdauer messen | |

---

## 8. ArgoCD

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-90 | Ⓡ | Namen der MQP-Apps in `argocd-lab` | ArgoCD UI oder `argocd app list \| grep -i -E "artemis\|activemq\|mqp"` | Für `argocd-update` und `argocd-wait` | |
| P-91 | Ⓡ | **Annotation `kargo.akuity.io/authorized-stage`** | `kubectl get application <app> -n <argocd-ns> -o jsonpath='{.metadata.annotations}'` | Ohne sie darf Kargo die App nicht anfassen | |
| P-92 | Ⓡ | Auto-Sync an oder aus? | App-Spec `syncPolicy` | Mit Auto-Sync synct ArgoCD selbst; Reihenfolge ArkMQ → PF muss trotzdem stimmen | |
| P-93 | Ⓡ | Health-Check für `PFActiveMQ` in ArgoCD? | ArgoCD-ConfigMap `argocd-cm`, Eintrag `resource.customizations.health.pf-activemq-operator.pnet.ch_PFActiveMQ` | Ob ArgoCD den Zustand der Instanzen kennt | |
| P-94 | Ⓡ | Welche Revision/Branch verfolgen die Apps? | App-Spec `source.targetRevision` | Kargo schreibt dorthin | |
| P-95 | Ⓡ | Wie lädt ArgoCD die ArkMQ-URL aus Artifactory? | Repo-Server-Logs beim Sync | Braucht es Anmeldedaten oder einen Proxy? | |

---

## 9. Netzwerk und Rechte (C08)

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-100 | Ⓦ | Erreicht ein Pod auf Lab-D die API von Lab-C? | Test-Pod auf Lab-D: `curl -k https://<lab-c-api>/version` | Voraussetzung für V, S, C auf Lab-C | |
| P-101 | Ⓦ | Erreicht ein Pod auf Lab-D Prometheus? | Test-Pod auf Lab-D: `curl <prometheus-url>/api/v1/query?query=up` | Voraussetzung für F und O | |
| P-102 | Ⓦ | Erreicht ein Pod auf Lab-D Artifactory? | `curl -I https://repo.pnet.ch/artifactory/api/system/ping` | Für den Kargo-Controller und Jobs | |
| P-103 | Ⓡ | NetworkPolicies im Kargo-Namespace | `kubectl get networkpolicy -n <kargo-ns>` | Blockieren sie Jobs? | |
| P-104 | Ⓡ | Wie kommt Kargo heute an Git? | Secrets im Kargo-Projekt, Token-Rechte in GitLab | Schreibrecht nur auf die Overlays von Lab-D/Lab-C | |
| P-105 | Ⓡ | CODEOWNERS / geschützte Branches in den kube-Repos | GitLab → Settings → Repository | Darf Kargo direkt auf `main` pushen? | |
| P-106 | Ⓡ | Rollen im Kargo-Projekt | `kubectl get roles,rolebindings -n <projekt>` | Ausgangslage für C08 | |

---

## 10. Lab-C vorbereiten

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-110 | Ⓡ | Clustername von Lab-C | `kubectl config get-contexts`, Label in Prometheus | Für Overlays, Stage-Variablen, Cluster-Filter | |
| P-111 | Ⓡ | Läuft auf Lab-C schon etwas von MQP? | `kubectl get ns \| grep -iE "artemis\|activemq\|mqp"` | Ausgangslage | |
| P-112 | Ⓡ | cert-manager und Issuer auf Lab-C | `kubectl get clusterissuer` | Die Instanzen brauchen Zertifikate | |
| P-113 | Ⓡ | StorageClass auf Lab-C | `kubectl get storageclass` | Der PF-Operator setzt fest `vsphere-csi` | |
| P-114 | Ⓡ | Ressourcen auf Lab-C (Nodes, CPU, RAM) | `kubectl top nodes` | Reichen sie für zwei bis drei Test-Instanzen + Canary? | |
| P-115 | Ⓡ | Wird Lab-C von Prometheus eingesammelt? | Grafana: `up{kubernetes_cluster_name="<lab-c>"}` | Sonst zuerst einrichten | |
| P-116 | Ⓡ | Pflicht-Secrets (Truststore, CES) auf Lab-C | Namen aus den Deployments, z. B. `ces-data-p1-shared-pki-all-store` | Monitor und Instanzen brauchen sie | |
| P-117 | Ⓡ | Eindeutige Namen festlegen | z. B. `appl-mqp-e2e-labc`, `labc-mqp-e2e-a` | Keine Vermischung mit Lab-D/E1 | |

---

## 11. Artifactory und Versionen erkennen

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-120 | Ⓡ | Tags des ArkMQ-Operator-Images, auch nie gezogene | Artifactory UI: `middleware-docker` → `arkmq-org/activemq-artemis-operator`, oder `curl https://middleware-docker.repo.pnet.ch/v2/arkmq-org/activemq-artemis-operator/tags/list` | Kann Kargo ArkMQ über das Image erkennen? | |
| P-121 | Ⓡ | Tags des Broker-Images | wie P-120 für `arkmq-org/arkmq-org-broker-kubernetes` | Format `artemis.x.y.z`, Strategie `Lexical` bestätigen | |
| P-122 | Ⓡ | Tags des Init-Images und des PF-Operators | `middleware-docker-local` | Format, nur Release-Repo verwenden | |
| P-123 | Ⓡ | Image-Tag = Release-Tag bei ArkMQ | Tags aus P-120 mit https://github.com/arkmq-org/arkmq-org-broker-operator/releases vergleichen | Annahme an der Schnittstelle | |
| P-124 | Ⓡ | Cache-Dauer der Remotes | Artifactory → Remote `middleware-docker` → „Metadata Retrieval Cache Period“ | Wie spät Kargo neue Tags sieht | |
| P-125 | Ⓡ | Inhalt des Generic-Remote | `middleware-github-generic-remote/arkmq-org/...` | Nur angeforderte Versionen (heute v2.1.7, v2.2.2) | |
| P-126 | Ⓡ | Image-Labels (Quelle) | Artifactory UI oder `skopeo inspect` | Beleg für den falschen Link in Renovate-MRs | |

---

## 12. Git, Renovate und Geschichte

| ID | | Prüfung | Befehl / Ort | Wofür | Ergebnis |
|---|---|---|---|---|---|
| P-130 | Ⓡ | **Letzte Upgrades pro Cluster** | GitLab-History von `kube-artemis-operator/envs/*` und `kube-pf-activemq-operator/overlays/*` | Dauer Lab → E1 → T1 → P1, wurde die Reihenfolge eingehalten? | |
| P-131 | Ⓡ | **Was passierte beim letzten Upgrade?** | Dashboard und Splunk zu den Zeitpunkten aus P-130 | Echte Fehlerbilder → Testfälle („Hätte Test X das gefunden?“) | |
| P-132 | Ⓡ | Gab es Incidents oder Rollbacks? | Tickets, Chat, Blog-Posts, Revert-Commits | Belegt den Nutzen der Teststrategie | |
| P-133 | Ⓡ | Offene Renovate-MRs in den kube-Repos | GitLab, Label `wf::Blocked` | Wie viele liegen heute herum? | |
| P-134 | Ⓡ | Renovate-Presets | `middleware/buildenv/renovate-presets` (`go`, `k8s-deployment`, `image`) | Automerge, `hostRules` für GitHub | |
| P-135 | Ⓡ | Renovate-Logs zum Broker-Image | Renovate-Dashboard-Issue im Repo | Beleg, dass Broker-Updates nicht mehr erkannt werden | |
| P-136 | Ⓡ | Blog-Posts zu früheren Upgrades | Doku-Seite/Blog | Inhalt und Zeitpunkt (Prozess vor E1) | |
| P-137 | Ⓡ | `PFACTIVEMQ_CRD_VERSION` im Repo `activemq/instances` | `.gitlab-ci.yml` | Passt sie zur deployten CRD? | |
| P-138 | Ⓡ | versync: ist die Doku aktuell? | `documentation/data/platform_data.json` mit den Overlays vergleichen | Ist-Analyse | |

---

## 13. Fehlerfälle für die Demo vorab ausprobieren

Alles **nur auf Lab-D**, vor der Probe-IPA, ohne Kargo. So weisst du, wie jeder Fehler im Cluster aussieht, bevor du den Test dafür schreibst.

| ID | | Fehler provozieren | Was beobachten | Testart | Ergebnis |
|---|---|---|---|---|---|
| P-140 | Ⓦ | Init-Image 2.56 zu Broker 2.57 | PF-Operator `os.Exit(1)` beim Reconcile? Nach wie vielen Sekunden? Health in ArgoCD? | K | |
| P-141 | Ⓦ | Broker 2.57.0 mit ArkMQ v2.1.7 | `BrokerVersionAligned=False`? Was zeigt `PFActiveMQ`? Laufen die Broker weiter? | K, S | |
| P-142 | Ⓦ | Image-Tag, das es nicht gibt | `ImagePullBackOff`, wie lange bis sichtbar? | V | |
| P-143 | Ⓦ | Consumer-Port blockieren (NetworkPolicy) | Wann steigt `delayed` im Dashboard? | F | |
| P-144 | Ⓦ | Operator-Pod löschen | Wie lange bis wieder Ready? Neustarts in den Metriken? | O | |
| P-145 | Ⓦ | Instanz mit ungültiger Spec anlegen | Welche Condition, welche Meldung? | C, S | |
| P-146 | Ⓦ | StatefulSet auf OnDelete stellen und Image ändern | Bleibt der Rollout hängen? Merkt man es nur an `updateRevision`? | V | |

---

## 14. Zusammenfassung: Was welche Prüfung belegt

| Beleg für | Prüfungen |
|---|---|
| Test K und Freight-Modell | P-20, P-25, P-27, P-140, P-141 |
| Test V | P-08, P-21 bis P-24, P-28, P-31, P-36, P-37, P-142, P-146 |
| Test S | P-01 bis P-11, P-141, P-145 |
| Test F | P-32, P-33, P-50 bis P-62, P-143 |
| Test O | P-40 bis P-49, P-144 |
| Test C | P-70 bis P-76, P-145 |
| Schwellwerte (E10) | P-32, P-33, P-41, P-42, P-48, P-51, P-57, P-60 |
| Machbarkeit Kargo/Rollouts | P-80 bis P-95, P-100 bis P-102 |
| Rollenkonzept (C08) | P-76, P-104 bis P-106 |
| Ist-Analyse (A13, D01) | P-03, P-52, P-54, P-62, P-120 bis P-138 |
| Demo-Planung (F&P04) | P-31, P-38, P-71, P-140 bis P-146 |

**Wenn die Zeit knapp ist, zuerst:** P-20, P-02, P-31/P-32, P-81/P-82, P-91, P-100/P-101, P-130/P-131.
