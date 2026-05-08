package main

import "fmt"

func cmdPruneConfigFile(conf *KubeConfig) error {
	clusters := make(map[string]Cluster)
	for _, cluster := range conf.Clusters {
		clusters[cluster.Name] = cluster
	}
	users := make(map[string]User)
	for _, user := range conf.Users {
		users[user.Name] = user
	}

	// find contexts unknown cluster or user references
	delContexts := make([]string, 0)
	for _, context := range conf.Contexts {
		if _, ok := clusters[context.Data.Cluster]; !ok {
			delContexts = append(delContexts, context.Name)
			continue
		}
		if _, ok := users[context.Data.User]; !ok {
			delContexts = append(delContexts, context.Name)
			continue
		}
	}

	// remove all referenced clusters and users from map to detect unreferenced ones
	for _, context := range conf.Contexts {
		delete(clusters, context.Data.Cluster)
		delete(users, context.Data.User)
	}
	delClusters := make([]string, 0)
	for clusterName := range clusters {
		delClusters = append(delClusters, clusterName)
	}
	delUsers := make([]string, 0)
	for userName := range users {
		delUsers = append(delUsers, userName)
	}

	if len(delClusters)+len(delUsers)+len(delContexts) == 0 {
		fmt.Println("nothing to prune")
		return nil
	}

	if len(delClusters) > 0 {
		fmt.Println("following clusters will be removed:")
		for _, clusterName := range delClusters {
			fmt.Println("->", clusterName)
		}
	}
	if len(delUsers) > 0 {
		fmt.Println("following users will be removed:")
		for _, userName := range delUsers {
			fmt.Println("->", userName)
		}
	}
	if len(delContexts) > 0 {
		fmt.Println("following contexts will be removed:")
		for _, contextName := range delContexts {
			fmt.Println("->", contextName)
		}
	}

	if !cli.Prune.Yes {
		fmt.Print("proceed? (y/N) ")
		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("user abort")
			return nil
		}
	}

	for _, clusterName := range delClusters {
		conf.RemoveClusterByName(clusterName)
	}
	for _, userName := range delUsers {
		conf.RemoveUserByName(userName)
	}
	for _, contextName := range delContexts {
		conf.RemoveContextByName(contextName)
	}

	if err := conf.Save(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Println("config pruned")
	return nil
}
