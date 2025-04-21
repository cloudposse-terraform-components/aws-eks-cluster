# Placeholder for Terraform validation.
# If you have a GitHub OIDC role deployed with this component, you can override this file.
#
# To do this, create a file called `github-actions-iam-policy.tf` in the same directory as this module
# and exclude this file from the vendor configuration, if necessary.
locals {
  github_actions_access_entry  = []
  github_actions_access_policy = []
}
