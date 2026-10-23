# pf-activemq-operator

The pf-activemq-operator helps to simplify the definition of an Artemis broker and queues.The operator watches 
PFActiveMQ resources and publishes based on the watched resource an ActiveMQArtemis resource in the namespace
defined, where the artemis-cloud-operator watches these and deploys the broker and queues.

## API

### Watched resource

```yaml
apiVersion: pf-activemq-operator.pnet.ch/v1alpha1
kind: PFActiveMQ
metadata:
  labels:
    app.kubernetes.io/name: pf-activemq-operator
    app.kubernetes.io/managed-by: kustomize
  name: mqp-egw-a
spec:
  teamMail: appl_xyz@postfinance.ch
  roles:
    owner: bancs/appl_xyz
    operators: bancs.mqp
  instance:
    namespace: appl-target-e1
    splunkIndex: midwitd
    logLevel: INFO
    replicas: 2
    storage: 2Gi
    resources:
      limits:
        memory: 2Gi
      requests:
        memory: 1Gi
        cpu: 1
    queues:
      - name: "mqp.egw.In_Queue"
        permission:
          send:
            - a-pf-auth-bancs@e1
          consume:
            - a-pf-auth-egw
            -  bancs/appl_xyz
          browse: []
    properties: # only for experiments
      - propertyA=abc
      - broker-0.propertyB=cde
```

## Development

### Project
The project is initialized with create-go-app and the operator code is generated with 
[Kubebuilder](https://book.kubebuilder.io/).
- The main function that defines the reconciliation, is located in the `Reconcile` function in the `pfactivemq-controller.go`
- To change CRD, do changes in `pfactivemq_types.go` and afterwards run `make manifests` to generate crd defintions

For more information like a guide to create a new API refer to quickstart in the kubebuilder 
documentation https://book.kubebuilder.io/quick-start#quick-start

### Working with go workspaces and modules
The project is structured as a go workspace with multiple modules. The main module is located in the root directory and the API submodules are located in `api/*`. This allows us to have a clear separation between the operator code and the API definitions, and also to version the API independently from the operator code.

In order to set up the workspace locally, you can run the following command in the root directory of the project:

```bash
go work use
```

### API submodule release tags

The Go module proxy requires submodule tags in the format `api/<version>/vX.Y.Z` (e.g. `api/v1alpha1/v0.12.10`)
for consumers to resolve `gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1` via `go get`.

The `tag-api-submodule` CI job in `post-release` handles this automatically on every release tag:
- It discovers all submodules by scanning for `api/*/go.mod`
- For each submodule it checks whether the directory has changed since its last tag
- A new submodule tag is only created when there are actual changes

Adding a new API version (e.g. `api/v2/`) with its own `go.mod` is picked up automatically without any pipeline changes.

### Tests
In addition to unit tests, there are integration tests that run using `envtest`. This spins up a Kubernetes cluster and 
creates the necessary resources. The setup is initialized in `suite_test.go`.

In the `CreateOtherControllerResources` function, a kubeconfig file is generated under `/tmp`. This can be used to debug
the reconciliation process. To do so, follow these steps:

- Set a breakpoint in the controller code 
- Run the integration test locally e.g. wsl
- Inspect the state of the resources, for example:

```bash
KUBECONFIG=/tmp/kubeconfig1234 kubectl describe pfactivemq 1-pfactivemq-sample-schmidomin
 ```

### Run Local
The best way is to use a local k3s cluster on wsl or your vm. To install the binary follow the instructions under:

- Install thin k3s kubernetes cluster https://k3s.io/:
````bash
curl -sfL https://get.k3s.io | sh -

# Check for Ready node, takes ~30 seconds
sudo k3s kubectl get node
````
- Install artemis operator.
`sudo k3s kubectl apply -f https://github.com/arkmq-org/activemq-artemis-operator/releases/latest/download/activemq-artemis-operator.yaml`

Or follw this guide for other methods: https://github.com/arkmq-org/activemq-artemis-operator/blob/main/docs/getting-started/quick-start.md

Create an alias for k3s
`echo "alias k3s='sudo k3s'" >> cat ~/.bash_aliases`

- Install necessary kubernetes resources:
````bash
# Cluster meta info
k3s kubectl apply -f testfiles/cluster-info-local.yaml

# Cert-manager 
k3s kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.20.0/cert-manager.yaml

# CA certificate
k3s kubectl apply -f local/ca-issuer.yaml

# PFActiveMQ custom resource definition
k3s kubectl apply -f onfig/crd/bases/pf-activemq-operator.pnet.ch_pfactivemqs.yaml¨
k3s kubectl apply -f config/samples/v1alpha1_pfactivmq.yaml
````

The best way to debug is to run the operator via ssh on a linux host, like wsl2 or an e1-dev-vm. As default the 
operator uses your local `.kubeconfig` and watches the resources in the active namespace.
