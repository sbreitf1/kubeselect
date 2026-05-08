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
		} `cmd:"update" help:"Create Contexts for all Namespaces in configured Clusters. Contexts of unreachable Clusters remain untouched."`

		Merge struct {
		} `cmd:"merge" help:"Enter a KubeConfig to merge to local KubeConfig."`

		//TODO rename cluster command

		//TODO remove cluster command

		//TODO rename user command

		//TODO remove user command

		Prune struct {
			Yes bool `short:"y" help:"Skip user confirmation."`
		} `cmd:"prune" help:"Remove invalid or unreferenced Entries from KubeConfig. Displays items first and waits for user confirmation."`
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

	case "merge":
		return cmdMergeKubeConfig(conf)

	case "prune":
		return cmdPruneConfigFile(conf)

	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}
