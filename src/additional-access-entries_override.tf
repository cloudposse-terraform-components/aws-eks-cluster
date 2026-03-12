# This file is used to override the default access entries and policies.
locals {
  # Github Action access is defined with the policy in github-actions-iam-policy.mixin.tf
  overridable_access_entry  = local.github_actions_access_entry
  overridable_access_policy = local.github_actions_access_policy
  overridable_access_map    = {}
}
