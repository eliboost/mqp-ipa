# Expiry Queue

The `pf-activemq-operator` automatically creates an expiry address and queue for each configured queue.

The suffix `_EXPIRY` is appended to the name of the original queue. The routing type is `ANYCAST`, just like the 
original queue.

A message is not kept indefinitely when it is no longer valid. If a message exceeds its time-to-live (TTL) before being 
consumed, it expires and is moved to the expiry queue.

Typical reasons for expiry include:
- consumers are temporarily unavailable
- backlog processing takes longer than the configured TTL
- producer-side TTL values are too short for the expected processing time

The expiry queue can be used to isolate expired messages in a controlled way, analyze them, and reprocess or discard 
them as needed, without losing visibility.

## Expiry behavior

A message expires when its TTL is reached. Expired messages are removed from the original queue and routed to the 
corresponding expiry queue.

With a message TTL of `10000` ms, behavior is as follows:

1. A message is sent with TTL `10000` ms.
2. The message is not consumed within `10000` ms.
3. The message expires and is moved to the expiry queue.

### Default expiry values

The following values are set as platform defaults:

| Name             | Value |
|------------------|-------|
| expiry-delay     | -1    |
| min-expiry-delay | -1    |
| max-expiry-delay | -1    |

### Custom expiry values

To set custom expiry settings, define `expiryDelay` per queue in your queue config. The other expiry settings are 
currently not supported for customization.

```yaml
queues:
  - name: mqp.example.TEST
    config:
      expiryDelay: 60000     # 60 seconds
```