terraform {
  required_version = ">= 1.3.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.9.0, < 6.0.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
    }
    # cloudposse/utils v1.33.0 introduced a regression where the embedded atmos no longer
    # returns `workspace` in utils_component_config output, breaking remote state lookups.
    # Pin below 1.33.0 until upstream resolves the regression.
    # See: https://github.com/cloudposse/terraform-provider-utils/issues (v1.33.0 / atmos v1.209.0)
    utils = {
      source  = "cloudposse/utils"
      version = ">= 1.7.1, != 1.4.0, < 1.33.0"
    }
    # We no longer use the Kubernetes provider, so we can remove it,
    # but since there are bugs in the current version, we keep this as a comment.
    #   kubernetes = {
    #     source = "hashicorp/kubernetes"
    #     # Version 2.25 and higher have bugs, so we cannot allow them,
    #     # but automation enforces that we have no upper limit.
    #     # It is less critical here, because the Kubernetes provider is being removed entirely.
    #     version = "2.24"
    #   }
  }
}
