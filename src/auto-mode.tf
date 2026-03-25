# IAM Role for EC2 instances launched by EKS Auto Mode
#
# https://docs.aws.amazon.com/eks/latest/userguide/auto-create-node-role.html
# https://docs.aws.amazon.com/eks/latest/userguide/automode.html

locals {
  auto_mode_enabled          = local.enabled && var.auto_mode_enabled
  auto_mode_node_role_needed = local.auto_mode_enabled && var.auto_mode_node_role_arn == null

  auto_mode_node_role_arn = local.auto_mode_enabled ? coalesce(
    var.auto_mode_node_role_arn,
    one(aws_iam_role.auto_mode_node[*].arn)
  ) : null

  # Used to determine correct partition (i.e. - `aws`, `aws-gov`, `aws-cn`, etc.)
  # Reuse partition from karpenter.tf if available, otherwise look it up
  auto_mode_partition = local.auto_mode_node_role_needed ? one(data.aws_partition.auto_mode[*].partition) : null
}

check "karpenter_auto_mode_conflict" {
  assert {
    condition     = !(var.auto_mode_enabled && var.karpenter_iam_role_enabled)
    error_message = "karpenter_iam_role_enabled and auto_mode_enabled cannot both be true. Auto Mode includes managed Karpenter."
  }
}

data "aws_partition" "auto_mode" {
  count = local.auto_mode_node_role_needed ? 1 : 0
}

module "auto_mode_node_label" {
  source  = "cloudposse/label/null"
  version = "0.25.0"

  enabled    = local.auto_mode_node_role_needed
  attributes = ["auto-mode-node"]

  context = module.this.context
}

data "aws_iam_policy_document" "auto_mode_node_assume_role" {
  count = local.auto_mode_node_role_needed ? 1 : 0

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "auto_mode_node" {
  count = local.auto_mode_node_role_needed ? 1 : 0

  name               = module.auto_mode_node_label.id
  description        = "IAM Role for EC2 instances launched by EKS Auto Mode"
  assume_role_policy = data.aws_iam_policy_document.auto_mode_node_assume_role[0].json
  tags               = module.auto_mode_node_label.tags
}

# AmazonEKSWorkerNodeMinimalPolicy - minimal permissions for Auto Mode nodes
resource "aws_iam_role_policy_attachment" "auto_mode_node_minimal" {
  count = local.auto_mode_node_role_needed ? 1 : 0

  role       = one(aws_iam_role.auto_mode_node[*].name)
  policy_arn = format("arn:%s:iam::aws:policy/AmazonEKSWorkerNodeMinimalPolicy", local.auto_mode_partition)
}

# AmazonEC2ContainerRegistryPullOnly - pull images from ECR
resource "aws_iam_role_policy_attachment" "auto_mode_node_ecr" {
  count = local.auto_mode_node_role_needed ? 1 : 0

  role       = one(aws_iam_role.auto_mode_node[*].name)
  policy_arn = format("arn:%s:iam::aws:policy/AmazonEC2ContainerRegistryPullOnly", local.auto_mode_partition)
}
