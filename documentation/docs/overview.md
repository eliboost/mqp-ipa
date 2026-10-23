# Overview

## What is ActiveMQ Artemis

ActiveMQ Artemis is a next generation messaging broker that is designed to replace ActiveMQ 5.

ActiveMQ Artemis is a high-performance messaging system developed by the Apache Software Foundation.
It is designed to replace ActiveMQ 5 and to handle a large volume of messages with low latency and
high throughput.

## Artemis @ PostFinance
Artemis was introduced to establish a possibility to migrate away from JMS over weblogic server (WLS).
The platform team DT Middleware in cooperation with DT ESBPF set out to engineer a decentralized ActiveMQ Artemis
platform. You’re reading the documentation of this ActiveMQ Artemis platform.

The platform is based on the [kubernetes@postfinane](https://kubernetes.pnet.ch/) platform and
integrated over
the [operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). On the top of
the operator we built a self-service, which supports you to order and configure ActiveMQ Artemis instances.

## Further reading

Please check out the other pages in this documentation to learn more about the ActiveMQ Artemis at
PostFinance. It should give you all relevant information and is actively maintained.

If you want to learn more about ActiveMQ Artemis in general. then we recommend the following
external resources:

* Official documentation: https://activemq.apache.org/components/artemis/
* History: https://www.openlogic.com/blog/exploring-activemq-artemis