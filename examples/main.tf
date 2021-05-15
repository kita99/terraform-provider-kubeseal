provider "kubeseal" {
}

data "kubeseal_secret" "my_secret" {
  name = "my-secret"
  namespace = kubernetes_namespace.example_ns.metadata.0.name
  type = "Opaque"

  secrets = {
    key = "value"
  }
  controller_name = "sealed-secret-controller"
  controller_namespace = "default"

  depends_on = [kubernetes_namespace.example_ns, var.sealed_secrets_controller_id]
}
