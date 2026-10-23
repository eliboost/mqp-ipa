# Dead-Letter Queue

The `pf-activemq-operator` automatically creates a Dead-Letter Queue (DLQ) address and queue for each configured queue.

The suffix `_DLQ` is appended to the name of the original queue. The routing type is `ANYCAST`, just like the original 
queue.

A message is not discarded immediately when it cannot be processed. Instead, the broker retries delivery (redelivery). 
Only after the maximum number of delivery attempts is reached, the message is moved to the DLQ.

Typical reasons for redelivery include:
- temporary consumer failures
- invalid payloads
- unmet business or technical preconditions

The DLQ is used to isolate problematic messages in a controlled way, analyze them, and reprocess them as needed, without 
permanently blocking the regular message flow.

## Redelivery behavior

The delay before the next delivery attempt starts at `redeliveryDelay` and increases by the factor `redeliveryMultiplier` 
(exponential backoff), capped at `maxRedeliveryDelay`. After `maxDeliveryAttempts`, the message is moved to the DLQ. 

With `redeliveryDelay=30000`, `redeliveryMultiplier="2.0"` and `maxRedeliveryDelay=600000`, delays evolve like this:

- after attempt 1: wait `30000` ms
- after attempt 2: wait `60000` ms
- after attempt 3: wait `120000` ms
- after attempt 4: wait `240000` ms
- after attempt 5: wait `480000` ms
- after attempt 6 and later: wait `600000` ms (cap reached)

After `maxDeliveryAttempts=16`, the message is moved to the DLQ.

### Default redelivery values

The following values are set as platform defaults:

| Name                  | Value  |
|-----------------------|--------|
| redelivery-delay (ms) | 30000  |
| redelivery-multiplier | "2.0"  |
| max-redelivery-delay  | 600000 |
| max-delivery-attempts | 16     |

### Custom redelivery values

To set custom redelivery settings, define them per queue in your queue config:

```yaml
queues:
  - name: mqp.example.TEST
    config:
      redeliveryDelay: 10000         # 10 seconds
      redeliveryMultiplier: "1.0"
      maxRedeliveryDelay: 300000     # 5 minutes
      maxDeliveryAttempts: 10
```

## Alarming

Alerting for DLQ events is already in place via configured alert rules see [Alarming](../howto/alarming.md) for more 
details.

When messages are moved to the DLQ, the responsible applications/teams are notified so they can take action.

Typical follow-up actions include:
- analyzing the root cause of repeated delivery failures
- moving messages for reprocessing after the issue is fixed
- handling or discarding messages when reprocessing is not appropriate

These actions can be performed via the [Ops-UI](../howto/operations-ui.md).
