package clusterauth

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/url"
	"strings"

	"io/ioutil"

	"github.com/danielm-codefresh/argo-multi-cluster/pkg/common"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func NewCluster(name string, conf *rest.Config, managerBearerToken string) *Cluster {
	tlsClientConfig := TLSClientConfig{
		Insecure:   conf.TLSClientConfig.Insecure,
		ServerName: conf.TLSClientConfig.ServerName,
		CAData:     conf.TLSClientConfig.CAData,
		CertData:   conf.TLSClientConfig.CertData,
		KeyData:    conf.TLSClientConfig.KeyData,
	}
	if len(conf.TLSClientConfig.CAData) == 0 && conf.TLSClientConfig.CAFile != "" {
		data, err := ioutil.ReadFile(conf.TLSClientConfig.CAFile)
		if err != nil {
			return nil
		}
		tlsClientConfig.CAData = data
	}
	if len(conf.TLSClientConfig.CertData) == 0 && conf.TLSClientConfig.CertFile != "" {
		data, err := ioutil.ReadFile(conf.TLSClientConfig.CertFile)
		if err != nil {
			return nil
		}
		tlsClientConfig.CertData = data
	}
	if len(conf.TLSClientConfig.KeyData) == 0 && conf.TLSClientConfig.KeyFile != "" {
		data, err := ioutil.ReadFile(conf.TLSClientConfig.KeyFile)
		if err != nil {
			return nil
		}
		tlsClientConfig.KeyData = data
	}

	clst := Cluster{
		Server: conf.Host,
		Name:   name,
		Config: ClusterConfig{
			TLSClientConfig: tlsClientConfig,
		},
	}

	// Bearer token will preferentially be used for auth if present,
	// Even in presence of key/cert credentials
	// So set bearer token only if the key/cert data is absent
	if len(tlsClientConfig.CertData) == 0 || len(tlsClientConfig.KeyData) == 0 {
		clst.Config.BearerToken = managerBearerToken
	}

	return &clst
}

func ClusterToSecret(c Cluster) (*apiv1.Secret, error) {
	secName, err := uriToSecretName("cluster", c.Server)
	if err != nil {
		return nil, err
	}
	secret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secName,
			Labels: map[string]string{
				common.LabelKeySecretType: common.LabelValueSecretTypeCluster,
			},
		},
	}

	data := make(map[string][]byte)

	data["server"] = []byte(c.Server)
	data["name"] = []byte(c.Name)

	configBytes, err := json.Marshal(c.Config)
	if err != nil {
		return nil, err
	}
	data["config"] = configBytes
	secret.Data = data

	return secret, nil
}

func SecretToCluster(s apiv1.Secret) (*Cluster, error) {
	var config ClusterConfig
	if len(s.Data["config"]) > 0 {
		err := json.Unmarshal(s.Data["config"], &config)
		if err != nil {
			return nil, err
		}
	}

	cluster := &Cluster{
		Name:   string(s.Data["name"]),
		Server: string(s.Data["server"]),
		Config: config,
	}

	return cluster, nil
}

func (c *Cluster) RESTConfig() *rest.Config {
	tlsClientConfig := rest.TLSClientConfig{
		Insecure:   c.Config.TLSClientConfig.Insecure,
		ServerName: c.Config.TLSClientConfig.ServerName,
		CertData:   c.Config.TLSClientConfig.CertData,
		KeyData:    c.Config.TLSClientConfig.KeyData,
		CAData:     c.Config.TLSClientConfig.CAData,
	}

	config := &rest.Config{
		Host:            c.Server,
		Username:        c.Config.Username,
		Password:        c.Config.Password,
		BearerToken:     c.Config.BearerToken,
		TLSClientConfig: tlsClientConfig,
	}

	return config
}

func GetClusterSecret(clientset kubernetes.Interface, name string) (*apiv1.Secret, error) {
	defaultLabelSelector := fields.ParseSelectorOrDie(common.LabelKeySecretType + "=" + common.LabelValueSecretTypeCluster)
	argoCDLabelSelector := fields.ParseSelectorOrDie(common.LabelKeyArgoCDSecretType + "=" + common.LabelValueSecretTypeCluster)

	secrets, err := getSecretsWithLabel(clientset, defaultLabelSelector)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets.Items {
		if string(secret.Data["name"]) == name {
			return &secret, nil
		}
	}

	// If no secrets with the default argo label were found try and look for secrets with the argocd label
	secrets, err = getSecretsWithLabel(clientset, argoCDLabelSelector)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets.Items {
		if string(secret.Data["name"]) == name {
			return &secret, nil
		}
	}

	return nil, fmt.Errorf("Cluster secret %s was not found", name)
}

func uriToSecretName(uriType, uri string) (string, error) {
	parsedURI, err := url.ParseRequestURI(uri)
	if err != nil {
		return "", err
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(uri))
	host := strings.ToLower(strings.Split(parsedURI.Host, ":")[0])
	return fmt.Sprintf("%s-%s-%v", uriType, host, h.Sum32()), nil
}
func getSecretsWithLabel(clientset kubernetes.Interface, label fields.Selector) (*apiv1.SecretList, error) {
	fmt.Printf("Label: %s\n", label.String())
	secrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{LabelSelector: label.String()})
	if err != nil {
		return nil, err
	}

	return secrets, nil
}
