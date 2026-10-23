---
date: 2026-10-08T10:00:00+01:00
authors:
  - middleware
---

MQP platform will be updated from ActiveMQ Artemis 2.53.0 to 2.57.0 on e1-k8s-pfnet-a on Fri 09.10.2026 morning
after running the new version stable on our lab environment. The ActiveMQ Artemis instances will be restarted
sequentially (rolling restart), ensuring continuous availability without expected downtime.

You can find the details about the update below.

## Components

| Component              | Version old | Version new |
|------------------------|-------------|-------------|
| Artemis Version        | 2.53.0      | 2.57.0      |
| Artemis Cloud Operator | 2.1.7       | 2.2.2       |
| PF ActiveMQ Operator   | 0.14.0      | 0.17.5      |

## Timeline

* 09.10.2026: e1-k8s-pfnet-a
* 15.10.2026: t1 clusters
* 29.10.2026: p1 clusters

## What's new

The highlights of ActiveMQ Artemis version 2.54.0, 2.55.0, 2.56.0 and 2.57.0 are described in the official Apache documentation.
The detailed release notes are linked below.

* [Apache Artemis 2.57.0](https://artemis.apache.org/components/artemis/documentation/latest/versions.html#version-2-57-0)
* [Apache Artemis 2.56.0](https://artemis.apache.org/components/artemis/documentation/latest/versions.html#version-2-56-0)
* [Apache Artemis 2.55.0](https://artemis.apache.org/components/artemis/documentation/latest/versions.html#version-2-55-0)
* [Apache Artemis 2.54.0](https://artemis.apache.org/components/artemis/documentation/latest/versions.html#version-2-54-0)

Additional information about the updated infrastructure components can also be found at the links below.

* [ActiveMQ Cloud Operator 2.2.2](https://github.com/arkmq-org/activemq-artemis-operator/compare/v2.1.7...v2.2.2)
* [PF ActiveMQ Operator](https://gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/-/releases)

### CVE-2025-14813 fixed

The CVE-2025-14813 vulnerability has been fixed in ActiveMQ Artemis 2.57.0, means any open security incidents related
to this CVE in the image `middleware-docker-local.repo.pnet.ch/pf/middleware/artemis-custom-init-image` can be closed 
after the update.