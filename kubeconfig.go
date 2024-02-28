package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func ReadKubeConfig() (*KubeConfig, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, fmt.Errorf("get file path: %w", err)
	}

	conf, err := readKubeConfigFromFile(kubeConfigFile)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return conf, nil
}

func getKubeConfigFile() (string, error) {
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

type KubeConfig struct {
	file string
	data map[string]interface{}
}

func readKubeConfigFromFile(file string) (*KubeConfig, error) {
	rawData, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(rawData, &data); err != nil {
		return nil, err
	}

	return &KubeConfig{
		file: file,
		data: data,
	}, nil
}

func (conf *KubeConfig) File() string {
	return conf.file
}

func (conf *KubeConfig) Save() error {
	rawData, err := yaml.Marshal(&conf.data)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(conf.file, rawData, os.ModePerm)
}

type Context struct {
	Context ContextData `yaml:"context"`
	Name    string      `yaml:"name"`
}

type ContextData struct {
	Cluster   string `yaml:"cluster"`
	Namespace string `yaml:"namespace"`
	User      string `yaml:"user"`
}

func (conf *KubeConfig) Contexts() ([]Context, error) {
	contextsObj, ok := conf.data["contexts"]
	if !ok {
		return nil, fmt.Errorf("missing 'contexts' in kube-config")
	}

	contextsList, ok := contextsObj.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type '%T' for 'contexts' in kube-config", contextsObj)
	}

	contexts := make([]Context, 0)
	for i, obj := range contextsList {
		objMap, ok := obj.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid type '%T' for 'contexts[%d]' in kube-config", obj, i)
		}

		nameObj, ok := objMap["name"]
		if !ok {
			return nil, fmt.Errorf("missing 'contexts[%d].name' in kube-config", i)
		}

		name, ok := nameObj.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type '%T' for 'contexts[%d].name' in kube-config", nameObj, i)
		}

		contextObj, ok := objMap["context"]
		if !ok {
			return nil, fmt.Errorf("missing 'contexts[%d].context' in kube-config", i)
		}

		contextMap, ok := contextObj.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid type '%T' for 'contexts[%d].context' in kube-config", contextObj, i)
		}

		clusterObj, ok := contextMap["cluster"]
		if !ok {
			return nil, fmt.Errorf("missing 'contexts[%d].context.cluster' in kube-config", i)
		}

		clusterName, ok := clusterObj.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type '%T' for 'contexts[%d].context.cluster' in kube-config", clusterObj, i)
		}

		namespaceObj, ok := contextMap["namespace"]
		if !ok {
			return nil, fmt.Errorf("missing 'contexts[%d].context.namespace' in kube-config", i)
		}

		namespace, ok := namespaceObj.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type '%T' for 'contexts[%d].context.namespace' in kube-config", namespaceObj, i)
		}

		userObj, ok := contextMap["user"]
		if !ok {
			return nil, fmt.Errorf("missing 'contexts[%d].context.user' in kube-config", i)
		}

		userName, ok := userObj.(string)
		if !ok {
			return nil, fmt.Errorf("invalid type '%T' for 'contexts[%d].context.user' in kube-config", userObj, i)
		}

		contexts = append(contexts, Context{
			Context: ContextData{
				Cluster:   clusterName,
				Namespace: namespace,
				User:      userName,
			},
			Name: name,
		})
	}
	return contexts, nil
}

func (conf *KubeConfig) SetContexts(contexts []Context) {
	conf.data["contexts"] = contexts
}

func (conf *KubeConfig) SelectedContext() (string, error) {
	currentContextObj, ok := conf.data["current-context"]
	if !ok {
		return "", nil
	}

	currentContext, ok := currentContextObj.(string)
	if !ok {
		return "", fmt.Errorf("invalid type '%T' for 'current-context' in kube-config", currentContextObj)
	}

	return currentContext, nil
}

func (conf *KubeConfig) SetSelectedContext(contextName string) {
	conf.data["current-context"] = contextName
}

type ClusterWithContexts struct {
	Name     string
	Contexts []Context
}

func GroupContextsByCluster(contexts []Context) []ClusterWithContexts {
	clustersMap := make(map[string][]Context)
	for _, c := range contexts {
		if cluster, ok := clustersMap[c.Context.Cluster]; ok {
			clustersMap[c.Context.Cluster] = append(cluster, c)
		} else {
			clustersMap[c.Context.Cluster] = []Context{c}
		}
	}

	clusters := make([]ClusterWithContexts, 0)
	for clusterName, contexts := range clustersMap {
		sort.Slice(contexts, func(i, j int) bool {
			return strings.Compare(contexts[i].Name, contexts[j].Name) < 0
		})

		clusters = append(clusters, ClusterWithContexts{
			Name:     clusterName,
			Contexts: contexts,
		})
	}

	sort.Slice(clusters, func(i, j int) bool {
		return strings.Compare(clusters[i].Name, clusters[j].Name) < 0
	})

	return clusters
}
