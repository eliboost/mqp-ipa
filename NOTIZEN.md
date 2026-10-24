# Notizen zur Probe-IPA und IPA

Sammlung der Erkenntnisse aus der Analyse der MQP-Landschaft (Stand 23.10.2026, ergänzt 24.10.2026). Die Planung steht im [README](README.md). Vertiefungen: [VARIANTENENTSCHEIDE.md](VARIANTENENTSCHEIDE.md), [TESTARTEN.md](TESTARTEN.md), [VERSIONEN.md](VERSIONEN.md).

**Aufteilung seit 24.10.2026:** Probe-IPA = Teststrategie Lab-D → Lab-C auf der Kargo-Test-Instanz, IPA = Promotion-Strategie (Lab →) E1 → T1 auf der Kargo-Prod-Instanz. Begründung im [README](README.md#übersicht).

> Diese Notizen sind mit Unterstützung von KI (Claude) entstanden. Für die IPA deklarieren (A02 Punkt 3, Doc07 Punkt 6).

## Inhalt

1. [Kargo bei uns](#1-kargo-bei-uns)
2. [MQP-Landschaft](#2-mqp-landschaft)
3. [Was auf Kubernetes läuft](#3-was-auf-kubernetes-läuft)
4. [Abhängigkeit PF-Operator und ArkMQ](#4-abhängigkeit-pf-operator-und-arkmq)
5. [Heutige Release-Strategie](#5-heutige-release-strategie)
6. [versync](#6-versync)
7. [Cluster und Stages](#7-cluster-und-stages)
8. [Technische Stolpersteine für Kargo](#8-technische-stolpersteine-für-kargo)
9. [Tests für die Operatoren](#9-tests-für-die-operatoren)
10. [Variantenentscheide](#10-variantenentscheide)
11. [Viele Kriterien in kurzer Zeit](#11-viele-kriterien-in-kurzer-zeit)
12. [Weitere Auffälligkeiten](#12-weitere-auffälligkeiten)

---

## 1. Kargo bei uns

- Kargo betreiben wir selbst. Ich baue es seit rund einem Jahr auf.
- Folge für die Arbeiten: Das Risiko „Kargo nicht verfügbar“ fällt weg. Dafür muss klar abgegrenzt werden, was vor der IPA schon da war (Kargo-Plattform, Anbindung, Grundsetup) und was die eigentliche IPA-Leistung ist. Sonst ist für die Experten nicht erkennbar, was in den IPA-Tagen entstanden ist.
- Die bestehende Kargo-Plattform ist **Ausgangslage**. Bewertet wird nur, was für MQP neu gebaut wird (Freight-Modell, Stages, Verifikationen, Gates).
- **D04 „Plattform wählen und bereitstellen“** passt deshalb nur, wenn das MQP-Kargo-Projekt erst während der Arbeit entsteht. Sonst lieber F03 oder D05. *Umgesetzt im README: Probe-IPA D02 + F03, IPA D02 + D05.*
- Zwei Kargo-Instanzen:
  - **Test-Instanz:** Control-Plane auf Lab-D, Controller auf Lab-D und Lab-C. Dient zum Testen von Kargo selbst (Versionen, Konfiguration).
  - **Prod-Instanz:** Control-Plane auf management-a. Die Controller aller anderen Cluster (E1, T1, P1, …) melden sich dort.
- Lab-D und Lab-C hängen beide an `argocd-lab`, gehören MQP und sind zum Kaputtmachen da. MQP kann auf Lab-C alles deployen, was es braucht. Prometheus sammelt heute nur Lab-D, für Lab-C lässt sich dasselbe aufbauen.
- Folge: Die Probe-IPA läuft auf der Test-Instanz mit Lab-D → Lab-C. E1 ist von dort aus nicht erreichbar. Die IPA läuft auf management-a.
- `PromotionTasks` und AnalysisTemplates gibt es schon im Repo `middleware/kargopromotion` (siehe `test.txt`).
- Vorteil: Ich kenne Kargo gut, das stärkt A09 (Fachkompetenz) und das Fachgespräch.

## 2. MQP-Landschaft

Repos aus `gitlab.pnet.ch/middleware/activemq`:

| Bereich | Repo | Zweck |
|---|---|---|
| Kern | `pf-activemq-operator` (Go) | Watcht `PFActiveMQ`, erzeugt `ActiveMQArtemis` |
| Kern | `kube-pf-activemq-operator` | Kustomize-Deployment des PF-Operators (ArgoCD) |
| Kern | `kube-artemis-operator` | Deployment des Upstream-ArkMQ-Operators (ArgoCD). Upstream-Repo heisst heute `arkmq-org/arkmq-org-broker-operator` |
| Kern | `artemis-custom-image` | Angepasstes Broker-Image (aktuell nirgends referenziert) |
| Kern | `artemis-custom-init-image` | Init-Container mit eigener Broker-Konfiguration, Logging-Plugin und JMX-Exporter |
| Kern | `pf-artemis-logging-plugin` (Java) | Broker-Plugin für detaillierteres Logging |
| Governance | `activemq-merge-operator` (Go) | Prüft Instanz-Definitionen (LDAP, CES, GitLab, K8s) |
| Governance | `versync` (Go) | Schreibt Versionen aus den kube-Repos in die Doku |
| Betrieb | `ops-ui` | MQP OPS UI (Backend, Frontend, ESProxy) |
| Betrieb | `artemis-monitor` (Spring Boot) | Sendet und konsumiert dauerhaft Testnachrichten |
| Betrieb | `deployments` | K8s-Deployments der Tools |
| Doku | `documentation` | TechDocs für activemq.pnet.ch |
| Doku | `architecture` | leer (nur GitLab-Template) |

Ablauf: Instanz-YAML → Merge-Operator prüft → ArgoCD deployt `PFActiveMQ` → PF-Operator erzeugt `ActiveMQArtemis` → ArkMQ-Operator startet den Broker mit Init-Image und Plugin → `artemis-monitor` und OPS UI überwachen.

## 3. Was auf Kubernetes läuft

| Tool | Namespace | Cluster |
|---|---|---|
| pf-activemq-operator | `kube-pf-activemq` | e1, e1-lab-d, t1-nucleus, t1-pcilight, p1-nucleus, p1-pcilight |
| ArkMQ-Operator | `kube-artemis-activemq` | dieselben 6 |
| Merge-Operator | `appl-mqp-merger-p1` | p1-management-a |
| OPS UI | `appl-mqp-ops-ui` | e1 (no-auth / with-auth), p1-management (nonprod / prod) |
| artemis-monitor | `appl-mqp-e2e-<env>` (z. B. `appl-mqp-e2e-e1`) | alle 6 Broker-Cluster |
| Vault-Sync | CronJob | – |

Nicht auf K8s: `versync` (läuft in der Doku-Pipeline), `documentation` (Backstage).

## 4. Abhängigkeit PF-Operator und ArkMQ

**Die Abhängigkeit geht nur in eine Richtung: PF-Operator → ArkMQ.**

- Compile-Zeit: `github.com/arkmq-org/activemq-artemis-operator/v2 v2.9.9` in `pf-activemq-operator/go.mod`
- Laufzeit: Watcht `ActiveMQArtemis` und wertet die ArkMQ-Conditions `Deployed`, `ZeroSize` und `Ready` aus.
- Der Status-Check vergleicht gewünschte und tatsächliche Spec mit `reflect.DeepEqual` (`controller/pfactivemq_controller.go:455`).
- Die Artemis-Version wird per Regex aus dem Tag des Broker-Images gelesen (`controller/resource/artemismq.go:49`).
- Stimmen Major.Minor von Broker- und Init-Image nicht überein, folgt `os.Exit(1)` (`artemismq.go:52-55`), und zwar erst beim Reconcile, nicht beim Start.

**Was passiert, wenn man eine Seite vergisst:**

| Szenario | Folge |
|---|---|
| ArkMQ neu, PF-Operator alt | Neue Defaults oder Felder → DeepEqual nie gleich → `ActiveMQBrokersReady=False (Mismatch)` als Dauerzustand, ständige Reconciles |
| PF-Operator neu, ArkMQ alt | Neue Felder werden abgelehnt oder still verworfen. Eine Broker-Version, die ArkMQ nicht kennt, führt dazu, dass kein Broker angelegt wird. |
| Init-Image vergessen | PF-Operator im CrashLoop (fällt wenigstens sofort auf) |

~~Offene Frage: Entspricht die Go-Modulversion `v2.9.9` den Release-Tags?~~ **Geklärt:** Laut Go-Proxy stammt `v2.9.9` vom 11.06.2026, einen Tag vor ArkMQ v2.2.0, und zeigt auf einen Commit nahe v2.2.0. Im GitHub-Repo gibt es den Tag nicht mehr. Kompiliert wird also gegen etwa den Stand von v2.2.0, nicht gegen eine neuere API.

**Neu (aus dem ArkMQ-Code):** Der PF-Operator setzt kein `spec.version`. ArkMQ nimmt dann die neueste Version seiner Liste als Sollwert und vergleicht sie per Jolokia exakt mit dem laufenden Broker. Broker-Version und ArkMQ-Release sind damit fest gekoppelt (v2.1.7 → 2.53.0, v2.2.2 → 2.57.0). Aus dem Code abgeleitet, in Lab bestätigen. Details: [VERSIONEN.md §5](VERSIONEN.md#5-abhängigkeiten-zwischen-den-versionen).

Weitere Punkte aus dem Code ([TESTARTEN.md §2](TESTARTEN.md#2-was-im-code-auffällt-und-die-tests-bestimmt)):

- Die Conditions des PF-Operators haben kein `observedGeneration`. Ein `Ready=True` direkt nach einem Upgrade kann noch vom alten Operator stammen.
- `Ready` hat auch bei `Status: False` den Reason `Ready`. Prüfungen immer auf `status`, nie auf `reason`.

**Versionsstand (Oktober 2026):**

| Cluster | ArkMQ | PF-Operator | Artemis (Init-Image) |
|---|---|---|---|
| e1, e1-lab-d, t1-nucleus, t1-pcilight | v2.2.2 | 0.17.5 | 2.57 |
| p1-nucleus, p1-pcilight | v2.1.7 | 0.17.1 (CRD v0.17.0) | 2.53 |

Lab-d nutzt das PF-Image aus der Staging-Registry und die CRD von `main`. Vollständige Tabelle: [VERSIONEN.md §4](VERSIONEN.md#4-versionsstand-pro-cluster).

## 5. Heutige Release-Strategie

### Build und Release

| Repo-Typ | Pipeline | MR | main | Tag `v*` |
|---|---|---|---|---|
| Go | `create-go-app@v0.49.4` | Test, Lint, Trivy (allow_failure), Snapshot | Staging-Image | Release, Changelog (git-cliff), Release Notes |
| Images | `pipeline-includes/images@v6.1.223` | Staging-Image | Release, Tag = Artemis-Version + Zeitstempel | – |
| Java | `pipeline-includes/java@v13.0.27` | Build, Test | Release | – |

### Renovate

- Presets aus `middleware/buildenv/renovate-presets`
- `kube-pf-activemq-operator`: Die Gruppe `artemis-version` bündelt Broker- und Init-Image (gut). Die Gruppe `pf-activemq-operator-version` bündelt Image und CRD. Ein MR pro Cluster, kein Automerge, Label `wf::Blocked`.
- `kube-artemis-operator`: Regex-Manager auf die GitHub-Release-URL, ein MR pro Cluster. Datasource ist `github-releases`: Renovate fragt die GitHub-API direkt, der Artifactory-Pfad ist nur der Text, in dem die Version steht.
- `deployments`: `automerge: false` nur für die OPS UI auf e1 und p1-nonprod. **P1-prod hat keine Regel** und verhält sich wie im Preset eingestellt. Prüfen, ob Prod automatisch gemergt wird und E1 nicht.
- Was fehlt: PF-Operator und ArkMQ werden nicht gemeinsam behandelt.

### Promotion heute

Manuell pro Cluster-MR, ArgoCD synct nach dem Merge. Die Reihenfolge steht in `documentation/docs/infra/guides/platform.md:81-89`: Lab → Blog-Post → E1 → T1 → P1, mit zwei Wochen Pause zwischen den Umgebungen. Technisch gibt es keine Sperre.

### Tests heute

| Was | Wie | Lücke |
|---|---|---|
| PF-Operator | Unit-Tests und envtest | ArkMQ-CRD von `releases/latest` (`controller/util_test.go:20`), Broker-Image fest auf 2.37 (`suite_test.go:82`) |
| artemis-monitor | docker-compose mit echtem Broker | nur bei MRs |
| logging-plugin | docker-compose mit Broker und Plugin | – |
| merge-operator, versync | Unit-Tests | – |
| `kube-*`, `deployments` | **keine CI** | kein `kustomize build`, kein Schema-Check |
| nach Deployment | `artemis-monitor` als Dauer-Canary | blockiert keine Promotion |

## 6. versync

- Liest die ArkMQ-Version aus `kube-artemis-operator/envs/<cluster>`, den PF-Operator und die Broker-Version aus `kube-pf-activemq-operator/overlays/<cluster>`.
- Vergleicht mit `documentation/data/platform_data.json` und committet bei einer Abweichung direkt auf den Branch.
- Läuft nur in der Doku-Pipeline auf `main` und reagiert nicht auf Änderungen in den kube-Repos.
- Die Doku hinkt eine Pipeline hinterher (`version-sync` und `create-platform-docs` laufen beide in der Stage `.pre`).
- Nicht erfasst werden das Init-Image, die ArkMQ-Library im PF-Operator und neue Cluster (die Liste kommt aus der JSON-Datei).
- Kein Kompatibilitätscheck, Fehler werden nur geloggt.
- Die ArkMQ-Regex verlangt, dass die URL der erste Eintrag unter `resources:` ist.
- Läuft mit einem Snapshot-Image (`versync:0.0.1-20260130080422`).
- Mit Kargo zeigt die Freight-Historie, welche Versionen wo laufen. versync könnte künftig den Stage-Status von Kargo lesen.

## 7. Cluster und Stages

| Cluster | ArgoCD | Betroffen | Rolle |
|---|---|---|---|
| `e1-k8s-pfnet-lab-d` | `argocd-lab` (eigene Instanz) | nur wir | Lab, Kargo-Test-Instanz |
| Lab-C | `argocd-lab` | nur wir | Lab, Controller der Kargo-Test-Instanz |
| `e1-k8s-pfnet-a` | `argocd-itd` | Entwicklerteams | E1 |
| `t1-k8s-nucleus-a`, `t1-k8s-pcilight-a` | `argocd-itd` | Testteams | T1 |
| `p1-*` | `argocd` | Prod | tabu |

- E1 und T1 hängen an derselben ArgoCD-Instanz. T1 dazuzunehmen kostet keine zusätzliche Anbindung.
- T1 hat dieselbe Topologie wie P1 (nucleus und pcilight) und ist damit die ideale Generalprobe für Prod.
- Erst mit T1 ist linear vs. parallel ein echter Variantenentscheid.
- Operatoren laufen clusterweit (`WATCH_NAMESPACE: "*"`). Ein Operator-Update betrifft **alle Broker im Cluster**, nicht nur einen Namespace.
- CRDs gelten im ganzen Cluster. Zwei Operator-Versionen parallel in einem Cluster gehen nicht. Mehrere Stages in einem Cluster wären nur mit virtuellen Clustern (z. B. vCluster) möglich.

## 8. Technische Stolpersteine für Kargo

1. **ArkMQ ist eine Release-URL, kein Image.** Optionen: das Operator-Image abonnieren und die Manifest-Version daraus ableiten, oder das Manifest in ein Git-Repo spiegeln. *Empfehlung: Operator-Image in `middleware-docker` abonnieren, siehe [VERSIONEN.md §3.3](VERSIONEN.md#33-wie-kargo-neue-versionen-sehen-würde).*
2. **Abhängigkeit im Freight-Modell abbilden.** Kargo kennt keine „ist kompatibel mit“-Regeln. Entweder bündeln oder per Verification absichern.
3. **Renovate und Kargo schreiben dieselben Dateien.** Die Kargo-Overlays gehören in die `ignorePaths` von Renovate.
4. **Zwei Repos pro Cluster.** Eine Promotion muss eventuell in `kube-artemis-operator` und `kube-pf-activemq-operator` committen und zwei Apps in der richtigen Reihenfolge synchronisieren.
5. **Zwei ArgoCD-Instanzen.** Lab (`argocd-lab`) und E1/T1 (`argocd-itd`).
6. **Kargo schreibt in Repos, in denen auch P1 liegt.** Absichern über CODEOWNERS oder Push-Rules, oder über einen eigenen Branch bzw. ein eigenes Repo.
7. **Lab-d ist eine Spielwiese.** Staging-Images, Debug-Logging, CRD ungepinnt von `main`. Klären, wem Lab gehört.
8. **Upgrade-Pfad testen**, nicht nur Neuinstallation (P1 springt von 2.53 / v2.1.7 / 0.17.1).
9. **Rollback ist heikel.** Downgrade von CRD und Broker-Journal ist nicht unbedingt sicher.
10. **Broker-Tag `artemis.2.57.0` ist kein SemVer.** Kargo braucht `Lexical` mit Regex.
11. **Image-Argumente stehen in einem Inline-Patch** (`patch: |-`). `yaml-update` kann dort nicht schreiben, der Patch muss in eine eigene Datei.
12. **Verifikations-Jobs laufen im Kargo-Cluster** und brauchen Zugriff auf die Ziel-Cluster. Argo-Rollouts-Controller und CRDs müssen dort laufen.

## 9. Tests für die Operatoren

| # | Testart | Prüft | Fängt ab | Umsetzung |
|---|---|---|---|---|
| 1 | Operator-Health | Deployments Ready, kein CrashLoop | Image-Versatz, kaputtes Image | `argocd-update` wartet auf Health (gratis) |
| 2 | Reconcile-Status | `ActiveMQBrokersReady=True`, `ActiveMQArtemis` Ready | CRD-/API-Inkompatibilität | AnalysisTemplate Job |
| 3 | Funktionstest | `monitor_messages_*`: Nachrichten fliessen, keine Fehler | Broker läuft, verarbeitet aber nichts | AnalysisTemplate Prometheus |
| 4 | Canary-Instanz | Temporäre `PFActiveMQ` anlegen, Ready, Nachricht senden, löschen | Neue Instanzen werden nach einem Upgrade nicht mehr korrekt angelegt | AnalysisTemplate Job |
| 5 | Kompatibilitätscheck | Kombination laut Matrix erlaubt | Ungültige Kombinationen | Job oder Prüfung vor der Promotion |
| 6 | Upgrade-Pfad | Upgrade von n-1 auf n | Probleme nur beim Upgrade | aufwändig, eher Ausblick |

- Am meisten Aussagekraft pro Aufwand haben 2 und 4.
- 3 kostet fast nichts, weil der `artemis-monitor` schon läuft. **Achtung:** Auf E1 verbindet sich der Monitor zweimal mit `ssl-0` statt mit `ssl-0` und `ssl-1`, testet also nur einen Broker.
- ~~Vorlagen für die Canary-Instanz: `testfiles/testrun/single.yaml` und `multi.yaml`.~~ **Korrektur:** Das sind Fixtures für envtest (fake Secrets, `pause`-Image). Als Vorlage taugt eine echte Instanz-Definition aus `activemq/instances`.
- Für die Demo einen Fehlerfall provozieren (z. B. falsches Init-Image), damit sichtbar wird, dass das Gate blockiert.
- **Weiterentwickelt in [TESTARTEN.md](TESTARTEN.md):** neue Kürzel K, V, S, F, O, C. Neu ist vor allem V („Version angekommen, Rollout fertig“), weil der Status des PF-Operators veraltet sein kann. Der Kompatibilitätscheck (5 = K) gehört vor den Commit, nicht in die Verifikation.

## 10. Variantenentscheide

Aktueller Stand mit Empfehlungen: [VARIANTENENTSCHEIDE.md](VARIANTENENTSCHEIDE.md). Hier der ursprüngliche Stand.

### Testarten (Probe-IPA, als Bündel, nicht einzeln)

| Variante | Inhalt | Aussagekraft | Aufwand |
|---|---|---|---|
| A, minimal | 1 + 2 | Operator läuft und hat reconciled | sehr klein |
| B | 1 + 2 + 3 | dazu: Broker verarbeiten Nachrichten | klein |
| C, umfassend | 1 + 2 + 3 + 4 + 5 | dazu: neue Instanzen, gültige Kombinationen | mittel |

- Alle Varianten bewerten, nur die gewählte umsetzen. Der Rest kommt in „Nächste Schritte“ (D03).
- A16 verlangt: Kriterien und Gewichtung vor der Bewertung festlegen, KO-Kriterien definieren (z. B. „darf andere Teams auf E1/T1 nicht beeinträchtigen“).
- ~~Vorschlag: Probe-IPA mit A oder B, IPA mit C.~~ **Neu:** Die Testarten sind der Variantenentscheid der Probe-IPA. Empfehlung dort: C (in TESTARTEN-Kürzeln K + V + S + F + O + C), Rückfall B.

### Freight-Modell (IPA)

| Variante | Beschreibung |
|---|---|
| A | Ein Warehouse, eine Freight bündelt alles |
| B | Ein Warehouse pro Komponente, Reihenfolge über die Stages |
| C | Artemis-Bundle (Broker und Init-Image) gebündelt, Operatoren getrennt |

In der Probe-IPA läuft nur ein vorläufiges Warehouse ohne Regel. Das Freight-Modell wird in der IPA evaluiert.

### Promotion-Strategie (IPA)

- Linear: lab → e1 → t1
- Parallel: t1-nucleus und t1-pcilight gleichzeitig
- Auf Komponentenebene: ArkMQ- und PF-Operator-Stage nacheinander oder gleichzeitig
- Pro Übergang: automatische oder manuelle Freigabe
- Beide bewerten, nur die gewählte umsetzen.

## 11. Viele Kriterien in kurzer Zeit

Bewertet wird vor allem, wie vorgegangen und dokumentiert wird, nicht wie viel gebaut wird.

**Ein Ergebnis deckt mehrere Kriterien ab:**

| Ergebnis | Kriterien |
|---|---|
| Umfeld- und Ist-Analyse (Diagramm, Abhängigkeiten) | A01, A03, A13, D01, eigenes „Versionsabhängigkeiten“ |
| Bewertungstabelle | A16, D02, A09 |
| Testkonzept und Testprotokoll | A12, F03, eigenes „Testkonzept“ |
| Kargo-RBAC und Ausschluss von P1 | C08 |
| Betriebsdoku und nächste Schritte | D03, A05 Punkt 4 |

**Regeln:**

- Wenig Code, gute Nachweise. Zwei bis drei AnalysisTemplates sind in Stunden gemacht.
- Alles, was nicht bewertet wird, vorher erledigen (Zugänge, ArgoCD-Anbindung, ArkMQ-Abo).
- Kriterien meiden, die viel Zusatzaufwand bedeuten.

**Beispiel Probe-IPA (Teststrategie Lab-D → Lab-C), 3 Tage à 8 h:**

| Tag | Inhalt | Kriterien |
|---|---|---|
| 1 | Auftragsanalyse, Methode, Rollen, Zeitplan (2 h). Umfeld- und Ist-Analyse inkl. Versionsabhängigkeiten (2 h). Variantenentscheid Testarten (3 h). Testkonzept mit Abnahmekriterien (1 h) | A01, A04, A11, A13, D01, A16, F03, eigenes „Testkonzept“ |
| 2 | Kompatibilitätsprüfung K (1 h). AnalysisTemplates V, S, F, O, C (4 h). Rechte der Jobs auf Lab-D/Lab-C (1 h). Journal, Überwachungskonzept (2 h) | D02, C08, E10, eigenes „Versionsabhängigkeiten“ |
| 3 | Tests in Lab-D und Lab-C durchführen, Fehlerfälle provozieren, Sperre nachweisen (3 h). Testprotokoll (1 h). Betriebsdoku, Fazit, Soll-/Ist-Vergleich (4 h) | A12, F03, D03, A05, Doc |

MQP auf Lab-C, Warehouse, Stages `lab-d`/`lab-c` und die einfache Promotion laufen schon vorher (Vorbereitung).

### Hinweise aus dem Kriterienkatalog

- A12 (Testdurchführung) ist seit Dezember 2025 Standardkriterium: reproduzierbare Testinfrastruktur, Szenarien mit erwarteten Ergebnissen bzw. DoD.
- F&P04: Die Demo muss live am Produkt laufen. Wartezeiten von Analysen und Syncs einplanen und eine verkürzte Demo vorbereiten.
- Projektmethode früh festlegen. A01, A04, A11, A12 und Doc01 hängen davon ab.
- Pflichtwahlkriterien unverändert übernehmen, Wahlkriterien dürfen angepasst oder selbst erstellt werden.
- Vorsicht bei der Nummerierung: Die Slots A13/A14 auf der Folie sind nicht dasselbe wie die Katalog-Kriterien A13/A14.
- G/H-Kriterien gelten für Applikationsentwicklung und werden für PE nicht verwendet.

## 12. Weitere Auffälligkeiten

Gut für die Ist-Analyse, nicht Teil der Arbeiten:

- **Broker-Image und Renovate** *(korrigiert)*: Nicht nur Lab-d, sondern **Lab, E1 und beide T1** nutzen `arkmq-org-broker-kubernetes` (umbenannt mit ArkMQ v2.2.2). Nur P1 hat noch `activemq-artemis-broker-kubernetes`. Die Renovate-Regex sucht den alten Namen und erkennt Broker-Updates deshalb auf keinem Nicht-Prod-Cluster.
- **Falscher Link in Renovate-MRs:** Das Broker-Image erbt `org.opencontainers.image.source` vom Red-Hat-Basis-Image. Renovate verlinkt deshalb `rh-openjdk/redhat-openjdk-containers` statt ArkMQ.
- **Artifactory-Remotes** sind Proxys mit Cache. In `middleware-github-generic-remote` liegen nur Versionen, die schon einmal angefordert wurden (heute v2.1.7, v2.2.2).
- **`artemis-custom-image`** wird in keinem Manifest referenziert.
- **`appl-mqp-test`** ist leer (`resources: []`).
- **Vault-Sync-Images** laufen mit `:latest` und `imagePullPolicy: Always`.
- **Grafana-Dashboard „Artemis E2E Monitor“ mischt E1 und Lab-D** (geprüft am 24.10.2026): Bei Cluster `e1-k8s-pfnet-a` erscheinen zwei Monitor-Pods. `artemis-monitor-7c89dcd99c-xn4p5` läuft auf Lab-D. Lab-D nutzt denselben Namespace (`appl-mqp-e2e-e1`) und Instanznamen (`e1-mqp-e2e-a`) wie E1. Folge für die Arbeiten: Jede Abfrage in einer AnalysisRun braucht einen festen Cluster-Filter, und Lab-C bekommt eindeutige Namen.
- **Monitor auf Lab-D läuft mit Log-Level `Debug`**, alle anderen mit `Info`. Lab-D schreibt dadurch pro Sekunde je eine Zeile „Send message“ und „Received message“.
- **Lab-d lädt die PF-CRD ungepinnt von `main`.**
- **`architecture`-Repo** enthält nur das GitLab-Template.
- **Schnelle Verbesserungen** ausserhalb der IPA: CI mit `kustomize build` und kubeconform für die kube-Repos, ArkMQ-URL im envtest auf die deployte Version pinnen, `spec.version` im PF-Operator setzen und `DeepEqual` durch einen Vergleich der eigenen Felder ersetzen (lockert zwei Abhängigkeiten, siehe [VERSIONEN.md §5.2](VERSIONEN.md#52-warum-die-abhängigkeiten-so-gross-sind)).
