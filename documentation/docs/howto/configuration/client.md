# Client

## Choosing a JMS Client Library

Pick a client artifact based on the JMS namespace your code uses:

| JMS Namespace            | Artemis Artifact         | Typical Use Case            |
|--------------------------|--------------------------|-----------------------------|
| `javax.jms` (JMS 2.0)    | `artemis-jms-client`     | Plain Java, Spring Boot 2.x |
| `jakarta.jms` (JMS 3.0+) | `artemis-jakarta-client` | Spring Boot 3.x +           |

Both artifacts run on any current Java release (8 through 25). The choice is
driven by the namespace your application or framework already uses, not by
the Java version.

**Spring Boot:** prefer `spring-boot-starter-artemis` — it pulls the matching client transitively 
(`artemis-jms-client` on Boot 2.x, `artemis-jakarta-client` on Boot 3.x) and auto-configures `ConnectionFactory` and 
`JmsTemplate`.

**Plain Java (no Spring):** depend directly on `artemis-jakarta-client` for new projects, or on `artemis-jms-client` 
if you must stay on `javax.jms`. You manage the `ConnectionFactory` lifecycle yourself.

### Notes

- Align the client version with the running broker. The current broker version is documented on the 
[Platform details](/docs/default/resource/activemq/reference/platform/) page.
- Do not mix Artemis clients with the legacy ActiveMQ 5.x `activemq-client`, they are wire-compatible but use a 
different API.

## Client Connection

The client connection of ActiveMQ Artemis is configured over the tcp broker connection url. Fist all the brokers that are
available within the cluster. Then after the `?` configurations like ssl or general client configs.

See the following example for a recommended base config: 

````yaml

broker.url=(tcp://e1-mqp-e2e-a-ssl-0.appl-mqp-e2e-e1.e1-pfnet-a.pnet.ch:7002,
            tcp://e1-mqp-e2e-a-ssl-1.appl-mqp-e2e-e1.e1-pfnet-a.pnet.ch:7002)
            ?sslEnabled=true
            &trustStoreType=JKS
            &trustStorePath=/etc/pki-all/truststore.jks
            &keyStorePath=/etc/certs/node_keystore_nocrypt.pem
            &keyStoreType=PEM
            &useTopologyForLoadBalancing=false
````

In case, the brokers run within the same namespace, your client runs, the brokers should connect directly to the service:
`<service-name>.<namespace>.svc.cluster.local:<service-port> -> tcp://e1-mqp-middleware-cluster-a-ssl-0-svc.appl-mqp-middleware-e1.svc.cluster.local:61617`

````shell
(e1-k8s-pfnet-a:appl-mqp-middleware-e1) $ kubectl get services | grep ssl
````

### SSL via system properties

As an alternative to setting the keystore/truststore parameters in the broker URL, the SSL settings can be provided 
via JVM system properties. This keeps secrets and certificate paths out of the connection string and is useful when 
you want to use password protected keystores and truststores without exposing the passwords in the URL. 
The following system properties are supported:

| System Property                              | Example Value                                             |
|----------------------------------------------|-----------------------------------------------------------|
| `org.apache.activemq.ssl.keyStore`           | `/var/spool/keybox/identity-mqp-e2e-e1/node_keystore.p12` |
| `org.apache.activemq.ssl.keyStorePassword`   | `*****`                                                   |
| `org.apache.activemq.ssl.keyStoreType`       | `PKCS12`                                                  |
| `org.apache.activemq.ssl.trustStore`         | `/var/spool/keybox/identity-mqp-e2e-e1/truststore.jks`    |
| `org.apache.activemq.ssl.trustStorePassword` | `*****`                                                   |
| `org.apache.activemq.ssl.trustStoreType`     | `JKS`                                                     |

When these system properties are set, the corresponding `keyStore*` / `trustStore*` URL parameters can be omitted
from the broker URL. The client resolves them from the JVM system properties.

!!! warning "Caution when tuning `-XX:MaxDirectMemorySize`"
    The Artemis client uses Netty as its transport and allocates Netty's buffers in native direct memory. This memory shows 
    up neither in the heap nor in the standard buffer metrics (`jvm.buffer.*` and the corresponding OTEL metrics only
    cover JDK direct buffers), but it still counts against the `-XX:MaxDirectMemorySize` limit. 

    Sizing the limit based on those metrics therefore leads to a value that is far too low.
  
    **What you may see when direct memory runs out:**
    
    - Clients can no longer establish connections — connection attempts fail during the Netty handshake and the client ends 
      up in a reconnect loop.
      - Consumers silently drop out and fail to re-attach after a reconnect. Queues keep filling up because producers are 
        still connected while nothing is consuming.
      - Failures can be partial and intermittent: existing connections keep working while new ones fail, depending on when 
        allocations happen.
      - The broker looks healthy — the errors appear only in the client logs, typically as 
        `io.netty.util.internal.OutOfDirectMemoryError` or `java.lang.OutOfMemoryError: Direct buffer memory`.
    
    **Therefore: choose `-XX:MaxDirectMemorySize` conservatively and verify the actual demand under realistic load, not at 
    idle.** When sizing pod memory limits, keep in mind that direct memory is consumed on top of the heap.