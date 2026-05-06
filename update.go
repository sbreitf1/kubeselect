package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func cmdUpdateConfigFile(conf *KubeConfig) error {
	if len(conf.Clusters) == 0 {
		return fmt.Errorf("no clusters defined")
	}
	if len(conf.Users) == 0 {
		return fmt.Errorf("no users defined")
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{Precedence: []string{conf.File()}}
	apiConf, err := loadingRules.Load()
	if err != nil {
		return err
	}

	var m sync.Mutex
	contextsByCluster := make(map[string][]Context)
	for _, context := range conf.Contexts {
		if _, ok := contextsByCluster[context.Data.Cluster]; !ok {
			contextsByCluster[context.Data.Cluster] = make([]Context, 0)
		}
		contextsByCluster[context.Data.Cluster] = append(contextsByCluster[context.Data.Cluster], context)
	}

	var wg sync.WaitGroup
	for _, cluster := range conf.Clusters {
		wg.Go(func() {
			clusterUser, err := findUserForCluster(conf, apiConf, cluster.Name)
			if err != nil {
				fmt.Println("ERR: failed to detect user for cluster "+cluster.Name+":", err)
				return
			}

			namespaces, err := getNamespacesInContextsCluster(apiConf, cluster.Name, clusterUser)
			if err != nil {
				fmt.Println("ERR: failed to gather namespaces for cluster "+cluster.Name+":", err)
				return
			}

			sort.Strings(namespaces)
			newContexts := make([]Context, 0, len(namespaces))
			for _, ns := range namespaces {
				newContexts = append(newContexts, Context{
					Data: ContextData{
						Cluster:   cluster.Name,
						Namespace: ns,
						User:      clusterUser,
					},
					Name: fmt.Sprintf("%s-%s", cluster.Name, ns),
				})
			}

			//fmt.Println("found", len(newContexts), "contexts for cluster", cluster.Name)

			m.Lock()
			defer m.Unlock()
			contextsByCluster[cluster.Name] = newContexts
		})
	}
	wg.Wait()

	conf.Contexts = make([]Context, 0)
	for _, contexts := range contextsByCluster {
		conf.Contexts = append(conf.Contexts, contexts...)
	}

	if err := conf.Save(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Println("contexts have been updated")
	return nil
}

func findUserForCluster(conf *KubeConfig, apiConf *api.Config, clusterName string) (string, error) {
	if user, ok := findUserForClusterFromExistingContext(conf, clusterName); ok {
		return user, nil
	}

	//fmt.Println("WARN: no contexts for cluster", clusterName, "defined yet. need to auto-detect user")

	//TODO auto-detect user for cluster
	return "", fmt.Errorf("user auto-detection not implemented")
}

func findUserForClusterFromExistingContext(conf *KubeConfig, clusterName string) (string, bool) {
	for _, context := range conf.Contexts {
		if context.Data.Cluster == clusterName {
			for _, user := range conf.Users {
				if user.Name == context.Data.User {
					// user exists, we can use it
					return context.Data.User, true
				}
			}
			fmt.Println("WARN: context", context.Name, "refers to undefined user", context.Data.User)
		}
	}
	return "", false
}

func getNamespacesInContextsCluster(apiConf *api.Config, clusterName, userName string) ([]string, error) {
	//config, err := clientcmd.NewDefaultClientConfig(*apiConf, &clientcmd.ConfigOverrides{CurrentContext: contextName}).ClientConfig()
	config, err := prepareClientConfig(apiConf, clusterName, userName)
	if err != nil {
		return nil, err
	}

	config.Timeout = 2 * time.Second
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("read namespaces from api: %w", err)
	}

	namespaceNames := make([]string, len(namespaces.Items))
	for i := range namespaces.Items {
		namespaceNames[i] = namespaces.Items[i].Name
	}
	return namespaceNames, nil
}

func prepareClientConfig(apiConf *api.Config, clusterName, userName string) (*rest.Config, error) {
	tmpConf := apiConf.DeepCopy()
	tmpConf.Contexts["kubeselect-discovery"] = &api.Context{
		Cluster:  clusterName,
		AuthInfo: userName,
	}
	return clientcmd.NewDefaultClientConfig(*tmpConf, &clientcmd.ConfigOverrides{CurrentContext: "kubeselect-discovery"}).ClientConfig()
}
