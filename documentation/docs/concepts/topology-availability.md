# Topology & Availablility

We use the [activemq-artemis-operator](https://github.com/artemiscloud/activemq-artemis-operator) to
deploy the ActiveMQ Artemis Brokers. In the default configuration we deploy an **active-active**
cluster topology setup with two brokers, based on these decisions -
[[1]](https://github.com/arkmq-org/activemq-artemis-operator/discussions/585#discussioncomment-5815655), 
[[2]](https://github.com/arkmq-org/activemq-artemis-operator/discussions/681).

## HA-Setup
We aim to provide high availability (HA) on the producer-side. Each broker in the cluster is able to
receive the messages from a producer. So if one broker is down for maintenance or in a case of a
failure, we still are able to receive new messages.

## Persistence
The messages are stored in a persistent volume claim (PVC). This is a persistent, redundant storage
provided by the PaaS team. 
Each broker has its own store and is only aware of its own messages. We do not
replicate or mirror any of the messages.
This circumstances results in possible short data unavailability on the consumer-side if one of the
brokers crashes or is restarted for maintenance purpose.
!!! warning
    Please note that there is currently **no** backup and restore strategy for the PVC in place.


## Redistribution
If only one consumer is active for a queue, there is the possibility that both of the brokers have stored
messages for this specific consumer. To prevent these messages from starving (never getting consumed) we configured
the [message-redistribution](https://activemq.apache.org/components/artemis/documentation/latest/clusters.html#message-redistribution)
feature for every queue. This feature re-routes the messages to other nodes in the cluster which do
have a matching consumer. 
