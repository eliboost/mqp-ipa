# JET (Jahresendtest) - Infra-Guide

If it is necessary to set the timestamps of a broker for the JET (Jahresendtest), the following steps are required.

## Prerequisites

- Please check the [K8s JET documentation](https://infradocs.pnet.ch/kubernetes/userguide/howto/example/jet/#background) if you are not familiar with the JET topic.
- The k8s namespace must have the JET flag enabled ([namespace label jet](https://infradocs.pnet.ch/kubernetes/userguide/howto/example/jet/#label-your-namespace)).

## Configure the ActiveMQArtemis resource

Make the following changes on the `ActiveMQArtemis` resource in the corresponding appl namespace, e.g. `e1-mqp-e2e`.

Find the resource name:

```
kubectl get aa
NAME                  READY   AGE
e1-mqp-e2e-a          True    704d
```

Edit the resource:

```
k edit aa e1-mqp-e2e-a
```

### Set the operator-managed label to `false`

```yaml
metadata:
  creationTimestamp: "2024-09-27T12:49:06Z"
  generation: 1762
  labels:
    pf-activemq-operator.pnet.ch/operator-managed: "false"
```

### Set the necessary JET annotations for the pods

```yaml
  deploymentPlan:
    affinity:
      podAntiAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              ActiveMQArtemis: e1-mqp-e2e-a
          topologyKey: kubernetes.io/hostname
    annotations:
      jet.k8s.postfinance.ch/faketime: +6h
      jet.k8s.postfinance.ch/libc: glibc-2-34
```
This step applies the pod annotations described here [K8s JET docs - apply annotations](https://infradocs.pnet.ch/kubernetes/userguide/howto/example/jet/#apply-annotations).

### Fix the `libfaketime` bug with Java based apps

!!! important
    Set the following env var to fix a bug in `libfaketime` with Java based applications ([link placeholder](#)). If not set, the broker won't work properly!

```yaml
deploymentPlan:
  ...
  env:
  - name: AMQ_EXTRA_ARGS
    value: --relax-jolokia
  - name: JAVA_ARGS_APPEND
    value: -Dwebconfig.bindings.artemis.sslAutoReload=true -XX:MaxRAMPercentage=60
  - name: TZ
    value: Europe/Zurich
  - name: LANG
    value: en_US.UTF-8
  - name: FAKETIME_FORCE_MONOTONIC_FIX
    value: "0"
```
For more background information see [K8s JET docs - Java Process Hangs or Hogs CPU](https://infradocs.pnet.ch/kubernetes/userguide/howto/example/jet/#java-process-hangs-or-hogs-cpu).

## Verify the timestamp

After saving, the broker pods will be restarted.

Check the logs to confirm that the settings are working and the timestamp matches your expectations (in our example, we added `+6h`):

```
kubectl logs e1-mqp-e2e-a-ss-0 | tail -n 5
```

```
Defaulted container "e1-mqp-e2e-a-container" out of: e1-mqp-e2e-a-container, e1-mqp-e2e-a-container-init (init), faketime (init)
2026-09-02 16:54:28,236INFO  [PFEventLogging] Message acknowledged commit: m_deliveryCount=1, m_isPaged=false, m_messageId=8694840418, m_correlationID=unavailable, m_address=mqp.example.TEST, m_size=509, m_isLargeMessage=false, m_isDurable=true, m_isExpired=false, m_priority=4, m_timestamp=1788339268230, m_scheduledDeliveryTime=0, m_expiration=0, m_timeSinceCreated=21600006, mh__type=ch.postfinance.activemq.monitor.MyMessage
```

## Things to consider

- Keep in mind the [broker TLS certificate expiry](https://developer.pnet.ch/docs/default/resource/activemq/infra/guides/platform/#certificate) if there is a bigger time warp (e.g. 3 months) needed.
- The operator will not make any changes as long as the `operator-managed` flag is set to false. This includes also any changes made in the self-service!
- If clients (consumer/producer) have a different time setting than the broker you may experience SSL/TLS errors like 
```
2026-09-02T11:44:54.135+02:00 ERROR 1 --- [artemis-monitor] [main] o.apache.activemq.artemis.core.client    : AMQ214016: Failed to create netty connection io.netty.handler.ssl.SslHandshakeTimeoutException: handshake timed out after 10000ms`
```
- If messages on a queue need to be cleared regularly because not all apps participate in the JET, it is possible to set the expiry on an address via self-service with `addressSettings.<match>.maxExpiryDelay=0`. See the docs: [Dropping expired messages](https://artemis.apache.org/components/artemis/documentation/latest/message-expiry.html#dropping-expired-messages).

  Example:

  ```yaml
  instance:
    namespace: appl-esbpf-e1
    replicas: 2
    storage: 2Gi
    resources:
      limits:
        memory: 2Gi
      requests:
        memory: 1Gi 
        cpu: 1
    splunkIndex: eaipfitd
  ```

## End of JET

- Set the `operator-managed` label to true and the k8s namespace label `jetParticipant` to false after the JET. Restart the broker instances and verify the timestamps in the logs.