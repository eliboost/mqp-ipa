# mqp-ipa

Automatisierung der MQP-Infrastruktur (ActiveMQ Artemis) mit Kargo.

Fachrichtung: **Plattformentwicklung**

## Übersicht

| | Probe-IPA | IPA |
|---|---|---|
| Thema | **Teststrategie**: Welche Verifikationen sichern eine neue Version ab? | **Promotion-Strategie**: Wie kommt eine Version sicher durch alle Stages? |
| Abschluss | Ende Januar | voraussichtlich April (Termin offen) |
| Kargo | Test-Instanz (Control-Plane Lab-D) | Prod-Instanz (Control-Plane management-a) |
| Umgebungen | Lab-D → Lab-C (beide Testcluster, nur MQP) | (Lab →) E1 → T1 (nucleus und pcilight) |
| Prod | ausgeschlossen | ausgeschlossen, nur als Konzept |
| Variantenentscheid | Welche Testarten laufen als Verifikation, und welche blockieren? | Wie wird die Versionsabhängigkeit im Freight-Modell abgebildet? Welche Promotion-Strategie (Topologie, Freigaben)? |
| Promotion | nur so viel wie nötig, damit die Verifikation läuft (Vorbereitung) | der eigentliche Inhalt |

```
Probe-IPA:  Kargo Lab-D (Test-Instanz)
            lab-d ──► lab-c
            (einfache Promotion als Voraussetzung,
             bewertet wird die Verifikation in beiden Stages)

IPA:        Kargo management-a (Prod-Instanz)
            (lab ──►) e1-pfnet-a ──┬──► t1-nucleus-a
                                   └──► t1-pcilight-a
            (Freight-Modell, Gates, Rechte, Rollback;
             Verifikation aus der Probe-IPA kommt aus Git)

            p1-*: nur Konzept, Kargo hat keine Stage und keine Rechte
```

**Warum diese Reihenfolge:** Für MQP ist der Schutz vor falschen Kombinationen wichtiger als ein schnellerer Rollout. Die Probe-IPA entwickelt und beweist diesen Schutz auf zwei eigenen Testclustern. Die Promotion über alle Stages ist das grössere Thema mit mehr Entscheiden und Beteiligten und passt darum in die IPA. Sie wendet die Tests der Probe-IPA auf E1 und T1 an.

**Warum Lab-D → Lab-C für die Probe-IPA:** Beide Cluster gehören MQP und sind zum Kaputtmachen da. Ihre Kargo-Controller melden sich bei derselben Test-Control-Plane auf Lab-D, und beide hängen an `argocd-lab`. So lässt sich die Sperre zwischen zwei Stages beliebig oft und ohne Risiko für andere Teams nachweisen. E1 meldet sich bei management-a und ist von der Test-Instanz aus nicht erreichbar.

Details: [VARIANTENENTSCHEIDE.md](VARIANTENENTSCHEIDE.md), [TESTARTEN.md](TESTARTEN.md), [VERSIONEN.md](VERSIONEN.md), [NOTIZEN.md](NOTIZEN.md).

## Ausgangslage (gilt für beide Arbeiten)

Das Artemis-Ökosystem besteht aus mehreren Komponenten mit Versionsabhängigkeiten:

- **ArkMQ-Operator** (Upstream, `arkmq-org/arkmq-org-broker-operator`), deployt über `kube-artemis-operator`
- **pf-activemq-operator** (eigener Operator), deployt über `kube-pf-activemq-operator`. Er erzeugt `ActiveMQArtemis`-Ressourcen und hängt damit vom ArkMQ-Operator ab.
- **Broker-Image** (Upstream) und **artemis-custom-init-image** (eigenes Image): Major.Minor müssen übereinstimmen, sonst beendet sich der PF-Operator (`os.Exit(1)`).
- Die Broker-Version muss sehr wahrscheinlich genau der neuesten Version in der Liste des ArkMQ-Release entsprechen (aus dem Code abgeleitet, siehe [VERSIONEN.md §5](VERSIONEN.md#5-abhängigkeiten-zwischen-den-versionen)).

Welche Reihenfolge und Kombination beim Ausrollen funktioniert, weiss man heute nur aus Erfahrung. Weitere Punkte:

- Renovate erstellt pro Cluster ein MR, bis zu 18 MRs pro Upgrade. Freigabe und Reihenfolge sind manuell (Lab → Blog-Post → E1 → T1 → P1, je zwei Wochen Pause laut Plattform-Doku).
- Broker-Updates erkennt Renovate seit der Umbenennung des Images bei ArkMQ v2.2.2 nicht mehr.
- `kube-*`-Repos und `deployments` haben keine CI (kein `kustomize build`, kein Schema-Check).
- Der envtest des PF-Operators testet gegen `releases/latest` von ArkMQ, nicht gegen die deployte Version.
- Der einzige Test nach dem Deployment ist der `artemis-monitor` (Dauer-Canary in `appl-mqp-e2e-<env>`). Er blockiert keine Promotion.
- Versionsstand (Oktober 2026): Lab/E1/T1 Artemis 2.57.0 / ArkMQ v2.2.2 / PF 0.17.5, P1 Artemis 2.53.0 / ArkMQ v2.1.7 / PF 0.17.1.
- Kargo betreiben wir selbst, zweimal: eine **Test-Instanz** (Control-Plane auf Lab-D, Controller auf Lab-D und Lab-C) zum Testen von Kargo selbst, und eine **Prod-Instanz** (Control-Plane auf management-a), bei der sich die Controller aller anderen Cluster melden. Die Plattform ist Ausgangslage, nicht Leistung der Arbeiten.
- Die Build-Kette bis `repo.pnet.ch` (unsere Pipelines, Artifactory-Remotes vor quay.io und GitHub) ist Ausgangslage. Kargo beginnt bei Artifactory.

Umgebungen und ArgoCD-Instanzen:

| Cluster | ArgoCD | Kargo-Controller meldet sich bei | Rolle |
|---|---|---|---|
| `e1-k8s-pfnet-lab-d` | `argocd-lab` | Lab-D (Test-Instanz) | Lab, MQP läuft heute schon |
| Lab-C | `argocd-lab` | Lab-D (Test-Instanz) | Lab, MQP wird für die Probe-IPA aufgebaut |
| `e1-k8s-pfnet-a` | `argocd-itd` | management-a | E1 |
| `t1-k8s-nucleus-a`, `t1-k8s-pcilight-a` | `argocd-itd` | management-a | T1 |
| `p1-k8s-nucleus-a`, `p1-k8s-pcilight-a` | `argocd` | management-a | Prod (tabu) |

---

## Probe-IPA: Teststrategie für die Verifikation von Lab-D nach Lab-C

### Ziel

Eine neue Kombination aus ArkMQ-Operator, PF-Operator, Broker- und Init-Image wird nach jeder Promotion in **Lab-D und Lab-C** automatisch geprüft. Sie erreicht Lab-C nur, wenn die Verifikation in Lab-D erfolgreich war. Die Teststrategie ist so gebaut, dass sie in der IPA unverändert für E1 und T1 übernommen werden kann: Lab-D → Lab-C ist das Modell für Lab → E1.

### Messbare Ergebnisse

1. Ein Testkonzept: Testarten begründet, Testfälle mit erwarteten Ergebnissen, Abstufung zwischen erster und zweiter Stage.
2. AnalysisTemplates für die gewählte Variante, als Verifikation an den Stages `lab-d` und `lab-c`. Mindestens:
   - **Version angekommen, Rollout fertig** (V): die Operatoren und alle Broker laufen mit den Versionen aus der Freight.
   - **Status aller Instanzen** (S): `PFActiveMQ` und `ActiveMQArtemis` sind Ready, ohne neue Fehler nach dem Upgrade.
   - **Nachrichtenfluss** (F): `artemis-monitor`-Metriken, Nachrichten fliessen, keine Verspätungen und Fehler.
3. Eine **Kompatibilitätsprüfung** (K) vor dem Commit: Die Kombination ist laut Matrix erlaubt, sonst bricht die Promotion ab.
4. Nachweis der Sperre: Eine provozierte, ungültige Kombination bleibt in `lab-d` hängen und erreicht `lab-c` nicht.
5. Testprotokoll und Betriebsdoku (Verifikation lesen, neu starten, Fehlerfälle).
6. Alle Kargo-Ressourcen (Warehouse, Stages, `PromotionTask`, AnalysisTemplates) liegen deklarativ in Git und lassen sich in der IPA auf management-a wiederverwenden.

### Variantenentscheid: Testarten

| Variante | Inhalt |
|---|---|
| A, minimal | V + S |
| B, Funktion | K + V + S + F |
| C, umfassend | K + V + S + F + O (Operator stabil) + C (Canary-Instanz anlegen und löschen) |

Mögliche Kriterien: Aussagekraft, Laufzeit, Stabilität, Aufwand, Auswirkung auf andere Teams. Details und Empfehlung: [VARIANTENENTSCHEIDE.md §2](VARIANTENENTSCHEIDE.md#2-probe-ipa-teststrategie-lab-d--lab-c), [TESTARTEN.md](TESTARTEN.md).

### Abgrenzung

- Nur die Testcluster Lab-D und Lab-C und nur die Kargo-Test-Instanz. Kein E1, kein T1, kein Prod.
- Die Teststrategie wird auf Testclustern **entwickelt und nachgewiesen**. Angewendet auf E1/T1 wird sie in der IPA.
- Auf Lab-C laufen keine Instanzen anderer Teams. Das wird mit eigenen Test-Instanzen nachgebildet (mindestens eine mit 1 und eine mit 2 Replicas).
- **Freight-Modell und Promotion sind nicht Teil der Bewertung.** Warehouse, Stages `lab-d`/`lab-c` und eine einfache Promotion (ein Warehouse, beide Stages manuell, direkter Push) laufen vor dem Start als Vorbereitung. Sie sind bewusst vorläufig und werden in der IPA evaluiert.
- Der Aufbau von MQP auf Lab-C (Operatoren, Test-Instanzen, `artemis-monitor`, Prometheus) ist Vorbereitung.
- Kargo-Plattform, Zugänge und Build-Kette bis Artifactory sind Ausgangslage.
- Upgrade-Pfad n-1 → n nur als Ausblick.

### Kriterien

**Standardkriterien:** A01–A12 (inklusive A12 Testdurchführung und Dokumentation)

**Pflichtwahlkriterien (unverändert):**

| Slot | Kriterium | Bezug |
|---|---|---|
| A13 | **D02** Standardisierung, Optimierung und Automatisierung | Manuelle Prüfung nach dem Rollout wird durch automatisierte Verifikation ersetzt |
| A14 | **F03** Abnahme einer Server-Infrastruktur oder von Server-Diensten | Abnahmekriterien, Testkonzept, Systemabhängigkeiten, Abnahme pro Stage |

**Wahlkriterien:**

| Slot | Kriterium | Art | Bezug |
|---|---|---|---|
| A15 | **A13** Erhebung und Dokumentation der Bedürfnisse und Umfeld | unverändert | Cluster, ArgoCD-Instanzen, Repos, Build-Kette, Schnittstellen |
| A16 | **A16** Durchführung einer Evaluation | unverändert | Variantenentscheid Testarten |
| A17 | **D01** Analyse | unverändert | Ist-Analyse: heutige Tests und Lücken |
| A18 | **D03** Sicherstellung des Betriebs und der Weiterentwicklung | unverändert | Betriebsdoku der Verifikation, nächste Schritte für die IPA |
| A19 | **E10** Konzept zur Überwachung | angepasst auf Broker statt Netzwerk | Überwachte Elemente, Schwellwerte und Aktionen entsprechen den Erfolgsbedingungen der AnalysisTemplates |
| A20 | **C08** Planung und Implementierung eines Rollenkonzepts | angepasst | Rechte der Verifikations-Jobs auf Lab-D/Lab-C, Least Privilege, kein Zugriff auf andere Cluster |
| A21 | Abbildung von Versionsabhängigkeiten in Prüfungen | eigenes | Abhängigkeiten aus dem Code belegt, als Kompatibilitätsprüfung und Verifikation automatisiert |
| A22 | Testkonzept für Verifikationen | eigenes | Testarten begründet, Testfälle mit erwarteten Ergebnissen, als AnalysisTemplates automatisiert, Fehlerfälle provoziert |

### Risiken

- Die Kargo-Test-Instanz ist zum Testen von Kargo selbst da. Ein Kargo-Upgrade oder ein Experiment während der Probe-IPA kann die Arbeit stören. **Für die Dauer der Probe-IPA Kargo-Version und Konfiguration auf Lab-D einfrieren** (mit dem Team absprechen).
- Lab-D und Lab-C sind zum Kaputtmachen da. Auch andere Tests im Team können MQP dort stören. Zeitfenster absprechen.
- Die Verifikations-Jobs laufen auf Lab-D (Argo-Rollouts-Controller und CRDs müssen dort laufen) und brauchen Leserechte auf Lab-C.
- Prometheus sammelt heute nur Lab-D. Für Lab-C muss dasselbe aufgebaut werden.
- Lab-C bildet E1 nur teilweise ab (keine fremden Instanzen, wenig Last). Die Aussagekraft für E1 ist in der Doku zu begründen.
- Verifikations-Laufzeiten (Rollout aller Broker plus Messzeit) bremsen die Live-Demo (F&P04). Eine verkürzte Demo-Variante vorbereiten.
- Lab-D läuft heute mit Staging-Image und CRD von `main`. Für die Stage `lab-d` festlegen, welcher Stand gilt.
- Die vorläufige Promotion darf den Entscheiden der IPA nicht vorgreifen.

---

## IPA: Promotion-Strategie von Lab bis T1

### Ziel

Die MQP-Komponenten werden mit der Kargo-Prod-Instanz (management-a) **über E1 auf beide T1-Cluster** promotet und nicht mehr über manuelle MRs pro Cluster ausgerollt. Die Versionsabhängigkeiten sind im Freight-Modell abgebildet, die Gates zwischen den Stages bauen auf der Verifikation der Probe-IPA auf. Prod wird nicht berührt, das Vorgehen ist aber so beschrieben, dass es später für P1 übernommen werden kann.

### Messbare Ergebnisse

1. Freight-Modell evaluiert und umgesetzt: Das Warehouse erzeugt nur Freight aus zusammenpassenden Versionen.
2. Kargo-Projekt auf management-a, aufgebaut aus den Manifesten der Probe-IPA (Git). Stages `e1`, `t1-nucleus`, `t1-pcilight` (und `lab`, falls management-a `argocd-lab` erreicht) mit einer gemeinsamen Promotion (gleiche Schritte, Werte pro Stage). Die Promotion schreibt beide kube-Repos und synchronisiert ArkMQ vor dem PF-Operator.
3. Gates pro Übergang umgesetzt: automatische oder manuelle Freigabe, Wartezeit (`requiredSoakTime`), Rechte pro Stage.
4. Parallele Promotion auf `t1-k8s-nucleus-a` und `t1-k8s-pcilight-a`.
5. Rollback (ältere Freight erneut promoten) beschrieben und getestet, inklusive Grenzen (CRD- und Journal-Downgrade).
6. Renovate ändert die von Kargo verwalteten Overlays nicht mehr (`ignorePaths`), Zuständigkeiten sind dokumentiert.
7. Konzept für die Erweiterung auf P1.

### Variantenentscheide

1. **Freight-Modell:** ein Warehouse, eines pro Komponente oder Artemis-Bundle + Operatoren; Regel als `freightCreationCriteria` oder als feste Vorgabe von Major.Minor.
2. **Promotion-Strategie:** linear oder parallel, automatische oder manuelle Freigabe pro Übergang, gleiche Promotion-Schritte für alle Stages oder pro Stage.

Details und Empfehlung: [VARIANTENENTSCHEIDE.md §3](VARIANTENENTSCHEIDE.md#3-ipa-promotion-strategie-lab--t1), Zielbild: [VERSIONEN.md §8](VERSIONEN.md#8-promotion-mit-kargo).

### Abgrenzung

- Kein Prod. P1 nur als Konzept, Kargo bekommt keine Rechte auf P1.
- E1 und T1 nur nach Absprache mit dem Team. Operator-Updates betreffen alle Broker im Cluster (`WATCH_NAMESPACE: "*"`).
- Die Verifikation der Probe-IPA ist Ausgangslage. Sie wird aus Git für E1 und T1 übernommen, nicht neu entwickelt. Der Umzug von der Test- auf die Prod-Instanz gehört zur Inbetriebnahme (D05).
- Was beim Schnitt Ende März bereits vorhanden ist (siehe Planung), ist Ausgangslage und nicht Teil der IPA.
- Build-Kette bis Artifactory und die Selfservice-Validierung in `activemq/instances` sind nicht Teil der Arbeit.

### Kriterien

**Standardkriterien:** A01–A12 (inklusive A12 Testdurchführung und Dokumentation)

**Pflichtwahlkriterien (unverändert):**

| Slot | Kriterium | Bezug |
|---|---|---|
| A13 | **D02** Standardisierung, Optimierung und Automatisierung | Manuelle MRs pro Cluster werden durch eine Promotion über die Stages ersetzt |
| A14 | **D05** Inbetriebnahme und Übergabe einer ICT-Lösung | Umzug von der Kargo-Test- auf die Prod-Instanz, Inbetriebnahme auf E1/T1 mit den Teams koordiniert, Übergabe an den Betrieb |

**Wahlkriterien:**

| Slot | Kriterium | Art | Bezug |
|---|---|---|---|
| A15 | **A13** Erhebung und Dokumentation der Bedürfnisse und Umfeld | unverändert | Stakeholder auf E1/T1, ArgoCD-Instanzen, Repos |
| A16 | **A16** Durchführung einer Evaluation | unverändert | Freight-Modell und Promotion-Strategie |
| A17 | **D03** Sicherstellung des Betriebs und der Weiterentwicklung | unverändert | Betriebsdoku, nächste Schritte (P1) |
| A18 | **F06** Durchführung der Wartung | angepasst auf Versions-Upgrade | Upgrade über Kargo durchgeführt, protokolliert, Verfügbarkeit danach geprüft |
| A19 | **C08** Planung und Implementierung eines Rollenkonzepts | angepasst | Kargo-Rechte, Freigaben pro Stage, P1 technisch ausgeschlossen |
| A20 | Abbildung von Versionsabhängigkeiten im Freight-Modell | eigenes | Abhängigkeiten identifiziert, im Freight-Modell abgebildet, ungültige Kombinationen verhindert |
| A21 | Gate- und Rollback-Strategie | eigenes (Vorlage: E05 Punkt 4) | Gate-Bedingungen definiert, Rollback getestet, Grenzen beschrieben |
| A22 | Zusammenspiel Renovate und Kargo | eigenes | Zuständigkeiten (bauen vs. ausrollen) definiert und umgesetzt |

### Risiken

- E1 und T1 werden von anderen Teams genutzt. Zeitfenster und Freigaben müssen vorher abgesprochen sein.
- Fehlende Berechtigungen auf die kube-Repos oder ArgoCD-Apps auf management-a.
- Ob management-a `argocd-lab` erreicht, ist offen. Sonst beginnt die Pipeline der IPA bei E1.
- Die Kargo-Prod-Instanz bedient auch andere Apps. Änderungen dort brauchen mehr Sorgfalt als auf der Test-Instanz.
- Darf die IPA nicht auf der Probe-IPA aufbauen, muss die Verifikation neu hergeleitet werden.
- Rollback ist bei Artemis heikel (CRD-Version, Journal).
- Lange Wartezeiten (`requiredSoakTime`) und Verifikationen bremsen die Live-Demo. Für die Demo kurze Werte setzen und begründen.

---

## Grobplanung

| Zeitraum | Was | Gilt als |
|---|---|---|
| bis Mitte Dezember | Rechte auf die Repos, Argo Rollouts auf Lab-D prüfen. **MQP auf Lab-C aufbauen** (Overlays in beiden kube-Repos, Test-Instanzen, `artemis-monitor`, Prometheus). **Vorläufiges Warehouse + Stages `lab-d`/`lab-c` + einfache Promotion** auf der Kargo-Test-Instanz (Image-Argumente in eigene Patch-Datei). Einfrieren der Test-Instanz absprechen | Vorbereitung |
| Dezember / Januar | Aufgabenstellung Probe-IPA einreichen, Probe-IPA durchführen | Probe-IPA |
| Ende Januar | Abschluss Probe-IPA, Feedback der Experten auswerten | Probe-IPA |
| Februar / März | Verifikation stabilisieren, Absprache mit den Teams zu E1/T1, Zugriffe auf management-a, E1 und T1, Aufgabenstellung IPA einreichen. **Projekt auf management-a, E1/T1-Stages und Freight-Modell nicht vorab bauen** (IPA-Inhalt) | Vorarbeit IPA |
| **Ende März** | **Schnitt:** Stand dokumentieren, was in Kargo läuft und was noch über Renovate und MRs läuft | Ausgangslage IPA |
| voraussichtlich April | IPA durchführen | IPA |

## Offene Punkte

- [x] Kargo bei PF: Wir betreiben es selbst, ich baue es seit rund einem Jahr auf. Die bestehende Plattform ist Ausgangslage, nicht IPA-Leistung.
- [ ] **Mit der verantwortlichen Fachkraft klären, ob die IPA auf der Probe-IPA aufbauen darf** (vor dem Einreichen der Probe-IPA, entscheidend für die Aufteilung)
- [ ] Mit der Fachkraft klären, dass die vorläufige Promotion und der Aufbau von MQP auf Lab-C als Vorbereitung gelten
- [x] Auf welcher Control-Plane läuft die Probe-IPA? → Test-Instanz auf Lab-D, Stages Lab-D und Lab-C (beide `argocd-lab`, beide gehören MQP)
- [ ] Laufen Argo-Rollouts-CRDs **und** -Controller auf Lab-D?
- [ ] Wie erreichen die Verifikations-Jobs Lab-C (Leserechte)?
- [ ] Kargo-Test-Instanz während der Probe-IPA einfrieren (Absprache im Team)
- [ ] Welcher Stand gilt für die Stage `lab-d` (heute Staging-Image, CRD von `main`)?
- [ ] Für die IPA: Erreicht management-a `argocd-lab`? Freigabe der Teams für E1 und T1
- [ ] Projektmethode festlegen (linear, z. B. IPERKA, oder agil) und für beide Arbeiten gleich halten
- [ ] Angepasste und eigene Kriterien im Katalogformat (Gütestufen) ausformulieren und von den Experten validieren lassen

## Hinweise

- KI-Nutzung muss deklariert werden (A02 Punkt 3, Doc07 Punkt 6). Unterstützung bei Analyse und Planung gleich laufend im Arbeitsjournal festhalten.
- Kriterienkatalog: `Kriterienkatalog_QV_BiVO2021_DE-20251025 (1).pdf` (V1.3), Aufbau Teil A: `Kriterien-Teil-A-V1.4.pdf`. Pflichtwahlkriterien: 2 aus HKB B–H, unverändert. Wahlkriterien: 8 aus HKB A–H, unverändert, angepasst oder selbst erstellt.
