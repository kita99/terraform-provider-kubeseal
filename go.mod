module github.com/kita99/terraform-provider-kubeseal

go 1.16

require (
	github.com/bitnami-labs/sealed-secrets v0.16.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.6.1
	github.com/mitchellh/go-homedir v1.1.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
)
