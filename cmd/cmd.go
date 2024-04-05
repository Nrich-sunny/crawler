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

var masterCmd = &cobra.Command{
	Use:   "master",
	Short: "run master service.",
	Long:  "run master service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		master.Run()
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
	rootCmd.AddCommand(workerCmd, masterCmd, versionCmd)
	rootCmd.Execute()
}

var MasterId string
var HTTPListenAddress string
var GRPCListenAddress string

func init() {
	masterCmd.Flags().StringVar(&MasterId, "id", "1", "set master id")
	masterCmd.Flags().StringVar(&HTTPListenAddress, "http", ":8081", "set HTTP listen address")
	masterCmd.Flags().StringVar(&GRPCListenAddress, "grpc", ":9091", "set GRPC listen address")
}
