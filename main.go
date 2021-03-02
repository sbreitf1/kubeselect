package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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
		fmt.Println("ERROR: no KUBECONFIG defined")
		os.Exit(1)
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

	var userSelectedContext string

	app := tview.NewApplication()
	rootNode := tview.NewTreeNode("Clusters").SetSelectable(false)
	treeView := tview.NewTreeView().SetRoot(rootNode)
	var firstContextNode *tview.TreeNode
	for _, cluster := range clusters {
		clusterNode := tview.NewTreeNode(cluster.Name).SetSelectable(false)
		clusterNode.SetColor(tcell.ColorGreen)
		for _, c := range cluster.Contexts {
			contextNode := tview.NewTreeNode(c.Name).SetSelectable(true)
			if c.Name == selectedContext {
				// this is the currently selected context
				contextNode.SetColor(tcell.ColorYellow)
				treeView.SetCurrentNode(contextNode)
			} else {
				contextNode.SetColor(tcell.ColorTurquoise)
			}
			// put iterator variable value in block-local variable so the lambda function
			// does not access the wrong field afterwards
			contextName := c.Name
			contextNode.SetSelectedFunc(func() {
				userSelectedContext = contextName
				app.Stop()
			})
			if firstContextNode == nil {
				// remember first visible node as fallback for default selection
				firstContextNode = contextNode
			}
			clusterNode.AddChild(contextNode)
		}
		rootNode.AddChild(clusterNode)
	}

	if treeView.GetCurrentNode() == nil {
		// no default selection? fall back to first visible node
		treeView.SetCurrentNode(firstContextNode)
	}

	app.SetRoot(treeView, true)
	if err := app.Run(); err != nil {
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

		fmt.Printf("switched to context %q\n", userSelectedContext)
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
