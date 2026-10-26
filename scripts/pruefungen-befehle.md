# Prüfungen vor der Probe-IPA: kubectl-Befehle (nur lesen)

Alle Befehle aus [Anhang C](../DETAILBESCHRIEB-VORLAGE.md#anhang-c-prüfungen-vor-der-probe-ipa), die mit `kubectl` gehen und **nichts im Cluster verändern** (kein `delete`, `restart`, `run`, `apply`, `edit`). Die IDs (P-xx) entsprechen Anhang C. Die Argumente mit `[*]` stehen in Anführungszeichen, sonst bricht zsh mit „no matches found“ ab.

Platzhalter in `<...>` durch die Namen aus dem jeweils vorherigen Befehl ersetzen.

Nicht enthalten: P-100 bis P-102 (Netzwerk), weil dafür ein Test-Pod mit `kubectl run` gestartet werden muss, sowie alle Ⓦ-Prüfungen (P-31 bis P-33, P-71 bis P-74, P-140 bis P-146).

Vor jedem Befehl den Kontext prüfen:

```bash
kubectl config current-context
```

## C.1 Instanzen und Status (S)

```bash
# P-01 Anzahl Instanzen
kubectl get pfactivemq -A --no-headers | wc -l

# P-02 / P-07 Status aller Instanzen (B1)
kubectl get pfactivemq -n kube-pf-activemq -o custom-columns='NAME:.metadata.name,ZIEL:.spec.instance.namespace,REPL:.spec.instance.replicas,READY:.status.conditions[?(@.type=="Ready")].status,READY_GRUND:.status.conditions[?(@.type=="Ready")].reason,BROKERS:.status.conditions[?(@.type=="ActiveMQBrokersReady")].status,GRUND:.status.conditions[?(@.type=="ActiveMQBrokersReady")].reason'

# P-03 Namespaces der Instanzen
kubectl get pfactivemq -A -o 'custom-columns=NS:.metadata.namespace' --no-headers | sort -u

# P-04 / P-09 / P-10 alle Conditions einer Instanz (B2)
kubectl get pfactivemq <name> -n kube-pf-activemq -o jsonpath='{range .status.conditions[*]}{.type}{"\t"}{.status}{"\t"}{.reason}{"\t"}{.lastTransitionTime}{"\t"}{.message}{"\n"}{end}'

# P-05 Instanzen mit 0 Replicas
kubectl get pfactivemq -A -o 'custom-columns=NAME:.metadata.name,REPL:.spec.instance.replicas' --no-headers | awk '$2==0'

# P-06 / P-11 / P-20 / P-24 ArkMQ-Seite (B3)
kubectl get activemqartemis -A -o custom-columns='NS:.metadata.namespace,NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status,VALID:.status.conditions[?(@.type=="Valid")].status,ALIGNED:.status.conditions[?(@.type=="BrokerVersionAligned")].status,PROPS:.status.conditions[?(@.type=="BrokerPropertiesApplied")].status,BROKER:.status.version.brokerVersion,IMAGE:.status.version.image,INIT:.status.version.initImage'

# P-08 observedGeneration vorhanden?
kubectl get pfactivemq <name> -n kube-pf-activemq -o yaml | grep -A6 conditions
```

## C.2 Versionen und Kompatibilität (K, V)

```bash
# P-21 Images der Operatoren
kubectl get deploy -n kube-pf-activemq -o 'custom-columns=NAME:.metadata.name,IMAGE:.spec.template.spec.containers[*].image'
kubectl get deploy -n kube-artemis-activemq -o 'custom-columns=NAME:.metadata.name,IMAGE:.spec.template.spec.containers[*].image'

# P-22 / P-27 Startargumente PF-Operator (Broker- und Init-Image)
kubectl get deploy pf-activemq-operator -n kube-pf-activemq -o jsonpath='{.spec.template.spec.containers[0].args}' | tr ',' '\n'

# P-23 Images in den Broker-Pods
kubectl get pods -A -l ActiveMQArtemis -o 'custom-columns=NS:.metadata.namespace,POD:.metadata.name,INIT:.spec.initContainers[*].image,IMG:.spec.containers[*].image'

# P-25 unterstützte Broker-Versionen im ArkMQ (letzter Eintrag = Sollwert)
kubectl get deploy -n kube-artemis-activemq -o yaml | grep -A1 RELATED_IMAGE_ActiveMQ_Artemis_Broker_Kubernetes | tail -6

# P-26 CRD-Versionen
kubectl get crd pfactivemqs.pf-activemq-operator.pnet.ch activemqartemises.broker.amq.io -o 'custom-columns=NAME:.metadata.name,VERS:.spec.versions[*].name,CREATED:.metadata.creationTimestamp'

# P-28 Digests
kubectl get pods -n kube-pf-activemq -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[*].imageID}{"\n"}{end}'
kubectl get pods -A -l ActiveMQArtemis -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.containerStatuses[*].imageID}{"\n"}{end}'
```

## C.3 Rollout (nur lesen)

```bash
# P-30 Anzahl Broker-Pods
kubectl get pods -A -l ActiveMQArtemis --no-headers | wc -l

# P-34 Probes der Broker
kubectl get sts e1-mqp-e2e-a-ss -n appl-mqp-e2e-e1 -o jsonpath='{.spec.template.spec.containers[0].readinessProbe}'

# P-35 PodDisruptionBudgets
kubectl get pdb -A | grep -iE 'mqp|artemis|activemq'

# P-36 / P-37 Update-Strategie und Revisionen
kubectl get sts -A -l ActiveMQArtemis -o 'custom-columns=NS:.metadata.namespace,NAME:.metadata.name,STRAT:.spec.updateStrategy.type,REPL:.spec.replicas,READY:.status.readyReplicas,UPDATED:.status.updatedReplicas,CUR:.status.currentRevision,UPD:.status.updateRevision'
```

## C.4 Operatoren (O)

```bash
# P-40 Neustarts
kubectl get pods -n kube-pf-activemq
kubectl get pods -n kube-artemis-activemq

# P-45 Wechsel der Conditions in 24 h
kubectl logs deploy/pf-activemq-operator -n kube-pf-activemq --since=24h | grep -ic condition

# P-46 Mismatch im PF-Operator
kubectl logs deploy/pf-activemq-operator -n kube-pf-activemq --since=24h | grep -i mismatch | head -20

# P-47 Fehler im ArkMQ-Operator (zuerst den Namen des Deployments nachsehen)
kubectl get deploy -n kube-artemis-activemq
kubectl logs deploy/<arkmq-deployment> -n kube-artemis-activemq --since=24h | grep -iE 'error|mismatch' | head -30

# P-49 Probes des Operators
kubectl get deploy pf-activemq-operator -n kube-pf-activemq -o jsonpath='{.spec.template.spec.containers[0].livenessProbe}'
```

## C.5 Monitor (F)

`<env>` z. B. `e1`; auf Lab-D ebenfalls `appl-mqp-e2e-e1`.

```bash
# P-53 Monitor-Pods
kubectl get pods -n appl-mqp-e2e-<env> -l app=artemis-monitor -o wide

# P-54 Broker-URL des Monitors
kubectl get deploy artemis-monitor -n appl-mqp-e2e-<env> -o jsonpath='{..env[?(@.name=="SPRING_ARTEMIS_BROKER_URL")].value}'

# P-62 Log-Level des Monitors
kubectl get deploy artemis-monitor -n appl-mqp-e2e-<env> -o jsonpath='{..env[?(@.name=="LOGGING_LEVEL_CH_POSTFINANCE_ACTIVEMQ")].value}'
```

## C.6 Rechte für den Canary-Job (C)

```bash
# P-76 ClusterRole des PF-Operators als Vergleich (Name zuerst suchen)
kubectl get clusterrole | grep -i pf-activemq
kubectl get clusterrole <name> -o yaml
```

## C.7 Kargo und Argo Rollouts

```bash
# P-81 Argo-Rollouts-CRDs
kubectl get crd analysistemplates.argoproj.io analysisruns.argoproj.io clusteranalysistemplates.argoproj.io

# P-82 Rollouts-Controller und Version
kubectl get pods -A -o 'custom-columns=NS:.metadata.namespace,POD:.metadata.name,IMG:.spec.containers[*].image' | grep -i rollouts

# P-80 / P-85 Kargo-Pods, Version, Shards
kubectl get pods -A -o 'custom-columns=NS:.metadata.namespace,POD:.metadata.name,IMG:.spec.containers[*].image' | grep -i kargo

# P-83 Rollouts-Integration aktiv? (Namespace und Pod aus P-80)
kubectl logs <kargo-controller-pod> -n <kargo-ns> | grep -i rollouts | head

# P-106 Rollen im Kargo-Projekt
kubectl get roles,rolebindings -n <projekt>
```

## C.8 ArgoCD

```bash
# P-90 MQP-Apps
kubectl get applications.argoproj.io -A | grep -iE 'artemis|activemq|mqp'

# P-91 Annotation authorized-stage
kubectl get application <app> -n <argocd-ns> -o jsonpath='{.metadata.annotations}'

# P-92 Auto-Sync
kubectl get application <app> -n <argocd-ns> -o jsonpath='{.spec.syncPolicy}'

# P-94 Ziel-Revision
kubectl get application <app> -n <argocd-ns> -o jsonpath='{.spec.source.targetRevision}'

# P-93 Health-Check für PFActiveMQ
kubectl get cm argocd-cm -n <argocd-ns> -o yaml | grep -A3 'pf-activemq-operator.pnet.ch_PFActiveMQ'
```

## C.9 Netzwerk und Rechte

```bash
# P-103 NetworkPolicies im Kargo-Namespace
kubectl get networkpolicy -n <kargo-ns>
```

## C.10 Lab-C (im Lab-C-Kontext ausführen)

```bash
# P-110 Kontexte und Clustername
kubectl config get-contexts

# P-111 MQP schon vorhanden?
kubectl get ns | grep -iE 'artemis|activemq|mqp'

# P-112 Issuer für Zertifikate
kubectl get clusterissuer

# P-113 StorageClass (vsphere-csi vorhanden?)
kubectl get storageclass

# P-114 Ressourcen der Nodes
kubectl top nodes
```
