# Variantenentscheide Test- und Promotion-Strategie

Vorschläge für die Variantenentscheide der Probe-IPA und der IPA (Stand 24.10.2026). Grundlage sind die [Notizen](NOTIZEN.md), das [README](README.md) und die offizielle Kargo-Doku (docs.kargo.io, aktuelles Release **v1.12.4** vom 12.10.2026).

## Inhalt

1. [Was Kargo on-prem (Open Source) bietet](#1-was-kargo-on-prem-open-source-bietet)
2. [Probe-IPA](#2-probe-ipa)
3. [IPA](#3-ipa)
4. [Vorgehen bei der Bewertung (A16)](#4-vorgehen-bei-der-bewertung-a16)
5. [In Lab zu prüfen](#5-in-lab-zu-prüfen)
6. [Quellen](#6-quellen)

---

## 1. Was Kargo on-prem (Open Source) bietet

Alle Varianten unten nutzen nur Funktionen, die in der Open-Source-Version stabil (ohne Label) dokumentiert sind. Funktionen mit **Beta**- oder **Enterprise Beta**-Label sind ausgeschlossen.

### Verwendbar

| Bereich | Funktion | Wofür bei uns |
|---|---|---|
| Warehouse | Subscriptions `image`, `git`, `chart` | Broker-, Init- und PF-Image; ArkMQ über Git-Tags oder Operator-Image |
| Warehouse | `freightCreationCriteria` (Expression) | Freight nur erzeugen, wenn die Kombination gültig ist (z. B. Broker = Init) |
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
| Promotion-Steps | `git-clone`, `kustomize-set-image`, `yaml-update`, `git-commit`, `git-push`, `git-open-pr`, `git-wait-for-pr`, `argocd-update`, `argocd-wait`, `http`, `fail`, `set-metadata` | Overlays schreiben, Apps in Reihenfolge synchronisieren, Vorprüfungen |
| Expressions | `imageFrom`, `commitFrom`, `semverParse`, `semverDiff`, `configMap`, `warehouse` | Versionen auslesen, Major.Minor vergleichen, Matrix aus ConfigMap lesen |

### Ausgeschlossen (Beta oder Enterprise)

| Funktion | Label | Ersatz in Open Source |
|---|---|---|
| `wait-for-approval`, `get-user-input` | Enterprise Beta | Auto-Promotion für die Stage aus + `promote`-Recht nur für berechtigte Personen |
| `send-message` | Enterprise Beta | Info ans Team ausserhalb von Kargo (Chat, Mail), im Betriebskonzept festhalten |
| `git-merge-pr` | Beta | `git-open-pr` + `git-wait-for-pr`, Merge von Hand in GitLab |
| `jira`, `snow-*`, `gh-*`, `gha-*`, `tf-*` | Enterprise Beta | nicht benötigt |
| `toml-update`, `toml-parse`, `github-push` | Beta | nicht benötigt |

### Wichtig für die Verifikation

- Kargo verwendet `AnalysisTemplate` und `AnalysisRun` von Argo Rollouts. Laut `values.yaml` des Kargo-Charts wertet der Kargo-Controller die AnalysisRuns **selbst** aus (`controller.rollouts.integrationEnabled: true`). Er braucht nur die **CRDs von Argo Rollouts**, keinen eigenen Rollouts-Controller. Fehlen die CRDs, schaltet Kargo die Integration still ab und jede Verifikation schlägt fehl. Damit ist der offene Punkt im README fast beantwortet: Es muss nur geprüft werden, ob die CRDs im Kargo-Cluster installiert sind.
- Die Jobs einer AnalysisRun laufen standardmässig im Cluster der Kargo-Control-Plane, nicht in Lab, E1 oder T1. Für Status-Checks braucht der Job also einen Zugriff auf den Ziel-Cluster (siehe [Entscheid 3.3](#33-ausführung-der-verifikationen)).
- Solange eine Verifikation läuft, warten andere Promotionen auf dieselbe Stage. Das ist für die Demo relevant (F&P04).

---

## 2. Probe-IPA

Thema: Freight-Modell und Promotion von Lab nach E1. Laut README gibt es nur **einen** bewerteten Variantenentscheid (A16). Ich empfehle, dass das Freight-Modell dieser Entscheid bleibt. Die Promotion-Mechanik (2.2) und der Health-Check (2.3) werden im Konzept begründet festgelegt, aber nicht als eigene Evaluation bewertet. So bleibt der Aufwand klein.

### 2.1 Freight-Modell (bewerteter Variantenentscheid)

| | A: ein Warehouse | B: ein Warehouse pro Komponente | C: Artemis-Bundle + Operatoren getrennt |
|---|---|---|---|
| Aufbau | 1 Warehouse mit 4 Subscriptions (ArkMQ, PF-Image, Broker, Init) | 4 Warehouses, je eine eigene Pipeline lab → e1 | Warehouse `artemis` (Broker + Init), Warehouse `operators` (ArkMQ + PF). Jede Stage verlangt beide (`requestedFreight` mit 2 Einträgen) |
| Broker = Init (Major.Minor) | `freightCreationCriteria` verhindert ungültige Freight | nicht abbildbar, Kargo kennt keine Abhängigkeit zwischen Pipelines | `freightCreationCriteria` im Warehouse `artemis` |
| PF-Operator braucht passendes ArkMQ | in einer Freight festgelegt, Reihenfolge in den Promotion-Steps | nur über Konvention | im Warehouse `operators` festgelegt |
| Reihenfolge ArkMQ vor PF | `argocd-update` ArkMQ-App, danach PF-App in derselben Promotion | über eigene Stages pro Komponente, doppelt so viele Stages | in der Promotion `operators`, aber Artemis und Operatoren können unabhängig voneinander promotet werden |
| Nachvollziehbarkeit | 1 Freight = 1 getestete Gesamtkombination | Gesamtstand muss aus 4 Pipelines zusammengesucht werden | 2 Freights pro Stage |
| Anzahl Freights | hoch (jede neue Version irgendeiner Komponente) | pro Pipeline niedrig, total am höchsten | mittel |
| Komponente einzeln ausrollen | nein, immer die ganze Kombination (unveränderte Teile sind ein No-op) | ja | teilweise |
| Erweiterung auf T1/P1 | einfach, nur Stages ergänzen | 4 × Stages pro Umgebung | 2 Pipelines pro Umgebung |
| Aufwand Probe-IPA | klein | gross | mittel |

**Empfehlung: Variante A.** Unser Hauptproblem ist, dass ungültige Kombinationen heute nur durch Erfahrung verhindert werden. A ist die einzige Variante, bei der jede Freight eine vollständige und geprüfte Kombination ist. Sie lässt sich auch am einfachsten auf T1 und P1 erweitern. Der Nachteil, dass nichts einzeln ausgerollt werden kann, ist bei uns eher ein Vorteil: Eine Kombination soll immer als Ganzes durch die Stages laufen.

C ist die ehrliche Alternative, falls das Init-Image deutlich häufiger neu gebaut wird als die Operatoren und dadurch zu viele Freights entstehen. B scheidet am KO-Kriterium „ungültige Kombinationen verhindern“ aus.

**Mögliche KO-Kriterien:** ungültige Kombination Broker/Init darf nicht promotbar sein; P1 darf nicht beschrieben werden.

**Zu beachten:** Das Init-Image hat das Tag-Format „Artemis-Version + Zeitstempel“. Die Criteria-Expression muss den Major.Minor-Teil aus dem Tag ziehen. In `freightCreationCriteria` sind nur die `repoURL`-Formen von `imageFrom` und `commitFrom` erlaubt. Ob `semverParse` dort auch verwendbar ist, sagt die Doku nicht → in Lab prüfen. Plan B ist ein `fail`-Step am Anfang der Promotion (siehe 3.1).

### 2.2 Promotion-Mechanik (festgelegt, nicht bewertet)

| Frage | Optionen | Vorschlag Probe-IPA | Begründung |
|---|---|---|---|
| Wie kommt die Änderung ins Repo? | a) `git-push` direkt auf `main`; b) `git-open-pr` + `git-wait-for-pr`, Merge von Hand in GitLab; c) eigener Branch pro Stage, auf den die ArgoCD-App zeigt | **a für Lab, b für E1** | Lab ist eine Spielwiese. Für E1 ist ein MR zusätzlich der sichtbare Nachweis für die Teams. Das automatische Mergen (`git-merge-pr`) ist Beta und fällt weg |
| Zwei Repos pro Cluster | ein Promotion-Template mit zwei `git-clone`/`git-push`-Blöcken | eine Promotion schreibt beide Repos | atomar aus Sicht von Kargo, auch wenn es zwei Commits sind |
| Reihenfolge | `argocd-update` ArkMQ-App → `argocd-wait` → `argocd-update` PF-App | so umsetzen | PF-Operator hängt von ArkMQ ab, nicht umgekehrt ([Notizen §4](NOTIZEN.md#4-abhängigkeit-pf-operator-und-arkmq)) |
| Freigabe | Auto-Promotion pro Stage über `ProjectConfig.promotionPolicies` | `lab` automatisch, `e1` manuell | entspricht dem README (Probe-IPA: manuelle Promotion nach E1) |
| P1 ausschliessen | keine P1-Stage; Kargo-Credentials ohne Schreibrecht auf P1-Pfade; CODEOWNERS | alle drei | C08 |

Bei Variante b muss vorher klar sein, ob `git-open-pr` mit unserem GitLab (`gitlab.pnet.ch`) funktioniert → in Lab prüfen.

### 2.3 Health-Check (fester Umfang, kein Variantenentscheid)

Das README verlangt als Ergebnis 4 `ActiveMQBrokersReady=True`. Das entspricht den Testarten **1 + 2** aus den Notizen (Variante A, minimal). Für die Probe-IPA wird also keine Teststrategie bewertet, sondern genau dieser Umfang umgesetzt:

- Test 1 (Operator-Health) ist durch die implizite Argo-CD-Health gratis.
- Test 2 (Reconcile-Status) ist ein `AnalysisTemplate` mit `job`-Provider, der `kubectl wait --for=condition=ActiveMQBrokersReady pfactivemq --all -A` gegen den Ziel-Cluster ausführt.

Damit ist auch der Widerspruch zwischen Notizen und README aufgelöst: In der Probe-IPA gibt es keinen Testarten-Entscheid. Er ist Thema der IPA.

---

## 3. IPA

Thema: Verifikation und Gates von Lab bis T1. Laut README gibt es zwei bewertete Entscheide (Testarten und Promotion-Strategie). Dazu kommt ein kleinerer technischer Entscheid (3.3), der nicht bewertet werden muss, aber im Konzept begründet sein sollte.

### 3.1 Testarten (bewerteter Variantenentscheid 1)

Die Tests werden als Bündel verglichen ([Notizen §10](NOTIZEN.md#10-variantenentscheide)). Neu gegenüber den Notizen: Der **Kompatibilitätscheck (5) gehört vor die Promotion**, nicht in die Verifikation. Eine Verifikation läuft erst nach dem Deployment. Dann ist eine ungültige Kombination schon im Cluster. Kargo bietet dafür zwei Stellen, die beide Open Source sind:

- `freightCreationCriteria` im Warehouse: Ungültige Freight entsteht gar nicht erst.
- `fail`-Step am Anfang der Promotion: Er liest die Matrix mit `configMap('mqp-compat-matrix')` und bricht ab, bevor etwas geschrieben wird.

| | A, minimal | B, Funktion | C, umfassend |
|---|---|---|---|
| Inhalt | 1 + 2 | 1 + 2 + 3 + 5 | 1 + 2 + 3 + 4 + 5 |
| 1 Operator-Health | implizit (Argo CD) | implizit | implizit |
| 2 Reconcile-Status | Job | Job | Job |
| 3 Funktion (`monitor_messages_*`) | – | `prometheus`-Provider | `prometheus`-Provider |
| 4 Canary-Instanz | – | – | Job: `PFActiveMQ` in eigenem Namespace anlegen, Nachricht senden, löschen |
| 5 Kompatibilität | – | vor der Promotion (Criteria/`fail`) | vor der Promotion (Criteria/`fail`) |
| Fängt ab | Crash, CRD-/API-Fehler | dazu: Broker verarbeitet nichts, ungültige Kombination | dazu: neue Instanzen werden nicht mehr korrekt angelegt |
| Laufzeit pro Stage | Minuten | + Messfenster (z. B. 10 min) | + Anlegen und Löschen der Instanz |
| Auswirkung auf andere Teams | keine | keine (nur lesen) | legt Ressourcen in E1/T1 an → Absprache nötig |
| Aufwand | sehr klein | klein | mittel |
| README-Ergebnis „mind. 3 Testarten“ | **nicht erfüllt** | erfüllt | erfüllt |

**Empfehlung: Variante C, aber mit Test 4 nur in `lab` und `e1`.** Auf T1 läuft B. Begründung:

- Test 4 ist die aussagekräftigste Prüfung für den Fall „Upgrade kaputt, bestehende Broker laufen aber weiter“. Das fängt sonst kein Test ab.
- Auf T1 (Testteams, gleiche Topologie wie P1) sollen möglichst keine fremden Ressourcen entstehen. Dort reicht B, weil die Kombination schon in Lab und E1 die Canary-Prüfung bestanden hat.
- Damit ist die Variantenbewertung nicht nur „C gewinnt, weil es am meisten ist“, sondern es gibt eine begründete Abstufung pro Stage. Das macht A16 stärker.

Variante A fällt am KO-Kriterium „README-Ergebnis 1 erfüllt“ heraus. B ist die Rückfallvariante, falls die Canary-Instanz in E1 vom Team nicht erlaubt wird.

**Welche Tests blockieren?** In Kargo blockiert jede fehlgeschlagene Verifikation die Freight für die nächste Stage. Soll ein Test nur informieren, kann die Metrik im AnalysisTemplate auf `dryRun` gesetzt werden. Das ist eine Funktion von Argo Rollouts. Ob Kargo sie auswertet, steht nicht in der Kargo-Doku → in Lab prüfen. Mein Vorschlag: Test 3 in Lab nur informativ (Lab-Broker haben wenig Last und das Messfenster ist unzuverlässig), ab E1 blockierend.

**Upgrade-Pfad (6)** bleibt Ausblick für D03. Er bräuchte eine Stage mit dem P1-Stand (2.53 / v2.1.7 / 0.17.1), und die gibt es in keinem Cluster.

### 3.2 Promotion-Strategie (bewerteter Variantenentscheid 2)

Die Promotion-Strategie hat drei Achsen. Ich würde sie als **drei Teilentscheide in einer Bewertung** zeigen, sonst wird die Tabelle unübersichtlich.

#### a) Stage-Topologie

| | Linear | Parallel (Fan-out) |
|---|---|---|
| Aufbau | lab → e1 → t1-nucleus → t1-pcilight | lab → e1 → {t1-nucleus, t1-pcilight}, beide mit `sources.stages: [e1]` |
| Durchlaufzeit | länger (eine Verifikation mehr in Serie) | kürzer |
| Risiko | Fehler auf t1-nucleus erreicht pcilight nicht | Fehler trifft beide T1-Cluster gleichzeitig |
| Übertragbarkeit auf P1 | p1 hängt an t1-pcilight | p1-Stages verlangen `availabilityStrategy: All` über beide T1-Stages |
| Aussagekraft T1 als Generalprobe | gut | gut, und gleiche Struktur wie später P1 |

**Empfehlung: parallel.** Die Verifikation auf E1 hat dieselbe Kombination schon geprüft. Der Stack (Operatoren, Images) ist auf beiden T1-Clustern gleich. Ob sich nucleus und pcilight sonst relevant unterscheiden (Netzzone, Last), sollte in der Ist-Analyse kurz geklärt werden. Wichtiger: Das P1-Konzept lässt sich mit `availabilityStrategy: All` direkt daran anhängen. „P1 erst, wenn beide T1 grün sind“ ist dann eine Konfiguration und keine Konvention mehr.

#### b) Komponentenebene

| | In einer Promotion nacheinander | Getrennte Stages pro Komponente |
|---|---|---|
| Aufbau | Promotion-Steps: ArkMQ-App sync → `argocd-wait` → PF-App sync | z. B. `e1-arkmq` → `e1-pf` |
| Hängt zusammen mit | Freight-Modell A oder C | Freight-Modell B |
| Verifikation | einmal für die ganze Kombination | pro Komponente, Zwischenzustände werden mitgetestet |

**Empfehlung: in einer Promotion nacheinander.** Das folgt aus dem Freight-Modell der Probe-IPA. Die IPA muss diesen Teilentscheid nur kurz begründen, nicht neu aufrollen.

#### c) Freigabe pro Übergang

In Open Source gibt es zwei Gate-Arten: automatisch (`autoPromotionEnabled: true`) oder manuell (Auto-Promotion aus, `promote`-Recht nur für berechtigte Personen). Die Enterprise-Funktion `wait-for-approval` fällt weg. Als Zusatz gibt es `requiredSoakTime`.

| Übergang | Variante 1: alles automatisch | Variante 2: gemischt | Variante 3: alles manuell |
|---|---|---|---|
| → lab | auto | auto | manuell |
| lab → e1 | auto | auto nach Verifikation | manuell |
| e1 → t1 | auto | **manuell** + `requiredSoakTime` auf e1 | manuell |

**Empfehlung: Variante 2**, wie im README vorgesehen. Ergänzungen:

- Für `e1` `selectionPolicy: MatchUpstream` setzen. So wird automatisch genau die Freight promotet, die in Lab läuft, und nicht einfach die neueste.
- `requiredSoakTime` auf dem Übergang e1 → t1 (z. B. `24h` im Betrieb). Für die Demo einen kurzen Wert setzen und das in der Doku begründen.
- Das manuelle Gate ist über RBAC umgesetzt: Rolle mit `promote` auf `resourceNames: [t1-nucleus, t1-pcilight]`, gebunden an die Gruppe, die T1 freigeben darf. Das deckt C08 direkt ab.
- KO-Kriterium für Variante 1: „darf andere Teams auf T1 nicht ohne Absprache beeinträchtigen“.

#### Rollback (für das eigene Kriterium Gate- und Rollback-Strategie)

- Rollback = ältere Freight erneut auf die Stage promoten. Kargo setzt dann automatisch einen Auto-Promotion-Hold (`status.autoPromotionHolds`). Die Stage wird nicht sofort wieder überschrieben. Der Hold wird aufgehoben, sobald wieder der aktuelle Kandidat promotet wird.
- Grenzen in der Doku festhalten: CRD-Downgrade und Broker-Journal ([Notizen §8](NOTIZEN.md#8-technische-stolpersteine-für-kargo), Punkt 9). Ein Rollback des PF-Operators innerhalb derselben CRD-Version ist unkritisch. Ein Rollback über eine Artemis-Minor-Version ist es nicht.
- Hotfix: `kargo approve` macht eine Freight für eine Stage verfügbar, ohne die Upstream-Verifikation. Das sollte nur Admins erlaubt sein (dafür braucht es die gleichen Rechte wie für die Promotion).

### 3.3 Ausführung der Verifikationen (technischer Entscheid, nicht bewertet)

Die Jobs laufen im Kargo-Cluster. Die Status-Checks brauchen aber Daten aus Lab, E1 und T1.

| Variante | Wie | Pro | Contra |
|---|---|---|---|
| a) Job mit Kubeconfig | Pro Ziel-Cluster ein ServiceAccount mit nur Leserechten auf `pfactivemqs` und `activemqartemises`, Token als Secret im Kargo-Projekt | einfach, genau | Secrets pro Cluster verwalten |
| b) Prometheus | Status über kube-state-metrics (Custom Resource State) abfragen | kein Cluster-Zugriff | kube-state-metrics muss die CRs exportieren, eher nicht vorhanden |
| c) `web`-Provider | direkt gegen die K8s-API | kein Job-Pod | Token im Template, unübersichtlich |

**Empfehlung: a für Test 2 und 4, Prometheus für Test 3.** Die Leserechte pro Cluster sind gleich ein Baustein für C08 (Least Privilege). Für Test 4 braucht der ServiceAccount zusätzlich Schreibrechte, aber nur im Canary-Namespace.

---

## 4. Vorgehen bei der Bewertung (A16)

A16 verlangt, Kriterien und Gewichtung **vor** der Bewertung festzulegen. Vorschlag:

| Kriterium | Gewicht Probe-IPA (Freight-Modell) | Gewicht IPA (Testarten) | Gewicht IPA (Promotion) |
|---|---|---|---|
| Verhindert ungültige Kombinationen / fängt Fehler ab | 30 % | 30 % | 15 % |
| Auswirkung auf andere Teams (E1/T1) | 10 % | 20 % | 25 % |
| Nachvollziehbarkeit / Auditierbarkeit | 20 % | 10 % | 15 % |
| Aufwand in der verfügbaren Zeit | 20 % | 15 % | 15 % |
| Laufzeit (Demo, Betrieb) | – | 15 % | 15 % |
| Übertragbarkeit auf P1 | 20 % | 10 % | 15 % |

**KO-Kriterien (für alle):** nur Open-Source-Funktionen ohne Beta-Label; kein Zugriff auf P1; README-Ergebnisse erfüllt.

Das KO-Kriterium „nur stabile Open-Source-Funktionen“ lohnt sich explizit, weil es die Recherche in der Doku als Nachweis zeigt (A09 Fachkompetenz).

---

## 5. In Lab zu prüfen

Diese Punkte sind in der Doku nicht eindeutig beschrieben und sollten vor der Probe-IPA kurz in Lab getestet werden (Vorbereitung, nicht bewertet):

- [ ] Sind die Argo-Rollouts-CRDs im Kargo-Cluster installiert? (Ohne sie scheitert jede Verifikation.)
- [ ] Funktioniert `semverParse` in `freightCreationCriteria`, und lässt sich Major.Minor aus dem Init-Image-Tag (Version + Zeitstempel) ziehen?
- [ ] Unterstützt `git-open-pr` / `git-wait-for-pr` unser GitLab (`gitlab.pnet.ch`) inklusive Token-Rechten?
- [ ] ArkMQ abonnieren: Git-Subscription auf die Tags von `arkmq-org/activemq-artemis-operator` (braucht Zugriff auf GitHub oder einen Mirror in GitLab) oder Image-Subscription auf das Operator-Image über unsere Registry
- [ ] Wertet Kargo `dryRun` in AnalysisTemplates aus?
- [ ] Ist Prometheus mit den `monitor_messages_*`-Metriken vom Kargo-Cluster aus erreichbar?
- [ ] Wie lange dauert eine Verifikation real (Rollout aller Broker + Messfenster)? Wichtig für die Demo-Planung.

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
- Kargo Helm-Chart `values.yaml` (Rollouts-Integration): https://github.com/akuity/kargo/blob/main/charts/kargo/values.yaml
