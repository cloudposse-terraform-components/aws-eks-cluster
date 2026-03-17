locals {
  github_role_enabled       = local.enabled && var.github_actions_iam_role_enabled
  github_actions_iam_policy = data.aws_iam_policy_document.github_actions_iam_policy.json

  github_actions_access_entry = local.github_role_enabled ? [
    {
      principal_arn = aws_iam_role.github_actions[0].arn
    }
  ] : []
  github_actions_access_policy = local.github_role_enabled ? [
    {
      principal_arn = aws_iam_role.github_actions[0].arn
      policy_arn    = "ClusterAdmin"
      access_scope  = {} # use defaults of cluster-wide, all namespaces
    }
  ] : []
}

data "aws_iam_policy_document" "github_actions_iam_policy" {
  # Allow actions on this EKS Cluster
  statement {
    sid    = "AllowEKSActions"
    effect = "Allow"
    actions = [
      "eks:DescribeCluster",
      "eks:DescribeNodegroup",
      "eks:ListClusters",
      "eks:ListNodegroups"
    ]
    resources = [module.eks_cluster.eks_cluster_arn]
  }

  # Allow chamber to read secrets
  statement {
    sid    = "AllowKMSAccess"
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:DescribeKey"
    ]
    resources = [
      "*"
    ]
  }
  statement {
    effect = "Allow"
    actions = [
      "ssm:GetParameters"
    ]
    resources = [
      "arn:aws:ssm:*:*:parameter/platform/${module.eks_cluster.eks_cluster_id}/*"
    ]
  }
  statement {
    effect = "Allow"
    actions = [
      "ssm:GetParametersByPath"
    ]
    resources = [
      "*"
    ]
  }
}
