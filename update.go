package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func cmdUpdateConfigFile(conf *KubeConfig) error {
	contexts, err := conf.Contexts()
	if err != nil {
		return fmt.Errorf("parse contexts: %w", err)
	}

	if len(contexts) == 0 {
		fmt.Println("no contexts defined")
		return nil
	}

	clusters := GroupContextsByCluster(contexts)

	loadingRules := &clientcmd.ClientConfigLoadingRules{Precedence: strings.Split(conf.file, ":")}
	apiConf, err := loadingRules.Load()
	if err != nil {
		return err
	}

	newContexts := make([]Context, 0)
	for _, cluster := range clusters {
		namespaces, err := getNamespacesInContextsCluster(apiConf, cluster.Contexts[0].Name)
		if err != nil {
			return fmt.Errorf("gather namespaces for cluster %q: %w", cluster.Name, err)
		}
		sort.Strings(namespaces)
		for _, ns := range namespaces {
			newContexts = append(newContexts, Context{
				Context: ContextData{
					Cluster:   cluster.Name,
					Namespace: ns,
					User:      cluster.Contexts[0].Context.User,
				},
				Name: fmt.Sprintf("%s-%s", cluster.Name, ns),
			})
		}
	}

	conf.SetContexts(newContexts)
	if err := conf.Save(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Println("contexts have been updated")
	return nil
}

func getNamespacesInContextsCluster(apiConf *api.Config, contextName string) ([]string, error) {
	config, err := clientcmd.NewDefaultClientConfig(*apiConf, &clientcmd.ConfigOverrides{CurrentContext: contextName}).ClientConfig()
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
