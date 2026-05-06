package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

var (
	cli struct {
		Select struct {
		} `cmd:"select" default:"withargs" help:"Select active Kubernetes Context."`

		Update struct {
		} `cmd:"update" help:"Create Contexts for all Namespaces in configured Clusters."`
	}
)

func main() {
	ctx := kong.Parse(&cli)
	if err := execCmd(ctx.Command()); err != nil {
		fmt.Println("ERR:", err)
		os.Exit(1)
	}
}

func execCmd(cmd string) error {
	conf, err := ReadKubeConfig()
	if err != nil {
		fmt.Println("ERROR reading kube config:", err.Error())
		os.Exit(1)
	}

	switch cmd {
	case "select":
		// print config warnings for default usage (will be visible when the selection screen closes)
		conf.SanityCheck()
		return cmdSelectContext(conf)

	case "update":
		return cmdUpdateConfigFile(conf)

		//TODO add cluster command to enter new kubeconfig
		//TODO remove cluster command

	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}
