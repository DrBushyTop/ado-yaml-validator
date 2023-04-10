package main

import (
	"context"
	"github.com/drbushytop/ado-yaml-validator/ado"
)

func main() {
	//cmd.Execute()

	env, err := ado.NewAzureDevOpsEnvironmentFromPR()
	if err != nil {
		panic(err)
	}

	client := ado.NewValidationClient(context.Background(), env)

	client.ValidateAllPrChanges(context.Background())
}