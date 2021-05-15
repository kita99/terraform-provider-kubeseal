package utils

import (
    "io"
	"crypto/sha256"
	"fmt"
	"log"
	"bytes"
    "text/template"
)

type SecretManifest struct {
    Name string
    Namespace string
    Secrets map[string]interface {}
}

func SHA256(src string) string {
	h := sha256.New()
	h.Write([]byte(src))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Log(message string) {
	log.Printf("[kubeseal_provider] ================= %s\n", message)
}

func GenerateSecretManifest(name string, namespace string, secrets map[string]interface {}) (io.Reader, error) {
    secretManifestYAML := new(bytes.Buffer)

    paths := []string{
        "secret.tmpl",
	}

    secretManifest := SecretManifest{
        Name: name,
        Namespace: namespace,
        Secrets: secrets,
    }

    t := template.Must(template.New("secret.tmpl").ParseFiles(paths...))
    err := t.Execute(secretManifestYAML, secretManifest)
	if err != nil {
		return nil, err
	}

    return secretManifestYAML, nil
}

func ExpandStringSlice(s []interface{}) []string {
	result := make([]string, len(s), len(s))
	for k, v := range s {
		// Handle the Terraform parser bug which turns empty strings in lists to nil.
		if v == nil {
			result[k] = ""
		} else {
			result[k] = v.(string)
		}
	}
	return result
}

