variable "map_additional_worker_roles" {
  type        = list(string)
  description = <<-EOT
    (Deprecated) AWS IAM Role ARNs of unmanaged Linux worker nodes to grant access to the EKS cluster.
    In earlier versions, this could be used to grant access to worker nodes of any type
    that were not managed by the EKS cluster. Now EKS requires that unmanaged worker nodes
    be classified as Linux or Windows servers, in this input is temporarily retained
    with the assumption that all worker nodes are Linux servers. (It is likely that
    earlier versions did not work properly with Windows worker nodes anyway.)
    This input is deprecated and will be removed in a future release.
    In the future, this component will either have a way to separate Linux and Windows worker nodes,
    or drop support for unmanaged worker nodes entirely.
    EOT
  default     = []
  nullable    = false
}
