package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ado-yaml-validator",
	Short: "A tool to validate Azure Pipelines",
	Long: `A tool to validate Azure Pipelines.

This tool can be used to validate Azure Pipelines YAML files. The main use is during development of pipelines, but also
as a check during PRs.
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("bearer", "", "oAuth token for Azure DevOps. Either this or the -pat argument is needed. Use $(System.AccessToken) in pipelines.")
	rootCmd.PersistentFlags().String("pat", "", "personal access token. Either this or the -bearer argument needs to be given.")
}