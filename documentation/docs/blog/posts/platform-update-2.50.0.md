---
date: 2026-03-04T10:00:00+01:00
authors:
  - middleware
---

MQP platform will be updated from ActiveMQ Artemis 2.42.0 to 2.50.0 on e1-k8s-pfnet-a on Tue 05.03.2026 morning 
after running the new version stable on our lab environment. The ActiveMQ Artemis instances will be restarted 
sequentially (rolling restart), ensuring continuous availability without expected downtime.

You can find the details about the update below.

## Components

| Component              | Version old | Version new |
|------------------------|-------------|-------------|
| Artemis Version        | 2.42.0      | 2.50.0      |
| Artemis Cloud Operator | 2.1.0       | 2.1.2       |
| PF ActiveMQ Operator   | 0.10.0      | 0.12.10     |

## Timeline

* 05.03.2026: e1-k8s-pfnet-a
* 12.03.2026: t1 clusters
* 26.03.2026: p1 clusters

## What's new

The highlights of ActiveMQ Artemis version 2.50.0 are described in the official Apache documentation.
The detailed release notes are linked below.

* [Apache Artemis 2.50.0](https://artemis.apache.org/components/artemis/documentation/latest/versions.html#2-50-0)

Additional information about the updated infrastructure components can also be found at the links below.

* [ActiveMQ Cloud Operator 2.1.2](https://github.com/arkmq-org/activemq-artemis-operator/releases/tag/v2.1.2)
* [PF ActiveMQ Operator](https://gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/-/releases)
