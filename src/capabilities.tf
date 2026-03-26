# EKS Capabilities: Argo CD, ACK, KRO
# https://docs.aws.amazon.com/eks/latest/userguide/capabilities.html

locals {
  enabled_capabilities = {
    for k, v in var.capabilities : k => v if local.enabled && v.enabled
  }

  capabilities_needing_roles = {
    for k, v in local.enabled_capabilities : k => v if v.role_arn == null
  }

  # Build the map passed to the module (strip component-only fields, inject role ARNs)
  module_capabilities = {
    for k, v in var.capabilities : k => {
      enabled                   = v.enabled
      type                      = v.type
      role_arn                  = v.role_arn != null ? v.role_arn : try(aws_iam_role.capability[k].arn, null)
      delete_propagation_policy = v.delete_propagation_policy
      configuration             = v.configuration
      create_timeout            = v.create_timeout
      update_timeout            = v.update_timeout
      delete_timeout            = v.delete_timeout
    }
  }

  # Flatten policy attachments for for_each
  capability_policy_attachments = merge([
    for cap_name, cap in local.capabilities_needing_roles : {
      for idx, arn in cap.iam_policy_arns :
      "${cap_name}-${idx}" => {
        role_name  = aws_iam_role.capability[cap_name].name
        policy_arn = arn
      }
    }
  ]...)
}

module "capability_label" {
  for_each = local.capabilities_needing_roles

  source  = "cloudposse/label/null"
  version = "0.25.0"

  enabled    = true
  attributes = ["capability", each.key]

  context = module.this.context
}

data "aws_iam_policy_document" "capability_assume_role" {
  count = length(local.capabilities_needing_roles) > 0 ? 1 : 0

  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole", "sts:TagSession"]

    principals {
      type        = "Service"
      identifiers = ["capabilities.eks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "capability" {
  for_each = local.capabilities_needing_roles

  name               = module.capability_label[each.key].id
  description        = "IAM Role for EKS ${each.value.type} capability"
  assume_role_policy = one(data.aws_iam_policy_document.capability_assume_role[*].json)
  tags               = module.capability_label[each.key].tags
}

resource "aws_iam_role_policy_attachment" "capability" {
  for_each = local.capability_policy_attachments

  role       = each.value.role_name
  policy_arn = each.value.policy_arn
}
