---
date: 2026-04-20T13:30:00+01:00
authors:
  - middleware
---

MQP platform will be updated from ActiveMQ Artemis 2.50.0 to 2.53.0 on e1-k8s-pfnet-a on Tue 29.04.2026 morning 
after running the new version stable on our lab environment. The ActiveMQ Artemis instances will be restarted 
sequentially (rolling restart), ensuring continuous availability without expected downtime.

You can find the details about the update below.

## Components

| Component              | Version old | Version new |
|------------------------|-------------|-------------|
| Artemis Version        | 2.50.0      | 2.53.0      |
| Artemis Cloud Operator | 2.1.2       | 2.1.7       |
| PF ActiveMQ Operator   | 0.12.10     | 0.14.0      |

## Timeline

* 29.04.2026: e1-k8s-pfnet-a
* 21.05.2026: t1 clusters
* 11.06.2026: p1 clusters

## What's new

The highlights of ActiveMQ Artemis version 2.51.0, 2.52.0 and 2.53.0 are described in the official Apache documentation.
The detailed release notes are linked below.

* [Apache Artemis 2.53.0](https://artemis.apache.org/components/artemis/documentation/latest/versions.html#version-2-53-0)
* [Apache Artemis 2.52.0](https://artemis.apache.org/components/artemis/documentation/latest/versions.html#version-2-52-0)
* [Apache Artemis 2.51.0](https://artemis.apache.org/components/artemis/documentation/latest/versions.html#version-2-51-0)

Additional information about the updated infrastructure components can also be found at the links below.

* [ActiveMQ Cloud Operator 2.1.7](https://github.com/arkmq-org/activemq-artemis-operator/compare/v2.1.2...v2.1.7)
* [PF ActiveMQ Operator](https://gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/-/releases)

### New Config possibility
- Memory, paging limits (maxSizeBytes, pageLimitBytes) and page full behaviour (pageFullMessagePolicy) on queue level can now be set in the queue config section. Until now, it was only possible to set this using properties. For more details check the docs [Sizing Guide](https://activemq.pnet.ch/docs/howto/configuration/sizing-guide/)
- You can override the TTL for all messages on a specific queue using config.expiryDelay. This value takes precedence over the TTL set by the producer. For more details check the docs [Expiry Queue](https://activemq.pnet.ch/docs/concepts/expiry-queue/)

**Example for the new config options**

```yaml
queues:
  - name: mqp.dcp.DCA_DCP_IN_Queue
    permission:
      send:
        - a-pf-identity-mqp-dcp-e1
      consume:
        - a-pf-identity-mqp-dcp-e1
      browse: []
    config:
      maxSizeBytes: 104857600           # (optional) max mermory usage of queue
      pageLimitBytes: 209715200         # (optional) max storage usage of paging
      pageFullMessagePolicy: FAIL       # (optional) behavior on max storage of paging used FAIL | DROP | BLOCK
      expiryDelay: 2                    # (optional) message which are older than this value will be sent to expiryQueue
```