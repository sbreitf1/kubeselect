package main

import (
	"fmt"
	"os"
)

func main() {
	conf, err := ReadKubeConfig()
	if err != nil {
		fmt.Println("ERROR reading kube config:", err.Error())
		os.Exit(1)
	}

	if len(os.Args) == 2 {
		if os.Args[1] == "help" || os.Args[1] == "-help" || os.Args[1] == "--help" {
			fmt.Println("kubeselect usage")
			fmt.Println("--help   Display this help")
			fmt.Println(" -u      Create contexts for all namespaces in all clusters")
			os.Exit(0)
		}
		if os.Args[1] == "-u" {
			fmt.Println("update config file")
			if err := cmdUpdateConfigFile(conf); err != nil {
				fmt.Println("ERROR:", err.Error())
				os.Exit(1)
			}
			os.Exit(0)
		}
		fmt.Println("unsupported args")
		os.Exit(1)
	}
	if len(os.Args) > 2 {
		fmt.Println("unsupported args")
		os.Exit(1)
	}

	if err := cmdSelectContext(conf); err != nil {
		fmt.Println("ERR:", err)
		os.Exit(1)
	}
}
