# Alarming

## What you’ll learn

* Which generic alert rules for ActiveMQ Artemis are already in place and how you can configure custom rules for your needs.

## What you need

* A GitLab account
* Git is correctly configured

## Generic Alert Rules
MQP instances are kubernetes based; therefore the general Kubernetes alert rules, such as pod memory alert for example, are
in place next to the general Artemis alert rules.

The following general Artemis rules are configured:

| Category            | Description                                                                                                                                                                                       | Severtiy | Alert name                      |
|---------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------------------------------|
| High Disk Usage     | Disk usage has reached 80% of the usable disk space. **At 100% utilization, clients may be blocked or fail to produce messages**. Note: Only 60% of the total disk capacity is considered usable. | CRITICAL | Artemis_High_DiskUsage          |
| High Address Memory | Memory usage across all addresses has reached 80% of available memory. Paging will start once memory is exhausted.                                                                                | MAJOR    | Artemis_High_AddressMemoryUsage |
| Stale Consumer      | Processing of messages in queue is stopped caused by a stalled consumer.                                                                                                                          | MAJOR    | Artemis_Stale_Consumer          |
| Dead Letter Queue   | There are messages in the dead letter queue which need to be checked.                                                                                                                             | MAJOR    | Artemis_DLQ_Messages            |

The implementation of the generic alerts can be found here: [Observability Alertrules](https://gitlab.pnet.ch/observability/alertmanager/alertrules/-/blob/main/rules/general-activemq-artemis.yml?ref_type=heads)

## Discover and explore the metrics

A nice and handy tool is the [Grafana metrics explorer](https://pfportal.pnet.ch/grafana/explore?schemaVersion=1&panes=%7B%227o6%22:%7B%22datasource%22:%22PF3D020BCD42CE705%22,%22queries%22:%5B%7B%22refId%22:%22A%22,%22expr%22:%22artemis_consumer_count%7Bkubernetes_cluster_name%3D%5C%22e1-k8s-pfnet-a%5C%22,%20env%3D%5C%22E1%5C%22,%20kubernetes_namespace%3D%5C%22appl-mqp-e2e-e1%5C%22,%20queue%3D%5C%22mqp.example.TEST%5C%22%7D%20%22,%22range%22:true,%22instant%22:true,%22datasource%22:%7B%22type%22:%22prometheus%22,%22uid%22:%22PF3D020BCD42CE705%22%7D,%22editorMode%22:%22code%22,%22legendFormat%22:%22__auto%22%7D%5D,%22range%22:%7B%22from%22:%22now-3h%22,%22to%22:%22now%22%7D%7D%7D&orgId=1), where you can test your query expressions.

The relevant metrics for ActiveMQ Artemis start with **artemis_**

## Configure Custom Alert Rules

First of all, please make sure you read the instruction for
the [Alertrules Self-Service](https://gitlab.pnet.ch/observability/alertmanager/alertrules/-/blob/main/README.md)

Cool generic rules at all, but we want a critical alert if things are going south on PROD! Sure, here is an example for the stale consumer alert:

```yaml
      - alert: Artemis_Stale_Consumer
        expr: |
         (artemis_delivering_message_count{kubernetes_cluster_name="p1-k8s-nucleus-a", env="P1", kubernetes_namespace="p1-myapp-p1"} > 0 and increase(artemis_messages_acknowledged{kubernetes_cluster_name="p1-k8s-nucleus-a", env="P1", kubernetes_namespace="p1-myapp-p1"}[10m]) == 0) > 0
        for: 30m
        labels:
          severity: critical
          notify: msgfwd
        annotations:
          summary: 'Processing of messages in queue {{ $labels.queue }} on pod {{ $labels.statefulset_kubernetes_io_pod_name }} is stopped'
          description: 'There are messages in your queue that are not processed because the consumer is stale.'
          msgfwd_mail_to: 'myteam@postfinance.ch'
          msgfwd_mon_enabled: "true"
          msgfwd_mon_send_resolved: "false"
          msgfwd_mon_object_class: APPLICATION
          msgfwd_mon_object: MYAPP
          msgfwd_mon_parameter: 'Artemis_Stale_Consumer_{{ $labels.queue }}'
          msgfwd_mon_parameter_value: 30m
          msgfwd_mon_it_service: MYAPP
          msgfwd_mon_severity: CRITICAL
          msgfwd_mon_alarm_id: 'MYAPP-Prometheus-Artemis_Stale_Consumer'
```

### Additional ideas and examples

```yaml

- alert: ArtemisQueueGrowthTrend
  expr: predict_linear(artemis_message_count[8h], 3600 * 2) > artemis_message_count
  for: 30m
  labels:
    severity: warning
  annotations:
    summary: "Queue growth trend detected"
    description: "Queue {{ $labels.queue }} is predicted to grow continuously over the next 2 hours."
```

```yaml

- alert: ArtemisAuthFailures
  expr: increase(artemis_authentication_count{result="failure"}[5m]) > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Authentication failures detected"
    description: "More than 5 failed authentication attempts on broker {{ $labels.broker }} in the last 5 minutes."
```

```yaml

- alert: ArtemisConnectionCountHigh
  expr: max(artemis_connection_count) by (kubernetes_namespace, ActiveMQArtemis) > 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High number of broker connections"
    description: "Broker {{ $labels.broker }} has more than 100 active connections."
```
