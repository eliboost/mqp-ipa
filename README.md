# mqp-ipa

Automatisierung der MQP-Infrastruktur (ActiveMQ Artemis) mit Kargo.

Fachrichtung: **Plattformentwicklung**

## Übersicht

| | Probe-IPA | IPA |
|---|---|---|
| Thema | Freight-Modell und Promotion | Verifikation und Gates |
| Abschluss | Ende Januar | voraussichtlich April (Termin offen) |
| Umgebungen | Lab → E1 | Lab → E1 → T1 (nucleus und pcilight) |
| Prod | ausgeschlossen | ausgeschlossen, nur als Konzept |
| Variantenentscheid | Wie wird die Versionsabhängigkeit in Kargo abgebildet? | Welche Testarten? Welche Promotion-Strategie (linear/parallel)? |

```
Probe-IPA:  lab-d ──► e1-pfnet-a
            (manuelle Promotion, einfacher Health-Check)

IPA:        lab-d ──► e1-pfnet-a ──┬──► t1-nucleus-a
            (auto,     (auto nach  └──► t1-pcilight-a
             Verif.)    Verif.)         (parallel, manuelle Freigabe)

            p1-*: nur Konzept, Kargo hat keine Stage und keine Rechte
```

## Ausgangslage (gilt für beide Arbeiten)

Das Artemis-Ökosystem besteht aus mehreren Komponenten mit Versionsabhängigkeiten:

- **ArkMQ Artemis Operator** (Upstream), deployt über `kube-artemis-operator`
- **pf-activemq-operator** (eigener Operator), deployt über `kube-pf-activemq-operator`. Er erzeugt `ActiveMQArtemis`-Ressourcen und hängt damit vom ArkMQ-Operator ab.
- **Broker-Image** und **artemis-custom-init-image**: Major.Minor müssen übereinstimmen, sonst beendet sich der PF-Operator (`os.Exit(1)`).

Welche Reihenfolge und Kombination beim Ausrollen funktioniert, weiss man heute nur aus Erfahrung. Weitere Punkte:

- Renovate erstellt pro Cluster ein MR, Freigabe und Reihenfolge sind manuell.
- `kube-*`-Repos und `deployments` haben keine CI (kein `kustomize build`, kein Schema-Check).
- Der envtest des PF-Operators testet gegen `releases/latest` von ArkMQ, nicht gegen die deployte Version.
- Der einzige Test nach dem Deployment ist der `artemis-monitor` (Dauer-Canary in `appl-mqp-e2e`). Er blockiert keine Promotion.
- Versionsstand (Oktober 2026): e1/t1 Artemis 2.57 / ArkMQ v2.2.2 / PF 0.17.5, p1 Artemis 2.53 / ArkMQ v2.1.7 / PF 0.17.1.

Umgebungen und ArgoCD-Instanzen:

| Cluster | ArgoCD | Rolle |
|---|---|---|
| `e1-k8s-pfnet-lab-d` | `argocd-lab` (eigene Instanz) | Lab |
| `e1-k8s-pfnet-a` | `argocd-itd` | E1 |
| `t1-k8s-nucleus-a`, `t1-k8s-pcilight-a` | `argocd-itd` | T1 |
| `p1-k8s-nucleus-a`, `p1-k8s-pcilight-a` | `argocd` | Prod (tabu) |

---

## Probe-IPA: Freight-Modell und Promotion

### Ziel

Die MQP-Komponenten (ArkMQ-Operator, PF-Operator, Broker- und Init-Image) werden mit Kargo von **Lab nach E1** promotet und nicht mehr über manuelle MRs ausgerollt. Die Versionsabhängigkeiten sind im Freight-Modell abgebildet.

### Messbare Ergebnisse

1. Ein Kargo-Projekt mit Warehouse(s) und den Stages `lab` und `e1`.
2. Neue Versionen werden vom Warehouse erkannt und als Freight bereitgestellt.
3. Eine Promotion schreibt die Versionen in die richtigen Overlays beider kube-Repos und synchronisiert die ArgoCD-Apps in der richtigen Reihenfolge (ArkMQ vor PF-Operator).
4. Ein einfacher Health-Check als Verification (`PFActiveMQ` steht auf `ActiveMQBrokersReady=True`).
5. Renovate ändert die von Kargo verwalteten Overlays nicht mehr (`ignorePaths`).

### Variantenentscheid: Freight-Modell

| Variante | Beschreibung |
|---|---|
| A | Ein Warehouse, eine Freight bündelt alle Komponenten |
| B | Ein Warehouse pro Komponente, Reihenfolge über die Stages |
| C | Artemis-Bundle (Broker und Init-Image) gebündelt, die Operatoren getrennt |

Mögliche Kriterien: Abbildung der Abhängigkeiten, Verhindern ungültiger Kombinationen, Nachvollziehbarkeit, Aufwand, Erweiterbarkeit auf T1/P1.

### Abgrenzung

- Kein T1, kein Prod.
- Keine Teststrategie, nur ein einfacher Health-Check als Nachweis, dass der Mechanismus funktioniert.
- Kargo-Installation und Zugriffe gehören zur Vorbereitung, nicht zur Probe-IPA.

### Kriterien

**Standardkriterien:** A01–A12 (inklusive A12 Testdurchführung und Dokumentation)

**Pflichtwahlkriterien (unverändert):**

| Slot | Kriterium | Bezug |
|---|---|---|
| A13 | **D02** Standardisierung, Optimierung und Automatisierung | Manuelles Rollout wird durch Kargo-Promotion ersetzt |
| A14 | **D04** Plattform wählen und bereitstellen | Kargo-Projekt, Warehouses und Stages bereitstellen |

**Wahlkriterien:**

| Slot | Kriterium | Art | Bezug |
|---|---|---|---|
| A15 | **A13** Erhebung und Dokumentation der Bedürfnisse und Umfeld | unverändert | Cluster, ArgoCD-Instanzen, Repos, Schnittstellen |
| A16 | **A14** Machbarkeitsstudie (Proof of Concept) | unverändert | Kargo für MQP als PoC |
| A17 | **A16** Durchführung einer Evaluation | unverändert | Variantenentscheid Freight-Modell |
| A18 | **D01** Analyse | unverändert | Ist-Analyse Rollout-Prozess und Lücken |
| A19 | **D03** Sicherstellung des Betriebs und der Weiterentwicklung | unverändert | Betriebsdoku und nächste Schritte für die IPA |
| A20 | **C08** Planung und Implementierung eines Rollenkonzepts | angepasst | Kargo-Rechte auf Repos und Stages, Least Privilege, kein Zugriff auf P1 |
| A21 | Abbildung von Versionsabhängigkeiten | eigenes | Abhängigkeiten identifiziert, im Freight-Modell abgebildet, ungültige Kombinationen verhindert |
| A22 | Zusammenspiel Renovate und Kargo | eigenes | Zuständigkeiten (Artefakt erkennen vs. promoten) definiert und umgesetzt |

### Risiken

- Fehlende Berechtigungen auf die kube-Repos oder ArgoCD-Apps.
- Die bestehende Kargo-Plattform (seit rund einem Jahr im Aufbau) muss klar von der Leistung in der Probe-IPA abgegrenzt sein.
- Der ArkMQ-Operator ist eine GitHub-Release-URL und kein Image. Wie das abonniert wird, muss vor der Probe-IPA geklärt sein.
- Lab-d hat eine eigene ArgoCD-Instanz, E1 läuft über `argocd-itd`. Kargo muss beide erreichen.
- Lab-d ist heute eine Entwickler-Spielwiese (Staging-Images, CRD ungepinnt von `main`).

---

## IPA: Verifikation und Gates von Lab bis T1

### Ziel

Eine neue Version wird nach jeder Promotion automatisch getestet. Sie gelangt nur nach erfolgreichen Tests von Lab nach E1 und von dort parallel auf beide T1-Cluster. Prod wird nicht berührt. Das Vorgehen ist so beschrieben, dass es später für P1 übernommen werden kann.

### Messbare Ergebnisse

1. AnalysisTemplates für mindestens drei Testarten, als Verification an den Stages:
   - **Status:** `PFActiveMQ` und `ActiveMQArtemis` sind Ready.
   - **Funktion:** `artemis-monitor`-Metriken (`monitor_messages_*`): Nachrichten fliessen, Fehlerrate = 0.
   - **Versionskompatibilität:** Die Kombination aus ArkMQ, PF-Operator, Broker und Init-Image ist laut Matrix erlaubt.
   - optional **Upgrade-Pfad:** Upgrade von n-1 auf n (Stand P1 auf neue Version).
2. Gates: Lab → E1 automatisch nach grüner Verification, E1 → T1 mit manueller Freigabe.
3. Parallele Promotion auf `t1-k8s-nucleus-a` und `t1-k8s-pcilight-a`.
4. Rollback-Vorgehen (ältere Freight erneut promoten) beschrieben und getestet, inklusive Grenzen (CRD- und Journal-Downgrade).
5. Konzept für die Erweiterung auf P1.

### Variantenentscheide

1. **Testarten:** Welche Tests laufen als Verification, und welche davon blockieren die Promotion? Kriterien: Aussagekraft, Laufzeit, Stabilität, Aufwand, Auswirkung auf andere Teams.
2. **Promotion-Strategie:** linear oder parallel, für Stages und Komponenten. Dazu: automatische oder manuelle Freigabe pro Übergang.

### Abgrenzung

- Kein Prod. P1 nur als Konzept, Kargo bekommt keine Rechte auf P1.
- E1 und T1 nur nach Absprache mit dem Team. Operator-Updates betreffen alle Broker im Cluster (`WATCH_NAMESPACE: "*"`).
- Was beim Schnitt Ende März bereits vorhanden ist (siehe Planung), ist Ausgangslage und nicht Teil der IPA.

### Kriterien

**Standardkriterien:** A01–A12 (inklusive A12 Testdurchführung und Dokumentation)

**Pflichtwahlkriterien (unverändert):**

| Slot | Kriterium | Bezug |
|---|---|---|
| A13 | **D02** Standardisierung, Optimierung und Automatisierung | Getestete, automatisierte Promotion über die Stages |
| A14 | **F03** Abnahme einer Server-Infrastruktur oder von Server-Diensten | Abnahmekriterien und Testkonzept als Gates zwischen den Stages |

**Wahlkriterien:**

| Slot | Kriterium | Art | Bezug |
|---|---|---|---|
| A15 | **A16** Durchführung einer Evaluation | unverändert | Testarten und Promotion-Strategie |
| A16 | **D05** Inbetriebnahme und Übergabe einer ICT-Lösung | unverändert | Inbetriebnahme auf E1/T1 mit dem Team koordiniert |
| A17 | **D03** Sicherstellung des Betriebs und der Weiterentwicklung | unverändert | Übergabe und nächste Schritte (P1) |
| A18 | **E10** Konzept zur Überwachung | angepasst auf Broker statt Netzwerk | Überwachte Elemente, Schwellwerte und Aktionen entsprechen den Erfolgsbedingungen der AnalysisTemplates |
| A19 | **F06** Durchführung der Wartung | angepasst auf Versions-Upgrade | Upgrade durchgeführt, protokolliert, Verfügbarkeit danach geprüft |
| A20 | **C08** Planung und Implementierung eines Rollenkonzepts | angepasst | Kargo-Rechte, Freigaben pro Stage, P1 technisch ausgeschlossen |
| A21 | Testkonzept für Verifikationen | eigenes | Testarten begründet, Testfälle mit erwarteten Ergebnissen, als AnalysisTemplates automatisiert |
| A22 | Gate- und Rollback-Strategie | eigenes (Vorlage: E05 Punkt 4) | Gate-Bedingungen definiert, Rollback getestet, Grenzen beschrieben |

### Risiken

- E1 und T1 werden von anderen Teams genutzt. Zeitfenster und Freigaben müssen vorher abgesprochen sein.
- Verification-Laufzeiten (Rollout aller Broker plus Messzeit) bremsen die Live-Demo (F&P04). Eine verkürzte Demo-Variante vorbereiten.
- Rollback ist bei Artemis heikel (CRD-Version, Journal).

---

## Grobplanung

| Zeitraum | Was | Gilt als |
|---|---|---|
| bis Mitte Dezember | Kargo-Zugang, Rechte auf die Repos, Anbindung `argocd-lab` und `argocd-itd`, ArkMQ als Freight abonnieren, Absprache Lab-d | Vorbereitung |
| Dezember / Januar | Aufgabenstellung Probe-IPA einreichen, Probe-IPA durchführen | Probe-IPA |
| Ende Januar | Abschluss Probe-IPA, Feedback der Experten auswerten | Probe-IPA |
| Februar / März | Probe-IPA stabilisieren, T1 als Stages anbinden (ohne Verification), Absprache mit dem Team zu E1/T1, Aufgabenstellung IPA einreichen | Vorarbeit IPA |
| **Ende März** | **Schnitt:** Stand dokumentieren, was in Kargo läuft und was noch über Renovate und MRs läuft | Ausgangslage IPA |
| voraussichtlich April | IPA durchführen | IPA |

## Offene Punkte

- [x] Kargo bei PF: Wir betreiben es selbst, ich baue es seit rund einem Jahr auf. Die bestehende Plattform ist Ausgangslage, nicht IPA-Leistung.
- [ ] Sind Argo Rollouts bzw. die AnalysisTemplate-CRDs für die Verification vorhanden?
- [ ] Freigabe des Teams für Promotionen auf E1 und T1 (Zeitfenster, Kommunikation)
- [ ] Wem gehört Lab-d künftig (Kargo-Stage oder Entwickler-Spielwiese)?
- [ ] Projektmethode festlegen (linear, z. B. IPERKA, oder agil) und für beide Arbeiten gleich halten
- [ ] Mit der verantwortlichen Fachkraft klären, ob die IPA auf der Probe-IPA aufbauen darf
- [ ] Angepasste und eigene Kriterien im Katalogformat (Gütestufen) ausformulieren und von den Experten validieren lassen

## Hinweise

- KI-Nutzung muss deklariert werden (A02 Punkt 3, Doc07 Punkt 6). Unterstützung bei Analyse und Planung gleich laufend im Arbeitsjournal festhalten.
- Kriterienkatalog: `Kriterienkatalog_QV_BiVO2021_DE-20251025 (1).pdf` (V1.3), Aufbau Teil A: `Kriterien-Teil-A-V1.4.pdf`.
