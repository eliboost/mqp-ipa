# Avoid double delivery

There is a potential risk in message processing when using non-XA transactional consumers, especially in failure scenarios
such as abrupt pod termination (e.g., Kubernetes Chaos Monkey). The goal is to help understand how unintended duplicate
processing can occur, and how to reduce the risk through reliable consumption patterns and appropriate ActiveMQ Artemis
configuration.

In a typical flow, a consumer receives a message with `receive()`, processes it (e.g. by calling a downstream system), and
then acknowledges the message via `commit()`.

If the pod terminates during this critical window, after the message is received but before the `commit()`, the broker
rolls back the transaction. Upon restart, the message is redelivered, even though the target system may have already
processed it. This can lead to unintended side effects or duplicate state changes downstream.

This pattern reduces duplicate-processing risk during graceful shutdown and broker restarts. It does not provide
exactly-once side effects. If downstream side effects happen before `commit()` succeeds, the downstream operation must
still be idempotent, deduplicated, or protected by another consistency mechanism such as XA.

### Reliable Consumption Pattern: One Session Per Message

To mitigate this risk, consumers should open a new session for each message, while reusing the same connection. This
allows:

- Graceful shutdown to complete cleanly, as Artemis stops accepting new sessions once shutdown begins.
- Ongoing processing to finish within the configured shutdown timeout window which is 20 seconds, see chapter Broker Setup.

If your application routinely takes longer than 20 seconds between `receive()` and `commit()`, both Artemis and
Kubernetes shutdown timers should be adjusted accordingly.

### Recommended Connection Parameters

The following settings should be added to the consumer connection string to support predictable redelivery and
transactional isolation:

```properties
initialConnectAttempts=-1
reconnectAttempts=-1
retryInterval=60000
consumerWindowSize=0
```

- `initialConnectAttempts=-1`: Retries indefinitely if the broker is unavailable during application startup.
- `reconnectAttempts=-1`: Retries indefinitely after an established broker connection is lost.
- `retryInterval=60000`: Waits one minute between retry attempts.
- `consumerWindowSize=0`: Ensures strict one-at-a-time message delivery. Prevents prefetching or buffering, enforcing
  clean message boundaries per session.

### Summary

By isolating each message into its own session and tuning the connection behavior, consumers can achieve at-least-once
delivery semantics with a significantly reduced risk of duplicate processing during graceful shutdown and broker restarts.

This consumption pattern enables transactional consistency without requiring XA, but it does not eliminate the need for
idempotent or deduplicated downstream side effects.
