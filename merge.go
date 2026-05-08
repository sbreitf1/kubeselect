package main

import (
	"fmt"
	"io"
	"os"
)

func cmdMergeKubeConfig(conf *KubeConfig) error {
	fmt.Println("paste kubeconfig data and confirm with Enter, then Ctrl + D") //TODO switch for windows -> Ctrl + Z then Enter
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	newConf, err := ParseKubeConfig(data)
	if err != nil {
		return fmt.Errorf("entered config is invalid: %w", err)
	}

	if len(newConf.Clusters) == 0 && len(newConf.Users) == 0 {
		fmt.Println("no clusters or users defined in config")
		return nil
	}

	if len(newConf.Clusters) != 1 || len(newConf.Users) != 1 {
		//TODO implement smart logic for more complex configs
		return fmt.Errorf("merge only works for kubeconfigs with exactly one cluster and user")
	}

	//TODO check for clusters and users that do already exist

	fmt.Print("enter cluster name (" + newConf.Clusters[0].Name + ")> ")
	var clusterName string
	fmt.Scanln(&clusterName)
	if len(clusterName) == 0 {
		clusterName = newConf.Clusters[0].Name
	}
	//TODO ask again if cluster name already exists
	userName := clusterName
	//TODO add suffix if user name already exists

	// rename cluster in config
	newConf.Clusters[0].Name = clusterName
	newConf.Users[0].Name = userName
	for i := range newConf.Contexts {
		newConf.Contexts[i].Name = genContextName(clusterName, newConf.Contexts[i].Data.Namespace)
		newConf.Contexts[i].Data.Cluster = clusterName
		newConf.Contexts[i].Data.User = userName
	}

	conf.Clusters = append(conf.Clusters, newConf.Clusters...)
	conf.Contexts = append(conf.Contexts, newConf.Contexts...)
	conf.Users = append(conf.Users, newConf.Users...)

	if err := conf.Save(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Println("config has been merged")
	return nil
}
