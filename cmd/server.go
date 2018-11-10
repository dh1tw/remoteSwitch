package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Switch Server",
	Long: `Run a remoteSwitch server

Start a remoteSwitch server using a specific transport protocol`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please select the server type (--help for available options)")
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
