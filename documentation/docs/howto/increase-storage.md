# Increase Storage

The storage of MQP instances is based on PersistentVolumeClaims (PVCs) managed by a StatefulSet.
Kubernetes supports resizing PVCs if the underlying `StorageClass` allows volume expansion.

### NOTE

Reducing the storage size of a PVC is not supported by Kubernetes.

To decrease storage, the storage on the MQP Instance needs to be updated and afterward the existing PVCs must be 
deleted. The PVCs will be recreated with a smaller size automatically. This will result in data loss unless the data 
is backed up and restored manually.

### Step 1: Increase Namespace Storage Quota

Increase the storage quota defined via namespace annotations.

The namespace configuration is managed here: https://gitlab.pnet.ch/kubernetes/namespaces

````yaml
  namespace:
    name: appl-mqp-test-e1
    annotations:
      postfinance.ch/opa-gatekeeper-exclusion.block-storage: "2Gi"

````

### Step 2: Update the Instance Storage Configuration

Adjust the `storage` value on the MQP instance.

````yaml
 instance:
   namespace: appl-mqp-test-e1
   replicas: 2
   storage: 2Gi
````

### Step 3: Operator Resizes PVCs and Restarts Pods

!!! warning 
    Updating the storage configuration will cause the StatefulSet to be deleted and recreated. Existing PVCs will remain intact, but Pods will be temporarily unavailable during this process.

Once the instance configuration is updated, the `pf-activemq-operator` automatically reconciles the desired state:

1. The operator patches each PVC to expand it to the configured size.
2. After the PVCs have been successfully resized, the Pods are restarted.

The entire process typically completes within **5 minutes**. During this time, the ActiveMQ instance will be temporarily unavailable.

### Step 4: Verify the Storage Increase

Once all Pods are running again, verify that the PVC was resized correctly by executing `df -h` inside the Pod:

```bash
kubectl exec -it <pod-name> -- df -h
```

The output should reflect the new storage size on the data volume. Look for the mount point corresponding to the ActiveMQ data directory (e.g., `/opt/<instance-name>/data`):

```
Filesystem      Size  Used Avail Use% Mounted on
overlay         222G   62G  161G  28% /
tmpfs            64M     0   64M   0% /dev
/dev/sda4       222G   62G  161G  28% /etc/hosts
shm              64M     0   64M   0% /dev/shm
/dev/sdd        2.9G   23M  2.8G   1% /opt/<instance-name>/data
[...]
```

Confirm that the `Size` column for the data mount reflects the newly configured storage capacity.
