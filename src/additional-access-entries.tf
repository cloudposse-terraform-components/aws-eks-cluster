locals {
  # If you have custom access entries and policies, create a file called `additional-access-entries_override.tf`
  # and in that file override any of the following declarations as needed.
  #
  # For example, we use this to add a policy for the GitHub Actions OIDC role
  overridable_access_entry  = []
  overridable_access_policy = []
}
