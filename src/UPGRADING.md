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

## EKS Capabilities (Argo CD, ACK, KRO)

EKS Capabilities are independently-enableable managed platform features that work on any cluster
(Auto Mode or standard). They require AWS provider `>= 6.25.0`.

### Enabling Capabilities

Add capabilities to your stack configuration:

```yaml
components:
  terraform:
    eks/cluster:
      vars:
        capabilities:
          argocd:
            type: ARGOCD
            configuration:
              argo_cd:
                namespace: argocd
                aws_idc:
                  idc_instance_arn: "arn:aws:sso:::instance/ssoins-abc123"
                rbac_role_mapping:
                  - role: ADMIN
                    identity:
                      - id: "user-id-here"
                        type: SSO_USER
          ack:
            type: ACK
            iam_policy_arns:
              - "arn:aws:iam::aws:policy/AmazonRDSFullAccess"
              - "arn:aws:iam::aws:policy/AmazonS3FullAccess"
          kro:
            type: KRO
```

### IAM Roles

Each capability requires its own IAM role with a trust policy for `capabilities.eks.amazonaws.com`.

- **Auto-created**: When `role_arn` is not provided, a role is created automatically
- **Custom**: Provide `role_arn` to use an existing role
- **ACK policies**: Use `iam_policy_arns` to attach service-specific IAM policies to the
  auto-created role (e.g., `AmazonRDSFullAccess` for managing RDS instances)

### Provider Version Upgrade

This component requires AWS provider `>= 6.25.0`. If upgrading from 5.x:

1. Review the [AWS Provider 6.0 Upgrade Guide](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/guides/version-6-upgrade)
2. Update your provider lock file: `terraform init -upgrade`
3. Run `terraform plan` to check for breaking changes before applying

### Verification

```bash
# List capabilities on a cluster
aws eks list-capabilities --cluster-name <cluster-name>

# Describe a specific capability
aws eks describe-capability --cluster-name <cluster-name> --capability-name argocd
```

## Migrating from Self-Managed Tools to EKS Capabilities

If you already have Argo CD, ACK, or KRO installed via Helm charts (e.g., via CloudPosse
components), follow these migration guides. Each tool has a different migration model.

### Migrating Argo CD (cutover migration)

Argo CD migration is a **cutover** -- you cannot run self-managed and EKS-managed Argo CD
simultaneously. Plan for ~70 minutes of migration time.

**Before migrating**, review unsupported features in EKS-managed Argo CD:
- Notifications controller is not available
- Config Management Plugins (CMPs) are not supported
- Custom SSO providers are not supported (AWS Identity Center only)
- Custom Lua health assessment scripts are not supported
- Direct access to configuration ConfigMaps is not available
- Fixed 120-second sync timeout cannot be changed

**Step 1: Export your Argo CD resources**

Back up all Applications, ApplicationSets, and AppProjects:

```bash
kubectl get applications -n argocd -o yaml > applications-backup.yaml
kubectl get applicationsets -n argocd -o yaml > applicationsets-backup.yaml
kubectl get appprojects -n argocd -o yaml > appprojects-backup.yaml
```

These resources use standard `argoproj.io/v1alpha1` API and work without modification on the
EKS-managed capability.

**Step 2: Scale down self-managed Argo CD**

```bash
kubectl scale deployment/argocd-application-controller --replicas=0 -n argocd
kubectl scale deployment/argocd-server --replicas=0 -n argocd
kubectl scale deployment/argocd-repo-server --replicas=0 -n argocd
kubectl scale deployment/argocd-redis --replicas=0 -n argocd
```

**Step 3: Enable the EKS Argo CD capability**

Add the capability to your stack configuration:

```yaml
components:
  terraform:
    eks/cluster:
      vars:
        capabilities:
          argocd:
            type: ARGOCD
            configuration:
              argo_cd:
                namespace: argocd
                aws_idc:
                  idc_instance_arn: "arn:aws:sso:::instance/ssoins-abc123"
                rbac_role_mapping:
                  - role: ADMIN
                    identity:
                      - id: "<identity-center-user-or-group-id>"
                        type: SSO_GROUP
```

Apply the change. The EKS-managed Argo CD will start up and pick up existing Application
resources in the namespace.

**Step 4: Disable the self-managed Argo CD component**

Set `enabled: false` on your `eks/argocd` component (or remove it) and apply to clean up
the Helm release and its IAM resources.

**Step 5: Verify**

```bash
# Check capability status
aws eks describe-capability --cluster-name <name> --capability-name argocd

# Verify applications are syncing
argocd app list
```

**RBAC mapping**: EKS-managed Argo CD uses AWS Identity Center instead of Argo CD's built-in RBAC.
Map your existing roles to the three available roles: `ADMIN`, `EDITOR`, `VIEWER`.

### Migrating ACK (gradual migration)

ACK migration supports **gradual, side-by-side operation**. You can run self-managed and
EKS-managed ACK controllers simultaneously with proper coordination.

**Step 1: Update self-managed ACK leader election namespace**

For each ACK controller, update leader election to use `kube-system`:

```bash
helm upgrade --install ack-s3-controller \
  oci://public.ecr.aws/aws-controllers-k8s/s3-chart \
  --namespace ack-system \
  --set leaderElection.namespace=kube-system
```

This prevents conflicts when the EKS-managed capability starts.

**Step 2: Enable the EKS ACK capability**

```yaml
components:
  terraform:
    eks/cluster:
      vars:
        capabilities:
          ack:
            type: ACK
            iam_policy_arns:
              - "arn:aws:iam::aws:policy/AmazonS3FullAccess"
              - "arn:aws:iam::aws:policy/AmazonRDSFullAccess"
              # Add policies for each AWS service you manage via ACK
```

The EKS-managed ACK capability will recognize existing ACK custom resources and assume
reconciliation responsibility.

**Step 3: Scale down self-managed ACK controllers**

Gradually remove self-managed controllers as the managed capability takes over:

```bash
helm uninstall ack-s3-controller -n ack-system
```

**Step 4: Disable the self-managed ACK component**

Set `enabled: false` on your ACK component and apply.

**Key difference**: EKS-managed ACK uses a capability IAM role (trusted by
`capabilities.eks.amazonaws.com`) instead of IRSA. No service account annotations are needed.
Existing ACK-managed AWS resources continue to work without modification.

### Migrating KRO (gradual migration)

KRO migration supports **gradual, side-by-side operation**, similar to ACK.

**Step 1: Update self-managed KRO leader election namespace**

```bash
helm upgrade --install kro \
  oci://ghcr.io/awslabs/kro/kro-chart \
  --namespace kro \
  --set leaderElection.namespace=kube-system
```

**Step 2: Enable the EKS KRO capability**

```yaml
components:
  terraform:
    eks/cluster:
      vars:
        capabilities:
          kro:
            type: KRO
```

The managed capability uses the same upstream KRO controllers. ResourceGraphDefinitions and
instances work identically.

**Step 3: Scale down and remove self-managed KRO**

```bash
helm uninstall kro -n kro
```

### Delete Propagation Policy

When removing a capability from Terraform, the `delete_propagation_policy = RETAIN` (currently
the only supported value) means:

- **Kubernetes resources are retained** -- Applications, ACK resources, ResourceGraphDefinitions
  all remain in the cluster
- **AWS resources managed by ACK are governed by their individual deletion policies** -- they may
  be retained or deleted based on the ACK resource's own `deletionPolicy` annotation

> **Important**: Delete all Kubernetes resources managed by a capability **before** removing the
> capability itself to avoid orphaned resources that are no longer reconciled.
