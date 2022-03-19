package main

import (
	"github.com/danielm-codefresh/argo-multi-cluster/cmd/config"
	"github.com/spf13/cobra"
)

func main() {
	cmd := cobra.Command{
		Use: "amc",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	cmd.AddCommand(config.NewClusterAddCommand())
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}}
