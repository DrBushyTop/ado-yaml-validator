package cmd

import (
	"context"
	"fmt"
	"github.com/drbushytop/ado-yaml-validator/ado"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ado-yaml-validator",
	Short: "A tool to validate Azure Pipelines",
	Long: `A tool to validate Azure Pipelines.

This tool can be used to validate Azure Pipelines YAML files. The main use is during development of pipelines, but also
as a check during PRs.
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: RunRoot,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Printf(err.Error())
		os.Exit(1)
	}
}

func RunRoot(cmd *cobra.Command, args []string) error {
	// Local Case:
	// Take in project, organization, branch, auth token, branch to compare to (defaulting to master) from user
	// Get changed .yaml files from git diff?
	// Get all pipelines in project, filter for pipelines that directly use those yaml files (Later: OR use the file as a template)
	// If no pipelines are found (this file is a template), just select the first one returned for the validation call (we override the contents)
	// for each file, call the validation api, using the yamloverride by parsing local yaml.

	org := cmd.Flag("org").Value.String()
	project := cmd.Flag("project").Value.String()
	repo := cmd.Flag("repo").Value.String()

	var orgUrl string

	if org != "" {
		// If org is given, project and repo must be given as well
		orgUrl = createOrgUrl(org)
	} else {
		// Parse from current git repo
		var err error
		orgUrl, project, repo, err = parseOrgFromGit()
		if err != nil {
			return err
		}
	}

	bearer := cmd.Flag("bearer").Value.String()
	pat := cmd.Flag("pat").Value.String()
	if bearer == "" && pat == "" {
		// Unable to set mutually exclusive but still required
		return fmt.Errorf("either the --bearer or --pat argument must be given")
	}

	var conn *azuredevops.Connection
	if bearer != "" {
		conn = ado.NewOauthConnection(orgUrl, bearer)
	} else {
		conn = azuredevops.NewPatConnection(orgUrl, pat)
	}

	branch := cmd.Flag("branch").Value.String()
	env, err := ado.NewAzureDevOpsEnvironment(conn, project, branch, repo)
	if err != nil {
		return err
	}

	client := ado.NewValidationClient(context.Background(), env)

	fmt.Printf("%v", *client)
	//client.ValidateAllChanges(context.Background())

	return nil
}

func init() {
	rootCmd.Flags().String("bearer", "", "oAuth token for Azure DevOps. Either this or the -pat argument is needed unless the pr command is used.")
	rootCmd.Flags().String("pat", "", "personal access token. Either this or the -bearer argument needs to be given. unless the pr command is used.")
	rootCmd.MarkFlagsMutuallyExclusive("bearer", "pat")

	rootCmd.Flags().String("org", "", "Azure DevOps organization name. For example, if the Org URL is https://dev.azure.com/organization, then the organization name is 'organization'. If not given, project will be tried to be determined from the current git repository.")
	rootCmd.Flags().String("project", "", "Azure DevOps project name. If not given, project will be tried to be determined from the current git repository.")
	rootCmd.Flags().String("repo", "", "Azure DevOps repository name. If not given, repository will be tried to be determined from the current git repository.")
	rootCmd.MarkFlagsRequiredTogether("org", "project", "repo")

	rootCmd.Flags().String("branch", "master", "Branch name in the repository to compare against. Defaults to master.")
}

func createOrgUrl(org string) string {
	return "https://dev.azure.com/" + org
}

func createProjectUrl(orgUrl string, project string) string {
	return orgUrl + "/" + project
}

func createRepoUrl(projectUrl string, repo string) string {
	return projectUrl + "/_git/" + repo
}

func parseOrgFromGit() (string, string, string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	gitOriginUrl, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("error executing command: %s", err)
	}

	splitUrl := strings.Split(string(gitOriginUrl), "/")
	if len(splitUrl) < 6 {
		return "", "", "", fmt.Errorf("error parsing git origin url: %s", gitOriginUrl)
	}

	orgUrl := createOrgUrl(splitUrl[3])

	return orgUrl, splitUrl[4], splitUrl[6], nil
}