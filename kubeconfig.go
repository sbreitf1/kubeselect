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

	conf, err := ParseKubeConfig(rawData)
	if err != nil {
		return nil, err
	}

	conf.file = kubeConfigFile
	return conf, nil
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

func ParseKubeConfig(data []byte) (*KubeConfig, error) {
	var conf KubeConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func (conf *KubeConfig) SanityCheck() {
	clusters := make(map[string]Cluster)
	for _, cluster := range conf.Clusters {
		clusters[cluster.Name] = cluster
	}
	users := make(map[string]User)
	for _, user := range conf.Users {
		users[user.Name] = user
	}

	// display unknown cluster or user references
	for _, context := range conf.Contexts {
		if _, ok := clusters[context.Data.Cluster]; !ok {
			fmt.Println("WARN: context", context.Name, "references unknown cluster", context.Data.Cluster)
		}
		if _, ok := users[context.Data.User]; !ok {
			fmt.Println("WARN: context", context.Name, "references unknown user", context.Data.User)
		}
	}

	// remove all referenced clusters and users from map to detect unreferenced ones
	for _, context := range conf.Contexts {
		delete(clusters, context.Data.Cluster)
		delete(users, context.Data.User)
	}
	for clusterName := range clusters {
		fmt.Println("WARN: cluster", clusterName, "is not referenced by any context")
	}
	for userName := range users {
		fmt.Println("WARN: user", userName, "is not referenced by any context")
	}
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

func (conf *KubeConfig) RemoveClusterByName(clusterName string) bool {
	for i := range conf.Clusters {
		if conf.Clusters[i].Name == clusterName {
			conf.Clusters = append(conf.Clusters[:i], conf.Clusters[i+1:]...)
			return true
		}
	}
	return false
}

func (conf *KubeConfig) RemoveContextByName(contextName string) bool {
	for i := range conf.Contexts {
		if conf.Contexts[i].Name == contextName {
			conf.Contexts = append(conf.Contexts[:i], conf.Contexts[i+1:]...)
			return true
		}
	}
	return false
}

func (conf *KubeConfig) RemoveUserByName(userName string) bool {
	for i := range conf.Users {
		if conf.Users[i].Name == userName {
			conf.Users = append(conf.Users[:i], conf.Users[i+1:]...)
			return true
		}
	}
	return false
}
