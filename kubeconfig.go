package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type KubeConfig struct {
	file           string
	APIVersion     string                   `yaml:"apiVersion"`
	Kind           string                   `yaml:"kind"`
	CurrentContext string                   `yaml:"current-context"`
	Clusters       []Cluster                `yaml:"clusters"`
	Contexts       []Context                `yaml:"contexts"`
	Users          []User                   `yaml:"users"`
	Preferences    map[string]interface{}   `yaml:"preferences"`
	Extensions     []map[string]interface{} `yaml:"extensions,omitempty"`
}

type Cluster struct {
	Name string                 `yaml:"name"`
	Data map[string]interface{} `yaml:"cluster"`
}

/*type ClusterData struct {
	Server                   string                   `yaml:"server"`
	TLSServerName            string                   `yaml:"tls-server-name,omitempty"`
	InsecureSkipTLSVerify    bool                     `yaml:"insecure-skip-tls-verify,omitempty"`
	CertificateAuthority     string                   `yaml:"certificate-authority,omitempty"`
	CertificateAuthorityData string                   `yaml:"certificate-authority-data,omitempty"`
	ProxyURL                 string                   `yaml:"proxy-url,omitempty"`
	DisableCompression       bool                     `yaml:"disable-compression,omitempty"`
	Extensions               []map[string]interface{} `yaml:"extensions,omitempty"`
}*/

type Context struct {
	Name string      `yaml:"name"`
	Data ContextData `yaml:"context"`
}

type ContextData struct {
	Cluster    string                   `yaml:"cluster"`
	User       string                   `yaml:"user"`
	Namespace  string                   `yaml:"namespace,omitempty"`
	Extensions []map[string]interface{} `yaml:"extensions,omitempty"`
}

type User struct {
	Name string                 `yaml:"name"`
	Data map[string]interface{} `yaml:"user"`
}

func ReadKubeConfig() (*KubeConfig, error) {
	kubeConfigFile, err := getKubeConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("get file path: %w", err)
	}

	rawData, err := os.ReadFile(kubeConfigFile)
	if err != nil {
		return nil, err
	}

	var conf KubeConfig
	if err := yaml.Unmarshal(rawData, &conf); err != nil {
		return nil, err
	}

	conf.file = kubeConfigFile
	return &conf, nil
}

func getKubeConfigFilePath() (string, error) {
	kubeConfigFile := os.Getenv("KUBECONFIG")

	if len(kubeConfigFile) == 0 {
		// try default config path in user home dir
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		kubeConfigFile = filepath.Join(usr.HomeDir, "/.kube/config")
	}

	return kubeConfigFile, nil
}

func (conf *KubeConfig) File() string {
	return conf.file
}

func (conf *KubeConfig) Save() error {
	rawData, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(conf.file, rawData, os.ModePerm)
}

func (conf *KubeConfig) GetSortedClusterNames() []string {
	clusterNames := make([]string, 0, len(conf.Clusters))
	for _, cluster := range conf.Clusters {
		clusterNames = append(clusterNames, cluster.Name)
	}
	sort.Strings(clusterNames)
	return clusterNames
}

func (conf *KubeConfig) GetContextsForCluster(clusterName string) []Context {
	contexts := make([]Context, 0)
	for _, context := range conf.Contexts {
		if context.Data.Cluster == clusterName {
			contexts = append(contexts, context)
		}
	}
	return contexts
}
