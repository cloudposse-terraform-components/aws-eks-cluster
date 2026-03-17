# aws-team-roles-no-account-map.tf
#
# Disables aws_team_roles_rbac for architectures without account-map.
# Use map_additional_iam_roles with explicit full role ARNs instead.

locals {
  aws_team_roles_auth             = []
  aws_team_roles_access_entry_map = {}
}
