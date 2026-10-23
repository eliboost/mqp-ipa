# Advanved Features

These configurations are for special cases and are in moment for internal use.

## Service Load Balancer

To randomly distribute the **consumers** in a Cluster of Brokers PODs, We can create our own Kubernetes Service that load balances between all available pods with brokers.

Namespace: `appl-mqp-middleware-e1`

```yaml
kind: Service
apiVersion: v1
metadata:
  name: my-mqp-middleware-cluster-jms-lb-svc
spec:
  ports:
    - name: artemis
      protocol: TCP
      port: 61617
  type: ClusterIP
  selector:
    application: e1-mqp-middleware-cluster-a-app
```

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-mqp-middleware-cluster-lb-svc-ing
  namespace: appl-mqp-middleware-e1
  annotations:
    nginx.ingress.kubernetes.io/ssl-passthrough: "true"
spec:
  ingressClassName: nginx
  rules:
    - host: jms.mqp-middleware-cluster.e1-pfnet-a.pnet.ch
      http:
        paths:
          - backend:
              service:
                name: my-mqp-middleware-cluster-jms-lb-svc
                port:
                  name: artemis
            path: /
            pathType: Prefix
  tls:
    - hosts:
        - e1-mqp-middleware-cluster-a-ssl.appl-mqp-middleware-e1.e1-pfnet-a.pnet.ch
```

````bash
(⎈|e1-k8s-pfnet-a:appl-mqp-middleware-e1) $ kubectl get services | grep lb
my-mqp-middleware-cluster-jms-lb-svc        ClusterIP   10.127.19.254   <none>        61617/TCP                              37m
````

````bash
(⎈|e1-k8s-pfnet-a:appl-mqp-middleware-e1) $ kubectl get ingress | grep lb
my-mqp-middleware-cluster-lb-svc-ing            nginx   jms.mqp-middleware-cluster.e1-pfnet-a.pnet.ch       172.16.178.21   80, 443   39m
````

Example `Broker URL` with Service Load Balances from another namespace:

```bash
broker.url=tcp://jms.mqp-middleware-cluster.e1-pfnet-a.pnet.ch:7002?...
```
The **producer** uses the same Broker URL to send messages.

Example client configuration (consumer and producer):

```yaml 
apiVersion: apps/v1
kind: Deployment
metadata:
  name: artemis-monitor
  labels:
    app: artemis-monitor
spec:
  replicas: 3
  strategy:
    type: Recreate
  revisionHistoryLimit: 3
  selector:
    matchLabels:
      app: artemis-monitor
  template:
    metadata:
      annotations:
        prometheus.io/path: "/actuator/prometheus"
        prometheus.io/scheme: "http"
        prometheus.io/scrape: "true"
      labels:
        app: artemis-monitor
    spec:
      containers:
        - image: middleware-docker-local.repo.pnet.ch/pf/middleware/activemq/artemis-monitor:0.1.202412090202
          name: artemis-monitor
          env:
            - name: LOGGING_LEVEL_CH_POSTFINANCE_ACTIVEMQ
              value: Info
            - name: APPLICATION_DESTINATION
              value: mqp.example.TEST 
            - name: SPRING_ARTEMIS_BROKER_URL
              value: (tcp://jms.mqp-middleware-cluster.e1-pfnet-a.pnet.ch:7002)?...

```

### Links:

[ActiveMQ Artemis with Spring Boot on Kubernetes ](https://www.springcloud.io/post/2022-08/activemq-artemis-spring-boot-k8s)

## Client-Side Load balancing

With Apache ActiveMQ Artemis client-side load balancing, subsequent sessions created using a single session factory can be connected to different PODs of the cluster.
This allows sessions to spread smoothly across the nodes of a cluster and not be "clumped" on any particular node.

* `connectionLoadBalancingPolicyClassName=configure what load balancing policy to use`
* `useTopologyForLoadBalancing=false`

It is required to set the useTopologyForLoadBalancing parameter to false on the URL because we specify servers explicitly in the URL.
The possible values for `connectionLoadBalancingPolicyClassName` are defined in the [documentation](https://activemq.apache.org/components/artemis/documentation/latest/clusters.html#client-side-load-balancing).

Example Broker URL with random load balancing:

```bash
broker.url=(tcp://e1-mqp-middleware-cluster-a-ssl-0.appl-mqp-middleware-e1.e1-pfnet-a.pnet.ch:7002,tcp://e1-mqp-middleware-cluster-a-ssl-1.appl-mqp-middleware-e1.e1-pfnet-a.pnet.ch:7002)?useTopologyForLoadBalancing=false&connectionLoadBalancingPolicyClassName=org.apache.activemq.artemis.api.core.client.loadbalance.RandomConnectionLoadBalancingPolicy&...
```

Default ist Round Robin: `org.apache.activemq.artemis.api.core.client.loadbalance.RoundRobinConnectionLoadBalancingPolicy`