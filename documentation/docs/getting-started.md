# Getting Started

This guide explains the prerequisites and provides guidelines for a smooth ActiveMQ onboarding experience.

## What you need

- **A valid IDM role**  
  You need a valid IDM role of **your** application. You can either use an existing role or create a new one 
  specifically to manage authorization for your ActiveMQ instances. For example, **ESBPF** uses its **operator** role 
  to manage ActiveMQ instances.

- **A Kubernetes namespace**  
  Since the ActiveMQ instance is deployed in a Kubernetes namespace, you need an existing namespace or a newly created 
  one. You can request this via the [namespaces repository](https://gitlab.pnet.ch/kubernetes/namespaces/).

- **Block storage exclusion annotation on the namespace**  
  Ensure that the following annotation is configured on your namespace:

  ```yaml
  namespace:
    name: appl-myappl-e1
  annotations:
    # Set this value to the required disk storage.
    # This is the maximum storage a broker can get and must be
    # equal to the configured storage of the instance to be deployed.
    postfinance.ch/opa-gatekeeper-exclusion.block-storage: "2Gi"
  ```

!!! warning "Namespace request / update" 
    If you request a new namespace or add the block storage exclusion annotation to an existing namespace, add the 
    following comment to the merge request in the namespaces repository:

    `Block storage for ActiveMQ Artemis <link to Instances MR>`