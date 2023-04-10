/*
Copyright © 2022 Pasi Huuhka pasi@huuhka.net
*/
package main

import (
	"context"
	"github.com/drbushytop/ado-yaml-validator/ado"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"os"
)

func main() {
	//cmd.Execute()

	conn := azuredevops.NewPatConnection(os.Getenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI"), os.Getenv("SYSTEM_ACCESSTOKEN"))
	env, err := ado.NewAzureDevOpsEnvironment(conn, ado.WithPrEnv())
	if err != nil {
		panic(err)
	}

	client := ado.NewValidationClient(context.Background(), env)

	client.ValidateAllPrChanges(context.Background())
}