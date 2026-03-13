# aws-team-roles.tf
# IAM role ARN resolution for aws_team_roles_rbac using account-map.
#
# Non-account-map architectures: exclude this file and replace with the
# aws-team-roles-no-account-map mixin (see mixins repo).
# Those users should use map_additional_iam_roles with explicit full ARNs.

locals {
  this_account_name = module.iam_roles.current_account_account_name

  role_map = {
    (local.this_account_name) = var.aws_team_roles_rbac[*].aws_team_role
  }

  aws_team_roles_auth = [for role in var.aws_team_roles_rbac : {
    rolearn = module.iam_arns.principals_map[local.this_account_name][role.aws_team_role]
    groups  = role.groups
  }]

  aws_team_roles_access_entry_map = {
    for role in local.aws_team_roles_auth : role.rolearn => {
      kubernetes_groups = role.groups
    }
  }
}

module "iam_arns" {
  source = "../../account-map/modules/roles-to-principals"

  role_map = local.role_map

  context = module.this.context
}
