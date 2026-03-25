# Upgrading to EKS Auto Mode

This guide covers migrating an existing (brownfield) EKS cluster from self-managed compute, networking,
and storage to EKS Auto Mode, as well as the simplified version upgrade workflow once Auto Mode is enabled.

## Prerequisites

- AWS provider >= 5.79.0
- Kubernetes >= 1.29
- EKS cluster deployed with this component

## Brownfield Migration (existing cluster to Auto Mode)

### Step 1: Pre-flight -- bump provider

Deploy with `auto_mode_enabled: false` (the default) to verify the provider upgrade causes no
unexpected changes:

```yaml
components:
  terraform:
    eks/cluster:
      vars:
        auto_mode_enabled: false
```

Apply and verify the plan shows only the new `auto_mode_enabled` output and no destructive changes.

### Step 2: Enable Auto Mode with upgrade flag

Set `auto_mode_enabled: true` and `auto_mode_upgrade: true`. **Keep your existing addons and node
groups in place** -- the upgrade flag silently filters Auto Mode-managed addons (`vpc-cni`,
`kube-proxy`, `coredns`, `aws-ebs-csi-driver`) so they are removed in the same apply that enables
Auto Mode.

```yaml
components:
  terraform:
    eks/cluster:
      vars:
        auto_mode_enabled: true
        auto_mode_upgrade: true
        # Keep existing node groups running during transition
        managed_node_groups_enabled: true
        # Keep existing Karpenter IAM role for now
        karpenter_iam_role_enabled: true
```

Apply. The plan will:
- Enable compute, storage, and networking Auto Mode capabilities on the cluster
- Create the Auto Mode node IAM role
- Attach Auto Mode IAM policies to the cluster role
- Remove the 4 Auto Mode-managed EKS add-ons (they become AWS-managed)
- Set `bootstrap_self_managed_addons = false`

> **Warning**: `bootstrap_self_managed_addons` is a ForceNew attribute. If your cluster was
> originally created without this set, changing it may trigger cluster recreation. Check your plan
> carefully. If it shows cluster recreation, you may need to import or ignore this attribute.

### Step 3: Verify Auto Mode nodes

After apply, verify Auto Mode nodes are joining the cluster:

```bash
kubectl get nodes -l eks.amazonaws.com/compute-type=auto
```

Wait for workloads to schedule on Auto Mode nodes. The managed Karpenter in Auto Mode will
provision nodes from the `general-purpose` and `system` node pools.

### Step 4: Clean up configuration

Once workloads are running on Auto Mode nodes, remove the upgrade flag and self-managed resources:

```yaml
components:
  terraform:
    eks/cluster:
      vars:
        auto_mode_enabled: true
        auto_mode_upgrade: false  # or remove entirely (false is the default)
        # Remove addons managed by Auto Mode
        addons:
          aws-efs-csi-driver:
            addon_version: "v2.0.8-eksbuild.1"
        # Disable self-managed compute
        managed_node_groups_enabled: false
        node_groups: {}
        karpenter_iam_role_enabled: false
        fargate_profiles: {}
        addons_depends_on: false
```

### Step 5: Disable downstream components

In your stack configuration, disable or remove:

- **`eks/karpenter`** -- Set `eks_auto_mode_enabled: true` or remove entirely. Auto Mode includes
  managed Karpenter.
- **`eks/alb-controller`** -- Set `eks_auto_mode_enabled: true` or remove entirely. Auto Mode includes
  elastic load balancing.
- **`eks/karpenter-node-pool`** -- Remove if not using custom NodePools. If you need custom pools,
  set `eks_auto_mode_enabled: true` to use the `eks.amazonaws.com/v1` NodeClass API.

### Step 6 (optional): Remove managed node groups

Once all workloads are verified on Auto Mode nodes:

```bash
# Verify no pods on old nodes (except DaemonSets)
kubectl get pods --all-namespaces -o wide | grep -v auto
```

Then remove node groups from your configuration (already done in Step 4 above).

## EBS Storage Migration

Auto Mode uses a different EBS CSI provisioner (`ebs.csi.eks.amazonaws.com`) than self-managed
clusters (`ebs.csi.aws.com`). Existing PersistentVolumeClaims created by the old provisioner
**cannot** mount on Auto Mode nodes.

### Options

1. **New StorageClass** -- Create a new StorageClass referencing `ebs.csi.eks.amazonaws.com` and
   migrate workloads to use it. Existing PVCs remain on managed node group nodes.

2. **Migration tool** -- Use the
   [eks-auto-mode-ebs-migration-tool](https://github.com/awslabs/eks-auto-mode-ebs-migration-tool)
   to re-provision existing PVCs under the new provisioner.

3. **Node affinity** -- During transition, use node selectors or taints to pin EBS-dependent
   workloads to the correct node type (managed node group or Auto Mode).

### StorageClass example for Auto Mode

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3-auto-mode
provisioner: ebs.csi.eks.amazonaws.com
parameters:
  type: gp3
  encrypted: "true"
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

## Kubernetes Version Upgrades (Auto Mode clusters)

Once Auto Mode is enabled, version upgrades are significantly simpler than traditional EKS:

| Step | Traditional EKS | Auto Mode |
|------|----------------|-----------|
| Control plane | Bump `cluster_kubernetes_version`, apply | Same |
| Add-ons | Manually bump each addon version | Automatic |
| Nodes | `force_update_version` or manual rolling | Automatic (drift detection) |
| Orchestration | Multi-step, order matters | Single apply |

### Upgrade workflow

1. **Change `cluster_kubernetes_version`** in your version mixin:
   ```yaml
   # catalog/eks/cluster/mixins/k8s-1-32.yaml
   components:
     terraform:
       eks/cluster:
         vars:
           cluster_kubernetes_version: "1.32"
           # No addon versions needed -- Auto Mode manages them
           addons:
             aws-efs-csi-driver:
               addon_version: "v2.1.0-eksbuild.1"  # only non-Auto-Mode addon
   ```

2. **Apply** -- Control plane upgrades in place (~10-15 minutes).

3. **Wait for node replacement** -- Auto Mode's managed Karpenter detects version drift and
   automatically cordons, drains, and replaces nodes. This happens gradually over minutes to hours
   depending on cluster size and PodDisruptionBudgets.

4. **Verify**:
   ```bash
   # Check all nodes are on the new version
   kubectl get nodes -o custom-columns=NAME:.metadata.name,VERSION:.status.nodeInfo.kubeletVersion

   # Check cluster version
   aws eks describe-cluster --name <cluster-name> --query 'cluster.version'
   ```

### Important notes

- **PodDisruptionBudgets** are respected during node replacement. Ensure workloads have appropriate
  PDBs to maintain availability.
- The **21-day max node lifetime** acts as a backstop -- even without a version upgrade, nodes are
  cycled within 21 days.
- **Non-Auto-Mode add-ons** (like `aws-efs-csi-driver`) still require manual version bumps.
- Kubernetes supports **N-1 kubelet version skew**, so existing nodes continue working during the
  rolling replacement window.

## Disabling Auto Mode

To revert from Auto Mode to self-managed:

1. Set `auto_mode_enabled: false` and apply. This disables Auto Mode capabilities but does not
   remove the cluster.
2. Re-add your managed node groups, Karpenter configuration, ALB controller, and EKS add-ons.
3. Apply again to provision self-managed resources.

> **Note**: You cannot simply remove the Auto Mode blocks -- you must explicitly set
> `auto_mode_enabled: false` first, then apply, before removing the configuration entirely.
