// Package main is the entry point of the application
package main

import (
	"log"
	"os"

	gitlab "gitlab.com/gitlab-org/api/client-go/v3"
	"gitlab.pnet.ch/middleware/activemq/versync/internal/activemqpf"
	"gitlab.pnet.ch/middleware/activemq/versync/internal/artemis"
	"gitlab.pnet.ch/middleware/activemq/versync/internal/config"
	"gitlab.pnet.ch/middleware/activemq/versync/internal/operator"
)

type ProcessFunc func(client *gitlab.Client, pid string, envPath string, branch string) error

type ProjectConfig struct {
	Path         string
	Environments map[string]string
	Process      ProcessFunc
}

func main() {
	gitlabToken := os.Getenv(config.GitlabTokenEnvKey)
	if gitlabToken == "" {
		log.Fatalf("%s environment variable is not set", config.GitlabTokenEnvKey)
	}

	branch := os.Getenv(config.BranchName)
	if branch == "" {
		log.Fatalf("%s environment variable is not set", config.BranchName)
	}

	git, err := gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(config.GitlabAPIURL))
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	if err := checkBranchExists(git, config.ArtemisPath, branch); err != nil {
		log.Fatalf("branch '%q' doesn't exist: %v", branch, err) //nolint:gosec // G706: branch is quoted via %q
	}

	configs := []ProjectConfig{
		{
			Path:         config.ArtemisPath,
			Process:      artemis.ProcessEnvironmentFiles,
			Environments: make(map[string]string),
		},
		{
			Path:         config.PfActivemqPath,
			Process:      activemqpf.ProcessEnvironmentFiles,
			Environments: make(map[string]string),
		},
		{
			Path: config.DocumentationPath,
			Process: func(client *gitlab.Client, pid string, _ string, branch string) error {
				return operator.GetRepoVersions(client, pid, branch, config.ArtemisPath, config.PfActivemqPath)
			},
			Environments: make(map[string]string),
		},
		{
			Path: config.DocumentationPath,
			Process: func(client *gitlab.Client, pid string, _ string, branch string) error {
				return operator.VersionUpdater(client, pid, branch, config.ArtemisPath, config.PfActivemqPath)
			},
		},
	}

	for i := range configs {
		if configs[i].Path == config.DocumentationPath {
			continue
		}

		rootPath := config.EnvsRootPath
		if configs[i].Path == config.PfActivemqPath {
			rootPath = config.OverlaysRootPath
		}

		foundEnvs, err := getEnvironmentsFromGit(git, configs[i].Path, rootPath, branch)
		if err != nil {
			log.Printf("couln't find environments in path '%s' in project %s %v", rootPath, configs[i].Path, err)

			continue
		}

		for _, envName := range foundEnvs {
			fullPath := rootPath + "/" + envName
			configs[i].Environments[envName] = fullPath
		}
	}

	for _, project := range configs {
		if project.Path == config.DocumentationPath {
			err := project.Process(git, project.Path, "", branch)
			if err != nil {
				log.Printf("error in project %s: %v", project.Path, err)
			}

			continue
		}

		for name, envPath := range project.Environments {
			err := project.Process(git, project.Path, envPath, branch)
			if err != nil {
				log.Printf("error in project %s, environment %s: %v", project.Path, name, err)

				continue
			}
		}
	}
}

func checkBranchExists(git *gitlab.Client, projectID, branch string) error {
	_, _, err := git.Branches.GetBranch(projectID, branch)
	if err != nil {
		return err
	}

	return nil
}

func getEnvironmentsFromGit(git *gitlab.Client, projectID, rootPath, branch string) ([]string, error) {
	var envs []string

	nodes, _, err := git.Repositories.ListTree(projectID, &gitlab.ListTreeOptions{
		Path: new(rootPath),
		Ref:  new(branch),
	})
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node.Type == "tree" {
			envs = append(envs, node.Name)
		}
	}

	return envs, nil
}
