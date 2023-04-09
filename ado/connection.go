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
			return fmt.Errorf("getInstanceInfo: failed to retrieve organization URL from environment variables")
		}
		project := os.Getenv("SYSTEM_TEAMPROJECT")
		if project == "" {
			return fmt.Errorf("getInstanceInfo: failed to retrieve project name from environment variables")
		}

		env.organizationUrl = orgUrl
		env.project = project

		return nil
	}
}