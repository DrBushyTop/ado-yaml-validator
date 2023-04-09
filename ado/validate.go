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

// Validate validates a single pipeline file
func (c ValidationClient) Validate(ctx context.Context, pipelineFilePath string) (ValidationResult, error) {
	return ValidationResult{}, nil
}

// GetChangedPipelines returns a list of pipelines that have changed in the given pull request
func (c ValidationClient) GetChangedPipelines(ctx context.Context, pullRequestId int) ([]string, error) {
	return nil, nil
}

// ValidateAllChanges validates all pipelines that have changed in the given pull request
func (c ValidationClient) ValidateAllChanges(ctx context.Context, pullRequestId int) []error {
	errs := make([]error, 0)
	changed, err := c.GetChangedPipelines(ctx, pullRequestId)
	if err != nil {
		return append(errs, fmt.Errorf("ValidateAllChanges: failed to get changed pipelines: %w", err))
	}

	results := make(chan ValidationResult)
	for i := range changed {
		go func(pipeline string) {
			result, err := c.Validate(ctx, pipeline)
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

		}(changed[i])
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
	previewRun   bool
	yamlOverride string
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

func (c ValidationClient) callPreviewApi(ctx context.Context, args previewPipelineArgs) (*pipelines.Run, error) {
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
	locationId, _ := uuid.Parse("7859261e-d2e9-4a68-b820-a5d84cc5bb3d")
	resp, err := c.pipelineClient.Client.Send(ctx, http.MethodPost, locationId, "7.0", routeValues, queryParams, bytes.NewReader(body), "application/json", "application/json", nil)
	if err != nil {
		return nil, err
	}

	var responseValue pipelines.Run
	err = c.pipelineClient.Client.UnmarshalBody(resp, &responseValue)
	return &responseValue, err
}