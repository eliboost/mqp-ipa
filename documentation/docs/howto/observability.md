# Observability of ActiveMQ Instances

A brief description on how to observe ActiveMQ Instances and the messages send or read from these.

## Instance Metrics
A dedicated Grafana dashboard provides insights into your ActiveMQ instance, such as instance state, storage usage,
and JVM metrics.

URL: https://pfportal.pnet.ch/grafana/goto/yk-RtE_Hg?orgId=1

## Queue Metrics
Another Dashboard visualizes the Metrics of the Artemis Adresses and Queues. 

URL: https://pfportal.pnet.ch/grafana/goto/R0W-PCXHR?orgId=1

## Message Logs
To provide proper logs for the messages, the logging from activemq was enhanced by a custom logging plugin
[pf-artemis-logging-plugin](https://gitlab.pnet.ch/middleware/activemq/pf-artemis-logging-plugin). The instance logs are
available in Splunk under the index configured for that instance. The following query can be reused by replacing the 
index and namespace with the values matching the corresponding application.

Example MQP:

````
index=midwitd kubernetes_namespace=appl-mqp-e2e-e1 kubernetes_container_name=e1-mqp-* sourcetype=kubernetes_logs "[PFEventLogging]"
````
