terraform-provider-kubeseal
================================

A very barebones provider that exposes basic `kubeseal` functionality as a terraform data source.


### Usage

```HCL
terraform {
  required_providers {
    kubeseal = {
      source = "kita99/kubeseal"
      version = "0.1.0"
    }
  }
}

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
```


### Argument Reference

The following arguments are supported:
- `name` - Name of the secret, must be unique.
- `namespace` - Namespace defines the space within which name of the secret must be unique.
- `type` -  The secret type. ex: `Opaque`
- `secrets` - Key/value pairs to populate the secret
- `controller_name` - Name of the SealedSecrets controller in the cluster
- `controller_namespace` - Namespace of the SealedSecrets controller in the cluster
- `depends_on` - For specifying hidden dependencies.

*NOTE: All the arguments above are required*
