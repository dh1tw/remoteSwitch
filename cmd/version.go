package cmd

import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var version string
var commitHash string

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of remoteSwitch",
	Long:  `All software has versions. This is remoteSwitch's.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func printVersion() {
	buildDate := time.Now().Format(time.RFC3339)
	fmt.Println("copyright Tobias Wellnitz, DH1TW, 2018")
	fmt.Printf("remoteSwitch Version: %s, %s/%s, BuildDate: %s, Commit: %s\n",
		version, runtime.GOOS, runtime.GOARCH, buildDate, commitHash)
}
