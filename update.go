package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
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
					Name: genContextName(cluster.Name, ns),
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

func genContextName(clusterName, namespace string) string {
	return fmt.Sprintf("%s-%s", clusterName, namespace)
}

func findUserForCluster(conf *KubeConfig, apiConf *api.Config, clusterName string) (string, error) {
	if user, ok := findUserForClusterFromExistingContext(conf, clusterName); ok {
		return user, nil
	}

	//TODO maybe interactive selection instead?
	fmt.Println("WARN: no contexts for cluster", clusterName, "defined yet. need to auto-detect user")

	userNames := make([]string, 0, len(conf.Users))
	for _, user := range conf.Users {
		userNames = append(userNames, user.Name)
	}

	// sort user names by similarity to cluster name to increase probability of finding the right one early
	metric := metrics.NewSorensenDice()
	metric.CaseSensitive = false
	metric.NgramSize = 2
	sort.Slice(userNames, func(i, j int) bool {
		si := strutil.Similarity(clusterName, userNames[i], metric)
		sj := strutil.Similarity(clusterName, userNames[j], metric)
		return si > sj
	})

	// now try every user in the list
	for _, userName := range userNames {
		//TODO try in parallel?
		fmt.Println("-> try user", userName, "for cluster", clusterName)
		ok, err := isCorrectUserForCluster(apiConf, clusterName, userName)
		if err != nil {
			return "", err
		}

		if ok {
			return userName, nil
		}
	}

	return "", fmt.Errorf("no suitable user defined")
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

func isCorrectUserForCluster(apiConf *api.Config, clusterName, userName string) (bool, error) {
	//TODO some smarter way to decide if user is correct
	_, err := getNamespacesInContextsCluster(apiConf, clusterName, userName)
	return err == nil, nil
}

func getNamespacesInContextsCluster(apiConf *api.Config, clusterName, userName string) ([]string, error) {
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
