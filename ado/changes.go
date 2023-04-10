package ado

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/pipelines"
	"io"
	"log"
	"net/http"
	"path/filepath"
)

func (c ValidationClient) getPullRequestChangedYamlFiles(pullRequestId int) ([]string, error) {
	ctx := context.Background()

	gitClient, err := git.NewClient(ctx, c.environment.connection)
	if err != nil {
		return nil, fmt.Errorf("getPullRequestChangedYamlFiles: failed to create git client: %w", err)
	}

	var changedYamlFiles []string
	params := git.GetPullRequestIterationChangesArgs{
		Project:       Pointer(c.environment.project),
		RepositoryId:  Pointer(c.environment.repositoryId),
		PullRequestId: Pointer(pullRequestId),
		Top:           Pointer(1000),
	}

	for {
		changes, err := gitClient.GetPullRequestIterationChanges(ctx, params)
		if err != nil {
			return nil, err
		}

		changeEntries := changes.ChangeEntries
		for _, change := range *changeEntries {
			if change.SourceServerItem != nil && (filepath.Ext(*change.SourceServerItem) == ".yaml" || filepath.Ext(*change.SourceServerItem) == ".yml") && *change.ChangeType != "Delete" {
				changedYamlFiles = append(changedYamlFiles, *change.SourceServerItem)
			}
		}

		if changes.NextSkip == nil || *changes.NextSkip == 0 {
			break
		}

		params.Skip = changes.NextSkip
		params.Top = changes.NextTop
	}

	return changedYamlFiles, nil
}

// getChangedPipelines returns a list of pipelines that have changed in the given pull request
func (c ValidationClient) getChangedPipelines(ctx context.Context, changedYamlFiles []string) ([]Pipeline, error) {
	pipes, err := c.getAllProjectPipelines(ctx)
	if err != nil {
		return nil, fmt.Errorf("getChangedPipelines: failed to get all pipelines: %w", err)
	}
	result := make([]Pipeline, 0)

	fileMap := make(map[string]bool)

	for _, yamlFile := range changedYamlFiles {
		fileMap[yamlFile] = true
	}

	for _, pipeline := range pipes {
		if _, ok := fileMap[pipeline.FilePath]; ok {
			result = append(result, pipeline)
		}
	}

	return result, nil
}

type RestListPipelinesResponse struct {
	Id            *int                             `json:"id"`
	Configuration *pipelines.PipelineConfiguration `json:"configuration"`
}

type RestGetPipelineResponse struct {
	Process struct {
		YamlFilename string `json:"yamlFilename"`
	} `json:"process"`
}

func (c ValidationClient) getAllProjectPipelines(ctx context.Context) ([]Pipeline, error) {
	url := fmt.Sprintf("%s/%s/_apis/pipelines?api-version=7.0", c.environment.organizationUrl, c.environment.project)

	httpClient := http.DefaultClient
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("getAllProjectPipelines: failed to create request: %w", err)
	}
	req.Header.Add("Authorization", c.environment.connection.AuthorizationString)

	// TODO: Pagination handling
	response, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getAllProjectPipelines: failed to get response: %w", err)
	}

	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("getAllProjectPipelines: failed to read response body: %w", err)
	}

	var listResult []RestListPipelinesResponse
	err = json.Unmarshal(body, &listResult)
	if err != nil {
		return nil, fmt.Errorf("getAllProjectPipelines: failed to unmarshal response body: %w", err)
	}

	result := make([]Pipeline, 0)
	resultChan := make(chan Pipeline)
	for _, pipeline := range listResult {
		if pipeline.Configuration != nil && *pipeline.Configuration.Type == pipelines.ConfigurationTypeValues.Yaml {
			go c.getSinglePipeline(ctx, *pipeline.Id, resultChan)
		}
	}
	for p := range resultChan {
		result = append(result, p)
	}

	return result, nil
}

func (c ValidationClient) getSinglePipeline(ctx context.Context, pipelineId int, results chan<- Pipeline) {
	url := fmt.Sprintf("%s/%s/_apis/pipelines/%d?api-version=7.0", c.environment.organizationUrl, c.environment.project, pipelineId)

	httpClient := http.DefaultClient
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("getAllProjectPipelines: failed to create request: %v", err)
		return
	}
	req.Header.Add("Authorization", c.environment.connection.AuthorizationString)
	response, err := httpClient.Do(req)
	if err != nil {
		log.Printf("getAllProjectPipelines: failed to get response: %v", err)
		return
	}

	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	var pipelineResult RestGetPipelineResponse
	err = json.Unmarshal(body, &pipelineResult)
	if err != nil {
		log.Printf("getAllProjectPipelines: failed to unmarshal response body: %v", err)
		return
	}

	select {
	case results <- Pipeline{
		Id:       pipelineId,
		FilePath: pipelineResult.Process.YamlFilename,
	}:
	case <-ctx.Done():
		return
	}

}