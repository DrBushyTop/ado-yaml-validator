package main

import "github.com/drbushytop/ado-yaml-validator/cmd"

func main() {
	cmd.Execute()

	//env, err := ado.NewAzureDevOpsEnvironmentFromPR()
	//if err != nil {
	//	panic(err)
	//}
	//
	//client := ado.NewValidationClient(context.Background(), env)
	//
	//client.ValidateAllPrChanges(context.Background())
}