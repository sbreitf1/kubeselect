package main

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func cmdSelectContext(conf *KubeConfig) error {
	if len(conf.Contexts) == 0 {
		fmt.Println("no contexts defined")
		return nil
	}

	userSelectedContext, err := showSelectionUI(conf)
	if err != nil {
		return fmt.Errorf("show selection ui: %w", err)
	}

	if len(userSelectedContext) > 0 {
		if userSelectedContext == conf.CurrentContext {
			// nothing to do
			return nil
		}

		conf.CurrentContext = userSelectedContext
		if err := conf.Save(); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Println("switched to context", userSelectedContext)
	}
	return nil
}

func showSelectionUI(conf *KubeConfig) (string, error) {
	var expandMode bool // if true, a fully functional tree-view with collapseable cluster nodes will be displayed
	if len(conf.Contexts) > 20 {
		expandMode = true
	}

	var userSelectedContext string

	app := tview.NewApplication()
	rootNode := tview.NewTreeNode("Clusters").SetSelectable(false)
	treeView := tview.NewTreeView().SetRoot(rootNode)
	var firstContextNode *tview.TreeNode
	for _, clusterName := range conf.GetSortedClusterNames() {
		clusterContexts := conf.GetContextsForCluster(clusterName)
		if len(clusterContexts) == 0 {
			continue
		}

		clusterNode := tview.NewTreeNode(clusterName).SetSelectable(expandMode)
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
		for _, c := range clusterContexts {
			var nodeName string
			if expandMode {
				nodeName = c.Data.Namespace
			} else {
				nodeName = c.Name
			}
			contextNode := tview.NewTreeNode(nodeName).SetSelectable(true)
			if c.Name == conf.CurrentContext {
				// this is the currently selected context
				contextNode.SetColor(tcell.ColorYellow)
				treeView.SetCurrentNode(contextNode)
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
						// expand collapsed cluster node
						treeView.GetCurrentNode().Expand()
					} else {
						// jump to last children of selected cluster node
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
					// collapse cluster node
					treeView.GetCurrentNode().Collapse()
				} else if ref := node.GetReference(); ref != nil {
					// jump to parent cluster node of selected context
					if ref, ok := ref.(*tview.TreeNode); ok {
						treeView.SetCurrentNode(ref)
					}
				}
			}
			return nil
		}
		r := event.Rune()
		if r >= 'A' && r <= 'Z' {
			r -= 'A'
			r += 'a'
		}
		if r >= 'a' && r <= 'z' {
			if node := treeView.GetCurrentNode(); node != nil {
				if len(node.GetChildren()) > 0 {
					if node.IsExpanded() {
						// select first child context node of selected cluster node beginning with type rune
						for _, child := range node.GetChildren() {
							if strings.HasPrefix(child.GetText(), string(r)) {
								treeView.SetCurrentNode(child)
								break
							}
						}

					} else {
						// select first (or next) cluster node beginning with typed rune
						childs := rootNode.GetChildren()
						var offset int
						for i := range childs {
							if childs[i] == treeView.GetCurrentNode() {
								offset = i + 1
								break
							}
						}
						for i := 0; i < len(childs); i++ {
							child := childs[(offset+i+len(childs))%len(childs)]
							if strings.HasPrefix(child.GetText(), string(r)) {
								treeView.SetCurrentNode(child)
								break
							}
						}
					}

				} else if ref := node.GetReference(); ref != nil {
					// select next child context node in parent cluster node of current selection beginning with typed rune
					if ref, ok := ref.(*tview.TreeNode); ok {
						childs := ref.GetChildren()
						var offset int
						for i := range childs {
							if childs[i] == treeView.GetCurrentNode() {
								offset = i + 1
								break
							}
						}
						for i := 0; i < len(childs); i++ {
							child := childs[(offset+i+len(childs))%len(childs)]
							if strings.HasPrefix(child.GetText(), string(r)) {
								treeView.SetCurrentNode(child)
								break
							}
						}
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
