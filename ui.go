package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func selectContext(clusters []cluster, selectedContext string) (string, error) {
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
		return "", err
	}
	return userSelectedContext, nil
}
