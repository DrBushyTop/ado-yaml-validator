package ado

import (
	"fmt"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"os"
	"strings"
)

func NewOauthConnection(organizationUrl string, accessToken string) *azuredevops.Connection {
	authorizationString := "Bearer " + accessToken
	normalizedUrl := normalizeUrl(organizationUrl)

	return &azuredevops.Connection{
		AuthorizationString:     authorizationString,
		BaseUrl:                 normalizedUrl,
		SuppressFedAuthRedirect: true,
	}
}

func normalizeUrl(url string) string {
	return strings.ToLower(strings.TrimRight(url, "/"))
}

type AzureDevOpsEnvironment struct {
	organizationUrl string
	project         string
	connection      *azuredevops.Connection
	runBranch       string
	repositoryId    string
}

func NewAzureDevOpsEnvironment(conn *azuredevops.Connection, opts ...EnvOption) (*AzureDevOpsEnvironment, error) {
	env := &AzureDevOpsEnvironment{
		connection: conn,
	}

	for _, opt := range opts {
		err := opt(env)
		if err != nil {
			return nil, err
		}
	}

	return env, nil
}

type EnvOption func(*AzureDevOpsEnvironment) error

func WithPrEnv() EnvOption {
	return func(env *AzureDevOpsEnvironment) error {
		orgUrl := os.Getenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI")
		if orgUrl == "" {
			return fmt.Errorf("WithPrEnv: failed to retrieve organization URL from environment variables")
		}
		project := os.Getenv("SYSTEM_TEAMPROJECT")
		if project == "" {
			return fmt.Errorf("WithPrEnv: failed to retrieve project name from environment variables")
		}
		runBranch := os.Getenv("BUILD_SOURCEBRANCH")
		if runBranch == "" {
			return fmt.Errorf("WithPrEnv: failed to retrieve run branch from environment variables")
		}
		repositoryId := os.Getenv("BUILD_REPOSITORY_ID")
		if repositoryId == "" {
			return fmt.Errorf("WithPrEnv: failed to retrieve repository ID from environment variables")
		}

		env.organizationUrl = orgUrl
		env.project = project
		env.runBranch = runBranch
		env.repositoryId = repositoryId

		return nil
	}
}