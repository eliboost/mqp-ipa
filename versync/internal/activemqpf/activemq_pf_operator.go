// Package activemqpf contains functions to manage ActiveMQ PF operator versions
package activemqpf

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go/v3"
	"gopkg.in/yaml.v3"
)

// Image struct to parse the images section in kustomization.yaml
type Image struct {
	Name   string `yaml:"name"`
	NewTag string `yaml:"newTag"`
}

// Kustomization struct to parse the yaml file
type Kustomization struct {
	Resources []string `yaml:"resources"`
	Images    []Image  `yaml:"images"`
}

// GetOperatorVersion extracts the PF ActiveMQ Operator Version from the kustomization.yaml images section
func GetOperatorVersion(git *gitlab.Client, projectPath, envPath, branch string) (string, error) {
	kustomizationFile := envPath + "/kustomization.yaml"

	file, _, err := git.RepositoryFiles.GetFile(
		projectPath,
		kustomizationFile,
		&gitlab.GetFileOptions{
			Ref: new(branch),
		},
	)
	if err != nil {
		return "", fmt.Errorf("API error: %w", err)
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return "", fmt.Errorf("decoding error: %w", err)
	}

	var kust Kustomization
	if err := yaml.Unmarshal(content, &kust); err != nil {
		return "", fmt.Errorf("YAML parsing error: %w", err)
	}

	for _, img := range kust.Images {
		if strings.Contains(img.Name, "pf-activemq-operator") && img.NewTag != "" {
			version, _, _ := strings.Cut(img.NewTag, "@")
			return version, nil
		}
	}

	return "", errors.New("could not extract version from images or resource URL")
}

// GetArtemisVersion extracts the Artemis Broker Version from the kustomization.yaml
func GetArtemisVersion(git *gitlab.Client, projectPath, envPath, branch string) (string, error) {
	kustomizationFile := envPath + "/kustomization.yaml"

	file, _, err := git.RepositoryFiles.GetFile(
		projectPath,
		kustomizationFile,
		&gitlab.GetFileOptions{
			Ref: new(branch),
		},
	)
	if err != nil {
		return "", fmt.Errorf("API error: %w", err)
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return "", fmt.Errorf("decoding error: %w", err)
	}

	contentStr := string(content)

	artemisRegex := regexp.MustCompile(`--artemis-image=[^\s]*:artemis\.(\d+\.\d+\.\d+)`)
	artemisMatches := artemisRegex.FindStringSubmatch(contentStr)

	if len(artemisMatches) < 2 {
		return "", errors.New("couldn't find any Artemis image versions ")
	}

	return artemisMatches[1], nil
}

// ProcessEnvironmentFiles processes the environment files
func ProcessEnvironmentFiles(git *gitlab.Client, projectPath, envPath string, branch string) error {
	_, err := GetOperatorVersion(git, projectPath, envPath, branch)
	if err != nil {
		return fmt.Errorf("error checking Operator version: %w", err)
	}

	_, err = GetArtemisVersion(git, projectPath, envPath, branch)
	if err != nil {
		return fmt.Errorf("error checking Artemis version: %w", err)
	}

	return nil
}
