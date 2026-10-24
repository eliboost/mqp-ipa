# Variantenentscheide Test- und Promotion-Strategie

Vorschläge für die Variantenentscheide der Probe-IPA und der IPA (Stand 24.10.2026). Grundlage sind die [Notizen](NOTIZEN.md), das [README](README.md) und die offizielle Kargo-Doku (docs.kargo.io, aktuelles Release **v1.12.4** vom 12.10.2026).

**Aufteilung (seit 24.10.2026):** Die Probe-IPA behandelt die **Teststrategie** für Lab → E1. Die IPA behandelt die **Promotion-Strategie** für Lab → E1 → T1 und baut auf den Tests der Probe-IPA auf. Begründung im [README](README.md#übersicht).

## Inhalt

1. [Was Kargo on-prem (Open Source) bietet](#1-was-kargo-on-prem-open-source-bietet)
2. [Probe-IPA: Teststrategie Lab → E1](#2-probe-ipa-teststrategie-lab--e1)
3. [IPA: Promotion-Strategie Lab → T1](#3-ipa-promotion-strategie-lab--t1)
4. [Vorgehen bei der Bewertung (A16)](#4-vorgehen-bei-der-bewertung-a16)
5. [In Lab zu prüfen](#5-in-lab-zu-prüfen)
6. [Quellen](#6-quellen)

---

## 1. Was Kargo on-prem (Open Source) bietet

Alle Varianten unten nutzen nur Funktionen, die in der Open-Source-Version stabil (ohne Label) dokumentiert sind. Funktionen mit **Beta**- oder **Enterprise Beta**-Label sind ausgeschlossen.

### Verwendbar

| Bereich | Funktion | Wofür bei uns |
|---|---|---|
| Warehouse | Subscriptions `image`, `git`, `chart` | ArkMQ-, PF-, Broker- und Init-Image aus `repo.pnet.ch` |
| Warehouse | `freightCreationCriteria` (Expression) | Freight nur erzeugen, wenn die Kombination gültig ist (z. B. Broker = Init) |
| Warehouse | `allowTags`, `semverConstraint` | Major.Minor fest vorgeben |
| Warehouse | `freightCreationPolicy: Manual` | Freight nur von Hand erzeugen (z. B. für Prod-Konzept) |
| Stage | `requestedFreight[].sources.direct` / `stages` | Woher eine Stage Freight annimmt |
| Stage | `availabilityStrategy: OneOf` / `All` | Fan-in: Freight erst nach Verifikation in allen Upstream-Stages (P1-Konzept) |
| Stage | `requiredSoakTime` | Mindestzeit, die Freight in der Upstream-Stage laufen muss |
| Stage | `autoPromotionOptions.selectionPolicy` (`NewestFreight`, `MatchUpstream`) | Welche Freight automatisch promotet wird |
| Stage | `spec.verification.analysisTemplates` (auch `ClusterAnalysisTemplate`), `args` | Tests nach der Promotion |
| Stage | Implizite Argo-CD-Health | Verifikation startet erst, wenn die Apps `Healthy` sind |
| ProjectConfig | `promotionPolicies` mit `stageSelector` und `autoPromotionEnabled` | Automatische oder manuelle Freigabe pro Stage |
| RBAC | Verb `promote` auf `stages` mit `resourceNames`, Rollen `kargo-admin`, `kargo-promoter`, `kargo-viewer` | Wer auf welche Stage promoten darf (C08) |
| Freight | `kargo approve` | Freight für eine Stage freigeben ohne Upstream-Verifikation (Hotfix) |
| Verifikation | `kargo verify stage` (auch `--abort`) | Verifikation erneut laufen lassen |
| Rollback | ältere Freight promoten → Auto-Promotion-Hold | Rollback ohne neues Tooling |
| Promotion | `PromotionTask` | Gleiche Schritte für alle Stages, Werte pro Stage |
| Promotion-Steps | `git-clone`, `kustomize-set-image`, `yaml-update`, `git-commit`, `git-push`, `git-open-pr`, `git-wait-for-pr`, `argocd-update`, `argocd-wait`, `http`, `fail`, `set-metadata` | Overlays schreiben, Apps in Reihenfolge synchronisieren, Vorprüfungen |
| Expressions | `imageFrom`, `commitFrom`, `semverParse`, `semverDiff`, `configMap`, `warehouse` | Versionen auslesen, Major.Minor vergleichen, Matrix aus ConfigMap lesen |

### Ausgeschlossen (Beta oder Enterprise)

| Funktion | Label | Ersatz in Open Source |
|---|---|---|
| `wait-for-approval`, `get-user-input` | Enterprise Beta | Auto-Promotion für die Stage aus + `promote`-Recht nur für berechtigte Personen |
| `send-message` | Enterprise Beta | Info ans Team ausserhalb von Kargo (Blog-Post, Chat, Mail), im Betriebskonzept festhalten |
| `git-merge-pr` | Beta | `git-open-pr` + `git-wait-for-pr`, Merge von Hand in GitLab |
| `jira`, `snow-*`, `gh-*`, `gha-*`, `tf-*` | Enterprise Beta | nicht benötigt |
| `toml-update`, `toml-parse`, `github-push` | Beta | nicht benötigt |

### Wichtig für die Verifikation

- Kargo verwendet `AnalysisTemplate` und `AnalysisRun` von Argo Rollouts. Kargo **erzeugt** die AnalysisRun nur und liest danach ihre Phase. **Ausgewertet wird sie vom Argo-Rollouts-Controller**, der im Cluster der Kargo-Control-Plane laufen muss (Kargo-Quellcode v1.12.4, `pkg/rollouts/analysis_run_builder.go`; der Quickstart installiert Rollouts als eigenen Schritt). Fehlen die CRDs, schaltet Kargo die Integration ohne Meldung ab. Es müssen also CRDs **und** Controller vorhanden sein.
- Die Jobs einer AnalysisRun laufen deshalb im Cluster der Kargo-Control-Plane, nicht in Lab, E1 oder T1. Für Status-Checks braucht der Job einen Zugriff auf den Ziel-Cluster (siehe [§2.2](#22-ausführung-der-verifikationen-technischer-entscheid-nicht-bewertet)). Details zu den Testarten: [TESTARTEN.md](TESTARTEN.md).
- Solange eine Verifikation läuft, warten andere Promotionen auf dieselbe Stage. Das ist für die Demo relevant (F&P04).

---

## 2. Probe-IPA: Teststrategie Lab → E1

Thema: Welche Verifikationen sichern eine neue Kombination in Lab und E1 ab? Es gibt **einen** bewerteten Variantenentscheid (Testarten). Die Ausführung der Verifikationen (§2.2) wird im Konzept begründet, aber nicht bewertet. Die Promotion (§2.3) ist Vorbereitung und bewusst vorläufig.

### 2.1 Testarten (bewerteter Variantenentscheid)

Die Testarten sind in [TESTARTEN.md](TESTARTEN.md) beschrieben. Kurz:

| Kürzel | Testart | Wo in Kargo |
|---|---|---|
| K | Kompatibilität der Kombination (Matrix ArkMQ → Artemis, Broker = Init) | **vor** dem Commit, `fail`-Step in der Promotion |
| V | Version angekommen, Rollout fertig | Verifikation, Job |
| S | Status aller Instanzen ohne Regression | Verifikation, Job |
| F | Nachrichtenfluss (`artemis-monitor`) | Verifikation, Prometheus |
| O | Operator stabil (keine Neustarts, kein Dauer-Reconcile) | Verifikation, Prometheus |
| C | Canary-Instanz anlegen, testen, löschen | Verifikation, Job |

Die Kompatibilitätsprüfung K gehört **vor** den Commit, nicht in die Verifikation. Eine Verifikation läuft erst nach dem Deployment, dann ist eine ungültige Kombination schon im Cluster. Weil K in den Promotion-Schritten steht, läuft sie auch dann, wenn jemand eine Freight mit `kargo approve` an Lab vorbei freigibt.

| | A, minimal | B, Funktion | C, umfassend |
|---|---|---|---|
| Inhalt | V + S | K + V + S + F | K + V + S + F + O + C |
| Fängt ab | Crash, CRD-/API-Fehler, veralteter Status | dazu: Broker verarbeitet nichts, ungültige Kombination vor dem Commit | dazu: Dauer-Reconcile, neue Instanzen werden nicht mehr korrekt angelegt |
| Laufzeit pro Stage | Rollout aller Broker | + Messfenster (z. B. 10 min) | + Anlegen und Löschen der Instanz |
| Auswirkung auf andere Teams | keine | keine (nur lesen) | legt auf E1 eine Instanz in `appl-mqp-test` an → Absprache nötig |
| Aufwand | klein | mittel | gross |
| README-Ergebnis (V, S, F + K) | **nicht erfüllt** | erfüllt | erfüllt |

**Empfehlung: Variante C**, abgestuft pro Stage:

| Stage | K | V | S | F | O | C |
|---|---|---|---|---|---|---|
| lab | ✓ | ✓ | ✓ | `dryRun` | `dryRun` | ✓ |
| e1 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

Begründung:

- V ist die Grundlage für alles andere. Der Status des PF-Operators kann veraltet sein (kein `observedGeneration`), ohne V messen die anderen Tests eventuell noch den alten Stand.
- C ist die einzige Prüfung für den Fall „Upgrade kaputt, bestehende Broker laufen aber weiter“.
- F und O sind in Lab nur informativ: Lab-Broker haben wenig Last, und das Messfenster ist unzuverlässig. Ab E1 blockieren sie.
- Die Abstufung pro Stage macht die Bewertung stärker als „C gewinnt, weil es am meisten ist“.

Variante A fällt am KO-Kriterium „README-Ergebnisse erfüllt“ heraus. **B ist die Rückfallvariante**, falls die Canary-Instanz auf E1 vom Team nicht erlaubt wird.

**Welche Tests blockieren?** In Kargo blockiert jede nicht erfolgreiche Verifikation die Freight für die nächste Stage, auch `Inconclusive`. Soll ein Test nur informieren, wird die Metrik im AnalysisTemplate auf `dryRun` gesetzt. Kargo übernimmt `dryRun` aus dem Template (Quellcode v1.12.4), der Rollouts-Controller wertet es aus.

**Upgrade-Pfad (U)** bleibt Ausblick. Er bräuchte eine Stage mit dem P1-Stand (2.53 / v2.1.7 / 0.17.1), und die gibt es in keinem Cluster.

### 2.2 Ausführung der Verifikationen (technischer Entscheid, nicht bewertet)

Die Jobs laufen im Kargo-Cluster. Die Status-Checks brauchen aber Daten aus Lab und E1.

| Variante | Wie | Pro | Contra |
|---|---|---|---|
| a) Job mit Kubeconfig | Pro Ziel-Cluster ein ServiceAccount mit nur Leserechten auf `pfactivemqs`, `activemqartemises`, StatefulSets und Pods, Token als Secret im Kargo-Projekt | einfach, genau | Secrets pro Cluster verwalten |
| b) Prometheus | Status über kube-state-metrics (Custom Resource State) abfragen | kein Cluster-Zugriff | kube-state-metrics muss die CRs exportieren, eher nicht vorhanden |
| c) `web`-Provider | direkt gegen die K8s-API | kein Job-Pod | Token im Template, unübersichtlich |

**Empfehlung: a für V, S und C, Prometheus für F und O.** Die Leserechte pro Cluster sind gleich ein Baustein für C08 (Least Privilege). Für C braucht der ServiceAccount zusätzlich Schreibrechte, aber nur im Namespace `appl-mqp-test`.

### 2.3 Promotion als Voraussetzung (Vorbereitung, nicht bewertet)

Damit die Verifikation läuft, braucht es Warehouse, Stages und eine Promotion. Sie werden **vor** der Probe-IPA aufgebaut und sind **bewusst vorläufig**, damit die Entscheide der IPA offen bleiben.

| Punkt | Vorläufig in der Probe-IPA | Wird in der IPA entschieden |
|---|---|---|
| Freight | ein Warehouse mit vier Subscriptions, **ohne** `freightCreationCriteria` und ohne feste Vorgabe. Ungültige Kombinationen fängt K ab | Freight-Modell (§3.1) |
| Stages | `lab` (Freight direkt vom Warehouse), `e1` (Freight nur aus `lab`, nach erfolgreicher Verifikation) | Topologie mit T1 (§3.2 a) |
| Freigabe | beide Stages **manuell** | automatisch oder manuell pro Übergang (§3.2 c) |
| Git | `git-push` direkt | Git-Weg (§3.2 c) |
| Reihenfolge | `argocd-update` ArkMQ-App → `argocd-wait` → PF-App | bestätigt (§3.2 b) |
| P1 | keine Stage, keine Rechte | Konzept |

Die Sperre „E1 nur nach grüner Verifikation in Lab“ kommt aus `sources.stages: [lab]`. Sie ist Teil der Teststrategie und wird in der Probe-IPA nachgewiesen (Fehlerfall provozieren).

---

## 3. IPA: Promotion-Strategie Lab → T1

Thema: Wie kommt eine Version sicher von Lab bis auf beide T1-Cluster? Es gibt zwei bewertete Entscheide (Freight-Modell und Promotion-Strategie). Die Verifikation aus der Probe-IPA ist Ausgangslage. Zielbild: [VERSIONEN.md §8](VERSIONEN.md#8-promotion-mit-kargo).

### 3.1 Freight-Modell (bewerteter Variantenentscheid 1)

**a) Aufteilung der Komponenten**

| | A: ein Warehouse | B: ein Warehouse pro Komponente | C: Artemis-Bundle + Operatoren getrennt |
|---|---|---|---|
| Aufbau | 1 Warehouse mit 4 Subscriptions (ArkMQ, PF, Broker, Init) | 4 Warehouses, je eine eigene Pipeline | Warehouse `artemis` (Broker + Init), Warehouse `operators` (ArkMQ + PF). Jede Stage verlangt beide (`requestedFreight` mit 2 Einträgen) |
| Broker = Init (Major.Minor) | im Warehouse abbildbar | nicht abbildbar, Kargo kennt keine Abhängigkeit zwischen Pipelines | im Warehouse `artemis` |
| ArkMQ ↔ Broker-Version | in einer Freight, abbildbar | nicht abbildbar | **über zwei Warehouses verteilt**, nicht abbildbar |
| Reihenfolge ArkMQ vor PF | in derselben Promotion | über eigene Stages pro Komponente, viermal so viele Stages | in der Promotion `operators` |
| Nachvollziehbarkeit | 1 Freight = 1 getestete Gesamtkombination | Gesamtstand aus 4 Pipelines zusammensuchen | 2 Freights pro Stage |
| Komponente einzeln ausrollen | nein (unveränderte Teile sind ein No-op) | ja | teilweise |
| Erweiterung auf P1 | nur Stages ergänzen | 4 × Stages pro Umgebung | 2 Pipelines pro Umgebung |

ArkMQ und Broker-Version sind enger gekoppelt als ursprünglich angenommen ([VERSIONEN.md §5](VERSIONEN.md#5-abhängigkeiten-zwischen-den-versionen)). Damit spricht noch mehr für A: Bei C liegen ArkMQ und Broker in verschiedenen Warehouses.

**b) Wie die Regel im Warehouse steht** ([VERSIONEN.md §8.2](VERSIONEN.md#82-wie-aus-den-versionen-eine-freight-wird))

| | 1: `freightCreationCriteria` | 2: feste Vorgabe von Major.Minor |
|---|---|---|
| Wie | Ausdruck vergleicht Major.Minor von Broker und Init | `allowTags`/`semverConstraint` für Broker, Init und ArkMQ auf die aktuelle Minor |
| Deckt ab | Broker = Init | Broker = Init **und** ArkMQ ↔ Broker |
| Minor-Upgrade von Artemis | automatisch, sobald alle Images da sind | bewusste Änderung am Warehouse (passt zum Blog-Post) |
| Patches, neue Init-Builds, PF-Releases | automatisch | automatisch |
| Blockade | eine fehlende Komponente blockiert alle anderen | keine, 2.58 ist bis zur Änderung unsichtbar |
| Matrix nötig | ja, für ArkMQ ↔ Broker | nein |

**Empfehlung: A mit fester Vorgabe (A2).** Jede Freight ist eine vollständige, gültige Kombination. Minor-Upgrades sind bei MQP selten, kommen mit einem ArkMQ-Release und sind ohnehin ein bewusster Entscheid. Die Prüfung K aus der Probe-IPA bleibt als zweite Sicherung.

**KO-Kriterien:** ungültige Kombination darf nicht promotbar sein; P1 darf nicht beschrieben werden. B scheidet aus.

### 3.2 Promotion-Strategie (bewerteter Variantenentscheid 2)

Vier Teilentscheide in einer Bewertung.

#### a) Stage-Topologie

| | Linear | Parallel (Fan-out) |
|---|---|---|
| Aufbau | lab → e1 → t1-nucleus → t1-pcilight | lab → e1 → {t1-nucleus, t1-pcilight}, beide mit `sources.stages: [e1]` |
| Durchlaufzeit | länger (eine Verifikation mehr in Serie) | kürzer |
| Risiko | Fehler auf t1-nucleus erreicht pcilight nicht | Fehler trifft beide T1-Cluster gleichzeitig |
| Übertragbarkeit auf P1 | p1 hängt an t1-pcilight | p1-Stages verlangen `availabilityStrategy: All` über beide T1-Stages |

**Empfehlung: parallel.** E1 hat die Kombination schon geprüft. Der Stack ist auf beiden T1-Clustern gleich; ob sich nucleus und pcilight sonst relevant unterscheiden (Netzzone, Last), wird in der Ist-Analyse geklärt. Wichtiger: „P1 erst, wenn beide T1 grün sind“ wird mit `availabilityStrategy: All` eine Konfiguration statt einer Konvention.

#### b) Komponentenebene

| | In einer Promotion nacheinander | Getrennte Stages pro Komponente |
|---|---|---|
| Aufbau | ArkMQ-App sync → `argocd-wait` → PF-App sync | z. B. `e1-arkmq` → `e1-pf` |
| Hängt zusammen mit | Freight-Modell A oder C | Freight-Modell B |

**Empfehlung: in einer Promotion nacheinander.** Folgt aus dem Freight-Modell A.

#### c) Freigabe und Git-Weg pro Übergang

In Open Source gibt es zwei Arten von Freigabe: automatisch (`autoPromotionEnabled: true`) oder manuell (Auto-Promotion aus, `promote`-Recht nur für berechtigte Personen). `wait-for-approval` (Enterprise) fällt weg. Als Zusatz gibt es `requiredSoakTime`.

| Übergang | Variante 1: alles automatisch | Variante 2: gemischt | Variante 3: alles manuell |
|---|---|---|---|
| → lab | auto | auto | manuell |
| lab → e1 | auto | **manuell nach Blog-Post** | manuell |
| e1 → t1 | auto | **manuell** + `requiredSoakTime` 336h auf e1 | manuell |
| Git | Push | Push | Push |

**Empfehlung: Variante 2.**

- Die Freigabe liegt immer in Kargo, Git überall per Push. Eine Art Freigabe, eine Historie, die Kargo-Rechte decken alles ab (C08).
- Der Blog-Post vor E1 und die zwei Wochen Pause aus `documentation/docs/infra/guides/platform.md` werden zu festen Stellen im Ablauf.
- Die manuellen Freigaben sind über RBAC umgesetzt: Rolle mit `promote` auf `resourceNames: [e1]` bzw. `[t1-nucleus, t1-pcilight]`.
- Für die Demo einen kurzen `requiredSoakTime` setzen und begründen.
- KO-Kriterium für Variante 1: „darf andere Teams nicht ohne Absprache beeinträchtigen“.

Ein MR (`git-open-pr` + `git-wait-for-pr`) ist nur dort sinnvoll, wo ein Vier-Augen-Prinzip nötig ist. Das gilt für **P1** (Change-Prozess, Muster wie auf management-a: Kargo öffnet ein MR, das im SPOC-Board landet), nicht für E1 und T1.

#### d) Gleiche Schritte oder pro Stage unterschiedlich

| | Gleiche Schritte für alle Stages | Schritte pro Stage unterschiedlich |
|---|---|---|
| Umsetzung | eine `PromotionTask`, pro Stage nur andere Werte (Cluster, Pfade, Apps) | eigenes Template pro Stage |
| Lab testet das Ausrollen mit | ja | nur teilweise |
| Pflege | ein Ort | pro Stage |
| Übertragbarkeit auf P1 | neue Stage ruft dieselbe Task auf | neues Template |

**Empfehlung: gleiche Schritte, unterschiedliche Tore.** Strenger werden mit der Stage nur die Kontrollen davor (Freigabe, Wartezeit) und danach (Verifikation). Die Kompatibilitätsprüfung K läuft in jeder Stage. Mit Variante 2 aus c) gibt es keine Ausnahme mehr, weil Git überall per Push läuft.

#### Rollback (für das eigene Kriterium Gate- und Rollback-Strategie)

- Rollback = ältere Freight erneut auf die Stage promoten. Kargo setzt dann einen Auto-Promotion-Hold (`status.autoPromotionHolds`). Er wird aufgehoben, sobald wieder der aktuelle Kandidat promotet wird.
- Grenzen in der Doku festhalten: CRD-Downgrade und Broker-Journal ([Notizen §8](NOTIZEN.md#8-technische-stolpersteine-für-kargo), Punkt 9). Ein Rollback des PF-Operators innerhalb derselben CRD-Version ist unkritisch, einer über eine Artemis-Minor-Version nicht.
- Hotfix: `kargo approve` macht eine Freight für eine Stage verfügbar, ohne die Upstream-Verifikation. Das sollte nur Admins erlaubt sein.

---

## 4. Vorgehen bei der Bewertung (A16)

A16 verlangt, Kriterien und Gewichtung **vor** der Bewertung festzulegen. Vorschlag:

| Kriterium | Probe-IPA (Testarten) | IPA (Freight-Modell) | IPA (Promotion) |
|---|---|---|---|
| Verhindert ungültige Kombinationen / fängt Fehler ab | 30 % | 30 % | 15 % |
| Auswirkung auf andere Teams (E1/T1) | 20 % | 10 % | 25 % |
| Nachvollziehbarkeit / Auditierbarkeit | 10 % | 20 % | 15 % |
| Aufwand in der verfügbaren Zeit | 15 % | 20 % | 15 % |
| Laufzeit (Demo, Betrieb) | 15 % | – | 15 % |
| Übertragbarkeit auf weitere Stages / P1 | 10 % | 20 % | 15 % |

**KO-Kriterien (für alle):** nur Open-Source-Funktionen ohne Beta-Label; kein Zugriff auf P1; README-Ergebnisse erfüllt.

Das KO-Kriterium „nur stabile Open-Source-Funktionen“ lohnt sich explizit, weil es die Recherche in der Doku als Nachweis zeigt (A09 Fachkompetenz).

---

## 5. In Lab zu prüfen

Diese Punkte sind in der Doku nicht eindeutig beschrieben und sollten vor der Probe-IPA kurz in Lab getestet werden (Vorbereitung, nicht bewertet):

- [ ] Laufen die Argo-Rollouts-CRDs **und** der Rollouts-Controller im Kargo-Cluster? (Ohne sie scheitert jede Verifikation.)
- [ ] Ist Prometheus mit den `monitor_messages_*`-Metriken vom Kargo-Cluster aus erreichbar?
- [ ] Wie lange dauert eine Verifikation real (Rollout aller Broker + Messfenster)? Wichtig für die Demo-Planung.
- [ ] Regel „Broker = neueste Version in der ArkMQ-Liste“ bestätigen.
- [ ] Zeigt `middleware-docker` für `arkmq-org/activemq-artemis-operator` auch Tags, die noch nie gezogen wurden?
- [ ] Für die IPA: Funktionieren `let`, `split` und `trimPrefix` in `freightCreationCriteria`? (nur nötig, falls dort Variante A1 gewählt wird)

---

## 6. Quellen

- Kargo Docs, Verification: https://docs.kargo.io/user-guide/how-to-guides/verification
- Kargo Docs, Working with Stages: https://docs.kargo.io/user-guide/how-to-guides/working-with-stages
- Kargo Docs, Working with Warehouses: https://docs.kargo.io/user-guide/how-to-guides/working-with-warehouses
- Kargo Docs, Working with Projects (ProjectConfig): https://docs.kargo.io/user-guide/how-to-guides/working-with-projects
- Kargo Docs, Working with Freight: https://docs.kargo.io/user-guide/how-to-guides/working-with-freight
- Kargo Docs, Promotion Steps Reference (mit Beta- und Enterprise-Labels): https://docs.kargo.io/user-guide/reference-docs/promotion-steps/
- Kargo Docs, Expression Language: https://docs.kargo.io/user-guide/reference-docs/expressions
- Kargo Docs, Access Controls: https://docs.kargo.io/user-guide/security/access-controls
- Kargo Docs, Basic Installation: https://docs.kargo.io/operator-guide/basic-installation
- Kargo Quellcode v1.12.4: https://github.com/akuity/kargo/tree/v1.12.4
