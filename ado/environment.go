package ado

import (
	"context"
	"fmt"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"os"
	"strconv"
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
	pullRequestId   int
}

func NewAzureDevOpsEnvironment(conn *azuredevops.Connection, project string, runBranch string, repositoryName string, opts ...EnvOption) (*AzureDevOpsEnvironment, error) {
	env := &AzureDevOpsEnvironment{
		connection:      conn,
		organizationUrl: conn.BaseUrl,
		project:         project,
		runBranch:       runBranch,
	}

	repoId, err := env.getRepoId(repositoryName)
	if err != nil {
		return nil, fmt.Errorf("NewAzureDevOpsEnvironment: failed to retrieve repository ID: %w", err)
	}

	env.repositoryId = repoId

	for _, opt := range opts {
		err := opt(env)
		if err != nil {
			return nil, err
		}
	}

	return env, nil
}

func NewAzureDevOpsEnvironmentFromPR() (*AzureDevOpsEnvironment, error) {
	env := &AzureDevOpsEnvironment{}

	orgUri := os.Getenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI")
	if orgUri == "" {
		return nil, fmt.Errorf("failed to retrieve organization URL from environment variables")
	}
	token := os.Getenv("SYSTEM_ACCESSTOKEN")
	if token == "" {
		return nil, fmt.Errorf("WithPrEnv: failed to retrieve access token from environment variables")
	}

	conn := NewOauthConnection(orgUri, token)

	orgUrl := os.Getenv("SYSTEM_TEAMFOUNDATIONCOLLECTIONURI")
	if orgUrl == "" {
		return nil, fmt.Errorf("WithPrEnv: failed to retrieve organization URL from environment variables")
	}
	project := os.Getenv("SYSTEM_TEAMPROJECT")
	if project == "" {
		return nil, fmt.Errorf("WithPrEnv: failed to retrieve project name from environment variables")
	}
	runBranch := os.Getenv("BUILD_SOURCEBRANCH")
	if runBranch == "" {
		return nil, fmt.Errorf("WithPrEnv: failed to retrieve run branch from environment variables")
	}
	repositoryId := os.Getenv("BUILD_REPOSITORY_ID")
	if repositoryId == "" {
		return nil, fmt.Errorf("WithPrEnv: failed to retrieve repository ID from environment variables")
	}

	pullRequestId, err := strconv.Atoi(os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID"))
	if err != nil {
		return nil, fmt.Errorf("WithPrEnv: failed to retrieve pull request ID from environment variables. %w", err)
	}

	env.connection = conn
	env.organizationUrl = orgUrl
	env.project = project
	env.runBranch = runBranch
	env.repositoryId = repositoryId
	env.pullRequestId = pullRequestId

	return env, nil
}

type EnvOption func(*AzureDevOpsEnvironment) error

func (e AzureDevOpsEnvironment) getRepoId(repoName string) (string, error) {
	client, err := git.NewClient(context.Background(), e.connection)
	if err != nil {
		return "", fmt.Errorf("getRepoId: failed to create git client. %w", err)
	}

	allRepos, err := client.GetRepositories(context.Background(), git.GetRepositoriesArgs{
		Project: Pointer(e.project),
	})

	if err != nil {
		return "", fmt.Errorf("getRepoId: failed to retrieve repositories. %w", err)
	}

	for _, repo := range *allRepos {
		if *repo.Name == repoName {
			return (*repo.Id).String(), nil
		}
	}

	return "", nil
}