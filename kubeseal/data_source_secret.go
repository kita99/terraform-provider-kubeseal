package kubeseal

import (
	b64 "encoding/base64"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/kita99/terraform-provider-kubeseal/utils"
	"github.com/kita99/terraform-provider-kubeseal/utils/kubeseal"
)

func resourceSecret() *schema.Resource {
	return &schema.Resource{
		Create:   resourceSecretCreate,
		Read:   resourceSecretRead,
		Update:   resourceSecretUpdate,
		Delete:   resourceSecretDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the secret, must be unique",
			},
			"namespace": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Namespace of the secret",
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The secret type (ex. Opaque)",
			},
			"secrets": &schema.Schema{
				Type:        schema.TypeMap,
				Required:    true,
				Description: "Key/value pairs to populate the secret",
			},
			"controller_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the SealedSecrets controller in the cluster",
			},
			"controller_namespace": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Namespace of the SealedSecrets controller in the cluster",
			},
            "manifest": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Whatever output",
			},
		},
	}
}

func resourceSecretCreate(d *schema.ResourceData, kubeConfig interface{}) error {
    sealedSecretManifest, err := createSealedSecret(d, kubeConfig.(*KubeConfig))
    if err != nil {
        return err
    }

	utils.Log("Sealed secret has been created")

	d.SetId(utils.SHA256(sealedSecretManifest))
    d.Set("manifest", sealedSecretManifest)

	return nil
}

func resourceSecretRead(d *schema.ResourceData, kubeConfig interface{}) error {
	return nil
}

func resourceSecretUpdate(d *schema.ResourceData, kubeConfig interface{}) error {
    if !d.HasChange("secrets") {
        log.Printf("TOINE: Secrets remained the same skipping update")
        return nil
    }

    sealedSecretManifest, err := createSealedSecret(d, kubeConfig.(*KubeConfig))
    if err != nil {
        return err
    }

	utils.Log("Sealed secret has been updated")

	d.SetId(utils.SHA256(sealedSecretManifest))
    d.Set("manifest", sealedSecretManifest)

	return nil
}

func resourceSecretDelete(d *schema.ResourceData, kubeConfig interface{}) error {
    return nil
}

func createSealedSecret(d *schema.ResourceData, kubeConfig *KubeConfig) (string, error) {
	secrets := d.Get("secrets").(map[string]interface {})
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)

    secretsBase64 := map[string]interface{}{}
    for key, value := range secrets {
        strValue := fmt.Sprintf("%v", value) // rlly tho? jeez
        secretsBase64[key] = b64.StdEncoding.EncodeToString([]byte(strValue))
    }

    secretManifest, err := utils.GenerateSecretManifest(name, namespace, secretsBase64)
	if err != nil {
		return "", err
	}

	controllerName := d.Get("controller_name").(string)
	controllerNamespace := d.Get("controller_namespace").(string)

    rawCertificate, err := kubeseal.FetchCertificate(controllerName, controllerNamespace, kubeConfig.ClientConfig)
	if err != nil {
		return "", err
	}
	defer rawCertificate.Close()

    publicKey, err := kubeseal.ParseKey(rawCertificate)
	if err != nil {
		return "", err
	}

    sealedSecretManifest, err := kubeseal.Seal(secretManifest, publicKey, 0, false)
    if err != nil {
        return "", err
    }

    return sealedSecretManifest, nil
}
