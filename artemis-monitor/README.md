## Artemis-Monitor

It is a Springboot application to test the ActiveMQ Artemis Brokers.

## Kubernetes client

The image can be deployed in a K8s Namespace in [Deployments](https://gitlab.pnet.ch/middleware/activemq/deployments), 
for example in Namespace [appl-mqp-middleware](https://gitlab.pnet.ch/middleware/activemq/deployments/-/blob/main/appl-mqp-middleware/e1-k8s-pfnet-a/deployment.yaml?ref_type=heads) to test the brokers in `e1-mqp-middleware-cluster-a`.

In [ArgoCD](https://pfportal.pnet.ch/argocd-itd/applications/e1-k8s-pfnet-a-appl-mqp-middleware-e1).

## Standalone client

To test the brokers from a Lab machine, it is necessary to clone the project and to edit the [application.properties](https://gitlab.pnet.ch/middleware/activemq/artemis-monitor/-/blob/main/src/main/resources/application.properties) file.
For example to test the Artemis brokers in [e1-mqp-middleware-cluster-a](https://gitlab.pnet.ch/activemq/instances/-/blob/main/instances/e1-k8s-pfnet-a/e1-mqp-middleware-cluster-a.yaml)  

````
spring.artemis.broker-url=(tcp://e1-mqp-middleware-cluster-a-ssl-0.appl-mqp-middleware-e1.e1-pfnet-a.pnet.ch:7002,tcp://e1-mqp-middleware-cluster-a-ssl-1.appl-mqp-middleware-e1.e1-pfnet-a.pnet.ch:7002)?sslEnabled=true&verifyHost=false&trustStoreType=JKS&trustStorePath=/var/opt/neviskeybox/default/identity-mqp-middleware-e1/truststore.jks&keyStorePath=/var/opt/neviskeybox/default/identity-mqp-middleware-e1/node_keystore_nocrypt.pem&keyStoreType=PEM&useTopologyForLoadBalancing=false&ha=true

# comment user and pwd to use the certificate in the broker-url
#spring.artemis.user=admin
#spring.artemis.password=admin
````

To start sending/receiving messages
```bash
$ cd artemis-monitor
$ mvn clean install
$ java -jar target/artemis-monitor-0.1.0-SNAPSHOT.jar
```