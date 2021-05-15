module github.com/kita99/terraform-provider-kubeseal

go 1.16

require (
	github.com/bitnami-labs/sealed-secrets v0.16.0
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.4.0
	github.com/mitchellh/go-homedir v1.1.0
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
)
