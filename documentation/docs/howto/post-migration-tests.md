# Post-Migration Tests – Recommended Test Checklist

After migrating from WLS JMS to ActiveMQ Artemis you should cover four main blocks of tests: functional JMS tests, non-functional/performance tests, Kubernetes/infrastructure tests, and regression/cutover tests.

For all tests, the following shared tools and views should be used:

- **Application / test clients**: Business applications, JMS test client, load tools (k6, [Load Performance Testing Team](https://confluence.pnet.ch/spaces/NT/pages/77872739/PF85-42+Performance+Test)).
- **Kubernetes commands or automation**: Deleting pods, simulating node failures, chaos monkey and rolling updates.
- **MQP Operations UI**: [MQP Operations UI](/docs/default/resource/activemq/howto/use-operations-ui/) for queue and Dead Letter Queue (DLQ) monitoring (state, consumers, messages), controlling (production enable/disable, consumption pause/resume).
- **Logs**: [Splunk](/docs/default/resource/activemq/howto/observe-activemq/) and application log for producer, broker and consumer logs.
- **Metrics**: Metrics/monitoring ([Grafana dashboards](/docs/default/resource/activemq/howto/observe-activemq/)) for broker and queue metrics.

## 1. Functional JMS system tests

Goal: Prove that messaging behaves the same as on WLS from a business perspective.

For each relevant queue/topic, test:

- **Send/receive**: Text/binary, properties, headers like header_correlationId, header_messageId and verify whether any WLS‑proprietary _"\_WLS\_"_ headers are being used.
- **Transactions**: Client and producer test with commit, rollback, redelivery.
- **Redelivery semantics**: Max redeliveries, backoff/delay, dead letter queue, message expiration (TTL). If it is unclear whether the configuration on WLS deviates from the default, please ask ESBPF for the exact configuration values.
- **Selectors and filters**: Same selector expressions as on WLS, including more complex `WHERE` clauses.
- **Edge cases**: Large messages, message buffering, charset/encoding edge cases, compression, DLQ handling for poison messages.


### 1.1 Example tests and Pass / No‑Pass criteria

#### 1.1.1 Send/receive including headers and properties

**Example:**

1. Prepare some test messages for a given queue with:
    - Text and binary payload variants.
    - Explicit `header_messageId`.
2. Send messages using a JMS client (e.g. business application or technical like springboot-test).
3. Consume the messages with the normal consumer application.
4. Use [MQP Operations UI](/docs/default/resource/activemq/howto/use-operations-ui/) to verify that the queue is empty after processing and no messages appear in DLQ.
5. Optional: use [Splunk](/docs/default/resource/activemq/howto/observe-activemq/) to search for the test `header_messageId` and confirm producer/broker/consumer logs are all present.

**Pass:**

- All 10 messages are received exactly once, no duplicates or lost messages.
- All required headers and properties are present and have the expected values.
- [MQP Operations UI](/docs/default/resource/activemq/howto/use-operations-ui/) shows the queue drained and DLQ unchanged.
- Logs show successful processing without errors.

**No‑Pass:**

- Messages are lost, processed multiple times, or stuck in the queue.
- Missing or incorrect headers/properties that impact business logic.
- DLQ receives messages when there should be none.
- Logs show mapping issues or unexpected exceptions.

---

#### 1.1.2 Transactions: commit, rollback, redelivery

**Example:**

1. Configure a test consumer to process messages within a local-transaction.
2. Send a batch of messages (e.g. 5) for a given business operation.
3. For one run, let the consumer process and commit successfully.
4. For another run, force a rollback (e.g. by throwing an exception on processing).
5. Use [MQP Operations UI](/docs/default/resource/activemq/howto/use-operations-ui/) to observe message visibility and redelivery attempts.
6. Use [Splunk](/docs/default/resource/activemq/howto/observe-activemq/) or application log to:
   - Verify transaction boundaries (successful commit vs rollback).
   - Check that redelivery attempts are visible in logs.

**Pass:**

- Commit: messages are not redelivered and business changes are persisted.
- Rollback: messages are redelivered according to the configured redelivery policy and no business changes are persisted.
- No „in-between“ or partially applied changes are visible in downstream systems.
- [MQP Operations UI](/docs/default/resource/activemq/howto/use-operations-ui/) and Log ([Splunk](/docs/default/resource/activemq/howto/observe-activemq/) or application log) show consistent transaction and redelivery behavior.

**No‑Pass:**

- Messages disappear without commit and without DLQ.
- DB changes persist despite rollback.
- Redelivery occurs even after successful commit, or does not occur after rollback.

---

#### 1.1.3 Selectors and filters

**Example:**

1. Create a set of messages with different by the application used header property values (`header_sender`, `header_messageType`) and unused values.
2. Deploy consumers with selectors identical to those used on WLS (e.g. `header_sender = 'BANCS' AND header_messageType = 'AccountOpening'`).
3. Send messages across all variants.
4. Use [MQP Operations UI](/docs/default/resource/activemq/howto/use-operations-ui/) to verify which messages remain and which are consumed.
5. Use [Splunk](/docs/default/resource/activemq/howto/observe-activemq/) or application log to check that only messages that match the selector are processed by each consumer.

**Pass:**

- Each consumer receives only messages that match its selector.
- No matching messages remain unprocessed.
- Behavior matches previous WLS selector behavior (if reference exists).

**No‑Pass:**

- Consumers process messages that do not match their selector.
- Messages that should match remain indefinitely in the queue.
- Differences compared to WLS selector semantics that impact business logic.

---

## 2. Non-functional / performance tests

Goal: Ensure the new k8s/Artemis setup can handle current and future load.

Important scenarios:

- **Throughput test**: Typical and peak load (msgs/s) per queue/use case, measure end-to-end latency.
- **Load profiles**: Model real-world daily curves and peaks (e.g. batch jobs, end-of-day, month-end processing).
- **Concurrency**: Setup producer/consumer counts like in production and find bottlenecks.
- **Soak tests**: Run realistic load to detect memory leaks, connection issues, k8s resource limits.
- **Resource usage**: Monitor CPU/RAM of broker pods with [Grafana](/docs/default/resource/activemq/howto/observe-activemq/).

## 3. Kubernetes and infrastructure tests

Goal: Prove the operational model (pods, services, storage) is robust.

Test in particular:

- **Broker failover**: Kill pods, simulate node failure, rolling deployments; observe client behavior (reconnect, message loss or not).
- **Storage robustness**: Restart pods during ongoing load and verify no committed messages are lost.
- **Scaling**: Horizontal scaling of consumers/application pods, behavior with Artemis clustering.
- **Network behavior**: Ingress/service config, timeouts, connection limits, behavior during short network glitches.
- **Observability**: Logs, metrics, alerts for queue depths, DLQ growth, connection errors, storage pressure, etc. Verify that all relevant data points are being recorded and exposed.

### 3.1 Example tests and Pass / No‑Pass criteria

#### 3.1.1 Broker failover

**Example:**

1. Start continuous producers and consumers for a representative set of queues.
2. Manually delete one broker pod or trigger a rolling update ([k8s how-to](https://infradocs.pnet.ch/kubernetes/userguide/howto/example/)).
3. Observe client behavior:
   - Automatic reconnect.
   - Handling of in-flight messages (e.g. rollbacking, discarding).
4. Use [MQP Operations UI](/docs/default/resource/activemq/howto/use-operations-ui/):
   - Check that queues do not lose committed messages.
5. Use [Grafana](/docs/default/resource/activemq/howto/observe-activemq/):
   - Verify that resource and connection metrics adapt as pods restart.
6. Use [Splunk](/docs/default/resource/activemq/howto/observe-activemq/):
   - Check for errors during reconnect and confirm that clients recover.

**Pass:**

- Producers and consumers reconnect automatically without manual intervention.
- No committed messages are lost.
- Only short, acceptable interruptions in throughput occur.

**No‑Pass:**

- Clients stay disconnected or require manual restart.
- Messages disappear or are not recoverable after failover.
- Long outages or inconsistency between queues and downstream state.

---

## 4. Regression and cutover tests

Finally, run end-to-end business flows that are currently in production on WLS JMS and execute them 1:1 on Artemis.

- For each **critical business process**, define an end-to-end test (input, expected DB/system state, runtime).
- **Negative scenarios**: Downstream system failures, rollbacks, DLQ routing, manual recovery.