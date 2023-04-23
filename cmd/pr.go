/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"github.com/drbushytop/ado-yaml-validator/ado"
	"github.com/spf13/cobra"
)

// prCmd represents the pr command
var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Trigger PR mode",
	Long:  `This command triggers the tool in PR mode. It will validate all YAML files in the PR. It will attempt to use the System.AccessToken variable.`,
	Run: func(cmd *cobra.Command, args []string) {

		env, err := ado.NewAzureDevOpsEnvironmentFromPR()
		if err != nil {
			panic(err)
		}

		client := ado.NewValidationClient(context.Background(), env)

		client.ValidateAllPrChanges(context.Background())
	},
}

func init() {
	rootCmd.AddCommand(prCmd)
}