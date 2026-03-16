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
    # cloudposse/utils v1.32.0+ (released 2026-03-11) disabled template and YAML function
    # processing in the embedded atmos (PR #527: atmos v1.209.0), which broke path template
    # evaluation and workspace detection in utils_component_config, causing remote state
    # lookups to fail. Pin to v1.31.0 until upstream resolves the regression.
    # See: https://github.com/cloudposse/terraform-provider-utils/pull/527
    utils = {
      source  = "cloudposse/utils"
      version = ">= 1.7.1, != 1.4.0, < 1.32.0"
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
