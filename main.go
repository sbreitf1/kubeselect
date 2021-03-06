package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type cluster struct {
	Name     string
	Contexts []context
}

type context struct {
	Name        string
	ClusterName string
	Namespace   string
}

func main() {
	kubeConfigFile := os.Getenv("KUBECONFIG")

	if len(kubeConfigFile) == 0 {
		// try default config path in user home dir
		usr, err := user.Current()
		if err != nil {
			fmt.Println("ERROR:", err.Error())
			os.Exit(1)
		}
		kubeConfigFile = filepath.Join(usr.HomeDir, "/.kube/config")
	}

	data, err := ioutil.ReadFile(kubeConfigFile)
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	var conf interface{}
	if err := yaml.Unmarshal(data, &conf); err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	contexts, err := getContexts(conf)
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	if len(contexts) == 0 {
		fmt.Println("no contexts defined")
		os.Exit(0)
	}

	selectedContext, err := getSelectedContext(conf)
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	clusters := groupContextsByCluster(contexts)

	userSelectedContext, err := selectContext(clusters, selectedContext)
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	if len(userSelectedContext) > 0 {
		if userSelectedContext == selectedContext {
			// nothing to do
			return
		}

		setSelectedContext(&conf, userSelectedContext)

		data, err := yaml.Marshal(&conf)
		if err != nil {
			fmt.Println("ERROR:", err.Error())
			os.Exit(1)
		}

		if err := ioutil.WriteFile(kubeConfigFile, data, os.ModePerm); err != nil {
			fmt.Println("ERROR:", err.Error())
			os.Exit(1)
		}

		fmt.Println(fmt.Sprintf("switched to context %q", userSelectedContext))
	}
}

func getContexts(conf interface{}) ([]context, error) {
	confMap, ok := conf.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid root type '%T' for kube-config", conf)
	}

	contextsObj, ok := confMap["contexts"]
	if !ok {
		return nil, fmt.Errorf("missing 'contexts' in kube-config")
	}

	contextsList, ok := contextsObj.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid type '%T' for 'contexts' in kube-config", contextsObj)
	}

	contexts := make([]context, 0)
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

		contexts = append(contexts, context{
			Name:        name,
			ClusterName: clusterName,
			Namespace:   namespace,
		})
	}
	return contexts, nil
}

func groupContextsByCluster(contexts []context) []cluster {
	clustersMap := make(map[string][]context)
	for _, c := range contexts {
		if cluster, ok := clustersMap[c.ClusterName]; ok {
			clustersMap[c.ClusterName] = append(cluster, c)
		} else {
			clustersMap[c.ClusterName] = []context{c}
		}
	}

	clusters := make([]cluster, 0)
	for clusterName, contexts := range clustersMap {
		sort.Slice(contexts, func(i, j int) bool {
			return strings.Compare(contexts[i].Name, contexts[j].Name) < 0
		})

		clusters = append(clusters, cluster{
			Name:     clusterName,
			Contexts: contexts,
		})
	}

	sort.Slice(clusters, func(i, j int) bool {
		return strings.Compare(clusters[i].Name, clusters[j].Name) < 0
	})

	return clusters
}

func getSelectedContext(conf interface{}) (string, error) {
	confMap, ok := conf.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid root type '%T' for kube-config", conf)
	}

	currentContextObj, ok := confMap["current-context"]
	if !ok {
		return "", nil
	}

	currentContext, ok := currentContextObj.(string)
	if !ok {
		return "", fmt.Errorf("invalid type '%T' for 'current-context' in kube-config", currentContextObj)
	}

	return currentContext, nil
}

func setSelectedContext(conf *interface{}, contextName string) {
	(*conf).(map[string]interface{})["current-context"] = contextName
}
