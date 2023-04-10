package ado

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/pipelines"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

type ValidationClient struct {
	environment    AzureDevOpsEnvironment
	pipelineClient *pipelines.ClientImpl
	targetBranch   string
}

func NewValidationClient(ctx context.Context, environment AzureDevOpsEnvironment) *ValidationClient {
	pClient := environment.connection.GetClientByUrl(environment.connection.BaseUrl)

	return &ValidationClient{
		environment:    environment,
		pipelineClient: &pipelines.ClientImpl{Client: *pClient},
	}
}

type ValidationResult struct {
	pipelinePath string
	err          error
}

// ValidatePR validates a single pipeline with the given ID
func (c ValidationClient) ValidatePR(ctx context.Context, pipeline Pipeline) (ValidationResult, error) {
	args := c.newPreviewPipelineArgs(pipeline.Id)

	// TODO: handle error in case where call does not go through
	_, err := c.callPreviewApi(ctx, args)
	result := ValidationResult{
		pipelinePath: pipeline.FilePath,
	}
	if err != nil {
		result.err = err
	}

	return result, nil
}

type Pipeline struct {
	FilePath string
	Id       int
}

// ValidateAllPrChanges validates all pipelines that have changed in the given pull request
func (c ValidationClient) ValidateAllPrChanges(ctx context.Context, pullRequestId int) []error {
	errs := make([]error, 0)
	changes, err := c.getPullRequestChangedYamlFiles(pullRequestId)
	if err != nil {
		return append(errs, fmt.Errorf("ValidateAllChanges: failed to get changed files: %w", err))
	}
	changedPipelines, err := c.getChangedPipelines(ctx, changes)
	if err != nil {
		return append(errs, fmt.Errorf("ValidateAllChanges: failed to get changed pipelines: %w", err))
	}

	results := make(chan ValidationResult)
	for i := range changedPipelines {
		go func(pipeline Pipeline) {
			result, err := c.ValidatePR(ctx, pipeline)
			if err != nil {
				log.Printf("ValidateAllChanges: failed to validate pipeline %s: %v", pipeline, err)
			}

			for {
				select {
				case results <- result:
					return
				case <-ctx.Done():
					return
				}
			}

		}(changedPipelines[i])
	}

	for result := range results {
		if result.err != nil {
			errs = append(errs, result.err)
			fmt.Printf("pipeline %s failed validation: %s", result.pipelinePath, result.err)
		} else {
			fmt.Printf("pipeline %s passed validation", result.pipelinePath)
		}
	}

	return errs
}

// previewParameters are the Body parameters for the Preview call: https://learn.microsoft.com/en-us/rest/api/azure/devops/pipelines/preview/preview?view=azure-devops-rest-7.0#request-body
type previewParameters struct {
	resources    *pipelines.RunResourcesParameters
	previewRun   *bool
	yamlOverride *string
}

// Arguments for the callValidationApi function
type previewPipelineArgs struct {
	// (required) Body parameters for the Preview call: https://learn.microsoft.com/en-us/rest/api/azure/devops/pipelines/preview/preview?view=azure-devops-rest-7.0#request-body
	PreviewParameters *previewParameters
	// (required) Project ID or project name
	Project *string
	// (required) The pipeline id
	PipelineId *int
	// (optional) The pipeline version
	PipelineVersion *int
}

func (c ValidationClient) newPreviewPipelineArgs(pipelineId int, opts ...previewPipelineArgOpt) previewPipelineArgs {
	repoMap := make(map[string]pipelines.RepositoryResourceParameters)
	repoMap["self"] = pipelines.RepositoryResourceParameters{
		RefName: Pointer(c.environment.runBranch),
	}

	runResourceParams := pipelines.RunResourcesParameters{
		Repositories: &repoMap,
	}

	previewParams := previewParameters{
		resources:  &runResourceParams,
		previewRun: Pointer(true),
	}

	args := previewPipelineArgs{
		PipelineId:        Pointer(pipelineId),
		PreviewParameters: &previewParams,
		Project:           Pointer(c.environment.project),
	}

	for _, opt := range opts {
		opt(&args)
	}

	return args
}

type previewPipelineArgOpt func(*previewPipelineArgs)

func withYamlOverride(yamlOverride string) previewPipelineArgOpt {
	return func(args *previewPipelineArgs) {
		args.PreviewParameters.yamlOverride = Pointer(yamlOverride)
	}
}

type PreviewRun struct {
	FinalYaml *string `json:"finalYaml,omitempty"`
}

func (c ValidationClient) callPreviewApi(ctx context.Context, args previewPipelineArgs) (*PreviewRun, error) {
	routeValues := make(map[string]string)
	if args.Project == nil || *args.Project == "" {
		return nil, &azuredevops.ArgumentNilOrEmptyError{ArgumentName: "args.Project"}
	}
	routeValues["project"] = *args.Project
	if args.PipelineId == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.PipelineId"}
	}
	routeValues["pipelineId"] = strconv.Itoa(*args.PipelineId)

	queryParams := url.Values{}
	if args.PipelineVersion != nil {
		queryParams.Add("pipelineVersion", strconv.Itoa(*args.PipelineVersion))
	}
	body, marshalErr := json.Marshal(*args.PreviewParameters)
	if marshalErr != nil {
		return nil, marshalErr
	}
	locationId, _ := uuid.Parse("53df2d18-29ea-46a9-bee0-933540f80abf")
	resp, err := c.pipelineClient.Client.Send(ctx, http.MethodPost, locationId, "7.0", routeValues, queryParams, bytes.NewReader(body), "application/json", "application/json", nil)
	if err != nil {
		return nil, err
	}

	var responseValue PreviewRun
	err = c.pipelineClient.Client.UnmarshalBody(resp, &responseValue)
	return &responseValue, err
}

// PR Case:
// Get project, organization, PR branch? (merge/something) from PR
// Get changed .yaml files from PR
// Get all pipelines in project, filter for pipelines that directly use those yaml files OR use the file as a template
// for each file, call the validation api.
// As this is a PR, we do not need to overwrite the yaml, as the changes are already in the repository.

// Local Case:
// Take in project, organization, branch, auth token, branch to compare to (defaulting to master) from user
// Get changed .yaml files from git diff?
// Get all pipelines in project, filter for pipelines that directly use those yaml files (Later: OR use the file as a template)
// If no pipelines are found (this file is a template), just select the first one returned for the validation call (we override the contents)
// for each file, call the validation api, using the yamloverride by parsing local yaml.