# JET (Jahresendtest) - User Guide

## Overview

End-of-year (JET) and other time-sensitive testing often requires producer applications to simulate a future date ("time warping"). Because ActiveMQ Artemis derives the Core API `AMQTimestamp` property from the local system clock, shifting a client's clock forward while the broker stays on real time can break several messaging behaviors: message expiration (TTL), scheduled delivery, paging, and clustering.

This guide covers two mitigation strategies, ranked by preference, followed by a list of side effects to watch for regardless of which approach you take.

| | Option 1: Custom Business Headers | Option 2: Synchronized Infrastructure Warp |
|---|---|---|
| **Broker changes required** | None | Full OS/cluster clock shift |
| **Blast radius** | Application-level only | Entire broker environment |
| **Best for** | Most application-level date logic | Logic tightly coupled to native TTL/scheduling |
| **Risk** | Low | Higher — cluster and TLS risks below |

---

## Option 1: Use Custom Business Headers (Preferred)

Decouple your test date from the broker's messaging fabric entirely. Keep the producer's real system clock intact for the actual Core API transmission, and carry the simulated "warp date" as an explicit property on the message instead of relying on `AMQTimestamp`.

**Why this is preferred:**
- No infrastructure changes — the broker stays on real, stable time.
- TTL expiration, paging, and scheduled delivery all continue to function normally, since they still key off the real clock.
- Easy to reason about and roll back — the warp logic lives entirely in application code.

### Simplified Producer Side Example

```java
import org.apache.activemq.artemis.api.core.client.ClientMessage;
import org.apache.activemq.artemis.api.core.client.ClientSession;

// Create a standard core message — the native timestamp stays authentic
ClientMessage message = session.createMessage(true);

// Carry the simulated "JET" date as application metadata, not as AMQTimestamp
message.putStringProperty("test_jet_date", "2030-12-31T23:59:59");

producer.send(message);
```

### Simplified Consumer Side Example

```java
import org.apache.activemq.artemis.api.core.client.ClientMessage;

public void onMessage(ClientMessage message) {
    // Read the testing context property instead of message.getTimestamp()
    String processingDate = message.getStringProperty("test_jet_date");

    if (processingDate != null) {
        // Drive year-end / date-sensitive logic off this value
    } else {
        // Fall back to normal production behavior
    }
}
```

**Tip:** Standardize the property name and format (e.g., ISO-8601 string, or a `long` epoch millis) across all producers and consumers involved in the test, and treat its absence as "not a JET test" so other test traffic is unaffected.

---

## Option 2: Synchronized Infrastructure Time Warping

Use this only when application logic is tightly bound to Artemis's native time-based features — broker-side expiration, scheduled messages (`_AMQ_SCHED_TIMESTAMP`), or other behavior you cannot intercept at the application layer.

**How it works:** The infrastructure team (DT Middleware <smmwpf@postfinance.ch>) shifts the OS-level clock on the pods hosting the Artemis broker instances as described in the docs here: [JET (Jahresendtest) - Infra-Guide](/docs/default/resource/activemq/infra/guides/jet/)

**Requirements:**
- All Queues on the Broker should be empty prior the JET tests, no messages are laying around!

### Guarding against unexpected message expiration

Even with the broker clock matching the warped producer, protect against messages being dropped unexpectedly during the JET by overriding TTL behavior for the affected addresses. Check your expiry queue settings, see als the docs [Expiry Queue](/docs/default/resource/activemq/concepts/expiry-queue/) for help.

---

## Pitfalls & Architectural Risks

These apply regardless of which option you choose, but are especially relevant for Option 2 since it changes real OS clocks:

- **SSL/TLS handshake failures.** If the client and broker clocks diverge significantly, the JSSE layer will throw `CertificateNotYetValidException` or `CertificateExpiredException` once system time falls outside the certificate's validity window. Either warp both sides consistently or use certificates with a validity window wide enough to cover the test date range.
- **Broken scheduled messages.** A future value in the core `_AMQ_SCHED_TIMESTAMP` property sent to a broker still on real time will sit parked in the queue until real-world time catches up — it won't be delivered early.
- **Metrics and log correlation fragmentation.** Monitoring systems (Prometheus, Grafana, audit logs) will ingest timestamps "from the future." This can break rate-of-consumption metrics, fire false alerting thresholds, and fragment time-series dashboards.
- **Downstream systems and dependencies.** Anything else reading the broker's clock or consuming these messages (databases, schedulers, batch jobs) may be affected the same way — map out dependencies before warping infrastructure time.
- **Cleanup verification.** After the JET test window, explicitly verify time is resynced all warped nodes and that no lingering scheduled messages, TTLs, or cached certificates are still operating on warped assumptions.

---

## Recommendation

Default to **Option 1** unless you have a specific, identified dependency on broker-native time behavior. It's lower-risk, requires no infrastructure coordination, and is trivial to disable once testing concludes. If you're unsure or still have a question, reach [out to us](/docs/default/resource/activemq/about/contact/). Happy JETing!