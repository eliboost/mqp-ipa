# Configuration & Sizing Guide

This document describes how to size and configure an ActiveMQ Artemis instance on the MQP platform.

The platform configures the Artemis instance with defaults matching PF best practices, applied by the
[pf-activemq-operator](https://gitlab.pnet.ch/middleware/activemq/pf-activemq-operator). To cover all use cases,
the configuration can be adjusted via the self-service YAML file. All available options are described in the
[PFActiveMQ API Reference](/docs/default/resource/activemq/reference/api_reference/). This guide explains how to
choose the right values for your use case.

## Starting Point

Unless your traffic profile requires more, start from this configuration and adjust it using the decisions below.
`4Gi` memory is the platform minimum; smaller limits leave too little effective in-memory capacity for paging and
redistribution, which are enabled by default in our setup at PostFinance.

Replace `namespace` and `splunkIndex` with your own values. The following sections tell you how to determine your
exact `storage` and `memory` requirements.

```yaml
instance:
  namespace: appl-esbpf-e1
  replicas: 2
  storage: 4Gi
  resources:
    limits:
      memory: 4Gi
    requests:
      memory: 2Gi
      cpu: 1
  splunkIndex: esbpfitd
```

## Controller-Managed Constraints

You cannot change these values, but every sizing decision depends on them:

| Constraint                | Value                    | Meaning                                            |
|---------------------------|--------------------------|----------------------------------------------------|
| JVM heap                  | 60 % of the memory limit | Heap available to the broker process               |
| In-memory message limit   | 40 % of the JVM heap     | Beyond this, the full-message policy applies       |
| Disk protection threshold | 60 % of the storage      | Broker starts protecting itself at this disk usage |

## Message Prioritization (Paging)

**Important:** ActiveMQ Artemis potentially loses message prioritization once paging starts.

Ask yourself:

> Is JMS prioritization needed in the future or already in place?

- **No** (default, no config needed) – paging is acceptable and expected, disk is used as a backlog buffer.
- **Yes** → set `addressFullMessagePolicy` to `FAIL`, `BLOCK` or `DROP`. Paging must never occur, so backpressure
  happens in memory and producers are notified once memory is exhausted.

| Policy  | Behaviour                                                                                                      | Side effects                                                                                                                                 |
|---------|----------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| `PAGE`  | The broker starts paging messages to disk when memory limits are exceeded.                                     | Maintains throughput at the cost of I/O overhead. Can degrade performance and increase disk usage under pressure. Prioritization may be lost. |
| `FAIL`  | Delivery fails immediately if memory is full. The broker throws an exception and drops the producer connection. | Guarantees protection from overload, but causes connection churn and producer-side errors.                                                   |
| `BLOCK` | Producers are blocked (flow control) until memory becomes available. The connection stays open, sends pause.    | Ensures backpressure on the producer. Can cause upstream delays. Useful in tightly controlled synchronous systems.                            |
| `DROP`  | Messages are silently dropped when memory is full. The producer is not notified.                                | Highly discouraged in transactional systems. Causes data loss and unpredictable downstream state.                                            |

The policy is applied at the **address level**, which means it applies to all queues bound to that address. You can
override the default behavior on a per-queue basis:

```yaml
queues:
  - name: mqp.example.TEST
    config:
      addressFullMessagePolicy: FAIL
```

## Memory & Storage

Two questions define both values:

1. **How much must fit into RAM before paging starts (or before producers are blocked)?** → `memory`
2. **How large can the backlog on disk grow if consumers are down?** → `storage`

Only a part of each value is usable, because of the controller-managed constraints:

```
usable RAM  ≈ memory_limit × 0.24     (60 % heap × 40 % message limit)
usable disk ≈ storage      × 0.6      (disk protection threshold)
```

This gives the following rules:

```
memory_limit ≥ in_memory_target / 0.24
memory_limit ≥ 4Gi                      # platform minimum

storage      ≥ memory_limit × 0.4       # in-memory messages are persisted too
storage      ≥ backlog / 0.6            # only if paging is used
```

Whichever bound is higher wins, then round up to a sensible value.

The second storage rule only applies with the default policy `PAGE`. If you disabled paging (`FAIL`, `BLOCK` or
`DROP`), no backlog is ever written to disk: the memory limit caps the amount of data, and storage only has to
cover the persisted in-memory messages.

### Example

Traffic profile:

| Input                            | Value                 |
|----------------------------------|-----------------------|
| Average message size             | `10 KB`               |
| Peak rate                        | `50 messages/second`  |
| Max. tolerated consumer downtime | `60 minutes` (3600 s) |
| Should be held in RAM            | `400 MB`              |

**Step 1 – memory**

```
memory_limit ≥ 400 MB / 0.24 ≈ 1.7 GB
             → below the platform minimum → 4Gi
```

**Step 2 – storage, with paging (default)**

```
backlog = 10 KB × 50 × 3600 s ≈ 1.8 GB
storage ≥ 1.8 GB / 0.6        = 3.0 GB
storage ≥ 4Gi × 0.4           = 1.6 GB
        → 3.0 GB wins → 4Gi
```

**Result:** `memory: 4Gi`, `storage: 4Gi` - the starting point above.

**Step 2b – storage, with paging disabled**

There is no backlog on disk, so only the in-memory messages have to be persisted:

```
storage ≥ 4Gi × 0.4 = 1.6 GB → 2Gi
```

**Result:** `memory: 4Gi`, `storage: 2Gi`.

## Limiting Queue Size

Broker-level sizing applies to the whole instance. Single queues can be limited additionally, so that one queue
cannot consume the whole broker capacity:

| Property         | Description                                                                                                       | Default |
|------------------|-------------------------------------------------------------------------------------------------------------------|---------|
| `maxSizeBytes`   | Maximum queue size. Once reached, the full-message policy applies. `-1` = no limit.                               | `-1`    |
| `pageLimitBytes` | In-memory buffer before messages are paged to disk. Only relevant when `maxSizeBytes` is set. Max. 60 % of limits. | `-1`    |

```yaml
queues:
  - name: mqp.example.TEST
    config:
      maxSizeBytes: 104857600   # 100 MB
      pageLimitBytes: 10485760  # 10 MB
```

`pageFullMessagePolicy` decides what happens when the page store itself is full:

- `FAIL` – the producer gets an exception. Use it where reliability matters.
- `DROP` – messages are silently lost. Only for non-critical streams (telemetry, logs).

```yaml
queues:
  - name: mqp.example.TEST
    config:
      pageFullMessagePolicy: FAIL
```