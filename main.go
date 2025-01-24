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
		} `cmd:"update" help:"Create contexts for all Namespaces in configured clusters."`
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
		return cmdSelectContext(conf)

	case "update":
		return cmdUpdateConfigFile(conf)

	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}
