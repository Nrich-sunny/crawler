package cmd

import (
	"github.com/Nrich-sunny/crawler/cmd/master"
	"github.com/Nrich-sunny/crawler/cmd/worker"
	"github.com/Nrich-sunny/crawler/version"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "run worker service.",
	Long:  "run worker service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		worker.Run()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print version.",
	Long:  "print version.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		version.Printer()
	},
}

func Execute() {
	var rootCmd = &cobra.Command{
		Use: "crawler",
	}
	rootCmd.AddCommand(workerCmd, master.MasterCmd, versionCmd)
	rootCmd.Execute()
}
