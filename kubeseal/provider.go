package kubeseal

import (
    "os"
    "fmt"
    "bytes"
    "path/filepath"
    "sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/kita99/terraform-provider-kubeseal/utils"

    "github.com/mitchellh/go-homedir"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/rest"
    clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
    apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
            "kubernetes": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Kubernetes configuration.",
				Elem:        kubernetesResource(),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"kubeseal_secret": resourceSecret(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func kubernetesResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_HOST", ""),
				Description: "The hostname (in form of URI) of Kubernetes master.",
			},
			"username": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_USER", ""),
				Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_PASSWORD", ""),
				Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_INSECURE", false),
				Description: "Whether server should be accessed without verifying the TLS certificate.",
			},
			"client_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLIENT_CERT_DATA", ""),
				Description: "PEM-encoded client certificate for TLS authentication.",
			},
			"client_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLIENT_KEY_DATA", ""),
				Description: "PEM-encoded client certificate key for TLS authentication.",
			},
			"cluster_ca_certificate": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CLUSTER_CA_CERT_DATA", ""),
				Description: "PEM-encoded root certificates bundle for TLS authentication.",
			},
			"config_paths": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
			},
			"config_path": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("KUBE_CONFIG_PATH", nil),
				Description:   "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
				ConflictsWith: []string{"kubernetes.0.config_paths"},
			},
			"config_context": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX", ""),
			},
			"config_context_auth_info": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX_AUTH_INFO", ""),
				Description: "",
			},
			"config_context_cluster": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_CTX_CLUSTER", ""),
				Description: "",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KUBE_TOKEN", ""),
				Description: "Token to authenticate an service account",
			},
			"exec": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"api_version": {
							Type:     schema.TypeString,
							Required: true,
						},
						"command": {
							Type:     schema.TypeString,
							Required: true,
						},
						"env": {
							Type:     schema.TypeMap,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"args": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				Description: "",
			},
		},
	}
}

type KubeConfig struct {
	ClientConfig clientcmd.ClientConfig

	sync.Mutex
}

var k8sPrefix = "kubernetes.0."

func k8sGetOk(d *schema.ResourceData, key string) (interface{}, bool) {
	value, ok := d.GetOk(k8sPrefix + key)

	// For boolean attributes the zero value is Ok
	switch value.(type) {
	case bool:
		// TODO: replace deprecated GetOkExists with SDK v2 equivalent
		// https://github.com/hashicorp/terraform-plugin-sdk/pull/350
		value, ok = d.GetOkExists(k8sPrefix + key)
	}

	// fix: DefaultFunc is not being triggered on TypeList
	s := kubernetesResource().Schema[key]
	if !ok && s.DefaultFunc != nil {
		value, _ = s.DefaultFunc()

		switch v := value.(type) {
		case string:
			ok = len(v) != 0
		case bool:
			ok = v
		}
	}

	return value, ok
}

func k8sGet(d *schema.ResourceData, key string) interface{} {
	value, _ := k8sGetOk(d, key)
	return value
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
    overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}

	if v, ok := k8sGetOk(d, "config_path"); ok && v != "" {
		configPaths = []string{v.(string)}
	} else if v, ok := k8sGetOk(d, "config_paths"); ok {
		for _, p := range v.([]interface{}) {
			configPaths = append(configPaths, p.(string))
		}
	} else if v := os.Getenv("KUBE_CONFIG_PATHS"); v != "" {
		// NOTE we have to do this here because the schema
		// does not yet allow you to set a default for a TypeList
		configPaths = filepath.SplitList(v)
	}

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}

			utils.Log(fmt.Sprintf("[DEBUG] Using kubeconfig: %s", path))
			expandedPaths = append(expandedPaths, path)
		}

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctx, ctxOk := k8sGetOk(d, "config_context")
		authInfo, authInfoOk := k8sGetOk(d, "config_context_auth_info")
		cluster, clusterOk := k8sGetOk(d, "config_context_cluster")
		if ctxOk || authInfoOk || clusterOk {
			if ctxOk {
				overrides.CurrentContext = ctx.(string)
                utils.Log(fmt.Sprintf("[DEBUG] Using custom current context: %q", overrides.CurrentContext))
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfoOk {
				overrides.Context.AuthInfo = authInfo.(string)
			}
			if clusterOk {
				overrides.Context.Cluster = cluster.(string)
			}
            utils.Log(fmt.Sprintf("[DEBUG] Using overidden context: %#v", overrides.Context))
		}
	}

	// Overriding with static configuration
	if v, ok := k8sGetOk(d, "insecure"); ok {
		overrides.ClusterInfo.InsecureSkipTLSVerify = v.(bool)
	}
	if v, ok := k8sGetOk(d, "cluster_ca_certificate"); ok {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(d, "client_certificate"); ok {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(d, "host"); ok {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(v.(string), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, err
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if v, ok := k8sGetOk(d, "username"); ok {
		overrides.AuthInfo.Username = v.(string)
	}
	if v, ok := k8sGetOk(d, "password"); ok {
		overrides.AuthInfo.Password = v.(string)
	}
	if v, ok := k8sGetOk(d, "client_key"); ok {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := k8sGetOk(d, "token"); ok {
		overrides.AuthInfo.Token = v.(string)
	}

	if v, ok := k8sGetOk(d, "exec"); ok {
		exec := &clientcmdapi.ExecConfig{}
		if spec, ok := v.([]interface{})[0].(map[string]interface{}); ok {
			exec.APIVersion = spec["api_version"].(string)
			exec.Command = spec["command"].(string)
			exec.Args = utils.ExpandStringSlice(spec["args"].([]interface{}))
			for kk, vv := range spec["env"].(map[string]interface{}) {
				exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: kk, Value: vv.(string)})
			}
		} else {
            utils.Log("[ERROR] Failed to parse exec")
			return nil, fmt.Errorf("failed to parse exec")
		}
		overrides.AuthInfo.Exec = exec
	}

	overrides.Context.Namespace = "default"

    namespace := d.Get("namespace")

	if namespace != nil {
		overrides.Context.Namespace = namespace.(string)
	}

	client := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	if client == nil {
        utils.Log("[ERROR] Failed to initialize kubernetes config")
		return nil, nil
	}
    utils.Log("[INFO] Successfully initialized kubernetes config")

	return &KubeConfig{ClientConfig: client}, nil
}
