package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func cmdSelectContext(conf *KubeConfig) error {
	contexts, err := conf.Contexts()
	if err != nil {
		return fmt.Errorf("parse contexts: %w", err)
	}

	if len(contexts) == 0 {
		fmt.Println("no contexts defined")
		return nil
	}

	clusters := GroupContextsByCluster(contexts)

	selectedContext, err := conf.SelectedContext()
	if err != nil {
		return fmt.Errorf("get selected context: %w", err)
	}

	userSelectedContext, err := showSelectionUI(clusters, selectedContext)
	if err != nil {
		return fmt.Errorf("show selection ui: %w", err)
	}

	if len(userSelectedContext) > 0 {
		if userSelectedContext == selectedContext {
			// nothing to do
			return nil
		}

		conf.SetSelectedContext(userSelectedContext)
		if err := conf.Save(); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Println("switched to context", userSelectedContext)
	}
	return nil
}

func showSelectionUI(clusters []ClusterWithContexts, selectedContext string) (string, error) {
	var contextCount int
	for _, cluster := range clusters {
		contextCount += len(cluster.Contexts)
	}

	var expandMode bool
	if contextCount > 30 {
		expandMode = true
	}

	var userSelectedContext string

	var currentSelectionClusterNode, currentSelectionContextNode *tview.TreeNode

	app := tview.NewApplication()
	rootNode := tview.NewTreeNode("Clusters").SetSelectable(false)
	treeView := tview.NewTreeView().SetRoot(rootNode)
	var firstContextNode *tview.TreeNode
	for _, cluster := range clusters {
		clusterNode := tview.NewTreeNode(cluster.Name).SetSelectable(expandMode)
		if expandMode {
			clusterNode.SetSelectedFunc(func() {
				if clusterNode.IsExpanded() {
					clusterNode.Collapse()
				} else {
					clusterNode.Expand()
				}
			})
		}
		clusterNode.SetColor(tcell.ColorGreen)
		var containsSelection bool
		for _, c := range cluster.Contexts {
			var nodeName string
			if expandMode {
				nodeName = c.Context.Namespace
			} else {
				nodeName = c.Name
			}
			contextNode := tview.NewTreeNode(nodeName).SetSelectable(true)
			if c.Name == selectedContext {
				// this is the currently selected context
				contextNode.SetColor(tcell.ColorYellow)
				treeView.SetCurrentNode(contextNode)
				currentSelectionContextNode = contextNode
				containsSelection = true
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
			if expandMode {
				contextNode.SetReference(clusterNode)
			}
			clusterNode.AddChild(contextNode)
		}
		if expandMode {
			if containsSelection {
				currentSelectionClusterNode = clusterNode
				clusterNode.SetColor(tcell.ColorYellow)
			} else {
				clusterNode.Collapse()
			}
		}
		rootNode.AddChild(clusterNode)
	}

	if treeView.GetCurrentNode() == nil {
		// no default selection? fall back to first visible node
		treeView.SetCurrentNode(firstContextNode)
	}

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.Stop()
			return nil
		}
		if event.Key() == tcell.KeyRight {
			if expandMode {
				if node := treeView.GetCurrentNode(); node != nil {
					if len(node.GetChildren()) > 0 && !node.IsExpanded() {
						treeView.GetCurrentNode().Expand()
					} else if currentSelectionContextNode != nil && node == currentSelectionClusterNode {
						treeView.SetCurrentNode(currentSelectionContextNode)
					} else {
						if childs := node.GetChildren(); len(childs) > 0 {
							treeView.SetCurrentNode(childs[len(childs)-1])
						}
					}
				}
			}
			return nil
		}
		if event.Key() == tcell.KeyLeft {
			if node := treeView.GetCurrentNode(); node != nil {
				if len(node.GetChildren()) > 0 && node.IsExpanded() {
					treeView.GetCurrentNode().Collapse()
				} else if ref := node.GetReference(); ref != nil {
					if ref, ok := ref.(*tview.TreeNode); ok {
						treeView.SetCurrentNode(ref)
					}
				}
			}
			return nil
		}
		return event
	})

	app.SetRoot(treeView, true)
	if err := app.Run(); err != nil {
		return "", err
	}
	return userSelectedContext, nil
}
