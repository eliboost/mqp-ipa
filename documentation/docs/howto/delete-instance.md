# Delete Instance

Deleting an ActiveMQ instance is a two-step process: remove the instance definition, then clean up the remaining PVCs yourself.

## Step 1: Remove the Instance Definition

Remove the instance's YAML file from the [instances repository](https://gitlab.pnet.ch/activemq/instances) and open a merge request. TD (Tagesdienst) will merge it manually.

Once merged, the broker pods and related resources are deleted automatically. The PVCs however are **not** removed automatically and must be cleaned up manually.

## Step 2: Remove the PVCs

!!! info
    If the PVCs are not removed, you will receive alerts for unused persistent volumes.

After the instance has been removed, you must manually delete the PersistentVolumeClaims (PVCs) to avoid alerts for unused PVCs.

Identify the PVCs in the instance's namespace:

```bash
kubectl get pvc
```

Delete each PVC:

```bash
kubectl delete pvc <pvc-name>
```

